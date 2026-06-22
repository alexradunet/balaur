# Plan 112: The `/ui/cards` palette renders via a gomponents node builder instead of the `ucard_palette` template

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> "STOP conditions" item occurs, stop and report — do not improvise. When done,
> update this plan's status row in `plans/readme.md` unless a reviewer told you
> they maintain the index.
>
> **Drift check (run first)**: `git diff --stat ea79dae..HEAD -- internal/web/cards.go internal/cards web/templates/cards.html`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (independent of 111, 113–115; all of 111–115 must land before 116/117)
- **Category**: migration / tech-debt
- **Planned at**: commit `0dd2457`, 2026-06-19 — **refreshed 2026-06-22 against `ea79dae`; see "## Refresh" below**

## Refresh (2026-06-22, against `ea79dae`)

Still **valid and unstarted**, near-zero drift. `ucard_palette` is live at
`internal/web/cards.go:46`; the handler block (cards.go:42-50) and the `cards.html`
excerpt are byte-identical at HEAD. Minor anchors: the `ui.Tag` precedent is now
wrapped `g.If(v.RecurLine != "", ui.Tag(...))` at `taskcards/taskcard.go:30`;
render-to-writer precedent → `knowledge.go:179`
(`ui.ErrorStrip("could not load this card").Render(...)`, +1 from plan 134);
`commandPaletteNode` precedent `home.go:24` unchanged. **Test-plan correction:**
an end-to-end test already exists — `TestUiCardPalette` (`cards_test.go:60`,
`GET /ui/cards` asserting `ucard-palette` + all card types) covers
`cardPaletteNode` after the repoint with **no test edit** (just rerun it); replace
the "add one if none exists" guidance accordingly. Done-criteria greps hold
(`ucard_palette` in `internal/web/` = 1 today at cards.go:46, must be 0 after).

## Why this matters

The migration to `gomponents` as Balaur's single UI engine leaves one card
template behind: `ucard_palette`, the human/agent index of all registered card
types served at `GET /ui/cards`. It is the last `{{define}}` in
`web/templates/cards.html` and the only thing keeping that file alive. Porting
it to a small gomponents node builder removes another `ExecuteTemplate` caller
and lets plan 117 delete `cards.html`. The palette is a server-internal registry
listing (not a reusable Hearthwood design atom), so it follows the existing
precedent of inline web node builders like `commandPaletteNode` (`home.go:24`)
rather than a storybook component.

## Current state

- `internal/web/cards.go:42-50` serves the palette via the template:
  ```go
  // uiCardPalette handles GET /ui/cards — the palette listing all card specs.
  func (h *handlers) uiCardPalette(e *core.RequestEvent) error {
  	specs := cards.All()
  	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
  	if err := h.tmpl.ExecuteTemplate(e.Response, "ucard_palette", specs); err != nil {
  		return e.InternalServerError("rendering card palette", err)
  	}
  	return nil
  }
  ```
  `cards.All()` returns `[]cards.Spec` (package `internal/cards`).

- The template to port, `web/templates/cards.html` (the whole file is one
  `{{define "ucard_palette"}}` over `[]cards.Spec`):
  ```html
  <section class="k-section ucard-palette">
    <h2 class="k-heading">Card palette</h2>
    <ul class="ucard-list">
      {{range .}}
      <li class="ucard-row">
        <span class="ucard-title">
          <img class="tool-icon" src="/static/icons/{{.Icon}}.png" alt="">
          <code>{{.Type}}</code> — {{.Label}}
        </span>
        <span class="kcard-meta">w={{.W}}</span>
        {{if .Params}}
        <ul class="ucard-params">
          {{range .Params}}
          <li>
            <code>{{.Name}}</code>{{if .Required}} <span class="tag">required</span>{{end}}
            {{with .Enum}} [{{range $i, $e := .}}{{if $i}}, {{end}}{{$e}}{{end}}]{{end}}
            — {{.Doc}}
          </li>
          {{end}}
        </ul>
        {{end}}
      </li>
      {{end}}
    </ul>
  </section>
  ```

- **Spec fields used** (confirm exact names/types by reading `internal/cards` —
  the type behind `cards.All()` and the element type of `Spec.Params`): `Icon`
  (string), `Type` (string), `Label` (string), `W` (int), `Params` (slice). Each
  param has `Name` (string), `Required` (bool), `Enum` ([]string), `Doc` (string).

- **Conventions to match**:
  - Inline web node builder precedent: `internal/web/home.go:24-39`
    `commandPaletteNode()` returns `g.Node` from `internal/web`. Follow that shape.
  - gomponents imports used across the package: `g "maragu.dev/gomponents"` and
    `h "maragu.dev/gomponents/html"` (see `internal/web/panel.go`). `cards.go`
    already imports `g`; add `h` if you use it.
  - The `tag` span is the `ui.Tag` atom — `internal/feature/taskcards/taskcard.go:30`
    uses `ui.Tag(g.Text(v.RecurLine))` to emit `<span class="tag">…</span>`.
    `cards.go` already imports `internal/ui`.
  - Render-to-writer: components render straight to `e.Response` via `.Render(w)`
    (see `internal/web/knowledge.go:178` `ui.ErrorStrip(...).Render(e.Response)`).

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
- `internal/web/cards.go`
- `internal/web/cards_test.go` (only if an assertion drifts; see Test plan)

