package turn

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/ext"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/storetest"
	"github.com/alexradunet/balaur/internal/verify"
)

func TestRunPersistsHonestCaptureTurn(t *testing.T) {
	app := storetest.NewApp(t)
	client := llmtest.New(
		llmtest.ToolCall("c1", "task_add", `{"title":"Call the notary","due":"2026-06-12T10:00"}`),
		llmtest.Text("I've added the notary call for tomorrow at 10."),
	)

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
	client := llmtest.New(
		llmtest.Text("I've set the reminder for tomorrow morning."), // claim, no deed
		llmtest.Text("It is already set."),                          // repair pass still lies
	)

	res, err := Run(context.Background(), app, client, "remind me tomorrow", nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.CheckNote != verify.Note {
		t.Fatalf("check note = %q, want verify.Note", res.CheckNote)
	}
	if client.Calls != 2 {
		t.Errorf("repair pass should run exactly once: %d model calls", client.Calls)
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
	// Hermetic against the session: when Balaur runs this suite through
	// its own bash tool during self-development, BALAUR_OS_ACCESS=1 is in
	// the inherited env — the default-off assertion must not trust ambient
	// state (found by the first live devloop rehearsal).
	t.Setenv("BALAUR_OS_ACCESS", "")
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

func TestApprovedExtensionJoinsEveryGateway(t *testing.T) {
	t.Setenv("BALAUR_EXT_DIR", filepath.Join(t.TempDir(), "pb_extensions"))
	app := storetest.NewApp(t)
	dir := ext.Dir(app)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "// balaur-extension: test greeter\n" +
		"balaur.registerTool({name: \"greet\", description: \"d\", parameters: {type:\"object\"}, handler: function(a){return \"hi\"}})\n"
	if err := os.WriteFile(filepath.Join(dir, "greet.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	names := func() map[string]bool {
		out := map[string]bool{}
		for _, tool := range Tools(app) {
			out[tool.Spec.Name] = true
		}
		return out
	}
	if names()["greet"] {
		t.Fatal("a proposed extension must not join the turn")
	}
	if !names()["propose_extension"] {
		t.Fatal("propose_extension must always be available")
	}
	if _, err := ext.Approve(app, "greet"); err != nil {
		t.Fatal(err)
	}
	if !names()["greet"] {
		t.Fatal("an approved extension must join the turn's tool set")
	}
}

func TestMaxStepsEnvRaisesTheCap(t *testing.T) {
	app := storetest.NewApp(t)
	t.Setenv("BALAUR_MAX_STEPS", "1")
	// Two tool rounds scripted against a cap of one: the loop must stop
	// after the first round with the exceeded error.
	client := llmtest.New(
		llmtest.ToolCall("c1", "task_list", `{}`),
		llmtest.ToolCall("c2", "task_list", `{}`),
		llmtest.Text("never reached"),
	)
	_, err := Run(context.Background(), app, client, "list everything twice", nil)
	if err == nil || !strings.Contains(err.Error(), "exceeded 1 tool rounds") {
		t.Errorf("cap of 1 must trip the loop, got %v", err)
	}
}

// seedHeadForTurn creates an active head record for turn tests.
func seedHeadForTurn(t *testing.T, app core.App, name string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		t.Fatalf("heads collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("status", "active")
	rec.SetEmail(fmt.Sprintf("head-%d@balaur.local", time.Now().UnixNano()))
	rec.SetRandomPassword()
	if err := app.Save(rec); err != nil {
		t.Fatalf("saving head: %v", err)
	}
	return rec
}

func TestRunForPersistsFocusedTurn(t *testing.T) {
	app := storetest.NewApp(t)
	head := seedHeadForTurn(t, app, "Scout")

	conv, err := conversation.ForHead(app, head)
	if err != nil {
		t.Fatalf("ForHead: %v", err)
	}
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("Master: %v", err)
	}

	client := llmtest.New(llmtest.Text("On it."))

	var eventKinds []string
	emit := func(ev agent.Event) { eventKinds = append(eventKinds, ev.Kind) }

	res, err := RunFor(context.Background(), app, client, conv, "Scout", "find flights", "hello", emit)
	if err != nil {
		t.Fatalf("RunFor: %v", err)
	}
	if res.Reply != "On it." {
		t.Errorf("reply = %q, want %q", res.Reply, "On it.")
	}

	// Persisted messages for the branch conversation: user + assistant.
	msgs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv}", "@rowid", 0, 0,
		map[string]any{"conv": conv.Id})
	if err != nil {
		t.Fatalf("branch messages: %v", err)
	}
	var roles []string
	for _, m := range msgs {
		roles = append(roles, m.GetString("role"))
	}
	wantRoles := []string{"user", "assistant"}
	if strings.Join(roles, ",") != strings.Join(wantRoles, ",") {
		t.Errorf("branch roles = %v, want %v", roles, wantRoles)
	}

	// Master conversation must have gained NO messages.
	masterMsgs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv}", "@rowid", 0, 0,
		map[string]any{"conv": master.Id})
	if err != nil {
		t.Fatalf("master messages: %v", err)
	}
	if len(masterMsgs) != 0 {
		t.Errorf("master gained %d message(s), want 0", len(masterMsgs))
	}

	// emit saw a text event.
	joined := strings.Join(eventKinds, ",")
	if !strings.Contains(joined, "text") {
		t.Errorf("emit missing 'text' event; got: %s", joined)
	}
}

