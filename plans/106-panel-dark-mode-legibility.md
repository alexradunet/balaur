# Plan 106: Fix pale-on-parchment text in dark mode across the right-panel domain views (Models, Quests, Life, Skills)

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

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `85c0d0d`, 2026-06-18
- **Issue**: —

## Why this matters

Owner-reported: *"Some UI is not shown properly. Sometimes I have white text on
parchment background which can't be seen properly in dark mode. Example Models
panel, quest log, life, skills. I think in all of them, let's fix that."*

The Balaur web UI is a single-page chat shell with a single right-hand artifact
panel (`#panel.app-panel`). That panel is always a **parchment** surface (a
light cream material that stays light in *both* light and dark themes). But the
panel column never sets a text `color`, so its content inherits the page-level
`--fg` token — which is correct *pale tan* text for the dark page background and
**near-invisible on parchment**. A few headings/labels go further and explicitly
use `--fg-strong`, which in dark mode is near-white cream (`#ecdcb2`) — that is
the "white text on parchment" the owner sees. The result: in dark mode the
Models / Quests / Life / Skills panels (and any other panel) render headings and
plain text that you can barely read.

This is a one-source-of-truth CSS bug: the parchment panel forgot to pair its
parchment background with ink text the way every other parchment surface does.
Fixing it makes every right-panel domain view legible in dark mode at once.

## Root cause (read this before touching anything)

The design system pairs each *material* with a text color:

- **Parchment** material (`.parch`, `.card`, `.kcard`) → `background: var(--surface)` + `color: var(--ink)`.
- **Wood** chrome material (`.wood`) → `background: var(--chrome)` + `color: var(--chrome-fg)`.

`--surface` (parchment) and `--ink` (its text) are **constant across light/dark**
— parchment is always light, ink is always dark `#2c2012`. The page-background
tokens `--fg` / `--fg-strong`, by contrast, **flip** with the scheme (dark text on
the light page in light mode; pale text on the near-black page in dark mode).

The right-panel column declares the parchment background but **omits the matching
`color: var(--ink)`**, so panel text falls through to the page default
(`html, body { color: var(--fg) }`). In dark mode that is pale tan on parchment.
Elements that set `color: var(--fg-strong)` explicitly (panel headings, the
empty-state title, timeline labels, the download-name) are the most visible
casualties — near-white on parchment.

## Current state

Files involved:

- `internal/web/assets/static/basm.css` — the single design-token + component
  stylesheet (canonical token source). The bug and the fix both live here.
- `internal/web/assets/css_tokens_test.go` — Go test that asserts CSS invariants
  by substring-matching the embedded `basm.css` (this is the established way the
  repo guards CSS rules; you will add one test here).
- `internal/ui/chat/panel.go` — builds the panel DOM (`#panel-inner` →
  `.panel-head` + `#panel-body.panel-body`). **Read-only context; do not edit.**

### The relevant tokens (`internal/web/assets/static/basm.css`, lines 18–44)

```css
:root {
  color-scheme: dark light;
  ...
  /* Parchment content panels (constant across modes) */
  --surface:    light-dark(#f4e9c4, #e8d9ae);   /* parchment — light in BOTH schemes */
  ...
  /* Text */
  --fg:        light-dark(#3a2c18, #c9b894);  /* body on page bg — pale tan in dark */
  --fg-strong: light-dark(#241a0c, #ecdcb2);  /* headings on page bg — near-WHITE in dark */
  --muted:     light-dark(#7a6644, #8a7355);
  ...
  /* Ink — text on parchment (constant) */
  --ink:       #2c2012;        /* constant dark brown — correct on parchment */
  --ink-muted: #6f5e3c;
}
```

### The page default that leaks into the panel (lines 213–219)

```css
html, body {
  ...
  color: var(--fg);   /* pale tan in dark mode */
}
```

### The exemplar to mirror — parchment material pairs bg WITH ink (lines 289–298)

