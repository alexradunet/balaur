# Plan 077: Clamp the fluid side padding so ultrawide and narrow viewports keep sane gutters

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 077 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css` — if the file changed since this plan was written, compare the "Current state" excerpts below to the live code; on mismatch, STOP. (At planning time HEAD *was* `12a2ff5` and the file was 3218 lines.)

## Status
- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: soft: plans/075-*.md (shares the `:root` "Layout" block in basm.css — see "Out of scope" for how to coexist if 075 already landed)
- **Category**: responsiveness
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
The outer page gutter is expressed as an uncapped viewport unit (`main { padding: 0 6vw }`), so on an ultrawide display the 1080px column has already centered (`margin: 0 auto`) yet `6vw` keeps growing — at 2560px that is ~154px of dead air *per side* stacked on top of the auto-centering, and at 360px it collapses to ~22px. Worse, the gutter is spelled four different ways across four selectors (`6vw`, `5vw`, `5vw`, `calc(--sidebar-w + 4vw)`) with no shared token, so the pages don't share a left edge. Clamping the gutter to a single `--pad` token (the CANONICAL DECISION `clamp(16px, 6vw, 64px)`) gives a stable reading edge at the extremes and one source of truth — consistent with the Hearthwood intent of a deliberate, inspectable layout (DESIGN.md). The common 1024–1440 range is intentionally left visually unchanged.

## Current state
All of this lives in one file: `internal/web/assets/static/basm.css` (3218 lines; ALL css for the app). New `:root` tokens go in the existing "Layout" group; the four padding sites are edited in place. Confirmed excerpts at HEAD `12a2ff5`:

The `:root` Layout block — the canonical home for the new token, right after `--maxw` (basm.css:154-159):
```css
:root {
  /* Layout */
  --radius: 0px;            /* RPG panels are square; rounding is for blobs */
  --maxw: 1080px;
  --chatbar-space: 210px;
```

Site 1 — `main`, the page column (basm.css:194-198):
```css
main {
  max-width: var(--maxw);
  margin: 0 auto;
  padding: 0 6vw;
}
```

Site 2 — `.profile-page` (basm.css:1511-1515):
```css
.profile-page {
  max-width: var(--maxw);
  margin: 0 auto;
  padding: 48px 5vw 120px;
}
```

Site 3 — `.with-sidebar`, the rail gutter; only the `4vw` term is in scope (basm.css:2360-2362):
```css
:root { --sidebar-w: 360px; }

.with-sidebar { padding-right: calc(var(--sidebar-w) + 4vw); }
```
The `4vw` term is ALREADY bounded away from narrow/home/fullscreen layouts — it is zeroed in three places, so only its wide-viewport growth needs clamping:
- `html.home .with-sidebar { padding-right: 0; }` (basm.css:3177)
- `.dock-full .with-sidebar { padding-right: 0; }` (basm.css:2444)
- `@media (max-width: 900px) { .with-sidebar { padding-right: 0; } ... }` (basm.css:2456-2457)

Site 4 — `.sb-canvas`, the storybook canvas (basm.css:2681):
```css
.sb-canvas { height: 100vh; overflow-y: auto; max-width: none; margin: 0; padding: 0 5vw 96px; background: var(--bg); }
```

Design constraint (CANONICAL DECISIONS, shared across plans 075–085): the fluid side-padding token is exactly
```css
--pad: clamp(16px, 6vw, 64px);
```
Sanity arithmetic the clamp preserves: at 1280px `6vw`=76.8px (inside `[16,64]`→ clamps to 64px; note: above the cap, see "STOP / assumption" below), at 1067px `6vw`=64px (the cap point), at 360px `6vw`=21.6px→ clamps up to 16px floor. The 5vw sites grow more slowly (5vw at 1280=64px) so sharing `--pad` widens their narrow-end gutter slightly and caps their wide end identically — an accepted, documented unification.

There is NO `--pad` token in the file today (grep below returns nothing) — if it exists when you read, STOP.

## Commands you will need
| Purpose | Command | Expected |
| --- | --- | --- |
| Confirm `--pad` absent | `grep -n -- '--pad:' internal/web/assets/static/basm.css` | empty (before Step 1) |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test | `go test ./...` | all pass |
| Whitespace | `git diff --check` | no output |
| Drift | `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css` | empty until you edit |
| Visual (force theme) | run app (may already serve on 127.0.0.1:8090; else `go run . serve --http=127.0.0.1:8090`), then in devtools `document.documentElement.className='theme-hearthwood dark'` (or `light`) and resize | gutters sane per "Done criteria" |

CSS-only change: there is no storybook *story* to add (no component changes), but `go test ./...` must still pass.

## Scope
**In scope** (only file you may modify): `internal/web/assets/static/basm.css` — add the `--pad` token to `:root` and rewire the four padding expressions named above.

**Out of scope** (do NOT touch):
- Any `.go` file — this is pure CSS.
- `--sidebar-w: 360px` (basm.css:2360) — the rail width is correct as-is; only the `4vw` *additive* term is clamped.
- The spacing scale tokens `--space-1..7` and width/measure tokens (`--measure`, `--w-chat-home`, etc.) — those belong to plan 075. If 075 has already landed and `--measure`/`--space-*` are present, that is fine; just add `--pad` near `--maxw` without disturbing them. If 075 already added `--pad`, STOP (see STOP conditions).
- `.topbar` (basm.css:302, `padding: 0 6vw`), `.chatbar` (basm.css:854, `padding: 14px 6vw 16px`), `.chatbar-slim` (basm.css:2353, `padding: 8px 6vw`) — these wood-chrome bars ALSO use `6vw` and visually align their inner content with `main`. They are deliberately OUT of scope for this plan to keep the change small and the risk LOW. **Caveat to note in your handoff**: because `main`'s gutter now clamps at ≥1067px but the chrome bars' `6vw` keeps growing, the topbar/chatbar inner content will drift slightly wider than `main`'s edge on ultrawide screens. This is an accepted, deferred follow-up (a future plan can switch those three to `var(--pad)` too). Do not change them here.
- `width: min(240px, 56vw)` (basm.css:332), the `clamp(...vw...)` font-sizes (basm.css:2808 is `clamp(28px, 5vw, 40px)`; basm.css:3084 is `clamp(26px, 4vw, 34px)`), `width: min(86vw, 322px)` (basm.css:3046) — unrelated `vw` usages, not side padding. Note the 2808 `5vw` is a deliberate out-of-scope font-size that REMAINS after this plan.

## Git workflow
Branch `improve/077-fluid-padding-clamp`. Conventional commit, e.g. `fix(web): clamp fluid page gutter to a shared --pad token`. Do NOT push or open a PR unless explicitly told.

## Steps
### Step 1: Add the `--pad` token to the `:root` Layout block
In `internal/web/assets/static/basm.css`, in the `:root` block under the `/* Layout */` comment, add `--pad` immediately after the `--maxw` line (basm.css:157). Target shape:
```css
  /* Layout */
  --radius: 0px;            /* RPG panels are square; rounding is for blobs */
  --maxw: 1080px;
  --pad: clamp(16px, 6vw, 64px); /* fluid outer page gutter, capped both ends */
  --chatbar-space: 210px;
```
**Verify**: `grep -n -- '--pad: clamp(16px, 6vw, 64px)' internal/web/assets/static/basm.css` → one match at ~line 158. Then `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Rewire `main` to use `--pad`
Replace `padding: 0 6vw;` inside the `main { ... }` rule (basm.css:197) with `padding: 0 var(--pad);`. Do NOT touch the `.topbar`/`.chatbar`/`.chatbar-slim` `6vw` lines (302, 854, 2353).
**Verify**: `grep -n 'padding: 0 var(--pad);' internal/web/assets/static/basm.css` → exactly one match (line ~197). `grep -cn '0 6vw' internal/web/assets/static/basm.css` → 1 remaining (only the topbar 302; chatbar-slim:2353 is `8px 6vw` and chatbar:854 is `14px 6vw 16px`, so neither matches the `0 6vw` substring — they legitimately keep `6vw` but are not counted here). `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Rewire `.profile-page` to use `--pad`
In the `.profile-page { ... }` rule (basm.css:1514), change `padding: 48px 5vw 120px;` to `padding: 48px var(--pad) 120px;` (keep the 48px top / 120px bottom literals — they are vertical, not gutter).
**Verify**: `grep -n 'padding: 48px var(--pad) 120px;' internal/web/assets/static/basm.css` → one match (~line 1514).

### Step 4: Rewire `.sb-canvas` to use `--pad`
In the `.sb-canvas { ... }` one-liner (basm.css:2681), change `padding: 0 5vw 96px;` to `padding: 0 var(--pad) 96px;` (keep the 96px bottom literal).
**Verify**: `grep -n 'padding: 0 var(--pad) 96px;' internal/web/assets/static/basm.css` → one match (~line 2681). After this step `grep -cn 'padding.*5vw' internal/web/assets/static/basm.css` → 0 (both 5vw *gutter* sites converted). Do NOT use a bare `grep -cn '5vw'` here: it returns 1 because the out-of-scope `font-size: clamp(28px, 5vw, 40px)` at line 2808 deliberately keeps its `5vw` — that single remaining `5vw` is expected, not a failure.

### Step 5: Clamp the `.with-sidebar` rail gutter `4vw` term
Change the `.with-sidebar` rule (basm.css:2362) from
```css
.with-sidebar { padding-right: calc(var(--sidebar-w) + 4vw); }
```
to
```css
.with-sidebar { padding-right: calc(var(--sidebar-w) + clamp(0px, 4vw, 48px)); }
```
The `0px` floor keeps the narrow-viewport gutter from forcing extra space (and the zeroing rules at :3177/:2444/:2456-2457 still override entirely on home/full/≤900px); the `48px` cap stops the rail gutter ballooning on ultrawide. Do NOT introduce `--pad` here — the rail gutter is a *different* measure (it sits next to a 360px fixed rail) and intentionally has its own cap of 48px.
**Verify**: `grep -n 'calc(var(--sidebar-w) + clamp(0px, 4vw, 48px))' internal/web/assets/static/basm.css` → one match (~line 2362). `grep -cn '+ 4vw' internal/web/assets/static/basm.css` → 0.

### Step 6: Full build + lint gates
**Verify** (all must pass):
- `gofmt -l` — N/A (no Go files changed).
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass (CSS is served from `embed.FS`; this confirms nothing in the embed/asset pipeline broke)
- `git diff --check` → no output (no trailing whitespace)

### Step 7: Visual verification in both modes
Run the app (it may already be serving on 127.0.0.1:8090; otherwise `go run . serve --http=127.0.0.1:8090`). For each mode set `document.documentElement.className='theme-hearthwood dark'` then `'theme-hearthwood light'` in the devtools console, and check these viewport widths by resizing:
- A domain page (uses `main`, e.g. `/today` or `/profile`) and the storybook `/storybook` (uses `.sb-canvas`).
- **360px**: outer gutter is ~16px (the clamp floor), no horizontal scrollbar appears.
- **768px**: gutter is `6vw`≈46px (mid-range, unchanged behavior).
- **1280px**: gutter is the 64px cap — verify this looks the SAME as `main` did before for typical content (the column is centered at 1080px; the cap is the practical wide-end edge). No layout jump versus pre-change.
- **2560px**: gutter is capped at 64px (NOT the old ~154px) — the page column sits with sane, fixed gutters instead of huge dead air.
- A page that renders the right rail (`.with-sidebar`, e.g. a non-home domain page with the dock in its rail) at **1280px and 2560px**: the gap between content and the dock is sane at 2560 (rail gutter capped at 48px), and the dock is NOT clipped or overlapped at either width.
**Expected**: gutters sane at both extremes, 1280 unchanged, no horizontal scroll introduced, identical behavior in dark and light (this change has no token that flips with mode, so they must match).

## Test plan
This is a CSS-only change with no component or markup edits, so there is no storybook story to add or update and no new Go test. The existing suite is the guardrail: `go test ./...` (and specifically `go test ./internal/feature/storybook/...` → ok) must still pass, confirming the embedded asset still loads and every story still renders. The substantive verification is the manual visual check in Step 7 across 360 / 768 / 1280 / 2560 in both modes.

## Done criteria
- [ ] `grep -n -- '--pad: clamp(16px, 6vw, 64px)' internal/web/assets/static/basm.css` → exactly one match in the `:root` Layout block.
- [ ] `grep -cn 'padding.*5vw' internal/web/assets/static/basm.css` → 0 (both 5vw *gutter* sites converted; a bare `grep -cn '5vw'` returns 1 — the deliberately out-of-scope `font-size: clamp(28px, 5vw, 40px)` at line 2808, which is expected to remain).
- [ ] `grep -cn '+ 4vw' internal/web/assets/static/basm.css` → 0 (rail term clamped).
- [ ] `grep -n 'padding: 0 var(--pad);' internal/web/assets/static/basm.css` → one match (`main`); `.topbar`/`.chatbar`/`.chatbar-slim` `6vw` lines untouched.
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0.
- [ ] `go vet ./...` → exit 0.
- [ ] `go test ./...` → all pass (incl. `go test ./internal/feature/storybook/...`).
- [ ] `git diff --check` → no output.
- [ ] `git diff --name-only` → only `internal/web/assets/static/basm.css` changed.
- [ ] VISUAL (both modes, dark + light): at 2560px gutters are capped (no ~154px dead air); at 360px gutter ≈16px with no horizontal scroll; at 1280px the page looks unchanged; the `.with-sidebar` rail gutter is sane and the dock is not clipped at 1280/2560.
- [ ] plans/readme.md 077 row updated to done (add the row if it is not present yet, matching the existing column format).

## STOP conditions
- **`--pad` already exists** (`grep -n -- '--pad:' internal/web/assets/static/basm.css` returns a match before Step 1) — likely plan 075 already added it. STOP and report; if the value matches `clamp(16px, 6vw, 64px)` just skip Step 1 and proceed; if it differs, STOP and reconcile with 075 before changing anything.
- **Drift**: any "Current state" excerpt does not match the live file at the cited line — STOP and report which one.
- **`.with-sidebar` clamp collapses the dock gutter at a tested width** (Step 7: content visibly crowds or the dock overlaps content at 1280px or 2560px) — per SPEC, revert ONLY Step 5 (leave `.with-sidebar` as `calc(var(--sidebar-w) + 4vw)`), keep Steps 1–4, and note in your handoff that the rail term was left unclamped.
- **A horizontal scrollbar appears at 360px** after the change — STOP; the floor or a `box-sizing` interaction is wrong. Report the measured overflow.
- **Any Verify command fails twice** in a row — STOP and report the command + output.
- **You need to edit any file other than `internal/web/assets/static/basm.css`** — STOP; that is out of scope for this plan.

## Maintenance notes
- The deliberate seam left here: `.topbar`/`.chatbar`/`.chatbar-slim` still use raw `6vw` and so will drift slightly wider than `main`'s clamped edge above ~1067px. A future plan should migrate those three chrome bars to `var(--pad)` to restore pixel-aligned edges on ultrawide; it was deferred to keep this change LOW risk.
- `--pad` is now the single source of truth for the outer page gutter — new full-width pages should use `padding: 0 var(--pad)` rather than re-inventing a `vw` value.
- The `.with-sidebar` rail gutter intentionally does NOT use `--pad` (it has its own 48px cap next to the fixed 360px rail); a reviewer should confirm the two caps were not accidentally merged.
- This plan shares the `:root` Layout block with plan 075 (spacing/measure/width/z tokens). If both land, expect a small merge in that block; the tokens are independent, so resolve by keeping all of them.
