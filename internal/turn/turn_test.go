package turn

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/storetest"
	"github.com/alexradunet/balaur/internal/verify"
)

// fakeClient replays scripted replies, one per ChatStream call. Tests
// never hit a real model (AGENTS.md).
type fakeClient struct {
	mu      sync.Mutex
	replies []fakeReply
	calls   int
}

type fakeReply struct {
	text  string
	calls []llm.ToolCall
}

func (f *fakeClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	ch := make(chan llm.Chunk, 2)
	if len(f.replies) == 0 {
		ch <- llm.Chunk{Done: true}
		close(ch)
		return ch, nil
	}
	r := f.replies[0]
	f.replies = f.replies[1:]
	if r.text != "" {
		ch <- llm.Chunk{Content: r.text}
	}
	ch <- llm.Chunk{Done: true, ToolCalls: r.calls}
	close(ch)
	return ch, nil
}

func (f *fakeClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

func TestRunPersistsHonestCaptureTurn(t *testing.T) {
	app := storetest.NewApp(t)
	client := &fakeClient{replies: []fakeReply{
		{calls: []llm.ToolCall{{ID: "c1", Name: "task_add", Args: `{"title":"Call the notary","due":"2026-06-12T10:00"}`}}},
		{text: "I've added the notary call for tomorrow at 10."},
	}}

	var kinds []string
	emit := func(ev agent.Event) { kinds = append(kinds, ev.Kind) }
	res, err := Run(context.Background(), app, client, "remind me to call the notary tomorrow at 10", emit)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(res.Reply, "notary") {
		t.Errorf("reply does not mention the task: %q", res.Reply)
	}
	if res.CheckNote != "" {
		t.Errorf("honest capture must not be noted, got %q", res.CheckNote)
	}

	// The deed happened: the task row exists.
	tasks, err := app.FindRecordsByFilter("tasks", "title = 'Call the notary'", "", 2, 0)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("want exactly one task, got %d (err %v)", len(tasks), err)
	}

	// The whole turn persisted: user, assistant tool round, tool result,
	// final assistant text.
	msgs, err := app.FindRecordsByFilter("messages", "id != ''", "@rowid", 0, 0)
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	var roles []string
	for _, m := range msgs {
		roles = append(roles, m.GetString("role"))
	}
	want := []string{"user", "assistant", "tool", "assistant"}
	if strings.Join(roles, ",") != strings.Join(want, ",") {
		t.Errorf("persisted roles = %v, want %v", roles, want)
	}
	if name := msgs[2].GetString("tool_name"); name != "task_add" {
		t.Errorf("tool row records name %q, want task_add", name)
	}

	// Events streamed for the gateway to render.
	joined := strings.Join(kinds, ",")
	for _, k := range []string{"tool_start", "tool_result", "text", "done"} {
		if !strings.Contains(joined, k) {
			t.Errorf("emit missed %q event (got %s)", k, joined)
		}
	}
}

func TestRunNotesUnbackedCaptureClaim(t *testing.T) {
	app := storetest.NewApp(t)
	client := &fakeClient{replies: []fakeReply{
		{text: "I've set the reminder for tomorrow morning."}, // claim, no deed
		{text: "It is already set."},                          // repair pass still lies
	}}

	res, err := Run(context.Background(), app, client, "remind me tomorrow", nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.CheckNote != verify.Note {
		t.Fatalf("check note = %q, want verify.Note", res.CheckNote)
	}
	if client.calls != 2 {
		t.Errorf("repair pass should run exactly once: %d model calls", client.calls)
	}

	// The note is on the record with its origin, so the owner sees it in
	// history too — and the correction scaffolding is NOT persisted.
	notes, err := app.FindRecordsByFilter("messages", "origin = 'check'", "", 0, 0)
	if err != nil || len(notes) != 1 {
		t.Fatalf("want one check note persisted, got %d (err %v)", len(notes), err)
	}
	scaffold, err := app.FindRecordsByFilter("messages",
		"role = 'user' && content ~ 'runtime check'", "", 0, 0)
	if err != nil || len(scaffold) != 0 {
		t.Errorf("verify.Correction must never persist, found %d rows", len(scaffold))
	}
}

func TestToolsRespectsOSAccessGate(t *testing.T) {
	app := storetest.NewApp(t)
	names := func() map[string]bool {
		out := map[string]bool{}
		for _, tool := range Tools(app) {
			out[tool.Spec.Name] = true
		}
		return out
	}

	got := names()
	for _, want := range []string{"task_add", "remember", "log_entry", "journal_write", "self"} {
		if !got[want] {
			t.Errorf("default tool set missing %q", want)
		}
	}
	if got["bash"] {
		t.Error("OS tools must stay off without BALAUR_OS_ACCESS=1")
	}

	t.Setenv("BALAUR_OS_ACCESS", "1")
	if !names()["bash"] {
		t.Error("BALAUR_OS_ACCESS=1 must enable the OS tools")
	}
}

func TestMaxStepsEnvRaisesTheCap(t *testing.T) {
	app := storetest.NewApp(t)
	t.Setenv("BALAUR_MAX_STEPS", "1")
	// Two tool rounds scripted against a cap of one: the loop must stop
	// after the first round with the exceeded error.
	client := &fakeClient{replies: []fakeReply{
		{calls: []llm.ToolCall{{ID: "c1", Name: "task_list", Args: `{}`}}},
		{calls: []llm.ToolCall{{ID: "c2", Name: "task_list", Args: `{}`}}},
		{text: "never reached"},
	}}
	_, err := Run(context.Background(), app, client, "list everything twice", nil)
	if err == nil || !strings.Contains(err.Error(), "exceeded 1 tool rounds") {
		t.Errorf("cap of 1 must trip the loop, got %v", err)
	}
}

func TestNowLineGroundsTheMoment(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Bucharest")
	if err != nil {
		t.Skipf("tzdata: %v", err)
	}
	now := time.Date(2026, 6, 11, 14, 32, 0, 0, loc)
	line := nowLine(now)
	for _, want := range []string{"Thursday, June 11 2026", "14:32", "UTC+03:00", "this moment"} {
		if !strings.Contains(line, want) {
			t.Errorf("now line missing %q in: %s", want, line)
		}
	}
}
