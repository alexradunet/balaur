# Plan 108: Collapse the CSS to a single Hearthwood dark theme (remove light mode + forest/dungeon palettes)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If any
> STOP condition occurs, stop and report — do not improvise. When done, update
> the status row for this plan in `plans/readme.md` — unless a reviewer
> dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat b2c312a..HEAD -- internal/web/assets/static/basm.css internal/web/assets/css_tokens_test.go DESIGN.md`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (touches every color token; a wrong collapse changes the look)
- **Depends on**: none
- **Category**: tech-debt (theming simplification, owner-requested)
- **Planned at**: commit `b2c312a`, 2026-06-18
- **Issue**: —

## Why this matters

The owner wants **one fully functional theme — Hearthwood dark — instead of
managing both a light and a dark mode plus two extra palettes (forest, dungeon)**.
Today the CSS carries every color twice (a `light-dark(lightValue, darkValue)`
pair per token), flips via `color-scheme` + a JS toggle, and defines two
additional `:root.theme-*` palette blocks. That is four palettes × two modes to
keep correct on every change.

This plan does the **CSS half**: collapse every token to its single dark value,
lock `color-scheme: dark`, and delete the light-mode overrides and the
forest/dungeon palette blocks. The rendered result must be **pixel-identical to
today's Hearthwood dark mode** — we are removing machinery, not restyling.

A follow-up (plan 109) removes the now-inert theme/mode UI and JS (the toggle
button, the palette pickers, the no-flash bootstrap's theme logic). This plan is
intentionally first and standalone: after it lands, `color-scheme: dark` is
fixed, so the leftover toggle/palette buttons become harmless no-ops (clicking
adds a `.light`/`.theme-forest` class that no longer has any CSS to apply) and
the app renders a single theme. Do **not** touch the JS or the button markup
here.

## How CSS `light-dark()` works (so you collapse correctly)

`light-dark(A, B)` resolves to **A** when the element's `color-scheme` is `light`
and **B** when it is `dark`. Today `:root` has `color-scheme: dark light` and a JS
toggle adds `.light`/`.dark`. We are making dark permanent. So:

- **The dark value is always the SECOND argument.** Collapsing `light-dark(A, B)`
  means replacing the whole call with **`B`** (keep the second arg, drop the
  first and the wrapper).
- Setting `color-scheme: dark` (no `light`) makes every remaining `light-dark()`
  (if any were missed) resolve to its dark value too — but the done criteria
  require **zero** `light-dark(` remain.

## Current state

Files in scope:

- `internal/web/assets/static/basm.css` — the token + theme source. All CSS edits here.
- `internal/web/assets/css_tokens_test.go` — CSS-invariant tests; update `TestThemePaletteBlocks`.
- `DESIGN.md` — the design reference; update its "Color tokens" / theming section so it stops describing a dual-mode `light-dark()` system.

### The token block (`basm.css` lines 18–76) — every `light-dark()` to collapse

```css
:root {
  color-scheme: dark light;

  /* Page & wood chrome */
  --bg:        light-dark(#efe2bd, #140c06);
  --chrome:    light-dark(#3a2210, #2a1709);  /* topbar, chatbar, tool rows, tags */
  --chrome-2:  light-dark(#2c1a0c, #1d0f06);  /* deeper wood inset */
  --chrome-fg: light-dark(#d6bb92, #b59872);  /* text on wood */

  /* Parchment content panels (constant across modes) */
  --surface:    light-dark(#f4e9c4, #e8d9ae);
  --surface-2:  light-dark(#e4d49e, #d6c188);
  --surface-3:  light-dark(#d2bf82, #c4ab74);
  --parch-edge: light-dark(#9a7f4c, #8a6f3c);

  /* Text */
  --fg:        light-dark(#3a2c18, #c9b894);  /* body on page bg */
  --fg-strong: light-dark(#241a0c, #ecdcb2);  /* headings on page bg */
  --muted:     light-dark(#7a6644, #8a7355);
  --smoke:     light-dark(#8f7a52, #6b5639);
  --hair:      light-dark(#c8b488, #3b2a16);
  --outline-2: light-dark(#241708, #120a04);  /* the near-black pixel outline */

  /* Ink — text on parchment (constant) */
  --ink:       #2c2012;
  ... (the --ink*, --on-surface, --portrait-bg, --candle-glow, --bevel-dark,
       --gold-ink, --teal-ink, --indigo-ink, --good-ink lines have NO
       light-dark() — leave them exactly as-is)

  --bevel-light: light-dark(rgba(255,222,166,.30), rgba(255,196,118,.18));

  /* Accents — candlelit on dark, ink-weight on light */
  --gold:       light-dark(#8a6212, #f2c14e);
  --gold-deep:  light-dark(#7a5a14, #a87b24);
  --ember:      light-dark(#d04f12, #ff7a33);
  --ember-deep: light-dark(#7e3210, #8f3a12);
  --ember-red:  light-dark(#a8201f, #e5484d);
  --teal:       light-dark(#0d6e5c, #3ecfb8);
  --teal-deep:  light-dark(#0a574a, #169a82);
  --folkred:    light-dark(#983f20, #e0563b);
  --indigo:     light-dark(#3d54a0, #a8c0f0);
  --indigo-deep: light-dark(#2c3a72, #6f86c8);
  --violet:     light-dark(#6d3bb8, #c084fc);
  --good:       light-dark(#3f6f2f, #7fcf6a);
  --steel:      light-dark(#7a6644, #9b8a6c);
}
```

There are also **two** `light-dark()` uses outside the token block:

```css
/* basm.css lines 2491–2492 — badge foreground flips by scheme */
.badge-gold  { ... --badge-fg: light-dark(var(--surface), var(--ink)); }
.badge-ember { ... --badge-fg: light-dark(var(--surface), var(--ink)); }
```
Their dark value is `var(--ink)`, so both collapse to `var(--ink)`.

`grep -c "light-dark(" internal/web/assets/static/basm.css` currently returns
**31** (30 real calls + 1 mention in the file header comment on line 5).

### The mode overrides (`basm.css` lines 78–80) — delete

```css
/* Manual color-scheme overrides (toggled via JS on <html>) */
:root.light { color-scheme: light; }
:root.dark  { color-scheme: dark;  }
```

### The forest/dungeon palette blocks (`basm.css` lines ~2783–2816) — delete

A comment header followed by four blocks: `:root.theme-forest`,
`:root.theme-forest.light`, `:root.theme-dungeon`, `:root.theme-dungeon.light`.
The header starts:

```css
/* ── Themes — Forest & Dungeon palettes (Hearthwood is the base :root) ───────
   Additive overrides of the theme-varying tokens only; ... */
:root.theme-forest {
  --bg:#0b140e; --chrome:#16271a; ...
}
:root.theme-forest.light { ... }
:root.theme-dungeon { ... }
:root.theme-dungeon.light { ... }
```

Find the exact extent yourself: the region runs from the `/* ── Themes — Forest
& Dungeon palettes` comment line through the closing `}` of the
`:root.theme-dungeon.light` block (the last of the four). Delete the comment
header and all four blocks. The next rule after them must remain untouched.

### The CSS test (`internal/web/assets/css_tokens_test.go`, `TestThemePaletteBlocks`, lines 31–52)

```go
func TestThemePaletteBlocks(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	...
	for _, want := range []string{
		"--wood-planks: none;",
		":root.theme-forest {",
		":root.theme-forest.light {",
		":root.theme-dungeon {",
		":root.theme-dungeon.light {",
		".theme-toggle {",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("basm.css missing theme block marker: %q", want)
		}
	}
}
```

This test currently asserts the forest/dungeon blocks exist — it will fail after
deletion and must be rewritten (Step 5) to assert the single-theme invariants
instead. Note: `.theme-toggle {` (the toggle button's CSS) is still present after
this plan (its removal is plan 109's job), so you may keep asserting it — but the
forest/dungeon/`.light` assertions must go.

### DESIGN.md theming section (lines ~217–260)

Contains a reference token table headed "Reference copy (dark-mode values)" with
`color-scheme: dark light; /* tokens resolve via light-dark() */`, a comment
"Parchment content panels — constant across dark and light", and a prose
paragraph (≈ lines 258–260): *"Theming is standard `light-dark()` with
`color-scheme: dark light` — the OS preference applies automatically; `<html
class="dark">` or `<html class="light">` force a mode. The light theme applies
the same material language at daylight intensity ..."*. This must be updated to
describe a single fixed Hearthwood dark theme (Step 6). Leave the topbar
light/dark-toggle mention (DESIGN.md line ~161) for plan 109 — the toggle still
exists after this plan.

## Commands you will need

| Purpose         | Command                                                   | Expected            |
|-----------------|-----------------------------------------------------------|---------------------|
| CSS-rule tests  | `go test ./internal/web/assets/`                          | `ok`                |
| Full tests      | `go test ./...`                                           | all ok/no test files |
| Vet             | `go vet ./...`                                            | exit 0              |
| CGO-free build  | `CGO_ENABLED=0 go build ./...`                            | exit 0              |
| Format check    | `gofmt -l internal/web/assets/css_tokens_test.go`         | empty output        |
| Whitespace      | `git diff --check`                                        | no output           |

Sandbox note: TLS-intercepting sandbox (Hyperagent) needs the GOPROXY shim — see
`docs/hyperagent-sandbox.md`. There is no CSS build step — `basm.css` is embedded
via `embed.FS`.

## Scope

**In scope** (only files you may modify):
- `internal/web/assets/static/basm.css`
- `internal/web/assets/css_tokens_test.go`
- `DESIGN.md`

**Out of scope** (do NOT touch — these are plan 109):
- `internal/web/assets/static/basm.js` (the `basmToggleTheme`/`basmSetPalette` JS).
- Any `.go` file (the theme-toggle button, the Appearance settings tab, the storybook switcher, the no-flash script in `internal/ui/shell/shell.go`).
- The switcher CSS in `basm.css`: `.theme-toggle`, `.appearance-themes`, `.appearance-theme-btn` (+ its `html.theme-*` highlight rules), `.sb-theme-btn`, `.sb-foot-mode`, `.sb-foot-themes`. Leave them — plan 109 removes them with their markup. (You only delete the `:root.theme-forest/dungeon` *palette* blocks and the `:root.light/.dark` rules.)
- The non-`light-dark` constant tokens (`--ink`, `--ink-muted`, `--on-surface`, `--ink-deep`, `--portrait-bg`, `--candle-glow`, `--gold-ink`, `--teal-ink`, `--indigo-ink`, `--good-ink`, `--bevel-dark`, the `--grain-*`/`--wood-planks` lines). Leave them exactly as-is.

## Git workflow

- Branch: `improve/108-single-hearthwood-dark-theme-css`, based on `main` (`b2c312a`).
- One commit. Conventional message, e.g.
  `refactor(web): collapse CSS to a single Hearthwood dark theme`.
- Match the trailer style of recent commits (`git log -1 --format='%b'`); attribute the model you actually are.
- Do NOT push or open a PR.

## Steps

### Step 1: Lock the color scheme to dark

In `basm.css` line 19, change `color-scheme: dark light;` → `color-scheme: dark;`.

**Verify**: `grep -nE "^\s*color-scheme:" internal/web/assets/static/basm.css` → the `:root` line now reads `color-scheme: dark;` (the only other `color-scheme:` lines are inside `:root.light`/`:root.dark`, which Step 3 deletes).

### Step 2: Collapse every `light-dark(A, B)` to its dark value `B`

For **each** `light-dark(A, B)` occurrence in `basm.css` (the ~28 token lines
22–70 and the 2 badge lines 2491–2492), replace the entire `light-dark(...)` call
with just its **second argument**. Examples:

- `--bg:        light-dark(#efe2bd, #140c06);` → `--bg:        #140c06;`
- `--gold:       light-dark(#8a6212, #f2c14e);` → `--gold:       #f2c14e;`
- `--bevel-light: light-dark(rgba(255,222,166,.30), rgba(255,196,118,.18));` → `--bevel-light: rgba(255,196,118,.18);`
- `.badge-gold  { ... --badge-fg: light-dark(var(--surface), var(--ink)); }` → `... --badge-fg: var(--ink); }`

Do this for all of them. You may keep the existing inline comments; if a comment
is now plainly wrong (e.g. "Accents — candlelit on dark, ink-weight on light",
"constant across modes") you may trim the "on light"/"across modes" wording, but
this is optional and must not change any value.

**Verify**: `grep -c "light-dark(" internal/web/assets/static/basm.css` → **0**
(the header-comment mention on line 5 should also be updated in Step 6, but if you
prefer, removing the word in that comment is what brings this to 0 — the count
must be 0). If any non-zero, you missed a call.

### Step 3: Delete the manual mode overrides

Remove `basm.css` lines 78–80 (the comment `/* Manual color-scheme overrides ... */`
and the `:root.light { ... }` and `:root.dark { ... }` rules).

**Verify**: `grep -cE ":root\.(light|dark)" internal/web/assets/static/basm.css` → **0**.

### Step 4: Delete the forest/dungeon palette blocks

Remove the `/* ── Themes — Forest & Dungeon palettes ... */` comment header and all
four `:root.theme-forest`, `:root.theme-forest.light`, `:root.theme-dungeon`,
`:root.theme-dungeon.light` blocks (≈ lines 2783–2816). Leave the rule that
follows them intact.

**Verify**: `grep -cE ":root\.theme-(forest|dungeon)" internal/web/assets/static/basm.css` → **0**.

### Step 5: Update `TestThemePaletteBlocks`

Rewrite the test (keep the function name) to assert the **single-theme**
invariants instead of the deleted blocks. Replace its `want` loop so it:
- still requires `"--wood-planks: none;"` and `".theme-toggle {"` (both still present),
- additionally requires `"color-scheme: dark;"`,
- and asserts the removed things are **absent**: no `"light-dark("`, no
  `":root.theme-forest"`, no `":root.theme-dungeon"`, no `":root.light"`.

Target shape:

```go
// TestThemePaletteBlocks guards plan 108: Balaur ships a single fixed Hearthwood
// dark theme. color-scheme is locked to dark, every token is a single value (no
// light-dark() pairs), and the light-mode + forest/dungeon palette blocks are
// gone. (.theme-toggle / --wood-planks remain until plan 109 removes the UI.)
func TestThemePaletteBlocks(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)
	for _, want := range []string{"--wood-planks: none;", "color-scheme: dark;"} {
		if !strings.Contains(css, want) {
			t.Errorf("basm.css missing single-theme marker: %q", want)
		}
	}
	for _, gone := range []string{"light-dark(", ":root.light", ":root.theme-forest", ":root.theme-dungeon"} {
		if strings.Contains(css, gone) {
			t.Errorf("basm.css still references removed dual-theme machinery: %q", gone)
		}
	}
}
```

**Verify**: `go test ./internal/web/assets/ -run TestThemePaletteBlocks -v` → PASS.

### Step 6: Update the basm.css header comment + DESIGN.md theming section

- `basm.css` header (lines ~1–16) and the inline comment on line 5
  (`Color scheme strategy: light-dark() + color-scheme ...`): update so they
  describe a single fixed Hearthwood dark theme (no `light-dark()`, no mode
  flip). This also removes the line-5 `light-dark(` mention so the Step-2 count
  stays 0.
- `DESIGN.md` "Color tokens — Hearthwood palette" / theming section (≈ lines
  217–260): change `color-scheme: dark light; /* tokens resolve via light-dark() */`
  to `color-scheme: dark;`, drop the "resolve via light-dark()" note, and rewrite
  the prose paragraph (≈ 258–260) to state Balaur ships **one fixed Hearthwood
  dark theme** — no OS-preference flip, no `light` class, no forest/dungeon
  palettes. Keep it short and factual; do not invent new design claims. Leave the
  topbar toggle mention (≈ line 161) for plan 109.

**Verify**: `grep -c "light-dark" DESIGN.md` → 0 in the theming section you edited
(a stray mention elsewhere is fine to leave if unrelated, but the token table +
theming paragraph must no longer claim a dual-mode system).

### Step 7: Full verification gates

1. `gofmt -l internal/web/assets/css_tokens_test.go` → empty.
2. `go vet ./...` → exit 0.
3. `CGO_ENABLED=0 go build ./...` → exit 0.
4. `go test ./...` → all ok/no test files; no FAIL.
5. `git diff --check` → no output.
6. `git status --porcelain` → only the three in-scope files.

### Step 8: Visual confirmation (best-effort but IMPORTANT for this plan)

This plan must not change the look. If you can run the app:
`CGO_ENABLED=0 go build -o /tmp/balaur-108 . && /tmp/balaur-108 serve --http=127.0.0.1:8096 --dir=<a writable temp dir copied from ./pb_data>`, open
`http://127.0.0.1:8096/`, and confirm it renders in the Hearthwood **dark**
palette by default (deep oak-dark page, parchment panels, candlelit gold) with no
visual regression vs. before. Spot-check a panel (e.g. `/storybook/modelspanel`)
too. Clicking the (still-present) theme toggle should now do **nothing** visible —
that is expected and correct.

If you cannot run a browser, say so explicitly and rely on the CSS analysis +
tests + gates. Do NOT install a model or download anything.

## Test plan

- **Updated test**: `TestThemePaletteBlocks` (Step 5) now guards the single-theme
  invariants (color-scheme dark, no `light-dark(`, no light/forest/dungeon
  blocks).
- No new component behavior, so no new render tests here. Plan 109 updates the
  markup/JS tests.
- **Verification**: `go test ./...` → no package regresses (a wrong collapse that
  produced invalid CSS wouldn't fail Go tests, which is why Step 8's visual check
  matters — flag if you couldn't run it).

## Done criteria

ALL must hold:

- [ ] `grep -c "light-dark(" internal/web/assets/static/basm.css` → `0`.
- [ ] `grep -cE ":root\.(light|dark)" internal/web/assets/static/basm.css` → `0`.
- [ ] `grep -cE ":root\.theme-(forest|dungeon)" internal/web/assets/static/basm.css` → `0`.
- [ ] `grep -n "color-scheme:" internal/web/assets/static/basm.css` shows exactly one line: `:root { ... color-scheme: dark; ... }` (the `:root.light/.dark` color-scheme lines are gone).
- [ ] The constant tokens (`--ink`, `--surface` family value, `--gold-ink`, etc.) are unchanged in value.
- [ ] `go test ./internal/web/assets/` exits 0; `TestThemePaletteBlocks` updated and passes.
- [ ] `go vet ./...` / `CGO_ENABLED=0 go build ./...` / `go test ./...` all exit 0.
- [ ] `git diff --check` clean; `git status --porcelain` shows only the three in-scope files.
- [ ] `DESIGN.md` theming section no longer describes a dual-mode `light-dark()` system.

## STOP conditions

Stop and report if:

- The drift check shows `basm.css` changed since `b2c312a` and the token block or
  theme blocks no longer match the excerpts.
- You find a `light-dark()` call with a **nested** `light-dark()` or a non-trivial
  expression where "take the second argument" is ambiguous (none are expected —
  all current calls are 2-arg simple). Report it rather than guess.
- The visual check (if you can run it) shows the default render is **light** or
  visibly different from the prior Hearthwood dark — that means a collapse took
  the wrong (first) argument somewhere.
- A verification fails twice after a reasonable fix attempt.
- The change appears to require editing a `.go` file or `basm.js` (it must not —
  that is plan 109).

## Maintenance notes

- After this lands, `color-scheme: dark` is fixed and every token is single-valued.
  **Plan 109** then removes the now-inert theme/mode UI + JS (toggle button,
  palette pickers, the Settings "Appearance" tab, the storybook switcher, and the
  no-flash bootstrap's theme-class logic) plus the orphaned switcher CSS.
- A reviewer should diff the **computed** colors, not just the source: the one
  real risk is a collapse that kept the light (first) value. Step 8's visual check
  is the guard.
- If light mode is ever wanted again, it returns as a deliberate feature, not as
  ambient `light-dark()` debt — that is the point of this change.
