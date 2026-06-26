# Plan 199: Collapse two verbatim duplications in `internal/tools` and `internal/web` (card-registry vocabulary + task view-model), and delete dead `web.questGroup`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/tools/ui.go internal/tools/artifact.go internal/feature/taskcards/quests.go internal/feature/taskcards/questsfocus.go internal/web/tasks.go internal/web/tasks_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

Two small verbatim duplications make the same fact live in two places, so a
change has to be made twice or it silently drifts:

1. **Card-registry vocabulary builder.** `cardShowTool` (`internal/tools/ui.go`)
   and `showCardsTool` (`internal/tools/artifact.go`) each contain a
   byte-for-byte ~20-line loop over `cards.All()` that formats each card type as
   `"type (label) — params: name (required): doc; …"` for the tool description
   the model reads. Only the lead-in prose differs. If the registry's param
   shape changes, both loops must change together.

2. **Task view-model.** `internal/web/tasks.go` re-derives the same `TaskView`
   the card layer already owns: its `taskView` struct + `taskViewOf` +
   `taskCardViewOf` rebuild what `internal/feature/taskcards`'s (unexported)
   `taskViewOf` already builds, and `web.questGroup` is a byte-for-byte copy of
   `taskcards.questGroupName`. `web/tasks.go:64` even comments that the two
   paths "must behave identically." `web.questGroup` is **dead** — its only
   caller is its own test (`TestQuestGroup`).

This plan removes both duplications with pure extraction/relocation — no new
abstractions. The card package stays the single source of truth for the task
view-model; the layering (`web → taskcards`, never the reverse) is preserved.

## Current state

### Duplication 1 — card-registry vocabulary

`internal/tools/ui.go` lines 65–90 (`cardShowTool`):

```go
func cardShowTool(_ core.App) agent.Tool {
	// Build a rich description that embeds the real registry vocabulary,
	// so the model sees the actual types and their param docs.
	var b strings.Builder
	fmt.Fprint(&b, "Render a live UI card into the conversation. Choose a type from the registry; "+
		"the server renders it from the owner's real data. Available types:\n")
	for _, spec := range cards.All() {
		fmt.Fprintf(&b, "  %s (%s)", spec.Type, spec.Label)
		if len(spec.Params) > 0 {
			fmt.Fprint(&b, " — params: ")
			ps := make([]string, 0, len(spec.Params))
			for _, p := range spec.Params {
				entry := p.Name
				if p.Required {
					entry += " (required)"
				}
				if p.Doc != "" {
					entry += ": " + p.Doc
				}
				ps = append(ps, entry)
			}
			fmt.Fprint(&b, strings.Join(ps, "; "))
		}
		fmt.Fprint(&b, "\n")
	}
	desc := b.String()
	...
```

`internal/tools/artifact.go` lines 61–89 (`showCardsTool`) — the loop body
(lines 70–88) is **identical**; only the lead-in `fmt.Fprint` prose differs:

```go
func showCardsTool(_ core.App) agent.Tool {
	var b strings.Builder
	fmt.Fprint(&b, "Render a cluster of live UI cards into the conversation as ONE artifact "+
		"(e.g. 'show my quests and my weight together'). Pick 1–8 cards; each is a "+
		"{type, params} from the registry; the server renders each from the owner's real "+
		"data. To draw the owner's individual quests as separate cards, use the \"tasks\" "+
		"card (a bare stack of task cards) with a status/bucket/terms filter. Available types:\n")
	for _, spec := range cards.All() {
		fmt.Fprintf(&b, "  %s (%s)", spec.Type, spec.Label)
		// ... IDENTICAL to ui.go lines 73-88 ...
	}
	desc := b.String()
	...
```

### Duplication 2 — task view-model in `internal/web/tasks.go`

