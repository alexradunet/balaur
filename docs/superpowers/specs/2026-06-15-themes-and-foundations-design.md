# Themes (Hearth / Forest / Dungeon) + Foundation Pages — Design

**Status:** Design approved (pending written-spec review) · **Date:** 2026-06-15

## Context

The Claude Design export (`Balaur_Design/Balaur Storybook.dc.html`) defines a
three-theme palette system — **Hearthwood**, **Forest**, **Dungeon** — each with a
day/night mode, plus a **Foundations** section of guideline pages (Colors,
Typography, Materials, Chrome). The repo currently ships only Hearthwood via
`light-dark()` + a `.light`/`.dark` mode toggle, and has no foundation pages.

This spec migrates the theme system **app-wide** (the whole Balaur UI re-tints, not
just the storybook) and adds the foundation pages. Locked decisions:

1. **App-wide themes** — the real app chrome (and storybook) gets theme + mode
   controls; chat/tasks/memory/etc. all re-tint.
2. **Theme foundation first** (Slice 1), then foundation pages (Slice 2).
3. **Flat dither default** — the wood chrome drops its plank lines: `--wood-planks:
   none`. The interactive "Chrome" texture-explorer page is **dropped** (the texture
   choice is settled). Foundation pages = **Colors, Typography, Materials** (3).

## Key architectural finding

The repo's existing `:root` `light-dark()` values **are** the Hearthwood palette for
both modes (verified: `--bg: light-dark(#efe2bd, #140c06)` == export hearthwood
light `#efe2bd` / dark `#140c06`; same for `--gold`, `--chrome`, …). Therefore:

- **Hearthwood = the untouched base `:root`.** No new CSS.
- **Mode axis (`.light`/`.dark`) is unchanged** — the existing toggle, no-flash
  script, `color-scheme` rules (basm.css:76-77) all stay.
- **Forest and Dungeon are pure additive override blocks**: `:root.theme-forest`
  (dark) + `:root.theme-forest.light` + `:root.theme-dungeon` +
  `:root.theme-dungeon.light`, each overriding only the ~23 theme-varying tokens
  (page/wood/text/accents/`--bevel-light`/`--grain-warm`) with flat hex from the
  export. Parchment/ink/material constants (`--surface*`, `--parch-edge`, `--ink*`,
  `--*-ink`, `--grain-ink`) are **not** redefined — parchment stays constant across
  themes (this is intentional in the export).

This makes the migration low-blast-radius: two additive CSS blocks + a palette axis
on the switcher, not a foundation rewrite.

> **Token-layer note:** these theme blocks are the **token definition layer** — raw
> hex is the source of truth here, exactly like the existing `:root`/`light-dark()`
> block. The "no raw hex in declarations" convention applies to **component** CSS,
> not to the `:root`/theme token layer. The `css_tokens_test` (which only guards
> stale tokens) is unaffected.

---

## Slice 1 — Theme foundation (app-wide)

### 1a. DOM contract

`<html>` carries two orthogonal axes:
- **Palette:** exactly one of `theme-hearthwood` / `theme-forest` / `theme-dungeon`.
- **Mode:** `light` or `dark` (existing classes; dark is the practical default).

Cycle order (export): `hearthwood → forest → dungeon → hearthwood`. Defaults:
palette `hearthwood`, mode `dark`.

### 1b. CSS — `internal/web/assets/static/basm.css`

**Change 1 — flat dither default.** Line 72: `--wood-planks: repeating-linear-
gradient(...)` → `--wood-planks: none;` (keep the `/* @kind other */` annotation).

**Change 2 — append the Forest + Dungeon override blocks** at the END of basm.css.
Hearthwood needs none (it is the base). Map the export's `.is-light` →
the repo's `.light`. Exact blocks (raw hex is the token source of truth):

