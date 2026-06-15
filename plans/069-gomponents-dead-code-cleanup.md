# Plan 069: delete the heads-card duplicate renderer and two migration fossils — one renderer per card, no dead struct/field

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 1f8f55e..HEAD -- internal/web/cards.go internal/web/heads.go web/templates/cards.html internal/web/templates_test.go internal/ollama/manager.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

The gomponents migration (cards moved from `html/template` to feature-owned
gomponents renderers under `internal/feature/*`) left three fossils behind. They
are not bugs today, but they are *live liabilities*: each one is a second copy of
something that already exists once, so the next person who edits the real thing
can edit the fossil instead and ship a silent divergence.

1. **The heads card has two renderers.** The live one is the gomponents
   `headscards.HeadsCard` (registered as card type `"heads"`, served through the
   registry). The fossil is `renderCardHeads` in `internal/web/cards.go` driving
   the legacy `ucard_heads` html/template. The legacy renderer survives only
   because two SSE re-patch sites in `heads.go` still call it directly. Pointing
   those two sites at the registry collapses it to one renderer.
2. **A calendar struct + template are retained only to feed one smoke test.** The
   real calendar tile is gomponents (`taskcards.CalendarCard`, with its own
   tests). `calendarCardView` + the `ucard_calendar` define + one test case in
   `templates_test.go` exist purely to test each other.
3. **`ollama.Model.Path` is a dead field.** It is always empty, no consumer
   reads it, and its doc comment says it is "kept so existing templates bind
   unchanged" — but the only template over `[]ollama.Model` reads `.Name` and
   `.Size`, never `.Path`.

After this lands: one renderer per card, no struct/template/test that exists only
to prove the other exists, and no dead field in the Ollama model type.

## Current state

### (A) Heads card — duplicated renderer

`internal/web/cards.go` (lines 159–210, as of `1f8f55e`) holds the legacy
renderer and its three view structs:

```go
// ---- heads tile (still legacy) ----
// ... (comment block) ...
type headGroupChoice struct {
	Key string
	On  bool
}

type headManageRow struct {
	ID, Name, Purpose, AvatarURL string
	BuiltIn, Active              bool
	Groups                       []headGroupChoice
}

type cardHeadsView struct {
	Heads   []headManageRow
	Avatars []store.AvatarEntry // new-head avatar picker
	Groups  []string            // group checkboxes for the new-head form
}

func (h *handlers) renderCardHeads(w io.Writer, _ map[string]string) error {
	activeID := heads.Active(h.app).ID
	var rows []headManageRow
	for _, hd := range heads.List(h.app) {
		// ... builds rows ...
	}
	return h.tmpl.ExecuteTemplate(w, "ucard_heads", cardHeadsView{
		Heads:   rows,
		Avatars: store.BalaurHeads(),
		Groups:  heads.Groups,
	})
}
```

The **only callers** of `renderCardHeads` are two SSE re-patch sites in
`internal/web/heads.go`:

`setActiveHead` (lines 34–39):

```go
	// Also refresh the manage card's active badges if it is on the page; the
	// patch is a no-op when #ucard-heads is absent.
	var card strings.Builder
	if err := h.renderCardHeads(&card, nil); err == nil {
		_ = sse.PatchElements(card.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	}
	return nil
```

`renderHeadsCard` (lines 43–52):

```go
// renderHeadsCard re-renders the heads manage card (#ucard-heads) via SSE.
func (h *handlers) renderHeadsCard(e *core.RequestEvent) error {
	var b strings.Builder
	if err := h.renderCardHeads(&b, nil); err != nil {
		return e.InternalServerError("rendering heads card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	return nil
}
```

(`renderHeadsCard` is in turn called by `createHead` and `deleteHead` in the same
file — leave those untouched; they call `renderHeadsCard`, which is the layer you
are rewiring.)

The **registry renderer** is `internal/feature/headscards/heads.go`:
`HeadsCard(buildHeads(app))`, registered as card type `"heads"` in that package's
`register.go` (`ui.RegisterCard("heads", ...)`). It is served via
`cardInto` → `ui.LookupCard` in `cards.go`:

```go
func (h *handlers) cardInto(w io.Writer, typ string, params map[string]string) error {
	if fn, ok := ui.LookupCard(typ); ok {
		node, err := fn(ui.Tile, params)
		if err != nil {
			return err
		}
		return node.Render(w)
	}
	return fmt.Errorf("unhandled card type %q", typ)
}
```