```go
// lines 36-59
type taskView struct {
	ID, Title, Notes, Status string
	DueLine                  string
	Overdue                  bool
	RecurLine                string
}

func taskViewOf(rec *core.Record, now time.Time) taskView { ... }

// lines 61-79  — DEAD: only caller is TestQuestGroup
func questGroup(recur string, hasDue bool) string { ... }

// lines 99-119
func (h *handlers) taskCardHTML(rec *core.Record) (string, error) {
	return renderNodeHTML(taskcards.TaskCard(taskCardViewOf(rec))), nil
}

func taskCardViewOf(rec *core.Record) taskcards.TaskView {
	now := time.Now()
	v := taskViewOf(rec, now)
	tv := taskcards.TaskView{
		ID: v.ID, Title: v.Title, Status: v.Status,
		DueLine: v.DueLine, RecurLine: v.RecurLine, Notes: v.Notes, Overdue: v.Overdue,
		Recur: rec.GetString("recur"),
	}
	if due := rec.GetDateTime("due").Time(); !due.IsZero() {
		tv.DueInput = due.In(now.Location()).Format("2006-01-02T15:04")
	}
	return tv
}
```

The canonical superset mapper already exists in
`internal/feature/taskcards/quests.go` lines 87–108 (currently **unexported**):

```go
// taskViewOf builds the full task view-model (mirrors web/tasks.go taskCardViewOf,
// including Recur/DueInput so the inline Edit form pre-fills on the quests/cluster
// render paths, not only the standalone card route).
func taskViewOf(rec *core.Record, now time.Time) TaskView {
	v := TaskView{
		ID:     rec.Id,
		Title:  rec.GetString("title"),
		Notes:  rec.GetString("notes"),
		Status: rec.GetString("status"),
		Recur:  rec.GetString("recur"),
	}
	if d := rec.GetDateTime("due").Time(); !d.IsZero() {
		v.Overdue = d.In(now.Location()).Before(now) && v.Status == "open"
		v.DueLine = tasks.DueLine(d, now, v.Status)
		v.DueInput = d.In(now.Location()).Format("2006-01-02T15:04")
	}
	if rule, err := tasks.Parse(rec.GetString("recur")); err == nil && !rule.IsZero() {
		v.RecurLine = tasks.Describe(rule)
	}
	return v
}
```

This `taskcards.taskViewOf` is functionally a superset of web's
`taskCardViewOf` (it sets the same fields plus `RecurLine` from the rule). It
is called inside `taskcards` at three sites: `quests.go:113` (`viewsOf`),
`questsfocus.go:68`, and `questsfocus.go:83`.

`web.questGroup` (lines 66–79) is byte-for-byte
`taskcards.questGroupName` (`internal/feature/taskcards/questsfocus.go:94`),
and `web.questGroup`'s **only** caller is `internal/web/tasks_test.go:36`
inside `TestQuestGroup` (lines 15–40).

`internal/web/tasks.go` already imports `taskcards`
(`"github.com/alexradunet/balaur/internal/feature/taskcards"`, line 13).

### Repo conventions

- Standard Go; `gofmt` is law. Exported identifiers get a doc comment.
- No assertion frameworks; table-driven tests with the standard `testing`
  package. `make test` runs `go test ./...`.

## Commands you will need

| Purpose   | Command                                   | Expected on success     |
|-----------|-------------------------------------------|-------------------------|
| Build     | `CGO_ENABLED=0 go build ./...`            | exit 0                  |
| Vet       | `go vet ./...`                            | exit 0                  |
| Test pkg  | `go test ./internal/tools/... ./internal/web/... ./internal/feature/taskcards/...` | ok / PASS |
| Full test | `go test ./...`                           | all pass                |
| gofmt     | `gofmt -l internal/tools internal/web internal/feature/taskcards` | prints nothing |

> Note: the full `go test ./...` link step can fail with "No space left on
> device" on a small tmpfs `/tmp`. If that happens, set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry — it is an environment limit, not
> a test failure.

## Scope

**In scope** (the only files you should modify):
- `internal/tools/ui.go`
- `internal/tools/artifact.go`
- `internal/feature/taskcards/quests.go` (rename `taskViewOf` → `TaskViewOf`, update internal callers)
- `internal/feature/taskcards/questsfocus.go` (update internal callers of the renamed func)
- `internal/web/tasks.go` (delete `taskView`/`taskViewOf`/`taskCardViewOf`/`questGroup`; route through `taskcards.TaskViewOf`)
- `internal/web/tasks_test.go` (delete `TestQuestGroup`)

