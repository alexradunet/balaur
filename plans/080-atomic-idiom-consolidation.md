# Plan 080: Consolidate UI idioms — one ErrorStrip, variadic attrs on interactive atoms, typed Datastar + qualified imports

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 080 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/ui internal/web/cards.go internal/web/knowledge.go internal/feature internal/web/assets/static/basm.css` — if any in-scope file changed since this plan was written, compare the "Current state" excerpts to the live code; on mismatch, STOP.

## Status
- **Priority**: P2
- **Effort**: M
- **Risk**: LOW-MED
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
The component layer has drifted from its own conventions, recorded in `.claude/skills/ui-development/SKILL.md`: the card-error strip is implemented four separate ways (one safe atom plus three raw-string copies), five interactive atoms omit the required trailing `attrs ...g.Node` pass-through that every caller needs to attach Datastar wiring, two feature packages dot-import `gomponents/html` against the qualified-`h.` rule, and Datastar attributes are written two ways (the typed `data.*` helper in some cards, raw `g.Attr("data-…")` strings in others). Each divergence is a small correctness/maintainability hazard: the raw error-string copies bypass the no-raw-HTML escaping firewall the atom enforces, and atoms without attr pass-through force callers to hand-roll markup the storybook is supposed to own. Consolidating to one idiom per concern makes the system inspectable and keeps the storybook honest as the source of truth.

## Current state

### C1 — the error strip is implemented four times
`internal/ui/components.go:12-14` is the firewall atom (g.Text auto-escapes; never g.Raw):
```go
// internal/ui/components.go:12
func ErrorStrip(msg string) g.Node {
	return h.Div(h.Class("card-note card-note-error"), g.Text(msg))
}
```
Three more copies render the same markup as raw strings:
- `internal/web/cards.go:139-141` — the helper `cardErrorStrip`:
```go
func cardErrorStrip(msg string) template.HTML {
	return template.HTML(`<div class="card-note card-note-error">` + html.EscapeString(msg) + `</div>`)
}
```
  used at `cards.go:103,107,112,123,126,132` (six call sites).
- `internal/web/cards.go:63` — an inline `fmt.Fprintf` variant that adds an `id`:
```go
fmt.Fprintf(e.Response, `<div class="card-note card-note-error" id="ucard-%s">%s</div>`, typ, html.EscapeString(err.Error()))
```
- `internal/web/knowledge.go:230-235` — the helper `cardError`, called at `knowledge.go:135,172,213`:
```go
func (h *handlers) cardError(e *core.RequestEvent, err error) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.WriteHeader(http.StatusUnprocessableEntity)
	fmt.Fprintf(e.Response, `<div class="card-note card-note-error">%s</div>`, html.EscapeString(err.Error()))
	return nil
}
```
The atom and the no-id `cardErrorStrip` emit byte-identical markup, so routing `cardErrorStrip` through the atom is output-stable. The `cards.go:63` variant has an extra `id="ucard-<typ>"` attribute; the atom takes no id, so this one needs an id-carrying variant (see Step 2).

### C2 — interactive atoms missing the variadic `attrs ...g.Node` pass-through
SKILL.md states the rule (re-read confirmed):
> - Atoms take trailing variadic `attrs ...g.Node` so callers can pass Datastar attributes through without the atom knowing about them. (`.claude/skills/ui-development/SKILL.md:63`)
> - **Always use the QUALIFIED `h.` import — never dot-import `html`.** … This is non-negotiable in `internal/ui`. (SKILL.md:55-57)

Exemplar with the pass-through (`internal/ui/button.go:36-41`):
```go
func Button(p ButtonProps, children ...g.Node) g.Node {
	if p.Href != "" {
		return h.A(append([]g.Node{h.Class(buttonClass(p)), h.Href(p.Href)}, children...)...)
	}
	return h.Button(append([]g.Node{h.Class(buttonClass(p))}, children...)...)
}
```
Atoms WITHOUT the pass-through (confirmed signatures at HEAD):
- `internal/ui/listitem.go:20` — `func ListItem(p ListItemProps) g.Node` (root is `h.A` or `h.Div`, line 49-52).
- `internal/ui/list.go:17` — `func List(p ListProps) g.Node` (root `h.Div`, line 26).
- `internal/ui/pagination.go:22` — `func Pagination(p PagerProps) g.Node` (root `h.Nav`, line 44).
- `internal/ui/breadcrumb.go:16` — `func Breadcrumb(items []Crumb) g.Node` (root `h.Nav`, line 29).
- `internal/ui/calendarcell.go:24` — `func CalendarCell(p CalendarCellProps) g.Node` (root `h.Button`, line 43-46).
- `internal/ui/dayentry.go:20` — `func DayEntry(p DayEntryProps) g.Node` (root `h.Div`, line 33).
- `internal/ui/tabs.go:18` — `func Tabs(items []TabItem) g.Node`. Tabs has TWO axes (the strip and per-item); per the spec, add a per-item `Attrs []g.Node` to `TabItem` (struct at `tabs.go:9-12`) rather than a variadic on the strip — the storybook tab needs per-tab Datastar.

### #16 — dot-imports in feature-card packages (confirmed)
- `internal/feature/journalcards/journalfocus.go:9` — `. "maragu.dev/gomponents/html"`
- `internal/feature/knowledgecards/memory.go:14` — `. "maragu.dev/gomponents/html"`

The recent convergence uses qualified `h.` (see `internal/ui/avatar.go`, `internal/feature/modelcards/*`). Note: SKILL.md's "non-negotiable" wording scopes the rule to `internal/ui`, but the project is converging feature packages onto `h.` too — this plan finishes the two stragglers. (Other feature files like `knowledgefocus.go`, `today.go`, `taskcard.go`, `dayfocus.go` ALSO dot-import; they are OUT of scope for this plan unless a Step below names them, to keep the diff small — note them in Maintenance.)

### #17 — Datastar written two ways
Typed `data.*` helper users (import `data "maragu.dev/gomponents-datastar"`): `headscards/heads.go`, `taskcards/taskcard.go`, `taskcards/today.go`, `knowledgecards/memory.go`, `knowledgecards/skills.go`, `modelcards/modelcard.go`, `modelcards/panel.go`. Exemplar (`internal/feature/headscards/heads.go:152`):
```go
data.On("submit", "@post('/ui/heads/active', {contentType:'form'})", data.ModifierPrevent),
```
Raw `g.Attr("data-…")` users (count of raw sites per file, confirmed): `journalcards/dayfocus.go` (5), `journalcards/journalfocus.go` (3), `settingscards/settingsfocus.go` (5), `knowledgecards/knowledgefocus.go` (8), `taskcards/questsfocus.go` (2).

**Typed-helper API reality (v0.3.3, re-read from the module source):** the package has `data.On(event, expression string, modifiers ...Modifier)`, `data.Bind(name)`, `data.Signals(map[string]any, ...Modifier)`, `data.Class(pairs...)`, `data.Attr(pairs...)`, `data.Computed`, `data.Text`, `data.Show`, `data.Ref`, `data.Indicator`. **There is NO `data.Get` / `data.Post`** — the spec's reference to `data.Get` is wrong; `@get(...)`/`@post(...)` are just the *expression string* you pass as the second arg to `data.On`. Modifiers are `data.ModifierPrevent`, `data.ModifierDebounce` (a duration is appended via the modifier's argument form), etc.

**LOAD-BEARING TRAP for #17:** `data.Class("k-tab-active", "$category===''")` renders the OBJECT form `data-class="{k-tab-active: $category===''}"`, but the existing raw code at `knowledgefocus.go:105,117` uses the KEY-IN-ATTRIBUTE form `data-class:k-tab-active="$category===''"`. These are DIFFERENT bytes. Likewise `data.Signals` renders `data-signals` JSON, not the `data-signals:q` key form at `knowledgefocus.go:84-85`. Mechanically swapping these changes rendered HTML. Only `data-on:event__mods` → `data.On(event, expr, mods…)` is a clean, byte-equivalent swap (it produces `data-on-<event>` / `data-on:<event>` matching the existing string). **Scope #17 to the safe `data-on` swaps only; leave `data-signals:`/`data-class:`/`data-bind:` raw and note why.**

### sidebar-style — string-concat background
`internal/ui/shell/sidebar.go:111`:
```go
attrs = append(attrs, h.Span(h.Class("sb-nav-dot"), h.Style("background:"+it.Dot)))
```
The convention (custom-prop, see `internal/ui/avatar.go:40` `h.Style("--avatar-size:"+…)` and `internal/ui/skeleton.go`) is to set a custom property and let CSS consume it. The CSS rule already exists (`internal/web/assets/static/basm.css:2666`) but has NO background today — the inline style is the only source:
```css
.sb-nav-dot { width: 6px; height: 6px; flex-shrink: 0; }
```
**Byte-output pin:** `internal/ui/shell/sidebar_test.go:64` asserts the literal `style="background:var(--teal)"`. Changing the emit form REQUIRES updating that test line in the same step.

### Tests that pin byte output (must stay green)
- `internal/web/templates_test.go` — `TestQuestsFocusListRenders` (:121), `TestDayPageRenders` (:176), `TestModelsPageAndCleanChatbarRender` (:45), chat structure tests, etc. These exercise feature renderers; a markup change there is a STOP.
- Atom tests: `internal/ui/button_test.go` (variadic exemplar), `internal/ui/tabs_test.go:14-16`, `internal/ui/listitem_test.go:18,29`, `internal/ui/components_test.go` (ErrorStrip escaping), plus `breadcrumb_test.go`, `pagination_test.go`, `calendarcell_test.go`, `dayentry_test.go`, `list_test.go`. Adding a trailing variadic with NO attrs passed is byte-stable (append of empty slice) — confirm each still passes.
- `internal/ui/shell/sidebar_test.go` — pins the sidebar dot style (see sidebar-style above).
- `internal/feature/storybook/story_test.go` — `TestAllStoriesRender`; stories must still render after Props-table / signature changes.

## Commands you will need
| Purpose | Command | Expected |
| Drift check | `git diff --stat 12a2ff5..HEAD -- internal/ui internal/web/cards.go internal/web/knowledge.go internal/feature internal/web/assets/static/basm.css` | empty (else compare excerpts) |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Test (ui atoms) | `go test ./internal/ui/...` | ok |
| Test (storybook render) | `go test ./internal/feature/storybook/...` | ok |
| Test (web byte pins) | `go test ./internal/web/...` | ok |
| Format | `gofmt -l <changed.go files>` | empty output |
| Whitespace | `git diff --check` | no output |
| Grep: remaining dot-imports in scope | `grep -n '\. "maragu.dev/gomponents/html"' internal/feature/journalcards/journalfocus.go internal/feature/knowledgecards/memory.go` | no output after Step 5 |
| Grep: raw data-on in scope | `grep -rn 'g.Attr("data-on' internal/feature/journalcards/journalfocus.go internal/feature/settingscards/settingsfocus.go internal/feature/taskcards/questsfocus.go` | no output after Step 7 (if done) |

> Sandbox note: in a TLS-intercepting Hyperagent sandbox, Go commands need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. GOSUMDB stays on.

## Scope
**In scope** (only files you may modify):
- `internal/ui/components.go` (ErrorStrip + an id variant)
- `internal/ui/tabs.go`, `internal/ui/listitem.go`, `internal/ui/list.go`, `internal/ui/pagination.go`, `internal/ui/breadcrumb.go`, `internal/ui/calendarcell.go`, `internal/ui/dayentry.go` (variadic attrs / TabItem.Attrs)
- `internal/ui/shell/sidebar.go` (custom-prop dot)
- `internal/web/cards.go`, `internal/web/knowledge.go` (route error strips through the atom)
- `internal/feature/journalcards/journalfocus.go`, `internal/feature/knowledgecards/memory.go` (qualified `h.` import)
- `internal/feature/settingscards/settingsfocus.go`, `internal/feature/taskcards/questsfocus.go`, `internal/feature/journalcards/journalfocus.go` (raw `data-on` → typed `data.On`, ONLY if Step 7 stays ≤S)
- `internal/web/assets/static/basm.css` (add `background: var(--sb-nav-dot)` to the existing `.sb-nav-dot` rule)
- `internal/feature/storybook/stories_navigation.go`, `internal/feature/storybook/stories_display.go` (Props tables for changed atoms)
- The `*_test.go` that pin signatures/output: `internal/ui/*_test.go` for the changed atoms, `internal/ui/shell/sidebar_test.go`

**Out of scope** (do NOT touch):
- The chat organisms `internal/ui/chat/*` — owned by plan 084.
- `internal/ui/emptystate.go` and EmptyState consolidation — owned by plan 081.
- `journalcards/journalfocus.go` hand-rolled tab JS at lines 96-110 (#9) — DEFER unless Step 6 stays ≤S; see Step 6 STOP. Converting it needs a Datastar-driven Tabs variant and risks changing rendered markup pinned elsewhere.
- `data-signals:` / `data-class:` / `data-bind:` raw attrs in `knowledgefocus.go` and `dayfocus.go` — the typed helpers render different bytes (see the #17 trap above). Leave raw.
- Other feature files that dot-import `html` (`knowledgefocus.go`, `today.go`, `taskcard.go`, `dayfocus.go`, etc.) — not in this plan's scope; note as deferred.

## Git workflow
Branch `improve/080-atomic-idiom-consolidation`. Conventional commits, one per logical step (e.g. `refactor(ui): add variadic attrs to interactive atoms`, `refactor(web): route card error strips through ui.ErrorStrip`, `refactor(ui): sidebar dot uses --sb-nav-dot custom prop`). Do NOT push or open a PR unless told.

## Steps

Order is chosen so the build stays green and the lowest-risk, byte-stable changes land first. After EACH step run the Verify before continuing.

### Step 1: Add the variadic `attrs ...g.Node` pass-through to the six single-root atoms
In each file, change the signature to accept trailing `attrs ...g.Node` and append them to the ROOT element's attribute slice (the pattern from `button.go:36-41`). Append AFTER the existing class/href attrs but the exact position among attrs is fine as long as it stays on the root element and tests pass.

- `internal/ui/listitem.go`: `func ListItem(p ListItemProps, attrs ...g.Node) g.Node`. The root is built in `root := []g.Node{h.Class(cls)}` (line 29) and returned as `h.A(root...)` or `h.Div(root...)` (line 49-52). Append `attrs` to `root` just before the return.
- `internal/ui/list.go`: `func List(p ListProps, attrs ...g.Node) g.Node`. Root `kids := []g.Node{h.Class("list")}` (line 18), returned `h.Div(kids...)` (line 26). Append `attrs` to `kids` before return.
- `internal/ui/pagination.go`: `func Pagination(p PagerProps, attrs ...g.Node) g.Node`. Root `kids` (line 32), returned `h.Nav(kids...)` (line 44). Append `attrs` before return.
- `internal/ui/breadcrumb.go`: `func Breadcrumb(items []Crumb, attrs ...g.Node) g.Node`. Root `kids` (line 17), returned `h.Nav(kids...)` (line 29). Append `attrs` before return.
- `internal/ui/calendarcell.go`: `func CalendarCell(p CalendarCellProps, attrs ...g.Node) g.Node`. Root is the `h.Button(...)` at line 43. Restructure to build a `root := []g.Node{h.Class(cls), h.Type("button")}`, append the two `h.Span` children, then `append(root, attrs...)`, and return `h.Button(root...)`.
- `internal/ui/dayentry.go`: `func DayEntry(p DayEntryProps, attrs ...g.Node) g.Node`. Root `h.Div(h.Class(cls), …)` at line 33. Restructure to a `root := []g.Node{h.Class(cls)}` slice, append the three child divs, then `attrs...`, return `h.Div(root...)`.

Do NOT change any existing caller — appending an empty variadic is output-stable. (A hook gofmt-s edited files.)

**Verify**: `go test ./internal/ui/...` → ok (byte-output tests for these atoms still pass with no attrs passed); `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Add `TabItem.Attrs` for per-tab Datastar
In `internal/ui/tabs.go`, add `Attrs []g.Node` to the `TabItem` struct (after line 11). In the loop (line 25-30), append `it.Attrs...` to the per-tab `attrs` slice before `g.Text(it.Label)` (so attrs land on the `<a>`). Existing callers leave `Attrs` nil → byte-stable.

**Verify**: `go test ./internal/ui/...` → ok (`TestTabs` in `tabs_test.go` still passes — nil Attrs).

### Step 3: Add an id-carrying ErrorStrip variant and route the four error strips through the atom
In `internal/ui/components.go`, add a sibling that emits the optional id while keeping the g.Text firewall (NEVER g.Raw):
```go
// ErrorStripID renders the card-error strip with an element id (used where the
// SSE target needs to address it). Same firewall as ErrorStrip: g.Text escapes.
func ErrorStripID(id, msg string) g.Node {
	return h.Div(h.Class("card-note card-note-error"), h.ID(id), g.Text(msg))
}
```
Then re-point the four web copies at the atom. Render the atom to a string with a small helper or `node.Render(&b)`:

- `internal/web/cards.go:139-141` `cardErrorStrip`: rebuild it on the atom so all six callers (`:103,107,112,123,126,132`) flow through it without touching them:
```go
func cardErrorStrip(msg string) template.HTML {
	var b strings.Builder
	_ = ui.ErrorStrip(msg).Render(&b)
	return template.HTML(b.String())
}
```
  (`strings`, `template`, and `ui` are already imported in cards.go.) **Byte check:** the atom emits `<div class="card-note card-note-error">…escaped…</div>` — identical to the current raw string. If `go test ./internal/web/...` flags a diff, STOP.
- `internal/web/cards.go:52-69` `uiCard` — replace the `fmt.Fprintf` at line 63 with the id variant:
```go
e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
e.Response.WriteHeader(http.StatusOK)
_ = ui.ErrorStripID("ucard-"+typ, err.Error()).Render(e.Response)
return nil
```
  This drops the `fmt`/`html` use HERE only — do NOT remove the `fmt`/`html` imports yet (other call sites in cards.go still use them; confirm with `go build`).
- `internal/web/knowledge.go:230-235` `cardError` — replace the `fmt.Fprintf`:
```go
func (h *handlers) cardError(e *core.RequestEvent, err error) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.WriteHeader(http.StatusUnprocessableEntity)
	return ui.ErrorStrip(err.Error()).Render(e.Response)
}
```
  Add `"github.com/alexradunet/balaur/internal/ui"` to knowledge.go's imports if not present (check the import block). If `fmt`/`html` become unused in knowledge.go after this, remove ONLY those now-orphaned imports (your change created the orphan — per the surgical-changes rule).

**Verify**: `go vet ./...` → exit 0; `go test ./internal/web/... ./internal/ui/...` → ok; `gofmt -l internal/web/cards.go internal/web/knowledge.go internal/ui/components.go` → empty.

### Step 4: Convert the sidebar dot to a `--sb-nav-dot` custom property
In `internal/ui/shell/sidebar.go:111`, change:
```go
attrs = append(attrs, h.Span(h.Class("sb-nav-dot"), h.Style("background:"+it.Dot)))
```
to:
```go
attrs = append(attrs, h.Span(h.Class("sb-nav-dot"), h.Style("--sb-nav-dot:"+it.Dot)))
```
In `internal/web/assets/static/basm.css:2666`, add the background consumer to the existing rule:
```css
.sb-nav-dot { width: 6px; height: 6px; flex-shrink: 0; background: var(--sb-nav-dot); }
```
In `internal/ui/shell/sidebar_test.go:64`, update the pinned string from `style="background:var(--teal)"` to `style="--sb-nav-dot:var(--teal)"`.

**Verify**: `go test ./internal/ui/shell/...` → ok; `git diff --check` → no output. Visual check deferred to Done criteria (the dot must still show the group colour in BOTH modes).

### Step 5: Replace the two dot-imports with qualified `h.`
- `internal/feature/journalcards/journalfocus.go:9`: change `. "maragu.dev/gomponents/html"` → `h "maragu.dev/gomponents/html"`, then prefix every bare HTML constructor in the file with `h.` (`Div`→`h.Div`, `Form`→`h.Form`, `Class`→`h.Class`, `ID`→`h.ID`, `Textarea`/`Name`/`Rows`/`Placeholder`/`Button`/`Type`/`Article`/`Span`/`A`/`Href`/`P`→`h.…`). Leave `g.Text`/`g.Attr`/`g.Group`/`g.Node` as-is. (Note: `journalfocus.go` ALSO appears in Step 7's #17 scope — do Step 5 first, then Step 7 edits the now-`h.`-qualified file.)
- `internal/feature/knowledgecards/memory.go:14`: same conversion — change the dot-import to `h "…/html"` and qualify every bare HTML constructor in the file with `h.`. (memory.go already imports `data "maragu.dev/gomponents-datastar"` and `g`; keep those.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./internal/feature/...` → ok; `grep -n '\. "maragu.dev/gomponents/html"' internal/feature/journalcards/journalfocus.go internal/feature/knowledgecards/memory.go` → no output.

### Step 6: (#9, OPTIONAL — DEFER if it grows) hand-rolled tab JS
The inline `document.getElementById(...).classList…` tab switching in `journalfocus.go:99-106` is hand-rolled DOM JS in a `data-on:click` string. Converting it to a Datastar signal + `ui.Tabs` with the new `TabItem.Attrs` is desirable but RISKS changing rendered markup. **Treat this as a STOP-gate, not a requirement:** if a clean conversion is not obviously ≤S and byte-safe, SKIP it, leave the JS as-is, and add a Maintenance note. Do NOT spend the plan's budget here.

> NOTE the SPEC cited `knowledgefocus.go:105-118` as a second hand-rolled-JS site — re-reading shows those lines are Datastar `data-class:k-tab-active` / `data-on:click__prevent` expressions (real Datastar, not DOM JS), and they use the key-in-attribute `data-class:` form that the typed helper does NOT reproduce byte-for-byte. So `knowledgefocus.go` has NO hand-rolled-JS to convert and its tabs stay raw. Do not touch it.

**Verify (if attempted)**: `go test ./...` → all pass; manual click of both tabs on `/focus/journal` toggles the active state. If any byte-pin test fails, revert this step.

### Step 7: (#17 tail) raw `data-on` → typed `data.On` — ONLY the byte-equivalent swaps
For each raw `g.Attr("data-on:<event>__<mods>", "<expr>")` in `settingscards/settingsfocus.go`, `taskcards/questsfocus.go`, and `journalcards/journalfocus.go`, replace with the typed form, adding `data "maragu.dev/gomponents-datastar"` to the import block:
```go
// before
g.Attr("data-on:submit__prevent", "@post('/ui/profile/name', {contentType:'form'})"),
// after
data.On("submit", "@post('/ui/profile/name', {contentType:'form'})", data.ModifierPrevent),
```
**Mapping rules (re-read from the v0.3.3 module):**
- `data-on:click__prevent` → `data.On("click", expr, data.ModifierPrevent)`
- `data-on:submit__prevent` → `data.On("submit", expr, data.ModifierPrevent)`
- A debounce like `data-on:input__debounce.250ms` → keep RAW (the typed debounce modifier's duration-argument form needs verification and is not worth the risk); only convert the plain `__prevent`/no-modifier `data-on` sites.
- Do NOT convert `data-signals:`, `data-class:`, `data-bind:`, `data-on:input__debounce` — they are NOT byte-equivalent under the typed helpers (see the #17 trap).
- **In `journalfocus.go` convert ONLY the line-68 `data-on:submit__prevent` site.** The two `data-on:click` sites at lines 99/104 carry the hand-rolled DOM-JS expressions that Step 6 defers — leave those raw regardless of Step 7 (do NOT convert them just because they have no modifier).

**STOP-gate:** if the typed output for any converted site differs in bytes from the raw string AND a test pins that markup (run `go test ./internal/web/...` after each file), revert that file and leave it raw. If the conversion across the three files exceeds S effort, do what's clean and note the rest.

**Verify**: `go vet ./...` → exit 0; `go test ./...` → all pass; `gofmt -l` on each touched file → empty.

### Step 8: Update the storybook Props tables for changed atoms
The Props tables document the public surface; they must match the new signatures. Add a row to each changed atom's `Props: []Prop{…}` in:
- `internal/feature/storybook/stories_navigation.go` — `tabsStory()` (Props at :33-37; add a `{"Attrs", "[]g.Node", "nil", "Per-tab pass-through attributes — Datastar wiring for in-place switching."}` row to TabItem props), `breadcrumbStory()` (:60-63), `paginationStory()` (:90-94). For Breadcrumb/Pagination add a trailing-attrs note row, e.g. `{"attrs …g.Node", "variadic", "—", "Extra root attributes (Datastar) passed through."}`.
- `internal/feature/storybook/stories_display.go` — `listStory()` (ID `list`, Props near :29/:57), `calendarcellStory()` (ID `calendarcell`), `dayentryStory()` (ID `dayentry`). Add the same trailing-attrs note row.

Keep wording in the existing voice (short, lowercase content notes). The Prop struct shape is `{Name, Type, Default, Desc string}` (4 string fields, per the existing rows).

**Verify**: `go test ./internal/feature/storybook/...` → ok (`TestAllStoriesRender`); `gofmt -l` on the two story files → empty.

## Test plan
- **Variadic pass-through (new assertions).** Add one focused test per concern proving an attr renders through to the root. Pattern is `internal/ui/button_test.go` (render to string, assert substring). Extend the existing atom tests (do NOT create parallel files):
  - `internal/ui/listitem_test.go`: a case `ui.ListItem(ui.ListItemProps{Title:"x"}, g.Attr("data-test","1"))` asserting the rendered root `<div …>` (or `<a>`) contains `data-test="1"`.
  - `internal/ui/tabs_test.go`: a `TabItem{Label:"x", Attrs: []g.Node{g.Attr("data-test","1")}}` case asserting the `<a>` carries `data-test="1"`.
  - Add equivalent one-liner cases to `pagination_test.go`, `breadcrumb_test.go`, `calendarcell_test.go`, `dayentry_test.go`, `list_test.go` (each: pass `g.Attr("data-test","1")`, assert it appears). Keep them minimal.
- **ErrorStrip firewall preserved.** `internal/ui/components_test.go:TestErrorStripRendersAndEscapes` must still pass; add a sibling `TestErrorStripIDRendersAndEscapes` asserting the id appears AND `<script>` is escaped.
- **Byte pins unchanged.** `go test ./internal/web/...` and `internal/ui/shell/sidebar_test.go` (after the Step 4 edit) must pass — these are the regression guard for "output-stable" claims.
- **Storybook render.** `go test ./internal/feature/storybook/...` (`TestAllStoriesRender`) green after Props-table edits.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `go vet ./...` → exit 0
- [ ] `go test ./...` → all pass
- [ ] `go test ./internal/feature/storybook/...` → ok
- [ ] `grep -n '\. "maragu.dev/gomponents/html"' internal/feature/journalcards/journalfocus.go internal/feature/knowledgecards/memory.go` → no output
- [ ] `grep -c 'card-note card-note-error' internal/web/cards.go internal/web/knowledge.go` → cards.go shows ONLY the `cardErrorStrip` helper body has no raw literal (the raw `<div class="card-note…` literals at the old `:63` and knowledge.go `:233` are gone — i.e. those two files no longer contain the raw `<div class="card-note card-note-error"` string)
- [ ] `gofmt -l` on every changed `.go` file → empty
- [ ] `git diff --check` → no output
- [ ] Only in-scope files changed (`git status` shows nothing outside Scope)
- [ ] update the 080 row in plans/readme.md (add the row if it is not present yet, matching the existing column format)
- [ ] **VISUAL (CSS/markup changed):** run the app (it may already be serving on 127.0.0.1:8090; else `go run . serve --http=127.0.0.1:8090`), open `/storybook/tabs`, `/storybook/list`, and a page using the sidebar; force `document.documentElement.className='theme-hearthwood dark'` then `='theme-hearthwood light'`; confirm the sidebar group-dots still show their colour (the `--sb-nav-dot` custom prop resolves) and tabs/list render unchanged in BOTH modes and at ≤920px width.

## STOP conditions
- The drift check shows an in-scope file changed since `12a2ff5` and its excerpt no longer matches — STOP and report.
- A byte-output test in `internal/web/templates_test.go` or any atom test fails after a change labelled "output-stable" — you altered markup. Revert THAT item, leave a Maintenance note, keep the rest.
- A typed-Datastar conversion (#17) produces different bytes than the raw string for a pinned site — revert that site to raw (this is expected for `data-signals:`/`data-class:`/`data-bind:`/debounce; only `__prevent`/plain `data-on` are convertible).
- Step 6 (#9) or Step 7 (#17) balloons past S effort — STOP that item, leave it as-is, note it. C1/C2/#16/sidebar-style (Steps 1-5, 8) are the required core; #9/#17 are the discretionary tail.
- You need to edit a file outside Scope to make a step compile — STOP and report; the scope boundary is deliberate (chat=084, EmptyState=081).
- Any Verify command fails twice in a row after a fix attempt — STOP and report the failing command + output.

## Maintenance notes
- After this lands, `ui.ErrorStrip` / `ui.ErrorStripID` are the ONE error-strip source; future card error paths must call them, never re-emit the raw `<div class="card-note card-note-error">` string. A reviewer should grep for that literal in any new web handler.
- The variadic-attr convention is now uniform across the interactive atoms; new atoms should follow `button.go` (trailing `attrs ...g.Node` on a single-root atom; per-item `Attrs []g.Node` on a list-of-items atom like Tabs).
- **Deferred — #9 tab JS** in `journalcards/journalfocus.go` (if Step 6 skipped): the free/guided tab toggle is still hand-rolled DOM JS in a `data-on:click` string. A future change can move it to a Datastar signal + the new `TabItem.Attrs` once a byte-safe approach is confirmed.
- **Deferred — #17 non-`data-on` raw attrs**: `data-signals:`, `data-class:`, `data-bind:` (in `knowledgefocus.go`, `dayfocus.go`) and `__debounce` stay raw because the v0.3.3 typed helpers render the OBJECT/JSON form, not the key-in-attribute form, and would change pinned markup. Reconcile only if/when those byte pins are intentionally rewritten.
- **Deferred — other dot-imports**: `knowledgefocus.go`, `today.go`, `taskcard.go`, `dayfocus.go`, and any other feature file still dot-importing `gomponents/html` are NOT in this plan; a follow-up can finish the convergence to qualified `h.`.
- A reviewer should scrutinize: (1) that the ErrorStrip rerouting produced byte-identical markup (the web byte-pin tests are the proof), and (2) that `--sb-nav-dot` resolves in all three palettes (`theme-hearthwood`/`forest`/`dungeon`) × both modes, since the dot colour is supplied per-item as `var(--teal)`-style tokens.

## SPEC reconciliation (drift found while planning — for the orchestrator)
- `cardErrorStrip` call sites: SPEC said `cards.go:103,107,112,122,126,132`; actual at HEAD are `:103,107,112,123,126,132` (one line shifted: 122→123).
- `knowledge.go` error helper: SPEC said "~:230-234"; actual `cardError` is `knowledge.go:230-235` (called at :135,172,213).
- `data.Get`: SPEC referenced `data.On/data.Get`; the v0.3.3 module has NO `data.Get`/`data.Post` — only `data.On(event, expr, mods…)`. `@get`/`@post` are expression strings.
- `knowledgefocus.go` "hand-rolled JS tabs" (#9, SPEC ~:105-118): those lines are real Datastar (`data-class:k-tab-active` / `data-on:click__prevent`), NOT DOM JS — and use the key-in-attribute `data-class:` form the typed helper does not reproduce. Only `journalfocus.go:99-106` is genuine hand-rolled DOM JS.
- `.sb-nav-dot` CSS: already exists at `basm.css:2666` (no background); plan ADDS `background: var(--sb-nav-dot)` to it rather than creating a new rule. Sidebar dot is pinned in `internal/ui/shell/sidebar_test.go:64` as `style="background:var(--teal)"` — that test line must be updated.
