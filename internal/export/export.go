// Package export is Balaur's sovereign-export mirror (plans 192/194): a one-way,
// read-only renderer of the active knowledge record to Markdown. ExportMirror
// writes the full Johnny Decimal vault mirror — every owner-authored node type
// into its JD folder, full re-export, committed to a git history under the dest
// (no encryption yet — that is Phase 3). ExportType remains as the one-type
// primitive. The redaction boundary is hard: it reads ONLY the `nodes`
// collection, ONLY status=active rows, and never touches any secret/token
// collection (api_key, OAuth tokens, vault entries — see the design note at
// docs/superpowers/specs/2026-06-25-sovereign-export-design.md).
package export

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
)

// ExportType renders every ACTIVE node of one type to a Markdown file under
// destDir (one file per node: YAML frontmatter from props + an H1 title + the
// node body, which already carries [[wikilinks]]). It returns the relative file
// paths written. It is read-only on PocketBase: it calls only the active-filtered
// nodes.ListByTypeStatus reader and writes only under destDir. It never reads
// llm_providers, extensions, owner_settings, or any token/secret collection.
func ExportType(app core.App, typ, destDir string) ([]string, error) {
	// The consent filter: status=active is non-negotiable. Proposed, rejected,
	// and archived nodes are never read, so they can never be exported.
	recs, err := nodes.ListByTypeStatus(app, typ, nodes.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("export: listing %q nodes: %w", typ, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("export: making %s: %w", destDir, err)
	}

	used := map[string]bool{}
	paths := make([]string, 0, len(recs))
	for _, rec := range recs {
		name := uniqueName(slug(rec.GetString("title"), rec.Id), rec.Id, used)
		path := filepath.Join(destDir, name)
		if err := os.WriteFile(path, render(rec), 0o644); err != nil {
			return nil, fmt.Errorf("export: writing %s: %w", path, err)
		}
		paths = append(paths, name)
	}
	return paths, nil
}

// jdFolder maps a node type to its Johnny Decimal folder (design table). A type
// absent from this map falls back to the Unsorted bucket via jdFolderFor.
var jdFolder = map[string]string{
	"note":   "10-19 Knowledge/11 Notes",
	"idea":   "10-19 Knowledge/12 Ideas",
	"person": "20-29 People/21 People",
	"book":   "30-39 Library/31 Books",
	"place":  "40-49 Places/41 Places",
	"day":    "50-59 Journal/51 Days",
	// "task" has a JD folder in the design (60-69 Tasks/61 Tasks) but is
	// DEFERRED — see deferredTypes. It is intentionally NOT exported until its
	// content redaction pass lands (a future plan).
}

// unsortedFolder is the fallback for any owner-authored type without an explicit
// JD mapping (design Q1: 90-99 Unsorted/91 Other/).
const unsortedFolder = "90-99 Unsorted/91 Other"

// deferredTypes are owner-authored types whose faithful export needs its own
// redaction pass. Exporting them raw could surface un-reviewed content, so they
// are skipped entirely until their redaction pass + leak test land.
// Do NOT remove a type from this set without adding its redaction pass + leak test.
var deferredTypes = map[string]bool{
	"task": true,
}

// jdFolderFor returns the relative JD folder for a node type, defaulting to the
// Unsorted bucket for any unmapped type.
func jdFolderFor(typ string) string {
	if f, ok := jdFolder[typ]; ok {
		return f
	}
	return unsortedFolder
}

// ExportMirror renders every owner-authored node type into a Johnny Decimal
// folder tree under destDir: one Markdown file per ACTIVE node, full re-export
// (every file rewritten every run, so a second run over unchanged data is
// byte-identical — design Q3). It is read-only on PocketBase: it lists types via
// nodes.OwnerAuthoredTypes and nodes only via the active-filtered
// nodes.ListByTypeStatus reader. It NEVER opens llm_providers, llm_models,
// extensions, owner_settings, audit_log, or any token/secret/conversation
// collection — the sovereign redaction boundary (asserted by the canary test).
//
// task is DEFERRED (deferredTypes): its content needs its own redaction pass, so
// it is skipped here. day was un-deferred in plan 225 (leak test proved the body
// carries only the owner's journal text — recap/summaries never touch the body).
//
// After writing the tree it commits the mirror to a git history under destDir
// (offline; skipped cleanly when git is absent). It returns the relative file
// paths written (slash-separated, under destDir), sorted, so the result is
// deterministic.
func ExportMirror(app core.App, destDir string) ([]string, error) {
	types, err := nodes.OwnerAuthoredTypes(app)
	if err != nil {
		return nil, fmt.Errorf("export: listing owner-authored types: %w", err)
	}
	slices.Sort(types) // deterministic iteration

	var written []string
	for _, typ := range types {
		if deferredTypes[typ] {
			continue
		}
		recs, err := nodes.ListByTypeStatus(app, typ, nodes.StatusActive)
		if err != nil {
			return nil, fmt.Errorf("export: listing %q nodes: %w", typ, err)
		}
		if len(recs) == 0 {
			continue
		}
		relDir := jdFolderFor(typ)
		absDir := filepath.Join(destDir, filepath.FromSlash(relDir))
		if err := os.MkdirAll(absDir, 0o755); err != nil {
			return nil, fmt.Errorf("export: making %s: %w", absDir, err)
		}
		// Per-folder collision map: two nodes of different types never share a
		// folder, so resetting per type is correct (matches ExportType).
		used := map[string]bool{}
		for _, rec := range recs {
			name := uniqueName(slug(rec.GetString("title"), rec.Id), rec.Id, used)
			abs := filepath.Join(absDir, name)
			if err := os.WriteFile(abs, render(rec), 0o644); err != nil {
				return nil, fmt.Errorf("export: writing %s: %w", abs, err)
			}
			written = append(written, path.Join(relDir, name))
		}
	}
	slices.Sort(written)

	// Commit the mirror to a local git history. A real git failure is surfaced;
	// a missing git binary or an unchanged tree is not an error (the files are
	// already written).
	if _, err := commitMirror(destDir); err != nil {
		return nil, fmt.Errorf("export: committing mirror: %w", err)
	}
	return written, nil
}

// commitMirror initialises a git repo under destDir (if none exists) and commits
// the current mirror state. It is owner-initiated and offline (no remote, no
// network). When git is not on PATH it returns (false, nil) — the export already
// wrote the files; lacking git is not a failure. Returns true when a commit was
// made, false when skipped (no git, or nothing to commit). Any git invocation
// error (other than a missing binary) is returned so the caller can surface it.
func commitMirror(destDir string) (committed bool, err error) {
	gitBin, lookErr := exec.LookPath("git")
	if lookErr != nil {
		return false, nil // git skipped — files are written, this is not an error
	}
	// The export commits to its OWN dedicated repo under destDir; it must be
	// hermetic — never inherit the caller's GIT_DIR/index/hooks (otherwise an
	// outer `git commit` — e.g. running tests under a pre-commit hook — would
	// leak its hooksPath and fire foreign hooks inside the export repo). Scrub
	// every inherited GIT_* var and force hooks off for this repo's git calls.
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_") {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "GIT_CONFIG_NOSYSTEM=1")
	run := func(args ...string) error {
		// -c core.hooksPath=/dev/null disables any inherited hooksPath so an
		// automated export commit never runs unrelated repo hooks.
		full := append([]string{"-c", "core.hooksPath=/dev/null"}, args...)
		cmd := exec.Command(gitBin, full...)
		cmd.Dir = destDir
		cmd.Env = env
		if out, e := cmd.CombinedOutput(); e != nil {
			return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), e, strings.TrimSpace(string(out)))
		}
		return nil
	}
	if _, statErr := os.Stat(filepath.Join(destDir, ".git")); os.IsNotExist(statErr) {
		if err := run("init"); err != nil {
			return false, err
		}
		// Identity for the commit, scoped to this repo so the export works on a
		// box with no global git identity configured.
		if err := run("config", "user.email", "balaur@localhost"); err != nil {
			return false, err
		}
		if err := run("config", "user.name", "Balaur Export"); err != nil {
			return false, err
		}
	}
	if err := run("add", "-A"); err != nil {
		return false, err
	}
	// Nothing to commit (unchanged data) is success, not an error: `git commit`
	// exits non-zero when the tree is clean, so check first.
	if diffErr := run("diff", "--cached", "--quiet"); diffErr == nil {
		return false, nil // no changes staged → nothing to commit
	}
	// Fixed message (no wall-clock) so unchanged data never produces a noisy diff.
	if err := run("commit", "-m", "balaur export: sovereign Markdown mirror"); err != nil {
		return false, err
	}
	return true, nil
}

