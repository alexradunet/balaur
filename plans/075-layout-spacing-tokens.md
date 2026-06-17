# Plan 075: Establish the Hearthwood spacing-scale tokens and migrate the dominant spacing literals

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 075 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css` — if basm.css changed since this plan was written, compare the "Current state" excerpts below to the live code; on mismatch, STOP and report.

## Status
- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: none (soft ordering: run BEFORE plans/076 (width/measure tokens), plans/077 (padding/fluid-gutter token), and plans/082 (z-index/breakpoint tokens) — they all add sibling tokens to the SAME `:root` layout block at basm.css:154-177, so landing this first avoids merge churn)
- **Category**: tech-debt / architecture
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
`basm.css` tokenizes color and typography exhaustively but has NO layout vocabulary: the `:root` layout block (basm.css:154-177) defines only `--radius`, `--maxw`, `--chatbar-space`, plus bevel/motion/focus tokens. Spacing is hand-picked literals scattered across 3218 lines — `grep -oE '(padding|gap|margin)[^;]*: *[^;]*8px'` alone matches 86 times, `12px` 45 times, `16px` 25 times. There is no shared rhythm, so any global retune means hand-editing hundreds of literals, and there is no single source of truth for the spacing system. The storybook overview already advertises an "8px row unit" stat (`internal/feature/storybook/overview.go:34`), proving the base unit exists conceptually but is untokenized. This plan introduces the canonical 4px-base spacing scale and migrates the dominant literals to it as a **value-preserving** sweep (every migrated declaration must resolve to the SAME px it was before), establishing the vocabulary that 076/077/082 build on. Aligns with the AGENTS.md "one source of truth per concern" SUCKLESS rule and the DESIGN.md token-driven discipline.

## Current state

**File**: `internal/web/assets/static/basm.css` — the single CSS file (3218 lines); "Canonical token source: if DESIGN.md and this file disagree, this file wins." CSS is organized into numbered `Section` comment banners. The layout tokens live in **Section 3** (basm.css:149-177).

Current layout `:root` block (basm.css:154-177) — confirmed at HEAD:
```css
154 :root {
155   /* Layout */
156   --radius: 0px;            /* RPG panels are square; rounding is for blobs */
157   --maxw: 1080px;
158   --chatbar-space: 210px;
159
160   /* Bevels & drops (the shadow system) */
161   --bevel-up: inset 0 3px 0 var(--bevel-light), ...
```
There is currently **no** `--space-*` token defined or used: `grep -nE '\-\-space\-[0-9]' internal/web/assets/static/basm.css` returns nothing, and `grep -n 'var(--space-' ...` returns nothing.

**The "8px row unit" claim** (the conceptual base unit, untokenized) lives at `internal/feature/storybook/overview.go:34`:
```go
34   stat("8px", "row unit"),
```
(NOTE: an earlier audit lead said this was in `storybook.go`; it is actually in `overview.go`. This file is OUT OF SCOPE — do not edit it; it is cited only as evidence the base unit is intentional.)

**Worked-reference region 1 — the topbar (shell)** (basm.css:295-347). Confirmed excerpt:
```css
295 .topbar {
...
300   gap: 14px;          /* 14px is OFF-SCALE — LEAVE LITERAL */
301   height: 62px;       /* geometry, not spacing — LEAVE LITERAL */
302   padding: 0 6vw;     /* fluid gutter — owned by plan 077, LEAVE for now */
...
338 .topbar nav { margin-left: auto; display: flex; gap: 18px; }  /* 18px OFF-SCALE — LEAVE */
```
Note: the topbar's spacing values (`14px`, `18px`, `9px`) are mostly OFF-SCALE — this region has FEW migratable literals. Migrate only exact scale matches.

**Worked-reference region 2 — the chat message organism `.cmsg-*`** (basm.css:3140-3168). Confirmed excerpt:
```css
3143 .cmsg-row { display: flex; gap: 10px; align-items: stretch; max-width: 88%; }   /* 10px OFF-SCALE — LEAVE */
3146   overflow: hidden; padding: 3px; ...   /* 3px bevel/inset — LEAVE LITERAL */
3157   border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel); padding: 16px 16px 13px;  /* 16px -> token; 13px OFF-SCALE, LEAVE */
3164   ... padding: 1px 9px; line-height: 1.6; }   /* 1px / 9px OFF-SCALE — LEAVE */
3166 .cmsg-balaur .cmsg-name { left: 12px; ... }   /* 12px positional offset -> token (see judgment note) */
3167 .cmsg-user .cmsg-name { right: 12px; ... }
```

**The canonical scale to add** (CANONICAL DECISIONS, shared across plans 075-085):
```
--space-1:4px; --space-2:8px; --space-3:12px; --space-4:16px; --space-5:24px; --space-6:32px; --space-7:48px;
```
Mapping for migration: `4px -> var(--space-1)`, `8px -> var(--space-2)`, `12px -> var(--space-3)`, `16px -> var(--space-4)`, `24px -> var(--space-5)`, `32px -> var(--space-6)`, `48px -> var(--space-7)`.
**OFF-SCALE values (LEAVE LITERAL, do not round):** `1px, 2px, 3px, 5px, 6px, 7px, 9px, 10px, 13px, 14px, 18px, 20px` and any non-multiple-of-4 / non-listed value. Bevel offsets (`3px`), the `7px` gold notch, portrait sizes, `2px` outlines, and any geometric one-off stay literal.

**Design constraints to honor** (DESIGN.md / CANONICAL DECISIONS):
- This is a **value-preserving mechanical sweep**. Rendered output must be byte-identical: `8px` must continue to compute to `8px`. If a literal is not an EXACT scale match, LEAVE IT — never round `10px` or `14px` to a token.
- Only migrate `padding`, `gap`, and `margin` literals. Do NOT migrate `width`, `height`, `top/left/right/bottom` border/inset geometry, `box-shadow` offsets, `font-size`, `line-height`, `border-width`, or `flex-basis`. (Positional offsets like the `.cmsg-name` `left: 12px` are a judgment call — default to LEAVE them literal unless the step explicitly migrates them; positional geometry is out of the "spacing rhythm" the tokens express.)
- New token definitions belong in the EXISTING `:root` layout block right after `--maxw` (basm.css:157), not in a new block.
- Colors stay `var(--token)`; this plan touches spacing only.

## Commands you will need
| Purpose | Command | Expected |
| Drift check | `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css` | no output (or compare excerpts) |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test | `go test ./...` | all pass |
| Storybook render | `go test ./internal/feature/storybook/...` | ok |
| Tokens defined | `grep -nE '\-\-space\-[1-7]: *[0-9]+px' internal/web/assets/static/basm.css` | 7 lines, in `:root` |
| Tokens used | `grep -c 'var(--space-' internal/web/assets/static/basm.css` | > 0 (grows each step) |
| No off-scale token defs | `grep -nE '\-\-space\-[1-7]: *(5|6|7|9|10|13|14|18|20)px' internal/web/assets/static/basm.css` | empty |
| Whitespace | `git diff --check` | no output |
| Visual (both modes) | run the app (may already serve on 127.0.0.1:8090; else `go run . serve --http=127.0.0.1:8090`), open `/storybook` and `/`, force `document.documentElement.className='theme-hearthwood dark'` (then `...light`), screenshot, compare to a pre-change baseline; check responsiveness <=920px | NO visual change |

Sandbox note: in a TLS-intercepting Hyperagent sandbox, Go commands need the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope
**In scope** (only file you may modify):
- `internal/web/assets/static/basm.css` — add the scale tokens; migrate dominant `padding`/`gap`/`margin` literals (`8/12/16/24/32px` exact matches) to the tokens.

**Out of scope** (do NOT touch):
- Any `.go` file — including `internal/feature/storybook/overview.go` (the "8px row unit" stat stays as a literal label; it is documentation, not CSS).
- Any **visual value change** — this is byte-identical-output only. No proportional retuning, no rounding off-scale values.
- `--maxw`, `--measure`, `--w-chat-*`, the `--pad` fluid-gutter token, `--z-*`, and breakpoint constants — owned by plans 076 / 077 / 082. Do NOT add them here.
- `DESIGN.md` — its layout-token documentation note is plan 085.
- `width`/`height`/positional/`box-shadow`/`font`/`border-width`/`flex-basis` literals — not spacing rhythm.

## Git workflow
- Branch: `improve/075-layout-spacing-tokens` off `main`.
- Conventional commits, one per CSS region so a regression is bisectable, e.g.:
  - `feat(web): add Hearthwood spacing-scale tokens (--space-1..7)`
  - `refactor(web): migrate shell/chat spacing literals to scale tokens`
  - `refactor(web): migrate card/feedback spacing literals to scale tokens`
- Do NOT push or open a PR unless explicitly told.

## Steps

### Step 1: Add the spacing-scale tokens to the `:root` layout block
In `internal/web/assets/static/basm.css`, inside the layout `:root` block, insert the scale immediately after `--maxw: 1080px;` (basm.css:157) and before `--chatbar-space: 210px;` (basm.css:158). Target shape:
```css
  --maxw: 1080px;

  /* Spacing scale — 4px base. Dominant literals (8/12/16/24/32) map to these;
     10px and 14px are intentionally OFF-SCALE and stay literal, as do bevel
     offsets (3px), the 7px gold notch, and portrait/geometry one-offs. */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 24px;
  --space-6: 32px;
  --space-7: 48px;

  --chatbar-space: 210px;
