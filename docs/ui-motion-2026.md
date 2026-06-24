# Modern web motion & interactivity for Balaur (2026 snapshot)

Durable record of a deep-research pass on the latest browser animation/interaction
standards, scoped to Balaur's exact stack and aesthetic. Compiled **2026-06-24**
(deep-research harness: 5 angles, 24 sources, 25 claims adversarially verified →
23 confirmed / 2 killed). **Baseline status is a moving target** — re-check the
"watch" items against caniuse / web-features before relying on them.

## Scope this was researched against

- **Stack:** server-rendered Go `gomponents`, DOM patched over **Datastar** SSE
  (hypermedia + signals, not a SPA). **No Node / no build step.** CSS is one
  hand-written `basm.css`; JS is a few small vanilla files.
- **Support bar:** *Baseline Newly Available* — must work cross-engine
  (Chromium + WebKit/Safari + Gecko/Firefox), recent OK, progressively enhance
  older. No single-engine features as load-bearing.
- **Motion character:** *pixel-true & snappy* — 16-bit game feel: `steps()` /
  `linear()` easing, hard cuts, **no blur, no soft fade, no spring, no parallax.**
- **Hard a11y bar:** every animation fully neutralized under
  `prefers-reduced-motion`.

## Headline finding: Datastar already does View Transitions on SSE patches

Datastar's own docs (`data-star.dev/examples/view_transition_api`,
`/reference/sse_events`, and the `datastar-go` package) show it wraps an
SSE-driven DOM swap in `document.startViewTransition()` automatically when the
browser supports it **and the patch opts in** (a `useViewTransition` option on
the patch event). So most server-driven motion — chat append, task-card
Done/Snooze/Drop, tool-card expand — is a **per-event flag, not new JS**.
*(Documented in primary Datastar sources; not part of the 3-0 adversarial
subset — confirm the exact option name against the vendored `datastar.js`
version before building.)*

## Tier 1 — adopt now (confirmed cross-engine Baseline)

| Capability | Baseline | Balaur surface | Pixel-snappy / reduced-motion |
|---|---|---|---|
| **Entry/exit suite** — `@starting-style` + `transition-behavior: allow-discrete` + `overlay` | Newly, **2024-08-06** (Firefox 129 closed it) | Focus-panel open/close, "All pages" overflow, tooltips, command palette, wiring the unused `Toast` atom | `steps()`/`linear()` easing; `transition-duration:0` under RM. Gotchas: `@starting-style` must come **after** the open-state rule (equal specificity); `allow-discrete` must be declared **after** the `transition` shorthand or it resets to `normal`; top-layer entry/exit needs `display` **and** `overlay` in the transition list with `allow-discrete`. |
| **Same-document View Transitions** (`startViewTransition`, `view-transition-class`) | Newly, **2025-10-14** (Firefox 144) | Chat append, task-card reorder/optimistic state, tool-card expand — via Datastar's flag | ⚠️ Default `::view-transition-old/new` crossfade is **blurry** → override easing to a `steps()` wipe or opacity hard-cut. Gate the call behind `matchMedia('(prefers-reduced-motion: reduce)')` and apply the patch directly under RM. |
| **Container queries** (`@container`, `cqi/cqw`) | **Widely**, 2025-08 | The fix for the chat dock **rail-vs-canvas** (and the rail-collapse bug); cards/sparklines adapting to slot width | Layout, not motion — no aesthetic/RM concern. Lowest-risk, highest-value. |
| **Popover API** + `::backdrop` | Newly, **2025-01-27** | "All pages" overflow, tooltips, command palette — declarative top-layer + light-dismiss, no JS state | Flat (non-blur) scrim; `steps()` entry; snap under RM. Compose with the entry/exit suite. |
| **`@property`** (typed custom props) | Newly, **2024-07-09** | Discrete color/angle/number animation on badges, recall-pips, streak counters | Inherently steppable. |
| **`grid-template-rows: 0fr→1fr`** (single-row grid + `overflow:hidden` child) | Long-Baseline | Tool-call "arguments" expander (the **fallback** for `interpolate-size`) | `steps()` reveal; snap under RM. |
| **`steps()` / `image-rendering: pixelated` / WAAPI `element.animate` / `prefers-reduced-motion`** | Long-Baseline | Avatar 16-bit idle/blink (sprite + stepped `background-position`), FLIP, the **global motion-off switch** | This *is* the pixel-true toolkit. |

