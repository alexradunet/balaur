package export_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

// TestMirrorSkipsDeferredTypes is the deferral GUARD: an active day node AND an
// active task node must produce NO file anywhere in the mirror tree, and no
// returned path — proving day/task are never exported raw (their recap/transcript
// content needs its own redaction pass). It walks the WHOLE tree, so it bites if
// either type is removed from deferredTypes.
func TestMirrorSkipsDeferredTypes(t *testing.T) {
	app := storetest.NewApp(t)

	// A real active day node (DayNode creates a type=day active node titled with
	// the human-readable date).
	dayNode, err := nodes.DayNode(app, time.Now())
	if err != nil {
		t.Fatalf("create day node: %v", err)
	}
	if dayNode.GetString("status") != nodes.StatusActive {
		t.Fatalf("day node not active: %q", dayNode.GetString("status"))
	}
	daySlug := slugForTest(dayNode.GetString("title")) + ".md"

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

	// No returned path may carry the day/task JD folder prefix or the node slug.
	for _, p := range paths {
		if strings.HasPrefix(p, "50-59 Journal/") || strings.HasPrefix(p, "60-69 Tasks/") {
			t.Errorf("deferred type leaked a returned path: %q", p)
		}
		base := filepath.Base(p)
		if base == daySlug {
			t.Errorf("day node file returned: %q", p)
		}
		if base == taskSlug {
			t.Errorf("task node file returned: %q", p)
		}
	}

	// Walk the whole written tree: no day/task file may exist under any folder.
	for rel := range readAll(t, dir) {
		if strings.HasPrefix(rel, ".git") {
			continue
		}
		base := filepath.Base(rel)
		if base == daySlug {
			t.Errorf("day node file exists on disk: %q", rel)
		}
		if base == taskSlug {
			t.Errorf("task node file exists on disk: %q", rel)
		}
		if strings.Contains(rel, "50-59 Journal") || strings.Contains(rel, "60-69 Tasks") {
			t.Errorf("deferred JD folder written: %q", rel)
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