**Out of scope** (do NOT touch):
- `web/templates/cards.html` — plan 117 deletes the whole `web/templates/` dir.
- `internal/cards` — read-only here; you only consume `cards.Spec`.
- Any other `ExecuteTemplate` caller — separate plans.

## Git workflow

- Branch: `improve/112-card-palette-gomponents`.
- One commit; conventional message, e.g.
  `refactor(web): render /ui/cards palette via gomponents (plan 112)`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Read `internal/cards` and confirm the Spec/Param field names

Open `internal/cards` and find the type returned by `All()` and the element type
of its `Params` field. Confirm the fields named in "Current state" exist with
those types. If `Enum` is not `[]string` or `Params`'s element type differs,
adapt the node builder in Step 2 accordingly (and note it in your report).

**Verify**: you can name the exact struct + field types.

### Step 2: Add `cardPaletteNode` and repoint `uiCardPalette`

In `internal/web/cards.go`, add a node builder that reproduces the template
exactly (preserve the `[a, b]` enum join with its leading space, and the
`— {{.Doc}}` em-dash). Target shape:
```go
// cardPaletteNode renders the GET /ui/cards palette: the human/agent index of
// every registered card spec. Port of ucard_palette (web/templates/cards.html).
func cardPaletteNode(specs []cards.Spec) g.Node {
	rows := make([]g.Node, 0, len(specs))
	for _, s := range specs {
		var params g.Node = g.Text("")
		if len(s.Params) > 0 {
			items := make([]g.Node, 0, len(s.Params))
			for _, p := range s.Params {
				var req g.Node = g.Text("")
				if p.Required {
					req = g.Group([]g.Node{g.Text(" "), ui.Tag(g.Text("required"))})
				}
				enum := ""
				if len(p.Enum) > 0 {
					enum = " [" + strings.Join(p.Enum, ", ") + "]"
				}
				items = append(items, h.Li(
					h.Code(g.Text(p.Name)), req,
					g.Text(enum+" — "+p.Doc),
				))
			}
			params = h.Ul(h.Class("ucard-params"), g.Group(items))
		}
		rows = append(rows, h.Li(h.Class("ucard-row"),
			h.Span(h.Class("ucard-title"),
				h.Img(h.Class("tool-icon"), h.Src("/static/icons/"+s.Icon+".png"), h.Alt("")),
				h.Code(g.Text(s.Type)), g.Text(" — "+s.Label),
			),
			h.Span(h.Class("kcard-meta"), g.Text(fmt.Sprintf("w=%d", s.W))),
			params,
		))
	}
	return h.Section(h.Class("k-section ucard-palette"),
		h.H2(h.Class("k-heading"), g.Text("Card palette")),
		h.Ul(h.Class("ucard-list"), g.Group(rows)),
	)
}
```
Then repoint the handler:
```go
func (h *handlers) uiCardPalette(e *core.RequestEvent) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return cardPaletteNode(cards.All()).Render(e.Response)
}
```
Add imports as needed: `h "maragu.dev/gomponents/html"`, `"strings"`, `"fmt"`
(check which are already present in `cards.go` before adding — `fmt` and
`strings` already are). Do not duplicate an existing import.

**Verify**: `go build ./internal/web/` → exit 0.

### Step 3: Build, vet, test

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass, exit 0
- `gofmt -l internal/` → empty

## Test plan

- If a test in `internal/web/cards_test.go` (or `templates_test.go`) asserts the
  palette markup, point it at `cardPaletteNode(...)` rendered to a string (use
  `renderNodeHTML` from `panel.go`) instead of the template, and keep the same
  class/content assertions (`ucard-palette`, `ucard-row`, `w=`, the `required`
  tag, an enum `[...]`). If no such test exists, add a small one in
  `cards_test.go` rendering `cardPaletteNode(cards.All())` and asserting it
  contains `class="ucard-palette"` and at least one `class="ucard-row"` (model
  it after any existing `cards_test.go` render assertion).
- `templates_test.go` still parses `cards.html` (the template file is still
  present) — leave that test alone; plan 117 removes it with the file.
- Verification: `go test ./internal/web/...` → all pass.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l internal/` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `grep -rn 'ucard_palette' internal/web/` returns **no** matches (the name lives only in `web/templates/cards.html`)
- [ ] `grep -rn 'ExecuteTemplate' internal/web/cards.go` returns **no** matches
- [ ] `web/templates/cards.html` still exists (untouched)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:

- The "Current state" excerpts don't match the live code (drift since `0dd2457`).
- `cards.Spec` / its param type lacks the fields above, or `Enum` is not a
  string slice — the join logic must be re-derived; report what you found.
- The palette is consumed by something other than `uiCardPalette` (grep
  `ucard_palette` and `cardPaletteNode` to be sure).

## Maintenance notes

- After this lands, `web/templates/cards.html` is fully dead (no `{{define}}`
  is executed) and is deleted in plan 117.
- The palette is intentionally a web-local node builder (not a storybook
  component) — it lists registry internals, mirroring `commandPaletteNode`.
- Reviewer: eyeball `GET /ui/cards` markup parity (classes `ucard-palette`,
  `ucard-row`, `ucard-title`, `ucard-params`, `kcard-meta`, the `tag`/`code`
  structure) against the template excerpt above.
