package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/ext"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/storetest"
)

// executeEnvelope runs a command and returns the raw v1 envelope from stdout.
// It also asserts that the envelope is structurally valid: v==1, kind non-empty.
func executeEnvelope(t *testing.T, cmd *cobra.Command, args ...string) (map[string]any, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	execErr := cmd.Execute()
	if execErr != nil {
		// The failure contract: stderr carries a v1 error envelope.
		var e map[string]any
		if err := json.Unmarshal(errOut.Bytes(), &e); err != nil {
			t.Fatalf("stderr is not JSON on failure: %q", errOut.String())
		}
		if v, _ := e["v"].(float64); v != 1 {
			t.Errorf("error envelope must have v:1, got %v", e["v"])
		}
		if e["kind"] != "error" {
			t.Errorf("error envelope must have kind:error, got %v", e["kind"])
		}
		return e, execErr
	}
	var env map[string]any
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out.String())
	}
	if v, _ := env["v"].(float64); v != 1 {
		t.Errorf("envelope must have v:1, got %v", env["v"])
	}
	if k, _ := env["kind"].(string); k == "" {
		t.Errorf("envelope must have a non-empty kind, got %v", env["kind"])
	}
	if _, hasData := env["data"]; !hasData {
		t.Errorf("envelope must have a data field")
	}
	return env, nil
}

// execute runs one CLI command tree against a test app and returns the data
// field of the v1 envelope, parsed as a JSON object. The storetest app is
// already migrated, so RunAllMigrations inside run() is a no-op.
func execute(t *testing.T, cmd *cobra.Command, args ...string) (map[string]any, error) {
	t.Helper()
	env, err := executeEnvelope(t, cmd, args...)
	if err != nil {
		return nil, err
	}
	obj, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("want envelope.data to be a JSON object, got %T", env["data"])
	}
	return obj, nil
}

func executeList(t *testing.T, cmd *cobra.Command, args ...string) ([]any, error) {
	t.Helper()
	env, err := executeEnvelope(t, cmd, args...)
	if err != nil {
		return nil, err
	}
	list, ok := env["data"].([]any)
	if !ok {
		t.Fatalf("want envelope.data to be a JSON array, got %T", env["data"])
	}
	return list, nil
}

