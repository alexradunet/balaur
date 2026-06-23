package turn

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/ext"
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

	// The fabricated claims are quarantined: persisted (owner still sees them)
	// but tagged OriginUncommitted, so the next turn's context never replays
	// the lie back as a pattern to imitate — the poisoned-thread failure.
	uncommitted, err := app.FindRecordsByFilter("messages",
		"role = 'assistant' && origin = 'uncommitted'", "", 0, 0)
	if err != nil || len(uncommitted) == 0 {
		t.Fatalf("fabricated claims must be tagged uncommitted, got %d (err %v)", len(uncommitted), err)
	}
	master, _ := conversation.Master(app)
	ctxTurns, err := conversation.RecentTurns(app, master.Id, 50)
	if err != nil {
		t.Fatalf("RecentTurns: %v", err)
	}
	var sawUser bool
	for _, m := range ctxTurns {
		if strings.Contains(m.Content, "set the reminder") || strings.Contains(m.Content, "already set") {
			t.Errorf("uncommitted claim leaked into context: %q", m.Content)
		}
		if m.Role == "user" && strings.Contains(m.Content, "remind me tomorrow") {
			sawUser = true
		}
	}
	if !sawUser {
		t.Error("the owner's real message must stay in context")
	}
}

func TestRunRepairPassSucceeds(t *testing.T) {
	// The model first claims capture without a deed, then on the repair pass
	// actually performs the tool call. The honesty check must see the deed and
	// leave CheckNote empty.
	app := storetest.NewApp(t)
	client := llmtest.New(
		llmtest.Text("I've set the reminder for tomorrow morning."), // claim, no deed — triggers repair
		llmtest.ToolCall("c1", "task_add", `{"title":"Call the notary","due":"2026-06-12T10:00"}`),
		llmtest.Text("Saved it for tomorrow."), // repair completes
	)

	res, err := Run(context.Background(), app, client, "remind me to call the notary tomorrow at 10", nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.CheckNote != "" {
		t.Errorf("repair-success must not set CheckNote, got %q", res.CheckNote)
	}
	if client.Calls != 3 {
		t.Errorf("repair pass must run exactly once (3 total calls): %d model calls", client.Calls)
	}

	// The deed happened: the task row exists.
	tasks, err := app.FindRecordsByFilter("tasks", "title = 'Call the notary'", "", 2, 0)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("want exactly one task, got %d (err %v)", len(tasks), err)
	}

	// No check-origin message must be persisted (repair succeeded — no note).
	notes, err := app.FindRecordsByFilter("messages", "origin = 'check'", "", 0, 0)
	if err != nil || len(notes) != 0 {
		t.Errorf("repair-success must not persist any check note, got %d (err %v)", len(notes), err)
	}

	// The correction scaffolding must never be persisted.
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

func TestRunPlainReplyNoCaptureClaim(t *testing.T) {
	app := storetest.NewApp(t)
	client := llmtest.New(llmtest.Text("The capital of France is Paris."))

	res, err := Run(context.Background(), app, client, "what's the capital of France?", nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.CheckNote != "" {
		t.Errorf("a plain reply must not be noted, got %q", res.CheckNote)
	}
	if client.Calls != 1 {
		t.Errorf("a plain reply needs exactly one model call, got %d", client.Calls)
	}
	if !strings.Contains(res.Reply, "Paris") {
		t.Errorf("reply lost: %q", res.Reply)
	}
	// Only user + assistant persist — no tool round, no check note.
	msgs, err := app.FindRecordsByFilter("messages", "id != ''", "@rowid", 0, 0)
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	var roles []string
	for _, m := range msgs {
		roles = append(roles, m.GetString("role"))
	}
	if strings.Join(roles, ",") != "user,assistant" {
		t.Errorf("persisted roles = %v, want [user assistant]", roles)
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