```css
/* Parchment panel — the content material. Ink text, paper grain,
   paper-edge bevel, hard drop onto the wood below. */
.parch {
  background-color: var(--surface);
  background-image: var(--grain-ink);
  background-size: 4px 4px;
  color: var(--ink);          /* ← the panel column is missing exactly this */
  ...
}
```

### The bug — the panel column has the parchment bg but NO color (lines 3393–3398)

```css
/* Right panel column — single-active artifact canvas (plan 098). */
html.app #panel.app-panel {
  position: relative; height: 100%; overflow-y: auto; z-index: var(--z-base);
  border-left: 2px solid var(--parch-edge);
  background-color: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
}
```

Note: the panel header `.panel-head` (line ~3401) already correctly sets
`color: var(--ink)`. It is only the *body* content that inherits `--fg`.

### The explicit `--fg-strong` (near-white) offenders that render inside the panel

These five selectors set `color: var(--fg-strong)` directly, so the container
fix alone (an *inherited* color) will not reach them — they need an explicit
override. Confirmed locations:

- `h1, h2, h3` — line ~250–251 (global base heading color).
- `.k-heading` — line ~1028–1032 (section headings used by Quests/Life/Skills/Models panels).
- `.empty-title` — line ~2731 (the EmptyState title, e.g. "No local models yet").
- `.tl-label` — line ~1447 (timeline day labels — the Day/timeline panel).
- `.pull-download-name` — line ~1934 (model-download progress row).

```css
/* line ~250 */
h1, h2, h3 {
  font-family: var(--font-display);
  color: var(--fg-strong);
  ...
}

/* line ~1028 */
.k-heading {
  display: flex; align-items: center; gap: 10px; font-size: 22px;
  color: var(--fg-strong);
}
.k-heading-proposed { color: var(--gold); }   /* MODIFIER — must stay gold */
.k-heading-muted    { color: var(--muted); }   /* MODIFIER — must stay muted */

/* line ~2731 */
.empty-title { margin: 0; font-family: var(--font-display); font-size: 22px; color: var(--fg-strong); }

/* line ~1447 */
.tl-label { ...; color: var(--fg-strong); }

/* line ~1934 */
.pull-download-name { font-family: var(--font-mono); font-size: 12px; color: var(--fg-strong); ... }
```

**Critical subtlety — do not clobber the `.k-heading` modifiers.** `.k-heading`
is also worn with `.k-heading-proposed` (intentionally gold) and
`.k-heading-muted` (intentionally muted). The override must target *plain*
`.k-heading` only, via `:not(.k-heading-proposed):not(.k-heading-muted)`, so the
gold "Proposed" heading in the Skills panel and the muted variant keep their
colors.

### Why the fix is panel-scoped (and not a global token change)

`h1,h2,h3`, `.empty-title`, and `.tl-label` are *also* used outside the panel —
on the page/wood background and in the `/storybook`, where `--fg-strong` is the
*correct* light-on-dark color. Changing those base rules globally to `--ink`
would make them dark-on-dark (invisible) there. So the fix is scoped under
`html.app #panel.app-panel` — it changes panel rendering only and leaves the
storybook and page-background renderings untouched.

### What is intentionally NOT changed

`--muted`-colored secondary text inside the panel (`.k-sub`, `.k-empty`,
`.k-heading-muted`, `.empty-line`, the `.kcard-*` metadata) is *medium brown* in
dark mode (`#8a7355` on `#e8d9ae`) — legible, and an intentional hierarchy step
below ink. The owner reported *white* text, not muted text. Leave `--muted`
alone (see "Maintenance notes" for the deferred contrast follow-up). Likewise
`.k-count` uses `--chrome-fg` on a `--chrome` (dark wood) pill background — that
pairing is correct; do not touch it.

## Commands you will need

