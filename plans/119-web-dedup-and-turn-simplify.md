# Plan 119: Collapse two web-layer duplications and one misleading signature

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat ce2ba72..HEAD -- internal/web/cards.go internal/web/panel.go internal/web/recap.go internal/turn/models.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Why this matters

Three small, safe simplifications surfaced by a cleanup audit. Each is
output-preserving (the rendered HTML and model labels do not change), so the
existing test suite is the regression guard. The goal is fewer places to change
when the card-rendering or model-labeling logic evolves — KISS/suckless, "one
source of truth per concern" (`AGENTS.md`). None of these is a bug; this is
debt reduction only.

## Current state

### Duplication 1 — `cardHTML` and `cardFocusHTML` share validate+error logic

`internal/web/cards.go:104-118` and `:145-159` are near-identical: both look up
the card type, validate params, render into a `strings.Builder`, and return the
`cardErrorStrip` on any failure. The ONLY differences are the render size and
the log message.

```go
// cards.go:104-118
func (h *handlers) cardHTML(typ string, params map[string]string) template.HTML {
	if _, ok := cards.Get(typ); !ok {
		return cardErrorStrip("no such card type: " + typ)
	}
	cleaned, err := cards.Validate(typ, params)
	if err != nil {
		return cardErrorStrip(err.Error())
	}
	var b strings.Builder
	if err := h.cardInto(&b, typ, cleaned); err != nil {           // ui.Tile
		h.app.Logger().Warn("board card render failed", "type", typ, "err", err)
		return cardErrorStrip("could not render this card")
	}
	return template.HTML(b.String())
}
```
```go
// cards.go:145-159
func (h *handlers) cardFocusHTML(typ string, params map[string]string) template.HTML {
	if _, ok := cards.Get(typ); !ok {
		return cardErrorStrip("no such card type: " + typ)
	}
	cleaned, err := cards.Validate(typ, params)
	if err != nil {
		return cardErrorStrip(err.Error())
	}
	var b strings.Builder
	if err := h.cardSizeInto(&b, typ, cleaned, ui.Focus); err != nil {  // ui.Focus
		h.app.Logger().Warn("focus card render failed", "type", typ, "err", err)
		return cardErrorStrip("could not render this card")
	}
	return template.HTML(b.String())
}
```
Note `cardInto(w, typ, params)` is exactly `cardSizeInto(w, typ, params, ui.Tile)`
(see `cards.go:76-78`), so both branches are `cardSizeInto` differing only by the
`ui.CardSize` argument.

### Duplication 2 — title/icon extraction from `cards.Get` repeated 3×

The pattern "look up the card spec; use its Label/Icon, else fall back to the
type name and empty icon" appears three times:

```go
// panel.go:74-78  (panelNode)
title, icon := typ, ""
if spec, ok := cards.Get(typ); ok {
	title, icon = spec.Label, spec.Icon
}
```
```go
// panel.go:101-105  (chipNode) — identical
title, icon := typ, ""
if spec, ok := cards.Get(typ); ok {
	title, icon = spec.Label, spec.Icon
}
```
```go
// recap.go:281-287  (messageViews) — equivalent (miss → title=typ, icon stays "")
if spec, ok := cards.Get(typ); ok {
	mv.ArtifactTitle, mv.ArtifactIcon = spec.Label, spec.Icon
} else {
	mv.ArtifactTitle = typ
}
```

### Misleading signature — `modelBadge` takes a param it never reads

`internal/turn/models.go:148-150`:
```go
func modelBadge(_ store.LLMConfig) string {
	return "local"
}
```
Called once, at `models.go:120`: `Badge: modelBadge(cfg)`. The blank-identifier
parameter advertises configurability that does not exist — every model gets the
same badge.