## Tier 2 — progressive enhancement (no-ops cleanly where unsupported)

- **Cross-document (MPA) View Transitions** (`@view-transition { navigation: auto }`,
  both docs opt in) — Chrome/Edge 126+, Safari 18.2 (Dec 2024), **Firefox
  missing** in 2026. Only relevant for *real* navigations (Balaur's are mostly
  SSE patches → low priority). Pure CSS opt-in; degrades to a hard cut.
- **Scroll-driven animations** (`animation-timeline: scroll()/view()`) — Chrome
  115+, Safari 26, **Firefox flagged**. Reveal-on-scroll / progress on long
  lists. Content stays visible if unsupported. Use `steps()` bands — **never
  parallax** (forbidden by the aesthetic).

## Tier 3 — watch / not ready

- **`interpolate-size: allow-keywords` + `calc-size()`** (animate to `height:auto`)
  — **Chromium-only, explicitly NOT Baseline** (Chrome/Edge 129+, Sep 2024);
  Firefox + Safari unsupported through 2026. → use the `0fr→1fr` grid trick.
- **CSS Anchor Positioning** (`anchor-name`, `position-area`, `@position-try`) —
  cross-engine status **could not be established** here (the Interop-2026
  carryover claim was refuted/split-voted; Chromium-led, Safari/Firefox 2026
  unclear). For tooltip/overflow placement, **verify separately** and keep a
  small JS positioning fallback.
- **Popover invoker commands** (`command` / `commandfor`) — not confirmed
  cross-engine; keep a tiny vanilla-JS toggle for open/close.

## Motion architecture (one place)

Put all motion in a single CSS `@layer`, and inside one
`@media (prefers-reduced-motion: reduce)` block zero every
`animation-duration` / `transition-duration` globally, then layer targeted
exceptions. This also closes the standing a11y gap where several transitions
escape reduced-motion — solve it structurally, once.

## Open questions to nail before building

1. Exact Datastar `useViewTransition` option name/shape in the **vendored**
   `datastar.js` version (the load-bearing integration detail).
2. CSS Anchor Positioning Safari/Firefox 2026 status (drives tooltip/overflow
   placement strategy).
3. Canvas vs SSE reconciliation for the graph/sparklines — canvas is opaque to
   Datastar's DOM diffing, so redraws need their own hook.
4. Whether Popover invoker commands are cross-engine, or a JS toggle is still
   needed.

## Recommended sequencing

1. **Container queries → fix rail-collapse** (closes a real responsive bug *and*
   introduces the slot-adaptive pattern). Pure layout, zero risk. First move.
2. **Datastar `useViewTransition`** on chat append + task-card actions, with a
   `steps()` override of the default fade + RM gate. Big polish gain, ~no new JS.
3. **Entry/exit suite** for panels + Popover for overflow/command-palette; wire
   the unused `Toast` atom through it.
4. **Avatar 16-bit idle/blink** (sprite + `steps()`, `image-rendering: pixelated`)
   — keeps the existing PNGs, adds life.

## Sources (primary unless noted)

- web.dev/blog/baseline-entry-animations · developer.chrome.com/blog/entry-exit-animations
- web.dev/baseline/2025 · web.dev/blog/baseline-digest-aug-2025 · web.dev/blog/popover-baseline · web.dev/blog/at-property-baseline
- MDN: `@starting-style`, `transition-behavior`, `interpolate-size`, `View_Transition_API`, `@view-transition`, `animation-timeline`, `Popover_API`, `prefers-reduced-motion`, Games/Crisp_pixel_art_look
- developer.chrome.com/docs/css-ui/animate-to-height-auto · webkit.org/blog/17818 (Interop 2026)
- Datastar: data-star.dev/examples/view_transition_api, /reference/sse_events, pkg.go.dev/github.com/starfederation/datastar-go · htmx.org/essays/view-transitions
- css-tricks.com/css-grid-can-do-auto-height-transitions (blog) · joshwcomeau.com/animation/sprites (blog) · teamtreehouse sprite-sheet steps (blog)

**Refuted/uncertain in this pass:** two Interop-2026 carryover claims (incl.
Anchor Positioning's exact status) were refuted/split — no firm cross-engine
claim is made about Anchor Positioning. Not independently verified here:
invoker commands, `:has()`/`:focus-visible`/`field-sizing`, `content-visibility`
perf, `prefers-reduced-data`, APNG-vs-sprite — confirm before adoption.