```css

/* ── Themes — Forest & Dungeon palettes (Hearthwood is the base :root) ───────
   Additive overrides of the theme-varying tokens only; parchment/ink constants
   stay. Mode is the orthogonal .light/.dark axis (unchanged). */
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

Specificity is correct: `.theme-forest.light` (2 classes) beats both `.theme-forest`
(1) and the base. The `.light`/`.dark` mode classes still drive `color-scheme`
(basm.css:76-77), so the constants (`--surface` etc., still `light-dark()`) follow
mode within any theme.

### 1c. Switcher JS — `internal/web/assets/static/basm.js`

Keep `basmToggleTheme` (mode) and `basmUpdateThemeButtons` exactly as-is. **Add** a
palette axis:

```js
// ── Theme palette cycle (hearthwood → forest → dungeon) ────────────
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

Persistence: `basm-palette` localStorage (the existing `basm-theme` keeps holding
the mode — light/dark — unchanged).

### 1d. No-flash script — `internal/ui/shell/shell.go`

`noFlashScript` must also apply the saved palette before paint, defaulting to
`theme-hearthwood`, **and must always set an explicit mode class** (defaulting to
`dark`). The explicit-mode default is load-bearing: the Forest/Dungeon flat-hex
accents are mode-independent, so without a `.light`/`.dark` class a non-Hearthwood
palette would pair its dark accents with OS-light parchment (`--surface` etc. are
`light-dark()`, following `color-scheme`) — a tone mismatch. Hearthwood is immune
(all its tokens are `light-dark()` and move together), but Forest/Dungeon are not.
New body (adds the palette line; defaults mode to `dark`; keeps dock logic):

```js
(function(){var d=document.documentElement;d.classList.add(localStorage.getItem('basm-theme')||'dark');d.classList.add('theme-'+(localStorage.getItem('basm-palette')||'hearthwood'));if(localStorage.getItem('basm-dock-full')==='1')d.classList.add('dock-full');var w=parseInt(localStorage.getItem('basm-dock-w'),10);if(w>=280&&w<=720)d.style.setProperty('--sidebar-w',w+'px');}());
```

> **Behavior change:** previously a fresh visitor with no `basm-theme` got the OS-
> preferred mode (base `color-scheme: dark light`); now they get explicit `dark`.
> This matches the export's default-dark and is what guarantees the palette/mode
> axes can never disagree.

### 1e. Chrome controls — app topbar + storybook sidebar footer

The app `topbar` (shell.go) and the storybook sidebar footer (`internal/web/
storybook.go` `sidebarFor`) each currently render one `.theme-toggle` (mode) button.
**Add** a `.theme-cycle` button beside it:

```go
h.Button(h.Class("theme-cycle"), h.Type("button"),
    g.Attr("onclick", "basmCycleTheme()"),
    h.Title("Cycle theme"), h.Aria("label", "Cycle theme"),
    g.Text("Hearth")),
```

The `basmUpdatePaletteButtons` DOMContentLoaded hook corrects the label to the
persisted palette on load.

**CSS — group `.theme-cycle` with `.theme-toggle`** so the two pills are byte-
identical and cannot drift. Edit the two existing selectors in basm.css (lines
~919 and ~933) in place:
- `.theme-toggle {` → `.theme-toggle, .theme-cycle {`
- `.theme-toggle:hover {` → `.theme-toggle:hover, .theme-cycle:hover {`

No new declarations — `.theme-cycle` reuses the existing mono-pill rule (font-mono,
1px `--chrome-fg` border, `--radius`, `--chrome-fg` text, `margin-left:10px`). This
is the only basm.css edit for the buttons.

### 1f. Slice-1 verification

- `go test ./...`, `go vet`, `CGO_ENABLED=0 go build` green.
- Screenshot the storybook (e.g. `/storybook/button` + `/storybook/card`) in **all
  3 themes × 2 modes** (6 shots) by setting `localStorage` palette/mode (or toggling)
  — confirm atoms re-tint: Forest greens, Dungeon cold-steel/violet, parchment
  constant. Content-assert the route first (stale-debug-binary guard).
- Confirm the live app home (`/`) re-tints too (app-wide), not just the storybook.
- Wood chrome shows flat dither (no plank lines), grain intact.

---

## Slice 2 — Foundation pages (storybook stories)

Three new stories under a new **Foundations** group (registry: `{id, "Foundations",
name, canvas}`), built in `internal/feature/storybook`. They compose existing atoms
and re-tint live (every swatch reads `var(--token)`).

### 2a. Colors — `f-colors` → story `colors`

