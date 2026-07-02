# Plan 242: Prune files for no-longer-active nodes from the export mirror's managed folders

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/export/export.go internal/export/mirror_test.go internal/self/knowledge.md .tours/15-sovereign-export.tour plans/README.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. (Exception: `plans/README.md` is
> pure status-row bookkeeping with no "Current state" excerpt — churn there
> from other plans landing is expected and is NOT drift.)

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: MED
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

`balaur export` writes a one-way Markdown mirror of every ACTIVE node into a
Johnny Decimal folder tree and commits it to a git history under the
destination. The writer only ever ADDS files: nothing removes the file of a
node that was archived or dropped (consent withdrawn from the active set), or
the old-slug file left behind when a node is retitled. The default destination
is persistent (`<data dir>/export`), and the commit step stages with
`git add -A`, so stale plaintext accumulates in the working tree and is
re-committed on every run — the mirror silently violates its own framing that
the tree contains exactly one file per ACTIVE node. This plan adds a prune
pass, scoped strictly to the folders the exporter manages, so an archived or
renamed node's old file disappears from the working tree on the next export
(and `git add -A` records the deletion). Git HISTORY retention is inherent to
a git mirror and is NOT the defect; the working tree is.

## Current state

Relevant files:

- `internal/export/export.go` — the sovereign-export mirror. `ExportMirror`
  (lines 109–154) writes the tree; `commitMirror` (lines 162–218) commits it;
  `jdFolder`/`unsortedFolder` (lines 59–73) define which folders the exporter
  owns.
- `internal/export/mirror_test.go` — the mirror test suite (`storetest.NewApp`
  temp-dir PocketBase apps, `readAll` walker from `export_test.go:17-39`).
  Contains the redaction canary `TestMirrorNeverLeaksStoredSecret` (line 212)
  which "must never be deleted or weakened", and the leak test
  `TestDayJournalExportLeakTest` (line 295).
- `internal/export/export_test.go` — `ExportType` tests + the shared `readAll`
  helper. Not modified by this plan.
- `internal/cli/export.go` — the CLI surface; shows the persistent default
  destination. Not modified by this plan.
- `internal/self/knowledge.md` — the binary's self-description; its export
  paragraph (lines 321–327) describes mirror behavior and gains one clause.
- `.tours/15-sovereign-export.tour` — maintained tour anchored into
  `internal/export/export.go` at lines 1, 59, 109, 223 and
  `internal/export/mirror_test.go` at line 229; this change shifts THREE
  anchors — 59 (`jdFolder`, pushed down by the two new import lines at the
  top of the file), 109 (`ExportMirror`, pushed down by the imports plus the
  doc-comment insertion), and 223 (`render`, pushed down by all of the above
  plus the wiring inside `ExportMirror`). Anchor 1 (the package comment) and
  mirror_test.go's 229 stay put.

### The write loop only adds files

`internal/export/export.go:116-154` (inside `ExportMirror`):

```go
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
```

Every written relative path is `path.Join(relDir, name)` — slash-separated,
relative to `destDir`, and `relDir` is always a managed folder
(`jdFolderFor(typ)`). Nothing here ever deletes a file.

### The managed folders

`internal/export/export.go:59-73`:

```go
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
```

A separate future plan may un-defer `task` by adding it to `jdFolder`. The
prune loop MUST therefore derive the managed-folder set dynamically from
`jdFolder`'s values plus `unsortedFolder` — never a second hard-coded list —
so the two changes compose without coordination.

### The destination is persistent and the commit stages everything

`internal/cli/export.go:42-45`:

```go
		dest := out
		if dest == "" {
			dest = filepath.Join(app.DataDir(), "export")
		}
```

`internal/export/export.go:205` (inside `commitMirror`):

```go
	if err := run("add", "-A"); err != nil {
```

So a stale file survives run after run and is re-committed each time. (The
`--encrypt` path in `internal/cli/export.go:71-77` renders into a fresh
`os.MkdirTemp` dir wiped by `defer os.RemoveAll(tmp)` — it is immune and needs
no change.)

