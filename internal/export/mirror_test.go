package export_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/export"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestMirrorLayoutPerType proves each mapped, non-deferred owner type lands in
// its Johnny Decimal folder, with the rendered file present and well-formed.
func TestMirrorLayoutPerType(t *testing.T) {
	app := storetest.NewApp(t)

	// One active node per mapped, non-deferred owner-authored type, with the JD
	// path each is expected to land at.
	cases := []struct {
		typ   string
		title string
		want  string
	}{
		{"note", "My Note", "10-19 Knowledge/11 Notes/my-note.md"},
		{"idea", "Big Idea", "10-19 Knowledge/12 Ideas/big-idea.md"},
		{"person", "Jane Doe", "20-29 People/21 People/jane-doe.md"},
		{"book", "Some Book", "30-39 Library/31 Books/some-book.md"},
		{"place", "Cozy Cafe", "40-49 Places/41 Places/cozy-cafe.md"},
	}
	for _, c := range cases {
		if _, err := nodes.Create(app, c.typ, c.title, "Body for "+c.title+".",
			nodes.StatusActive, nil); err != nil {
			t.Fatalf("create %s: %v", c.typ, err)
		}
	}

	dir := t.TempDir()
	paths, err := export.ExportMirror(app, dir)
	if err != nil {
		t.Fatalf("export mirror: %v", err)
	}

	got := map[string]bool{}
	for _, p := range paths {
		got[p] = true
	}
	for _, c := range cases {
		if !got[c.want] {
			t.Errorf("type %s: missing returned path %q (got %v)", c.typ, c.want, paths)
			continue
		}
		body, rErr := os.ReadFile(filepath.Join(dir, filepath.FromSlash(c.want)))
		if rErr != nil {
			t.Errorf("type %s: read %s: %v", c.typ, c.want, rErr)
			continue
		}
		text := string(body)
		if !strings.Contains(text, "# "+c.title) {
			t.Errorf("type %s: file missing H1 %q:\n%s", c.typ, c.title, text)
		}
		if !strings.Contains(text, `type: "`+c.typ+`"`) {
			t.Errorf("type %s: file missing type frontmatter:\n%s", c.typ, text)
		}
	}
}

// TestMirrorByteIdenticalReexport proves a second full re-export over unchanged
// data is byte-identical (design Q3) — no time.Now() leaks into render or write.
func TestMirrorByteIdenticalReexport(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "First", "First body.",
		nodes.StatusActive, map[string]any{"tag": "a"}); err != nil {
		t.Fatalf("create note: %v", err)
	}
	if _, err := nodes.Create(app, "idea", "Second", "Second body.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create idea: %v", err)
	}

	dir := t.TempDir()
	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("first export: %v", err)
	}
	first := readAll(t, dir)

	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("second export: %v", err)
	}
	second := readAll(t, dir)

	// Drop git's own bookkeeping (under .git) — it is allowed to differ. The
	// MIRROR files must be byte-identical.
	stripGit := func(m map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range m {
			if strings.HasPrefix(k, ".git/") || k == ".git" {
				continue
			}
			out[k] = v
		}
		return out
	}
	if !reflect.DeepEqual(stripGit(first), stripGit(second)) {
		t.Errorf("re-export not byte-identical:\nfirst:  %v\nsecond: %v",
			stripGit(first), stripGit(second))
	}
}

// TestMirrorUnmappedTypeGoesUnsorted proves an owner-authored type without an
// explicit JD mapping falls back to 90-99 Unsorted/91 Other/. measure is seeded,
// owner-authored, and unmapped — no registry mutation needed.
func TestMirrorUnmappedTypeGoesUnsorted(t *testing.T) {
	app := storetest.NewApp(t)
	// measure requires props kind + noted_at (see the measures migration).
	if _, err := nodes.Create(app, "measure", "Weight Reading", "Logged value.",
		nodes.StatusActive, map[string]any{
			"kind":     "weight",
			"noted_at": "2026-06-25 09:00:00.000Z",
		}); err != nil {
		t.Fatalf("create measure: %v", err)
	}

	dir := t.TempDir()
	paths, err := export.ExportMirror(app, dir)
	if err != nil {
		t.Fatalf("export mirror: %v", err)
	}

	const want = "90-99 Unsorted/91 Other/weight-reading.md"
	found := false
	for _, p := range paths {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Errorf("unmapped measure: want %q in Unsorted bucket, got %v", want, paths)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(want))); err != nil {
		t.Errorf("unmapped measure file not on disk: %v", err)
	}
}

