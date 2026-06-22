# Plan 113: The recap telescope (bands + expandable cards) renders via gomponents instead of `recap-bands.html` / `recap-cards.html`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm its expected result before moving on. If a
> "STOP conditions" item occurs, stop and report — do not improvise. When done,
> update this plan's status row in `plans/readme.md` unless a reviewer told you
> they maintain the index.
>
> **Drift check (run first)**: `git diff --stat ea79dae..HEAD -- internal/web/recap.go web/templates/recap-bands.html web/templates/recap-cards.html`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code; on a mismatch, treat it as a
> STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none (independent of 111, 112, 114, 115; all of 111–115 must land before 116/117)
- **Category**: migration / tech-debt
- **Planned at**: commit `0dd2457`, 2026-06-19 — **refreshed 2026-06-22 against `ea79dae`; see "## Refresh" below**

## Refresh (2026-06-22, against `ea79dae`)

Still **valid and unstarted**. Both `ExecuteTemplate` sites live:
`recap-bands.html` → `recap.go:89`, `recap-cards.html` → `recap.go:154`. Excerpts
byte-accurate; only anchors drifted — `recapView` `recap.go:26-34`, bandView/exec
block `recap.go:68-96`, expand block `recap.go:145-156`; `renderNodeHTML`
`panel.go:100`; `commandPaletteNode` `home.go:24` unchanged. **Out-of-scope drift
to ignore:** `recap.go` `messageViews` got a `cards.Get→cardTitleIcon` change +
dropped `internal/cards` import (that's plan-116 territory) — not this plan's
concern. `renderMessages` returns `template.HTML` (recap.go:192) — relevant to 116,
not here. Done-criteria greps hold (the two recap-template greps match exactly
recap.go:89 + :154 today, → 0 after).

## Why this matters

The recap telescope — the lazily-loaded summary bands above the chat history,
each card expandable to its children or day transcript — is the **largest
remaining `html/template` dependency on a live serving path**. Two templates,
`recap-bands.html` and `recap-cards.html`, are still `ExecuteTemplate`'d at
runtime. To finish making `gomponents` Balaur's single UI engine and let plan
117 delete `web/templates/`, these must become gomponents node builders.

There is **no reusable component to swap to**: `internal/ui/recapcard.go`'s
`RecapCard` renders a *different* card (`.recapcard` parchment summary, used only
in the storybook) — it has none of the telescope's `.recap-band` / `.recap-card`
structure, expand buttons, or `/ui/recap/expand` wiring. So this plan **builds
new** node builders that reproduce the two templates exactly. They live in
`internal/web/recap.go` as web-local builders (the markup is web-coupled: it
carries live Datastar `@get` expressions to web routes), mirroring the existing
inline-web-node precedent `commandPaletteNode` (`home.go:24`).

## Current state

- `internal/web/recap.go:27-35` — the per-card payload:
  ```go
  type recapView struct {
  	Type     string
  	Label    string
  	Content  string
  	Start    string // Unix seconds for expand requests (URL-safe)
  	Date     string // day cards: YYYY-MM-DD link to the day page
  	HasChild bool
  	Missing  bool // period in range but not summarised yet
  }
  ```

- `internal/web/recap.go:69-97` — `recapBands` builds a **function-local**
  `bandView` and renders the telescope into `#recap`:
  ```go
  	type bandView struct {
  		Heading string
  		Cards   []recapView
  	}
  	var view []bandView
  	for _, band := range recap.Bands(time.Now().In(loc), oldest) {
  		bv := bandView{Heading: bandHeading(band.Type)}
  		... // append non-Missing cards
  	}
  	var b strings.Builder
  	if err := h.tmpl.ExecuteTemplate(&b, "recap-bands.html", view); err != nil {
  		return e.InternalServerError("rendering recap", err)
  	}
  	sse := datastar.NewSSE(e.Response, e.Request)
  	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("recap"), datastar.WithModeInner())
  	return nil
  ```

- `internal/web/recap.go:139-159` — `recapExpand` non-day branch renders child
  cards into `#recap-children-{type}-{start}`:
  ```go
  	} else {
  		var cards []recapView
  		for _, child := range recap.Children(p) {
  			if rec := recap.Find(h.app, master.Id, child); rec != nil {
  				cards = append(cards, h.recapCard(child, rec))
  			}
  		}
  		if len(cards) == 0 {
  			b.WriteString(`<p class="k-empty">Nothing recorded in this stretch.</p>`)
  		} else if err := h.tmpl.ExecuteTemplate(&b, "recap-cards.html", cards); err != nil {
  			return e.InternalServerError("rendering recap cards", err)
  		}
  	}
  	_ = sse.PatchElements(b.String(), datastar.WithSelectorID(targetID), datastar.WithModeInner())
  ```

- The two templates to port:

  `web/templates/recap-bands.html` — ranges over `[]bandView`, **reversed** (so
  the year band sits highest), then a trailing `stitch`; renders nothing when
  empty:
  ```html
  {{if .}}
  {{range reverse .}}
  <section class="recap-band">
    <h2 class="recap-heading"><span class="recap-rune">◇</span> {{.Heading}}</h2>
    {{template "recap-cards.html" .Cards}}
  </section>
  {{end}}
  <div class="stitch"></div>
  {{end}}
  ```

  `web/templates/recap-cards.html` — one `.recap-card` per `recapView`:
  ```html
  {{range .}}
  <article class="recap-card recap-{{.Type}}">
    <header class="recap-head">
      <span class="recap-label">{{.Label}}</span>
      {{if .HasChild}}
      <button class="recap-expand" type="button"
              data-on:click="el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type={{.Type}}&start={{.Start}}')">
        open
      </button>
      {{else}}
      {{if .Date}}<a class="recap-daylink" href="/ui/show/day?date={{.Date}}">visit</a>{{end}}
      <button class="recap-expand" type="button"
              data-on:click="el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type=day&start={{.Start}}')">
        transcript
      </button>
      {{end}}
    </header>
    <p class="recap-body">{{.Content}}</p>
    <div class="recap-children" id="recap-children-{{.Type}}-{{.Start}}"></div>
  </article>
  {{end}}
  ```

- **Conventions to match**:
  - Inline web node builder precedent: `home.go:24` `commandPaletteNode() g.Node`.
  - gomponents imports: `g "maragu.dev/gomponents"` (already imported in
    `recap.go`) and `h "maragu.dev/gomponents/html"` (add it).
  - For the Datastar `data-on:click` expression, emit the attribute **verbatim**
    with `g.Attr("data-on:click", expr)` — the expression contains inline JS
    (`el.closest(...).classList.add('recap-open');`) plus a Datastar `@get`, so a
    literal attribute is the faithful port (do not try to rebuild it with the
    `data.On` helper — keep the exact string).
  - Render-to-string for SSE: `renderNodeHTML(n)` (`panel.go:94`).

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
- `internal/web/recap.go`
- `internal/web/recap_test.go` (create if absent / extend; see Test plan)

**Out of scope** (do NOT touch):
- `web/templates/recap-bands.html`, `recap-cards.html` — plan 117 deletes them.
- `internal/ui/recapcard.go` — a different (storybook-only) card; do not reuse
  or modify it.
- `renderMessages` / `chatBodyHTML` / `messageView.CardBody` and the
  `html/template` import in `recap.go` — those are the `template.HTML` bridge,
  removed in plan 116. **This plan leaves the `html/template` import in place.**

## Git workflow

- Branch: `improve/113-recap-telescope-gomponents`.
- One commit; conventional message, e.g.
  `refactor(web): render recap telescope via gomponents (plan 113)`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Lift `bandView` to package scope

Move the `bandView` type out of `recapBands` to package level in `recap.go`
(just below the `recapView` type):
```go
// bandView is one telescope band (a heading over a row of recap cards).
type bandView struct {
	Heading string
	Cards   []recapView
}
```
Delete the now-duplicate local declaration inside `recapBands`.

**Verify**: `go build ./internal/web/` → exit 0.

### Step 2: Add the two node builders

Add to `recap.go` (faithful ports — preserve the reverse order, the trailing
`stitch`, the empty-input no-op, and the exact `data-on:click` strings):
```go
// recapBandsNode renders the telescope bands into #recap (port of
// recap-bands.html). Bands render in reverse (the year band sits highest);
// empty input renders nothing, matching the template's {{if .}} guard.
func recapBandsNode(view []bandView) g.Node {
	if len(view) == 0 {
		return g.Text("")
	}
	bands := make([]g.Node, 0, len(view)+1)
	for i := len(view) - 1; i >= 0; i-- {
		b := view[i]
		bands = append(bands, h.Section(h.Class("recap-band"),
			h.H2(h.Class("recap-heading"),
				h.Span(h.Class("recap-rune"), g.Text("◇")), g.Text(" "+b.Heading)),
			recapCardsNode(b.Cards),
		))
	}
	bands = append(bands, h.Div(h.Class("stitch")))
	return g.Group(bands)
}

// recapCardsNode renders a row of expandable recap cards (port of recap-cards.html).
func recapCardsNode(cards []recapView) g.Node {
	items := make([]g.Node, 0, len(cards))
	for _, c := range cards {
		var controls []g.Node
		controls = append(controls, h.Span(h.Class("recap-label"), g.Text(c.Label)))
		if c.HasChild {
			expr := "el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type=" + c.Type + "&start=" + c.Start + "')"
			controls = append(controls, h.Button(h.Class("recap-expand"), h.Type("button"),
				g.Attr("data-on:click", expr), g.Text("open")))
		} else {
			if c.Date != "" {
				controls = append(controls, h.A(h.Class("recap-daylink"),
					h.Href("/ui/show/day?date="+c.Date), g.Text("visit")))
			}
			expr := "el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type=day&start=" + c.Start + "')"
			controls = append(controls, h.Button(h.Class("recap-expand"), h.Type("button"),
				g.Attr("data-on:click", expr), g.Text("transcript")))
		}
		items = append(items, h.Article(h.Class("recap-card recap-"+c.Type),
			h.Header(h.Class("recap-head"), g.Group(controls)),
			h.P(h.Class("recap-body"), g.Text(c.Content)),
			h.Div(h.Class("recap-children"), h.ID("recap-children-"+c.Type+"-"+c.Start)),
		))
	}
	return g.Group(items)
}
```
Add `h "maragu.dev/gomponents/html"` to the import block.

**Verify**: `go build ./internal/web/` → exit 0.

### Step 3: Repoint the two call sites

In `recapBands` (Step 1's function), replace the `ExecuteTemplate` block:
```go
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(recapBandsNode(view)),
		datastar.WithSelectorID("recap"), datastar.WithModeInner())
	return nil
```
(Delete the `var b strings.Builder` + `ExecuteTemplate` lines it replaces. Check
whether `strings` is still used elsewhere in `recap.go` before removing the
import — it is used by other functions, so keep it.)

In `recapExpand`, replace the non-day `else` branch:
```go
	} else {
		var cards []recapView
		for _, child := range recap.Children(p) {
			if rec := recap.Find(h.app, master.Id, child); rec != nil {
				cards = append(cards, h.recapCard(child, rec))
			}
		}
		if len(cards) == 0 {
			b.WriteString(`<p class="k-empty">Nothing recorded in this stretch.</p>`)
		} else {
			b.WriteString(renderNodeHTML(recapCardsNode(cards)))
		}
	}
```
(The day branch above it still uses `b.WriteString(string(h.renderMessages(...)))`
— leave it; that is the `template.HTML` bridge, removed in plan 116. `b` stays.)

**Verify**: `go build ./internal/web/` → exit 0.

### Step 4: Build, vet, test

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass, exit 0
- `gofmt -l internal/` → empty

## Test plan

- Add `internal/web/recap_test.go` (or extend an existing test) with table cases
  that render `recapBandsNode` / `recapCardsNode` to a string via `renderNodeHTML`
  and assert the structural invariants:
  - empty `[]bandView` → empty string (no `recap-band`, no `stitch`).
  - a band with cards → contains `class="recap-band"`, `class="recap-heading"`,
    the `◇` rune, the heading text, and exactly one trailing `class="stitch"`.
  - reverse order: given two bands `[{Heading:"A"},{Heading:"B"}]`, "B" appears
    **before** "A" in the output.
  - a `HasChild:true` card → an `open` button whose `data-on:click` contains
    `@get('/ui/recap/expand?type=<type>&start=<start>')` and a
    `recap-children-<type>-<start>` child div.
  - a `HasChild:false` card with `Date` set → a `recap-daylink` to
    `/ui/show/day?date=<date>` and a `transcript` button targeting `type=day`.
  - Model the test file after any existing `internal/web/*_test.go` that builds a
    `*handlers` and asserts rendered markup (e.g. `home_test.go`).
- `templates_test.go` still parses `recap-bands.html`/`recap-cards.html` (files
  present) — leave it; plan 117 removes it.
- Verification: `go test ./internal/web/...` → all pass, including the new cases.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0; new recap node tests exist and pass
- [ ] `gofmt -l internal/` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `grep -rn 'recap-bands.html\|recap-cards.html' internal/web/` returns **no** matches
- [ ] `grep -rn 'ExecuteTemplate' internal/web/recap.go` returns **no** matches
- [ ] `web/templates/recap-bands.html` and `recap-cards.html` still exist (untouched)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:

- The "Current state" excerpts don't match the live code (drift since `0dd2457`).
- `recapView` / the band shape no longer has the fields above.
- `recapExpand`'s day branch turns out NOT to use `renderMessages` (you'd be
  removing `b`/`strings` it still needs).
- You cannot reproduce the `data-on:click` expression verbatim — the
  `recap-open` class toggle is load-bearing for the open animation.

## Maintenance notes

- After this lands, `recap-bands.html` / `recap-cards.html` are dead and removed
  in plan 117.
- The telescope builders are web-local because their markup embeds live `@get`
  routes. If the telescope ever needs a storybook entry, promote `recapCardsNode`
  to an `internal/ui` component taking a prop struct (deferred — not needed for
  the migration).
- Reviewer: verify the `#recap-children-{type}-{start}` id format is unchanged —
  `recapExpand` targets it with `WithSelectorID(targetID)` built from the same
  `type`+`start`, so a format change would break expansion.