### Repo conventions that apply here

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in library
  code. Match the existing `"export: ..."` prefixes in this file.
- No global mutable state; no `fmt.Print*` in service code. The prune helper
  needs no logging at all.
- Tests: standard `testing` package; PocketBase-dependent logic uses
  `storetest.NewApp(t)` (see `internal/export/mirror_test.go:23`); no
  `time.Sleep`-based synchronization. Model new tests on the existing mirror
  tests in that file.
- Node lifecycle helpers (from `internal/nodes/nodes.go`):
  `nodes.Create(app, typ, title, body, nodes.StatusActive, props)` creates;
  `nodes.Transition(app, id, nodes.StatusArchived, "node")` archives (the
  active→archived transition is valid per `ValidTransitions`, line 40-45);
  `nodes.Update(app, id, &newTitle, nil, nil)` retitles (title is a
  `*string`).
- `.tours/` are maintained artifacts: `tours_test.go` fails on missing files /
  out-of-range lines — it does NOT check what a line contains, so a
  mis-pointed anchor passes `TestTours` silently. This change shifts the
  `jdFolder` (59), `ExportMirror` (109), and `render` (223) anchors in
  `internal/export/export.go` — Step 4 repoints all three by grep.
- KISS/YAGNI: smallest correct change — plain-file removal only, no directory
  cleanup, no config knob.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted export tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1` | exit 0, all pass |
| One test | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -run TestMirrorPrunes -count=1 -v` | listed tests PASS |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |

(Note: the host `/tmp` is a small tmpfs and the Go linker OOMs there —
always keep the `TMPDIR=$HOME/.cache/go-tmp` prefix on test commands.)

## Scope

**In scope** (the only files you should modify):

- `internal/export/export.go` — add the prune pass to `ExportMirror`.
- `internal/export/mirror_test.go` — new prune tests (append at END of file).
- `internal/self/knowledge.md` — one clause in the export paragraph (Step 5).
- `.tours/15-sovereign-export.tour` — repoint the three shifted anchors, one
  prose addition (Step 4).
- `plans/README.md` — status row only.

**Out of scope** (do NOT touch, even though they look related):

- `internal/export/encrypt.go` — the encrypt path renders into a wiped temp
  dir; it has no stale-file problem.
- `internal/export/export_test.go` — `ExportType` is the one-type primitive
  writing a flat directory with no managed-folder structure; extending prune
  to it is a separate behavior decision. Leave the file and the function
  untouched (you still USE its `readAll` helper from tests — same package).
- `internal/cli/export.go` — no CLI change is needed; prune lives below the
  gateway line.
- The `deferredTypes` set and the `jdFolder` map contents — un-deferring
  `task` is a different plan; do not add or remove entries.
- Git history rewriting of the export repo — history retention is inherent to
  a git mirror, explicitly not this plan's defect.
- `TestMirrorNeverLeaksStoredSecret` and `TestDayJournalExportLeakTest` — the
  redaction canaries stay byte-for-byte untouched and must stay green.

## Git workflow

- The executor runs in an isolated git worktree branched from `origin/main`;
  branch name `advisor/242-export-mirror-prune-stale`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/
  `chore`); e.g. `fix(export): prune stale files from the mirror's managed folders`.
- Commit per logical unit with explicit pathspecs (the main checkout is shared
  by parallel agents — stage only your own files, never `git add -A` in the
  repo).
- NEVER push; the reviewer merges.

## Steps

### Step 1: Add `pruneStale` and call it from `ExportMirror` before the commit

In `internal/export/export.go`, add a new unexported function at the END of
the file (after `uniqueName`, currently line 294-301 — placing it last keeps
the new function itself below every existing tour anchor; note the import
and doc-comment edits later in this step still shift the anchors at lines
59, 109, and 223 — Step 4 repoints them):