**Out of scope** (do NOT touch):
- `internal/cards/*` — pushing the vocabulary builder down into `cards` would
  couple the domain registry to agent-loop prose. Keep the helper in `tools`.
- `internal/feature/taskcards/today.go` — its `taskViewOf` mention is a comment
  only; it has its own field-limited mapping. Leave it.
- The `taskcards.TaskView` struct definition and `TaskCard` renderer — unchanged.

## Git workflow

- Branch: `advisor/199-dedup-card-vocab-and-task-view`
- Conventional-commit subject, e.g. `refactor(tools,web): collapse card-vocab + task-view duplication`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Extract `cardRegistryVocab()` in `internal/tools`

Add one unexported helper in `internal/tools` (put it in `ui.go`, just above
`cardShowTool`). It builds ONLY the per-type vocabulary block (no lead-in
prose), so each tool keeps its own intro:

```go
// cardRegistryVocab renders the live card registry as a model-facing vocabulary
// block: one line per card type with its label and param docs. cardShowTool and
// showCardsTool each prepend their own lead-in prose and append this.
func cardRegistryVocab() string {
	var b strings.Builder
	for _, spec := range cards.All() {
		fmt.Fprintf(&b, "  %s (%s)", spec.Type, spec.Label)
		if len(spec.Params) > 0 {
			fmt.Fprint(&b, " — params: ")
			ps := make([]string, 0, len(spec.Params))
			for _, p := range spec.Params {
				entry := p.Name
				if p.Required {
					entry += " (required)"
				}
				if p.Doc != "" {
					entry += ": " + p.Doc
				}
				ps = append(ps, entry)
			}
			fmt.Fprint(&b, strings.Join(ps, "; "))
		}
		fmt.Fprint(&b, "\n")
	}
	return b.String()
}
```

Then in `cardShowTool` replace lines 68–90 (the `var b strings.Builder` through
`desc := b.String()`) with:

```go
	desc := "Render a live UI card into the conversation. Choose a type from the registry; " +
		"the server renders it from the owner's real data. Available types:\n" +
		cardRegistryVocab()
```

And in `showCardsTool` (`artifact.go`) replace lines 64–89 with:

```go
	desc := "Render a cluster of live UI cards into the conversation as ONE artifact " +
		"(e.g. 'show my quests and my weight together'). Pick 1–8 cards; each is a " +
		"{type, params} from the registry; the server renders each from the owner's real " +
		"data. To draw the owner's individual quests as separate cards, use the \"tasks\" " +
		"card (a bare stack of task cards) with a status/bucket/terms filter. Available types:\n" +
		cardRegistryVocab()
```

After this, `artifact.go` no longer uses `cards.All()` directly but still uses
`cards` (e.g. `cards.Card`, `cards.ValidateCards`) and still uses `fmt`/`strings`
elsewhere — do NOT remove those imports. `ui.go` likewise keeps `fmt`/`strings`/`cards`.

**Verify**:
- `gofmt -l internal/tools` → prints nothing
- `go build ./internal/tools/...` → exit 0
- `go vet ./internal/tools/...` → exit 0
- The model-facing description is byte-identical to before. Confirm with a
  quick test or by eye: the only textual change is consolidation, not content.

### Step 2: Export `taskcards.TaskViewOf`

In `internal/feature/taskcards/quests.go`, rename the unexported `taskViewOf`
(line 90) to exported `TaskViewOf` and update its doc comment:

```go
// TaskViewOf builds the full task view-model from a hydrated task node:
// title/notes/status, the due line + overdue flag, the recurrence summary, and
// the raw Recur DSL + datetime-local DueInput the inline Edit form pre-fills.
// It is the single source of truth for a task's card view-model (web's
// single-card route calls it too).
func TaskViewOf(rec *core.Record, now time.Time) TaskView {
```

Update the two internal call sites:
- `quests.go:113` (inside `viewsOf`): `taskViewOf(r, now)` → `TaskViewOf(r, now)`
- `questsfocus.go:68`: `taskViewOf(rec, now)` → `TaskViewOf(rec, now)`
- `questsfocus.go:83`: `taskViewOf(r, now)` → `TaskViewOf(r, now)`

**Verify**:
- `grep -rn "\btaskViewOf\b" internal/feature/taskcards/` → only the `today.go`
  comment reference remains (no function definition or call).