```
**Verify**:
- `grep -nE '\-\-space\-[1-7]: *[0-9]+px' internal/web/assets/static/basm.css` → exactly 7 lines.
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `go test ./internal/feature/storybook/...` → ok.

### Step 2: Capture a visual baseline BEFORE migrating any literal
Run the app (it may already serve on 127.0.0.1:8090; else `go run . serve --http=127.0.0.1:8090`). Open `/storybook` and `/`. In each, force the theme via `document.documentElement.className='theme-hearthwood dark'`, screenshot; then `document.documentElement.className='theme-hearthwood light'`, screenshot. Save these four screenshots as the BASELINE to diff against after Steps 3-4. (Tokens were added in Step 1 but no literal migrated yet, so the page is unchanged from `main` — this baseline is the ground truth for "no visual change".)
**Verify**: four baseline screenshots exist (storybook×{dark,light}, home×{dark,light}). STOP if the app will not start.

### Step 3: Migrate the shell + chat regions (the worked reference)
Migrate ONLY exact-match `padding`/`gap`/`margin` literals to tokens in these regions, leaving every off-scale value literal:
- The shell rules around basm.css:194-347 (`main`, `.topbar` family) — note most topbar values are off-scale (`14px`, `18px`, `9px`); only migrate true `8/12/16/24/32px` `padding`/`gap`/`margin` declarations. Do NOT migrate `padding: 0 6vw` (fluid gutter, plan 077).
- The chat `.cmsg-*` rules around basm.css:3140-3168 — `padding: 16px 16px 13px` (basm.css:3157) becomes `padding: var(--space-4) var(--space-4) 13px` (the `13px` stays literal). Leave `gap: 10px` (basm.css:3143), `padding: 3px` (basm.css:3146), `padding: 1px 9px` (basm.css:3164). Treat the `.cmsg-name` `left/right: 12px` (basm.css:3166-3167) as positional offset — LEAVE literal.
For multi-value shorthand, migrate each component independently and only if it is an exact scale match (e.g. `padding: 8px 16px` → `padding: var(--space-2) var(--space-4)`; `margin: 8px 10px` → `margin: var(--space-2) 10px`).
**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./internal/feature/storybook/...` → ok.
- `git diff --check` → no output.
- Re-screenshot `/storybook` and `/` in BOTH modes and compare to the Step-2 baseline → **NO visual difference**. If anything moved by a pixel, you mis-mapped a literal — revert that line. (This is a STOP condition.)
- Commit: `refactor(web): migrate shell/chat spacing literals to scale tokens`.