```go
// pruneStale removes every *.md file under destDir's MANAGED folders (the
// jdFolder values plus the Unsorted bucket) that this run did not write, so
// the working tree holds exactly one file per ACTIVE node: a node archived,
// dropped, or renamed to a new slug loses its old file on the next export and
// `git add -A` records the deletion. The managed set is derived from jdFolder
// dynamically so a newly mapped type is covered without touching this code.
// Everything OUTSIDE the managed folders — owner files elsewhere under
// destDir, the .git directory, non-.md files even inside managed folders —
// is never touched. Managed folders are owned by the exporter: any .md file
// in one that does not correspond to a written path is removed, whoever put
// it there. Emptied directories are left in place (git does not track them).
func pruneStale(destDir string, written []string) error {
	managed := make([]string, 0, len(jdFolder)+1)
	for _, f := range jdFolder {
		managed = append(managed, f)
	}
	managed = append(managed, unsortedFolder)
	slices.Sort(managed) // deterministic removal order

	keep := make(map[string]bool, len(written))
	for _, p := range written {
		keep[p] = true
	}
	for _, rel := range managed {
		absDir := filepath.Join(destDir, filepath.FromSlash(rel))
		entries, err := os.ReadDir(absDir)
		if errors.Is(err, fs.ErrNotExist) {
			continue // folder never written — nothing to prune
		}
		if err != nil {
			return fmt.Errorf("export: reading %s: %w", absDir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			if keep[path.Join(rel, e.Name())] {
				continue
			}
			if err := os.Remove(filepath.Join(absDir, e.Name())); err != nil {
				return fmt.Errorf("export: pruning stale %s: %w", filepath.Join(absDir, e.Name()), err)
			}
		}
	}
	return nil
}
```

Add `"errors"` and `"io/fs"` to the import block at
`internal/export/export.go:12-25`.

Then wire it into `ExportMirror`, between `slices.Sort(written)` (line 145)
and the `commitMirror` call (line 150), so the deletions land in the same
commit as the writes:

```go
	slices.Sort(written)

	// Prune stale files (archived/dropped/renamed nodes) from the managed
	// folders BEFORE committing, so `git add -A` records the deletions in the
	// same commit as the writes.
	if err := pruneStale(destDir, written); err != nil {
		return nil, err
	}

	// Commit the mirror to a local git history. ...
```

Finally, extend the `ExportMirror` doc comment (lines 92-108): after the
sentence "After writing the tree it commits the mirror to a git history under
destDir (offline; skipped cleanly when git is absent).", insert:

```
Between writing and committing it prunes stale *.md files from the managed
JD folders (see pruneStale), so the tree holds exactly one file per ACTIVE
node.
```

Note `slices`, `path`, `filepath`, `os`, `strings`, `fmt` are already
imported; only `errors` and `io/fs` are new.

**Verify**:
`gofmt -l . && go vet ./... && CGO_ENABLED=0 go build ./...` → empty gofmt
output, exit 0 overall.