// render builds the Markdown for one node: YAML frontmatter (type/status/
// created/updated + each props scalar), an H1 title, then the body verbatim
// (it already carries [[wikilinks]]).
func render(rec *core.Record) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(yamlLine("type", rec.GetString("type")))
	b.WriteString(yamlLine("status", rec.GetString("status")))
	b.WriteString(yamlLine("created", rec.GetDateTime("created").String()))
	b.WriteString(yamlLine("updated", rec.GetDateTime("updated").String()))
	// Props in a stable order so the export is deterministic (full re-export
	// must produce byte-identical output for unchanged data).
	props := nodes.Props(rec)
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		b.WriteString(yamlLine(k, scalar(props[k])))
	}
	b.WriteString("---\n\n")
	b.WriteString("# " + rec.GetString("title") + "\n\n")
	b.WriteString(rec.GetString("body"))
	b.WriteString("\n")
	return []byte(b.String())
}

// yamlLine writes one `key: "value"` frontmatter line. The value is always
// %q-quoted so a `:`, quote, or newline in titles/values cannot break the YAML.
func yamlLine(key, value string) string {
	return fmt.Sprintf("%s: %q\n", key, value)
}

// scalar renders a props value as a frontmatter scalar. Strings/numbers/bools
// stringify directly; a nested map or slice is JSON-encoded inline (nested-prop
// frontmatter is deferred to Phase 2 — see the design note).
func scalar(v any) string {
	switch v.(type) {
	case map[string]any, []any:
		if raw, err := json.Marshal(v); err == nil {
			return string(raw)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// slug lowercases the title and collapses runs of non-alphanumerics to a single
// '-'. An empty result falls back to the node id so every file has a name.
func slug(title, id string) string {
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
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return id
	}
	return s
}

// uniqueName resolves a slug collision within the destination by appending the
// node id, which is unique per record.
func uniqueName(base, id string, used map[string]bool) string {
	name := base + ".md"
	if used[name] {
		name = base + "-" + id + ".md"
	}
	used[name] = true
	return name
}
