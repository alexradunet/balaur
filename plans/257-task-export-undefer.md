# Plan 257: Un-defer task export — the last owner-authored type missing from the vault mirror

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/export/export.go internal/export/mirror_test.go .tours/15-sovereign-export.tour internal/self/knowledge.md AGENTS.md`
> This plan DEPENDS on two earlier plans that touch the same files
> (`plans/242-export-mirror-prune-stale.md` adds a `pruneStale` pass to
> `internal/export/export.go` + prune tests to `mirror_test.go` + one
> knowledge.md clause + a tour-anchor repoint; and
> `plans/252-docs-truth-sync-post-230-234.md` rewrites tour 15's prose so only
> `task` reads as deferred, and rewrites the AGENTS.md vault-mirror bullet).
> Diffs attributable to those two plans are EXPECTED — Step 0 verifies they
> landed. Any OTHER divergence from the "Current state" excerpts below is a
> STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/242-export-mirror-prune-stale.md, plans/252-docs-truth-sync-post-230-234.md (all three touch `internal/export/export.go` and/or tour 15 — land 242 and 252 first; this plan is written against their landed state)
- **Category**: direction
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

`balaur export` writes a one-way Johnny Decimal Markdown mirror of every
owner-authored, ACTIVE node — except `task`, the one type still parked in
`deferredTypes` "until its content redaction pass lands (a future plan)". This
plan IS that future plan. The exit criterion is written into the code itself
(`internal/export/export.go`): "Do NOT remove a type from this set without
adding its redaction pass + leak test." Plan 225 established the exact recipe
when it un-deferred `day`: pre-verify what the node body and props actually
carry, seed marker strings into every adjacent surface the type touches, and
assert those markers never appear anywhere in the export tree. `PRODUCT.md`'s
thirty-day success criterion says the owner must "own it — their conversation,
tasks, journal, and life-log are all theirs in SQLite" (PRODUCT.md:81-82) and
"never in a vendor's database" (PRODUCT.md:62); the Markdown mirror is that
ownership made portable, and today it silently omits tasks. After this plan,
every owner-authored type exports, each gated by a leak test.

## Current state

Relevant files:

- `internal/export/export.go` — the sovereign-export mirror. `jdFolder` maps
  node types to JD folders; `deferredTypes` skips types pending a redaction
  pass; `ExportMirror` writes the tree; `render` emits YAML frontmatter
  (type/status/created/updated + each props scalar, sorted) + H1 title + body
  verbatim.
- `internal/export/mirror_test.go` — the mirror test suite
  (`storetest.NewApp(t)` temp-dir PocketBase apps; the recursive `readAll`
  walker lives in the sibling `internal/export/export_test.go:17-39`).
  Contains the deferral GUARD `TestMirrorSkipsDeferredTypes` (asserts a task
  file exists NOWHERE in the tree — it will correctly FAIL when task is
  un-deferred and is replaced in Step 4), the day leak test
  `TestDayJournalExportLeakTest` (the plan-225 recipe this plan mirrors), and
  the secret canary `TestMirrorNeverLeaksStoredSecret` which "must never be
  deleted or weakened".
- `internal/tasks/tasks.go` — the task domain package (tasks are `type=task`
  nodes). Read-only for this plan; it is the evidence for the redaction
  pre-verification in Step 1.
- `internal/tasks/nudge.go` — the nudger. Read-only; evidence that nudge text
  goes to the `messages` collection, never into the task node.
- `.tours/15-sovereign-export.tour` — maintained tour of `internal/export`;
  anchors and prose must track this change (`tours_test.go` fails the suite
  on out-of-range line anchors).
- `internal/self/knowledge.md` — the binary's self-description; its export
  paragraph (around lines 321-328 at `077318a`) says task is deferred.
- `AGENTS.md` — after plan 252 lands, its vault-mirror bullet says "Only the
  `task` type stays deferred pending its content redaction pass … Do not
  claim task export in user-facing copy until that redaction pass lands."
  This plan makes that claim stale and must update it.

### The deferral and its exit criterion

`internal/export/export.go:59-81` at `077318a` (plan 242 may have shifted
these lines slightly — the CONTENT must match):

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
```

```go
// deferredTypes are owner-authored types whose faithful export needs its own
// redaction pass. Exporting them raw could surface un-reviewed content, so they
// are skipped entirely until their redaction pass + leak test land.
// Do NOT remove a type from this set without adding its redaction pass + leak test.
var deferredTypes = map[string]bool{
	"task": true,
}
```

And the `ExportMirror` doc comment, `internal/export/export.go:101-103` at
`077318a`:

```go
// task is DEFERRED (deferredTypes): its content needs its own redaction pass, so
// it is skipped here. day was un-deferred in plan 225 (leak test proved the body
// carries only the owner's journal text — recap/summaries never touch the body).
```

`ExportMirror` skips deferred types before any read
(`internal/export/export.go:117-120` at `077318a`):

```go
	for _, typ := range types {
		if deferredTypes[typ] {
			continue
		}
```

### What render emits per node

`internal/export/export.go:220-246` at `077318a` — `render` writes YAML
frontmatter `type`/`status`/`created`/`updated` plus EVERY props key (sorted,
`%q`-quoted via `yamlLine`), then `# <title>`, then the body verbatim. So
un-deferring task means: every task props scalar lands in frontmatter, and the
task body lands in the file. That is why Step 1's pre-verification of what
tasks actually carry is the real work here.

### What a task node carries (advisor-verified; executor re-verifies in Step 1)

Tasks are `type=task` nodes. The full property schema, from
`migrations/1750000020_tasks_to_nodes.go:47-56`:

```go
	taskProps := []propDef{
		{Key: "state", Label: "State", Type: "select", Required: true, Options: []string{"open", "done", "dropped"}},
		{Key: "due", Label: "Due", Type: "text"},
		{Key: "recur", Label: "Recurrence", Type: "text"},
		{Key: "recur_from_done", Label: "Recur from done", Type: "bool"},
		{Key: "snoozed_until", Label: "Snoozed until", Type: "text"},
		{Key: "nudged_at", Label: "Nudged at", Type: "text"},
		{Key: "done_at", Label: "Done at", Type: "text"},
		{Key: "source", Label: "Source", Type: "text"},
	}
```

**Body** = owner notes only. `internal/tasks/tasks.go:52` (Create):

```go
	rec, err := nodes.Create(app, "task", title, strings.TrimSpace(o.Notes), nodes.StatusActive, props)
```

and `internal/tasks/tasks.go:121-124, 156-157` (Update):

```go
	body := rec.GetString("body")
	if o.Notes != nil {
		body = strings.TrimSpace(*o.Notes)
	}
```

```go
	rec.Set("title", title)
	rec.Set("body", body)
```

The only other body writer that can reach a task node is the owner's own edit
form (`internal/web/knowledge.go:146`, `rec.Set("body", e.Request.FormValue("body"))`
in `nodeEdit` — owner-typed text). No model-generated text is ever written
into a task node's body or title.

**Props writers**, all in `internal/tasks/` — every value is a workflow token,
an owner-set scheduling value, a provenance tag, or a timestamp Balaur wrote
on the owner's behalf:

- `Create` (tasks.go:42-50): `state="open"`, `recur` (owner-set rule string),
  `recur_from_done` (bool), `source` (provenance tag — callers pass `"cli"`
  (`internal/cli/task.go:84`), `"chat"` (`internal/tools/tasks.go:103`), or the
  seed marker), `due` (PB-time string).
- `Done` (tasks.go:201-202, 227-229): `state="done"` + `done_at`, or for
  recurring tasks a new `due` and `delete(props, "nudged_at")` /
  `delete(props, "snoozed_until")`.
- `Snooze` (tasks.go:260-261): `snoozed_until`, deletes `nudged_at`.
- `Drop` (tasks.go:278): `state="dropped"`.
- `Nudge` (nudge.go:116-117): `props["nudged_at"] = store.PBTime(now.UTC())` —
  a timestamp only.

**Adjacent surfaces a task touches — and why they can't leak:**

- Nudge TEXT (possibly model-composed via `composeNudge`) goes into the master
  conversation as a message — `internal/tasks/nudge.go:110-113`:

```go
	if err := app.RunInTransaction(func(txApp core.App) error {
		if err := conversation.AppendOrigin(txApp, master.Id,
			llm.Message{Role: "assistant", Content: text}, "", "nudge"); err != nil {
			return err
		}
```

  That is the `messages` collection, which `ExportMirror` never reads.

- Completion text goes into the `entries` collection —
  `internal/tasks/tasks.go:237`:

```go
		if err := addEntry(txApp, "completion", rec.Id, nil, rec.GetString("title"), now); err != nil {
```

  `entries` is also never read by the exporter (it reads ONLY `nodes`).

**Redaction decision this plan implements (make no other):** export ALL task
props unfiltered. `nudged_at` and `snoozed_until` are behavioral metadata, but
they are timestamps only — no content, no secrets, no third-party data — and
they are the owner's own record of the owner's own behavior, in the owner's
own local (optionally encrypted) mirror. Filtering them would add the first
per-type special case to the deliberately generic `render` for zero privacy
gain, and would make the mirror claim less than the database holds — the
opposite of PRODUCT.md's "own it". So: no props filtering, no body rewriting;
the redaction pass for task is the PROOF (the leak test) that nothing beyond
the node itself reaches the mirror, exactly as plan 225 did for `day`.

