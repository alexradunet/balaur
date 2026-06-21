# Plan 111: Per-record card SSE fragments render via the existing gomponents components (task, memory, skill) instead of `html/template`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 0dd2457..HEAD -- internal/web/tasks.go internal/web/knowledge.go internal/feature/taskcards internal/feature/knowledgecards`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (independent of 112–115; all of 111–115 must land before 116/117)
- **Category**: migration / tech-debt
- **Planned at**: commit `0dd2457`, 2026-06-19

## Why this matters

Balaur's UI is being unified on `gomponents` (the typed Go component system) as
the single way to build markup; the legacy `html/template` engine + the
`web/templates/*.html` files are being removed (see `AGENTS.md` "gomponents is
the one way to build UI"). The per-record **task** and **knowledge (memory /
skill)** cards are the easiest leg: their gomponents ports **already exist and
are already used elsewhere** — `taskcards.TaskCard` is explicitly "the
gomponents port of card-task.html", and `knowledgecards.MemoryRecordCard` /
`SkillRecordCard` are explicit "Port of card-{memory,skill}.html". Today the web
gateway still renders these three cards through `h.tmpl.ExecuteTemplate(...)`,
running a *second, duplicate* renderer in parallel with the components. This
plan repoints the three call sites at the existing components, deleting the
duplication and removing three `ExecuteTemplate` callers — a prerequisite for
deleting the templates entirely (plan 117).

This is a behavior-preserving swap: the components were written to match the
templates byte-for-byte in structure, and the SSE patches that consume them key
off the root element id (`tcard-{id}` / `kcard-{id}`), which both renderers
emit identically.

## Current state

Three live `ExecuteTemplate` call sites render per-record cards into strings for
SSE patches / HTTP fragments:

- `internal/web/tasks.go:204-210` — `taskCardHTML` renders `card-task.html`:
  ```go
  // taskCardHTML renders the card-task.html partial for one record to a string,
  // for embedding in an SSE patch.
  func (h *handlers) taskCardHTML(rec *core.Record) (string, error) {
  	var b strings.Builder
  	if err := h.tmpl.ExecuteTemplate(&b, "card-task.html", taskViewOf(rec, time.Now())); err != nil {
  		return "", err
  	}
  	return b.String(), nil
  }
  ```
  Called by `taskCard` (GET /ui/tasks/{id}/card), `taskTransition` (POST
  /ui/tasks/{id}/transition, `tasks.go:256`), and `proposalBody`
  (`cards.go:184`). `taskViewOf` (`tasks.go:34`) returns the local `web.taskView`
  struct: `{ID, Title, Notes, Status string; DueLine string; Overdue bool; RecurLine string}`.

- `internal/web/knowledge.go:139-147` — `renderCardHTML` renders the per-kind
  partial into a buffer (consumed by `knowledgeTransition`, `knowledgeEdit`, and
  `proposalBody`):
  ```go
  func (h *handlers) renderCardHTML(kind knowledge.Kind, rec *core.Record) (string, error) {
  	var b strings.Builder
  	if err := h.tmpl.ExecuteTemplate(&b, cardTemplateName(kind), rec); err != nil {
  		return "", err
  	}
  	return b.String(), nil
  }
  ```
  where `cardTemplateName` (`knowledge.go:132`) returns `"card-skill.html"` for
  `knowledge.Skill`, else `"card-memory.html"`.

- `internal/web/knowledge.go:163-173` — `renderCard` writes the same partial
  straight to the response (GET /ui/knowledge/{kind}/{id}/card):
  ```go
  func (h *handlers) renderCard(e *core.RequestEvent, kind knowledge.Kind, rec *core.Record) error {
  	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
  	name := "card-memory.html"
  	if kind == knowledge.Skill {
  		name = "card-skill.html"
  	}
  	if err := h.tmpl.ExecuteTemplate(e.Response, name, rec); err != nil {
  		return e.InternalServerError("rendering card", err)
  	}
  	return nil
  }
  ```

**The gomponents replacements already exist** (same package layering — the
feature packages own the view-models, never import `internal/web`):

- `internal/feature/taskcards/taskcard.go:25` —
  `func TaskCard(v TaskView) g.Node`, root id `tcard-{id}`. `TaskView`
  (`taskcard.go:13`) is `{ID, Title, Status, DueLine, RecurLine, Notes string; Overdue bool}`
  — field-for-field identical to `web.taskView`.
- `internal/feature/knowledgecards/memory.go:84` —
  `func MemoryRecordCard(r MemoryRecord) g.Node`, root id `kcard-{id}`.
  `MemoryRecord` (`memory.go:34`) = `{ID, Status, Category, Title, Content, WhenToUse string; Importance, UseCount int}`.
  A **private** slice mapper exists: `mapMemoryRecords([]*core.Record) []MemoryRecord` (`memory.go:272`).