// executeRaw is kept for the envelope family test: returns the parsed data
// field (any) without asserting its type.
func executeRaw(t *testing.T, cmd *cobra.Command, args ...string) (any, error) {
	t.Helper()
	env, err := executeEnvelope(t, cmd, args...)
	if err != nil {
		return nil, err
	}
	return env["data"], nil
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

func withScriptedClient(t *testing.T, c llm.Client) {
	t.Helper()
	prev := chatClients
	chatClients = func(core.App) (llm.Client, error) { return c, nil }
	t.Cleanup(func() { chatClients = prev })
}

func TestChatReportsToolsAndVerdict(t *testing.T) {
	app := storetest.NewApp(t)
	withScriptedClient(t, llmtest.New(
		llmtest.ToolCall("c1", "task_add", `{"title":"Water the plants","due":"2027-03-01"}`),
		llmtest.Text("I've added watering the plants for March 1."),
	))

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
	withScriptedClient(t, llmtest.New(
		llmtest.Text("I've set the reminder for tomorrow morning."), // lie
		llmtest.Text("It is already set."),                          // repair pass lies again
	))

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
	withScriptedClient(t, llmtest.New(
		llmtest.Text("Hello. Quiet day on the book."),
	))
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

// TestEnvelopeFamilies proves that each output family (object, array, error)
// emits the v1 envelope with v:1, the correct kind, and the prior value
// nested under data. Assertions unmarshal — they do not string-match.
func TestEnvelopeFamilies(t *testing.T) {
	t.Run("object kind (model)", func(t *testing.T) {
		app := storetest.NewApp(t)
		env, err := executeEnvelope(t, modelCmd(app))
		if err != nil {
			t.Fatalf("model: %v", err)
		}
		if env["kind"] != "model" {
			t.Errorf("kind: want model, got %v", env["kind"])
		}
		data, ok := env["data"].(map[string]any)
		if !ok {
			t.Fatalf("data must be an object, got %T", env["data"])
		}
		if _, has := data["chat_ready"]; !has {
			t.Errorf("data must contain chat_ready, got %v", data)
		}
	})

	t.Run("array kind (audit)", func(t *testing.T) {
		app := storetest.NewApp(t)
		env, err := executeEnvelope(t, auditCmd(app))
		if err != nil {
			t.Fatalf("audit: %v", err)
		}
		if env["kind"] != "audit" {
			t.Errorf("kind: want audit, got %v", env["kind"])
		}
		if _, ok := env["data"].([]any); !ok {
			t.Fatalf("data must be an array for audit, got %T", env["data"])
		}
	})

	t.Run("error envelope on failure", func(t *testing.T) {
		app := storetest.NewApp(t)
		// A bad --due triggers a well-defined failure path.
		env, err := executeEnvelope(t, taskCmd(app), "add", "--title", "x", "--due", "not-a-time")
		if err == nil {
			t.Fatal("bad --due must fail")
		}
		if env["kind"] != "error" {
			t.Errorf("error kind: want error, got %v", env["kind"])
		}
		data, ok := env["data"].(map[string]any)
		if !ok || data["error"] == "" {
			t.Errorf("error data must have error field: %v", env["data"])
		}
	})
}

// TestDoctorHealthyBox verifies that a fresh test app passes all fatal
// checks and exits with code 0 (top-level ok:true).
func TestDoctorHealthyBox(t *testing.T) {
	app := storetest.NewApp(t)
	exitCode.Store(0) // reset any residual state
	out, err := execute(t, doctorCmd(app))
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if out["ok"] != true {
		t.Errorf("healthy box must have top-level ok:true, got %v", out)
	}
	checks, _ := out["checks"].([]any)
	if len(checks) == 0 {
		t.Fatal("doctor must return at least one check")
	}
	// data_dir_writable and collections_present must both pass on a fresh box.
	names := map[string]bool{}
	for _, ch := range checks {
		m := ch.(map[string]any)
		names[m["name"].(string)] = m["ok"].(bool)
	}
	if !names["data_dir_writable"] {
		t.Error("data_dir_writable must be ok on a fresh box")
	}
	if !names["collections_present"] {
		t.Error("collections_present must be ok on a fresh box")
	}
	// model_ready is expected false on a bare box, but must not block top-level ok.
	if out["ok"] != true {
		t.Errorf("top-level ok must be true even when model_ready is false")
	}
	// version block must be present.
	if _, hasVersion := out["version"]; !hasVersion {
		t.Error("doctor must include a version block")
	}
}

// TestDoctorMissingCollectionFails injects a deliberately absent collection
// name into the check list and confirms top-level ok becomes false and the
// command exits non-zero.
func TestDoctorMissingCollectionFails(t *testing.T) {
	app := storetest.NewApp(t)
	exitCode.Store(0)

	// Inject a checker list with a non-existent collection.
	prev := doctorCheckers
	doctorCheckers = func(a core.App) []doctorCheck {
		return []doctorCheck{
			checkDataDir(a),
			checkCollections(a, []string{"messages", "definitely_does_not_exist"}),
			checkOSAccess(),
		}
	}
	t.Cleanup(func() { doctorCheckers = prev })

	env, err := executeEnvelope(t, doctorCmd(app))
	// doctor itself should not return a RunE error — it uses exitCode instead.
	if err != nil {
		t.Fatalf("doctor RunE must not error: %v", err)
	}
	data := env["data"].(map[string]any)
	if data["ok"] != false {
		t.Errorf("missing collection must make top-level ok:false, got %v", data["ok"])
	}
	if int(exitCode.Load()) != 1 {
		t.Errorf("exit code must be 1 when top-level ok is false, got %d", exitCode.Load())
	}
}

// TestDoctorModelReadyNonFatal verifies that a box without a model has
// ok:true (model_ready is non-fatal) but model_ready.ok is false.
func TestDoctorModelReadyNonFatal(t *testing.T) {
	app := storetest.NewApp(t)
	exitCode.Store(0)
	out, err := execute(t, doctorCmd(app))
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if out["ok"] != true {
		t.Errorf("top-level ok must be true when only non-fatal checks fail: %v", out)
	}
	checks, _ := out["checks"].([]any)
	var modelReady map[string]any
	for _, ch := range checks {
		m := ch.(map[string]any)
		if m["name"] == "model_ready" {
			modelReady = m
			break
		}
	}
	if modelReady == nil {
		t.Fatal("model_ready check must be present")
	}
	if modelReady["ok"] != false {
		t.Errorf("model_ready must be false on a bare box: %v", modelReady)
	}
	if modelReady["fatal"] != false {
		t.Errorf("model_ready must not be fatal: %v", modelReady)
	}
}