### The plan-225 leak test to mirror

`internal/export/mirror_test.go:288-307` at `077318a` (the pattern for Step 4):

```go
// TestDayJournalExportLeakTest is the redaction proof for day-journal export
// (plan 225). It seeds a day node with a known journal body AND a real record
// in the `summaries` collection with a DISTINCT marker string, then asserts:
//  1. The journal body appears in an exported file (day node is exported).
//  2. The recap marker appears in NO exported file (the exporter never opens
//     the summaries collection — the redaction boundary holds).
//  3. A second ExportMirror over unchanged data produces byte-identical files.
func TestDayJournalExportLeakTest(t *testing.T) {
```

### The deferral guard that must be replaced

`internal/export/mirror_test.go:152-206` at `077318a`: `TestMirrorSkipsDeferredTypes`
creates an active task node and asserts NO task file exists anywhere in the
tree ("It walks the WHOLE tree, so it bites if task is removed from
deferredTypes"). Once task is un-deferred this guard's assertion is inverted;
Step 4 deletes it (and its private helper `slugForTest`,
mirror_test.go:389-407, whose only caller it is) and replaces it with the task
leak test. Deleting `slugForTest` matters: leaving it unused fails staticcheck
(U1000).

### The docs/tour claims that become false

- `internal/self/knowledge.md:327` at `077318a` (one long line):
  "… `task` is deferred pending its own content redaction pass; `day` exports
  its owner-authored journal body (plan 225 — …)".
- `AGENTS.md` after plan 252: "Only the `task` type stays deferred pending its
  content redaction pass; `day` journal bodies export behind a leak test. Do
  not claim task export in user-facing copy until that redaction pass lands."
- `.tours/15-sovereign-export.tour` after plan 252: steps 15.1 / 15.2 / 15.4
  say `task` is absent from `jdFolder` / the only deferred type. The tour also
  anchors `internal/export/export.go` at lines 1, 59, 109 and `render`
  (originally 223; plan 242 repoints it), plus `internal/export/mirror_test.go`
  at line 229 (inside `TestMirrorNeverLeaksStoredSecret`) — this plan's edits
  shift the `ExportMirror`, `render`, and mirror_test anchors.
- The tour's rule sentence — "**Do not add a type here without simultaneously
  removing it from `deferredTypes` and landing a leak test.**" — must be KEPT;
  it stays true for future types.

### Repo conventions that apply here

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in library
  code. (This plan adds no new error paths in `export.go` — only map/comment
  edits — but the test code follows `t.Fatalf` style as in the existing file.)
- Tests: standard `testing` package; PocketBase-dependent logic uses
  `storetest.NewApp(t)` (see `internal/export/mirror_test.go:23`);
  `t.TempDir()` for output; no `time.Sleep`; no assertion frameworks. Model
  the new test on `TestDayJournalExportLeakTest` in the same file.
- Audit-after-write, structured logging, gomponents rules: not touched by this
  plan (no service-code logic changes, no UI).
- `internal/self/knowledge.md` is the binary's self-description — a change
  that alters user-visible capability (tasks now export) MUST update it in the
  same change (Step 6).
- `.tours/` are maintained artifacts; `tours_test.go` only catches missing
  files / out-of-range anchors, so falsified PROSE must be fixed by hand in
  the same change (Step 5).
- KISS/YAGNI: smallest correct change — no props filtering, no config knob, no
  removal of the (now empty) `deferredTypes` mechanism.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted export tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1` | exit 0, all pass |
| One test | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -run TestTaskExportLeakTest -count=1 -v` | PASS |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |

(The host `/tmp` is a small tmpfs and the Go linker OOMs there — keep the
`TMPDIR=$HOME/.cache/go-tmp` prefix on every test command. `make test` exports
it automatically but is cached; the `-count=1` uncached form is the gate.)

## Suggested executor toolkit

- If a `go-standards` skill is available, invoke it before writing the test in
  Step 4 (testing idioms: table-driven where it helps, `t.TempDir`, no sleep).

## Scope

**In scope** (the only files you should modify):

- `internal/export/export.go` — un-defer task (map entry + empty
  `deferredTypes` + comment updates).
- `internal/export/mirror_test.go` — delete the deferral guard +
  `slugForTest`; add `TestTaskExportLeakTest`.
- `.tours/15-sovereign-export.tour` — prose + anchor updates.
- `internal/self/knowledge.md` — one clause in the export paragraph.
- `AGENTS.md` — the vault-mirror bullet (post-plan-252 text).
- `plans/README.md` — status row only.

**Out of scope** (do NOT touch, even though they look related):

- `internal/export/encrypt.go` and the `--encrypt`/`restore` path — it wraps
  whatever `ExportMirror` wrote; task export flows through automatically.
- `internal/export/export_test.go` — `ExportType` is the one-type primitive
  and already takes any type by name; no change. (You still USE its `readAll`
  helper — same test package.)
- `internal/cli/export.go` and any CLI flag — no gateway change; the mirror
  gains a type below the gateway line.
- `internal/tasks/*` — read-only evidence for Step 1; no task-domain change.
- Plan 242's `pruneStale` and its tests — its managed-folder set is derived
  from `jdFolder` at call time, so adding `"task"` makes `60-69 Tasks/61 Tasks`
  managed automatically; do not add a second folder list.
- `TestMirrorNeverLeaksStoredSecret` and `TestDayJournalExportLeakTest` — the
  existing canaries stay byte-for-byte untouched and must stay green.
- `docs/superpowers/specs/2026-06-25-sovereign-export-design.md` — a
  point-in-time design record, not a living doc; its deferral wording is
  historical and stays.

## Git workflow

- The executor runs in an isolated git worktree branched from `origin/main`;
  branch name `advisor/257-task-export-undefer`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/
  `chore`); e.g. `feat(export): un-defer task — leak-tested vault-mirror export`.