### Step 4: Migrate the card + feedback sections
Continue the same exact-match sweep through the remaining sections (cards, feedback/alerts, tags, composer, sidebar, board/domain rules) for `padding`/`gap`/`margin` only. Work section-by-section using the `Section` comment banners as boundaries. Re-confirm each candidate is an EXACT `8/12/16/24/32px` match before replacing; LEAVE `6px`, `10px`, `14px`, `18px`, `20px`, `13px`, and all geometry/bevel/positional values. Do NOT migrate inside `@media` breakpoint blocks if the surrounding value is part of responsive geometry rather than spacing rhythm — when unsure, LEAVE it (deferring a literal is always safe here).
**Verify** (after each section or small batch):
- `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./...` → all pass.
- `git diff --check` → no output.
- `grep -nE '\-\-space\-[1-7]: *(5|6|7|9|10|13|14|18|20)px' internal/web/assets/static/basm.css` → empty (no off-scale value snuck into a token def).
- Re-screenshot `/storybook` and `/` in BOTH modes vs the Step-2 baseline → NO visual difference.
- Commit per logical batch: `refactor(web): migrate card/feedback spacing literals to scale tokens`.

### Step 5: Final full verification
Run the complete gate (see Done criteria). Update the 075 row in `plans/readme.md` (add the row if it is not present yet, matching the existing column format).
**Verify**: all Done-criteria checkboxes pass.

