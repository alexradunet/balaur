# Themes — Slice 1: Theme Foundation (app-wide) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Hearthwood / Forest / Dungeon a real app-wide theme system: add Forest+Dungeon palette override blocks (Hearthwood is the untouched base), flat-dither wood default, a palette-cycle switcher orthogonal to the existing light/dark mode, and the controls in the app topbar + storybook chrome.

**Architecture:** Themes are a **palette axis** (`theme-{name}` class on `<html>`) orthogonal to the existing **mode axis** (`.light`/`.dark`). Hearthwood = the existing `light-dark()` base `:root` (no new CSS). Forest/Dungeon = additive flat-hex override blocks of the ~23 theme-varying tokens. Mode mechanism (`color-scheme`, no-flash, toggle) is unchanged except the no-flash script now also sets the palette class and an explicit default mode.

**Tech Stack:** vanilla `basm.css` + `basm.js` (no build step), Go (gomponents shell).

**Spec:** `docs/superpowers/specs/2026-06-15-themes-and-foundations-design.md`. Hex values + mechanism adversarially verified.

**Conventions:** The theme blocks are the **token-definition layer** — raw hex is the source of truth (like the existing `:root`/`light-dark()`), so the "no raw hex" rule does NOT apply here. Theme blocks append at the END of basm.css. After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows any file other than the task's own as modified, do NOT stage it — `git checkout --` it.

---

## File Structure

- **Modify** `internal/web/assets/static/basm.css` — `--wood-planks: none`; append Forest+Dungeon blocks; group `.theme-toggle, .theme-cycle`.
- **Modify** `internal/web/assets/css_tokens_test.go` — guard the new blocks.
- **Modify** `internal/web/assets/static/basm.js` — add `basmCycleTheme` + `basmUpdatePaletteButtons`.
- **Modify** `internal/ui/shell/shell.go` — no-flash script (palette + default mode) + topbar `.theme-cycle` button.
- **Modify** `internal/ui/shell/shell_test.go` — assert the palette no-flash line.
- **Modify** `internal/web/storybook.go` — add `.theme-cycle` button to the sidebar footer.

---

## Task 1: Theme CSS — flat dither, Forest + Dungeon blocks, `.theme-cycle` grouping

**Files:** Modify `internal/web/assets/static/basm.css`, `internal/web/assets/css_tokens_test.go`.

- [ ] **Step 1: Write the failing guard test**

In `internal/web/assets/css_tokens_test.go`, add this test (keep the existing `TestNoUndefinedHearthwoodTokens`):
```go
// TestThemePaletteBlocks guards Slice-1: flat-dither wood default and the Forest
// + Dungeon palette override blocks (Hearthwood is the base :root, no block).
func TestThemePaletteBlocks(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)
	for _, want := range []string{
		"--wood-planks: none;",
		":root.theme-forest {",
		":root.theme-forest.light {",
		":root.theme-dungeon {",
		":root.theme-dungeon.light {",
		".theme-toggle, .theme-cycle {",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("basm.css missing theme block marker: %q", want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/web/assets/ -run TestThemePaletteBlocks -v` — Expected: FAIL (markers absent).

- [ ] **Step 3: Flat-dither default**

In `internal/web/assets/static/basm.css`, change the `--wood-planks` definition (line ~72) from the `repeating-linear-gradient(...)` value to:
```css
  --wood-planks: none; /* @kind other */
```
(flat dither — the wood chrome keeps `--grain-warm` but drops the plank lines.)

- [ ] **Step 4: Group `.theme-cycle` with `.theme-toggle`**

In `internal/web/assets/static/basm.css`, edit the two existing selectors in place:
- `.theme-toggle {` (line ~919) → `.theme-toggle, .theme-cycle {`
- `.theme-toggle:hover {` (line ~933) → `.theme-toggle:hover, .theme-cycle:hover {`

(No new declarations — the cycle button reuses the mono-pill rule.)

- [ ] **Step 5: Append the Forest + Dungeon palette blocks**

