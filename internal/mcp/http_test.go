package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer() *Server {
	s := NewServer("runtz-mcp", "test")
	s.SetInstructions("hello")
	s.AddTool(&Tool{
		Name:        "echo",
		Description: "echoes",
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			return "pong", nil
		},
	})
	return s
}

func decode(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("response is not JSON (%v): %s", err, raw)
	}
	return m
}

func TestHandleMessageInitialize(t *testing.T) {
	resp := testServer().HandleMessage(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	m := decode(t, resp)
	result, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result object: %v", m)
	}
	info, ok := result["serverInfo"].(map[string]any)
	if !ok || info["name"] != "runtz-mcp" {
		t.Fatalf("unexpected serverInfo: %v", result["serverInfo"])
	}
}

func TestHandleMessageToolsList(t *testing.T) {
	resp := testServer().HandleMessage(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	if !strings.Contains(string(resp), `"echo"`) {
		t.Fatalf("tools/list did not include the registered tool: %s", resp)
	}
}

func TestHandleMessageToolCall(t *testing.T) {
	resp := testServer().HandleMessage(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`))
	if !strings.Contains(string(resp), "pong") {
		t.Fatalf("tools/call did not run the handler: %s", resp)
	}
}

func TestHandleMessageNotificationHasNoReply(t *testing.T) {
	resp := testServer().HandleMessage(context.Background(),
		[]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if len(resp) != 0 {
		t.Fatalf("notification should have no reply, got: %s", resp)
	}
}

func TestHandleMessageParseError(t *testing.T) {
	resp := testServer().HandleMessage(context.Background(), []byte(`{not json`))
	m := decode(t, resp)
	if _, ok := m["error"]; !ok {
		t.Fatalf("malformed request should return an error: %v", m)
	}
}

func TestMessageHandler(t *testing.T) {
	handler := testServer().MessageHandler()

	// POST with a request returns the JSON-RPC reply.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"echo"`) {
		t.Fatalf("POST body missing tool: %s", rec.Body.String())
	}

	// A notification yields 202 with no body.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("notification status = %d, want 202", rec.Code)
	}

	// GET is rejected.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", rec.Code)
	}
}