| Purpose         | Command                                                   | Expected on success            |
|-----------------|-----------------------------------------------------------|--------------------------------|
| CSS-rule tests  | `go test ./internal/web/assets/`                          | `ok`, all pass                 |
| Full tests      | `go test ./...`                                           | all packages `ok`/`no test files` |
| Vet             | `go vet ./...`                                            | exit 0, no output              |
| CGO-free build  | `CGO_ENABLED=0 go build ./...`                            | exit 0                         |
| Format check    | `gofmt -l internal/web/assets/css_tokens_test.go`         | empty output (no files listed) |
| Whitespace check| `git diff --check`                                        | no output                      |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
GOPROXY shim — see `docs/hyperagent-sandbox.md`. Ignore if not in that sandbox.

There is no CSS build step — `basm.css` is embedded via `embed.FS` and served
directly, so editing the file is the whole change. A PostToolUse hook runs
`gofmt -w` on edited Go files automatically; the format check above is a belt-and-braces confirmation.

## Scope

**In scope** (the only files you may modify):
- `internal/web/assets/static/basm.css`
- `internal/web/assets/css_tokens_test.go` (add one test)

**Out of scope** (do NOT touch, even though they look related):
- The base rules `h1, h2, h3` (~line 250), `.k-heading` (~1028), `.empty-title`
  (~2731), `.tl-label` (~1447), `.pull-download-name` (~1934) — do **not** change
  their token globally; override them *only* inside the panel scope (Step 2). A
  global change breaks the storybook/page-background renderings.
- `--muted`, `--chrome-fg`, `.k-count`, and any token *definitions* in `:root` —
  the token values are correct; only the panel's *use* of color is wrong.
- `internal/ui/chat/panel.go` and any `.go` component files — markup is fine; the
  fix is pure CSS.
- The chat/dock side of the shell, the storybook, themes (forest/dungeon).

## Git workflow

- Branch: `improve/106-panel-dark-mode-legibility` (the repo's observed convention — see plans 104/105).
- One commit is fine (small CSS + test change). Message style: conventional
  commits, e.g.
  `fix(web): ink text on the right panel so dark-mode parchment is legible`.
- End the commit message with the repo's trailer if one is in use (check
  `git log -1 --format='%b'` on a recent commit; match it).
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Give the parchment panel column its ink text color

In `internal/web/assets/static/basm.css`, find the `html.app #panel.app-panel`
block (around line 3393). Add `color: var(--ink);` to it and expand the comment
to explain why. The block becomes:

```css
/* Right panel column — single-active artifact canvas (plan 098). The panel
   surface is constant parchment in BOTH schemes (--surface), so its text must be
   ink — not the page-bg --fg* tokens, which flip pale in dark mode and would
   render near-invisible on parchment. Mirror the .parch bg+ink pairing. */
html.app #panel.app-panel {
  position: relative; height: 100%; overflow-y: auto; z-index: var(--z-base);
  border-left: 2px solid var(--parch-edge);
  background-color: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  color: var(--ink);
}
```