Append exactly this at the very END of `internal/web/assets/static/basm.css`:
```css

/* ── Themes — Forest & Dungeon palettes (Hearthwood is the base :root) ───────
   Additive overrides of the theme-varying tokens only; parchment/ink constants
   stay constant across themes. Mode is the orthogonal .light/.dark axis. This is
   the token-definition layer — raw hex is the source of truth here. */
:root.theme-forest {
  --bg:#0b140e; --chrome:#16271a; --chrome-2:#0d1a11; --chrome-fg:#a4bd94;
  --fg:#bcccab; --fg-strong:#dfead0; --muted:#80936f; --smoke:#586b4d; --hair:#21321c; --outline-2:#050c07;
  --gold:#e6c652; --gold-deep:#8c7a26; --ember:#ff9a45; --ember-deep:#7a4a15; --ember-red:#e0584d;
  --teal:#46d8b4; --teal-deep:#1a9a7e; --folkred:#d96b3c; --indigo:#a6c2ea; --violet:#bd8cf2; --good:#93d56e; --steel:#8a9c74;
  --bevel-light:rgba(190,255,170,.16);
  --grain-warm: repeating-conic-gradient(rgba(150,255,180,.025) 0% 25%, transparent 0% 50%);
}
:root.theme-forest.light {
  --bg:#e4eccf; --chrome:#243d28; --chrome-2:#16271a; --chrome-fg:#cfe0bd;
  --fg:#2e3d26; --fg-strong:#1a2614; --muted:#5e7050; --smoke:#7d9072; --hair:#bcd0a8; --outline-2:#0d1a11;
  --gold:#6f7a18; --gold-deep:#5a6312; --ember:#b8541c; --ember-deep:#7a4a15; --ember-red:#a83820;
  --teal:#0d7a5c; --teal-deep:#0a5a44; --folkred:#8a4320; --indigo:#3d54a0; --violet:#6d3bb8; --good:#3f7a2f; --steel:#5e7050;
  --bevel-light:rgba(220,255,200,.34);
  --grain-warm: repeating-conic-gradient(rgba(40,90,40,.04) 0% 25%, transparent 0% 50%);
}
:root.theme-dungeon {
  --bg:#0c0d12; --chrome:#20232c; --chrome-2:#15161d; --chrome-fg:#a0a3b6;
  --fg:#b7b9c8; --fg-strong:#e3e5f0; --muted:#7c7f93; --smoke:#565a6c; --hair:#2a2d39; --outline-2:#050609;
  --gold:#d8b552; --gold-deep:#8a6f28; --ember:#ff6a38; --ember-deep:#8a3014; --ember-red:#e5484d;
  --teal:#5bd2da; --teal-deep:#2a98a0; --folkred:#d84e3c; --indigo:#aab6f2; --violet:#cf90ff; --good:#7fcf8a; --steel:#8a8da2;
  --bevel-light:rgba(180,200,255,.16);
  --grain-warm: repeating-conic-gradient(rgba(160,180,255,.028) 0% 25%, transparent 0% 50%);
}
:root.theme-dungeon.light {
  --bg:#dcdde4; --chrome:#2c2f38; --chrome-2:#20232c; --chrome-fg:#cdd0de;
  --fg:#2a2d38; --fg-strong:#16181f; --muted:#5c5f70; --smoke:#7c7f93; --hair:#c2c4d0; --outline-2:#1a1c24;
  --gold:#7a6520; --gold-deep:#5e4e18; --ember:#c2461c; --ember-deep:#8a3014; --ember-red:#a8201f;
  --teal:#0d6e7a; --teal-deep:#0a545c; --folkred:#a8392c; --indigo:#3d54a0; --violet:#6d3bb8; --good:#3f6f4a; --steel:#5c5f70;
  --bevel-light:rgba(40,60,110,.20);
  --grain-warm: repeating-conic-gradient(rgba(40,50,90,.045) 0% 25%, transparent 0% 50%);
}
```

- [ ] **Step 6: Run to verify it passes + full suite + commit**