func TestRunForSystemPromptAndNoTools(t *testing.T) {
	app := storetest.NewApp(t)
	head := seedHeadForTurn(t, app, "Scout")

	conv, err := conversation.ForHead(app, head)
	if err != nil {
		t.Fatalf("ForHead: %v", err)
	}

	// Capture messages sent to the model.
	var capturedMsgs []llm.Message
	client := llmtest.New()
	client.Respond = func(msgs []llm.Message) string {
		capturedMsgs = msgs
		return "system prompt captured"
	}

	if _, err := RunFor(context.Background(), app, client, conv, "Scout", "find flights", "go", nil); err != nil {
		t.Fatalf("RunFor: %v", err)
	}

	if len(capturedMsgs) == 0 {
		t.Fatal("no messages sent to model")
	}
	sys := capturedMsgs[0]
	if sys.Role != "system" {
		t.Errorf("first message role = %q, want system", sys.Role)
	}
	if !strings.Contains(sys.Content, "You are Scout") {
		t.Errorf("system prompt missing 'You are Scout': %s", sys.Content)
	}
	if !strings.Contains(sys.Content, "find flights") {
		t.Errorf("system prompt missing purpose 'find flights': %s", sys.Content)
	}
}

func TestRunForNoToolExecution(t *testing.T) {
	// FINDING: Tools: nil means no tool can be found by name. When the model
	// requests a tool call, agent.Loop returns "error: unknown tool <name>" as
	// the tool result and persists a role='tool' message with that error
	// content. The tool itself is never executed (no task record is created),
	// but the loop does produce a persisted tool-error message.
	app := storetest.NewApp(t)
	head := seedHeadForTurn(t, app, "Scout")

	conv, err := conversation.ForHead(app, head)
	if err != nil {
		t.Fatalf("ForHead: %v", err)
	}

	// Script a tool call — the loop has nil tools, so it cannot execute it.
	client := llmtest.New(
		llmtest.ToolCall("c1", "task_add", `{"title":"buy milk","due":"2026-06-13T09:00"}`),
		llmtest.Text("I tried to add a task."),
	)

	_, _ = RunFor(context.Background(), app, client, conv, "Scout", "", "add a task", nil)

	// No task record created — tool body never ran.
	taskRecs, err := app.FindRecordsByFilter("tasks", "title = 'buy milk'", "", 0, 0)
	if err == nil && len(taskRecs) != 0 {
		t.Errorf("task was created despite Tools: nil — found %d task(s)", len(taskRecs))
	}

	// The loop does persist a role='tool' error message (unknown tool result).
	// This characterizes actual behavior: the agent loop replies with an error
	// string rather than silently dropping the call.
	toolMsgs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv} && role = 'tool'", "", 0, 0,
		map[string]any{"conv": conv.Id})
	if err != nil {
		t.Fatalf("querying tool messages: %v", err)
	}
	if len(toolMsgs) == 0 {
		t.Error("expected a tool-error message persisted by the loop, got 0")
	}
	if len(toolMsgs) > 0 {
		content := toolMsgs[0].GetString("content")
		if !strings.Contains(content, "unknown tool") {
			t.Errorf("tool message content = %q, want to contain 'unknown tool'", content)
		}
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