// TestMirrorSkipsDeferredTypes is the deferral GUARD: an active task node must
// produce NO file anywhere in the mirror tree, and no returned path — proving
// task is never exported raw (its content needs its own redaction pass). It walks
// the WHOLE tree, so it bites if task is removed from deferredTypes.
// (day was un-deferred in plan 225; see TestDayJournalExportLeakTest.)
func TestMirrorSkipsDeferredTypes(t *testing.T) {
	app := storetest.NewApp(t)

	// A real active task node (state is the one required prop).
	taskNode, err := nodes.Create(app, "task", "Buy Milk", "",
		nodes.StatusActive, map[string]any{"state": "open"})
	if err != nil {
		t.Fatalf("create task node: %v", err)
	}
	if taskNode.GetString("status") != nodes.StatusActive {
		t.Fatalf("task node not active: %q", taskNode.GetString("status"))
	}
	taskSlug := slugForTest(taskNode.GetString("title")) + ".md"

	// Also seed a non-deferred node so the export actually produces a tree to
	// walk (a fully-empty mirror would pass vacuously).
	if _, err := nodes.Create(app, "note", "Visible", "Plain.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create note: %v", err)
	}

	dir := t.TempDir()
	paths, err := export.ExportMirror(app, dir)
	if err != nil {
		t.Fatalf("export mirror: %v", err)
	}

	// No returned path may carry the task JD folder prefix or the task slug.
	for _, p := range paths {
		if strings.HasPrefix(p, "60-69 Tasks/") {
			t.Errorf("deferred task type leaked a returned path: %q", p)
		}
		if filepath.Base(p) == taskSlug {
			t.Errorf("task node file returned: %q", p)
		}
	}

	// Walk the whole written tree: no task file may exist under any folder.
	for rel := range readAll(t, dir) {
		if strings.HasPrefix(rel, ".git") {
			continue
		}
		if filepath.Base(rel) == taskSlug {
			t.Errorf("task node file exists on disk: %q", rel)
		}
		if strings.Contains(rel, "60-69 Tasks") {
			t.Errorf("deferred task JD folder written: %q", rel)
		}
	}
}

// TestMirrorNeverLeaksStoredSecret is the EXTENDED canary: a real stored api_key
// (store.SaveCloudModel writes llm_providers.api_key) must appear in NO file of
// the full mirror. A leak means ExportMirror read a collection beyond `nodes`.
// This canary must never be deleted or weakened.
func TestMirrorNeverLeaksStoredSecret(t *testing.T) {
	app := storetest.NewApp(t)

	const secret = "sk-SECRET-TOKEN-DO-NOT-LEAK"
	if _, err := store.SaveCloudModel(app, "TestProvider", "https://example.test",
		secret, "Test", "test-model", ""); err != nil {
		t.Fatalf("seed cloud model: %v", err)
	}

	// Active nodes across a couple of types, none containing the secret.
	if _, err := nodes.Create(app, "note", "Notes", "Body without secrets.",
		nodes.StatusActive, map[string]any{"tag": "demo"}); err != nil {
		t.Fatalf("create note: %v", err)
	}
	if _, err := nodes.Create(app, "person", "Someone", "A person, no secrets.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create person: %v", err)
	}

	dir := t.TempDir()
	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("export mirror: %v", err)
	}
	for name, content := range readAll(t, dir) {
		if strings.Contains(content, secret) {
			t.Fatalf("STORED SECRET LEAKED into %s — exporter read the wrong collection:\n%s", name, content)
		}
	}
}

// TestMirrorGitCommit proves the mirror is committed to a local git history: one
// commit, a clean working tree. Cleanly skipped when git is not on PATH.
func TestMirrorGitCommit(t *testing.T) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "Committed", "Body.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create note: %v", err)
	}

	dir := t.TempDir()
	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("export mirror: %v", err)
	}

	// Query the EXPORT repo hermetically: scrub any inherited GIT_* env (e.g.
	// GIT_DIR set when the suite runs under a pre-commit hook) so these reads hit
	// the temp export repo, not the surrounding balaur repo.
	env := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_") {
			continue
		}
		env = append(env, kv)
	}
	gitOut := func(args ...string) string {
		cmd := exec.Command(gitBin, args...)
		cmd.Dir = dir
		cmd.Env = env
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), e, out)
		}
		return strings.TrimSpace(string(out))
	}
	if log := gitOut("log", "--oneline"); strings.Count(log, "\n") != 0 || log == "" {
		t.Errorf("want exactly one commit, got:\n%s", log)
	}
	if status := gitOut("status", "--porcelain"); status != "" {
		t.Errorf("working tree not clean after commit:\n%s", status)
	}
}

