package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/ext"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/storetest"
)

// execute runs one CLI command tree against a test app and returns parsed
// stdout JSON. The storetest app is already migrated, so RunAllMigrations
// inside run() is an idempotent no-op — the same path production takes.
func execute(t *testing.T, cmd *cobra.Command, args ...string) (map[string]any, error) {
	t.Helper()
	v, err := executeRaw(t, cmd, args...)
	if err != nil {
		return nil, err
	}
	obj, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got %T", v)
	}
	return obj, nil
}

func executeList(t *testing.T, cmd *cobra.Command, args ...string) ([]any, error) {
	t.Helper()
	v, err := executeRaw(t, cmd, args...)
	if err != nil {
		return nil, err
	}
	list, ok := v.([]any)
	if !ok {
		t.Fatalf("want a JSON array, got %T", v)
	}
	return list, nil
}

func executeRaw(t *testing.T, cmd *cobra.Command, args ...string) (any, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	execErr := cmd.Execute()
	if execErr != nil {
		// The failure contract: stderr carries a JSON error object.
		var e map[string]any
		if err := json.Unmarshal(errOut.Bytes(), &e); err != nil {
			t.Fatalf("stderr is not JSON on failure: %q", errOut.String())
		}
		return nil, execErr
	}
	var v any
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out.String())
	}
	return v, nil
}

func TestTaskLifecycle(t *testing.T) {
	app := storetest.NewApp(t)

	added, err := execute(t, taskCmd(app), "add", "--title", "Call the notary", "--due", "2027-01-15T10:00", "--notes", "papers")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	id, _ := added["id"].(string)
	if id == "" || added["status"] != "open" {
		t.Fatalf("unexpected add output: %v", added)
	}

	listed, err := execute(t, taskCmd(app), "list", "--scope", "open")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	upcoming, _ := listed["upcoming"].([]any)
	if len(upcoming) != 1 {
		t.Fatalf("want the task in upcoming, got %v", listed)
	}

	done, err := execute(t, taskCmd(app), "done", id)
	if err != nil {
		t.Fatalf("done: %v", err)
	}
	if done["recurring"] != false {
		t.Errorf("one-off must not report recurring: %v", done)
	}
	rec, err := app.FindRecordById("tasks", id)
	if err != nil || rec.GetString("status") != "done" {
		t.Errorf("task not done in the database: %v %v", rec, err)
	}

	// The deed left its audit trail — the CLI inherits the domain layer's
	// auditing, no parallel path.
	audits, err := executeList(t, auditCmd(app), "--action", "task.")
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(audits) < 2 {
		t.Errorf("want create+done audit rows, got %d", len(audits))
	}
}

func TestTaskAddRejectsBadDue(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := execute(t, taskCmd(app), "add", "--title", "x", "--due", "not-a-time"); err == nil {
		t.Fatal("bad --due must fail")
	}
}

func TestMemoryConsentLifecycle(t *testing.T) {
	app := storetest.NewApp(t)

	prop, err := execute(t, memoryCmd(app), "propose",
		"--title", "Prefers tea", "--content", "The owner drinks tea, not coffee.", "--category", "preference", "--importance", "4")
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if prop["status"] != "proposed" {
		t.Fatalf("proposal must start proposed: %v", prop)
	}
	id := prop["id"].(string)

	// Not yet recallable: the consent boundary holds for the CLI too.
	hits, err := executeList(t, memoryCmd(app), "recall", "tea")
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("proposed memory must not be recallable, got %d", len(hits))
	}

	if _, err := execute(t, memoryCmd(app), "approve", id); err != nil {
		t.Fatalf("approve: %v", err)
	}
	hits, err = executeList(t, memoryCmd(app), "recall", "tea")
	if err != nil || len(hits) != 1 {
		t.Fatalf("approved memory must be recallable, got %d (err %v)", len(hits), err)
	}

	// Rejected proposals stay rejected: lifecycle rules apply unchanged.
	if _, err := execute(t, memoryCmd(app), "approve", id); err == nil {
		t.Error("active → active must be rejected by the lifecycle")
	}
}

func TestSkillShowLoadsActiveOnly(t *testing.T) {
	app := storetest.NewApp(t)
	prop, err := execute(t, skillCmd(app), "propose",
		"--name", "weekly-review", "--description", "Run the weekly review", "--content", "1. Look back.\n2. Look ahead.")
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := execute(t, skillCmd(app), "show", "weekly-review"); err == nil {
		t.Fatal("show must refuse a proposed skill")
	}
	if _, err := execute(t, skillCmd(app), "approve", prop["id"].(string)); err != nil {
		t.Fatalf("approve: %v", err)
	}
	shown, err := execute(t, skillCmd(app), "show", "weekly-review")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(shown["content"].(string), "Look back") {
		t.Errorf("show must include content: %v", shown)
	}
}