**Verify** (no regression):
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1` → ok, all
existing tests pass (including `TestMirrorNeverLeaksStoredSecret`,
`TestDayJournalExportLeakTest`, `TestMirrorByteIdenticalReexport`,
`TestMirrorGitCommit`).

### Step 2: Add the prune tests

Append the following tests at the END of `internal/export/mirror_test.go`
(after `slugForTest`, currently line 392-407 — appending keeps the tour's
`mirror_test.go:229` anchor stable). Model setup/assertions on the existing
tests in the same file (`storetest.NewApp(t)`, `t.TempDir()`, `readAll`).
Exact test names matter — the Done criteria grep for them.

1. **`TestMirrorPrunesArchivedNode`** — the consent-withdrawal case:
   - Create two active notes, e.g. `nodes.Create(app, "note", "Keep Me", ...)`
     and `nodes.Create(app, "note", "Archive Me", ...)` (capture the second
     record to get its `.Id`).
   - `export.ExportMirror(app, dir)`; assert both
     `10-19 Knowledge/11 Notes/keep-me.md` and
     `10-19 Knowledge/11 Notes/archive-me.md` exist on disk.
   - Archive the second: `nodes.Transition(app, rec.Id, nodes.StatusArchived, "node")`.
   - Re-export; assert `archive-me.md` is GONE from disk
     (`os.Stat` → `os.IsNotExist`), `keep-me.md` still exists, and the
     returned `written` slice does not contain the archived path.

2. **`TestMirrorPrunesRenamedNode`** — the retitle case:
   - Create one active note "Old Title"; export; assert
     `10-19 Knowledge/11 Notes/old-title.md` exists.
   - Retitle: `newTitle := "New Title"; nodes.Update(app, rec.Id, &newTitle, nil, nil)`.
   - Re-export; assert exactly one file for the node remains:
     `new-title.md` exists, `old-title.md` is gone. Also assert the notes
     folder contains exactly one `.md` file (walk `readAll(t, dir)` counting
     entries with prefix `10-19 Knowledge/11 Notes`, or `os.ReadDir` the
     folder).

3. **`TestMirrorPruneSparesOwnerFiles`** — the boundary invariant. The rule to
   assert (and which the plan documents): *prune touches only `*.md` files
   directly inside managed JD folders; everything else under destDir
   survives.*
   - Create one active note; export once.
   - Plant three owner files:
     - `destDir/notes.txt` (outside any managed folder),
     - `destDir/My Stuff/mine.md` (an `.md` file, but in an UNMANAGED dir —
       create the dir with `os.MkdirAll`),
     - `destDir/10-19 Knowledge/11 Notes/owner.txt` (non-`.md` INSIDE a
       managed folder).
   - Also plant `destDir/10-19 Knowledge/11 Notes/stray.md` — an `.md` file
     inside a managed folder that no active node produced.
   - Re-export. Assert `notes.txt`, `My Stuff/mine.md`, and
     `11 Notes/owner.txt` all still exist; assert `stray.md` is REMOVED
     (managed folders are exporter-owned — this is the documented rule, and
     it is exactly what makes the archived/renamed cases work).

4. **Determinism holds** — no new test needed: the existing
   `TestMirrorByteIdenticalReexport` (mirror_test.go:77) already asserts a
   second run over unchanged data is byte-identical; it must still pass
   unmodified. Likewise `TestMirrorGitCommit` (line 244) must still see a
   clean working tree after the commit.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -run 'TestMirrorPrunes|TestMirrorPruneSpares' -count=1 -v`
→ `PASS` for `TestMirrorPrunesArchivedNode`, `TestMirrorPrunesRenamedNode`,
`TestMirrorPruneSparesOwnerFiles`.

