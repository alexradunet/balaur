# Plan 076: Tokenize the chat-column widths and cap the prose reading measure

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 076 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css` — if `basm.css` changed since this plan was written, compare the "Current state" excerpts below to the live code; on mismatch, STOP and report.

## Status
- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: soft: plans/075-*.md (shares the `:root` layout block in basm.css — if 075 already landed, the new tokens go in the same block; no conflict, just edit around its additions)
- **Category**: proportions
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
The same companion chat renders at THREE different widths and only one of them is named: home is `max-width: 1800px`, the full-screen dock overlay is `max-width: 940px`, and ordinary page content is `--maxw: 1080px`. The `1800` was added in isolation by commit 12a2ff5 ("home chat + composer fill centered 1800px column"), leaving the structurally identical `.dock-full` block stale at a raw `940`, with no comment explaining why the two diverge — a reviewer cannot tell if the divergence is intentional or a missed edit. Separately the reading MEASURE is uncapped: long assistant replies and journal prose run ~110–130ch on wide displays, well past the ~66–75ch readability ceiling the repo already acknowledges (`.sb-head-blurb { max-width: 60ch }`, basm.css:3085). DESIGN.md's Hearthwood canon uses `var(--token)` for all dimensional values; naming these widths makes the intentional divergence visible and auditable, and capping the prose text run (not the chat row/portrait layout) restores legibility on wide screens without shrinking the deliberate 1800px home column.

## Current state
All edits are in **`internal/web/assets/static/basm.css`** (3218 lines; ALL css lives here). Rule blocks are appended under `/* ══ Section N: … ══ */` banners. The `:root` layout tokens live under Section 3.

**`:root` layout block** (basm.css:154–158) — where the new tokens belong, right after `--maxw`:
```
154:	:root {
155:	  /* Layout */
156:	  --radius: 0px;            /* RPG panels are square; rounding is for blobs */
157:	  --maxw: 1080px;
158:	  --chatbar-space: 210px;
```

**Home full-screen chat — the 1800 home column** (basm.css:3187–3196 and 3210):
```
3187:	html.home #dock .chat,
3188:	html.home #dock .msg-draft,
3189:	html.home #dock .chatbar,
3190:	html.home #dock .recap-zone,
3191:	html.home #dock .recap-band {
3192:	  width: 100%;
3193:	  max-width: 1800px;
3194:	  margin-left: auto;
3195:	  margin-right: auto;
3196:	}
```
```
3210:	html.home #dock .composer { width: 100%; max-width: 1800px; margin-left: auto; margin-right: auto; }
```

**Full-screen dock overlay — the stale 940** (basm.css:2447–2452); the `max-width: 940px` is on the closing line of a 6-selector group:
```
2447:	.dock-full #dock .chat,
2448:	.dock-full #dock .msg-draft,
2449:	.dock-full #dock .chatbar,
2450:	.dock-full #dock .dock-head,
2451:	.dock-full #dock .recap-zone,
2452:	.dock-full #dock .recap-band { width: auto; max-width: 940px; margin-left: auto; margin-right: auto; }
```

**The two prose runs to cap:**
- `.cmsg-body` (chat message text body), basm.css:3168 — currently `width: 100%`, no max:
```
3168:	.cmsg-body { font-size: 16px; line-height: 1.55; white-space: pre-wrap; overflow-wrap: anywhere; width: 100%; }
```
- `.journal-text`, basm.css:1442 — no own max-width, inherits `main`'s `--maxw` (1080px):
```
1442:	.journal-text { margin: 0; white-space: pre-wrap; font-size: 16px; line-height: 1.6; }
```

**Message alignment context** (so a `--measure` cap does NOT break user-right-alignment) — the cap goes on `.cmsg-body` which sits inside `.cmsg-panel` inside `.cmsg-row`; the row, not the body, owns horizontal placement (`flex-start` vs `flex-end`), basm.css:3140–3143:
```
3140:	.cmsg { display: flex; }
3141:	.cmsg-balaur { justify-content: flex-start; }
3142:	.cmsg-user { justify-content: flex-end; }
3143:	.cmsg-row { display: flex; gap: 10px; align-items: stretch; max-width: 88%; }
```
Inside the dock the row cap is lifted to full width (basm.css:3215): `#dock .chat .cmsg-row { max-width: 100%; }`.

