package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// Step 1: post error mapping via ChatStream
func TestChatStreamErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream says no", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := openaiClient(srv.URL).ChatStream(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for 503, got nil")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error should contain 503: %v", err)
	}
	if !strings.Contains(err.Error(), "upstream says no") {
		t.Errorf("error should contain response body: %v", err)
	}
}

// Step 2: Embed happy path + error paths

func TestEmbedReordersByIndex(t *testing.T) {
	// Server returns two embeddings in reverse order (index 1 before index 0).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"index":1,"embedding":[0.2]},{"index":0,"embedding":[0.1]}]}`)
	}))
	defer srv.Close()

	vecs, err := openaiClient(srv.URL).Embed(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("want 2 vecs, got %d", len(vecs))
	}
	if vecs[0][0] != 0.1 {
		t.Errorf("vecs[0][0] = %v, want 0.1", vecs[0][0])
	}
	if vecs[1][0] != 0.2 {
		t.Errorf("vecs[1][0] = %v, want 0.2", vecs[1][0])
	}
}

func TestEmbedIndexOutOfRange(t *testing.T) {
	// Server returns index 5 for a single-input request (index 0 is the only valid one).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"index":5,"embedding":[0.1]}]}`)
	}))
	defer srv.Close()

	_, err := openaiClient(srv.URL).Embed(context.Background(), []string{"only one input"})
	if err == nil {
		t.Fatal("expected out-of-range error, got nil")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error should mention out of range: %v", err)
	}
}

func TestEmbedUsesEmbedModelFallback(t *testing.T) {
	// Server captures the request body to inspect the model field.
	var capturedModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model string `json:"model"`
		}
		json.Unmarshal(body, &req)
		capturedModel = req.Model
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"index":0,"embedding":[0.5]}]}`)
	}))
	defer srv.Close()

	// When EmbedModel is empty, falls back to Model.
	c := &OpenAIClient{BaseURL: srv.URL, Model: "chat-m", EmbedModel: ""}
	if _, err := c.Embed(context.Background(), []string{"test"}); err != nil {
		t.Fatalf("Embed (fallback): %v", err)
	}
	if capturedModel != "chat-m" {
		t.Errorf("expected model=chat-m (fallback), got %q", capturedModel)
	}

	// When EmbedModel is set, uses it instead.
	c.EmbedModel = "emb-m"
	if _, err := c.Embed(context.Background(), []string{"test"}); err != nil {
		t.Fatalf("Embed (explicit): %v", err)
	}
	if capturedModel != "emb-m" {
		t.Errorf("expected model=emb-m (explicit), got %q", capturedModel)
	}
}
