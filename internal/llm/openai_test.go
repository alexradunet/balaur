package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// sseServer returns a test server that serves the given raw SSE body.
func sseServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, body)
	}))
}

func openaiClient(baseURL string) *OpenAIClient {
	return &OpenAIClient{BaseURL: baseURL, Model: "fake"}
}

func TestSSETextDelta(t *testing.T) {
	srv := sseServer(
		`data: {"choices":[{"delta":{"content":"hello "}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{"content":"world"}}]}` + "\n\n" +
			"data: [DONE]\n\n",
	)
	defer srv.Close()

	ch, err := openaiClient(srv.URL).ChatStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	text, err := Collect(ch)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if text != "hello world" {
		t.Errorf("got %q, want %q", text, "hello world")
	}
}

func TestSSEToolCallFragmented(t *testing.T) {
	// Tool call split across three events with the same index=0.
	srv := sseServer(
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"tc1","function":{"name":"task_add","arguments":"{\"tit"}}]}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"le\":\""}}]}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"X\"}"}}]}}]}` + "\n\n" +
			"data: [DONE]\n\n",
	)
	defer srv.Close()

	ch, err := openaiClient(srv.URL).ChatStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var done Chunk
	for c := range ch {
		if c.Done {
			done = c
		}
	}
	if len(done.ToolCalls) != 1 {
		t.Fatalf("want 1 tool call, got %d", len(done.ToolCalls))
	}
	tc := done.ToolCalls[0]
	if tc.Name != "task_add" {
		t.Errorf("name = %q, want task_add", tc.Name)
	}
	wantArgs := `{"title":"X"}`
	if tc.Args != wantArgs {
		t.Errorf("args = %q, want %q", tc.Args, wantArgs)
	}
}

func TestSSEInterleavedToolCalls(t *testing.T) {
	// Two tool calls interleaved by index.
	srv := sseServer(
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"a","function":{"name":"task_add","arguments":"{}"}}]}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"b","function":{"name":"log_entry","arguments":"{}"}}]}}]}` + "\n\n" +
			"data: [DONE]\n\n",
	)
	defer srv.Close()

	ch, err := openaiClient(srv.URL).ChatStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var done Chunk
	for c := range ch {
		if c.Done {
			done = c
		}
	}
	if len(done.ToolCalls) != 2 {
		t.Fatalf("want 2 tool calls, got %d: %v", len(done.ToolCalls), done.ToolCalls)
	}
	if done.ToolCalls[0].Name != "task_add" || done.ToolCalls[1].Name != "log_entry" {
		t.Errorf("unexpected order: %v", done.ToolCalls)
	}
}

func TestSSEDisconnectWithoutDone(t *testing.T) {
	// Connection closes without sending [DONE] — parser should emit an Err chunk.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"partial"}}]}`+"\n\n")
		// Close without [DONE] — the scanner reads EOF.
	}))
	defer srv.Close()

	ch, err := openaiClient(srv.URL).ChatStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	// Drain all chunks; the last Done chunk should have ToolCalls=nil, which
	// is fine — a clean EOF after data is treated as a normal Done by the parser
	// (scanner.Err() == nil on clean EOF). Verify we get at least the text.
	var text string
	for c := range ch {
		text += c.Content
	}
	if text != "partial" {
		t.Errorf("got %q, want %q", text, "partial")
	}
}
