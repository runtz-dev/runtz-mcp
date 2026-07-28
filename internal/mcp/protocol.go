// Package mcp implements a minimal, dependency-free Model Context Protocol
// server over the streamable HTTP transport using JSON-RPC 2.0. It
// intentionally avoids any third party modules so the server builds with
// just the Go standard library.
//
// The protocol surface implemented here is the subset runtz needs:
//   - initialize / notifications/initialized
//   - ping
//   - tools/list / tools/call
//   - resources/list / resources/read
//
// Reference: https://modelcontextprotocol.io/specification
package mcp

import "encoding/json"

// ProtocolVersion is the spec revision this server speaks. When a client sends a
// version we recognise we echo it back; otherwise we fall back to this value.
const ProtocolVersion = "2025-06-18"

var supportedVersions = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-06-18": true,
}

// JSON-RPC 2.0 error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// message is a JSON-RPC 2.0 envelope. A single struct covers requests,
// responses and notifications; absent fields are omitted on the wire.
type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *rpcError) Error() string { return e.Message }

// --- initialize ---

type initializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
	ClientInfo      implementation  `json:"clientInfo"`
}

type implementation struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    capabilities   `json:"capabilities"`
	ServerInfo      implementation `json:"serverInfo"`
	Instructions    string         `json:"instructions,omitempty"`
}

type capabilities struct {
	Tools     *toolsCapability     `json:"tools,omitempty"`
	Resources *resourcesCapability `json:"resources,omitempty"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type resourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

// --- tools ---

type toolDescriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type listToolsResult struct {
	Tools []toolDescriptor `json:"tools"`
}

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type callToolResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func textContent(text string) []contentBlock {
	return []contentBlock{{Type: "text", Text: text}}
}

// --- resources ---

type resourceDescriptor struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type listResourcesResult struct {
	Resources []resourceDescriptor `json:"resources"`
}

type readResourceParams struct {
	URI string `json:"uri"`
}

type resourceContents struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

type readResourceResult struct {
	Contents []resourceContents `json:"contents"`
}
