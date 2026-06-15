# Plan 071: collapse the hand-copied card-frame header into one ui.CardShell helper

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 1f8f55e..HEAD -- internal/ui/ internal/feature/`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: LOW–MED (touches all 6 feature card packages; behavior-preserving)
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

The gomponents migration's stated goal was that `internal/ui` dedupes shared
rendering. It delivered that for sparklines and text (`ui.ErrorStrip`,
`ui.Clip`, `ui.SparkPoints`, `ui.NumericValues`) — but **not** for the single
most-repeated structure: the card frame. The same `Header(Class("kcard-head"),
Span(Class("kcard-kind"), Img(Class("tool-icon"), Src(icon), Alt("")),
g.Text(title)), …)` header is hand-rolled in **17 sites** across all six
`internal/feature/*cards` packages. Every site re-types the identical
`Img(Class("tool-icon"), Src(...), Alt(""))` triple. When the card chrome
changes (a class rename, an `Alt` fix, an attribute reorder for CSS), all 17
must be edited in lockstep, and a missed one renders a subtly broken card.

This plan extracts one `ui.CardHead` helper for that header shape and routes the
17 tool-icon sites through it. It is a **pure refactor**: the rendered HTML
(classes, ids, attribute order) must be byte-for-byte identical. The per-feature
`*_test.go` files already assert rendered output (e.g.
`settings_test.go` checks `class="kcard ucard ucard-settings"` and
`/static/icons/key.png`); they are the safety net that proves identity.

## Current state

### The shared primitive surface today

`internal/ui` exposes (and this is the entire set of shared render helpers):

- `internal/ui/components.go` — `ErrorStrip(msg string) g.Node` (the only
  component helper). Full file:
  ```go
  package ui

  import (
  	g "maragu.dev/gomponents"
  	. "maragu.dev/gomponents/html"
  )

  // ErrorStrip is the inline card-error fragment … Never replace g.Text here with g.Raw.
  func ErrorStrip(msg string) g.Node {
  	return Div(Class("card-note card-note-error"), g.Text(msg))
  }
  ```
- `internal/ui/text.go` — `Clip`; `internal/ui/spark.go` — `SparkPoints`,
  `NumericValues`, `SparkW`, `SparkH`; `internal/ui/registry.go` —
  `CardSize`/`Tile`/`Focus`, `CardFunc`, `RegisterCard`/`LookupCard`.
- There is **no** `CardShell`/`CardHead` helper today (verify:
  `grep -rn 'func CardHead' internal/ui` returns nothing).

The import convention to match exactly (used in `components.go`): gomponents is
aliased `g "maragu.dev/gomponents"`, and the HTML package is dot-imported
`. "maragu.dev/gomponents/html"` so `Header`, `Span`, `Img`, `Class`, `Src`,
`Alt` are unqualified. `g.Text` / `g.Node` / `g.Group` stay `g.`-qualified.

### The repeated header shape — the 17 tool-icon sites

The exact common structure (from `internal/feature/taskcards/today.go:69-74`):

```go
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/scroll.png"), Alt("")),
				g.Text("Today"),
			),
		),
```

Three trailing-node shapes appear after the `Span`. Enumerated from the live
files:

1. **No trailing node** — `today.go:69`, `lifelog.go:127`, `settings.go:21`,
   `heads.go:92`.
2. **A `g.If(cond, Span(Class("kcard-meta"), g.Text(line)))` param line** —
   `memory.go:55` (`v.ParamLine`), `knowledgecards/skills.go:117`
   (`paramLine`), `taskcards/quests.go:27` (`v.ParamLine`). Shape:
   ```go
   			g.If(v.ParamLine != "", Span(Class("kcard-meta"), g.Text(v.ParamLine))),
   ```
3. **A trailing `A(Class("kcard-meta"), Href(...), g.Text("… →"))` on manage
   cards** — `memory.go:222` (`/focus/memory`, "manage all →"),
   `knowledgecards/skills.go:229` (`/focus/skills`, "manage all →"),
   `taskcards/quests.go:73` (`/focus/quests`, "all quests →").
4. **A `Span(Class("tag"), g.Text(v.Label))`** — only `journalcards/day.go:86-92`:
   ```go
   		Header(Class("kcard-head"),
   			Span(Class("kcard-kind"),
   				Img(Class("tool-icon"), Src("/static/icons/scroll.png"), Alt("")),
   				g.Text("day"),
   			),
   			Span(Class("tag"), g.Text(v.Label)),
   		),
   ```

The remaining tool-icon sites are in `taskcards/calendar.go:117`,
`taskcards/timeline.go:85`, `taskcards/habits.go:58`, `lifecards/measure.go:85`,
`lifecards/lines.go:67`, `journalcards/journal.go:67`. **Open each before
converting it** — most are shape 1 or 2; confirm the live trailing node.

Full enumeration command (expect **20** lines, of which **17** carry the
tool-icon Img — the 3 that do NOT are the record-card headers in §"Out of
scope"): `grep -rn 'Header(Class("kcard-head")' internal/feature`. (Verify the
counts: `grep -rc 'Header(Class("kcard-head")' internal/feature` sums to 20;
`grep -rc 'Img(Class("tool-icon")' internal/feature` sums to 17.)

### The 3 record-card headers that are OUT of scope

These headers have **no** tool-icon Img — they use a text glyph instead — so
`CardHead` does not apply. Do **not** touch them:

- `internal/feature/knowledgecards/memory.go:108` —
  `Span(Class("kcard-kind"), g.Text("▪ "+memoryCategory(r.Category)))` plus
  `recordPips(...)`.
- `internal/feature/knowledgecards/skills.go:157` —
  `Span(Class("kcard-kind"), g.Text("⌥ skill"))` plus a `kcard-on` span.
- `internal/feature/taskcards/taskcard.go:26` —
  `Span(Class("kcard-kind"), g.Text("▪ task"))` plus a `g.If(...Span.tag)`.

(That is why the grep yields 20 header lines but only 17 tool-icon Imgs:
20 − 17 = 3 record headers without a tool-icon.)

### How a feature test asserts the rendered HTML (the safety net)

`internal/feature/settingscards/settings_test.go` (full):

```go
func TestSettingsCard(t *testing.T) {
	var b strings.Builder
	if err := settingscards.SettingsCard().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`id="ucard-settings"`,
		`class="kcard ucard ucard-settings"`,
		"Settings",
		`href="/focus/settings?section=profile"`, "Profile",
		…
	} {
		if !strings.Contains(out, want) { t.Errorf("missing %q in:\n%s", want, out) }
	}
}
```

These `strings.Contains` checks pass only if the converted card renders the same
substrings. `lifelog_test.go:31` additionally asserts `/static/icons/orb.png`
and `"Life"` — exactly the header content `CardHead` must reproduce.

### Attribute-order constraint (why this is delicate)

gomponents renders element attributes **in call order**. The legacy Img is
`Img(Class("tool-icon"), Src("…"), Alt(""))` → `<img class="tool-icon"
src="…" alt="">`. `CardHead` MUST build the Img with `Class`, then `Src`, then
`Alt` in that order, or the output string changes and tests fail. Same for the
`Header`→`Span` nesting and class strings.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Drift | `git diff --stat 1f8f55e..HEAD -- internal/ui/ internal/feature/` | empty |
| Vet (ui) | `go vet ./internal/ui/` | exit 0 |
| Test (ui) | `go test ./internal/ui/` | all pass |
| Test (one feature pkg) | `go test ./internal/feature/taskcards/` | all pass |
| All feature tests | `go test ./internal/feature/...` | all pass |
| All tests | `go test ./...` | all pass |
| Build (CGO off) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Format check | `gofmt -l .` | prints nothing |
| Whitespace check | `git diff --check` | no output |
| Header sites | `grep -rn 'Header(Class("kcard-head")' internal/feature` | 20 lines |
| Tool-icon sites remaining | `grep -rc 'Img(Class("tool-icon")' internal/feature` | 0 after conversion (sum) |

## Scope

**In scope** (modify only these):

- `internal/ui/components.go` — add the `CardHead` helper (or a new
  `internal/ui/cardhead.go` if you prefer one helper per file — match the
  repo's one-helper-`components.go` precedent and keep it in `components.go`).
- `internal/ui/cardhead_test.go` (**create**) — or extend
  `components_test.go`; a unit test for the helper.
- The 17 tool-icon header sites across these files only:
  - `internal/feature/taskcards/today.go`, `quests.go`, `calendar.go`,
    `timeline.go`, `habits.go`
  - `internal/feature/journalcards/day.go`, `journal.go`
  - `internal/feature/knowledgecards/memory.go` (line 55 only), `skills.go`
    (lines 117 and 229 only)
  - `internal/feature/lifecards/lifelog.go`, `measure.go`, `lines.go`
  - `internal/feature/settingscards/settings.go`
  - `internal/feature/headscards/heads.go`
- Those packages' `*_test.go` files only if an assertion needs a trivial
  update (it should NOT — the output is identical; if it does, that is a
  STOP condition, see below).

**Out of scope** (do NOT touch):

- The 3 record-card headers without a tool-icon (`memory.go:108`,
  `skills.go:157`, `taskcard.go:26`). They use a text glyph, not the tool-icon
  shape — forcing them through `CardHead` would be a lossy fit. Leave them
  exactly as-is.
- `internal/web/*` — the `cardInto`/dispatch layer is unchanged.
- `web/templates/*` — this is the gomponents path only; do not touch templates.
- Any CSS / `static/` asset. **No new card chrome** — pixel-identical output.
- Card bodies and footers (`Footer(Class("kcard-actions"), …)`). This plan
  extracts the **header** only. Do not also try to extract the footer or the
  outer `Article`/`Class("kcard ucard …")` wrapper — that is deferred (see
  Maintenance notes), and bundling it expands the blast radius past LOW–MED.

## Git workflow

- Branch: `improve/071-ui-cardshell-helper`
- Commit per logical unit; conventional-commit style. Suggested sequence:
  `feat(ui): add CardHead helper for the shared kcard header`, then one commit
  per converted feature package, e.g.
  `refactor(taskcards): render card headers via ui.CardHead`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `ui.CardHead` and a unit test

In `internal/ui/components.go`, add a helper that produces the tool-icon header.
It must accept the icon src, the title, and an optional trailing node (to cover
the param-line / "manage all →" / `Span.tag` variants), and it must build the
Img attributes in `Class, Src, Alt` order.

Target shape:

```go
// CardHead renders the shared kcard header: a kcard-kind span with the
// tool-icon image and the card title, plus an optional trailing node (a
// kcard-meta param line, a "manage all →" link, a tag, …). It exists so the
// card frame lives once instead of being hand-copied across every feature card.
// Attribute order (class, src, alt on the img) is load-bearing: the rendered
// HTML must stay byte-identical to the hand-rolled headers it replaces.
func CardHead(iconSrc, title string, trailing ...g.Node) g.Node {
	return Header(Class("kcard-head"),
		g.Group(append([]g.Node{
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src(iconSrc), Alt("")),
				g.Text(title),
			),
		}, trailing...)),
	)
}
```

`g.Group(append(...))` flattens the variadic trailing nodes into the Header's
children so a `nil`/empty `trailing` renders exactly `<header
class="kcard-head"><span class="kcard-kind">…</span></header>` — identical to
shape 1. (Verify this against a test in Step 1's verification; if `g.Group`
wrapping changes the output vs. spreading children directly, prefer building the
`[]g.Node` and passing `Header(children...)` instead — pick whichever renders
the byte-identical string.)

Add `internal/ui/cardhead_test.go`:

```go
package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCardHeadNoTrailing(t *testing.T) {
	var b strings.Builder
	if err := ui.CardHead("/static/icons/scroll.png", "Today").Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	want := `<header class="kcard-head"><span class="kcard-kind"><img class="tool-icon" src="/static/icons/scroll.png" alt="">Today</span></header>`
	if got != want {
		t.Fatalf("header drift:\n got: %s\nwant: %s", got, want)
	}
}

func TestCardHeadWithTrailing(t *testing.T) {
	var b strings.Builder
	trailing := Span(Class("kcard-meta"), g.Text("limit: 6"))
	if err := ui.CardHead("/static/icons/tome.png", "Memory", trailing).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `<img class="tool-icon" src="/static/icons/tome.png" alt="">`) {
		t.Fatalf("img drift: %s", got)
	}
	if !strings.Contains(got, `<span class="kcard-meta">limit: 6</span></header>`) {
		t.Fatalf("trailing node misplaced: %s", got)
	}
}
```

The `want` string in `TestCardHeadNoTrailing` is the source of truth for the
exact bytes — if it does not match, fix `CardHead` until it does (do NOT loosen
the test to `Contains`).

**Verify**:
```
go test ./internal/ui/            # all pass, incl. the 2 new tests
gofmt -l internal/ui/             # prints nothing
go vet ./internal/ui/             # exit 0
```

### Step 2: Convert ONE package first — `settingscards` (simplest, shape 1)

`settings.go:21-26` currently:

```go
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/key.png"), Alt("")),
				g.Text("Settings"),
			),
		),
```

Replace with:

```go
		ui.CardHead("/static/icons/key.png", "Settings"),
```

Add the `"github.com/alexradunet/balaur/internal/ui"` import. Note: after this
change `settings.go` may no longer reference `Header`, `Span`, `Img`, `Src`,
`Alt` — but it still dot-imports `. "maragu.dev/gomponents/html"` for
`Article`, `Class`, `ID`, `Ul`, `Li`, `A`, `Footer`, so the dot-import stays.
`g` stays (for `g.Text` in the footer/body). Remove an import only if `go vet`
/ build reports it unused.

**Verify** (this is the gate that proves byte-identity):
```
go test ./internal/feature/settingscards/   # TestSettingsCard passes unchanged
gofmt -l internal/feature/settingscards/    # prints nothing
```
If `TestSettingsCard` fails on a missing substring, the output changed — that is
a STOP condition (see below), not something to fix by editing the test.

### Step 3: Convert the remaining 16 tool-icon sites, one package at a time

For each package below, replace every tool-icon `Header(...)` block with a
`ui.CardHead(iconSrc, title, trailing...)` call, then run that package's tests
**before** moving on. Keep the trailing node exactly as it was.

Conversion recipes by shape:

- **Shape 1 (no trailing)** → `ui.CardHead(icon, title)`.
- **Shape 2 (param line)** →
  `ui.CardHead(icon, title, g.If(cond, Span(Class("kcard-meta"), g.Text(line))))`.
  `g.If` returns a `g.Node`, so it is a valid trailing arg.
- **Shape 3 (manage "… →" link)** →
  `ui.CardHead(icon, title, A(Class("kcard-meta"), Href(url), g.Text(label)))`.
- **Shape 4 (day tag)** →
  `ui.CardHead("/static/icons/scroll.png", "day", Span(Class("tag"), g.Text(v.Label)))`.

Package order (run the listed test after each; never proceed on a failure):

1. `taskcards` — `today.go:69` (shape 1), `quests.go:27` (shape 2),
   `quests.go:73` (shape 3, `/focus/quests`, "all quests →"), plus
   `calendar.go:117`, `timeline.go:85`, `habits.go:58` (open each; apply the
   matching shape). Add the `internal/ui` import to each file that gains a
   `ui.CardHead` call.
   **Verify**: `go test ./internal/feature/taskcards/` → all pass.
2. `journalcards` — `day.go:86` (shape 4), `journal.go:67` (open; likely
   shape 1 or 2).
   **Verify**: `go test ./internal/feature/journalcards/` → all pass.
3. `knowledgecards` — **only** `memory.go:55` (shape 2, `v.ParamLine`),
   `memory.go:222` (shape 3, `/focus/memory`, "manage all →"),
   `skills.go:117` (shape 2, `paramLine`), `skills.go:229` (shape 3,
   `/focus/skills`, "manage all →"). Do **NOT** touch `memory.go:108` or
   `skills.go:157` (record headers, out of scope).
   **Verify**: `go test ./internal/feature/knowledgecards/` → all pass.
4. `lifecards` — `lifelog.go:127` (shape 1), `measure.go:85`, `lines.go:67`
   (open each).
   **Verify**: `go test ./internal/feature/lifecards/` → all pass.
5. `headscards` — `heads.go:92` (shape 1).
   **Verify**: `go test ./internal/feature/headscards/` → all pass.

After each package, also run `gofmt -l <pkg dir>` (prints nothing) and remove
any now-unused import the build flags.

### Step 4: Whole-tree verification

```
grep -rc 'Img(Class("tool-icon")' internal/feature   # every line → 0
go vet ./...
go test ./...
CGO_ENABLED=0 go build ./...
gofmt -l .
git diff --check
```

**Verify**: no file still contains `Img(Class("tool-icon")` (the 17 sites are
gone; the 3 record headers never had it); vet clean; **all** tests pass
(the feature `*_test.go` substring assertions are the byte-identity proof);
CGO-off build exits 0; `gofmt -l .` and `git diff --check` print nothing.

## Test plan

- **New tests** (`internal/ui/cardhead_test.go`): `TestCardHeadNoTrailing`
  (exact-string equality — the byte contract) and `TestCardHeadWithTrailing`
  (trailing node placed after the kind span, img attribute order intact).
  Model them after `internal/ui/components_test.go` (same `package ui_test`,
  `strings.Builder` + `.Render`).
- **Existing tests are the regression net**: every converted feature package
  has a `*_test.go` asserting rendered substrings (e.g.
  `settings_test.go`, `lifelog_test.go:31` → `/static/icons/orb.png` + `"Life"`,
  `memory_test.go`, `quests_test.go`, `skills_test.go`). They must pass
  **unchanged** — that is what proves the refactor preserved output. Do not
  edit them to make them pass.
- Verification: `go test ./...` → all pass, including the 2 new ui tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/ui/components.go` (or `internal/ui/cardhead.go`) defines
      `func CardHead(iconSrc, title string, trailing ...g.Node) g.Node`
- [ ] `internal/ui/cardhead_test.go` exists; `TestCardHeadNoTrailing` asserts
      exact-string equality and passes
- [ ] `grep -rc 'Img(Class("tool-icon")' internal/feature` → every line is `0`
- [ ] `go test ./...` passes (feature `*_test.go` assertions unchanged)
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `gofmt -l .` prints nothing and `git diff --check` has no output
- [ ] `git status --porcelain` shows only in-scope files modified plus the new
      `internal/ui/cardhead_test.go`; no `internal/web/*`, `web/templates/*`,
      or CSS touched
- [ ] The 3 record-card headers (`memory.go:108`, `skills.go:157`,
      `taskcard.go:26`) are unchanged (`git diff` shows no edit to those
      header blocks)
- [ ] `plans/readme.md` status row for 071 updated (unless your reviewer
      maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- The "Current state" excerpts don't match the live files (drift — the grep
  yields ≠20 header lines, or a cited line differs).
- A converted package's existing `*_test.go` fails on a rendered-HTML substring
  after conversion. That means `CardHead` changed the output (attribute order,
  class string, nesting). The refactor MUST be byte-identical — do **not** fix
  it by editing the test. Report the exact `got` vs. `want`.
- `TestCardHeadNoTrailing`'s exact-string `want` cannot be made to match by
  building the helper as shown (e.g. `g.Group` wrapping inserts/strips bytes) —
  report what the helper actually renders so the helper shape can be adjusted.
- **Two tool-icon header sites diverge so much that one helper can't cover them
  without 4+ option parameters** — report it. The fix may be two helpers, or
  leaving one outlier hand-rolled. Do NOT over-parameterize `CardHead` to force
  all sites through it (YAGNI). The four trailing shapes documented here are all
  expressible via the single `trailing ...g.Node` variadic; if you find a fifth
  that isn't, that is the trigger to stop.
- A site you open turns out to be a record-card header (no tool-icon Img) not
  listed in "Out of scope" — leave it, and report it so the out-of-scope list
  can be corrected.

## Maintenance notes

For the human/agent who owns this after it lands:

- A reviewer should diff one converted card's rendered output against the
  pre-change output (e.g. capture `SettingsCard().Render` before/after) to
  confirm byte-identity, and confirm the 3 record-card headers were left alone.
- **Deferred, intentionally not in this plan**: extracting the outer
  `Article(Class("kcard ucard …"), ID(...))` wrapper and the
  `Footer(Class("kcard-actions"), …)` into the helper too (a full `CardShell`).
  That is a larger blast radius and the footer/wrapper vary more (manage cards
  omit the footer; record cards use `kcard kcard-<status>` not `kcard ucard`).
  Do it as a follow-up only if the header extraction proves the pattern safe.
- **Deferred**: the 3 record-card headers (text-glyph kind span + pips/on
  trailing) could get their own `CardHeadGlyph`-style helper if that shape ever
  repeats more; today 3 sites is below the threshold that justifies it.
- If a new feature card is added, it should call `ui.CardHead` rather than
  re-typing the `Img(Class("tool-icon"), …)` triple — that is the whole point.