Repo conventions to follow: small helpers live next to their callers in the same
package (see `cards.go`'s `cardErrorStrip`/`queryToMap`); error strips use
`cardErrorStrip` / `ui.ErrorStrip`. Match the surrounding style.

## Commands you will need

| Purpose   | Command                                  | Expected on success |
|-----------|------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`           | exit 0              |
| Vet       | `go vet ./...`                           | exit 0              |
| Format    | `gofmt -l internal/web/cards.go internal/web/panel.go internal/web/recap.go internal/turn/models.go` | empty |
| Tests (web) | `go test ./internal/web/...`           | `ok`                |
| Tests (turn)| `go test ./internal/turn/...`          | `ok`                |
| Full tests | `go test ./...`                         | all `ok`            |
| Diff hygiene | `git diff --check`                    | no output           |

(In a TLS-intercepting sandbox, Go commands may need a GOPROXY shim; GOSUMDB
stays on.)

## Scope

**In scope**:
- `internal/web/cards.go`
- `internal/web/panel.go`
- `internal/web/recap.go`
- `internal/turn/models.go`

**Out of scope** (do NOT touch):
- `internal/web/knowledge.go` `cardTemplateName` / `renderCard` / `renderCardHTML`
  — that duplication lives inside the `html/template` code that plan 111 deletes;
  do not refactor it here.
- `internal/web/cards.go:46` `uiCardPalette` (`ExecuteTemplate "ucard_palette"`)
  — owned by plan 112. Do not touch it (it is in the same file but a different
  function).
- `modelDetail` (`models.go:144-146`) and the `availableChoices` Detail/Badge
  "compute then overwrite" block — the initial values ARE used on the happy path
  (a valid local file), so leave them; only `modelBadge`'s unused param changes.

## Git workflow

- Land on `main`; if dispatched, base off `origin/main`. Conventional-commit
  subjects (`refactor(web): …`, `refactor(turn): …`). Commit/push only when the
  operator instructs.
- `cards.go` is also edited by the (TODO) plan 112, in a different function. If
  both are in flight, expect a trivial merge; do not reorder either.

## Steps

### Step 1: Extract a shared `cardHTMLAt` helper (Duplication 1)

In `internal/web/cards.go`, add one helper that both functions delegate to.
**Name it `cardHTMLAt` — NOT `renderCardHTML`** (`renderCardHTML` already exists
in `knowledge.go`). Target shape:

```go
// cardHTMLAt server-renders one card at the given size to HTML for inline
// embedding, with validate + error-strip discipline. cardHTML/cardFocusHTML are
// thin wrappers choosing the size and the log context.
func (h *handlers) cardHTMLAt(typ string, params map[string]string, size ui.CardSize, logMsg string) template.HTML {
	if _, ok := cards.Get(typ); !ok {
		return cardErrorStrip("no such card type: " + typ)
	}
	cleaned, err := cards.Validate(typ, params)
	if err != nil {
		return cardErrorStrip(err.Error())
	}
	var b strings.Builder
	if err := h.cardSizeInto(&b, typ, cleaned, size); err != nil {
		h.app.Logger().Warn(logMsg, "type", typ, "err", err)
		return cardErrorStrip("could not render this card")
	}
	return template.HTML(b.String())
}
```
Then collapse the two callers (keep their doc comments):
```go
func (h *handlers) cardHTML(typ string, params map[string]string) template.HTML {
	return h.cardHTMLAt(typ, params, ui.Tile, "board card render failed")
}

func (h *handlers) cardFocusHTML(typ string, params map[string]string) template.HTML {
	return h.cardHTMLAt(typ, params, ui.Focus, "focus card render failed")
}
```
(`ui.Tile` is what `cardInto` passes, so `cardHTML` behavior is unchanged.)

**Verify**: `go build ./internal/web/` exit 0; `go test ./internal/web/...` `ok`.

### Step 2: Extract `cardTitleIcon` and use it at all three sites (Duplication 2)

In `internal/web/panel.go` (near `showURL`), add:
```go
// cardTitleIcon returns the display title and icon for a card type, falling
// back to the raw type name and no icon when the type is unregistered.
func cardTitleIcon(typ string) (title, icon string) {
	if spec, ok := cards.Get(typ); ok {
		return spec.Label, spec.Icon
	}
	return typ, ""
}
```
Replace the bodies:
- `panel.go` `panelNode` (lines 75-78): `title, icon := cardTitleIcon(typ)`
- `panel.go` `chipNode` (lines 102-105): `title, icon := cardTitleIcon(typ)`
- `recap.go` (lines 281-286): replace the `if/else` with
  `mv.ArtifactTitle, mv.ArtifactIcon = cardTitleIcon(typ)`
  (equivalent: on a miss the helper returns `(typ, "")`, matching the old
  else-branch which set title=typ and left icon as its zero value `""`).

**Verify**: `go build ./internal/web/` exit 0; `go test ./internal/web/...` `ok`.
Confirm `recap.go` still imports `cards` (it does — used elsewhere) and that
removing the inline `if spec, ok := cards.Get` did not orphan an import
(`go vet ./internal/web/`).

### Step 3: Drop the unused parameter on `modelBadge`

In `internal/turn/models.go`:
- Change `func modelBadge(_ store.LLMConfig) string {` to `func modelBadge() string {`.
- Change the call at `models.go:120` from `Badge: modelBadge(cfg),` to
  `Badge: modelBadge(),`.

**Verify**: `go build ./internal/turn/` exit 0; `go vet ./internal/turn/` exit 0
(this proves `store` is still imported/used in `models.go` — it is, by
`modelDetail` and `ClientFor`).

### Step 4: Full build, vet, format, test

Run the build, vet, gofmt, and full `go test ./...`.

**Verify**: all green. Because every change is output-preserving, no existing
test should change behavior.

## Test plan

No new tests required — all three changes are refactors that preserve output,
and the change is guarded by the existing `internal/web` and `internal/turn`
suites (which render cards and build model choices). Do NOT weaken or delete any
existing test to make this pass. Confirm `go test ./...` is fully green.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n "func (h \*handlers) cardHTMLAt" internal/web/cards.go` → matches
- [ ] `grep -n "func cardTitleIcon" internal/web/panel.go` → matches
- [ ] `grep -c "if spec, ok := cards.Get(typ)" internal/web/panel.go` → `0`
- [ ] `grep -n "func modelBadge() string" internal/turn/models.go` → matches; `grep -n "modelBadge(cfg)" internal/turn/models.go` → empty
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` all `ok`
- [ ] `gofmt -l` of the four files → empty
- [ ] `git diff --check` → no output
- [ ] Only the in-scope files are modified

## STOP conditions

Stop and report back if:

- The excerpts don't match the live code (drift since `ce2ba72`).
- Any existing test fails after a change — it means the refactor was NOT
  output-preserving; do not "fix" the test, report the diff instead.
- `go vet` flags an orphaned import after Step 2 or Step 3 that you cannot
  resolve by removing only the now-unused import.
- A name collision appears for `cardHTMLAt` or `cardTitleIcon` (a symbol with
  that name already exists) — pick a clearly distinct name and note it.

## Maintenance notes

- If plan 111 (knowledge card gomponents) lands first, `renderCard`/
  `renderCardHTML`/`cardTemplateName` in `knowledge.go` change or disappear;
  that is independent of `cardHTMLAt` here (different file/functions).
- A reviewer should confirm the rendered card HTML is byte-identical (the web
  tests assert card ids/content) and that no model badge/label text changed.
