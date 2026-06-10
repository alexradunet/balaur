package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/alexradunet/balaur/internal/llm"
)

// fakeClient scripts a sequence of turns: each call to ChatStream pops the
// next scripted reply. Tests never touch a real model (AGENTS.md).
type fakeClient struct {
	turns []fakeTurn
	calls int
}

type fakeTurn struct {
	text  string
	tools []llm.ToolCall
	err   error
}

func (f *fakeClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	if f.calls >= len(f.turns) {
		return nil, errors.New("fake: no more scripted turns")
	}
	turn := f.turns[f.calls]
	f.calls++

	ch := make(chan llm.Chunk, 4)
	go func() {
		defer close(ch)
		if turn.err != nil {
			ch <- llm.Chunk{Err: turn.err}
			return
		}
		if turn.text != "" {
			ch <- llm.Chunk{Content: turn.text}
		}
		ch <- llm.Chunk{Done: true, ToolCalls: turn.tools}
	}()
	return ch, nil
}

func (f *fakeClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, errors.New("fake: no embeddings")
}

func collect(events *[]Event) func(Event) {
	return func(e Event) { *events = append(*events, e) }
}

func TestRunPlainAnswer(t *testing.T) {
	loop := &Loop{Client: &fakeClient{turns: []fakeTurn{{text: "hello"}}}}

	var events []Event
	msgs, err := loop.Run(context.Background(), []llm.Message{{Role: "user", Content: "hi"}}, collect(&events))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	last := msgs[len(msgs)-1]
	if last.Role != "assistant" || last.Content != "hello" {
		t.Fatalf("unexpected final message: %+v", last)
	}
	if events[len(events)-1].Kind != "done" {
		t.Fatalf("expected done event, got %+v", events[len(events)-1])
	}
}

func TestRunToolRound(t *testing.T) {
	client := &fakeClient{turns: []fakeTurn{
		{tools: []llm.ToolCall{{ID: "c1", Name: "echo", Args: `{"s":"ping"}`}}},
		{text: "the tool said ping"},
	}}
	loop := &Loop{
		Client: client,
		Tools: []Tool{{
			Spec: llm.ToolSpec{Name: "echo"},
			Execute: func(ctx context.Context, args string) (string, error) {
				return "ping", nil
			},
		}},
	}

	var events []Event
	msgs, err := loop.Run(context.Background(), nil, collect(&events))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var sawStart, sawResult bool
	for _, e := range events {
		if e.Kind == "tool_start" && e.Tool == "echo" {
			sawStart = true
		}
		if e.Kind == "tool_result" && e.Text == "ping" {
			sawResult = true
		}
	}
	if !sawStart || !sawResult {
		t.Fatalf("missing tool events: %+v", events)
	}

	// history: assistant(tool_calls) → tool → assistant(final)
	if len(msgs) != 3 || msgs[1].Role != "tool" || msgs[1].ToolCallID != "c1" {
		t.Fatalf("unexpected history: %+v", msgs)
	}
}

func TestRunUnknownToolFeedsErrorBack(t *testing.T) {
	client := &fakeClient{turns: []fakeTurn{
		{tools: []llm.ToolCall{{ID: "c1", Name: "missing", Args: `{}`}}},
		{text: "recovered"},
	}}
	loop := &Loop{Client: client}

	var events []Event
	msgs, err := loop.Run(context.Background(), nil, collect(&events))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if msgs[1].Role != "tool" || msgs[1].Content == "" {
		t.Fatalf("expected error text in tool turn, got %+v", msgs[1])
	}
}

func TestRunStepLimit(t *testing.T) {
	// Model calls a tool forever; the loop must stop at MaxSteps.
	turns := make([]fakeTurn, 3)
	for i := range turns {
		turns[i] = fakeTurn{tools: []llm.ToolCall{{ID: "x", Name: "echo", Args: `{}`}}}
	}
	loop := &Loop{
		Client:   &fakeClient{turns: turns},
		MaxSteps: 2,
		Tools: []Tool{{
			Spec:    llm.ToolSpec{Name: "echo"},
			Execute: func(ctx context.Context, args string) (string, error) { return "ok", nil },
		}},
	}

	var events []Event
	_, err := loop.Run(context.Background(), nil, collect(&events))
	if err == nil {
		t.Fatal("expected step-limit error")
	}
}
