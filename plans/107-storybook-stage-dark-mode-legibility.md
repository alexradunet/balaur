# Plan 107: Fix pale-on-parchment text in dark mode on the storybook story stage (`.sb-view-stage`)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 85c0d0d..HEAD -- internal/web/assets/static/basm.css internal/web/assets/css_tokens_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (independent of plan 106 — see "Relationship to plan 106")
- **Category**: bug
- **Planned at**: commit `85c0d0d`, 2026-06-18
- **Issue**: —

## Why this matters

This is the sibling of plan 106. Plan 106 fixed the right-hand artifact panel
(`#panel.app-panel`) in the running app: it was a parchment surface that never
set a text `color`, so in dark mode its content inherited the page-background
`--fg`/`--fg-strong` tokens and went pale/near-white on parchment.

The **storybook** (`/storybook`, the component catalog that the project treats as
its UI source of truth) has the *identical* bug in a second place: its story
**stage** — the framed area each component variant renders inside — defaults to a
parchment background (`--surface`) but never sets ink text. So in dark mode every
story shown on the default (parchment) stage renders its `--fg-strong` headings
(e.g. a card/panel's `.k-heading`, `.empty-title`) near-white-on-parchment, and
any text that merely inherits `--fg` goes pale. This was confirmed visually in a
browser: on `/storybook/modelspanel` in dark mode, the model-card titles are
near-invisible, while the same component in the running app panel (post-106) is
crisp ink.

Fixing the stage makes the dark-mode component catalog legible, so the storybook
stays a trustworthy reference.

## Root cause

The storybook story stage has three background variants, chosen per story:

- **default** `.sb-view-stage` → `background: var(--surface)` (parchment — light in both schemes). **Needs ink text. It doesn't set one.** ← the bug.
- `.sb-views-dark .sb-view-stage` → `background-color: var(--bg)` (dark page bg). Its text should be the page text (`--fg`), which it currently gets only by inheriting `body { color: var(--fg) }`.
- `.sb-views-dock .sb-view-stage` → `background-color: var(--chrome)` (dark wood). Its text should be `--chrome-fg`.

Because the parchment default sets no color, text falls through to
`html, body { color: var(--fg) }` (pale tan in dark mode), and elements that set
`color: var(--fg-strong)` explicitly (panel/card headings) are near-white.

**Why the fix needs three color rules, not one.** If you add
`color: var(--ink)` to the base `.sb-view-stage` rule, that rule also matches the
*dark* and *dock* stages (same element, just with an ancestor class) — and would
paint ink text on those dark backgrounds, making them invisible. The
`.sb-views-dark .sb-view-stage` / `.sb-views-dock .sb-view-stage` selectors are
more specific (0,2,0 vs 0,1,0), so giving each of them its correct color
re-establishes the right text color on those stages. (Setting `--chrome-fg` on
the dock stage also fixes a latent light-mode issue there, where the inherited
dark `--fg` sat on dark wood — incidental, but correct.)

## Relationship to plan 106

Independent change. Plan 106 edits `basm.css` around lines 3390–3411
(`#panel.app-panel`) and inserts a test mid-file; this plan edits `basm.css`
around lines 3068–3085 (`.sb-view-stage`) and **appends** its test at the end of
`css_tokens_test.go`. The two never touch the same lines, so if both branches are
merged (in either order) git merges them cleanly. Do **not** depend on,
cherry-pick, or reference plan 106's changes — they may not be present in your
worktree, and this plan must stand alone.

## Current state

Files involved:

- `internal/web/assets/static/basm.css` — the design-token + component
  stylesheet. The storybook stage rules and the fix live here.
- `internal/web/assets/css_tokens_test.go` — Go test asserting CSS invariants by
  substring-matching the embedded `basm.css`. You will append one test.
- `internal/feature/storybook/story.go` — builds the storybook DOM. **Read-only
  context; do not edit.** It shows where the variant classes come from.

### Storybook DOM structure (read-only — from `story.go`)

`story.go` builds, per story: a `.sb-views` grid container whose class string gets
`sb-views-wide`/`sb-views-dark`/`sb-views-dock` appended from the story's
`Wide`/`OnDark`/`OnDock` flags (lines 131–139); inside it, each variant is a
`.sb-view` figure → `.sb-view-stage` div (holding the component) → `.sb-view-cap`
caption (lines 145–147). So **`.sb-views-dark` / `.sb-views-dock` are ancestors of
`.sb-view-stage`**, and a story is wholly parchment (default), dark, or dock.

### The stage CSS as it exists today (`internal/web/assets/static/basm.css`, lines 3066–3085)

```css
.sb-views-wide .sb-view-stage > * { width: 100%; }
.sb-view { margin: 0; display: flex; flex-direction: column; }
.sb-view-stage {
  flex: 1; display: flex; align-items: center; justify-content: center; min-height: 96px; padding: 20px;
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel);
}
/* Dark stage for dock/page components (chat ledges, the empty hearth, page
   titles) — they render light-on-dark and would vanish on parchment. */
.sb-views-dark .sb-view-stage {
  background-color: var(--bg); background-image: var(--grain-warm); background-size: 4px 4px;
  border-color: var(--outline-2); box-shadow: var(--bevel-up);
}
/* Dock stage: the always-dark wood ledge (--chrome) for chat ledge sub-pieces
   whose text is dock-light in both modes (so --bg, which flips, won't do). */
.sb-views-dock .sb-view-stage {
  background-color: var(--chrome); background-image: var(--wood-planks);
  border-color: var(--outline-2); box-shadow: var(--bevel-up);
}
.sb-view-cap { margin-top: 8px; ... }
```

### The token facts (lines 18–44) and the page default (213–219)

```css
--surface:    light-dark(#f4e9c4, #e8d9ae);   /* parchment — light in BOTH schemes */
--bg:         light-dark(#efe2bd, #140c06);   /* page bg — dark in dark mode */
--chrome:     light-dark(#3a2210, #2a1709);   /* wood chrome — dark in both */
--fg:         light-dark(#3a2c18, #c9b894);   /* body on page bg — pale tan in dark */
--fg-strong:  light-dark(#241a0c, #ecdcb2);   /* headings on page bg — near-WHITE in dark */
--chrome-fg:  light-dark(#d6bb92, #b59872);   /* text on wood */
--ink:        #2c2012;                         /* constant dark — correct on parchment */

html, body { ... color: var(--fg); }           /* what the parchment stage wrongly inherits */
```

### The explicit `--fg-strong` (near-white) heading classes that appear on the parchment stage

Same set as plan 106 (these set their color explicitly, so the stage's inherited
ink won't reach them):

- `h1, h2, h3` — line ~250 (`color: var(--fg-strong)`).
- `.k-heading` — line ~1028, with modifiers `.k-heading-proposed` (gold, line ~1039) and `.k-heading-muted` (muted, line ~1044) that must be preserved.
- `.empty-title` — line ~2731.
- `.tl-label` — line ~1447.
- `.pull-download-name` — line ~1934.

Do **not** edit those base rules (they're correct on the page background and on
the dark/dock stages). Override them only inside the parchment stage (Step 2).

## Commands you will need

| Purpose         | Command                                                   | Expected on success            |
|-----------------|-----------------------------------------------------------|--------------------------------|
| CSS-rule tests  | `go test ./internal/web/assets/`                          | `ok`, all pass                 |
| Full tests      | `go test ./...`                                           | all packages `ok`/`no test files` |
| Vet             | `go vet ./...`                                            | exit 0, no output              |
| CGO-free build  | `CGO_ENABLED=0 go build ./...`                            | exit 0                         |
| Format check    | `gofmt -l internal/web/assets/css_tokens_test.go`         | empty output                   |
| Whitespace check| `git diff --check`                                        | no output                      |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
GOPROXY shim — see `docs/hyperagent-sandbox.md`. Ignore if not in that sandbox.
There is no CSS build step — `basm.css` is embedded via `embed.FS`.

## Scope

**In scope** (the only files you may modify):
- `internal/web/assets/static/basm.css`
- `internal/web/assets/css_tokens_test.go` (append one test, at the END of the file)

**Out of scope** (do NOT touch):
- The base rules `h1,h2,h3`, `.k-heading`, `.empty-title`, `.tl-label`,
  `.pull-download-name`, and any `:root` token — do not change their token; only
  override inside the parchment stage (Step 2).
- The storybook chrome outside the stage (`.sb-canvas`, `.sb-head-*`, `.sb-nav-*`,
  `.sb-side`, `.sb-props`, etc.) — those sit on the page background where
  `--fg`/`--fg-strong` are correct.
- `#panel.app-panel` and the running-app shell (that is plan 106's territory).
- `internal/feature/storybook/*.go` and any other `.go` component file.

## Git workflow

- Branch: `improve/107-storybook-stage-dark-mode-legibility`, based on `main` (`85c0d0d`).
- One commit. Conventional-commit message, e.g.
  `fix(web): ink text on the storybook parchment stage for dark-mode legibility`.
- Match the trailer style of recent commits (`git log -1 --format='%b'`); attribute
  the model you actually are.
- Do NOT push or open a PR.

## Steps

### Step 1: Give each stage variant its correct text color

In `internal/web/assets/static/basm.css`, in the three stage rules (lines
~3068–3084), add a `color` to each so the parchment default is ink and the
dark/dock variants keep their correct text color:

- In `.sb-view-stage { ... }` add `color: var(--ink);`
- In `.sb-views-dark .sb-view-stage { ... }` add `color: var(--fg);`
- In `.sb-views-dock .sb-view-stage { ... }` add `color: var(--chrome-fg);`

Result (only the added `color:` lines are new; keep everything else):

```css
.sb-view-stage {
  flex: 1; display: flex; align-items: center; justify-content: center; min-height: 96px; padding: 20px;
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel);
  color: var(--ink);
}
.sb-views-dark .sb-view-stage {
  background-color: var(--bg); background-image: var(--grain-warm); background-size: 4px 4px;
  border-color: var(--outline-2); box-shadow: var(--bevel-up);
  color: var(--fg);
}
.sb-views-dock .sb-view-stage {
  background-color: var(--chrome); background-image: var(--wood-planks);
  border-color: var(--outline-2); box-shadow: var(--bevel-up);
  color: var(--chrome-fg);
}
```

**Verify**: `grep -n "color: var(--ink);" internal/web/assets/static/basm.css | head` shows a line inside the `.sb-view-stage` block (~line 3072), and
`grep -nE "\.sb-views-dock \.sb-view-stage" -A4 internal/web/assets/static/basm.css` shows `color: var(--chrome-fg);` within that block.

### Step 2: Re-anchor the explicit `--fg-strong` headings on the parchment stage only

The headings that set their own `--fg-strong` won't inherit Step 1's ink. Add an
override **scoped to the parchment (default) stage** — i.e. a `.sb-views` that is
neither dark nor dock — immediately after the `.sb-views-dock .sb-view-stage`
block (before `.sb-view-cap` at ~line 3085):

```css
/* Explicit --fg-strong headings/labels (panel/card headings) won't inherit the
   parchment stage's ink from the rule above, so re-anchor them — but ONLY on the
   parchment (default) stage. On the dark/dock stages --fg-strong stays correct.
   :not() leaves the gold "proposed" and muted .k-heading modifiers intact. */
.sb-views:not(.sb-views-dark):not(.sb-views-dock) .sb-view-stage :is(h1, h2, h3, .empty-title, .tl-label, .pull-download-name),
.sb-views:not(.sb-views-dark):not(.sb-views-dock) .sb-view-stage .k-heading:not(.k-heading-proposed):not(.k-heading-muted) {
  color: var(--ink);
}
```

**Verify**:
- `grep -c "sb-views:not(.sb-views-dark):not(.sb-views-dock) .sb-view-stage .k-heading:not(.k-heading-proposed)" internal/web/assets/static/basm.css` → `1`.
- `grep -c "color: var(--fg-strong)" internal/web/assets/static/basm.css` → `10` (unchanged baseline; you added overrides using `--ink`, edited no base rule).

### Step 3: Append a CSS-invariant regression test

Append a new test at the **END** of `internal/web/assets/css_tokens_test.go`
(after the last existing function — appending at the end avoids any merge
conflict with plan 106, which inserts mid-file). Follow the file's established
pattern (`package assets`, `FS.ReadFile("static/basm.css")`, substring asserts):

```go
// TestStorybookStageInkText guards plan 107: the storybook's default story stage
// (.sb-view-stage) is a parchment surface, so it must set ink text (else
// dark-mode content inherits the page-bg --fg/--fg-strong tokens and goes
// pale/near-white on parchment). The dark/dock stage variants keep their own
// (page/wood) text colors; the explicit --fg-strong headings are re-anchored to
// ink only on the parchment stage, preserving the proposed/muted modifiers.
func TestStorybookStageInkText(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)

	// The parchment-stage --fg-strong override is the load-bearing rule; its
	// exact selector exists only because of this change.
	if !strings.Contains(css, ".sb-views:not(.sb-views-dark):not(.sb-views-dock) .sb-view-stage .k-heading:not(.k-heading-proposed):not(.k-heading-muted)") {
		t.Error("storybook parchment stage must re-anchor headings to ink (panel-stage scoped, modifiers preserved) — plan 107")
	}
	// The dark and dock stages must re-assert their own text colors, else the
	// base .sb-view-stage ink would leak onto those dark backgrounds.
	if !strings.Contains(css, ".sb-views-dock .sb-view-stage") || !strings.Contains(css, "color: var(--chrome-fg)") {
		t.Error("storybook dock stage must set --chrome-fg text (ink would be invisible on wood) — plan 107")
	}
	// The modifier the override must NOT clobber is still gold.
	if !strings.Contains(css, ".k-heading-proposed { color: var(--gold)") {
		t.Error(".k-heading-proposed must stay gold (plan 107 must not flatten it to ink)")
	}
}
```

**Verify**: `go test ./internal/web/assets/ -run TestStorybookStageInkText -v` → `PASS`.

### Step 4: Full verification gates

Run, in order, and confirm each:
1. `gofmt -l internal/web/assets/css_tokens_test.go` → empty output.
2. `go vet ./...` → exit 0.
3. `CGO_ENABLED=0 go build ./...` → exit 0.
4. `go test ./...` → every package `ok` or `no test files`; no `FAIL`.
5. `git diff --check` → no output.
6. `git status --porcelain` → only the two in-scope files modified.

### Step 5: Visual confirmation (best-effort)

The CSS cascade makes this deterministic, but a dark-mode eyeball is the real
acceptance test. If you can run the app: build and serve this worktree
(`CGO_ENABLED=0 go build -o /tmp/balaur-107 . && /tmp/balaur-107 serve --http=127.0.0.1:8097 --dir=<a writable temp dir>`), open
`http://127.0.0.1:8097/storybook/modelspanel`, force dark mode (run
`document.documentElement.classList.add('dark')` in the devtools console, or set
OS dark), and confirm the model-card titles and "No local models yet" are dark
ink and legible on the parchment stage — and that any `OnDark`/`OnDock` story
(e.g. a chat-ledge or page-title story) is unchanged.

If you cannot run a browser, say so explicitly and rely on the CSS analysis + the
Step-3 test + the gates. Do NOT install a model or download anything.

## Test plan

- **New test**: `TestStorybookStageInkText`, appended to
  `internal/web/assets/css_tokens_test.go`, modeled on the existing
  `TestCmdPaletteActiveStyle` / `TestThemePaletteBlocks` in that file. Asserts
  (1) the parchment-stage heading override exists with the `:not()` modifier
  guard, (2) the dock stage sets `--chrome-fg`, (3) `.k-heading-proposed` stays
  gold.
- **Verification**: `go test ./internal/web/assets/` → all pass incl. the new
  test. `go test ./...` → no package regresses.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `.sb-view-stage` base rule contains `color: var(--ink);`; `.sb-views-dark .sb-view-stage` contains `color: var(--fg);`; `.sb-views-dock .sb-view-stage` contains `color: var(--chrome-fg);`.
- [ ] The parchment-stage override (`:is(h1,h2,h3,.empty-title,.tl-label,.pull-download-name)` + `.k-heading:not(.k-heading-proposed):not(.k-heading-muted)`, scoped to `.sb-views:not(.sb-views-dark):not(.sb-views-dock) .sb-view-stage`) is present (exactly 1 match for the `.k-heading` selector).
- [ ] `grep -c "color: var(--fg-strong)" internal/web/assets/static/basm.css` returns `10` (no base rule edited).
- [ ] `go test ./internal/web/assets/` exits 0; `TestStorybookStageInkText` exists and passes.
- [ ] `go vet ./...` exits 0; `CGO_ENABLED=0 go build ./...` exits 0; `go test ./...` exits 0 (no FAIL).
- [ ] `git diff --check` clean; `git status --porcelain` shows only the two in-scope files.
- [ ] `plans/readme.md` status row for 107 updated.

## STOP conditions

Stop and report back if:

- The drift check shows `basm.css` changed since `85c0d0d` and the stage excerpts
  (the three `.sb-view-stage` rules, or the `--surface`/`--bg`/`--chrome`/`--fg`/`--ink`
  tokens) no longer match the live file.
- The `.sb-view-stage` selector or the `.sb-views-dark`/`.sb-views-dock` variant
  classes no longer exist or were renamed (the storybook stage was refactored).
- Adding the override visibly changes an `OnDark`/`OnDock` story or any
  storybook chrome outside the stage.
- A verification fails twice after a reasonable fix attempt.
- The fix appears to require editing a base rule, a token, or any `.go` file.

## Maintenance notes

- This is the second instance of one root principle: **a parchment surface
  (`--surface` bg) must pair with ink text**, mirroring `.parch`/`.card`. Plan
  106 fixed `#panel.app-panel`; this fixes `.sb-view-stage`. If a future surface
  introduces a third parchment region, it needs the same pairing. A future
  consolidation could factor the "parchment → ink + heading override" block into
  one shared helper/selector list instead of repeating it per surface — deferred
  here to keep the fix surgical and independent of plan 106.
- A reviewer should confirm the override stayed scoped to the parchment stage
  (the `:not(.sb-views-dark):not(.sb-views-dock)` guard) and that the dark/dock
  stage variants still read correctly in both light and dark mode.
- Deferred (same as 106): `--muted` secondary text on parchment is ~3:1 contrast;
  not addressed here.