### Step 3: Run the whole export package uncached

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1` → ok, ALL
tests pass. Confirm with `git diff --stat internal/export/export_test.go` →
no output (the canary/`ExportType` test file is untouched), and
`git diff internal/export/mirror_test.go | grep '^-[^-]' | wc -l` → `0`
(no deleted lines from the existing tests — your change is append-only there).

### Step 4: Fix the tour

`.tours/15-sovereign-export.tour` anchors `internal/export/export.go` at
lines 1, 59, 109, 223 and `internal/export/mirror_test.go` at line 229. Your
Step 1 edits shift THREE of those anchors: the two new import lines
(`"errors"`, `"io/fs"` in the block at 12-25) sit above line 59, so the
`jdFolder` anchor (59) and everything below it moves down; the doc-comment
insertion before `func ExportMirror` shifts its anchor (109) further; the
wiring inside `ExportMirror` plus all of the above shift the `render` anchor
(223). Anchor 1 (the package comment) stays put, and mirror_test.go's 229
stays put because Step 2 appended at the end. Beware: `tours_test.go` checks
only that the anchored file exists and the line is in range — NOT what the
line contains — so a stale anchor passes `TestTours` silently. Repoint by
grep; the grep output is the truth:

1. Find the new anchor lines:
   - `grep -n "^var jdFolder" internal/export/export.go` → new line for the
     old-59 anchor.
   - `grep -n "^func ExportMirror" internal/export/export.go` → new line for
     the old-109 anchor.
   - `grep -n "^func render" internal/export/export.go` → new line for the
     old-223 anchor.
2. In the tour JSON, update the three steps whose `"file"` is
   `internal/export/export.go` and `"line"` is `59`, `109`, or `223` to the
   corresponding new numbers from the greps.
3. In the tour step formerly anchored at `export.go` line 109 (title
   "15.2 — The mirror writer"), its `"description"` lists numbered behaviors
   ending with "6. After writing, calls `commitMirror(destDir)` ...". Append
   one sentence to that description (inside the JSON string, escape as
   needed):
   `Between writing and committing, pruneStale removes any stale .md file from the managed JD folders (archived/dropped/renamed nodes), so the working tree always holds exactly one file per ACTIVE node and git add -A records the deletions.`

**Verify**: for each of the three greps in item 1, the tour step's `"line"`
equals the grep's line number (this is the real anchor check — `TestTours`
cannot catch a mis-pointed anchor).
**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok.
**Verify**: `python3 -c "import json; json.load(open('.tours/15-sovereign-export.tour'))"` → no error (the file is still valid JSON).

### Step 5: Update the self-description

`internal/self/knowledge.md:324` currently reads (mid-sentence):

```
`90-99 Unsorted/91 Other`), full re-export (byte-identical for unchanged data),
```

Change that clause to:

```
`90-99 Unsorted/91 Other`), full re-export (byte-identical for unchanged data;
stale files of archived or renamed nodes are pruned from the managed JD folders),
```

This is required: the self-description must not claim less (or more) than the
binary does, and prune is user-visible mirror behavior.

**Verify**: `grep -n "pruned from the managed JD folders" internal/self/knowledge.md` → one match.

### Step 6: Full gate and lint

**Verify** (each in turn):

- `gofmt -l .` → empty output
- `go vet ./...` → exit 0
- `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
- `CGO_ENABLED=0 go build ./...` → exit 0
- `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all packages pass
- `git diff --check` → no output

Commit per logical unit with explicit pathspecs, e.g.:

```
git add internal/export/export.go internal/export/mirror_test.go
git commit -m "fix(export): prune stale files from the mirror's managed folders"
git add .tours/15-sovereign-export.tour internal/self/knowledge.md
git commit -m "docs(export): tour anchor + self-knowledge for mirror pruning"
```

Then update the plan-242 row in `plans/README.md` (add the row if absent) and
commit it with an explicit pathspec.

## Test plan

- New tests, all in `internal/export/mirror_test.go`, appended after
  `slugForTest`, modeled structurally on `TestMirrorLayoutPerType` /
  `TestMirrorSkipsDeferredTypes` in the same file:
  - `TestMirrorPrunesArchivedNode` — archive-withdraws consent → file removed
    on re-export (the core bug this plan fixes).
  - `TestMirrorPrunesRenamedNode` — retitle → old-slug file removed, exactly
    one file per node.
  - `TestMirrorPruneSparesOwnerFiles` — owner files outside managed folders,
    `.md` files in unmanaged dirs, and non-`.md` files inside managed folders
    all survive; a stray `.md` inside a managed folder is removed (the
    documented ownership rule).
- Existing tests that must pass UNMODIFIED: `TestMirrorByteIdenticalReexport`
  (determinism), `TestMirrorGitCommit` (clean tree after commit),
  `TestMirrorSkipsDeferredTypes`, `TestMirrorNeverLeaksStoredSecret` and
  `TestDayJournalExportLeakTest` (the redaction canaries — byte-for-byte
  untouched), plus all of `export_test.go`.
- Verification:
  `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1 -v` → all
  pass, including the 3 new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output;
      `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `grep -c "func TestMirrorPrunesArchivedNode\|func TestMirrorPrunesRenamedNode\|func TestMirrorPruneSparesOwnerFiles" internal/export/mirror_test.go` → `3`
