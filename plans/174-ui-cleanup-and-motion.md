# 174 — UI cleanup sweep, rail-collapse fix, and pixel-snappy motion quick-wins

**Priority:** P1 (S1–S2 are correctness/UX bugs) · **Effort:** L (split into
independently-shippable steps) · **Depends on:** — · **Status:** TODO

Written by the advisory session on 2026-06-24 from a live browser tour +
a five-dimension UI audit (dead-UI, storybook drift, dead-CSS, incomplete
features, a11y/responsive) + the 2026 motion research in
[`docs/ui-motion-2026.md`](../docs/ui-motion-2026.md). All `file:line`
references are from that audit against the tree at this date — **a zero-context
executor MUST re-confirm each before acting** (the checkout is shared and moves).

## Why

The product is healthy, but three classes of debt accumulated: (a) one error
path leaks raw internals into chat; (b) an open focus-panel squeezes the chat
column to ~170px (unusable); (c) major past migrations (`cmsg-*` chat redesign,
plan 089 board/topnav retirement, the single-page shell of 088/098/102) left a
thick layer of **dead CSS and catalog-only components**. This plan removes the
debt and lands the two highest-leverage, lowest-risk motion wins from the
research, in a safe order.

## Constraints (load-bearing — read before any step)

1. **This checkout runs on the owner's live production VPS.** This plan touches
   **no migrations and no data** — it is UI/CSS/handler code only. Keep it that
   way; if a step seems to need a schema change, stop and re-scope.
2. **Land on `main`, gated on a green full suite.** Each step below is its own
   commit (conventional-commit subject). Run `go test ./...`, `go vet ./...`,
   `CGO_ENABLED=0 go build ./...`, `gofmt -l`, and `git diff --check` before
   every push. Never push red. The checkout is shared by parallel sessions —
   stage only your own files, `git fetch` + confirm a clean fast-forward first.
3. **Use graphify first for code exploration** (per `CLAUDE.md`):
   `graphify query "<q>"` / `graphify explain` / `graphify path` before grepping;
   run `graphify update .` after code changes.
4. **Verify UI in a real browser**, not just tests (`run-balaur` skill: `make run`,
   drive `http://127.0.0.1:8090/` + `/storybook/<id>` with Playwright). Check
   legibility on each surface and responsiveness at the 540/720/920 breakpoints.
5. **Storybook is the source of truth.** Any component you change updates its
   story in the same commit; any you delete loses its story + test together.
6. **Update `internal/self/knowledge.md`** in the same commit when a step alters
   architecture/capability (S1 error handling, S6/S7 motion + the `#toast` region).
7. **CSS rules** (`basm.css`): append new rules at the END, single-dash class
   names, square corners (`--radius`), no raw hex outside `:root`, no blur, use
   `--space-N` / `--pad` / `--z-*` tokens, snap `@media` to 540/720/920.

---

## Execution order

| Step | Goal | Risk | Ship |
|---|---|---|---|
| S1 | Sanitize raw tool errors in chat | low | standalone |
| S2 | Fix rail-collapse (chat column floor + container query) | low | standalone |
| S3 | Delete dead CSS + `board.js` | low | standalone |
| S4 | Wire-or-delete the catalog-only atoms + add coverage test | med | standalone |
| S5 | Fix storybook drift (GraphCard/SettingsFocus/DayFocus) | low | standalone |
| S6 | Motion win #1: Datastar view-transitions + one motion layer | med | standalone |
| S7 | Motion win #2: entry/exit panels + wire `Toast` | med | standalone |

Do S1–S2 first (they are user-facing bug fixes). S3–S5 are the cleanup sweep.
S6–S7 are the polish, and depend on the research doc.

---

## S1 — Sanitize tool-execution errors before they render in chat

**Bug:** a failed agent tool returns `fmt.Sprintf("error: %v", err)`
(`internal/agent/agent.go:152`), which flows as a `tool_result` event into
`internal/web/chatstream.go:197 handleToolResult` and is rendered verbatim via
`s.endTool(clipText(ev.Text, 2000), nil)` (`chatstream.go:222`). The friendly
sanitizer `chatErrText` (`internal/web/chat.go:78`, which rewrites provider URLs
to "the model is unreachable…" and logs the raw detail) is applied **only** to
`Kind=="error"` turn-level events (`chatstream.go:188`), never to tool errors.
So a private path / internal detail / provider URL in a tool error reaches the
owner unmodified. (Seen live: `card_show: invalid card: unknown card type ""`,
`show_cards: bad arguments: json: cannot unmarshal number…`.)

**Change:** in `handleToolResult`, detect a tool result with the `"error: "`
prefix and route it through a sanitizing path before `endTool` — reuse/adapt
`chatErrText` (strip URLs/private paths, **log the raw detail** via
`app.Logger()`, surface a short owner-facing line). Keep non-error tool results
on the existing `clipText` path.

**Verify:** unit-test that a tool error containing a filesystem path / `http://`
URL renders a sanitized string and that the raw text is logged, not shown. In
the browser, trigger a failing tool and confirm the tool card shows the friendly
line. `go test ./internal/web/...`. Update `internal/self/knowledge.md` if it
describes tool-error handling.