func TestLifeLogSeriesAndJournal(t *testing.T) {
	app := storetest.NewApp(t)

	if _, err := execute(t, lifeCmd(app), "log", "--kind", "weight", "--value", "82.5", "--unit", "kg"); err != nil {
		t.Fatalf("log: %v", err)
	}
	if _, err := execute(t, lifeCmd(app), "log", "--kind", "weight", "--value", "82.1", "--unit", "kg"); err != nil {
		t.Fatalf("log: %v", err)
	}

	series, err := execute(t, lifeCmd(app), "series", "--kind", "weight")
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	entries := series["entries"].([]any)
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	summary := series["summary"].(map[string]any)
	if summary["last"].(float64) != 82.1 {
		t.Errorf("summary.last = %v, want 82.1", summary["last"])
	}

	kinds, err := executeList(t, lifeCmd(app), "kinds")
	if err != nil || len(kinds) != 1 {
		t.Fatalf("kinds: %v (err %v)", kinds, err)
	}

	if _, err := execute(t, journalCmd(app), "write", "Astăzi a fost o zi bună.", "--noted-at", "2026-06-10"); err != nil {
		t.Fatalf("journal write: %v", err)
	}
	dayOut, err := execute(t, dayCmd(app), "2026-06-10")
	if err != nil {
		t.Fatalf("day: %v", err)
	}
	journal := dayOut["journal"].([]any)
	if len(journal) != 1 {
		t.Fatalf("day must show the journal entry, got %v", dayOut)
	}
	text := journal[0].(map[string]any)["text"].(string)
	if text != "Astăzi a fost o zi bună." {
		t.Errorf("journal must keep the owner's words verbatim, got %q", text)
	}
}

// scriptedClient drives the chat command without a real model.
type scriptedClient struct {
	mu      sync.Mutex
	replies []scriptedReply
}

type scriptedReply struct {
	text  string
	calls []llm.ToolCall
}