Run:
```bash
cd /home/alex/Projects/balaur
go test ./internal/web/assets/ -run 'TestThemePaletteBlocks|TestNoUndefinedHearthwoodTokens' -v
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
git status --short
```
Expected: both tests PASS, full suite green, build clean. If `git status --short` shows any file other than `internal/web/assets/static/basm.css` and `internal/web/assets/css_tokens_test.go`, do NOT stage it — `git checkout -- <file>`. Then:
```bash
git add internal/web/assets/static/basm.css internal/web/assets/css_tokens_test.go
git commit -m "$(printf 'feat(css): Forest + Dungeon theme palettes + flat-dither wood default\n\nHearthwood stays the base :root; Forest/Dungeon are additive flat-hex override\nblocks of the theme-varying tokens. --wood-planks: none (flat dither). Group\n.theme-cycle with .theme-toggle. Guarded by css_tokens_test.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: Palette-cycle JS

**Files:** Modify `internal/web/assets/static/basm.js`.

- [ ] **Step 1: Add the palette cycle**

In `internal/web/assets/static/basm.js`, immediately AFTER the `basmUpdateThemeButtons` block + its `document.addEventListener('DOMContentLoaded', basmUpdateThemeButtons);` line (around line 32), insert:
```js

// ── Theme palette cycle (hearthwood → forest → dungeon) ────────────
// Orthogonal to light/dark mode (basmToggleTheme). The <head> no-flash
// script applies the saved palette before paint; this handles the cycle.
window.basmCycleTheme = function () {
  var order = ['hearthwood', 'forest', 'dungeon'];
  var d = document.documentElement;
  var cur = order.find(function (t) { return d.classList.contains('theme-' + t); }) || 'hearthwood';
  var next = order[(order.indexOf(cur) + 1) % order.length];
  d.classList.remove('theme-hearthwood', 'theme-forest', 'theme-dungeon');
  d.classList.add('theme-' + next);
  localStorage.setItem('basm-palette', next);
  basmUpdatePaletteButtons();
};

function basmUpdatePaletteButtons() {
  var order = ['hearthwood', 'forest', 'dungeon'];
  var labels = { hearthwood: 'Hearth', forest: 'Forest', dungeon: 'Dungeon' };
  var d = document.documentElement;
  var cur = order.find(function (t) { return d.classList.contains('theme-' + t); }) || 'hearthwood';
  document.querySelectorAll('.theme-cycle').forEach(function (btn) {
    btn.textContent = labels[cur];
    btn.title = 'Cycle theme (now ' + labels[cur] + ')';
  });
}
document.addEventListener('DOMContentLoaded', basmUpdatePaletteButtons);
```

- [ ] **Step 2: Verify + commit**

JS is not compiled; verify it parses by building the app (assets embed) and a quick node syntax check if available:
```bash
cd /home/alex/Projects/balaur
node --check internal/web/assets/static/basm.js 2>&1 || echo "(node not available — skip; verified at runtime)"
CGO_ENABLED=0 go build ./... && echo BUILD_OK
git status --short
```
Expected: parses (or node absent), build clean. Stage only basm.js:
```bash
git add internal/web/assets/static/basm.js
git commit -m "$(printf 'feat(js): basmCycleTheme palette cycle (hearthwood/forest/dungeon)\n\nOrthogonal to the light/dark mode toggle; persists basm-palette and syncs\n.theme-cycle button labels.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: Go wiring — no-flash script, topbar + storybook buttons

**Files:** Modify `internal/ui/shell/shell.go`, `internal/ui/shell/shell_test.go`, `internal/web/storybook.go`.

- [ ] **Step 1: Write the failing no-flash test**

