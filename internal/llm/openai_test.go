package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sseServer returns a test server that streams the given SSE lines for
// /chat/completions and records the last decoded request body.
func sseServer(t *testing.T, lines []string, capture *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			_ = decodeJSON(r, capture)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		for _, ln := range lines {
			_, _ = w.Write([]byte(ln + "\n\n"))
			if fl != nil {
				fl.Flush()
			}
		}
	}))
}

func decodeJSON(r *http.Request, dst *map[string]any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func TestChatStreamContentAndReasoning(t *testing.T) {
	lines := []string{
		`data: {"choices":[{"delta":{"reasoning_content":"thinking"}}]}`,
		`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
		`data: {"choices":[{"delta":{"content":", world"}}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, lines, nil)
	defer srv.Close()

	c := &OpenAIClient{BaseURL: srv.URL, Model: "test"}
	ch, err := c.ChatStream(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var content, reasoning string
	var done bool
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk err: %v", chunk.Err)
		}
		content += chunk.Content
		reasoning += chunk.Reasoning
		if chunk.Done {
			done = true
		}
	}
	if content != "Hello, world" {
		t.Errorf("content = %q, want %q", content, "Hello, world")
	}
	if reasoning != "thinking" {
		t.Errorf("reasoning = %q, want %q", reasoning, "thinking")
	}
	if !done {
		t.Error("never received Done chunk")
	}
}

func TestChatStreamAssemblesFragmentedToolCalls(t *testing.T) {
	// Two tool calls, arguments streamed in pieces and out of map order.
	lines := []string{
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_a","function":{"name":"task_add"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_b","function":{"name":"task_list"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"title\":"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{}"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"buy milk\"}"}}]}}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, lines, nil)
	defer srv.Close()

	c := &OpenAIClient{BaseURL: srv.URL, Model: "test"}
	ch, err := c.ChatStream(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var calls []ToolCall
	for chunk := range ch {
		if chunk.Done {
			calls = chunk.ToolCalls
		}
	}
	if len(calls) != 2 {
		t.Fatalf("got %d tool calls, want 2", len(calls))
	}
	// order must follow first-seen index (0 then 1).
	if calls[0].ID != "call_a" || calls[0].Name != "task_add" {
		t.Errorf("call[0] = %+v", calls[0])
	}
	if calls[0].Args != `{"title":"buy milk"}` {
		t.Errorf("call[0].Args = %q", calls[0].Args)
	}
	if calls[1].ID != "call_b" || calls[1].Name != "task_list" || calls[1].Args != "{}" {
		t.Errorf("call[1] = %+v", calls[1])
	}
}

func TestChatStreamOmitsEmptyTools(t *testing.T) {
	var body map[string]any
	srv := sseServer(t, []string{`data: [DONE]`}, &body)
	defer srv.Close()

	c := &OpenAIClient{BaseURL: srv.URL, Model: "test"}
	ch, err := c.ChatStream(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	for range ch { //nolint:revive // drain
	}
	if _, ok := body["tools"]; ok {
		t.Error("tools key should be omitted when no tools are offered")
	}

	// With a tool, the key must be present.
	body = nil
	c2 := &OpenAIClient{BaseURL: srv.URL, Model: "test"}
	ch2, err := c2.ChatStream(context.Background(), []Message{{Role: "user", Content: "hi"}},
		[]ToolSpec{{Name: "t", Description: "d", Parameters: map[string]any{"type": "object"}}})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	for range ch2 { //nolint:revive // drain
	}
	if _, ok := body["tools"]; !ok {
		t.Error("tools key should be present when a tool is offered")
	}
}

func TestChatStreamErrorBodyWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	c := &OpenAIClient{BaseURL: srv.URL, Model: "test", APIKey: "bad"}
	_, err := c.ChatStream(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err == nil {
		t.Fatal("expected an error for a 401 response")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("error should carry the provider body, got %v", err)
	}
}

func TestChatStreamContextCancel(t *testing.T) {
	// Server that never sends [DONE], so the stream stays open until ctx fires.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"x"}}]}` + "\n\n"))
		if fl != nil {
			fl.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	c := &OpenAIClient{BaseURL: srv.URL, Model: "test"}
	ch, err := c.ChatStream(ctx, []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	// Read the first content chunk, then cancel and confirm the channel drains.
	<-ch
	cancel()
	for range ch { //nolint:revive // drain until close
	}
}

func TestChatStreamSurfacesProviderError(t *testing.T) {
	lines := []string{
		`data: {"error":{"message":"rate limited"}}`,
	}
	srv := sseServer(t, lines, nil)
	defer srv.Close()

	c := &OpenAIClient{BaseURL: srv.URL, Model: "test"}
	ch, err := c.ChatStream(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var gotErr error
	var gotDone bool
	for chunk := range ch {
		if chunk.Err != nil {
			gotErr = chunk.Err
		}
		if chunk.Done {
			gotDone = true
		}
	}
	if gotErr == nil {
		t.Fatal("expected a Chunk.Err from mid-stream provider error, got none")
	}
	if !strings.Contains(gotErr.Error(), "rate limited") {
		t.Errorf("error should contain provider message, got %v", gotErr)
	}
	if gotDone {
		t.Error("should not receive a Done chunk when provider error aborts the stream")
	}
}

func TestToWireEmptyArgsBecomesObject(t *testing.T) {
	var body map[string]any
	srv := sseServer(t, []string{`data: [DONE]`}, &body)
	defer srv.Close()

	// Message with a tool call whose Args is empty (no-argument tool).
	msgs := []Message{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{ID: "call_x", Name: "task_list", Args: ""},
			},
		},
	}
	c := &OpenAIClient{BaseURL: srv.URL, Model: "test"}
	ch, err := c.ChatStream(context.Background(), msgs, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	for range ch { //nolint:revive // drain
	}

	// Navigate the captured body to messages[0].tool_calls[0].function.arguments.
	rawMsgs, ok := body["messages"].([]any)
	if !ok || len(rawMsgs) == 0 {
		t.Fatalf("captured body has no messages: %+v", body)
	}
	msg, ok := rawMsgs[0].(map[string]any)
	if !ok {
		t.Fatalf("message is not an object: %T", rawMsgs[0])
	}
	toolCalls, ok := msg["tool_calls"].([]any)
	if !ok || len(toolCalls) == 0 {
		t.Fatalf("message has no tool_calls: %+v", msg)
	}
	tc, ok := toolCalls[0].(map[string]any)
	if !ok {
		t.Fatalf("tool_call is not an object: %T", toolCalls[0])
	}
	fn, ok := tc["function"].(map[string]any)
	if !ok {
		t.Fatalf("tool_call has no function: %+v", tc)
	}
	args, _ := fn["arguments"].(string)
	if args != "{}" {
		t.Errorf("empty tool-call args should serialize as {}, got %q", args)
	}
}

func TestEmbedIndexMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return rows out of order to exercise index-based placement.
		_, _ = w.Write([]byte(`{"data":[{"index":1,"embedding":[0.3,0.4]},{"index":0,"embedding":[0.1,0.2]}]}`))
	}))
	defer srv.Close()

	c := &OpenAIClient{BaseURL: srv.URL, Model: "test"}
	vecs, err := c.Embed(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 2 || vecs[0][0] != 0.1 || vecs[1][0] != 0.3 {
		t.Errorf("embeddings not placed by index: %+v", vecs)
	}
}