- `internal/feature/knowledgecards/skills.go:150` —
  `func SkillRecordCard(r SkillRecord) g.Node`, root id `kcard-{id}`.
  `SkillRecord` (`skills.go:32`) = `{ID, Status, Name, Description, WhenToUse, Content string; Enabled bool; UseCount int}`.
  A **private** slice mapper exists: `mapRecords([]*core.Record) []SkillRecord` (`skills.go:81`).

**Render-to-string helper** (already in the package, use it — do not hand-roll a
`strings.Builder` each time): `internal/web/panel.go:94`
```go
func renderNodeHTML(n g.Node) string {
	var b strings.Builder
	_ = n.Render(&b)
	return b.String()
}
```

**Convention to match**: feature packages own their record→view-model mapping
(`mapMemoryRecords`, `mapRecords` already do the slice form). Add the
single-record exported mappers in those packages, not in `internal/web`. See the
existing slice mappers as the exact pattern to copy. The package doc forbids
importing `internal/web` (layering law) — your new mappers take `*core.Record`,
which is fine (those packages already import `pocketbase/core`).

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tests (web) | `go test ./internal/web/... ./internal/feature/...` | all pass |
| Full tests | `go test ./...` | all pass, exit 0 |
| Format check | `gofmt -l internal/` | empty output |
| Whitespace | `git diff --check` | no output |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Suggested executor toolkit

- Invoke the `ui-development` skill if available — it documents the gomponents
  conventions and the storybook-as-source-of-truth workflow.

## Scope

**In scope** (the only files you should modify):
- `internal/feature/knowledgecards/memory.go` (add exported single-record mapper)
- `internal/feature/knowledgecards/skills.go` (add exported single-record mapper)
- `internal/web/tasks.go`
- `internal/web/knowledge.go`
- Test files for the above if assertions need updating (see Test plan)

**Out of scope** (do NOT touch, even though they look related):
- `web/templates/card-task.html`, `card-memory.html`, `card-skill.html` — leave
  in place; plan 117 deletes the whole `web/templates/` directory once every
  `ExecuteTemplate` caller is gone. Deleting them here breaks the still-live
  `ucard_palette`/`recap-*`/`chat_bar`/`chat-choices` parse in `Register`.
- The `tmpl` field, `funcs` FuncMap, `template.HTML` type usages — later plans.
- `taskcards.TaskCard` / `MemoryRecordCard` / `SkillRecordCard` bodies — reuse,
  do not edit their markup.

## Git workflow

- Branch: `improve/111-cards-swap-to-gomponents` (or the repo convention).
- One commit; message style: conventional commits, e.g.
  `refactor(web): render task/knowledge cards via gomponents (plan 111)`.
  End the commit body with the repo's `Co-Authored-By` trailer if you add one.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add exported single-record mappers to the feature packages

In `internal/feature/knowledgecards/memory.go`, add (place it just above
`mapMemoryRecords`):
```go
// MemoryRecordOf maps one memory *core.Record to the MemoryRecordCard view-model.
func MemoryRecordOf(r *core.Record) MemoryRecord {
	return MemoryRecord{
		ID:         r.Id,
		Status:     r.GetString("status"),
		Category:   r.GetString("category"),
		Title:      r.GetString("title"),
		Content:    r.GetString("content"),
		WhenToUse:  r.GetString("when_to_use"),
		Importance: r.GetInt("importance"),
		UseCount:   r.GetInt("use_count"),
	}
}
```
Then simplify `mapMemoryRecords` to call it (keeps one source of truth):
```go
func mapMemoryRecords(recs []*core.Record) []MemoryRecord {
	out := make([]MemoryRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, MemoryRecordOf(r))
	}
	return out
}
```

In `internal/feature/knowledgecards/skills.go`, add (just above `mapRecords`):
```go
// SkillRecordOf maps one skill *core.Record to the SkillRecordCard view-model.
func SkillRecordOf(r *core.Record) SkillRecord {
	return SkillRecord{
		ID:          r.Id,
		Status:      r.GetString("status"),
		Name:        r.GetString("name"),
		Description: r.GetString("description"),
		WhenToUse:   r.GetString("when_to_use"),
		Content:     r.GetString("content"),
		Enabled:     r.GetBool("enabled"),
		UseCount:    r.GetInt("use_count"),
	}
}
```
Then simplify `mapRecords` to call it.

**Verify**: `go build ./internal/feature/knowledgecards/` → exit 0.

### Step 2: Repoint the knowledge card renderers at the gomponents components

In `internal/web/knowledge.go`:
- Replace the body of `renderCardHTML` so it renders the component to a string:
  ```go
  func (h *handlers) renderCardHTML(kind knowledge.Kind, rec *core.Record) (string, error) {
  	return renderNodeHTML(knowledgeRecordNode(kind, rec)), nil
  }
  ```
- Replace the body of `renderCard` so it writes the component to the response:
  ```go
  func (h *handlers) renderCard(e *core.RequestEvent, kind knowledge.Kind, rec *core.Record) error {
  	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
  	return knowledgeRecordNode(kind, rec).Render(e.Response)
  }
  ```