// TestDayJournalExportLeakTest is the redaction proof for day-journal export
// (plan 225). It seeds a day node with a known journal body AND a real record
// in the `summaries` collection with a DISTINCT marker string, then asserts:
//  1. The journal body appears in an exported file (day node is exported).
//  2. The recap marker appears in NO exported file (the exporter never opens
//     the summaries collection — the redaction boundary holds).
//  3. A second ExportMirror over unchanged data produces byte-identical files.
func TestDayJournalExportLeakTest(t *testing.T) {
	app := storetest.NewApp(t)

	const journalText = "JOURNAL_BODY_OWNER_WORDS"
	const recapMarker = "RECAP_MARKER_MODEL_TEXT_DO_NOT_LEAK"

	// Seed a day node with a known journal body (verbatim owner text).
	dayNode, err := nodes.DayNode(app, time.Now())
	if err != nil {
		t.Fatalf("create day node: %v", err)
	}
	dayNode.Set("body", journalText)
	if err := app.Save(dayNode); err != nil {
		t.Fatalf("set day body: %v", err)
	}

	// Seed a recap summary in the `summaries` collection with a distinct marker.
	// summaries.conversation is a required RelationField — create a conversation first.
	convCol, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		t.Fatalf("find conversations collection: %v", err)
	}
	conv := core.NewRecord(convCol)
	conv.Set("title", "Test Conversation")
	conv.Set("kind", "master")
	conv.Set("status", "open")
	if err := app.Save(conv); err != nil {
		t.Fatalf("save test conversation: %v", err)
	}
	sumCol, err := app.FindCollectionByNameOrId("summaries")
	if err != nil {
		t.Fatalf("find summaries collection: %v", err)
	}
	now := time.Now().UTC().Truncate(24 * time.Hour) // start of today UTC
	sum := core.NewRecord(sumCol)
	sum.Set("conversation", conv.Id)
	sum.Set("period_type", "day")
	sum.Set("period_start", now)
	sum.Set("period_end", now.Add(24*time.Hour-time.Millisecond))
	sum.Set("content", recapMarker)
	sum.Set("message_count", 0)
	if err := app.Save(sum); err != nil {
		t.Fatalf("save test summary: %v", err)
	}

	dir := t.TempDir()
	paths, err := export.ExportMirror(app, dir)
	if err != nil {
		t.Fatalf("export mirror: %v", err)
	}

	// 1. The journal body must appear in an exported file.
	journalFound := false
	for _, content := range readAll(t, dir) {
		if strings.Contains(content, journalText) {
			journalFound = true
		}
	}
	if !journalFound {
		t.Errorf("journal body %q not found in any exported file (paths: %v)", journalText, paths)
	}

	// 2. The recap marker must appear in NO exported file — the exporter must
	// never open the summaries collection (the redaction boundary).
	for name, content := range readAll(t, dir) {
		if strings.HasPrefix(name, ".git") {
			continue
		}
		if strings.Contains(content, recapMarker) {
			t.Fatalf("RECAP MARKER LEAKED into %s — summaries collection read by exporter:\n%s", name, content)
		}
	}

	// 3. Determinism: a second run over unchanged data produces byte-identical files.
	first := readAll(t, dir)
	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("second export: %v", err)
	}
	second := readAll(t, dir)
	stripGit := func(m map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range m {
			if !strings.HasPrefix(k, ".git") {
				out[k] = v
			}
		}
		return out
	}
	if !reflect.DeepEqual(stripGit(first), stripGit(second)) {
		t.Errorf("re-export not byte-identical:\nfirst:  %v\nsecond: %v",
			stripGit(first), stripGit(second))
	}
}