This fixes every panel element that *inherits* its color (plain text, paragraphs,
buttons that don't set their own color, etc.).

**Verify**: `grep -n "color: var(--ink);" internal/web/assets/static/basm.css | head`
→ the `#panel.app-panel` block now contains a `color: var(--ink);` line. Visually
re-read the block to confirm the new line is *inside* the braces.

### Step 2: Override the explicit `--fg-strong` headings/labels, panel-scoped

Immediately **after** the `#panel.app-panel` block from Step 1 (before the
`#panel-inner` rule at ~line 3399), add a new rule. Headings/labels that set
their color explicitly won't pick up the inherited color from Step 1, so
re-anchor them to ink — but only inside the panel, and without clobbering the
gold/muted `.k-heading` modifiers:

```css
/* Headings/labels inside the panel set color explicitly to the page-bg
   --fg-strong (near-white in dark mode, correct only on the dark page) — so the
   inherited ink from the column rule above can't reach them. Re-anchor them to
   ink. Panel-scoped: the same classes on the page bg / in /storybook keep
   --fg-strong. :not() leaves the gold "proposed" and muted heading modifiers. */
html.app #panel.app-panel :is(h1, h2, h3, .empty-title, .tl-label, .pull-download-name),
html.app #panel.app-panel .k-heading:not(.k-heading-proposed):not(.k-heading-muted) {
  color: var(--ink);
}
```

**Verify**:
- `grep -n "k-heading:not(.k-heading-proposed)" internal/web/assets/static/basm.css`
  → exactly one match (your new rule).
- `grep -n ".k-heading-proposed { color: var(--gold)" internal/web/assets/static/basm.css`
  → still present and unchanged (you did not edit the modifier).

### Step 3: Add a CSS-invariant regression test

In `internal/web/assets/css_tokens_test.go`, add a test that asserts the panel
ink rules exist, following the exact pattern of the existing
`TestCmdPaletteActiveStyle` / `TestAppDockResetsTop` tests in that file (read
them first — same `package assets`, same `FS.ReadFile("static/basm.css")`,
substring asserts). Append:

```go
// TestPanelInkText guards plan 106: the right-panel column is a constant
// parchment surface, so it must set ink text (else dark-mode content inherits
// the page-bg --fg/--fg-strong tokens and goes pale/near-white on parchment).
// Two rules: the column defaults to ink, and the explicit --fg-strong
// headings/labels are re-anchored to ink (panel-scoped, modifiers preserved).
func TestPanelInkText(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)

	// The column must set its own ink color (not inherit the page-bg --fg).
	if !strings.Contains(css, "html.app #panel.app-panel {") ||
		!strings.Contains(css, "color: var(--ink)") {
		t.Error("right panel column must set color: var(--ink) — dark-mode parchment text would be illegible (plan 106)")
	}
	// The explicit --fg-strong headings/labels must be re-anchored, panel-scoped,
	// without clobbering the gold/muted .k-heading modifiers.
	if !strings.Contains(css, "html.app #panel.app-panel .k-heading:not(.k-heading-proposed):not(.k-heading-muted)") {
		t.Error("panel headings (.k-heading) must be re-anchored to ink while preserving the proposed/muted modifiers (plan 106)")
	}
	// The modifier the override must NOT clobber is still gold.
	if !strings.Contains(css, ".k-heading-proposed { color: var(--gold)") {
		t.Error(".k-heading-proposed must stay gold (plan 106 must not flatten it to ink)")
	}
}
```

**Verify**: `go test ./internal/web/assets/` → `ok`, including the new
`TestPanelInkText`. Run `go test ./internal/web/assets/ -run TestPanelInkText -v`
to confirm it is actually executed and passes.

### Step 4: Full verification gates

Run, in order, and confirm each:

1. `gofmt -l internal/web/assets/css_tokens_test.go` → empty output.
2. `go vet ./...` → exit 0.
3. `CGO_ENABLED=0 go build ./...` → exit 0.
4. `go test ./...` → every package `ok` or `no test files`; no `FAIL`.
5. `git diff --check` → no output.
6. `git status --porcelain` → only the two in-scope files modified.

### Step 5: Visual confirmation in the running app (best-effort)

The CSS cascade makes this fix deterministic, but a real dark-mode eyeball is the
true acceptance test and the regression test above only checks the rules *exist*,
not how they *render*.

If you can run the app: `make run` (or `make dev`), open the UI, **force dark
mode** (the app respects OS preference; force it with `<html class="dark">` via
the theme toggle or devtools), then open each panel — Models, Quests (quest log),
Life, Skills — through the composer `/`-command palette and confirm every heading
and line of text is dark ink and clearly readable on the parchment, with the
Skills "Proposed" heading still gold.

If you **cannot** run the app in this environment (no model installed, headless,
etc.), do not block on it — say so explicitly in your status update and rely on
the CSS analysis + the Step-3 test + the gates. Do NOT install a model or
download anything to satisfy this step.

## Test plan

- **New test**: `TestPanelInkText` in `internal/web/assets/css_tokens_test.go`,
  modeled structurally on the existing `TestCmdPaletteActiveStyle` in the same
  file. It covers: (1) the column sets `color: var(--ink)`; (2) the panel-scoped
  `.k-heading` override with the `:not()` modifier guard exists; (3) the
  `.k-heading-proposed` gold modifier is still present (i.e. the fix did not
  flatten it). This test fails if a future edit drops the panel ink rules or
  clobbers the modifier.
- **Verification**: `go test ./internal/web/assets/` → all pass, including the
  one new test. `go test ./...` → no package regresses (this is a CSS + test-only
  change; no behavior in other packages is touched).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] The `html.app #panel.app-panel { ... }` block contains `color: var(--ink);`.