**Precedent for the measure cap** (storybook-only today), basm.css:3085:
```
3085:	.sb-head-blurb { margin: 9px 0 0; max-width: 60ch; font-size: 15px; line-height: 1.55; color: var(--fg); }
```

**Provenance confirmed** — `git show 12a2ff5 --stat` shows the commit touched only `basm.css` (6 lines) and `home.html`; its message is "home chat + composer fill centered 1800px column". The 1800 is a DELIBERATE recent decision — GUARDRAIL: name it, never shrink it.

### Drift vs SPEC leads (reconciled by this read)
- `--maxw: 1080px` is at line **157** (SPEC said 157 — exact); `main { max-width: var(--maxw) }` is at line **195** (SPEC said 194–198 — exact, the rule spans 194–198).
- Home 1800 block confirmed at **3187–3196** and **3210** (SPEC said 3187–3197 / 3208–3217 — exact).
- `.dock-full` 940 confirmed at **2447–2452** (SPEC said "2447–2453 / 2452"; line 2453 is the *next* rule, `.dock-full #dock .chat { --portrait-size … }`, NOT part of the 940 group — do not touch 2453).
- `.cmsg-body` at **3168** (exact). `.journal-text` at **1442** (exact). `.sb-head-blurb` 60ch at **3085** (exact). `#dock .chat .cmsg-row { max-width: 100% }` at **3215** (exact).

## Commands you will need
| Purpose | Command | Expected |
| --- | --- | --- |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Storybook render | `go test ./internal/feature/storybook/...` | `ok` |
| Test (full) | `go test ./...` | all pass |
| Whitespace | `git diff --check` | no output |
| Confirm literals are gone | `grep -n "1800px\|940px" internal/web/assets/static/basm.css` | no output |
| Confirm tokens exist | `grep -n "\-\-w-chat-home\|\-\-w-chat-overlay\|\-\-measure" internal/web/assets/static/basm.css` | 8 hits at completion (3 defs + 5 uses: 2× `--w-chat-home`, 1× `--w-chat-overlay`, 2× `--measure`); fewer mid-plan as Steps 2/3 add the uses |
| Visual | run the app (may already be serving on `127.0.0.1:8090`; else `go run . serve --http=127.0.0.1:8090`), force theme via `document.documentElement.className='theme-hearthwood dark'` (or `light`) | as described in Done criteria |

## Scope
**In scope** (only file you may modify):
- `internal/web/assets/static/basm.css` — add 3 tokens; replace 3 literals (two `1800px`, one `940px`) with vars; add a `max-width: var(--measure)` cap to `.cmsg-body` and `.journal-text`.

**Out of scope** (do NOT touch):
- `web/templates/chat_dock.html`, `web/templates/home.html`, and the `chat.Dock` gomponents port — owned by plan 084 (the markup is unaffected; this is a pure CSS change).
- Any `.go` file — CSS-only change.
- The spacing scale tokens (`--space-*`) and breakpoint constants — owned by plan 075. If 075 already added tokens to the `:root` layout block, append yours after its additions; do not reorder or edit its lines.
- `--maxw` (1080px) itself and the `main` rule at 195 — leave the page column as-is; only the chat columns are renamed.
- `.cmsg-row` `max-width` (88% / 100%) — that governs the bubble+portrait layout width, not the text measure; leave it.

## Git workflow
Branch `improve/076-chat-width-measure-tokens`. Conventional commits, e.g. `refactor(web): tokenize chat-column widths and cap prose measure`. Do NOT push or open a PR unless instructed.

