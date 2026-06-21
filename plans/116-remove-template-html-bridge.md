# Plan 116: Remove the `template.HTML` bridge type — helpers and view-models carry `g.Node` / `string`, dropping `html/template` from five web files

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm its expected result before moving on. If a
> "STOP conditions" item occurs, stop and report — do not improvise. When done,
> update this plan's status row in `plans/readme.md` unless a reviewer told you
> they maintain the index.
>
> **Drift check (run first)**: `git diff --stat 0dd2457..HEAD -- internal/web/cards.go internal/web/recap.go internal/web/home.go internal/web/models.go internal/web/chatstream.go internal/web/panel.go internal/web/tasks.go internal/web/web.go`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code; on a mismatch, treat it as a STOP
> condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: 111, 112, 113, 114, 115 (do this after all `ExecuteTemplate`
  callers are gomponents — so the only remaining `html/template` use is the
  `template.HTML` bridge type)
- **Category**: migration / tech-debt
- **Planned at**: commit `0dd2457`, 2026-06-19

## Why this matters

`template.HTML` (the `html/template` string-wrapper type) is the last way
`html/template` is woven into the web gateway: it is the return type of a dozen
card/chat render helpers and the type of three view-model fields, bridged into
gomponents trees via `g.Raw(string(...))`. To make `gomponents` the single UI
engine and remove `html/template` entirely (the user's goal), these helpers and
fields must carry `g.Node` (when the value flows into another gomponents tree) or
plain `string` (when it is written into an SSE patch). This plan does that
conversion, dropping the `html/template` import from `cards.go`, `recap.go`,
`home.go`, `models.go`, and `chatstream.go`. It also deletes three values that
are already dead (`homeData.ComposerHTML`, the `modelsPageData` struct, and
`renderModelsPanel`).

After this plan only `internal/web/web.go` still imports `html/template` (for the
`funcs` FuncMap + `ParseFS` template engine) — plan 117 removes that.

## Current state

`internal/web/panel.go:94` provides the render-to-string helper used below:
```go
func renderNodeHTML(n g.Node) string { var b strings.Builder; _ = n.Render(&b); return b.String() }
```

**A. `internal/web/cards.go`** (imports `html/template`) — six helpers return
`template.HTML`:
- `cardErrorStrip(msg string) template.HTML` (`:122`): `_ = ui.ErrorStrip(msg).Render(&b); return template.HTML(b.String())`.
- `cardHTML(typ string, params map[string]string) template.HTML` (`:104`): validates, renders `cardInto(&b)`, returns `template.HTML(b.String())`; error paths `return cardErrorStrip(...)`.
- `uicardBody(typ, query string) template.HTML` (`:136`): `return h.cardFocusHTML(typ, queryToMap(vals))`.
- `cardFocusHTML(typ string, params map[string]string) template.HTML` (`:145`): like `cardHTML` but `cardSizeInto(&b, ..., ui.Focus)`.
- `artifactBody(title string, cs []cards.Card) template.HTML` (`:165`):
  ```go
  nodes := make([]g.Node, 0, len(cs))
  for _, c := range cs { nodes = append(nodes, g.Raw(string(h.cardHTML(c.Type, c.Params)))) }
  var b strings.Builder
  _ = chat.Cluster(chat.ClusterProps{Cards: nodes}).Render(&b)
  return template.HTML(b.String())
  ```
- `proposalBody(kind, id string) template.HTML` (`:178`): returns `template.HTML(s)` from `taskCardHTML`/`renderCardHTML` strings, or `""` when the record can't load.

Consumers of those helpers:
- `internal/web/panel.go:79`: `chat.Panel(chat.PanelProps{..., Body: g.Raw(string(h.uicardBody(typ, query)))})`
- `internal/web/panel.go:84`: `chat.Panel(chat.PanelProps{..., Body: g.Raw(string(h.artifactBody(title, cs)))})`
- `internal/web/chatstream.go:252` (`refreshCard`): `_ = s.sse.PatchElements(string(s.h.cardHTML(typ, nil)))`
- `internal/web/chatstream.go:199`: `s.endTool(rest, s.h.proposalBody(kind, id))`
- `internal/web/recap.go:290`: `mv.CardBody, mv.Content = h.proposalBody(kind, id), rest`

**B. `internal/web/recap.go`** (imports `html/template`):
- `messageView.CardBody template.HTML` field (`:170`).
- `:215-216`: `if mv.CardBody != "" { nodes = append(nodes, g.El("div", g.Attr("class", "k-inline"), g.Raw(string(mv.CardBody)))) }`
- `renderMessages(views []messageView) template.HTML` (`:193`): builds `nodes`, `_ = g.Group(nodes).Render(&b); return template.HTML(b.String())`.
- `chatBodyHTML(d homeData) template.HTML` (`:233`): returns `renderMessages(...)` or `g.Group([]g.Node{crest, greeting})` rendered to `template.HTML`.
- Consumers: `:145` `b.WriteString(string(h.renderMessages(...)))`; `tasks.go:306` `sse.PatchElements(string(h.renderMessages(...)), ...)`; `chatBodyHTML` is consumed via `homeData.ChatBodyHTML`.

**C. `internal/web/home.go`** (imports `html/template`):
- `composerHTML(d homeData) template.HTML` (`:56-61`) — **dead**: its only caller
  is `web.go:297` `data.ComposerHTML = composerHTML(data)`, and `ComposerHTML` is
  never read (homePage uses `composerNode(dock)` directly). Remove the func.
- The `html/template` import exists only for `composerHTML`'s return type.

**D. `internal/web/models.go`** (imports `html/template`):
- `homeData.ComposerHTML template.HTML` field (`:44`) — **dead** (see C). Remove.
- `homeData.ChatBodyHTML template.HTML` field (`:45`) — used by `home.go:84`.
- `modelsPageData` struct with `ModelsHTML template.HTML` (`:62-65`) — **dead**:
  the struct is never instantiated anywhere. Remove it.
- `renderModelsPanel(errMsg string) (template.HTML, error)` (`:138-148`) — **dead**:
  no callers (`grep -rn renderModelsPanel internal/` shows only its definition).
  Remove it.
- `template.HTMLEscapeString` (`:453`) inside `installRuntime`'s `sseLogger`:
  ```go
  sseLogger := func(_ context.Context, msg string, _ ...any) {
  	var b strings.Builder
  	b.WriteString(`<div id="runtime-dl-progress">`)
  	b.WriteString(template.HTMLEscapeString(msg))
  	b.WriteString(`</div>`)
  	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("runtime-dl-progress"), datastar.WithModeOuter())
  }
  ```

**E. `internal/web/chatstream.go`** (imports `html/template`):
- `endTool(content string, card template.HTML)` (`:219`): `if card == "" { return }`
  then `... g.Raw(string(card)))`.
- Callers in `handleToolResult`: `endTool(..., "")` at `:195, :203, :213` and
  `endTool(rest, s.h.proposalBody(kind, id))` at `:199`.

**Conventions**: `g.Raw(string)` and `g.Group([]g.Node)` are gomponents (package
`maragu.dev/gomponents`), **not** `html/template` — keep using them where a node
must wrap an already-rendered string. The goal is removing the `html/template`
*package*, not `g.Raw`.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tests | `go test ./...` | all pass, exit 0 |
| Format check | `gofmt -l internal/` | empty output |
| Whitespace | `git diff --check` | no output |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/web/cards.go`, `internal/web/recap.go`, `internal/web/home.go`,
  `internal/web/models.go`, `internal/web/chatstream.go`
- `internal/web/panel.go` (two consumer call sites), `internal/web/tasks.go`
  (one consumer call site), `internal/web/web.go` (the `ComposerHTML` assignment)
- Any `internal/web/*_test.go` whose assertions assign `messageView.CardBody` as
  a string or compare `renderMessages`/`chatBodyHTML` output as a string
  (see Test plan)

**Out of scope** (do NOT touch):
- `internal/web/web.go`'s `funcs` FuncMap, `template.Must`/`ParseFS`, the `tmpl`
  field, and its `html/template` import — those are plan 117. (You only edit the
  `data.ComposerHTML = composerHTML(data)` line in `dockData` here.)
- `web/templates/*.html` — plan 117.
- The gomponents component bodies — unchanged; you only change types/return values.

## Git workflow

- Branch: `improve/116-remove-template-html-bridge`.
- One commit; conventional message, e.g.
  `refactor(web): drop the template.HTML bridge for g.Node (plan 116)`.
- Do NOT push or open a PR unless instructed.

## Steps

Apply the conversion below, then build/vet/test once at the end (intermediate
per-file states may not compile — that's expected while a shared type changes).

### Step 1: `cards.go` — helpers return `g.Node`

| Helper | New signature | Body change |
|--------|---------------|-------------|
| `cardErrorStrip` | `(msg string) g.Node` | `return ui.ErrorStrip(msg)` (drop the builder) |
| `cardHTML` | `(typ string, params map[string]string) g.Node` | error paths `return cardErrorStrip(...)`; success `return g.Raw(b.String())` (keep the `cardInto(&b)` rendering) |
| `cardFocusHTML` | `(typ string, params map[string]string) g.Node` | same shape as `cardHTML` with `cardSizeInto(&b, ..., ui.Focus)` → `return g.Raw(b.String())`; errors `return cardErrorStrip(...)` |
| `uicardBody` | `(typ, query string) g.Node` | unchanged body (`return h.cardFocusHTML(...)`) |
| `artifactBody` | `(title string, cs []cards.Card) g.Node` | `nodes = append(nodes, h.cardHTML(c.Type, c.Params))` (drop `g.Raw(string(...))`); `return chat.Cluster(chat.ClusterProps{Cards: nodes})` (drop the builder) |
| `proposalBody` | `(kind, id string) g.Node` | return `g.Raw(s)` instead of `template.HTML(s)`; the can't-load paths `return nil` |

Remove the `"html/template"` import from `cards.go`.

### Step 2: `panel.go` — use the nodes directly

- `:79` → `Body: h.uicardBody(typ, query)`
- `:84` → `Body: h.artifactBody(title, cs)`

(Drop the `g.Raw(string(...))` wrappers. `panel.go` keeps its other imports.)

### Step 3: `recap.go` — `CardBody`/`renderMessages`/`chatBodyHTML` carry `g.Node`

- `messageView.CardBody` field → `g.Node`.
- `:215-216` → `if mv.CardBody != nil { nodes = append(nodes, g.El("div", g.Attr("class", "k-inline"), mv.CardBody)) }`
- `renderMessages` → returns `g.Node`: `return g.Group(nodes)` (drop the builder).
- `chatBodyHTML` → returns `g.Node`: return `h.renderMessages(d.History)` in the
  history branch, and `g.Group([]g.Node{crest, greeting})` in the greeting branch.
- `:145` (inside `recapExpand` day branch) → `b.WriteString(renderNodeHTML(h.renderMessages(h.messageViews(msgs))))`
- Remove the `"html/template"` import from `recap.go`.

### Step 4: `tasks.go` — `chatNudges` consumer

- `:306` → `_ = sse.PatchElements(renderNodeHTML(h.renderMessages(h.messageViews(recs))), datastar.WithSelectorID("chat"), datastar.WithModeAppend())`

### Step 5: `home.go` + `web.go` — remove the dead `ComposerHTML` path

- Delete `composerHTML` (`home.go:56-61`).
- Change `homeData.ChatBodyHTML` field type to `g.Node` (in `models.go`, Step 6).
- `home.go:84` → `Convo: dock.ChatBodyHTML` (drop `g.Raw(string(...))`).
- `web.go:297` (`dockData`): delete the line `data.ComposerHTML = composerHTML(data)`.
- Remove the `"html/template"` import from `home.go`.

### Step 6: `models.go` — drop dead types + the field + `HTMLEscapeString`

- Delete the `homeData.ComposerHTML template.HTML` field (`:44`).
- Change `homeData.ChatBodyHTML` field type to `g.Node` (`:45`).
- Delete the `modelsPageData` struct (`:62-65`).
- Delete `renderModelsPanel` (`:138-148`).
- Replace the `sseLogger` body (`:450-456`) with:
  ```go
  sseLogger := func(_ context.Context, msg string, _ ...any) {
  	node := h.Div(h.ID("runtime-dl-progress"), g.Text(msg))
  	_ = sse.PatchElements(renderNodeHTML(node), datastar.WithSelectorID("runtime-dl-progress"), datastar.WithModeOuter())
  }
  ```
- Update `models.go` imports: remove `"html/template"`; add
  `g "maragu.dev/gomponents"` and `h "maragu.dev/gomponents/html"` (needed for the
  `sseLogger` node and the `ChatBodyHTML g.Node` field type).

### Step 7: `chatstream.go` — `endTool` takes `g.Node`

- `endTool(content string, card g.Node)`; body `if card == nil { return }` and the
  inline-card append uses `card` directly (drop `g.Raw(string(card))`):
  ```go
  s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), card))
  ```
- Update its callers: `endTool("choices offered", nil)`, `endTool(clipText(rest, 2000), nil)`,
  `endTool(clipText(ev.Text, 2000), nil)` (the three `""` → `nil`); the
  `endTool(rest, s.h.proposalBody(kind, id))` caller is fine (`proposalBody` now
  returns `g.Node`).
- Remove the `"html/template"` import from `chatstream.go`.

### Step 8: Build, vet, test

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass, exit 0
- `gofmt -l internal/` → empty
- `grep -rn 'template\.HTML\|html/template' internal/web/cards.go internal/web/recap.go internal/web/home.go internal/web/models.go internal/web/chatstream.go` → **no** matches

## Test plan

- Build is the primary gate: every consumer is enumerated above, so a clean
  `go build ./...` proves the type change is complete.
- `grep -rn 'CardBody:' internal/web/*_test.go` — if a test assigns
  `messageView{CardBody: "<html>"}` (a string), change it to a node:
  `CardBody: g.Raw("<html>")`. If a test compares `renderMessages(...)` /
  `chatBodyHTML(...)` as a string, wrap with `renderNodeHTML(...)`.
- No new tests are required (behavior is unchanged: the same HTML reaches the
  browser; only the in-Go carrier type changes). The existing handler tests that
  exercise history rendering / proposal embeds are the regression guard.
- Verification: `go test ./...` → all pass.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l internal/` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `grep -rn 'html/template' internal/web/` returns matches **only** in `web.go` and `templates_test.go` (everything else is converted; plan 117 removes those last two)
- [ ] `grep -rn 'template.HTML' internal/web/` (excluding `web.go`/`templates_test.go`) returns **no** matches
- [ ] `grep -rn 'ComposerHTML\|modelsPageData\|renderModelsPanel' internal/web/` returns **no** matches (dead code removed)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:

- The "Current state" excerpts don't match the live code (drift since `0dd2457`),
  or plans 111–115 are not all DONE (an `ExecuteTemplate` caller still exists,
  meaning `template.HTML` is not the *only* remaining `html/template` use — your
  scope assumption is wrong).
- `ComposerHTML` / `modelsPageData` / `renderModelsPanel` turn out to have a real
  caller you didn't expect — do NOT delete them; report and convert instead.
- A test fails on a *content* difference (different HTML reaches the browser) —
  that means a `g.Raw`/`g.Group` conversion dropped or double-escaped content;
  fix the conversion, don't change the test's expected HTML.

## Maintenance notes

- After this lands, `html/template` survives in exactly two places —
  `internal/web/web.go` (the engine machinery) and
  `internal/web/templates_test.go` — both removed by plan 117.
- The card render helpers now return `g.Node`; future callers compose them
  directly into gomponents trees (no more `g.Raw(string(...))` round-trips
  except where a pre-rendered feature card string is wrapped).
- Reviewer: confirm no double-escaping crept in — `g.Raw(s)` emits `s` verbatim
  (correct for already-rendered card HTML); `g.Text(s)` would escape it (wrong
  for HTML). The `sseLogger` is the one place that *should* escape (`g.Text`),
  matching the old `HTMLEscapeString`.