// slugForTest mirrors export.slug for the deferral guard: it must compute the
// same filename the exporter WOULD have used, so the test asserts that file is
// absent. Kept tiny and standalone (export.slug is unexported).
func slugForTest(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// TestMirrorPrunesArchivedNode is the consent-withdrawal case: archiving an
// active node must remove its file from the mirror's managed folder on the
// next export.
func TestMirrorPrunesArchivedNode(t *testing.T) {
	app := storetest.NewApp(t)

	if _, err := nodes.Create(app, "note", "Keep Me", "Body.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create keep note: %v", err)
	}
	archiveMe, err := nodes.Create(app, "note", "Archive Me", "Body.",
		nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create archive note: %v", err)
	}

	dir := t.TempDir()
	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("first export: %v", err)
	}
	const keepPath = "10-19 Knowledge/11 Notes/keep-me.md"
	const archivePath = "10-19 Knowledge/11 Notes/archive-me.md"
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(keepPath))); err != nil {
		t.Fatalf("keep-me.md not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(archivePath))); err != nil {
		t.Fatalf("archive-me.md not written: %v", err)
	}

	if _, err := nodes.Transition(app, archiveMe.Id, nodes.StatusArchived, "node"); err != nil {
		t.Fatalf("archive node: %v", err)
	}

	written, err := export.ExportMirror(app, dir)
	if err != nil {
		t.Fatalf("second export: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(archivePath))); !os.IsNotExist(err) {
		t.Errorf("archive-me.md should be pruned, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(keepPath))); err != nil {
		t.Errorf("keep-me.md should still exist: %v", err)
	}
	for _, p := range written {
		if p == archivePath {
			t.Errorf("returned written slice still contains archived path %q", p)
		}
	}
}

// TestMirrorPrunesRenamedNode is the retitle case: renaming an active node
// must remove the old-slug file, leaving exactly one file for the node.
func TestMirrorPrunesRenamedNode(t *testing.T) {
	app := storetest.NewApp(t)

	rec, err := nodes.Create(app, "note", "Old Title", "Body.",
		nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	dir := t.TempDir()
	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("first export: %v", err)
	}
	const oldPath = "10-19 Knowledge/11 Notes/old-title.md"
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(oldPath))); err != nil {
		t.Fatalf("old-title.md not written: %v", err)
	}

	newTitle := "New Title"
	if _, err := nodes.Update(app, rec.Id, &newTitle, nil, nil); err != nil {
		t.Fatalf("retitle node: %v", err)
	}

	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("second export: %v", err)
	}
	const newPath = "10-19 Knowledge/11 Notes/new-title.md"
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(newPath))); err != nil {
		t.Errorf("new-title.md should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(oldPath))); !os.IsNotExist(err) {
		t.Errorf("old-title.md should be pruned, stat err = %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, filepath.FromSlash("10-19 Knowledge/11 Notes")))
	if err != nil {
		t.Fatalf("read notes dir: %v", err)
	}
	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount != 1 {
		t.Errorf("want exactly one .md file in notes folder, got %d", mdCount)
	}
}

// TestMirrorPruneSparesOwnerFiles proves the boundary invariant: prune only
// ever removes *.md files directly inside managed JD folders. Owner files
// outside managed folders, .md files in unmanaged dirs, and non-.md files
// inside managed folders all survive; a stray .md inside a managed folder
// (that no active node produced) is removed — the exact rule that makes the
// archived/renamed cases work.
func TestMirrorPruneSparesOwnerFiles(t *testing.T) {
	app := storetest.NewApp(t)

	if _, err := nodes.Create(app, "note", "Real Note", "Body.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create note: %v", err)
	}

	dir := t.TempDir()
	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("first export: %v", err)
	}

	// Plant owner files that must survive pruning.
	outsidePath := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(outsidePath, []byte("owner file"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	unmanagedDir := filepath.Join(dir, "My Stuff")
	if err := os.MkdirAll(unmanagedDir, 0o755); err != nil {
		t.Fatalf("mkdir unmanaged: %v", err)
	}
	unmanagedMD := filepath.Join(unmanagedDir, "mine.md")
	if err := os.WriteFile(unmanagedMD, []byte("owner md"), 0o644); err != nil {
		t.Fatalf("write unmanaged md: %v", err)
	}
	notesDir := filepath.Join(dir, filepath.FromSlash("10-19 Knowledge/11 Notes"))
	ownerTxt := filepath.Join(notesDir, "owner.txt")
	if err := os.WriteFile(ownerTxt, []byte("owner txt in managed dir"), 0o644); err != nil {
		t.Fatalf("write owner.txt: %v", err)
	}
	strayMD := filepath.Join(notesDir, "stray.md")
	if err := os.WriteFile(strayMD, []byte("stray"), 0o644); err != nil {
		t.Fatalf("write stray.md: %v", err)
	}

	if _, err := export.ExportMirror(app, dir); err != nil {
		t.Fatalf("second export: %v", err)
	}

	if _, err := os.Stat(outsidePath); err != nil {
		t.Errorf("notes.txt should survive: %v", err)
	}
	if _, err := os.Stat(unmanagedMD); err != nil {
		t.Errorf("My Stuff/mine.md should survive: %v", err)
	}
	if _, err := os.Stat(ownerTxt); err != nil {
		t.Errorf("11 Notes/owner.txt should survive: %v", err)
	}
	if _, err := os.Stat(strayMD); !os.IsNotExist(err) {
		t.Errorf("stray.md should be removed, stat err = %v", err)
	}
}