(Sandbox note: in a TLS-intercepting Hyperagent sandbox, Go commands need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. CSS edits need no network; only the Go verify commands do.)

## Steps

### Step 1: Add the three width/measure tokens to the `:root` layout block
In basm.css, after the `--maxw: 1080px;` line (currently line 157), insert the three tokens with a comment that documents the intentional divergence. Target shape:
```
  --maxw: 1080px;
  /* Chat-column widths diverge by surface, deliberately: home IS the full-screen
     chat (wide), the .dock-full overlay sits OVER page content (narrower). Named
     so the divergence is intentional and visible, not a missed edit. */
  --w-chat-home: 1800px;
  --w-chat-overlay: 940px;
  --measure: 68ch;          /* prose reading-width cap (~66–75ch legibility) */
```
Keep these on the canonical values from the shared decisions: `--w-chat-home: 1800px` (do NOT shrink), `--w-chat-overlay: 940px`, `--measure: 68ch`.
**Verify**: `grep -n "\-\-w-chat-home\|\-\-w-chat-overlay\|\-\-measure" internal/web/assets/static/basm.css` -> 3 definition hits in the `:root` block (more once Step 2/3 add uses). And `CGO_ENABLED=0 go build ./...` -> exit 0.

### Step 2: Replace the three raw width literals with the new tokens
Three exact replacements:
1. Home block (currently line 3193): `  max-width: 1800px;` -> `  max-width: var(--w-chat-home);`
2. Home composer (currently line 3210): in `html.home #dock .composer { … max-width: 1800px; … }` replace `max-width: 1800px;` -> `max-width: var(--w-chat-home);`
3. Dock-full overlay (currently line 2452): in `.dock-full #dock .recap-band { width: auto; max-width: 940px; … }` replace `max-width: 940px;` -> `max-width: var(--w-chat-overlay);`

Do NOT touch line 2453 (`.dock-full #dock .chat { --portrait-size: … }`) — it is a separate rule.
**Verify**: `grep -n "1800px\|940px" internal/web/assets/static/basm.css` -> no output (zero raw literals remain). Then `CGO_ENABLED=0 go build ./...` -> exit 0.

### Step 3: Cap the prose reading measure on the two text runs
1. `.cmsg-body` (currently line 3168): add `max-width: var(--measure);` to the rule. Keep `width: 100%` so the body still fills up to the cap inside its panel. Target:
   ```
   .cmsg-body { font-size: 16px; line-height: 1.55; white-space: pre-wrap; overflow-wrap: anywhere; width: 100%; max-width: var(--measure); }
   ```
2. `.journal-text` (currently line 1442): add `max-width: var(--measure);`. Target:
   ```
   .journal-text { margin: 0; white-space: pre-wrap; font-size: 16px; line-height: 1.6; max-width: var(--measure); }
   ```
Place the cap on `.cmsg-body` (inside the panel), NOT on `.cmsg-row` — the row owns left/right placement (`flex-start`/`flex-end` at basm.css:3141–3142), so capping the body does not disturb user-message right-alignment.
**Verify**: `CGO_ENABLED=0 go build ./...` -> exit 0; `git diff --check` -> no output.

### Step 4: Run the full gate
**Verify**:
- `go vet ./...` -> exit 0
- `go test ./internal/feature/storybook/...` -> `ok`
- `go test ./...` -> all pass
- `gofmt -l internal/web/assets/static/basm.css` is N/A (not Go); skip.

### Step 5: Visual check in BOTH modes (see Done criteria for the exact assertions)
Run the app, open the relevant surfaces, force each mode, and confirm the assertions below. This is a manual gate — record what you observed in your report.

## Test plan
This is a CSS-only change with no component API change, so no new Go test or storybook story is required. The existing `TestAllStoriesRender` (`internal/feature/storybook/story_test.go:35`) and `tours_test.go` are the regression net — they fail if any component/story stops rendering or a tour anchor breaks. Run `go test ./...` (Step 4) as the automated gate.