- `go build ./internal/feature/taskcards/...` → exit 0

### Step 3: Route `web` through `taskcards.TaskViewOf`, delete web's copies

In `internal/web/tasks.go`:

1. Change `taskCardHTML` (line 102) to call the exported mapper directly:
   ```go
   func (h *handlers) taskCardHTML(rec *core.Record) (string, error) {
       return renderNodeHTML(taskcards.TaskCard(taskcards.TaskViewOf(rec, time.Now()))), nil
   }
   ```
2. Delete the now-unused `taskView` struct (lines 37–42), `taskViewOf`
   (lines 44–59), `taskCardViewOf` (lines 105–119), and `questGroup`
   (lines 61–79, with its doc comment).
3. Confirm `time` is still imported (yes — used by `taskCardHTML`,
   `taskTransition`, `snoozeUntil`, etc.). Do NOT remove it.

In `internal/web/tasks_test.go`: delete `TestQuestGroup` (lines 15–40) and any
import that becomes unused as a result (run `gofmt`/`go vet` to confirm).

**Verify**:
- `grep -rn "questGroup\|taskCardViewOf\|func taskViewOf\|type taskView" internal/web/` → no matches
- `go build ./internal/web/...` → exit 0
- `go vet ./internal/web/...` → exit 0

### Step 4: Full verification

**Verify**:
- `gofmt -l internal/tools internal/web internal/feature/taskcards` → prints nothing
- `go vet ./...` → exit 0
- `go test ./internal/tools/... ./internal/web/... ./internal/feature/taskcards/...` → PASS
- `go test ./...` → all pass
- `git diff --check` → no whitespace errors

## Test plan

- No new tests required — this is a behavior-preserving dedup. The existing
  `taskcards` tests already cover `TaskViewOf` (via `viewsOf`/quest rendering);
  the existing `web` task-card tests cover `taskCardHTML`.
- `TestQuestGroup` is deleted because its subject (`web.questGroup`) is deleted;
  the identical logic stays covered by any existing `taskcards.questGroupName`
  test. Check `internal/feature/taskcards/*_test.go` for a `questGroupName`
  test; if none exists, ADD a small table-driven `TestQuestGroupName` in
  `internal/feature/taskcards/questsfocus_test.go` (create if absent) mirroring
  the deleted `TestQuestGroup` cases, calling `questGroupName` — so the rhythm
  bucketing stays tested somewhere.
- Verification: `go test ./internal/feature/taskcards/...` → PASS (including the
  moved/retained group test).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `gofmt -l internal/tools internal/web internal/feature/taskcards` prints nothing
- [ ] `grep -rn "func cardRegistryVocab" internal/tools/` returns exactly one match
- [ ] `grep -rzoP "for _, spec := range cards.All\(\)" internal/tools/ui.go internal/tools/artifact.go` returns at most ONE match total (the loop now lives only in the helper)
- [ ] `grep -rn "questGroup\b\|taskCardViewOf\|func taskViewOf\|type taskView " internal/web/` returns no matches
- [ ] `go test ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The `cards.All()` loop bodies in `ui.go` and `artifact.go` are NOT identical
  when you compare them (the excerpts above don't match live code) — the dedup
  assumption is false.
- `web.questGroup` turns out to have a caller other than `TestQuestGroup`
  (`grep -rn "questGroup" internal/web/` shows a non-test caller) — deleting it
  would break that caller.
- `taskcards.TaskViewOf` and web's `taskCardViewOf` produce different
  `taskcards.TaskView` values for the same record (e.g. a web task card test
  starts failing) — they are not equivalent and the swap changed behavior.

## Maintenance notes

- After this lands, the card registry's model-facing vocabulary has ONE builder
  (`tools.cardRegistryVocab`). A new card param field (e.g. an enum constraint)
  is rendered once.
- `taskcards.TaskViewOf` is now the public task view-model mapper. Any new task
  card surface should call it rather than rebuild the struct.
- Reviewer: confirm the two tool descriptions still read naturally (the lead-in
  prose + vocabulary concatenation) and that no `web` task card lost the
  `DueInput`/`Recur` prefill (the Edit form depends on them).