func (f *scriptedClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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

func (f *scriptedClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

func withScriptedClient(t *testing.T, c llm.Client) {
	t.Helper()
	prev := chatClients
	chatClients = func(core.App) (llm.Client, error) { return c, nil }
	t.Cleanup(func() { chatClients = prev })
}

func TestChatReportsToolsAndVerdict(t *testing.T) {
	app := storetest.NewApp(t)
	withScriptedClient(t, &scriptedClient{replies: []scriptedReply{
		{calls: []llm.ToolCall{{ID: "c1", Name: "task_add", Args: `{"title":"Water the plants","due":"2027-03-01"}`}}},
		{text: "I've added watering the plants for March 1."},
	}})

	out, err := execute(t, chatCmd(app), "remind me to water the plants on march 1")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if out["capture_succeeded"] != true || out["claims_capture"] != true {
		t.Errorf("verdict fields wrong: %v", out)
	}
	tools := out["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("want one tool event, got %v", tools)
	}
	ev := tools[0].(map[string]any)
	if ev["tool"] != "task_add" || ev["is_error"] != false || !strings.Contains(ev["result"].(string), "Task saved") {
		t.Errorf("tool event wrong: %v", ev)
	}
	if strings.Contains(ev["result"].(string), "balaur-proposal") {
		t.Errorf("proposal marker must not leak into result text: %q", ev["result"])
	}
	prop, _ := ev["proposal"].(map[string]any)
	if prop == nil || prop["kind"] != "tasks" || prop["id"] == "" {
		t.Errorf("proposal reference missing: %v", ev)
	}

	// The chat turn is now on the record and the deterministic verify
	// command agrees with the live verdict.
	verdict, err := execute(t, verifyCmd(app))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verdict["honest"] != true || verdict["capture_succeeded"] != true {
		t.Errorf("verify disagrees with the live turn: %v", verdict)
	}
}

func TestVerifyFlagsUnbackedClaim(t *testing.T) {
	app := storetest.NewApp(t)
	withScriptedClient(t, &scriptedClient{replies: []scriptedReply{
		{text: "I've set the reminder for tomorrow morning."}, // lie
		{text: "It is already set."},                          // repair pass lies again
	}})

	out, err := execute(t, chatCmd(app), "remind me tomorrow")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if out["check_note"] == "" {
		t.Errorf("unrepaired claim must carry the check note: %v", out)
	}

	verdict, err := execute(t, verifyCmd(app))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verdict["claims_capture"] != true || verdict["capture_succeeded"] != false {
		t.Errorf("verify must flag the unbacked claim: %v", verdict)
	}
	if verdict["check_noted"] != true {
		t.Errorf("the runtime's note must be visible to verify: %v", verdict)
	}
	if verdict["honest"] != false {
		t.Errorf("honest must be false on words without deeds: %v", verdict)
	}
}

func TestHistoryReadsPersistedTurn(t *testing.T) {
	app := storetest.NewApp(t)
	withScriptedClient(t, &scriptedClient{replies: []scriptedReply{
		{text: "Hello. Quiet day on the book."},
	}})
	if _, err := execute(t, chatCmd(app), "hello"); err != nil {
		t.Fatalf("chat: %v", err)
	}
	msgs, err := executeList(t, historyCmd(app), "--limit", "10")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("want user+assistant rows, got %d", len(msgs))
	}
	first := msgs[0].(map[string]any)
	if first["role"] != "user" || first["content"] != "hello" {
		t.Errorf("unexpected first row: %v", first)
	}
}

func TestSelfReportsInventory(t *testing.T) {
	app := storetest.NewApp(t)
	t.Setenv("BALAUR_OS_ACCESS", "")
	t.Setenv("BALAUR_SOURCE", "")

	out, err := execute(t, selfCmd(app))
	if err != nil {
		t.Fatalf("self: %v", err)
	}
	tools, _ := out["tools"].([]any)
	names := map[string]bool{}
	for _, n := range tools {
		names[n.(string)] = true
	}
	if !names["self"] || !names["task_add"] || names["bash"] {
		t.Errorf("inventory tools wrong: %v", tools)
	}
	gates := out["gates"].(map[string]any)
	if gates["os_access"] != false {
		t.Errorf("os_access gate must be off: %v", gates)
	}
	src := out["source"].(map[string]any)
	if src["ok"] != false {
		t.Errorf("source seam must report not ok on a bare box: %v", src)
	}

	sect, err := execute(t, selfCmd(app), "--section", "devloop")
	if err != nil {
		t.Fatalf("self --section: %v", err)
	}
	content := sect["section"].(map[string]any)["content"].(string)
	if !strings.Contains(content, "go test") {
		t.Errorf("devloop section missing the deeds: %.80s", content)
	}
}

func TestExtLifecycleViaCLI(t *testing.T) {
	t.Setenv("BALAUR_EXT_DIR", filepath.Join(t.TempDir(), "pb_extensions"))
	app := storetest.NewApp(t)
	dir := ext.Dir(app)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "// balaur-extension: cli greeter\n" +
		"balaur.registerTool({name: \"greet\", description: \"d\", parameters: {type:\"object\"}, handler: function(a){return \"hi\"}})\n"
	if err := os.WriteFile(filepath.Join(dir, "greet.js"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	listed, err := executeList(t, extCmd(app), "list")
	if err != nil || len(listed) != 1 {
		t.Fatalf("list must discover the file: %v (err %v)", listed, err)
	}
	if listed[0].(map[string]any)["status"] != "proposed" {
		t.Fatalf("discovery must propose: %v", listed[0])
	}

	approved, err := execute(t, extCmd(app), "approve", "greet")
	if err != nil || approved["status"] != "active" {
		t.Fatalf("approve: %v (err %v)", approved, err)
	}
	tools := approved["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["name"] != "greet" {
		t.Errorf("approve must record the extracted tools: %v", approved["tools"])
	}

	shown, err := execute(t, extCmd(app), "show", "greet")
	if err != nil || !strings.Contains(shown["code"].(string), "registerTool") {
		t.Errorf("show must include the code: %v", err)
	}

	disabled, err := execute(t, extCmd(app), "disable", "greet")
	if err != nil || disabled["status"] != "disabled" {
		t.Errorf("disable: %v (err %v)", disabled, err)
	}
}

func TestModelReportsNoModelOnBareBox(t *testing.T) {
	app := storetest.NewApp(t)
	out, err := execute(t, modelCmd(app))
	if err != nil {
		t.Fatalf("model: %v", err)
	}
	if out["chat_ready"] != false {
		t.Errorf("bare box must not be chat_ready: %v", out)
	}
}

func TestRecapShowFindsNothingOnFreshBox(t *testing.T) {
	app := storetest.NewApp(t)
	out, err := execute(t, recapCmd(app), "show", "--period", "day", "--date", "2026-06-10")
	if err != nil {
		t.Fatalf("recap show: %v", err)
	}
	if out["found"] != false {
		t.Errorf("fresh box has no summaries: %v", out)
	}
}