Visual verification (Step 5), in BOTH `light` and `dark` (set `document.documentElement.className='theme-hearthwood dark'` then re-check with `light`):
1. **Home at ~2560px wide**: paste/observe a LONG assistant message — its text body wraps at ~68ch (no longer runs the full column), while the chat column itself is still capped at 1800px (measure a `.chat` element’s rendered width ≈ 1800, not shrunk). Short messages still shrink-wrap as before.
2. **Full-screen dock overlay at ~1440px** (toggle dock-full): the chat column is ~940px and centered; long messages wrap at ~68ch.
3. **A journal focus page at ~1440px** (a page using `.journal-text`, e.g. journal-focus): long-form prose wraps at ~68ch, not the full 1080px page column.
4. **Responsiveness ≤920px**: at ≤900px the home/dock blocks already go fixed/full-bleed (basm.css:3200–3202, 2456); confirm the `--measure` cap does not introduce horizontal scroll or awkward narrow text on a phone-width viewport (≤540px) — the body should simply fill the available width when it is below 68ch.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` passes (incl. `go test ./internal/feature/storybook/...` -> `ok`).
- [ ] `grep -n "1800px\|940px" internal/web/assets/static/basm.css` returns NO output (all three literals tokenized).
- [ ] `grep -n "\-\-w-chat-home\|\-\-w-chat-overlay\|\-\-measure" internal/web/assets/static/basm.css` shows the 3 definitions plus their uses (2× `--w-chat-home`, 1× `--w-chat-overlay`, 2× `--measure` on `.cmsg-body`/`.journal-text`).
- [ ] `git diff --check` returns no output.
- [ ] `git diff --name-only` shows ONLY `internal/web/assets/static/basm.css` (plus `plans/readme.md` for the 076 row, added if not already present).
- [ ] VISUAL (both modes): home column still 1800px wide with long-message text wrapping ~68ch; dock-full overlay ~940px; journal prose ~68ch; no horizontal overflow ≤920px.
- [ ] update the 076 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).

## STOP conditions
- **Drift**: the `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css` shows changes AND any "Current state" excerpt no longer matches the live file (e.g. the `1800px`/`940px` literals are already replaced with vars by another plan) — STOP and report; the work may be partly done.
- **Right-alignment breakage**: if adding `max-width: var(--measure)` to `.cmsg-body` visibly breaks user-message right-alignment (user bubbles drift left/center instead of hugging the right edge), STOP applying it broadly — instead cap ONLY the assistant body via `.cmsg-balaur .cmsg-body { max-width: var(--measure); }`, leave `.cmsg-user .cmsg-body` uncapped, and report the divergence. (Per the alignment analysis the row owns placement, so this is unlikely — but verify visually before declaring done.)
- **A Verify fails twice**: if any build/vet/test command fails twice after your best fix, STOP and report the failure output verbatim.
- **Out-of-scope need**: if the cap can only be made to work by editing a `.go` file, a template, or `.cmsg-row`, STOP — that contradicts the scope and means an assumption is wrong; report it.

## Maintenance notes
- When plan 084 ports `chat.Dock` to gomponents, the markup classes (`.chat`, `.composer`, `.cmsg-body`, `.journal-text`) must stay stable for these CSS rules to keep applying; that plan should reuse these tokens, not reintroduce literals.
- The `--w-chat-home`/`--w-chat-overlay` divergence is intentional and commented — a future reviewer who sees two chat widths should read the comment before "fixing" them to match.
- Deferred: a true CSS custom-media breakpoint system and a shared `--pad` gutter token are owned by other 075–085 plans; this plan only adds the three width/measure tokens. The `--measure: 68ch` value is a single source of truth — if other prose surfaces (future markdown rendering, recap body) need a reading cap, reuse `var(--measure)` rather than re-literalizing a ch value.
