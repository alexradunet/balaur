# Plan 194: Ship the full one-way Johnny Decimal Markdown vault mirror (sovereign export Phase 2)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat e06346d..HEAD -- internal/export/ internal/cli/export.go internal/cli/export_test.go internal/self/knowledge.md README.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **Set `TMPDIR` before any `go` command** (tmpfs link bug on this box):
> `export TMPDIR=/home/alex/.cache/go-tmp` (create it first: `mkdir -p /home/alex/.cache/go-tmp`).

## Status

- **Priority**: P3
- **Effort**: L
- **Risk**: LOW
- **Depends on**: none (extends the plan-192 prototype already on `main`)
- **Category**: direction
- **Planned at**: commit `e06346d`, 2026-06-25

## Why this matters

Balaur's **Sovereignty** pillar promises the owner's life lives "in inspectable
SQLite **and exported Markdown** — never in a vendor's database." Today only a
SPIKE exists: `internal/export.ExportType` renders ONE node type (`note`) to a
flat directory, with no folder layout and no git. The README honesty ledger
still lists "Johnny Decimal Markdown vault mirror" as **not shipped**. This plan
turns the spike into the real Phase-2 mirror: every owner-authored node type
rendered into the Johnny Decimal folder tree, fully deterministic (byte-identical
re-export), committed to a git history under the data dir, and exposed as the
real `balaur export` verb. The redaction boundary — the SACRED invariant that the
exporter reads ONLY `status=active` rows of ONLY the `nodes` collection and NEVER
any secret/token collection — is preserved and its canary test extended to the
whole mirror. When this lands, the owner can `balaur export` and get a walkable,
git-versioned, secret-free Markdown vault of their knowledge record.

## Current state

HEAD at planning time **is** `e06346d`, so the excerpts below are live (the drift
check above will confirm). Files and roles:

- `internal/export/export.go` — the Phase-1 prototype. Exports ONE type via
  `ExportType`. Carries the reusable helpers `render`, `yamlLine`, `scalar`,
  `slug`, `uniqueName` you MUST reuse (one source of truth — do not re-implement
  rendering).
- `internal/export/export_test.go` — three tests, incl. the redaction canary
  `TestExportNeverLeaksStoredSecret`.
- `internal/cli/export.go` — the CLI STUB wiring `ExportType` behind `--type`/`--out`.
- `internal/cli/export_test.go` — two CLI tests.
- `internal/nodes/types.go` — `OwnerAuthoredTypes` / `TypeNames` (the registry readers).
- `internal/nodes/nodes.go` — `ListByTypeStatus`, `StatusActive`, `Props`, `Create`.
- `internal/cli/cli.go` — the `run(...)` envelope wrapper + the v1 `{v,kind,data}` contract.
- `internal/self/knowledge.md` — the running binary's self-description (lines 321-326 describe `balaur export` as a SPIKE; must be updated).
- `README.md:425` — the roadmap ledger line to flip ONLY if Phase 2 is truly complete.

### `internal/export/export.go` (verbatim — the helpers to reuse)

```go
// internal/export/export.go:29-51
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
```

```go
// internal/export/export.go:56-79 — render() builds one node's Markdown.
// Props are sorted (slices.Sort) so output is DETERMINISTIC — this is what
// makes byte-identical re-export possible. Reuse render() unchanged.
func render(rec *core.Record) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(yamlLine("type", rec.GetString("type")))
	b.WriteString(yamlLine("status", rec.GetString("status")))
	b.WriteString(yamlLine("created", rec.GetDateTime("created").String()))
	b.WriteString(yamlLine("updated", rec.GetDateTime("updated").String()))
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
```

`yamlLine` (line 83), `scalar` (line 90), `slug` (line 104), and `uniqueName`
(line 127) are unexported helpers in the SAME package — call them directly from
the new `ExportMirror`. Do NOT export them or duplicate them.

> **DETERMINISM CAVEAT (read carefully).** `render` includes
> `created`/`updated` from the record. Those are stable for a given stored
> node, so a second export over **unchanged data** is byte-identical (that is
> all Q3 / the done-criteria require). Do not add `time.Now()` anywhere in the
> render or the file write — the only timestamps that may appear are the
> record's own `created`/`updated` and (in git) the commit time. The git commit
> message MUST NOT embed a wall-clock timestamp, or the canary's
> "byte-identical second run" intent leaks into git noise; keep the message a
> fixed string (see Step 4).