- [ ] `grep -n "pruneStale(destDir, written)" internal/export/export.go` → one
      match, on a line BEFORE the `commitMirror(destDir)` call
- [ ] `git diff origin/main...HEAD -- internal/export/export_test.go internal/export/encrypt.go internal/cli/export.go` → empty (YOUR changes leave the out-of-scope export files untouched; the three-dot merge-base diff scopes the check to the executor's own commits, not upstream drift since `077318a`)
- [ ] `git diff origin/main...HEAD --stat` lists ONLY files from this set:
      `internal/export/export.go`, `internal/export/mirror_test.go`,
      `internal/self/knowledge.md`, `.tours/15-sovereign-export.tour`,
      `plans/README.md`, `plans/242-export-mirror-prune-stale.md`
      (status/plan bookkeeping). The worktree bases off `origin/main` at
      execution time — commits landed after `077318a` are upstream, not
      yours, and the three-dot diff correctly excludes them.
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `grep -n "pruned from the managed JD folders" internal/self/knowledge.md` → one match
- [ ] `plans/README.md` status row for 242 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The "Current state" excerpts do not match the live code — in particular if
  `ExportMirror`'s written-path bookkeeping no longer builds slash-relative
  `path.Join(relDir, name)` entries, or `jdFolder`/`unsortedFolder` have been
  restructured (e.g. task un-deferred, folders renamed). The prune's exact
  set-difference depends on `written` being the complete, slash-relative truth
  of this run.
- `TestMirrorNeverLeaksStoredSecret`, `TestDayJournalExportLeakTest`, or
  `TestMirrorByteIdenticalReexport` fails after your change, or making them
  pass appears to require MODIFYING any of them — the canaries and the
  determinism contract are load-bearing; a change that needs to weaken them is
  wrong.
- Any existing test turns out to assert that stale files PERSIST across
  re-exports (none was found when this plan was written — if one exists, the
  contract question goes back to the owner).
- `TestMirrorGitCommit` starts failing on the "working tree not clean"
  assertion — that would mean prune runs after the commit or removes something
  git still expects.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (e.g. changing
  `ExportType`, `deferredTypes`, or `internal/cli/export.go`).

## Maintenance notes

- **Composes with un-deferring `task`**: the managed-folder set is derived
  from `jdFolder` + `unsortedFolder` at call time. When a future plan adds
  `"task": "60-69 Tasks/61 Tasks"` to `jdFolder`, that folder becomes managed
  automatically — no prune change needed. Reviewers should reject any edit
  that re-introduces a second hard-coded folder list.
- **Renaming a JD folder orphans the old one**: if a `jdFolder` VALUE is ever
  changed (e.g. `11 Notes` → `11 Notebook`), the old folder is no longer
  managed and its files linger in existing export destinations. Such a plan
  must handle its own one-time cleanup.
- **Pre-existing latent issue, intentionally untouched**: the per-type
  `used := map[string]bool{}` collision map in `ExportMirror` (export.go:135)
  claims "two nodes of different types never share a folder", but TWO UNMAPPED
  types both fall back to `unsortedFolder` and could silently overwrite each
  other on a slug collision. Out of scope here (prune operates on the union
  `written` set and is unaffected) — worth its own small fix.
- **What a reviewer should scrutinize**: that prune runs strictly BEFORE
  `commitMirror` (deletions land in the same commit); that nothing outside the
  managed folders can ever be removed (the `managed` slice is the only walk
  root, and `.git` is not in it); that the canary tests are byte-for-byte
  unchanged (`git diff` on `mirror_test.go` shows only appended lines).
- **Explicitly deferred**: removing newly-emptied managed subdirectories
  (plain-file removal suffices — git does not track empty dirs, so the mirror
  repo stays correct); pruning inside `ExportType`'s flat output; any rewrite
  of the export repo's git history.