**Superset check (already done — record it, don't re-derive):** `HeadsCard`
renders `<article id="ucard-heads">` with: the active/built-in tags
(`headRowName`), the **Make active** form (`makeActiveForm`), the **Delete** form
(`deleteForm`, on non-built-in heads), and the full `+ New head` form with tools
checkboxes and avatar radios (`newHeadForm`). Its `register.go` comment states it
"Matches the legacy ucard_heads template exactly: same classes, ids, conditional
tags/forms, and new-head form." It is a faithful superset of `ucard_heads`. The
registered func signature is `func(_ ui.CardSize, _ map[string]string) (g.Node,
error)` and it ignores both args, building from live data — so calling it with
`nil` params is correct.

The `ucard_heads` define lives at `web/templates/cards.html` lines 41–98.

### (B) Calendar — struct + template + test that exist only for each other

`internal/web/cards.go` (lines 152–157):

```go
// calendarCardView feeds the legacy ucard_calendar template. The calendar tile
// itself is now gomponents (internal/feature/taskcards); this struct + template
// are retained only for the templates_test smoke test.
type calendarCardView struct {
	Cal calView
}
```

Its **only** user is `internal/web/templates_test.go` (lines 238–251):

```go
func TestCalendarCellsLinkToDayPages(t *testing.T) {
	tmpl := parseTemplates(t)
	var b strings.Builder
	// The calendar surface is now the calendar card (ucard_calendar); its cells
	// deep-link into the day card's focus (/focus/day?date=…), which replaced the
	// retired /day page.
	cal := buildCalendar(nil, "2026-06", time.Date(2026, 6, 11, 12, 0, 0, 0, time.Local))
	if err := tmpl.ExecuteTemplate(&b, "ucard_calendar", calendarCardView{Cal: cal}); err != nil {
		t.Fatalf("ucard_calendar: %v", err)
	}
	if !strings.Contains(b.String(), `href="/focus/day?date=2026-06-11"`) {
		t.Error("calendar cells do not link to the day focus")
	}
}
```

The `ucard_calendar` define lives at `web/templates/cards.html` lines 11–36. The
real calendar tile is `internal/feature/taskcards/calendar.go` (`CalendarCard`),
covered by `internal/feature/taskcards/calendar_test.go` (which builds its own
`taskcards.CalView` — independent of `internal/web`).

### (C) Dead `ollama.Model.Path`

`internal/ollama/manager.go` (lines 14–20):

```go
// Model is one model present in Ollama's local store. Path is always empty
// (Ollama owns the blob store); kept so existing templates bind unchanged.
type Model struct {
	Name string
	Size int64
	Path string
}
```

`fetchModels` (line 68) sets only `Name`/`Size`: `Model{Name: lm.Name, Size:
lm.Size}` — never `Path`. The only template over `[]ollama.Model` is
`web/templates/models.html` (the `{{range .GgufFiles}}` loop, lines 139–145),
which reads only `{{.Name}}` and `{{fmtBytes .Size}}`. A repo-wide grep confirms
nothing reads `Model.Path` (see Step verifications).

### Conventions that apply here

- gofmt is law: `gofmt -l .` must print nothing after your edits.
- `go vet ./...` must be clean; `CGO_ENABLED=0 go build ./...` must pass.
- Errors wrap with `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code (you are not adding error paths here, but match the file you edit).
- Do not "improve" adjacent code. Every changed line must trace to a deletion
  above.

## Commands you will need

| Purpose            | Command                                              | Expected on success      |
|--------------------|------------------------------------------------------|--------------------------|
| Drift check        | `git diff --stat 1f8f55e..HEAD -- internal/web/cards.go internal/web/heads.go web/templates/cards.html internal/web/templates_test.go internal/ollama/manager.go` | empty |
| Vet                | `go vet ./...`                                        | exit 0, no errors        |
| Web tests          | `go test ./internal/web`                             | all pass                 |
| Ollama tests       | `go test ./internal/ollama`                          | all pass                 |
| Headscards tests   | `go test ./internal/feature/headscards`             | all pass                 |
| All tests          | `go test ./...`                                       | all pass                 |
| Build (CGO off)    | `CGO_ENABLED=0 go build ./...`                        | exit 0                   |
| Format check       | `gofmt -l .`                                           | prints nothing           |
| Whitespace check   | `git diff --check`                                    | no output                |

## Scope

**In scope** (the only files you may modify):
- `internal/web/heads.go` — rewire the two re-patch sites onto the registry
- `internal/web/cards.go` — delete `renderCardHeads` + three view structs +
  `calendarCardView`; drop the now-unused `heads` and `store` imports
- `web/templates/cards.html` — delete the `ucard_heads` and `ucard_calendar`
  defines (and their leading comment banners)
- `internal/web/templates_test.go` — delete `TestCalendarCellsLinkToDayPages`
- `internal/ollama/manager.go` — delete the `Path` field; trim the doc comment

**Out of scope** (do NOT touch):
- `internal/feature/headscards/*` and `internal/feature/taskcards/*` — the live
  renderers. They are the source of truth; the whole point is to stop having a
  second copy.
- `internal/web/tasks.go` — defines `calView` and `buildCalendar`. After you
  delete the calendar test, `buildCalendar` (and via it `calView`) lose their
  last caller inside `internal/web`. **This is fine**: in Go an unused
  *package-level* function/type does not break the build (only unused *imports*
  and *locals* do). Leaving them is the smaller change; deleting them is a
  separate cleanup. Do NOT touch `tasks.go` — note them in Maintenance instead.
- The `ucard_palette` define in `cards.html` — still live (GET /ui/cards).
- `models.html` — it already reads only `.Name`/`.Size`; no edit needed.
- The Ollama gguf-tag rename (a separate, not-yet-written plan): there is no
  `plans/070-*.md` at HEAD. If such a plan later edits `manager.go`'s `Model`
  struct, the only overlap is the same struct — but the `Path` removal here is
  *independent* of any field rename and does not block it.

## Git workflow

- Branch: `improve/069-gomponents-dead-code-cleanup`
- One commit; conventional-commit style, e.g.
  `refactor(web): drop legacy heads renderer + calendar/Path fossils`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

Do these in order. After Step 1 the codebase still builds with the legacy
renderer present but unused; Step 2 removes it. Never leave the tree broken
between steps.

### Step 1: Point the two heads re-patch sites at the registry

In `internal/web/heads.go`, replace **both** `renderCardHeads` calls with a
`cardInto(..., "heads", nil)` render. `cardInto` is a method on `*handlers`
(in `cards.go`) and takes `(w io.Writer, typ string, params map[string]string)`.

`setActiveHead` — change lines 36–39 from:

```go
	var card strings.Builder
	if err := h.renderCardHeads(&card, nil); err == nil {
		_ = sse.PatchElements(card.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	}
```

to:

```go
	var card strings.Builder
	if err := h.cardInto(&card, "heads", nil); err == nil {
		_ = sse.PatchElements(card.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	}
```

`renderHeadsCard` — change lines 45–47 from:

```go
	var b strings.Builder
	if err := h.renderCardHeads(&b, nil); err != nil {
		return e.InternalServerError("rendering heads card", err)
	}
```

to:

```go
	var b strings.Builder
	if err := h.cardInto(&b, "heads", nil); err != nil {
		return e.InternalServerError("rendering heads card", err)
	}
```

Do not change anything else in `heads.go` (the `strings.Builder`, the
`PatchElements` call, `createHead`/`deleteHead` all stay).

**Verify**:
```
gofmt -l internal/web/heads.go      # prints nothing
go build ./internal/web             # exit 0 (renderCardHeads now unused but still defined → still builds)
```

### Step 2: Delete the legacy heads renderer and its three structs from `cards.go`

In `internal/web/cards.go`, delete the entire block from the
`// ---- heads tile (still legacy) ----` comment (line 159) through the end of
the `renderCardHeads` function (the closing `}` at line 210) — i.e. the comment
banner, `headGroupChoice`, `headManageRow`, `cardHeadsView`, and
`renderCardHeads`.

Then remove the two imports that this deletion orphans. After the deletion,
`heads.` and `store.` are no longer referenced anywhere in `cards.go` (they were
used only inside the deleted code). Delete these two lines from the import block:

```go
	"github.com/alexradunet/balaur/internal/heads"
```
```go
	"github.com/alexradunet/balaur/internal/store"
```

Leave every other import (`io`, `knowledge`, `cards`, `ui`, etc.) — they are
still used (`io` by `cardInto`, `knowledge` by `proposalBody`, …).

Also update the file's top-of-file package comment if it names the heads tile as
"still-legacy" (the doc block lines 3–11 mention "the still-legacy heads tile
(re-patched directly by heads.go after set-active/create)"). Trim that clause so
the comment no longer claims a legacy heads renderer lives here — keep it
accurate, do not expand it.

**Verify**:
```
grep -n 'renderCardHeads\|headManageRow\|cardHeadsView\|headGroupChoice' internal/web/cards.go   # no matches
grep -n '"github.com/alexradunet/balaur/internal/heads"\|"github.com/alexradunet/balaur/internal/store"' internal/web/cards.go   # no matches
go build ./internal/web             # exit 0
```

### Step 3: Delete the `ucard_heads` define from the template

In `web/templates/cards.html`, delete the `ucard_heads` block: the comment
banner at lines 38–40 (the `------` separator, `heads card …`, and
`context: cardHeadsView` comments) through `{{end}}` at line 98.

Then fix the file's header comment (lines 1–6). It currently says three templates
remain: `ucard_heads`, `ucard_calendar`, `ucard_palette`. After this step and
Step 4 only `ucard_palette` remains, so update that header to name only the
template(s) that still exist. (You will remove `ucard_calendar` in Step 4; you
may edit the header once now to its final state — naming only `ucard_palette` —
rather than twice.)

**Verify**:
```
grep -n 'ucard_heads' web/templates/cards.html     # no matches
go test ./internal/web -run TestTemplatesParse     # passes (templates still parse)
```

### Step 4: Delete the calendar fossil (struct + template + test)

Three coordinated deletions:

1. `internal/web/cards.go` — delete `calendarCardView` and its doc comment
   (lines 152–157, the `// calendarCardView feeds …` comment through the struct's
   closing `}`).
2. `web/templates/cards.html` — delete the `ucard_calendar` block: the comment
   banner at lines 8–10 through `{{end}}` at line 36.
3. `internal/web/templates_test.go` — delete the whole
   `TestCalendarCellsLinkToDayPages` function (lines 238–251).

Do not touch `buildCalendar`/`calView` in `tasks.go` (out of scope — see Scope).
The `time` import in `templates_test.go` is still used by other tests
(`TestLifeBodyRenders` uses `time.Now`, `TestDayPageRenders` etc. — actually
verify: `time` is used in `TestQuestsFocusListRenders` via `time.Now()`), so do
NOT remove it. Confirm with the grep in Verify.

**Verify**:
```
grep -n 'calendarCardView\|ucard_calendar\|TestCalendarCellsLinkToDayPages' internal/web/cards.go internal/web/templates_test.go web/templates/cards.html   # no matches
grep -n '"time"\|time\.' internal/web/templates_test.go   # "time" import still present AND still referenced (do not delete it)
go test ./internal/web                                     # all pass
```

### Step 5: Delete the dead `Path` field from `ollama.Model`

In `internal/ollama/manager.go`, change the `Model` type (lines 14–20) from:

```go
// Model is one model present in Ollama's local store. Path is always empty
// (Ollama owns the blob store); kept so existing templates bind unchanged.
type Model struct {
	Name string
	Size int64
	Path string
}
```

to:

```go
// Model is one model present in Ollama's local store. Field names mirror the
// `GgufFiles` template loop (web/templates/models.html), which reads Name and Size.
type Model struct {
	Name string
	Size int64
}
```

(Keep the comment honest and short — name only the fields that exist. Do not
touch `fetchModels`; it already constructs `Model{Name:…, Size:…}`.)

**Verify**:
```
grep -rn 'Model{' internal/ollama/manager.go               # the literal sets only Name/Size (already true)
grep -rn '\.Path' internal/ollama web/templates internal/web | grep -i 'model\|gguf' || echo "no Model.Path readers"   # → "no Model.Path readers"
go build ./internal/ollama                                  # exit 0
```

### Step 6: Whole-tree verification

```
gofmt -l .
go vet ./...
CGO_ENABLED=0 go build ./...
go test ./...
git diff --check
```

**Verify**: `gofmt -l .` prints nothing; vet clean; build exits 0; all tests
pass; `git diff --check` prints nothing.

## Test plan

- **No new tests.** This plan is pure deletion of duplicate/dead code.
- Coverage is preserved by tests that already exist:
  - The heads card UI is covered by `internal/feature/headscards/*_test.go` and
    by `internal/cards/cards_test.go` (asserts `"heads"` is a registered card
    type). `internal/web/cards_test.go`'s `TestUiCardAllTypesRender` renders every
    registry card type, including `"heads"`, through `cardInto`.
  - The calendar card is covered by `internal/feature/taskcards/calendar_test.go`
    (renders `CalendarCard` over its own `taskcards.CalView`, asserts the
    `/focus/day?date=…` deep-link — the exact assertion the deleted
    `TestCalendarCellsLinkToDayPages` made).
- Verification: `go test ./...` → all pass, with **no** test referencing
  `ucard_heads`, `ucard_calendar`, or `renderCardHeads`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/web/heads.go` calls `h.cardInto(..., "heads", nil)` in both
      `setActiveHead` and `renderHeadsCard`; `renderCardHeads` is gone.
- [ ] `grep -rn 'renderCardHeads\|cardHeadsView\|headManageRow\|headGroupChoice' internal/web` → no matches
- [ ] `grep -rn 'calendarCardView\|TestCalendarCellsLinkToDayPages' internal/web` → no matches
- [ ] `grep -rn 'ucard_heads\|ucard_calendar' web/templates internal/web` → no matches
      (the only remaining `ucard_*` defines are `ucard_palette`)
- [ ] `grep -n 'Path' internal/ollama/manager.go` → no matches (field and comment gone)
- [ ] `grep -rn '\.Path' internal/ollama web/templates internal/web | grep -i 'model'` → no matches
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./...` passes
- [ ] `gofmt -l .` prints nothing and `git diff --check` prints nothing
- [ ] `git status --porcelain` shows only the five in-scope files modified
- [ ] `plans/readme.md` status row for 069 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- **The registry "heads" card is NOT a superset of `ucard_heads`.** Before Step 1,
  open `internal/feature/headscards/heads.go` and confirm `HeadsCard` still
  renders all four pieces the template had: active/built-in **tags**, the
  **Make active** form, the **Delete** form (non-built-in heads), and the full
  **+ New head** form (tools checkboxes + avatar radios). If any is missing,
  switching the re-patch sites would ship a visible UI regression — STOP and
  report which piece is missing rather than dedup.
- **Any "Current state" excerpt has drifted** (the drift-check diff is non-empty
  and the live code differs from what is quoted here) — re-locate before editing;
  if line numbers moved but the code is identical, proceed; if the code itself
  changed, STOP.
- After deleting the two imports in Step 2, `go build ./internal/web` reports an
  import as **still used** (something other than the deleted code referenced
  `heads`/`store` after all) — STOP and report; do not blindly re-add or delete
  a still-referenced import.
- Removing the `time` import from `templates_test.go` is tempting in Step 4 —
  **don't**: other tests still use it. If the grep shows `time.` still
  referenced, leave the import. If you removed it and the build broke, restore it.
- `go test ./...` fails after any step and a single obvious fix does not resolve
  it — report the failing test and output.

## Maintenance notes

For whoever owns this code after the change lands:

- **`buildCalendar` and `calView` in `internal/web/tasks.go` become dead** once
  the calendar test is deleted (the test was their last `internal/web` caller).
  They still compile (unused package-level symbols are legal in Go), so this plan
  deliberately leaves them to keep the diff surgical. A follow-up can delete
  `buildCalendar`/`calView` from `tasks.go` — verify with
  `grep -rn 'buildCalendar\|calView' internal/web` returning only the definitions
  before removing. (taskcards has its own independent `CalView`; deleting the
  `internal/web` copies does not affect the live calendar card.)
- The heads card now has exactly one renderer (`headscards.HeadsCard`). Any future
  change to the heads manage UI goes there and nowhere else; there is no longer a
  template to keep in sync.
- A reviewer should confirm: (1) the two `cardInto(..., "heads", nil)` calls
  patch the same `#ucard-heads` element the old code did (selector unchanged),
  and (2) `go test ./internal/web` and `go test ./internal/feature/...` both pass
  — the registry path is what's exercised now.
- If a later plan renames Ollama's gguf tag handling and touches `Model`, this
  `Path` removal is orthogonal and should already be merged; no coordination
  needed beyond a trivial merge of the same struct.