- Add the shared helper and delete the now-unused `cardTemplateName` (it has no
  other callers — confirm with `grep -rn cardTemplateName internal/`):
  ```go
  // knowledgeRecordNode renders one knowledge record as its gomponents card
  // (the port of card-memory.html / card-skill.html).
  func knowledgeRecordNode(kind knowledge.Kind, rec *core.Record) g.Node {
  	if kind == knowledge.Skill {
  		return knowledgecards.SkillRecordCard(knowledgecards.SkillRecordOf(rec))
  	}
  	return knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecordOf(rec))
  }
  ```
- `knowledge.go` already imports `g "maragu.dev/gomponents"` and
  `knowledgecards`. It does **not** import `html/template`, so no import change
  is needed here.

**Verify**: `go build ./internal/web/` → exit 0 (a build error about an unused
`cardTemplateName` or a missing import is the signal you missed a deletion).

### Step 3: Repoint the task card renderer at `taskcards.TaskCard`

In `internal/web/tasks.go`, replace `taskCardHTML`:
```go
// taskCardHTML renders one task as its gomponents card (port of card-task.html)
// to a string, for embedding in an SSE patch.
func (h *handlers) taskCardHTML(rec *core.Record) (string, error) {
	return renderNodeHTML(taskcards.TaskCard(taskCardViewOf(rec))), nil
}

// taskCardViewOf maps the web taskView onto the taskcards.TaskView the component takes.
func taskCardViewOf(rec *core.Record) taskcards.TaskView {
	v := taskViewOf(rec, time.Now())
	return taskcards.TaskView{
		ID: v.ID, Title: v.Title, Status: v.Status,
		DueLine: v.DueLine, RecurLine: v.RecurLine, Notes: v.Notes, Overdue: v.Overdue,
	}
}
```
Add the import `"github.com/alexradunet/balaur/internal/feature/taskcards"` to
`tasks.go`'s import block. (`tasks.go` does not currently import `gomponents`;
`renderNodeHTML` returns a plain string, so no `gomponents` import is needed in
`tasks.go`.)

**Verify**: `go build ./internal/web/` → exit 0.

### Step 4: Build, vet, and run the suite; fix any drifted markup assertions

Run the full checks. Some web tests may assert on the old template's exact bytes.
The components are structural ports, but if a test breaks on a *cosmetic*
difference (e.g. a `title="importance N/5"` attribute the template emitted on the
pips that `ui.Pips` does not), update the assertion to match the component's
output — the load-bearing invariant is the **root element id** (`tcard-{id}` /
`kcard-{id}`) and the action `@post` URLs, which are unchanged. If a test breaks
on something that changes *behavior* (a missing form, a changed POST URL, a
different status branch), STOP — that means the component is not actually at
parity and this swap is unsafe.

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass, exit 0
- `gofmt -l internal/` → empty

## Test plan

- No new component tests are required: `taskcards` and `knowledgecards` already
  carry their own render tests (`taskcard_test.go`, `memory_test.go`,
  `skills_test.go`) which enforce the markup/id invariants.
- Run the existing web handler tests (`internal/web/*_test.go`) and the feature
  tests. Update only assertions that drifted on cosmetic markup, per Step 4.
- If `internal/web/templates_test.go` has a `TestCardTask`-style test that
  executes `card-task.html` directly (it does — it exercises the template), do
  **not** delete it here; it still passes because the template file still exists
  (plan 117 removes that test with the templates). Leave it untouched.
- Verification: `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l internal/` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `grep -rn 'ExecuteTemplate' internal/web/tasks.go internal/web/knowledge.go` returns **no** matches
- [ ] `grep -rn 'card-task.html\|card-memory.html\|card-skill.html' internal/web/` returns **no** matches (the names live only in `web/templates/` now)
- [ ] `web/templates/card-task.html`, `card-memory.html`, `card-skill.html` still exist on disk (untouched)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The "Current state" excerpts don't match the live code (drift since `0dd2457`).
- A web test fails on a *behavioral* difference (missing form, changed `@post`
  URL, wrong status branch) — the component is not at parity; do not paper over it.
- The `taskcards.TaskView` / `knowledgecards.MemoryRecord` / `SkillRecord`
  structs no longer have the fields listed above.
- Removing `cardTemplateName` reveals another caller you didn't expect.

## Maintenance notes

- After this lands, the **only** thing keeping `card-task.html` / `card-memory.html`
  / `card-skill.html` alive is `knowledge-grid.html`'s `{{template "card-memory.html"}}`
  reference and the template parse in `Register` — both removed in plan 117.
- Future task/knowledge card markup changes now happen **once**, in
  `taskcards.TaskCard` / `knowledgecards.*RecordCard`, with their storybook
  stories. There is no longer a template to keep in sync.
- Reviewer: confirm the root ids (`tcard-{id}`, `kcard-{id}`) are unchanged —
  the SSE `WithSelectorID(...)+WithModeOuter` patches in `tasks.go` /
  `knowledge.go` depend on them.