---

## S2 — Fix the rail-collapse (open panel squeezes chat to ~170px)

**Bug (root cause):** the app-shell grid is
`grid-template-columns: 1fr var(--w-panel) var(--w-rail)` (`basm.css:3475`,
= `1fr/480px/56px`). The chat column is a bare `1fr` with **no `minmax()` floor**,
and `#dock`'s `overflow:hidden` (`basm.css:2444`, inherited by
`html.app #dock.app-dock` at `:3528`) makes the grid item's auto-min resolve to
0. So an open panel shrinks the chat without limit; `.cmsg-body` /
`.cmsg-tool .tool-args pre` / the composer all wrap with `overflow-wrap:anywhere`
and break to one letter/word per line. The only width breakpoint is the ≤720px
overlay (`basm.css:3585`); between 721px and wide desktop nothing stops it.

**Change (robust fix):** give the chat column a content floor —
`grid-template-columns: minmax(420px, 1fr) minmax(0, var(--w-panel)) var(--w-rail)`
at `basm.css:3475` (floor ≈ the home composer min). Do **not** rely on removing
`overflow:hidden` alone. **Enhancement (the research's Tier-1 pattern):** set
`container-type: inline-size` on the dock and use `@container` to switch the
dock's rail-vs-canvas affordances by *its own* width rather than the viewport
(this is the durable, slot-aware pattern — see `docs/ui-motion-2026.md` Tier 1,
"Container queries"; Baseline Widely available). Optionally raise the off-canvas
overlay to ≤1100px so the panel goes off-canvas before the squeeze.

**Verify:** in-browser at ~900px and ~1000px with a focus panel open — the
composer and tool-card text stay readable; no horizontal scroll at ~390px.
Snap any new `@media` to 540/720/920.

---

## S3 — Delete dead CSS and the orphaned `board.js`

All grep-verified zero-reference at audit time. **Re-grep each class/token
across `*.go`/`*.js` (excluding `vendor/` and the `basm.css` definition) and
confirm 0 hits before deleting.** Group deletes by retired surface for a clean
diff.

1. **Legacy `.msg / .msg-*` chat family** (superseded by `.cmsg-*`):
   `basm.css:532–711, 873–883, 1296`, and the `.msg-draft`/`.msg-main` entries
   inside the `#dock` / `html.home` / `.dock-full` / `.dock-v-*` / `html.app`
   selector lists at `2409–2515, 3318, 3440, 3452, 3542`. Confirm `cmsg-*` /
   `composer-*` fully cover the live chat first.
2. **Boards** (retired plan 089): `.board-*` block `2212–2320`, `.btn-danger`
   `2231`, and delete `internal/web/assets/static/board.js` (confirm no `<script>`
   / embed serves it).
3. **Retired multi-page IA** (plans 088/098/102): `.profile-page`/`.profile-lede`
   `1548,1558`; `.settings-page` `1869`; `.candle-*` `2185,2196,2204` +
   `@keyframes candle-breathe` `2191` (then drop `--candle-glow` from `:root`);
   `.focus-*` `2323–2338`; `.recap-day` `2179`; `.t-views` `1307`; `.tl-quiet`
   `1427`; `.dayentry-ember` `2945`; `.h-icon-sm` `2398`; old model-switcher
   `.model-choice-*`/`.model-name`/`.model-detail`/`.model-error`/`.model-form-checks`
   `1660–1705,1694,1700,1842,2168`; head-detail `.head-meta`…`.head-name-label`
   `1752–1789`; `.chatbar-download*`/`.chatbar-back-link`/`.chatbar-head-context`
   `1779–1838`; `.capture-note*` `1311–1332,981`; pre-redesign calendar
   `.cal-nav`/`.cal-label`/`.cal-recur` `1362,1363,1415`; state anims
   `.task-fresh` `1339`, `.tcard-dropped` `1349`, `.type-cursor` `714`.
4. **Unused `:root` tokens:** `--on-surface` (`:42`, dupes `--ink`) and
   `--steel` (`:68`).

**Verify:** `CGO_ENABLED=0 go build ./...`; full `go test ./...`; load every
product surface + storybook in the browser and confirm nothing lost its styling.
Expect a large net-negative diff.

---

## S4 — Wire-or-delete the catalog-only atoms; enforce coverage

~15 exported `internal/ui` atoms are built, tested, and storied but rendered by
**no** product screen (grep-verified: only def + `_test.go` + story). For each,
**re-confirm zero `ui.X(` call sites**, then take ONE disposition:

- **Revive (3 have an obvious home that hand-rolls the markup today):**
  route `internal/feature/taskcards/calendar.go:155` through `ui.CalendarCell`
  (`calendarcell.go:24`); `internal/feature/journalcards/period.go:86` through
  `ui.Breadcrumb` (`breadcrumb.go:16`); the live nudge surface
  (`internal/web/nudge.go:21`) through `ui.NudgeBanner` (`nudgebanner.go:26`).
- **Delete atom + test + story together** for the ones with no home and no
  decorative role: `FolkBand`/`Stitch` (`spark.go:16,21`), and any of
  `StatCard, Sparkline, Tabs, Pagination, DayEntry, RecapCard, ScreenTitle,
  Tooltip, Toggle, GuardianCard` that you do not wire to a real screen in this
  step. (Per SUCKLESS: do not leave catalog-only components.)
- **`Skeleton`/`Toast`** are honestly self-disclaimed as staged — `Toast` is
  wired in **S7**; leave `Skeleton` only if you keep the disclaimer.

**Enforce so this never regresses:** add a test (mirror `tours_test.go`) that
enumerates `ui.RegisterCard` registrations / exported component funcs and fails
if any lacks a `storybook.Lookup` entry.

**Verify:** `go test ./internal/...`; storybook renders; revived surfaces look
identical-or-better in the browser.

---

## S5 — Fix storybook drift

- **GraphCard story is stale:** `stories_cards.go:306,330` still describe a
  "static 1-hop snapshot… no physics, no interactivity (deferred)", but
  `internal/feature/graphcards/graph.go` shipped a full interactive force-graph
  canvas (`graph-canvas.js`), with the SVG now the `<noscript>`/storybook
  fallback. Rewrite the blurb + the "don't add interactivity (deferred)" note.
- **Coverage gaps:** add stories for `HeadsCard` (entire `headscards` package
  has none) and the 6 registered cards with no story (`today, habits, calendar,
  timeline, measure, lines`). (The S4 coverage test will flag these.)
- **Props drift:** `SettingsFocus` story omits `Heads/Nudge/Capabilities` and
  names a non-existent "Appearance" section (`stories_cards.go:770–779`);
  `DayFocus` omits `ParentURL/ParentLabel` (`stories_cards.go:640–647`). Make
  each Props table 1:1 with its struct.

**Verify:** `go test ./internal/feature/storybook/...`; open each new/edited
story at `/storybook/<id>`.

---

## S6 — Motion win #1: Datastar view-transitions + one motion layer

See `docs/ui-motion-2026.md` (Tier 1: same-document View Transitions; the
"Datastar already does View Transitions on SSE patches" headline).

1. **Confirm the integration detail first:** find the `useViewTransition` option
   name/shape in the **vendored** `internal/web/assets/static/datastar.js`
   version (open question #1 in the research doc). If present, opt the relevant
   SSE patches in; if not, wrap the patch-apply in a 10-line vanilla
   `document.startViewTransition(() => …)` helper, guarded by feature-detection.
2. **Apply to:** chat message append, task-card `Done/Snooze/Drop` optimistic
   transitions, tool-call card expand. Add `view-transition-name` to the moving
   elements.
3. **Pixel-snappy:** override `::view-transition-old/new` easing to a `steps()`
   wipe or opacity **hard-cut** — the default crossfade blur violates the
   no-blur aesthetic.
4. **One motion `@layer` + reduced-motion kill-switch:** consolidate motion into
   a single CSS layer and a single `@media (prefers-reduced-motion: reduce)`
   block that zeroes all durations — this also closes the standing gap where
   `.app-panel` slide, `.toggle-knob`, `.dock-grip::after`, and `.avatar-choice`
   transitions currently escape reduced-motion. Gate the `startViewTransition`
   call behind `matchMedia('(prefers-reduced-motion: reduce)')` (apply patch
   directly under RM).

**Verify:** in-browser, transitions read as snappy hard cuts (no blur); with OS
"reduce motion" on, everything snaps instantly. Update `internal/self/knowledge.md`.

---

## S7 — Motion win #2: entry/exit panels + wire the `Toast` atom

See `docs/ui-motion-2026.md` (Tier 1: entry/exit suite — `@starting-style` +
`transition-behavior: allow-discrete` + `overlay`; Popover API).

1. **Animate focus-panel open/close and the "All pages" overflow** with the
   pure-CSS entry/exit suite (mind the gotchas: `@starting-style` after the
   open-state rule; `allow-discrete` after the `transition` shorthand; include
   `display` + `overlay` for top-layer). `steps()` easing; zero under RM.
   Consider the **Popover API** for the overflow menu / a future command palette.
2. **Wire the `Toast` atom** (`internal/ui/toast.go`, currently unused): add a
   `#toast` SSE region to the shell (`internal/ui/shell`) and emit a `Toast` on
   owner actions that today give no feedback (task transition, memory approve).
   Animate it in/out with the same entry/exit suite. Add/keep its story.

**Verify:** in-browser, panels/menus snap open/closed with no blur; a task
transition shows a toast that auto-dismisses; RM disables the motion. Update
`internal/self/knowledge.md` (new `#toast` capability). Update the `Toast`/
`Skeleton` story disclaimers.

---

## Out of scope (recorded so they aren't re-audited as gaps)

- Memory-facet nav promotion, graph legibility/edges, avatar 16-bit idle
  animation, scroll-driven reveals, MPA view transitions, Anchor Positioning
  (status unverified — see research doc). Each is a clean follow-on.
- The AI-generated ANSI/code-art avatar idea (explored, parked 2026-06-24 —
  keeping the existing PNG avatars for now).
