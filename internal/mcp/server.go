package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// ToolHandler runs a tool. arguments is the raw JSON object the client sent (may
// be nil). It returns human/agent readable text. A returned error is reported to
// the client as a tool execution error (isError: true) rather than a transport
// failure, which is what the MCP spec recommends for tool problems.
type ToolHandler func(ctx context.Context, arguments json.RawMessage) (string, error)

// Tool is a callable exposed through tools/list and tools/call.
type Tool struct {
	Name        string
	Description string
	// InputSchema is a JSON Schema object describing the tool arguments.
	InputSchema map[string]any
	Handler     ToolHandler
}

// Resource is read-only content exposed through resources/list and
// resources/read (used here for the offline documentation snapshot).
type Resource struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
	Read        func(ctx context.Context) (string, error)
}

// Server is a stdio JSON-RPC MCP server.
type Server struct {
	name    string
	version string

	instructions string

	tools     []*Tool
	toolIndex map[string]*Tool

	resources []*Resource
	resIndex  map[string]*Resource

	logf func(format string, args ...any)

	writeMu sync.Mutex
	enc     *json.Encoder
}

// NewServer creates a server identified by name/version.
func NewServer(name, version string) *Server {
	return &Server{
		name:      name,
		version:   version,
		toolIndex: map[string]*Tool{},
		resIndex:  map[string]*Resource{},
		logf:      func(string, ...any) {},
	}
}

// SetInstructions sets the optional onboarding text returned from initialize.
func (s *Server) SetInstructions(text string) { s.instructions = text }

// SetLogger installs a logging function used for diagnostics on stderr. stdout is
// reserved for the JSON-RPC stream and must never be written to directly.
func (s *Server) SetLogger(logf func(format string, args ...any)) {
	if logf != nil {
		s.logf = logf
	}
}

// AddTool registers a tool. Registration order is preserved in tools/list.
func (s *Server) AddTool(t *Tool) {
	if t == nil || t.Name == "" {
		return
	}
	if _, exists := s.toolIndex[t.Name]; exists {
		return
	}
	s.tools = append(s.tools, t)
	s.toolIndex[t.Name] = t
}

// AddResource registers a read-only resource.
func (s *Server) AddResource(r *Resource) {
	if r == nil || r.URI == "" {
		return
	}
	if _, exists := s.resIndex[r.URI]; exists {
		return
	}
	s.resources = append(s.resources, r)
	s.resIndex[r.URI] = r
}

// Serve runs the read/dispatch loop until the input stream closes or the context
// is cancelled. Messages are newline-delimited JSON values, which both the
// official SDKs and the reference stdio transport accept.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	s.enc = json.NewEncoder(out)
	reader := bufio.NewReaderSize(in, 1<<20)
	dec := json.NewDecoder(reader)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		var msg message
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			// A malformed frame should not kill the server; report and continue.
			s.logf("decode error: %v", err)
			s.respondError(nil, codeParseError, "parse error", err.Error())
			// json.Decoder cannot reliably resync mid-stream, so stop here.
			return nil
		}

		s.dispatch(ctx, &msg)
	}
}

func (s *Server) dispatch(ctx context.Context, msg *message) {
	isNotification := len(msg.ID) == 0

	switch msg.Method {
	case "initialize":
		s.handleInitialize(msg)
	case "notifications/initialized", "initialized":
		// No response for notifications.
	case "ping":
		s.respondResult(msg.ID, struct{}{})
	case "tools/list":
		s.respondResult(msg.ID, listToolsResult{Tools: s.toolDescriptors()})
	case "tools/call":
		s.handleCallTool(ctx, msg)
	case "resources/list":
		s.respondResult(msg.ID, listResourcesResult{Resources: s.resourceDescriptors()})
	case "resources/read":
		s.handleReadResource(ctx, msg)
	default:
		if isNotification {
			s.logf("ignoring unknown notification: %s", msg.Method)
			return
		}
		s.respondError(msg.ID, codeMethodNotFound, fmt.Sprintf("method not found: %s", msg.Method), nil)
	}
}

func (s *Server) handleInitialize(msg *message) {
	var params initializeParams
	if len(msg.Params) > 0 {
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			s.respondError(msg.ID, codeInvalidParams, "invalid initialize params", err.Error())
			return
		}
	}

	version := ProtocolVersion
	if supportedVersions[params.ProtocolVersion] {
		version = params.ProtocolVersion
	}

	s.logf("initialize from %s %s (protocol %s)", params.ClientInfo.Name, params.ClientInfo.Version, version)

	s.respondResult(msg.ID, initializeResult{
		ProtocolVersion: version,
		Capabilities: capabilities{
			Tools:     &toolsCapability{},
			Resources: &resourcesCapability{},
		},
		ServerInfo:   implementation{Name: s.name, Version: s.version},
		Instructions: s.instructions,
	})
}

func (s *Server) handleCallTool(ctx context.Context, msg *message) {
	var params callToolParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		s.respondError(msg.ID, codeInvalidParams, "invalid tools/call params", err.Error())
		return
	}

	tool, ok := s.toolIndex[params.Name]
	if !ok {
		s.respondError(msg.ID, codeInvalidParams, fmt.Sprintf("unknown tool: %s", params.Name), nil)
		return
	}

	text, err := tool.Handler(ctx, params.Arguments)
	if err != nil {
		// Tool errors are returned in-band so the model can read and react.
		s.respondResult(msg.ID, callToolResult{
			Content: textContent(fmt.Sprintf("%s failed: %v", params.Name, err)),
			IsError: true,
		})
		return
	}

	s.respondResult(msg.ID, callToolResult{Content: textContent(text)})
}

func (s *Server) handleReadResource(ctx context.Context, msg *message) {
	var params readResourceParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		s.respondError(msg.ID, codeInvalidParams, "invalid resources/read params", err.Error())
		return
	}

	res, ok := s.resIndex[params.URI]
	if !ok {
		s.respondError(msg.ID, codeInvalidParams, fmt.Sprintf("unknown resource: %s", params.URI), nil)
		return
	}

	text, err := res.Read(ctx)
	if err != nil {
		s.respondError(msg.ID, codeInternalError, "failed to read resource", err.Error())
		return
	}

	mime := res.MIMEType
	if mime == "" {
		mime = "text/plain"
	}
	s.respondResult(msg.ID, readResourceResult{
		Contents: []resourceContents{{URI: res.URI, MIMEType: mime, Text: text}},
	})
}

func (s *Server) toolDescriptors() []toolDescriptor {
	out := make([]toolDescriptor, 0, len(s.tools))
	for _, t := range s.tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		out = append(out, toolDescriptor{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}
	return out
}

func (s *Server) resourceDescriptors() []resourceDescriptor {
	out := make([]resourceDescriptor, 0, len(s.resources))
	for _, r := range s.resources {
		out = append(out, resourceDescriptor{
			URI:         r.URI,
			Name:        r.Name,
			Description: r.Description,
			MIMEType:    r.MIMEType,
		})
	}
	return out
}

func (s *Server) respondResult(id json.RawMessage, result any) {
	if len(id) == 0 {
		return // never reply to notifications
	}
	s.write(&message{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) respondError(id json.RawMessage, code int, msg string, data any) {
	s.write(&message{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg, Data: data}})
}

func (s *Server) write(m *message) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.enc == nil {
		return
	}
	if err := s.enc.Encode(m); err != nil {
		s.logf("write error: %v", err)
	}
}