### `internal/cli/export.go` (verbatim — the stub to replace)

```go
// internal/cli/export.go:16-34
func exportCmd(app core.App) *cobra.Command {
	var typ, out string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Spike: read-only one-type Markdown export of active nodes (no git, no encryption)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&typ, "type", "note", "node type to export (spike: note only)")
	cmd.Flags().StringVar(&out, "out", "", "destination directory (required; never the data dir)")
	_ = cmd.MarkFlagRequired("out")
	cmd.RunE = run(app, "export", func(cmd *cobra.Command, args []string) (any, error) {
		paths, err := export.ExportType(app, typ, out)
		if err != nil {
			return nil, fmt.Errorf("export: %w", err)
		}
		return map[string]any{"type": typ, "out": out, "files": paths}, nil
	})
	return cmd
}
```

### `internal/nodes` readers (verbatim — the ONLY data path the exporter may use)

```go
// internal/nodes/nodes.go:26-31 — the status constants.
const (
	StatusProposed = "proposed"
	StatusActive   = "active"
	StatusArchived = "archived"
	StatusRejected = "rejected"
)

// internal/nodes/nodes.go:228-233 — the active-filtered reader. THE ONLY DB read.
func ListByTypeStatus(app core.App, typ, status string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("nodes",
		"type = {:t} && status = {:s}", "-created", 0, 0,
		dbx.Params{"t": typ, "s": status})
}
```

```go
// internal/nodes/types.go:69-82 — the owner-authored type set to iterate.
func OwnerAuthoredTypes(app core.App) ([]string, error) {
	recs, err := app.FindRecordsByFilter("node_types", "born_status = 'active'", "name", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("node_types: listing owner-authored types: %w", err)
	}
	names := make([]string, 0, len(recs))
	for _, r := range recs {
		names = append(names, r.GetString("name"))
	}
	return names, nil
}
```

> **What `OwnerAuthoredTypes` actually returns on a seeded box.** The registry
> is built additively across migrations. The owner-authored (`born_status =
> 'active'`) types are: **`note`, `person`, `book`, `idea`, `place`, `measure`,
> `day`, `task`**. Verified in `migrations/1750000000_node_types.go:43-50`
> (`note`/`person`/`book`/`idea`/`place`), `1750000030_measures_to_nodes.go:60`
> (`measure`), `1750000020_tasks_to_nodes.go:63-66` (`task`),
> `1750000040_day_type.go:48-51` (`day`). `memory`/`skill` are born `proposed`
> and are NOT in the set. This matters for Steps 1-2: your JD map covers most
> of these, but `measure` is UNMAPPED (→ Unsorted) and `day`/`task` are DEFERRED
> (see the deferral rule below).

### The design constraints to honor (inlined — the executor has NOT read the design doc)

From `docs/superpowers/specs/2026-06-25-sovereign-export-design.md`:

**The redaction boundary (SACRED invariant, design lines 68-81):**
> `internal/export` may read exactly one collection (`nodes`) and exactly one
> status (`active`). It never opens `llm_providers`, `llm_models`, `extensions`,
> `owner_settings`, `audit_log`, or any token/secret/conversation collection.

The canary test seeds a real `api_key` via `store.SaveCloudModel` (writes
`llm_providers.api_key`), runs the export, walks every written file, and asserts
the secret substring appears in NONE. A leak means the exporter read the wrong
collection — a defect, never paper over it by string-filtering output.

**The type → Johnny Decimal folder map (design lines 89-98):**

| node type | folder                         |
|-----------|--------------------------------|
| note      | `10-19 Knowledge/11 Notes/`    |
| idea      | `10-19 Knowledge/12 Ideas/`    |
| person    | `20-29 People/21 People/`      |
| book      | `30-39 Library/31 Books/`      |
| place     | `40-49 Places/41 Places/`      |
| day       | `50-59 Journal/51 Days/`       |
| task      | `60-69 Tasks/61 Tasks/`        |

**Unsorted fallback (design lines 99-109, 158-160):** any registry type without
an explicit mapping → `90-99 Unsorted/91 Other/`.