- [ ] A panel-scoped rule re-anchors `:is(h1, h2, h3, .empty-title, .tl-label, .pull-download-name)` and `.k-heading:not(.k-heading-proposed):not(.k-heading-muted)` to `var(--ink)`.
- [ ] `grep -c "color: var(--fg-strong)" internal/web/assets/static/basm.css` returns the **same count as before your change** (you added overrides; you did not edit any base `--fg-strong` rule). Record the pre-change count first: `git show 85c0d0d:internal/web/assets/static/basm.css | grep -c "color: var(--fg-strong)"`.
- [ ] `go test ./internal/web/assets/` exits 0; `TestPanelInkText` exists and passes.
- [ ] `go vet ./...` exits 0.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./...` exits 0 (no FAIL).
- [ ] `git diff --check` is clean.
- [ ] `git status --porcelain` shows only `internal/web/assets/static/basm.css` and `internal/web/assets/css_tokens_test.go` modified.
- [ ] `plans/readme.md` status row for 106 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `basm.css` changed since `85c0d0d` and the "Current
  state" excerpts (the `#panel.app-panel` block, or the `--surface`/`--fg`/`--ink`
  token definitions, or any of the five `--fg-strong` selectors) no longer match
  the live file — the cascade reasoning may no longer hold.
- The `#panel.app-panel` selector no longer exists or has been renamed (the panel
  architecture was refactored) — the fix target moved; report what you found.
- Adding the override visibly changes a *light-mode* panel or the storybook (it
  should not — `--ink` matches the old light-mode `--fg-strong` closely; if
  something looks wrong, the scope is leaking).
- Any verification fails twice after a reasonable fix attempt.
- The fix appears to require editing a base rule, a token definition, or any
  `.go` component file — that means the scope assumption is wrong; stop.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **The real invariant**: any *new* panel surface or parchment material must pair
  its `--surface` background with `--ink` text. This fix patches the existing
  panel column; if a future layout introduces a second parchment region that sets
  a background but not a color, the same dark-mode bug returns. Consider this when
  reviewing any new `background: var(--surface)` rule — it should be accompanied
  by `color: var(--ink)` (or live inside `.parch`/`.card`/`.kcard`, which already
  pair them).
- **What a reviewer should scrutinize**: that the override stayed *panel-scoped*
  (under `html.app #panel.app-panel`) and did not touch the global `h1,h2,h3` /
  `.empty-title` / `.tl-label` base rules — those are still needed in light-on-dark
  contexts (storybook, page background).
- **Deferred (not in this plan)**: `--muted` secondary text on parchment in dark
  mode (`#8a7355` on `#e8d9ae`) is ~3:1 contrast — fine for large/secondary text,
  short of WCAG AA 4.5:1 for body copy. The owner did not flag it (it's not
  "white"), so it's out of scope here. If an accessibility pass later wants AA
  everywhere, the fix is a darker on-parchment muted token (e.g. reuse
  `--ink-muted: #6f5e3c`) applied panel-scoped the same way — a separate change.
- This change is CSS + a guard test only; it does not interact with the turn
  pipeline, Datastar wiring, or any Go component logic.