`internal/ui/shell/shell_test.go` is `package shell_test` (external), so it can't
reference the unexported `noFlashScript` const — it asserts on the **rendered page**.
The existing `TestPage` has a `want` slice (lines ~25-35) that already includes
`` `localStorage.getItem('basm-theme')` ``. Add two entries to that same slice,
right after the `basm-theme` line:
```go
		`localStorage.getItem('basm-palette')`,
		`'theme-'`,
```
So the slice asserts the rendered page's inline no-flash script applies the palette
before paint. (No new test function; `strings` is already imported.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/shell/ -run TestPage -v` — Expected: FAIL (the rendered
no-flash script doesn't yet contain `basm-palette`).

- [ ] **Step 3: Update the no-flash script**

In `internal/ui/shell/shell.go`, replace the `noFlashScript` const body with (adds the palette class + defaults mode to `dark`):
```go
const noFlashScript = `(function(){var d=document.documentElement;d.classList.add(localStorage.getItem('basm-theme')||'dark');d.classList.add('theme-'+(localStorage.getItem('basm-palette')||'hearthwood'));if(localStorage.getItem('basm-dock-full')==='1')d.classList.add('dock-full');var w=parseInt(localStorage.getItem('basm-dock-w'),10);if(w>=280&&w<=720)d.style.setProperty('--sidebar-w',w+'px');}());`
```

- [ ] **Step 4: Add the topbar `.theme-cycle` button**

In `internal/ui/shell/shell.go`, in `topbar(...)`, immediately BEFORE the existing `h.Button(h.Class("theme-toggle"), ...)` add:
```go
		h.Button(h.Class("theme-cycle"), h.Type("button"),
			g.Attr("onclick", "basmCycleTheme()"),
			h.Title("Cycle theme"), h.Aria("label", "Cycle theme"),
			g.Text("Hearth"),
		),
```
(The `basmUpdatePaletteButtons` DOMContentLoaded hook corrects the label to the saved palette on load.)

- [ ] **Step 5: Add the storybook sidebar `.theme-cycle` button**

In `internal/web/storybook.go`, `sidebarFor(...)`, the `Footer` currently is a single `hh.Button(hh.Class("theme-toggle"), ...)`. Wrap both buttons in a group so the footer has the cycle + mode buttons. Replace the `Footer:` value with:
```go
		Footer: g.Group([]g.Node{
			hh.Button(hh.Class("theme-cycle"), hh.Type("button"),
				g.Attr("onclick", "basmCycleTheme()"),
				hh.Title("Cycle theme"), hh.Aria("label", "Cycle theme"),
				g.Text("Hearth")),
			hh.Button(hh.Class("theme-toggle"), hh.Type("button"),
				g.Attr("onclick", "basmToggleTheme()"),
				hh.Title("Toggle light/dark mode"), hh.Aria("label", "Toggle light/dark mode"),
				g.Text("◑")),
		}),
```
(Preserve the existing import aliases — `g` for gomponents, `hh` for html in this file.)

- [ ] **Step 6: Run to verify it passes + full suite + commit**

Run:
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/shell/ -v 2>&1 | grep -E "PASS|FAIL|ok" | head
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
git status --short
```
Expected: shell tests PASS, full suite green, build+vet clean. Stage only the three files:
```bash
git add internal/ui/shell/shell.go internal/ui/shell/shell_test.go internal/web/storybook.go
git commit -m "$(printf 'feat(shell): palette-aware no-flash script + theme-cycle button\n\nNo-flash script applies the saved palette (default hearthwood) and an explicit\ndefault mode (dark, prevents Forest/Dungeon FOUC). Theme-cycle button added to\nthe app topbar and storybook sidebar footer.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification (controller does this — visual, not a subagent)

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — all green.
- [ ] Build a branch binary, free port 8090 of any stale `__debug_bin`, run the server.
- [ ] Content-assert + screenshot **6 combinations** of a representative storybook page (e.g. `/storybook/card` and the app home `/`): set `localStorage` `basm-palette` ∈ {hearthwood,forest,dungeon} × `basm-theme` ∈ {dark,light} (via `evaluate`/query params or a tiny JS injection), reload, screenshot. Confirm:
  - Hearthwood = oak/parchment/gold (unchanged from today).
  - Forest = mossy greens, dusk; green grain.
  - Dungeon = cold stone/steel, torch ember, violet; blue grain.
  - Parchment surface stays constant across all three (only page/wood/accents shift).
  - Wood chrome shows flat dither (no horizontal plank lines), grain intact.
  - Light mode of each reads correctly (ink-weight accents on light page).
- [ ] Confirm the **live app** (`/`) re-tints (app-wide), not just the storybook.
- [ ] The topbar + storybook sidebar show BOTH a `Hearth/Forest/Dungeon` cycle button and the `◑/☼` mode toggle, and cycling/toggling works + persists across reload.

## What this delivers / what's next

**Delivered:** app-wide 3-theme palette system on an orthogonal mode axis; flat-dither wood; palette-cycle + mode controls in app + storybook; no-flash palette application.

**Next (Slice 2):** the Colors / Typography / Materials foundation pages as storybook stories under a new "Foundations" group — Colors being the live theme showcase.
