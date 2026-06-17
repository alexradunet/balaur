package kronk

import (
	"context"
	"testing"

	"github.com/ardanlabs/kronk/sdk/kronk/model"

	"github.com/alexradunet/balaur/internal/llm"
)

func TestToKronkMessages(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", Content: "be brief"},
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "", ToolCalls: []llm.ToolCall{{ID: "tc1", Name: "foo", Args: `{"x":1}`}}},
		{Role: "tool", ToolCallID: "tc1", Content: "42"},
	}
	out := toKronkMessages(msgs)
	if len(out) != 4 {
		t.Fatalf("got %d messages, want 4", len(out))
	}
	if out[0]["role"] != "system" || out[0]["content"] != "be brief" {
		t.Errorf("system message = %v", out[0])
	}
	if out[1]["role"] != "user" || out[1]["content"] != "hi" {
		t.Errorf("user message = %v", out[1])
	}
	if out[2]["role"] != "assistant" {
		t.Errorf("assistant role = %v", out[2]["role"])
	}
	calls, ok := out[2]["tool_calls"].([]map[string]any)
	if !ok || len(calls) != 1 {
		t.Fatalf("assistant tool_calls = %v", out[2]["tool_calls"])
	}
	fn := calls[0]["function"].(map[string]any)
	if fn["name"] != "foo" || fn["arguments"] != `{"x":1}` {
		t.Errorf("tool call function = %v", fn)
	}
	if _, has := out[2]["content"]; has {
		t.Errorf("empty assistant content should be omitted: %v", out[2])
	}
	if out[3]["role"] != "tool" || out[3]["tool_call_id"] != "tc1" || out[3]["content"] != "42" {
		t.Errorf("tool message = %v", out[3])
	}
}

func TestToKronkTools(t *testing.T) {
	tools := []llm.ToolSpec{{
		Name:        "search",
		Description: "find things",
		Parameters:  map[string]any{"type": "object"},
	}}
	out := toKronkTools(tools)
	if len(out) != 1 || out[0]["type"] != "function" {
		t.Fatalf("tools = %v", out)
	}
	fn := out[0]["function"].(map[string]any)
	if fn["name"] != "search" || fn["description"] != "find things" {
		t.Errorf("function = %v", fn)
	}
}

// streamOf builds a channel that yields the given responses then closes, like
// Kronk's ChatStreaming output.
func streamOf(resps ...model.ChatResponse) <-chan model.ChatResponse {
	ch := make(chan model.ChatResponse, len(resps))
	for _, r := range resps {
		ch <- r
	}
	close(ch)
	return ch
}

func delta(m model.ResponseMessage) model.ChatResponse {
	return model.ChatResponse{Choices: []model.Choice{{Delta: &m}}}
}

func drain(t *testing.T, ch <-chan llm.Chunk) []llm.Chunk {
	t.Helper()
	var got []llm.Chunk
	for c := range ch {
		got = append(got, c)
	}
	return got
}

func TestBridgeContentAndToolCalls(t *testing.T) {
	src := streamOf(
		delta(model.ResponseMessage{Content: "Hello "}),
		delta(model.ResponseMessage{Reasoning: "thinking"}),
		delta(model.ResponseMessage{Content: "world"}),
		delta(model.ResponseMessage{ToolCalls: []model.ResponseToolCall{{
			Index:    0,
			ID:       "tc1",
			Function: model.ResponseToolCallFunction{Name: "foo", Arguments: model.ToolCallArguments{"a": "b"}},
		}}}),
	)
	got := drain(t, bridge(context.Background(), src, func() {}))

	// Last chunk must be the single terminal Done with the assembled tool call.
	last := got[len(got)-1]
	if !last.Done {
		t.Fatalf("final chunk not Done: %+v", last)
	}
	if len(last.ToolCalls) != 1 || last.ToolCalls[0].ID != "tc1" || last.ToolCalls[0].Name != "foo" {
		t.Fatalf("assembled tool calls = %+v", last.ToolCalls)
	}
	if last.ToolCalls[0].Args != `{"a":"b"}` {
		t.Errorf("tool call Args = %q, want raw JSON object {\"a\":\"b\"}", last.ToolCalls[0].Args)
	}
	// Exactly one Done chunk.
	doneCount := 0
	var content, reasoning string
	for _, c := range got {
		if c.Done {
			doneCount++
		}
		content += c.Content
		reasoning += c.Reasoning
	}
	if doneCount != 1 {
		t.Errorf("Done chunks = %d, want 1", doneCount)
	}
	if content != "Hello world" {
		t.Errorf("content = %q, want %q", content, "Hello world")
	}
	if reasoning != "thinking" {
		t.Errorf("reasoning = %q, want %q", reasoning, "thinking")
	}
}

func TestProcessor(t *testing.T) {
	t.Setenv("BALAUR_PROCESSOR", "")
	if got := Processor(); got != "cpu" {
		t.Errorf("default Processor() = %q, want cpu", got)
	}
	t.Setenv("BALAUR_PROCESSOR", "vulkan")
	if got := Processor(); got != "vulkan" {
		t.Errorf("Processor() = %q, want vulkan", got)
	}
}

func TestResolveLibDirInvalidProcessor(t *testing.T) {
	// An invalid processor errors before any filesystem resolution.
	if _, err := resolveLibDir("", "bogus"); err == nil {
		t.Fatal("expected error for an invalid processor")
	}
}

func TestBridgeError(t *testing.T) {
	reason := model.FinishReasonError
	src := streamOf(
		delta(model.ResponseMessage{Content: "partial"}),
		model.ChatResponse{Choices: []model.Choice{{FinishReasonPtr: &reason}}},
	)
	got := drain(t, bridge(context.Background(), src, func() {}))
	last := got[len(got)-1]
	if last.Err == nil {
		t.Fatalf("expected terminal error chunk, got %+v", last)
	}
	if last.Done {
		t.Errorf("error chunk should not also be Done: %+v", last)
	}
}
