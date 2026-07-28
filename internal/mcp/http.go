package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// maxRequestBytes caps a single JSON-RPC request body on the HTTP transport.
const maxRequestBytes = 4 << 20 // 4 MiB

// HandleMessage dispatches one JSON-RPC message and returns the response bytes.
// It returns nil for notifications (which have no reply). It is safe for
// concurrent use: the registered tools and resources are read-only after setup,
// and each call dispatches into its own output buffer.
func (s *Server) HandleMessage(ctx context.Context, raw []byte) []byte {
	var msg message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return encodeMessage(&message{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: codeParseError, Message: "parse error", Data: err.Error()},
		})
	}

	var buf bytes.Buffer
	s.forOutput(&buf).dispatch(ctx, &msg)
	return buf.Bytes()
}

// forOutput returns a shallow copy of the server whose responses are written to
// w, so concurrent requests each get their own encoder rather than racing on
// one shared writer. The tool/resource registries are shared by reference
// (read-only).
func (s *Server) forOutput(w io.Writer) *Server {
	return &Server{
		name:         s.name,
		version:      s.version,
		instructions: s.instructions,
		tools:        s.tools,
		toolIndex:    s.toolIndex,
		resources:    s.resources,
		resIndex:     s.resIndex,
		logf:         s.logf,
		enc:          json.NewEncoder(w),
	}
}

// MessageHandler implements the MCP streamable HTTP transport: each POST carries
// a single JSON-RPC message and the reply is returned as the response body.
// A notification (no id) yields 202 Accepted with an empty body.
func (s *Server) MessageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		if err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		resp := s.HandleMessage(r.Context(), body)
		if len(resp) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
	}
}

func encodeMessage(m *message) []byte {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(m)
	return buf.Bytes()
}