Four token groups, each a `ui.SectionLabel` heading (the export's heading markup is
byte-identical to `.section-label`) over a swatch grid (`repeat(auto-fill,
minmax(108px,1fr))`). A swatch = a 62px chip (`background: var(--token)`, 2px
`--outline-2`, `--bevel-in`) + mono label + mono var-name. Groups:
- **Page & wood:** `--bg --chrome --chrome-2 --chrome-fg --outline-2`
- **Parchment & ink:** `--surface --surface-2 --surface-3 --parch-edge --ink --ink-muted`
- **Accents:** `--gold --gold-deep --ember --ember-deep --ember-red --teal --teal-deep --folkred --indigo --violet --good`
- **Text on page:** `--fg --fg-strong --muted --hair`

New CSS: `.swatch`/`.swatch-chip`/`.swatch-label`/`.swatch-name`/`.swatch-grid`
(tokenized — the chip background is `var(--token)` set via an inline `--sw` custom
property per swatch, same idiom as `ui.SectionLabel`'s `--sl-accent`).

### 2b. Typography — `f-type` → story `typography`

Four role cards (parchment material: `.fdn-card`) — Display (Jersey 15, "A new head
wakes"), Pixel (Silkscreen, "BALAUR"), Body (Piazzolla, "I shall weigh the matter."),
Mono (JetBrains, "tool · search · used ×3") — each a baseline row: 84px mono role
label + the sample (its font token) + right-aligned mono note. Then a "Scale" card:
5 rows of "The hearth is lit" at 36/28/22/17/13 px. New CSS: `.fdn-card`,
`.type-role`, `.type-scale-row`.

### 2c. Materials — `f-materials` → story `materials`

A responsive grid (`repeat(auto-fit, minmax(min(230px,100%),1fr))`) of 6 specimen
tiles, each a 96px swatch + mono title + mono description: **Parchment**, **Wood
chrome**, **Carved well**, **Ornate parchment**, **Folk band** (the
`repeating-linear-gradient(135deg,...)` stripe — composes `ui.FolkBand` if it
matches, else inline), **Stitch · square corners** (the dashed `ui.Stitch` line).
Reuse existing material/bevel tokens; new CSS only for the tile layout (`.mat-grid`,
`.mat-tile`, `.mat-swatch`).

### 2d. Slice-2 verification

- Three stories registered; `/storybook/colors|typography|materials` render 200
  (content-assert the component class, not just status).
- Colors swatches re-tint when the palette is switched (the payoff of Slice 1).
- Screenshot Colors in Hearth + Forest + Dungeon.

---

## Testing strategy

- **Slice 1** is mostly CSS/JS (no Go logic) — covered by build + the existing
  `css_tokens_test` (still passes; stale-token guard unaffected) + manual 6-way
  screenshot. Add a small assertion to `css_tokens_test` that `.theme-forest` and
  `.theme-dungeon` blocks exist and `--wood-planks: none;` is set (guards regression).
- **No-flash guard:** the existing `internal/ui/shell/shell_test.go` asserts the
  `noFlashScript` contains `localStorage.getItem('basm-theme')` (still true). **Add**
  an assertion that it also contains `localStorage.getItem('basm-palette')` and
  `'theme-'` — the palette-before-paint line is the one new before-paint behavior and
  must not be silently dropped by a future edit.
- **Slice 2** foundation stories get golden tests (each renders its group headings +
  representative swatch/role/tile markup) like the atom stories, plus the registry
  test covers the new story IDs automatically.

## Known limitations / deferred

- **No theme picker UI beyond the cycle button** — cycling steps Hearth→Forest→
  Dungeon. A dropdown/3-button picker (as in the export sidebar footer) is a later
  polish.
- **Chrome texture explorer dropped** — flat dither is the fixed default. If a wood
  texture ever becomes a real preference, it'd be a separate persisted setting.
- Foundation pages are documentation; they don't gate behavior. The export's
  Overview "Themes" section + the atomic Atoms/Molecules/Organisms nav re-grouping
  are out of scope (we keep our current storybook group taxonomy + add Foundations).
- No automated visual-regression — theme correctness is verified by screenshot.