**Full re-export, not incremental (design Q3, lines 171-180):** rewrite every
file every run; deterministic; git diffs show what changed. No manifest, no
mtime state.

**Phase-2 scope (design lines 215-221):** all owner-authored types, the JD
layout, full re-export, `git init`/commit under the data dir, one-way + additive
+ owner-initiated + offline. Any NEW collection read must come with its own
"this secret never appears in output" test.

**THE DEFERRAL RULE (design lines 99-109; brief OUT OF SCOPE).** `day` recap
transcripts and `task` content are explicitly DEFERRED — "recap/transcript
content needs its own redaction pass." Even though a `day` node's body is
currently owner prose and a `task` is a typed object, the design author chose to
withhold both pending a dedicated redaction review, and the brief makes this a
HARD rule: **do NOT export `type=day` or `type=task` raw in this plan.** Exclude
them via an explicit skip-set (Step 1), document why in a comment, and report it
in your done summary. (This is the one place where the JD map above lists `day`
and `task` folders but you still must NOT emit files for them — the folders are
recorded for the future plan that adds their redaction pass.)

### Repo conventions that apply here

- gofmt is law; `go vet`/staticcheck/govulncheck gate CI. No assertion
  frameworks; tests are standard `testing`, table-driven where it helps. Use
  `storetest.NewApp(t)` for a PB app (it registers migrations, so `node_types`
  is seeded) and `t.TempDir()` for output dirs. NO `time.Sleep`.
- Errors wrap with `fmt.Errorf("...: %w", err)`; structured logging via
  `app.Logger()` if you log at all (the exporter currently does not log — keep it
  quiet unless git is skipped, see Step 4).
- `internal/export` is its own domain-ish package that reads `nodes` directly —
  do NOT route through `internal/store`. No new module dependency: the git step
  uses `os/exec` (stdlib), exactly as `internal/tools/os.go` and
  `internal/launch/launch.go` already do. Do NOT add `go-git` or any git library.
- Records ARE the domain model; read them through the `nodes` readers only.

## Commands you will need

| Purpose       | Command                                                              | Expected on success            |
|---------------|---------------------------------------------------------------------|--------------------------------|
| Prep tmpdir   | `mkdir -p /home/alex/.cache/go-tmp`                                  | exit 0                         |
| Set tmpdir    | `export TMPDIR=/home/alex/.cache/go-tmp`                             | (run once per shell)           |
| Build         | `CGO_ENABLED=0 go build ./...`                                       | exit 0                         |
| Test (export) | `go test ./internal/export/`                                         | `ok`, all pass                 |
| Test (cli)    | `go test ./internal/cli/`                                            | `ok`, all pass                 |
| Full suite    | `go test ./...`                                                      | all `ok` (gate before declaring done) |
| Vet           | `go vet ./...`                                                       | exit 0, no output              |
| Format check  | `gofmt -l internal/export/ internal/cli/`                            | empty output                   |
| Diff hygiene  | `git diff --check`                                                   | no output                      |
| Git on PATH?  | `command -v git`                                                     | a path (git IS present here)   |

## Suggested executor toolkit

- Invoke the `go-standards` skill before writing Go — it carries the repo's error
  wrapping, `slices`/`maps` idioms, testing conventions, and the `os/exec` style.
- Reference doc (already inlined above, read only if you need more):
  `docs/superpowers/specs/2026-06-25-sovereign-export-design.md`.

## Scope