## Test plan
- **No new Go tests** — this is a CSS-only, value-preserving change.
- Regression gate is the existing storybook render suite plus a manual both-modes visual diff:
  - `go test ./internal/feature/storybook/...` (covers `TestAllStoriesRender`) → ok.
  - `go test ./...` → all pass (includes `tours_test.go`).
  - Visual: screenshot `/storybook` and `/` (home) in BOTH `dark` and `light` (force via `document.documentElement.className`), diff against the Step-2 baseline → no change.
- **No storybook story added/updated**: this plan changes no component markup or props, only the CSS variable plumbing behind existing classes. (The "8px row unit" overview stat already documents the base unit; do not edit it.)

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0.
- [ ] `go vet ./...` → exit 0.
- [ ] `go test ./...` → all pass.
- [ ] `go test ./internal/feature/storybook/...` → ok.
- [ ] `grep -nE '\-\-space\-[1-7]: *[0-9]+px' internal/web/assets/static/basm.css` → exactly 7 lines, all inside the `:root` layout block, values `4px 8px 12px 16px 24px 32px 48px`.
- [ ] `grep -c 'var(--space-' internal/web/assets/static/basm.css` → > 0 (tokens are actually used, not just defined).
- [ ] `grep -nE '\-\-space\-[1-7]: *(5|6|7|9|10|13|14|18|20)px' internal/web/assets/static/basm.css` → empty (no off-scale value mapped to a token).
- [ ] `git diff --check` → no output.
- [ ] `git diff --stat` shows ONLY `internal/web/assets/static/basm.css` changed (plus `plans/readme.md`).
- [ ] **Visual, BOTH modes**: `/storybook` and `/` render byte-for-byte identically to the Step-2 baseline in `theme-hearthwood dark` and `theme-hearthwood light`; layout intact at <=920px width.
- [ ] `plans/readme.md` 075 row updated to done (add the row if it is not present yet, matching the existing column format).

## STOP conditions
- **Drift**: the Step-1 drift check shows basm.css changed since `12a2ff5` and the `:root` layout block (basm.css:154-177) no longer matches the "Current state" excerpt — STOP and report.
- **Tokens already exist**: `grep -nE '\-\-space\-[0-9]' internal/web/assets/static/basm.css` returns matches at the start — the scale was already added (drift). STOP and report.
- **Computed-value regression**: any post-migration screenshot differs from the Step-2 baseline in either mode — you mis-mapped a literal (e.g. snapped an off-scale value, or migrated a non-spacing property). Revert the offending line; if you cannot localize it after two attempts, STOP and report.
- **Out-of-scope file needed**: the migration appears to require editing a `.go` file, `DESIGN.md`, or `plans/readme.md` (beyond the final row update) — STOP; that means the scope is wrong.
- **A Verify fails twice** after a fix attempt — STOP and report the command + output.

## Maintenance notes
- Plans **076** (width/measure: `--maxw` stays 1080, add `--measure:68ch`, `--w-chat-home:1800px`, `--w-chat-overlay:940px`), **077** (`--pad: clamp(16px,6vw,64px)` fluid gutter — this is why Step 3 leaves `padding: 0 6vw`), and **082** (`--z-*` tier + breakpoint constants) all add sibling tokens to the SAME `:root` layout block this plan touches. Landing 075 first minimizes their merge churn; reviewers of those plans should confirm they extend rather than reorder the block.
- **Plan 085** documents the layout-token vocabulary in `DESIGN.md` — that is where the spacing scale gets its prose; do NOT pre-empt it here.
- A reviewer should scrutinize: (1) that NO migrated declaration changed its computed px (the visual baseline diff is the real gate, not the grep), and (2) that off-scale values (`10px`, `14px`, `18px`, bevel `3px`, notch `7px`, positional offsets) were left literal — the temptation to "tidy" them into the nearest token is the main regression risk.
- **Deferred**: off-scale literals (`10px`, `14px`) are intentionally NOT tokenized; revisit only with a deliberate design decision to add half-steps to the scale (`--space-2-5` etc.), which is out of scope here. The `font-size`/`line-height`/geometry literals are a separate (typographic/geometry) concern, not spacing rhythm.