- Commit per logical unit with explicit pathspecs (the main checkout is shared
  by parallel agents — stage only your own files; never `git add -A`).
- NEVER push; the reviewer merges.

## Steps

### Step 0: Confirm the dependencies landed

This plan is written against the tree AFTER plans 242 and 252.

**Verify**: `grep -c "pruneStale" internal/export/export.go` → a number ≥ 2
(plan 242 landed). If `0`, STOP — plan 242 has not landed; report back.

**Verify**: `grep -c '\`day\`, \`task\`' .tours/15-sovereign-export.tour` → `0`
(plan 252 landed — the tour no longer lists day as deferred). If non-zero,
STOP — plan 252 has not landed; report back.

**Verify** (green baseline):
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1` → ok.

### Step 1: Redaction pre-verification (the actual work — record what you find)

Re-verify, against the LIVE tree, every claim in "What a task node carries"
above. This is the redaction pass: the code change in Step 2 is only safe if
these all hold. Record the outcome of each check in your plan-execution
report.

1. **Body is owner notes only.**
   **Verify**: `grep -rn 'Set("body"' internal/ --include='*.go' | grep -v _test`
   → exactly these hits (line numbers may drift):
   `internal/knowledge/knowledge.go` (memory nodes),
   `internal/tasks/tasks.go` (Update — owner notes),
   `internal/nodes/nodes.go` (generic Create/Update — called with owner text),
   `internal/web/knowledge.go` (owner's edit form),
   `internal/life/journal.go` (day nodes),
   `internal/nodes/links.go` (empty stub bodies).
   None of these writes model-generated text into a `type=task` node body.
   If a NEW hit appears that can write model output into a task node, STOP.

2. **Nudge writes only a timestamp into the node; its (possibly
   model-composed) text goes to `messages`.**
   **Verify**: `grep -n 'props\[' internal/tasks/nudge.go` → exactly one hit:
   `props["nudged_at"] = store.PBTime(now.UTC())`.
   **Verify**: `grep -n "AppendOrigin" internal/tasks/nudge.go` → one hit
   (the nudge message into the master conversation).

3. **Completion text goes to `entries`, not the node.**
   **Verify**: `grep -n 'addEntry' internal/tasks/tasks.go` → the
   `"completion"` call inside `Done` plus the `addEntry` definition; no call
   writes into the node body/title.

4. **The exporter reads only `nodes`.**
   **Verify**: `grep -n 'FindCollectionByNameOrId\|FindRecordsByFilter\|FindFirstRecordByFilter' internal/export/export.go`
   → no output (all reads go through `nodes.ListByTypeStatus` /
   `nodes.OwnerAuthoredTypes`).

5. **Props inventory matches the schema** (state, due, recur,
   recur_from_done, snoozed_until, nudged_at, done_at, source).
   **Verify**: `grep -n '"state"\|"due"\|"recur"\|"recur_from_done"\|"snoozed_until"\|"nudged_at"\|"done_at"\|"source"' migrations/1750000020_tasks_to_nodes.go`
   → all eight keys present, and
   `grep -rn 'props\["' internal/tasks/*.go | grep -v _test` shows no props
   key OUTSIDE that set being written.

Decision confirmed by the checks: all eight props are owner-authored values,
workflow tokens, provenance tags, or behavioral timestamps — export them all,
unfiltered (rationale in "Current state"). If ANY check instead reveals
model-generated or secret content landing in a task node's title/body/props,
this plan becomes a redesign — STOP and report; do not ship a leaky export.

### Step 2: Un-defer task in `internal/export/export.go`

Three edits, no logic changes:

1. In the `jdFolder` map, replace the three-line deferral comment
   (`// "task" has a JD folder in the design …` through `… (a future plan).`)
   with a real entry after the `"day"` line:

```go
	"task":   "60-69 Tasks/61 Tasks",
```

2. Empty `deferredTypes` (keep the mechanism and the rule comment — it is the
   gate for future types and `ExportMirror` still consults it):

```go
// deferredTypes are owner-authored types whose faithful export needs its own
// redaction pass. Exporting them raw could surface un-reviewed content, so they
// are skipped entirely until their redaction pass + leak test land.
// Do NOT remove a type from this set without adding its redaction pass + leak test.
// Currently empty: day graduated in plan 225 and task in plan 257, each behind
// a dedicated leak test in mirror_test.go.
var deferredTypes = map[string]bool{}
```

3. In the `ExportMirror` doc comment, replace the paragraph

```go
// task is DEFERRED (deferredTypes): its content needs its own redaction pass, so
// it is skipped here. day was un-deferred in plan 225 (leak test proved the body
// carries only the owner's journal text — recap/summaries never touch the body).
```

   with:

```go
// deferredTypes is currently empty but remains the gate for future types. day
// was un-deferred in plan 225 (leak test: the body carries only the owner's
// journal text — recap/summaries never touch the node). task was un-deferred in
// plan 257 (leak test: the body carries only the owner's task notes — nudge and
// completion text live in messages/entries, which the exporter never reads).
```

**Verify**: `gofmt -l . && go vet ./... && CGO_ENABLED=0 go build ./...` →
empty gofmt output, exit 0 overall.

**Verify** (the old guard bites — expected RED):
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -run TestMirrorSkipsDeferredTypes -count=1`
→ FAIL, with messages like `task node file exists on disk` /
`deferred task JD folder written`. If this test PASSES here, the deferral was
not actually gating exports — STOP and report.

**Verify** (everything else still green):
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -run 'TestMirrorNeverLeaksStoredSecret|TestDayJournalExportLeakTest|TestMirrorByteIdenticalReexport' -count=1`
→ ok.

### Step 3: Delete the inverted guard

In `internal/export/mirror_test.go`, delete the whole
`TestMirrorSkipsDeferredTypes` function INCLUDING its doc comment
(at `077318a` lines 152-206), and delete the `slugForTest` helper INCLUDING
its doc comment (at `077318a` lines 389-407) — the guard was its only caller
and an unused helper fails staticcheck U1000. Touch nothing else in the file.

**Verify**: `grep -c "TestMirrorSkipsDeferredTypes\|slugForTest" internal/export/mirror_test.go` → `0`

**Verify**: `go run honnef.co/go/tools/cmd/staticcheck@latest ./internal/export/` → no output, exit 0.

### Step 4: Add the task leak test

Append the following test at the END of `internal/export/mirror_test.go`
(after plan 242's prune tests). It mirrors `TestDayJournalExportLeakTest`
structurally. Add these imports to the file's import block:
`"github.com/alexradunet/balaur/internal/conversation"`,
`"github.com/alexradunet/balaur/internal/llm"`,
`"github.com/alexradunet/balaur/internal/tasks"` (the others — `os`,
`filepath`, `reflect`, `strings`, `testing`, `time`, `core`, `export`,
`nodes`, `store`, `storetest` — are already imported; drop any of the three
that gofmt/goimports flags as unused, though all three are used below).

```go
// TestTaskExportLeakTest is the redaction proof for task export (plan 257,
// the same recipe plan 225 used for day). It seeds a real task (owner title +
// notes) PLUS distinct marker strings in every adjacent surface a task
// touches — a nudge-style assistant message in the master conversation
// (`messages`) and a completion row in `entries` — then asserts:
//  1. The task file exists in its JD folder with the H1 title, the owner's
//     notes body, and the state frontmatter (task IS exported).
//  2. Neither marker appears in ANY exported file — the exporter never opens
//     messages or entries (the redaction boundary holds).
//  3. A second ExportMirror over unchanged data produces byte-identical files.
func TestTaskExportLeakTest(t *testing.T) {
	app := storetest.NewApp(t)

	const notesText = "TASK_NOTES_OWNER_WORDS"
	const nudgeMarker = "NUDGE_MARKER_ADJACENT_TEXT_DO_NOT_LEAK"
	const entryMarker = "COMPLETION_ENTRY_MARKER_DO_NOT_LEAK"

	// A real task through the owning package: title + owner notes, no due.
	rec, err := tasks.Create(app, tasks.CreateOpts{Title: "Water Plants", Notes: notesText})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Adjacent surface 1: a nudge-style assistant message in the master
	// conversation (where Nudge posts its possibly model-composed text).
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master conversation: %v", err)
	}
	if err := conversation.AppendOrigin(app, master.Id,
		llm.Message{Role: "assistant", Content: "Reminder: " + nudgeMarker}, "", "nudge"); err != nil {
		t.Fatalf("append nudge message: %v", err)
	}

	// Adjacent surface 2: a completion row in `entries` (where tasks.Done
	// logs recurring completions).
	entriesCol, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("find entries collection: %v", err)
	}
	entry := core.NewRecord(entriesCol)
	entry.Set("kind", "completion")
	entry.Set("task", rec.Id)
	entry.Set("text", entryMarker)
	entry.Set("noted_at", time.Now().UTC())
	if err := app.Save(entry); err != nil {
		t.Fatalf("save completion entry: %v", err)
	}

	dir := t.TempDir()
	paths, err := export.ExportMirror(app, dir)
	if err != nil {
		t.Fatalf("export mirror: %v", err)
	}

	// 1. The task file exists in its JD folder with title, notes, state.
	const want = "60-69 Tasks/61 Tasks/water-plants.md"
	found := false
	for _, p := range paths {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Errorf("task file %q not in returned paths %v", want, paths)
	}
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(want)))
	if err != nil {
		t.Fatalf("read task file: %v", err)
	}
	text := string(raw)
	for _, needle := range []string{"# Water Plants", notesText, `state: "open"`} {
		if !strings.Contains(text, needle) {
			t.Errorf("task file missing %q:\n%s", needle, text)
		}
	}

	// 2. Neither adjacent-surface marker may appear anywhere in the tree.
	for name, content := range readAll(t, dir) {
		if strings.HasPrefix(name, ".git") {
			continue
		}
		if strings.Contains(content, nudgeMarker) {
			t.Fatalf("NUDGE MARKER LEAKED into %s — messages collection read by exporter:\n%s", name, content)
		}
		if strings.Contains(content, entryMarker) {
			t.Fatalf("ENTRY MARKER LEAKED into %s — entries collection read by exporter:\n%s", name, content)
		}
	}

	// 3. Determinism: a second run over unchanged data is byte-identical.
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
```

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -run TestTaskExportLeakTest -count=1 -v`
→ `PASS: TestTaskExportLeakTest`.

**Verify** (whole package, uncached):
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1` → ok, ALL
tests pass, including `TestMirrorNeverLeaksStoredSecret` and
`TestDayJournalExportLeakTest` unmodified.

**Commit**:
`git add internal/export/export.go internal/export/mirror_test.go && git commit -m "feat(export): un-defer task — leak-tested vault-mirror export"`

### Step 5: Fix tour 15 (prose + shifted anchors)

`.tours/15-sovereign-export.tour` is JSON. After plans 242/252 it anchors
`internal/export/export.go` at lines 1, 59, `<ExportMirror>`, `<render>` and
`internal/export/mirror_test.go` at one line inside
`TestMirrorNeverLeaksStoredSecret`, and its prose says only `task` is
deferred. Make these edits:

1. **Step 15.1** (the `jdFolder` step, `"file": "internal/export/export.go"`,
   `"line": 59`): in its `"description"` snippet, add
   `\n    \"task\":   \"60-69 Tasks/61 Tasks\",` after the `"day"` entry so
   the snippet matches the live map. Replace the sentence(s) claiming `task`
   is absent / listed in `deferredTypes` (post-252 wording begins "Notice that
   `task` is absent — …") with:
   `Every owner-authored type in the JD design is now mapped. \`day\` was un-deferred in plan 225 and \`task\` in plan 257, each behind a dedicated leak test (recap, nudge, and completion text live in the separate \`summaries\`/\`messages\`/\`entries\` collections and never reach the node).`
   KEEP the bolded rule sentence "**Do not add a type here without
   simultaneously removing it from `deferredTypes` and landing a leak test.**"
   and the closing YAGNI/KISS sentence.

2. **Step 15.2** (the `ExportMirror` step): change its numbered item 3
   (post-252 wording: "Skips `deferredTypes` (now only `task` — …)") to:
   `Skips \`deferredTypes\` — currently empty (\`day\` graduated in plan 225, \`task\` in plan 257, each behind a leak test); the gate stays for future types. A deferred type is never opened, never read, never written.`

3. **Step 15.4** (the package-header step, `"line": 1`): change the
   "**Type deferral**" bullet (post-252 wording: "`task` is in `deferredTypes`
   and skipped before any read attempt — …") to:
   `**Type deferral**: \`deferredTypes\` is currently empty — \`day\` (plan 225) and \`task\` (plan 257) both export behind dedicated leak tests (\`TestDayJournalExportLeakTest\`, \`TestTaskExportLeakTest\` in \`internal/export/mirror_test.go\`). The gate remains for any future type whose content needs its own redaction pass.`
   Adjust any trailing sentence still saying task bodies need a future pass.

4. **Repoint the shifted anchors.** Run:
   `grep -n "^func ExportMirror\|^func render" internal/export/export.go` and
   `grep -n "^func TestMirrorNeverLeaksStoredSecret" internal/export/mirror_test.go`.
   Set the tour's `"line"` for the `ExportMirror` step and the `render` step
   in `export.go` to the grepped numbers, and the `mirror_test.go` step's
   `"line"` to the grepped canary-function line. Anchors at `export.go` lines
   1 and 59 and `encrypt.go` 57 are before/outside your edits — eyeball each
   against the live file and leave them.

**Verify**: `python3 -m json.tool < .tours/15-sovereign-export.tour > /dev/null` → exit 0
**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
**Verify**: `grep -c "TestTaskExportLeakTest" .tours/15-sovereign-export.tour` → `1`
**Verify**: `grep -c "task.* is absent\|only .task. is deferred\|now only .task." .tours/15-sovereign-export.tour` → `0`

### Step 6: Update the self-description and AGENTS.md

1. `internal/self/knowledge.md` — in the export paragraph, replace the clause

   ```
   `task` is deferred pending its own content redaction pass; 
   ```

   (find it with `grep -n "task.*deferred" internal/self/knowledge.md`) with:

   ```
   `task` exports its owner-authored title, notes body, and scheduling props (plan 257 — nudge and completion text live in the separate `messages`/`entries` collections, never the node, and are leak-tested to stay out of the mirror); 
   ```

   Leave the neighboring `day` clause (plan 225) and plan 242's prune clause
   untouched.

2. `AGENTS.md` — in the vault-mirror bullet (post-plan-252 text under "Known
   limitations & deferred work"), replace the two sentences

   ```
   Only the `task` type stays deferred pending its content
     redaction pass; `day` journal bodies export behind a leak test. Do not
     claim task export in user-facing copy until that redaction pass lands.
   ```

   with:

   ```
   Every owner-authored type exports, each behind a leak test
     (`day` plan 225, `task` plan 257).
   ```

   (Keep the bullet's surrounding sentences and wrapping style; re-wrap to
   ~80 columns to match the file.)

**Verify**: `grep -c "task.*deferred" internal/self/knowledge.md` → `0`
**Verify**: `grep -c "leak-tested to stay out of the mirror" internal/self/knowledge.md` → `2` (day's existing clause + the new task clause)
**Verify**: `grep -c "stays deferred" AGENTS.md` → `0`
**Verify**: `grep -c "task. plan 257" AGENTS.md` → `1`

**Commit**:
`git add .tours/15-sovereign-export.tour internal/self/knowledge.md AGENTS.md && git commit -m "docs(export): tour + self-knowledge + AGENTS — task exports behind leak test"`

### Step 7: Full gate

**Verify** (each in turn):

- `gofmt -l .` → empty output
- `go vet ./...` → exit 0
- `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
- `CGO_ENABLED=0 go build ./...` → exit 0
- `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all packages pass
- `git diff --check` → no output

Then update the plan-257 row in `plans/README.md` (add the row if absent) and
commit it with an explicit pathspec.

## Test plan

- New test, appended at the end of `internal/export/mirror_test.go`, modeled
  structurally on `TestDayJournalExportLeakTest` (same file):
  - `TestTaskExportLeakTest` — the redaction proof: task file exported with
    title/notes/state (happy path); nudge marker seeded in `messages` never
    appears in the tree; completion marker seeded in `entries` never appears
    in the tree; second export byte-identical (determinism).
- Deleted test: `TestMirrorSkipsDeferredTypes` (its assertion — "no task file
  anywhere" — is inverted by this plan; the leak test is its successor) plus
  its only helper `slugForTest`.
- Existing tests that must pass UNMODIFIED: `TestMirrorNeverLeaksStoredSecret`
  and `TestDayJournalExportLeakTest` (the canaries — byte-for-byte untouched),
  `TestMirrorLayoutPerType`, `TestMirrorByteIdenticalReexport`,
  `TestMirrorUnmappedTypeGoesUnsorted`, `TestMirrorGitCommit`, plan 242's
  prune tests, and all of `export_test.go` / `encrypt_test.go`.
- Verification:
  `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1 -v` → all
  pass, including the 1 new test; then the full-suite gate in Step 7.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output;
      `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `grep -c '"task":   "60-69 Tasks/61 Tasks"' internal/export/export.go` → `1`
- [ ] `grep -c '"task": true' internal/export/export.go` → `0`
- [ ] `grep -c "Do NOT remove a type from this set" internal/export/export.go` → `1` (the rule comment survives)
- [ ] `grep -c "func TestTaskExportLeakTest" internal/export/mirror_test.go` → `1`
- [ ] `grep -c "TestMirrorSkipsDeferredTypes\|slugForTest" internal/export/mirror_test.go` → `0`
- [ ] `grep -c "func TestMirrorNeverLeaksStoredSecret" internal/export/mirror_test.go` → `1` and `grep -c "func TestDayJournalExportLeakTest" internal/export/mirror_test.go` → `1` (canaries intact)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `grep -c "task.*deferred" internal/self/knowledge.md` → `0`; `grep -c "stays deferred" AGENTS.md` → `0`
- [ ] `git diff --stat <worktree-base>..HEAD` (where `<worktree-base>` is
      `git merge-base HEAD origin/main`) lists ONLY:
      `internal/export/export.go`, `internal/export/mirror_test.go`,
      `.tours/15-sovereign-export.tour`, `internal/self/knowledge.md`,
      `AGENTS.md`, `plans/README.md`, `plans/257-task-export-undefer.md`
      (status/plan bookkeeping) — no out-of-scope files modified
- [ ] `plans/README.md` status row for 257 updated

## STOP conditions

Stop and report back (do not improvise) if:

- Step 0 shows plan 242 or 252 has not landed (`pruneStale` absent from
  `export.go`, or the tour still says `` `day`, `task` `` are deferred) — this
  plan's tour/AGENTS edits are written against their landed text.
- Step 1's redaction pre-verification finds ANY code path writing
  model-generated text, secrets, or third-party content into a task node's
  title, body, or props (i.e. the export would need actual content redaction,
  not just an inclusion decision). That is a redesign, not this plan — report
  instead of shipping a leaky export.
- After Step 2, `TestMirrorSkipsDeferredTypes` still PASSES — the deferral was
  not gating what this plan assumes; something structural drifted.
- `TestMirrorNeverLeaksStoredSecret` or `TestDayJournalExportLeakTest` fails
  at any point, or making the new test pass appears to require MODIFYING
  either — the canaries are load-bearing and "must never be deleted or
  weakened".
- `TestTaskExportLeakTest` fails on a MARKER assertion (a marker string found
  in the tree) — that is a real redaction-boundary breach somewhere in the
  exporter; do not weaken the assertion.
- The `deferredTypes`/`jdFolder` code no longer matches the "Current state"
  excerpts beyond what plans 242/252 describe (e.g. the defer mechanism was
  removed or restructured).
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (e.g.
  `internal/tasks/*`, `internal/export/encrypt.go`, `internal/cli/export.go`).

## Maintenance notes

- **`deferredTypes` is now empty but intentionally kept**: it is the
  documented gate for any future owner-authored type whose content needs a
  redaction pass before export. Reviewers should reject a change that deletes
  the mechanism "because it's empty" — the rule comment is the contract.
- **Prune composes automatically**: plan 242's `pruneStale` derives its
  managed-folder set from `jdFolder` values at call time, so
  `60-69 Tasks/61 Tasks` became a managed (pruned) folder the moment the map
  entry landed. A completed one-off task stays `status=active` with
  `props.state="done"`, so it KEEPS exporting (state is workflow, not
  consent); only archive/drop of the NODE (consent axis) removes it from the
  mirror via prune.
- **What a reviewer should scrutinize**: that the canary tests are
  byte-for-byte unchanged (`git diff` on `mirror_test.go` shows only the
  deleted guard/helper and the appended leak test); that no props filtering
  crept into `render` (the no-filter decision is deliberate — see "Current
  state"); that the tour prose keeps the "do not add a type without a leak
  test" rule.
- **Explicitly deferred**: exporting task completion HISTORY (the `entries`
  rows) — entries are life-log data, not nodes, and exporting them is a
  separate design decision; any per-prop redaction/filtering UI; updating the
  historical design spec (`docs/superpowers/specs/2026-06-25-sovereign-export-design.md`).
- **Future interaction**: if a task-adjacent feature ever writes model text
  into task node props (e.g. an AI-suggested "next step" prop), the leak test
  will NOT catch it (it guards adjacent collections, not new prop writers) —
  such a feature must extend `TestTaskExportLeakTest` with a marker for the
  new writer, per the same recipe.
