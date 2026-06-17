# Plan 082: Tokenize raw hex, add a z-index tier, snap near-duplicate breakpoints, delete the dead --shadow-hard token, dedup parchment, and collapse free-boards on phones

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 082 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css internal/ui internal/feature web/templates/boards.html` — if any in-scope file changed since this plan was written, compare the "Current state" excerpts to the live code; on mismatch, STOP.

## Status
- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: soft — plans/075-*.md (shares the `:root` layout token block; if 075 already added `--space-*`/width tokens there, append the new tokens after them, do not conflict). Not a hard blocker.
- **Category**: tech-debt
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
`basm.css` is the single source of truth for runtime tokens, and DESIGN.md:19-21 plus SKILL.md:84 are explicit: "templates must reference tokens (`var(--gold)`), never hand-picked hexes" and "Tokenized only: colors are `var(--token)` — no raw hex." Today ~6 raw hex literals, a dead `--shadow-hard` alias, a grab-bag of magic z-indexes, three near-duplicate breakpoint values, ~30 inline copies of the parchment recipe, and a free-board layout that never collapses on phones all sit in the file. None is a visible bug, but each is a place the design system silently drifts. This plan removes the literals behind named tokens, standardizes the z-index stacking into the canonical `--z-*` tier, snaps only the safe breakpoint pairs, and fixes the free-board phone reflow — all value-preserving, as independent commits.

## Current state

`internal/web/assets/static/basm.css` (3218 lines) is ALL of Balaur's CSS; new rule blocks are appended at the END under `Section`/`──` comment banners (SKILL.md:83). The `:root` layout token block lives at lines 154-177:

```
154	:root {
155	  /* Layout */
156	  --radius: 0px;            /* RPG panels are square; rounding is for blobs */
157	  --maxw: 1080px;
158	  --chatbar-space: 210px;
...
167	  --drop-hard: 0 3px 0 rgba(0,0,0,.55);
168	  /* legacy alias (pre-Hearthwood) */
169	  --shadow-hard: 5px 5px 0;
```

The first `:root` (colors) is at lines 23-73; `--ink: #2c2012` (line 42), `--ink-muted` (43), `--chrome-2: light-dark(#2c1a0c, #1d0f06)` (24, theme-VARIED), `--surface`/`--surface-2` (28-29), `--gold` (51), `--ember` (54), `--ember-deep: light-dark(#7e3210, #8f3a12)` (55).

**Theme blocks** at lines 2815-2845 (`:root.theme-forest`, `.theme-forest.light`, `.theme-dungeon`, `.theme-dungeon.light`) override `--gold`/`--ember`/`--chrome-2` etc. per theme, but the comment at 2812-2813 says "parchment/ink constants stay constant across themes" — `--ink`, `--surface` are NOT overridden there. This is the key safety fact for the hex work.

### #15 — raw hex literals (confirmed at HEAD)
- `#1c0d04` at **basm.css:370** (`.btn-primary { color: #1c0d04 }`, sits on `var(--ember)`), **:1052** (`.k-tab-active { color: #1c0d04 }`, on `var(--gold)`), **:1875** (`.settings-nav-active { color: #1c0d04 }`, on `var(--gold)`). This is "darkest ink on a bright accent fill" — DARKER than `--ink (#2c2012)`. Add a NEW constant token, do not reuse `--ink`.
- `#101314` at **basm.css:511** (`.portrait .balaur-avatar { background: #101314 }`) and **:3146** (`.cmsg-portrait` storybook clone, same backdrop). This is a near-black portrait backdrop. Add a constant `--portrait-bg`; do NOT map to `var(--chrome-2)` (that token is theme-varied — see lines 24, 2816/2824/2832/2840 — and would tint the portrait per theme).
- Candle glow `#c88600` at **basm.css:2055-2056**:
  ```
  2054	@keyframes candle-breathe {
  2055	  0%, 100% { filter: drop-shadow(0 0 6px #c8860088); }
  2056	  50%       { filter: drop-shadow(0 0 18px #c88600cc); }
  ```
  `#c88600` + `88`/`cc` alpha. This is a fixed warm-gold glow today. Add a dedicated `--candle-glow: #c88600` constant; do NOT swap to `var(--gold)` (gold flips with light-dark mode AND varies per theme — that would change the glow color).
- Dead fallback `var(--ember-deep, #c0392b)` at **basm.css:2094** (`.btn-danger`). `--ember-deep` is defined at line 55 (always present), so the `#c0392b` fallback is unreachable. Drop the fallback: `color: var(--ember-deep)`.

Existing precedent for tokenized-accent comment lives at basm.css:2501-2502 (badge note "Export hex #1c0d04 -> --ink" — note this comment is slightly inaccurate since `--ink` is `#2c2012`; do NOT rely on it, do NOT edit it, it is out of scope).

### #24 — z-index (confirmed at HEAD)
Page-tier z-indexes that map to the canonical tier:
- **:308** `.topbar { z-index: 5 }` (sticky chrome) → `--z-sticky`
- **:2683** `.sb-crumb { ... z-index: 5 }` (sticky crumb) → `--z-sticky`
- **:2551** `.tooltip-bubble { ... z-index: 30 }` → `--z-tooltip`
- **:2445** `.dock-full #dock { ... z-index: 50 }` → `--z-overlay`
- **:3035** `.sb-topbar { ... z-index: 50 }` (≤920px) → `--z-overlay`
- **:3181** `html.home #dock { z-index: 50 }` → `--z-overlay`
- **:3050** `.sb-backdrop.is-open { ... z-index: 55 }` → `--z-scrim`
- **:3046** `.sb-side { ... z-index: 60 }` (≤920px drawer) → `--z-drawer`

LOCAL component-internal stacking — DO NOT tokenize (these are tiny within-component layering values, not page tiers; mapping them to `--z-sticky:5` would CHANGE behavior):
- **:308** is page tier (already covered above).
- `#dock` base **:2377** `z-index: 4` (the dock rail sits above board content but below overlays — it has no clean tier slot; LEAVE as a literal `4` and add a `/* local: dock rail above content, below overlays */` note OR leave untouched).
- board slot internals: **:2139** (grip `2`), **:2158** (resize `2`), **:2175** (`.dragging 3`), **:2182** (remove `1`), **:2209** (expand `2`), **:2432** (`.dock-grip 3`), **:2920** (`.dayentry-node 1`). LEAVE all of these literal.

The native `<dialog>` top-layer (`.dlg` at basm.css:2774-2778; `internal/ui/dialog.go` renders the native `<dialog>` via `h.Dialog(...)` at line 56) is exempt — it uses the browser top-layer, no `z-index`.

### #25 — dead token (confirmed)
`--shadow-hard: 5px 5px 0;` at **basm.css:169** (with its `/* legacy alias (pre-Hearthwood) */` comment at :168) has **0 consumers** — `grep -rn "shadow-hard" internal/ web/` returns only the definition itself. The live token is `--drop-hard` (:167), used ~9× across the file. Delete lines 168-169. (Its DESIGN.md:308 mirror is plan 085's scope — leave DESIGN.md alone, just note it.)

### Breakpoints (confirmed — SPEC line numbers had minor drift, real list below)
Real `@media (max-width:…)` queries (non-`prefers-reduced-motion`):
- :838 (640), :1840 (**700**), :1987 (640), :1996 (480), :2009 (540), :2263 (**720**), :2332 (860), :2456 (**900**), :3032 (920), :3053 (**520**), :3059 (480), :3060 (**520**), :3123 (640), :3200 (**900**).

Canonical breakpoints (CANONICAL DECISIONS): 540 / 720 / 920. Snap ONLY the near-duplicate pairs (~20px apart):
- **700 → 720**: :1840.
- **900 → 920**: :2456, :3200.
- **520 → 540**: :3053, :3060.

LEAVE 480 (:1996, :3059), 640 (:838, :1987, :3123), 860 (:2332) — they are far from the canon and surfaces may depend on them; document why in the comment block. The existing 540 (:2009), 720 (:2263), 920 (:3032) are already canonical.

### Parch dedup (Pareto subset)
`.parch` class (basm.css:242-249) is the recipe:
```
242	.parch {
243	  background-color: var(--surface);
244	  background-image: var(--grain-ink);
245	  background-size: 4px 4px;
246	  color: var(--ink);
247	  border: 2px solid var(--parch-edge);
248	  box-shadow: var(--parch-bevel);
249	}
```
The four lines (`--surface` + `--grain-ink` + `--parch-edge` border + `--parch-bevel`) are re-declared inline at ~31 selectors (greppable: `var(--parch-bevel)` appears ~15×, `var(--grain-ink)` ~20×). This is a LOW-VALUE, HIGH-RISK-OF-PINNED-OUTPUT-CHANGE task. **Treat it as deferred-by-default**: see Step 6 — only do the cheap CSS-only subset (a shared selector list) IF time permits and it provably doesn't change rendering; otherwise SKIP and note it. Do NOT add `Class("parch")` to gomponents components in this plan (that risks changing storybook pinned output and is out of the value-stable budget).

### R3 — free-board phone collapse (confirmed)
```
2112	/* Free-layout mode: each row unit is 10px; cards use grid-column/grid-row. */
2113	.board-grid-free {
2114	  grid-auto-rows: 10px;
2115	}
...
2263	@media (max-width: 720px) {
2264	  .board-grid { grid-template-columns: 1fr; }
2265	  .board-slot { grid-column: span 1 !important; }
2266	}
```
The ≤720px override resets `grid-column` but NOT `grid-row`, and `.board-grid-free` keeps `grid-auto-rows: 10px`. The inline span comes from `web/templates/boards.html:79`:
```
79	       style="grid-column: {{.X1}} / span {{.W}}; grid-row: {{.Y1}} / span {{.H}}"
```
So on phones, free boards keep their tall explicit row spans and never collapse to a single column in flow order. FIX is **CSS-only** inside the 720 branch: reset `grid-auto-rows` and force `grid-row: auto`.

GUARD: `internal/web/boards_test.go:489-494` asserts the rendered HTML CONTAINS `grid-row` for free boards. Do NOT remove the inline `grid-row` from boards.html — a CSS `!important` override at the breakpoint is the correct, test-safe fix. NOTE: if plan 083 (retire `/boards`) LANDS FIRST and redirects boards away, R3 is moot — before doing Step 5 confirm whether 083 has actually been applied (`grep -n 'boards.html' internal/web/boards.go`: at HEAD it still renders, so R3 still matters); only if `boardsPage` already redirects to `/` should you SKIP Step 5 and note it.

## Commands you will need
| Purpose | Command | Expected |
| Drift stat | `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css internal/ui internal/feature web/templates/boards.html` | empty (no drift) |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test | `go test ./...` | all pass |
| Storybook render | `go test ./internal/feature/storybook/...` | ok |
| Boards test | `go test ./internal/web/...` | ok |
| Format | `gofmt -l <changed.go files>` | empty (only if .go touched) |
| Whitespace | `git diff --check` | no output |
| Dead-token check | `grep -rn "shadow-hard" internal/ web/` | only DESIGN.md / nothing in basm.css |
| Hex residue check | `grep -nE "#1c0d04\|#101314\|#c88600\|#c0392b" internal/web/assets/static/basm.css` | only the new token definitions in `:root` |

If `go` commands fail with "certificate signed by unknown authority" in a sandbox, apply the GOPROXY shim per `docs/hyperagent-sandbox.md` (keep GOSUMDB on).

## Scope
**In scope** (only files you may modify):
- `internal/web/assets/static/basm.css` — primary: hex tokens, `--z-*` tier, breakpoint snaps, dead-token delete, R3, optional parch dedup.

**Out of scope** (do NOT touch):
- `DESIGN.md` — the `--shadow-hard` mirror (:308) and hex docs are plan 085's job.
- The spacing/width/`--pad` tokens — plans 075/076/077 own the `:root` layout additions for those; this plan only ADDS the `--z-*` tier and the new color tokens. If they already added `--space-*` etc., append after, don't conflict.
- The `/boards` retire decision — plan 083.
- `web/templates/boards.html` — R3 is CSS-only; `internal/web/boards_test.go:489-494` pins `grid-row` in the markup, so the template MUST keep emitting it.
- `internal/ui/*` and `internal/feature/*cards` — the parch dedup in this plan is CSS-only; do NOT add `Class("parch")` to components here (defer to keep pinned storybook output stable).
- The badge comment at basm.css:2501-2502 — leave as-is.
- All LOCAL z-index literals listed under #24 (dock rail `4`, board internals `1/2/3`, dayentry-node `1`, dock-grip `3`).

## Git workflow
Branch `improve/082-css-hygiene-scales`. Conventional commits, ONE per sub-task so each is independently revertable:
1. `refactor(web): tokenize raw hex (ink-deep, portrait-bg, candle-glow) in basm.css`
2. `refactor(web): add --z-* tier and replace page-level z-index literals`
3. `refactor(web): delete dead --shadow-hard alias token`
4. `refactor(web): snap near-duplicate breakpoints to 540/720/920 canon`
5. `fix(web): collapse free-layout boards to one column ≤720px`
6. (optional) `refactor(web): dedup parchment recipe behind shared selector`

Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Tokenize the raw hex literals
In the COLOR `:root` block (lines 23-73), add three constant tokens near the existing ink/accent tokens. After line 44 (`--on-surface: #2c2012;`) add:
```
  --ink-deep:   #1c0d04;                       /* darkest ink on a bright gold/ember fill */
  --portrait-bg: #101314;                      /* near-black portrait backdrop (theme-constant) */
  --candle-glow: #c88600;                      /* candle breathe glow (theme-constant warm gold) */
```
Then replace the literals:
- :370 `color: #1c0d04;` → `color: var(--ink-deep);`
- :1052 `color: #1c0d04;` → `color: var(--ink-deep);`
- :1875 `color: #1c0d04;` → `color: var(--ink-deep);`
- :511 `background: #101314;` → `background: var(--portrait-bg);`
- :3146 `background: #101314;` → `background: var(--portrait-bg);` (keep the rest of that rule intact)
- :2055 `drop-shadow(0 0 6px #c8860088)` → `drop-shadow(0 0 6px color-mix(in srgb, var(--candle-glow) 53%, transparent))` — OR, simpler and exactly value-stable, keep alpha hex on the token by NOT using color-mix: define the token WITH no alpha and append alpha in the keyframe is not possible in plain CSS. **SAFEST**: define two tokens instead — keep the keyframe using the raw token plus alpha via `color-mix`. To avoid any rendering change, prefer the literal-preserving form: leave the keyframe as `drop-shadow(0 0 6px #c8860088)` is NOT allowed (hex). Use `color-mix(in srgb, var(--candle-glow) 53%, transparent)` for `88` (0x88=136/255≈53%) and `color-mix(in srgb, var(--candle-glow) 80%, transparent)` for `cc` (0xcc=204/255=80%). Verify visually that the glow is unchanged (Step 1 Verify).
- :2056 `drop-shadow(0 0 18px #c88600cc)` → `drop-shadow(0 0 18px color-mix(in srgb, var(--candle-glow) 80%, transparent))`
- :2094 `var(--ember-deep, #c0392b)` (BOTH occurrences on that line) → `var(--ember-deep)`.

STOP if any of these three literals also appears INSIDE a `:root.theme-*` block (grep first) — that would mean a per-theme override you'd break. (Confirmed at HEAD they do not; re-confirm.)

**Verify**:
- `grep -nE "#1c0d04|#101314|#c88600|#c0392b" internal/web/assets/static/basm.css` → only the three `:root` definitions (lines you just added), nothing in rule bodies/keyframes.
- `CGO_ENABLED=0 go build ./...` → exit 0; `git diff --check` → no output.
- VISUAL (Step has color risk): run the app, screenshot a `.btn-primary`, a `.k-tab-active`, the chat portrait, the candle page (`/candle` or storybook candle story) in BOTH modes and all three themes (`document.documentElement.className='theme-hearthwood dark'`, then `light`, then `theme-forest dark`, `theme-dungeon dark`). The button text, active tab text, portrait backdrop, and candle glow must look identical to pre-change.

### Step 2: Add the `--z-*` tier and replace page-level z-index literals
In the LAYOUT `:root` block (lines 154-177), after `--chatbar-space` (or after the width/space tokens 075/076 added), add:
```
  /* z-index tiers — page-level stacking only (component-internal stacking stays local) */
  --z-base: 1;
  --z-sticky: 5;
  --z-tooltip: 30;
  --z-overlay: 50;
  --z-scrim: 55;
  --z-drawer: 60;
```
Replace ONLY the page-tier literals (preserving exact stacking order):
- :308 `z-index: 5;` → `z-index: var(--z-sticky);`
- :2683 `z-index: 5;` → `z-index: var(--z-sticky);`
- :2551 `z-index: 30;` → `z-index: var(--z-tooltip);`
- :2445 `z-index: 50;` → `z-index: var(--z-overlay);`
- :3035 `z-index: 50;` → `z-index: var(--z-overlay);`
- :3181 `z-index: 50;` → `z-index: var(--z-overlay);`
- :3050 `z-index: 55;` → `z-index: var(--z-scrim);`
- :3046 `z-index: 60;` → `z-index: var(--z-drawer);`

DO NOT touch the local literals (dock base `4` at :2377; board grip/resize/dragging/remove/expand `1/2/3` at :2139/:2158/:2175/:2182/:2209; dock-grip `3` at :2432; dayentry-node `1` at :2920). The dialog top-layer has no z-index — leave it.

**Verify**:
- `grep -n "z-index: var(--z-" internal/web/assets/static/basm.css | wc -l` → 8.
- `grep -nE "z-index: (5|30|50|55|60);" internal/web/assets/static/basm.css` → empty (all page-tier literals replaced; the local `1/2/3/4` remain).
- `CGO_ENABLED=0 go build ./...` → exit 0.
- VISUAL: on storybook ≤920px (resize window to 900px), open the off-canvas drawer (`.sb-burger`) — the drawer (`--z-drawer:60`) must sit above the scrim (`--z-scrim:55`) above the topbar (`--z-overlay:50`); the tooltip bubble must float above content; the home/full dock must overlay content as before. No visual change vs pre-edit.

### Step 3: Delete the dead `--shadow-hard` alias
Remove lines 168-169 (the `/* legacy alias (pre-Hearthwood) */` comment and `--shadow-hard: 5px 5px 0;`). Leave `--drop-hard` (:167) untouched.

**Verify**:
- `grep -rn "shadow-hard" internal/ web/` → nothing in `basm.css` (DESIGN.md:308 may still match — that's plan 085's, ignore).
- `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./...` → all pass.

### Step 4: Snap the near-duplicate breakpoints
Add a documenting comment immediately before the FIRST `@media (max-width` block (basm.css:838) — a short banner stating the canon:
```
/* Breakpoints — canonical: 540 (phone) / 720 (tablet) / 920 (desktop-narrow).
   Native custom media needs a build step, so values are literal + standardized.
   480/640/860 are intentional outliers kept where a surface depends on them. */
```
Then change ONLY:
- :1840 `@media (max-width: 700px)` → `@media (max-width: 720px)`
- :2456 `@media (max-width: 900px)` → `@media (max-width: 920px)`
- :3200 `@media (max-width: 900px)` → `@media (max-width: 920px)`
- :3053 `@media (max-width: 520px)` → `@media (max-width: 540px)`
- :3060 `@media (max-width: 520px)` → `@media (max-width: 540px)`

LEAVE 480/640/860 untouched.

CAUTION on :3053 / :3060: both are storybook/composer padding tweaks. Bumping 520→540 widens the range slightly — confirm the `.sb-canvas` padding and `.composer-top` grid reflow still look right at exactly 530px and 540px.

**Verify**:
- `grep -nE "@media \(max-width: (700|900|520)px\)" internal/web/assets/static/basm.css` → empty.
- `grep -cn "max-width: 920px" internal/web/assets/static/basm.css` → 2 (was 1 at :3032; now +:2456,+:3200 = 3 total; adjust expectation: count occurrences → 3). Run `grep -nE "max-width: (540|720|920)px" internal/web/assets/static/basm.css` and confirm the snapped lines now appear.
- VISUAL: resize the window through 690-730px (board reflow, the :1840/:2263 board single-column), 510-545px (storybook/composer), and 895-925px (dock/storybook drawer). Each surface must reflow as before, just at the slightly snapped threshold. STOP if any surface visibly breaks at the new threshold.

### Step 5: Collapse free-layout boards to one column ≤720px (R3)
First, check whether plan 083 has ALREADY landed: run `grep -n 'boards.html' internal/web/boards.go`. At HEAD `boards.go:356` still renders `boards.html`, so `/boards` is still live and you should DO this step. Only SKIP it if 083 has already been applied (i.e. `boardsPage` no longer renders `boards.html` and instead redirects to `/`) — the mere EXISTENCE of `plans/083-*.md` is not enough, since that plan is unexecuted at HEAD. If you skip, note it in the summary. Otherwise, in the `@media (max-width: 720px)` block (currently :2263-2266) add two rules so free boards reflow:
```
@media (max-width: 720px) {
  .board-grid { grid-template-columns: 1fr; }
  .board-slot { grid-column: span 1 !important; }
  /* Free boards: neutralize the 10px row grid + inline grid-row so cards
     stack in flow order on phones instead of keeping desktop row spans. */
  .board-grid-free { grid-auto-rows: minmax(min-content, auto); }
  .board-grid-free .board-slot { grid-row: auto !important; }
}
```
Do NOT change boards.html (the inline `grid-row` must stay — `internal/web/boards_test.go:489` pins it).

**Verify**:
- `go test ./internal/web/...` → all pass (the `grid-row`-in-markup assertion still holds).
- `CGO_ENABLED=0 go build ./...` → exit 0.
- VISUAL: open a FREE-layout board (`/boards` with `FreeLay` true, or seed one), resize to 390px — every card must stack in a single column, top-to-bottom, no overlap, no horizontal scroll. Compare to a flow board at 390px (already worked).

### Step 6 (OPTIONAL — defer if it balloons): parch dedup
Only attempt a CSS-only collapse: find the 4-5 selectors that declare the EXACT recipe (`background-color: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px; color: var(--ink); border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel);`) with no other distinguishing property, and append a single shared selector-list rule at the END of basm.css. Do NOT touch selectors that vary the border (dashed, 1px, gold) or add other properties — those are not the same recipe. If more than ~5 minutes of judgement per selector is needed, STOP and defer the whole step with a note. Do NOT modify any `.go` file.

**Verify** (only if attempted):
- `go test ./internal/feature/storybook/...` → ok (no pinned-output change).
- VISUAL: spot-check the deduped surfaces in both modes — pixel-identical to pre-change. If anything shifts, REVERT the commit.

### Step 7: Final gates
Run the full suite and update the 082 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `git diff --check` → no output

## Test plan
This change is CSS-only (no `.go` touched unless Step 6 is skipped, which it is by default), so the existing test surface is the safety net:
- `go test ./internal/feature/storybook/...` (`TestAllStoriesRender`, `internal/feature/storybook/story_test.go:35`) — must stay green; CSS edits should not affect it, this confirms no story-render regression.
- `go test ./internal/web/...` — covers `internal/web/boards_test.go` including the free-board `grid-row` assertion (line 489-494); proves Step 5's CSS-only fix didn't break the markup contract.
- No new tests are required (CSS has no unit-test harness in this repo). The real verification is the per-step VISUAL gate in BOTH modes (`.light`/`.dark`) and all three themes (`theme-hearthwood`/`theme-forest`/`theme-dungeon`) via `document.documentElement.className = '<theme> <mode>'`, screenshotting the affected surface.
- No storybook story is added/updated — no component API changes (Step 6's component-class path is explicitly out of scope).

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` all pass (incl. storybook + boards)
- [ ] `grep -nE "#1c0d04|#101314|#c88600|#c0392b" internal/web/assets/static/basm.css` → only the new `:root` token definitions
- [ ] `grep -rn "shadow-hard" internal/ web/` → nothing under basm.css
- [ ] `grep -n "z-index: var(--z-" internal/web/assets/static/basm.css | wc -l` → 8
- [ ] `grep -nE "@media \(max-width: (700|900|520)px\)" internal/web/assets/static/basm.css` → empty
- [ ] Free-board collapse rule present: `grep -n "grid-row: auto !important" internal/web/assets/static/basm.css` → 1 (unless Step 5 skipped per 083)
- [ ] `git diff --check` → no output
- [ ] Only `internal/web/assets/static/basm.css` changed (no `.go`, no `DESIGN.md`, no `boards.html`)
- [ ] VISUAL confirmed in BOTH modes + all three themes: button/tab/portrait/candle colors unchanged (Step 1), overlay/drawer/scrim stacking unchanged (Step 2), surfaces reflow at snapped breakpoints (Step 4), free board single-column at 390px (Step 5)
- [ ] update the 082 row in plans/readme.md (add the row if it is not present yet, matching the existing column format)

## STOP conditions
- The drift check shows an in-scope file changed since `12a2ff5` and an excerpt no longer matches — STOP, report.
- A `#1c0d04` / `#101314` / `#c88600` literal is ALSO found inside a `:root.theme-*` block — STOP; tokenizing it as a constant would erase a per-theme override.
- After Step 1, the candle glow, a button, an active tab, or the portrait backdrop looks visibly different in ANY mode/theme — REVERT Step 1 and report (likely the `color-mix` alpha math or a theme-varied token assumption is wrong).
- A snapped breakpoint (Step 4) makes a surface reflow incorrectly at the new threshold — REVERT that one `@media` change and report.
- Step 5 makes `go test ./internal/web/...` fail (the `grid-row` markup assertion) — you edited boards.html instead of CSS; REVERT and use the CSS-only `!important` override.
- Plan 083 has ALREADY been applied — `boards.go:boardsPage` already redirects to `/` instead of rendering `boards.html` (verify with `grep -n 'boards.html' internal/web/boards.go`; at HEAD it still renders, so unless 083 has merged, DO Step 5) — only then SKIP Step 5 entirely and note it. The mere existence of `plans/083-*.md` is NOT a skip trigger (it is unexecuted at HEAD).
- Step 6 (parch dedup) changes any storybook pinned output or shifts a surface visually — REVERT Step 6; it is optional and value-stable-only.
- Any Verify command fails twice in a row after a fix attempt — STOP and report the command + output.

## Maintenance notes
- Future hex literals should reach for `--ink-deep` / `--portrait-bg` / `--candle-glow` rather than re-introducing raw values; the `:root` color block is the only place hex is allowed (plus theme blocks at 2815+, where hex IS the source of truth per the comment at 2812-2814).
- The `--z-*` tier is page-level only. New OVERLAYS/DRAWERS/TOOLTIPS should use the tokens; component-internal stacking (a few px of layering inside one card) should stay a small local literal — do NOT promote those to the tier.
- The dock base `z-index: 4` (:2377) deliberately sits between content and overlays and has no tier slot; if a future overlay must sit below the dock, add a `--z-rail: 4` token rather than reusing `--z-base`/`--z-sticky`.
- Breakpoints: the canon is 540/720/920 but 480/640/860 are intentional survivors — do not "finish the job" by snapping those without per-surface reflow testing.
- DESIGN.md:308 still documents `--shadow-hard`; plan 085 removes that mirror. A reviewer should confirm 085 and this plan don't both try to touch DESIGN.md.
- Deferred: the full parchment-recipe dedup (and the `Class("parch")` component path) is intentionally NOT done here to keep pinned storybook output stable — revisit as a dedicated UI plan that updates stories in the same change (per the ui-development skill).