**In scope** (the only files you should modify or create):
- `internal/export/export.go` — ADD `ExportMirror` + a JD-map helper + a git helper. Reuse the existing render/slug/yamlLine/scalar/uniqueName helpers unchanged.
- `internal/export/mirror_test.go` (create) — layout, byte-identical re-export, Unsorted, deferral-skip, git-commit, and the EXTENDED redaction canary tests.
- `internal/cli/export.go` — REWIRE the verb to `ExportMirror` with a `--dir` flag (default under the data dir) emitting `{files, dest}`.
- `internal/cli/export_test.go` — UPDATE the two existing tests to the new flag/envelope shape.
- `internal/self/knowledge.md` — update the `balaur export` description (lines 321-326) from SPIKE to the shipped mirror.
- `README.md` — flip the roadmap ledger line (line 425) to shipped ONLY if Phase 2 is truly complete (see Step 6's gate).
- `plans/README.md` — add/update the status row for plan 194.

**Out of scope** (do NOT touch, even though they look related):
- Encryption / `age` / passphrase KDF — that is Phase 3 = plan 195. No crypto here.
- Any `internal/web` UI surface (a storybook card, a settings button) — explicitly deferred; this plan is CLI + library only.
- `type=day` and `type=task` raw export — DEFERRED (see the deferral rule). Do NOT emit files for them; do NOT invent a redaction pass for them in this plan.
- The existing `ExportType` function — leave it in place (the per-type path is still a valid primitive; `ExportMirror` builds alongside it, not by deleting it). Do NOT remove or rename it.
- Incremental/manifest export — Q3 chose full re-export; do not add state.

## Git workflow

- You are in an execution worktree off `origin/main`. Commit per logical unit
  with conventional-commit subjects, e.g. `feat(export): full Johnny Decimal
  mirror + git commit + real export verb (194)`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add the JD folder map + deferral skip-set in `internal/export/export.go`

Add two package-level values and a small resolver. The map is the design table;
the skip-set is the deferral rule.

```go
// jdFolder maps a node type to its Johnny Decimal folder (design table). A type
// absent from this map falls back to the Unsorted bucket via jdFolderFor.
var jdFolder = map[string]string{
	"note":   "10-19 Knowledge/11 Notes",
	"idea":   "10-19 Knowledge/12 Ideas",
	"person": "20-29 People/21 People",
	"book":   "30-39 Library/31 Books",
	"place":  "40-49 Places/41 Places",
	// "day" and "task" have JD folders in the design (50-59 Journal/51 Days,
	// 60-69 Tasks/61 Tasks) but are DEFERRED — see deferredTypes. They are
	// intentionally NOT exported until their recap/transcript redaction pass
	// lands (a future plan).
}

// unsortedFolder is the fallback for any owner-authored type without an explicit
// JD mapping (design Q1: 90-99 Unsorted/91 Other/).
const unsortedFolder = "90-99 Unsorted/91 Other"

// deferredTypes are owner-authored types whose faithful export needs its own
// redaction pass (day recap pages, task content). Exporting them raw could
// surface un-reviewed content, so they are skipped entirely in this phase.
// Do NOT remove a type from this set without adding its redaction pass + leak test.
var deferredTypes = map[string]bool{
	"day":  true,
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
```

**Verify**: `gofmt -l internal/export/export.go` → empty. (Code does not compile
to a binary yet because `jdFolderFor`/`deferredTypes` are unused until Step 2 —
that is expected; the next step consumes them. Do not add a throwaway caller.)

### Step 2: Add `ExportMirror` to `internal/export/export.go`

Render EVERY owner-authored type (minus the deferred set) into the JD layout,
full re-export, deterministic. Reuse the existing helpers.

```go
// ExportMirror renders every owner-authored node type into a Johnny Decimal
// folder tree under destDir: one Markdown file per ACTIVE node, full re-export
// (every file rewritten every run, so a second run over unchanged data is
// byte-identical — design Q3). It is read-only on PocketBase: it lists types via
// nodes.OwnerAuthoredTypes and nodes only via the active-filtered
// nodes.ListByTypeStatus reader. It NEVER opens llm_providers, llm_models,
// extensions, owner_settings, audit_log, or any token/secret/conversation
// collection — the sovereign redaction boundary (asserted by the canary test).
//
// day and task are DEFERRED (deferredTypes): their recap/transcript content
// needs its own redaction pass, so they are skipped here.
//
// It returns the relative file paths written (slash-separated, under destDir),
// sorted, so the result is deterministic.
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
	return written, nil
}
```

Add imports as needed: `"path"` (for slash-joined relative paths in the return
value — keep `filepath` for on-disk paths). `slices` is already imported.

> **Note on `used` scoping.** The collision map is per-folder (reset for each
> type) which matches the existing `ExportType` per-dir behavior — two nodes of
> different types never share a folder, so a per-type map is correct.

**Verify**:
- `gofmt -l internal/export/` → empty.
- `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Write `internal/export/mirror_test.go`

Model its structure on the existing `internal/export/export_test.go` (reuse the
`readAll` walker pattern — copy it into the new file's package, OR keep both
tests in one `_test` package; since `export_test.go` already defines `readAll` in
`package export_test`, put the new tests in the SAME `package export_test` and
REUSE `readAll` — do not redefine it). Tests to write:

1. **`TestMirrorLayoutPerType`** — seed one active node of each MAPPED, non-deferred
   owner type (`note`, `idea`, `person`, `book`, `place`) via `nodes.Create(app,
   typ, title, body, nodes.StatusActive, props)`. Run `ExportMirror(app, dir)`.
   Assert the returned paths contain the expected JD-folder-prefixed file for each,
   e.g. a `note` titled "My Note" appears as `10-19 Knowledge/11 Notes/my-note.md`,
   and that the file on disk exists and contains `# My Note` and `type: "note"`.

2. **`TestMirrorByteIdenticalReexport`** — seed a couple of active nodes. Run
   `ExportMirror` into `dir`, capture `readAll(t, dir)`. Run `ExportMirror` into
   the SAME `dir` again, capture `readAll` again. Assert the two maps are
   deeply equal (same keys, same bytes). Use `reflect.DeepEqual` or compare
   key-by-key with a clear failure message. No `time.Sleep`, no mtime checks.

3. **`TestMirrorUnmappedTypeGoesUnsorted`** — register a NEW owner-authored type
   at runtime so it has no JD mapping, then export. To register: create a
   `node_types` record with `born_status='active'` and a name like `gizmo`
   (see how a test can do this below), create an active node of that type, run
   `ExportMirror`, and assert a file lands under `90-99 Unsorted/91 Other/`.
   If runtime type registration is awkward, you may instead rely on the SEEDED
   `measure` type (it IS owner-authored and UNMAPPED): create an active `measure`
   node and assert it lands in `90-99 Unsorted/91 Other/`. Prefer `measure` — it
   needs no registry mutation. (To create a `measure` node, check its schema first
   with a quick read of `migrations/1750000030_measures_to_nodes.go` for required
   props; if it has required props, pass them in the `nodes.Create` props map.)

4. **`TestMirrorSkipsDeferredTypes`** — create an active `day` node (use
   `nodes.DayNode(app, time.Now())` from `internal/nodes/day.go`, which creates a
   `type=day` active node) and an active `task` node if easily seedable; if `task`
   seeding is awkward, at minimum cover `day`. Run `ExportMirror`. Assert NO file
   exists under `50-59 Journal/51 Days/` and NO returned path starts with that
   prefix — proving the deferral skip. (This test is the guard that day/task are
   never exported raw.)

5. **`TestMirrorNeverLeaksStoredSecret`** — the EXTENDED canary. Copy the body of
   the existing `TestExportNeverLeaksStoredSecret` but call `ExportMirror(app,
   dir)` instead of `ExportType`. Seed the secret with
   `store.SaveCloudModel(app, "TestProvider", "https://example.test", secret,
   "Test", "test-model", "")` where `secret := "sk-SECRET-TOKEN-DO-NOT-LEAK"`,
   create at least one active `note` whose own content does NOT contain the
   secret, run `ExportMirror`, walk EVERY written file with `readAll`, and
   `t.Fatalf` if the secret substring appears in ANY file. This canary must never
   be deleted or weakened.

   **Prove the canary bites (do this manually, do NOT commit it):** temporarily
   add a line in `ExportMirror` that reads `llm_providers` and writes the key into
   a file (e.g. read the provider record's `api_key` and `os.WriteFile` it), run
   `go test ./internal/export/ -run TestMirrorNeverLeaksStoredSecret`, and confirm
   the test FAILS. Then REVERT that line completely. Record in your done summary
   that you verified the canary fails when a secret read is introduced. (The
   committed code must have NO read of any collection other than `nodes`.)

6. **`TestMirrorGitCommit`** (git present) — run `ExportMirror` then the new
   commit helper (Step 4) into a `t.TempDir()`; assert `git -C dir log --oneline`
   shows one commit and the working tree is clean. Guard with `if _, err :=
   exec.LookPath("git"); err != nil { t.Skip("git not on PATH") }` so the test is
   a clean skip on a git-less box, never a failure. (You will likely fold this
   into Step 4's helper test; either file is fine, keep it in this package.)

**Verify**: `go test ./internal/export/` → `ok`, all new tests pass.

### Step 4: Add the git commit helper to `internal/export/export.go`

`git init` (if absent) + add + commit the mirror under `destDir`. Owner-initiated,
offline. If git is not on PATH, write files and return cleanly with a noted
"git skipped" — NEVER fail the export for lack of git.

```go
// commitMirror initialises a git repo under destDir (if none exists) and commits
// the current mirror state. It is owner-initiated and offline (no remote, no
// network). When git is not on PATH it returns ("", nil) — the export already
// wrote the files; lacking git is not a failure. Returns the short commit hash
// on success, or "" when skipped. Any git invocation error (other than a missing
// binary) is returned so the caller can surface it.
func commitMirror(destDir string) (committed bool, err error) {
	gitBin, lookErr := exec.LookPath("git")
	if lookErr != nil {
		return false, nil // git skipped — files are written, this is not an error
	}
	run := func(args ...string) error {
		cmd := exec.Command(gitBin, args...)
		cmd.Dir = destDir
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
```

Add `"os/exec"` and `"strings"` to imports (`strings` is already imported; add
`os/exec`).

> **Why check `diff --cached --quiet` before commit:** `git commit` fails on a
> clean tree. A second run over unchanged data legitimately has nothing to
> commit — that is the deterministic re-export working, not an error. Return
> `(false, nil)` there.

Now wire git into `ExportMirror`: after the write loop and the final
`slices.Sort(written)`, call `commitMirror(destDir)`. If it errors, wrap and
return the error (a real git failure is worth surfacing); if it returns
`(false, nil)`, that is the git-skipped-or-nothing-to-commit path — succeed
silently. You may have `ExportMirror` return the written paths regardless of the
git outcome. Keep the git call INSIDE `ExportMirror` (so the CLI gets git for
free) OR return a small result struct — pick the simpler shape; the CLI in Step 5
needs `{files, dest}` and may also surface whether a commit happened. Recommended:
`ExportMirror` returns `([]string, error)` and does the commit internally; if you
want to report the commit, return a struct `MirrorResult{Files []string; Dest
string; Committed bool}` instead — choose ONE and make Step 5 match.

**Verify**: `go test ./internal/export/` → `ok`, including the git-commit test.

### Step 5: Rewire the `balaur export` verb in `internal/cli/export.go`

Replace the stub body. Default the destination under the data dir; emit the v1
envelope `{files, dest}`.

```go
func exportCmd(app core.App) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "One-way Johnny Decimal Markdown mirror of active nodes (+ git history)",
		Args:  cobra.NoArgs,
	}
	// Default: a "mirror" dir under the data dir. Resolved lazily in RunE so a
	// custom --dir wins and we never capture a stale path at construction time.
	cmd.Flags().StringVar(&dir, "dir", "", "destination directory (default: <data dir>/export)")
	cmd.RunE = run(app, "export", func(cmd *cobra.Command, args []string) (any, error) {
		dest := dir
		if dest == "" {
			dest = filepath.Join(app.DataDir(), "export")
		}
		files, err := export.ExportMirror(app, dest)
		if err != nil {
			return nil, fmt.Errorf("export: %w", err)
		}
		return map[string]any{"files": files, "dest": dest}, nil
	})
	return cmd
}
```

(If you chose the `MirrorResult` struct in Step 4, adapt: `res, err :=
export.ExportMirror(...)`; return `map[string]any{"files": res.Files, "dest":
res.Dest}`.)

Add `"path/filepath"` to the CLI file's imports; drop the now-unused `--type`/
`--out` flag wiring. The `run(...)` wrapper already applies migrations and emits
the envelope — do not change it.

> **`app.DataDir()` is the right anchor** — verified in use at
> `internal/cli/doctor.go:40` and `main.go:238`. The default `<data dir>/export`
> is a sibling of the SQLite, owner-local, never the internet.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 6: Update `internal/cli/export_test.go` to the new shape

The two existing tests reference `--type`/`--out` and a `files`/`out`/`type`
data shape that no longer exists. Update them:

- **`TestExportEmitsEnvelopeAndWritesFiles`**: seed an active `note`, run
  `executeEnvelope(t, exportCmd(app), "--dir", out)` (where `out := t.TempDir()`).
  Assert `env["kind"] == "export"`, that `data["files"]` has one entry, that
  `data["dest"] == out`, and that the file at `filepath.Join(out, files[0])`
  exists and contains `# Exported Note`. Note `files[0]` is now a SLASH-joined
  JD-prefixed path like `10-19 Knowledge/11 Notes/exported-note.md` — join it with
  `filepath.Join(out, filepath.FromSlash(name))` for the on-disk read.
- **`TestExportRequiresOut`**: this asserted `--out` was mandatory. `--out` is
  gone and `--dir` is optional (defaults under the data dir). Either DELETE this
  test (the requirement no longer holds) OR repurpose it to assert the default:
  rename to `TestExportDefaultsUnderDataDir`, run with NO `--dir`, and assert
  `data["dest"]` equals `filepath.Join(app.DataDir(), "export")`. Prefer the
  repurpose — it documents the default. (storetest's app has a real `DataDir()`
  under its temp root, so the export writes there harmlessly and is cleaned up.)

**Verify**: `go test ./internal/cli/` → `ok`, both tests pass.

### Step 7: Update `internal/self/knowledge.md`

Lines 321-326 describe `balaur export` as a SPIKE. Update to the shipped mirror.
Replace the SPIKE paragraph with copy like: "`balaur export` writes a one-way
Johnny Decimal Markdown mirror of every owner-authored, active node into
`<data dir>/export` (or `--dir`), grouped into JD folders, full re-export
(byte-identical for unchanged data), and committed to a git history under the
dest (skipped cleanly if git is absent). The redaction boundary holds: it reads
only `status=active` rows of the `nodes` collection, never any secret/token
collection. `day`/`task` are deferred pending their own redaction pass.
Encryption is Phase 3." Keep the design-doc path reference.

**Verify**: `gofmt`/`go vet` unaffected (it is Markdown). `git diff --check` →
no output.

### Step 8: Flip the README roadmap ledger — ONLY if Phase 2 is truly complete

`README.md:425` reads:
```
- Johnny Decimal Markdown vault mirror: one-way export + git history
```
under `## Roadmap (not shipped — honesty ledger)`.

**Gate:** flip this to shipped (move/remove the line, or add a "shipped (plan
194)" note) ONLY IF ALL of: ExportMirror covers every mapped owner type +
Unsorted fallback, full re-export is byte-identical, git commit works, the
extended canary passes, and the verb emits `{files, dest}`. If `day`/`task` being
deferred makes you judge Phase 2 "not truly complete" for the ledger's honesty
standard, LEAVE THE LINE as-is and note the deferral in your summary — the brief
permits leaving it. **Recommended:** the mirror IS the shipped feature even with
day/task deferred (the design itself defers them in Phase 2), so flipping the line
to shipped with a parenthetical "(day/task pages deferred to a later redaction
pass)" is honest and correct. Do NOT touch the `Encrypted export` line (line 428)
— that is Phase 3.

**Verify**: `git diff README.md` shows only the one ledger line changed.

### Step 9: Full-suite gate + the plans index row

- Run the full suite and static checks (see Done criteria).
- Add a row to `plans/README.md` (the table around lines 189-197) following the
  existing format, e.g.:
  `| 194 | Sovereign export Phase 2: full Johnny Decimal Markdown vault mirror (+ git, real export verb) | P3 | L | — | DONE |`

**Verify**: `go test ./...` → all `ok`.

## Test plan

New tests in `internal/export/mirror_test.go` (reuse `readAll` from
`export_test.go`, same `package export_test`):
- `TestMirrorLayoutPerType` — each mapped non-deferred type lands in its JD folder.
- `TestMirrorByteIdenticalReexport` — second run over unchanged data is byte-identical.
- `TestMirrorUnmappedTypeGoesUnsorted` — `measure` (seeded, owner-authored, unmapped) → `90-99 Unsorted/91 Other/`.
- `TestMirrorSkipsDeferredTypes` — an active `day` node produces NO file (deferral guard).
- `TestMirrorNeverLeaksStoredSecret` — extended canary over the full mirror; manually verified it FAILS when a secret read is introduced (then reverted).
- `TestMirrorGitCommit` — one commit created when git present; clean `t.Skip` when absent.

Updated tests in `internal/cli/export_test.go`:
- `TestExportEmitsEnvelopeAndWritesFiles` — `--dir`, `{files, dest}` envelope, JD-prefixed path.
- `TestExportDefaultsUnderDataDir` (repurposed) — no `--dir` defaults to `<data dir>/export`.

Structural pattern to copy: the existing `internal/export/export_test.go`
(`readAll`, `nodes.Create`, `storetest.NewApp`, `t.TempDir`) and
`internal/cli/export_test.go` (`executeEnvelope`).

Verification: `go test ./internal/export/ ./internal/cli/` → all pass including
the new tests; then `go test ./...` → all `ok`.

## Done criteria

Machine-checkable. ALL must hold (with `export TMPDIR=/home/alex/.cache/go-tmp` set):

- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./internal/export/` passes, including the 6 new mirror tests.
- [ ] `go test ./internal/cli/` passes, including the updated export tests.
- [ ] `go test ./...` all `ok`.
- [ ] `go vet ./...` exits 0 with no output.
- [ ] `gofmt -l internal/export/ internal/cli/` prints nothing.
- [ ] `git diff --check` prints nothing.
- [ ] `ExportMirror` reads ONLY `nodes` (via `nodes.OwnerAuthoredTypes` +
      `nodes.ListByTypeStatus`); a grep of `internal/export/export.go` for other
      collection names returns nothing:
      `grep -nE "llm_providers|llm_models|extensions|owner_settings|audit_log|conversations|messages" internal/export/export.go` → empty.
- [ ] The extended canary `TestMirrorNeverLeaksStoredSecret` passes, AND you
      manually confirmed it FAILS when a stray secret read is introduced (then
      reverted) — recorded in your summary.
- [ ] `day`/`task` produce NO exported files (`TestMirrorSkipsDeferredTypes` green).
- [ ] The `balaur export` verb emits `{"v":1,"kind":"export","data":{"files":[...],"dest":"..."}}`.
- [ ] `internal/self/knowledge.md` describes the shipped mirror (no longer "SPIKE").
- [ ] `README.md` ledger line handled per Step 8's gate.
- [ ] `plans/README.md` status row for 194 added.
- [ ] No files outside the in-scope list modified (`git status`).

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `e06346d` and its live
  code no longer matches the "Current state" excerpts.
- `nodes.OwnerAuthoredTypes` returns a type beyond `{note, person, book, idea,
  place, measure, day, task}` whose faithful export would need an undesigned
  redaction pass (e.g. a new transcript-bearing type). DEFER it: add it to
  `deferredTypes` (skip) and report — NEVER export unredacted transcript content.
- The redaction canary leaks the seeded secret with the committed code (no stray
  read) — that is a real boundary breach; stop and report, do not string-filter
  output to hide it.
- A required-props validation in `nodes.Create` blocks seeding a `measure` (or
  other) node for a test — read that type's schema migration, pass the required
  props, and only stop if the schema is genuinely incompatible with a fixture.
- Wiring git into `ExportMirror` forces a network call or a remote — it must be
  fully offline; if you cannot make it offline, stop.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (especially
  `internal/web/*`, any encryption code, or removing `ExportType`).

## Maintenance notes

For the human/agent who owns this after it lands:

- **Phase 3 (plan 195, encryption)** layers an `age`/archive over THIS mirror
  tree (design Q5). It must stay CGO-free. It should encrypt the directory
  `ExportMirror` produced, not re-export secret state.
- **Lifting the day/task deferral** is a future plan: it must add a recap/
  transcript redaction pass AND, in the same change, a leak test, BEFORE removing
  `day`/`task` from `deferredTypes`. The JD folders for them are already recorded
  in the design table.
- **New owner-authored types** automatically land in `90-99 Unsorted/91 Other/`
  via `jdFolderFor`. When a type deserves its own JD folder, add it to `jdFolder`
  — and if it carries transcript/secret-adjacent content, add it to
  `deferredTypes` with a leak test until reviewed.
- A reviewer should scrutinize: (1) that `ExportMirror` has NO read of any
  collection but `nodes`; (2) the byte-identical re-export holds (no `time.Now()`
  in render/write/commit-message); (3) the git commit is offline and skips
  cleanly without git; (4) `day`/`task` never produce files.
- Incremental export (manifest/mtime) remains deferred (design Q3) — only revisit
  when a real vault is large enough to feel full-rewrite cost.
