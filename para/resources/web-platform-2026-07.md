---
title: Web platform research, July 2026 (Chromium 150)
created: 2026-07-24
status: reference
companion: web-platform-2026-07-probe.json
---

# Web platform research, July 2026

What the platform gives a vanilla-JS, no-build-step app like Balaur right now, verified rather than recalled.

## 1. Method and environment

| Item | Value |
|---|---|
| Research date | 2026-07-24 |
| Current Chrome stable | 150.0.7871.186 (versionhistory.googleapis.com) |
| Probe engine | HeadlessChrome/150.0.0.0, feature detection over live DOM |
| Cross-browser versions | MDN browser-compat-data 8.0.8 (npm snapshot, same day) |
| ECMAScript pipeline | tc39/proposals README plus finished-proposals (July 2026 state) |
| Shipping status | chromestatus.com API plus developer.chrome.com docs |

Two rounds of in-browser feature detection ran against a local page; round 2 fixed round-1 probe artifacts (single-argument `CSS.supports` misuse, `typeof` on accessor getters, headless-only gaps). Raw results: [`web-platform-2026-07-probe.json`](web-platform-2026-07-probe.json). Headless caveats that are NOT missing features: `navigator.share` (real Chrome desktop has it since 89), `BarcodeDetector` (Linux desktop only), `navigator.ml` (no WebNN backend on Linux), `self.ai` (needs origin-trial enrollment plus a downloaded Gemini Nano model).

Balaur already uses: same-document View Transitions with a reduced-motion guard (`app.js:149`), popover for the add menu, native `dialog.showModal()`, `inert`, `matchMedia` for motion/width/transparency/contrast, custom elements, and a versioned shell-caching Service Worker (`sw.js`, `orbit-shell-v12`). Everything below is delta, not a restatement of the baseline.

## 2. Executive summary

Do now (ranked by impact per effort):

1. **Anchor positioning for every floating surface** (menus, inspector, tooltips). Chrome 125, Firefox 147, Safari 26: Baseline 2025. Replaces manual `getBoundingClientRect` math (`app.js:1011`, `app.js:1086`) with `anchor-name` on the card and `position-area`/`position-try-fallbacks` on the menu. Flip-side handling and viewport clamping come free.
2. **Dialog and menu ergonomics batch**: `closedby="any"` on dialogs (Chrome 134), invoker buttons `commandfor`/`command` (Chrome 135), `CloseWatcher` (Chrome 126), `:open` pseudo (Chrome 133), `beforetoggle`/`toggle` events. Removes hand-rolled close wiring and outside-click handlers.
3. **Built-in AI as a zero-key provider path.** Summarizer, Translator, and Language Detector are Chrome-stable since 138; the Prompt API is stable for extensions and in origin trial for web pages. A `built-in` provider behind the existing operator interface delivers local-first, keyless AI on Gemini Nano devices, with graceful fallback to the configured provider. This is the single biggest alignment between the platform and Balaur's AI-canvas architecture (§10).
4. **Rendering budget for large canvases**: `content-visibility: auto` plus `contain-intrinsic-size` on card elements (Chrome 85, Firefox 125, Safari 18), measured with Long Animation Frames (`long-animation-frame`, Chrome 123) instead of longtask. `scheduler.yield()` (Chrome 129) between render batches keeps pointer input responsive. The browser-check skill can assert LoAF budgets directly.
5. **Temporal for new date code, behind detection.** Temporal shipped in Chrome 144 and Firefox 139 (Safari preview only). It is the correct tool for the §4.4 conventions (local dates, IANA zones, scheduling intent vs deadline), but Safari stable lacks it, so adopt it in new paths (event timezone math, journal navigation) with the existing `localDateISO()` helpers as fallback, not as a wholesale rewrite.

Later, or nice to have:

1. **View Transition types** (`startViewTransition({ types })` plus `:active-view-transition-type()`): replaces the `body[data-canvas-navigation]` attribute dance in `styles/motion.css` with typed transitions. Chrome 125, Firefox 144, Safari 18.
2. **WebMCP** (origin trial from Chrome 149, flag `enable-webmcp-testing`): pages expose agent-controlled UI paths. Strategically interesting for an AI canvas app, but the API is early and Chromium-only. Watch, do not build.
3. **Sanitizer API** (Chrome 146, Firefox 148, no Safari) as defense in depth for any path that turns AI or imported content into HTML. Balaur already escapes; Sanitizer is a cheap extra layer where `setHTML` fits.
4. **Storage Buckets** (Chrome 122, Chromium-only) to partition the vault from caches and give IndexedDB eviction policy per bucket.
5. **Scroll-driven animations** (Chrome 115, Safari 26, Firefox preview) for sidebar and nav polish as pure progressive enhancement.

## 3. ECMAScript

TC39's July 2026 state: ES2025 is fully shipped, the ES2026 set is mostly shipped, and Temporal headlines the ES2027-bound finished proposals. A new "stage 2.7" tier now sits between stage 2 and stage 3 (decorators and ShadowRealm live there).

### 3.1 Shipped and usable today (all three engines, or with a trivial guard)

| Feature | Chrome | Firefox | Safari | Balaur relevance |
|---|---|---|---|---|
| `Object.groupBy` / `Map.groupBy` | 117 | 119 | 17.4 | Index projections, Today grouping |
| `Promise.withResolvers` | 119 | 121 | 17.4 | Cleaner `orbitVaultReady` boot promise |
| `Promise.try` | 128 | 134 | 18.2 | Uniform sync/async repository entry points |
| Set methods (`union`, `intersection`, `difference`, ...) | 122 | 127 | 17 | Placement diffs, canvas reconciliation |
| Iterator helpers (`map`, `filter`, `take`, ...) | 122 | 131 | 18.4 | Lazy graph traversal in `life-query.js` |
| `Iterator.concat` | 146 | 147 | 26.4 | Composing query streams |
| `Map.getOrInsert` / `getOrInsertComputed` | 145 | 144 | 26.2 | Index upserts without has/get/set dance |
| `RegExp.escape` | 136 | 134 | 18.2 | Safe search-term regexes |
| `Uint8Array.toBase64/fromBase64` (incl. base64url), `toHex/fromHex` | 140 | 133 | 18.2 | `content-hash.js` output, export payloads |
| `Math.sumPrecise` | 147 | 137 | 26.2 | Habit-value aggregation |
| `Error.isError` | 134 | 138 | 18.4 | Boundary error checks |
| `Float16Array`, `Math.f16round` | 135 | 129 | 18.2 | Widget data exchange (niche) |
| `Atomics.pause` | 133 | 137 | 18.4 | Spin-wait hygiene (niche) |
| `using` / `await using` (explicit resource management) | 134 | 141 | no | Scoped handles; Safari needs the old path |
| `Array.fromAsync` | 121 | 115 | 16.4 | Vault listing collection |
| `Intl.DurationFormat` | 129 | 136 | 16.4 | Estimate display in Today |
| `Intl.Segmenter`, `Intl.Locale.getTextInfo()` | 87 / yes | yes | yes | Text processing, direction-aware UI |
| `AbortSignal.any` / `timeout` | 116 | 124 | 17.4 | Provider call timeouts in AI operators |
| `fetch` upload streaming (`duplex: 'half'`) | 131 | no | no | Chromium-only; large export streaming |
| `structuredClone`, `Object.hasOwn`, `crypto.randomUUID` | yes | yes | yes | Already baseline |

### 3.2 The headline: Temporal (Chrome 144, Firefox 139, Safari preview)

Verified live: `Temporal.Now`, `PlainDate`, `PlainDateTime`, `ZonedDateTime`, `Instant`, `Duration` all present; the removed `Temporal.Calendar`/`Temporal.TimeZone` classes are gone (spec simplified in 2025); DST arithmetic over `Europe/Bucharest` works. The TC39 finished-proposals list puts Temporal in the ES2027 publication batch alongside Explicit Resource Management, `Atomics.pause`, and Joint Iteration.

For Balaur this is the first credible replacement for the §4.4 date conventions' hand-rolled helpers: `Temporal.PlainDate` is exactly the "local date `YYYY-MM-DD`" type, and `Temporal.ZonedDateTime` with IANA names is exactly the calendar-event model. Constraint: Safari stable does not ship it, and the GitHub Pages site serves Safari users. Recommendation: feature-detect and use it in new date logic (event timezone rendering, journal date navigation, recurrence previews) with the existing helpers as fallback; revisit a full switch when Temporal crosses into Baseline-wide.

### 3.3 Chromium-first or single-engine (do not depend on)

| Feature | Status |
|---|---|
| `Observable` / `Observable.from` (TC39 Observable) | Present in Chrome 150, not even tracked in BCD yet. Too early to adopt. |
| `Set.getOrInsert` / `getOrInsertComputed` | Map variants shipped (Chrome 145); Set variants absent in 150. |
| `Iterator.zip`, `zipKeyed`, `chunk`, `includes`, `join`, `range` | Stage 3; only `zip` ships anywhere (Firefox 148). |
| `JSON.rawJSON` | Shipped everywhere relevant (ES2025); useful if canvas patching ever needs raw-number preservation. |

### 3.4 Not shipped, stalled, or dead (negative results worth knowing)

| Feature | July 2026 state |
|---|---|
| Decorators | Stage 2.7, `SyntaxError` in Chrome 150. Still transpiler territory. |
| `import defer` (deferred module evaluation) | Stage 3, not implemented. |
| `import source` (source phase imports) | Stage 3, not implemented. |
| AsyncContext / AsyncLocalStorage | Demoted back to stage 2. Do not plan around it. |
| ShadowRealm | Stage 2.7, not implemented. |
| Signals | Stage 1. |
| Records and Tuples | Withdrawn. |
| Pattern matching, Decimal, Pipeline | Stage 2 / stage 1, years away. |

## 4. CSS

### 4.1 Positioning and layout (the practical wins)

| Feature | Chrome | Firefox | Safari | Note |
|---|---|---|---|---|
| Anchor positioning (`anchor-name`, `position-anchor`, `position-area`, `position-try-fallbacks`, `position-visibility`, `anchor-size()`) | 125 (area: 129, anchor: 144) | 147/151 | 26 | Baseline 2025. Use for card menus, inspector anchoring, tooltips. |
| Container queries, size plus `style()` queries, `cqw` units | 111/114 | 110+ | 16 | Style queries need no `container-type: style` (that value left the spec; probe confirms). |
| `container-type: scroll-state` plus `@container scroll-state(stuck: ...)` | 133 | no | no | Chromium-only. Stuck headers in the sidebar/Today panel. |
| Subgrid | 117 | 71 | 16 | Baseline. Inspector and dialog grids. |
| Nesting, `@scope`, `@layer` | 120 / 118 / 99 | 117 / 146 / yes | 17.2 / 17.4 / yes | Balaur already layers; nesting is safe in shipped CSS with the relaxed syntax. |
| `interpolate-size: allow-keywords` plus `calc-size()` | 129 | no | no | Chromium-only. Animates `height: auto` panels (Today, inspector) without JS measurement. |
| `field-sizing: content` | 123 | 152 | 26.2 | Auto-growing textareas (task notes, journal) with no JS. |
| `fit-content()` function, masonry | no | no | 26.2 / no | Not in Chromium. |

### 4.2 Color and theming

All Baseline: `color-mix()` (111), `oklch`/`oklab` (111), relative color syntax (119), `light-dark()` (123), `color-scheme` (81). For a token-driven theme layer (`styles/tokens.css`, `styles/themes.css`) this means: derive hover/pressed/disabled shades from a token with `color-mix(in oklab, var(--x), white 12%)` instead of shipping shade variants, and collapse light/dark token pairs with `light-dark()` where the theme maps to `color-scheme`. `color-contrast()` is still unshipped anywhere.

### 4.3 Motion

| Feature | Chrome | Firefox | Safari | Note |
|---|---|---|---|---|
| Same-document View Transitions, `view-transition-name` | 111 | 144 | 18 | Already used for portal navigation. |
| `view-transition-class`, `:active-view-transition-type()` | 125 | 144 | 18 | Typed transitions can replace the `data-canvas-navigation` body attribute. |
| `@view-transition { navigation: auto }` (MPA) | 126 | no | 18.2 | Irrelevant to an SPA shell. |
| Scroll-driven animations (`animation-timeline: scroll()/view()`, `timeline-scope`) | 115 | preview | 26 | Progress indicators, edge-fade on scrollable panels. Guard with `@supports`. |
| `@starting-style` plus `transition-behavior: allow-discrete` plus `overlay` | 117 | 129 | 17.4/17.5 (overlay: no) | CSS-only enter/exit transitions for dialogs and popovers. `overlay` is Chromium-only; elsewhere the exit transition needs the old `display` workaround. |
| `@property` | 85 | 128 | 16.4 | Registered custom properties for smooth theme-token animation. |

Balaur's existing `prefers-reduced-motion` guard (`app.js:146`, `styles/motion.css`) is the right pattern for all of the above; scroll-driven and VT additions slot under it unchanged.

### 4.4 Typography and forms

`text-wrap: balance` (114) and `pretty` (117), `text-box` trim (133, Safari 18.2, Firefox preview), `initial-letter` (110, Safari 9, no Firefox), `font-variant-emoji` (131), `text-autospace` (140), `text-spacing-trim` (123, Chromium-only), `::spelling-error` (121, no Firefox), `accent-color`, `caret-color`. `hanging-punctuation` is Safari-only (26.5); skip. Select customization is real now: `appearance: base-select`, `select::picker(select)`, and `<selectedcontent>` (Chrome 135, Safari 27, Firefox 149 for base-select) can restyle the canvas/status selects in `index.html` without a custom dropdown, and `:open` (Chrome 133) styles the open state. `:closed` is not in Chrome 150 despite earlier plans.

### 4.5 Functions and selectors

`if()` (Chrome 137, Chromium-only), `sibling-index()`/`sibling-count()` (Chrome 138, Safari 26.2, Firefox preview), typed `attr()` (Chrome 133), `random()` (Safari 26.2 only, not Chrome), full math (`trig`, `round/mod/rem`, `hypot/sqrt/pow/exp/log`, `sign/abs`) is Baseline. Selectors: `:has()` with relative syntax, `:user-valid`/`:user-invalid` (119), `:dir()` (120), `:modal`, `:state()` for custom-element states (125, pairs with `ElementInternals.states`), `:popover-open`, `::target-text` (89, useful for canvas text search highlighting together with the Custom Highlight API).

### 4.6 Rendering performance

`content-visibility: auto` (85/125/18) with `contain-intrinsic-size` (83/107/17) and the `contentvisibilityautostatechange` event (108/130/18) is the highest-leverage item for canvases with many cards: skip layout/paint for off-viewport cards while keeping scroll geometry stable. `contain: layout paint` on cards, `scrollbar-color`/`scrollbar-width` (121) for themed scrollbars, `overflow: clip`, `backdrop-filter` (76) for panel translucency under the existing `prefers-reduced-transparency` guard.

### 4.7 Chromium-only CSS (enhancement, never dependency)

`corner-shape` incl. `superellipse()` (139), Paint API `CSS.paintWorklet` (desktop only; a dot-grid canvas background without DOM), `object-view-box` (104), `overlay` (117), `text-spacing-trim` (123), `scroll-state()` queries (133), `if()` (137), `calc-size()`/`interpolate-size` (129), `device-posture` media (132), `prefers-reduced-data` (85). Also inverted: `inverted-colors` media is NOT in Chrome (Firefox/Safari only), and `hanging-punctuation` and `random()` are Safari-only.

## 5. Platform APIs

### 5.1 UI primitives (all verified live)

| Feature | Chrome | Firefox | Safari | Replaces |
|---|---|---|---|---|
| `commandfor`/`command` invoker buttons | 135 | 144 | 26.2 | JS click handlers that open dialogs/popovers |
| `closedby` on `dialog` plus `CloseWatcher` | 134 / 126 | 141 / 149 | preview | Manual Escape/outside-click handling |
| `popover` plus `beforetoggle`/`toggle` | 114 | 125 | 17 | Already adopted for the add menu |
| `popover="hint"` | 151 (next release) | 153 | no | Hover cards, validation bubbles; feature-detect |
| `details[name]` exclusive accordion | 120 | 130 | 17.2 | Sidebar sections without JS |
| `showPicker()` | 99 | 101 | 16 | Programmatic date/color pickers for scheduling |
| `ElementInternals.states` plus `:state()` | 125 | 126 | 17.4 | State styling for the custom elements in `elements/` |
| `autofocus` on any element, `beforematch` | 122 / 105 | yes | yes | Find-in-page targeting of cards |

### 5.2 Navigation, windows, PWA

Navigation API is Baseline (Chrome 102, Firefox 147, Safari 26.2): `navigation.currentEntry`, `onnavigate`, `activation`. It could give `enterSubcanvas`/`leaveSubcanvas` real browser-back behavior, but it changes history semantics for a canvas app, so it belongs in a grill, not a ticket. Document Picture-in-Picture (Chrome 116, Firefox 151, no Safari) is a plausible home for the Today panel (a floating always-on-top mini Today), filed under nice-to-have. Also present: `launchQueue` (102), Window Controls Overlay (105), Badging (81, no Firefox), `getInstalledRelatedApps` (85), Speculation Rules (109). `beforeinstallprompt` still exists but Chrome has deprecated it; the repo does not use it, and new install UX should rely on `appinstalled` plus an in-app button.

### 5.3 Storage and files

OPFS (stable since 102), Storage Buckets (122, Chromium-only), Web Locks (69), BroadcastChannel (54), File System Access `showDirectoryPicker`/`showSaveFilePicker` (86, desktop Chrome only), `Blob.bytes()` (144), `CompressionStream` with gzip/deflate (80). **zstd is NOT available**: the probe gets "Unsupported compression format: 'zstd'" in Chrome 150, and BCD does not track it anywhere. Whole-space backup compression (§4.5) should target gzip if size ever matters. Web Locks plus BroadcastChannel are the relevant pair for multi-tab coherence against the IndexedDB vault: the optimistic `expectedHash` precondition already detects conflicts, and a BroadcastChannel "vault changed" ping would let a second tab reconcile instead of failing its next write.

### 5.4 Service Worker: a retraction

The Static Routing API (`registration.router`) shipped in Chrome 121 and is **gone in Chrome 150**: the property is absent from `ServiceWorkerRegistration.prototype` (verified by property enumeration), and BCD no longer tracks it. Do not build `sw.js` on it. The current `sw.js` design (navigation preload, network-first with shell-only cache writes, authorization-header exclusion) remains the correct shape; the only standing improvement is the usual `APP_SHELL` drift discipline, and `cache.addAll` chunking if the list keeps growing.

### 5.5 Performance observability

Long Animation Frames (`long-animation-frame`) is Chrome 123 and Chromium-only; it supersedes longtask and attributes blocking time to scripts with `PerformanceScriptTiming` detail. `visibility-state` entries, Event Timing, LCP, and layout-shift are all observable. `scheduler.postTask` (94) and `scheduler.yield` (129, Firefox 142, no Safari) give priority-aware batching for render passes. `scrollend` (114) is Baseline; **scroll snap events (`scrollsnapchanged`) are absent in Chrome 150**, another quiet retraction to record. The browser-check skill can add a LoAF-based first-render budget assertion on top of its existing smoke suite.

### 5.6 Canvas, graphics, input

WebGPU is live in Chrome 150 on Linux (`navigator.gpu.requestAdapter` present; BCD marks the interface Baseline at 144, Firefox 141, Safari 26). For widgets, the §10 stance (WebGL2 inside self-contained widgets with 2D fallbacks) stays right: WebGPU is an optional upgrade path for `widgets/focus-orbit.html`-class widgets, feature-detected, never required. Canvas2D: `roundRect`, `letterSpacing`, `Path2D.addPath` present; **Canvas2D layers (`beginLayer`/`endLayer`) were removed** after shipping in Chrome 123. Pointer events expose `getCoalescedEvents`/`getPredictedEvents` for smoother drag previews. Custom Highlight API (`CSS.highlights`, Chrome 105, Firefox 140, Safari 17.2) plus `::target-text` can render future canvas search highlights without mutating card DOM. EyeDropper (95) and Local Font Access (103) remain Chromium-only desktop curiosities; EditContext (121) is irrelevant because Balaur uses native inputs.

### 5.7 Security

Sanitizer API shipped: constructible `Sanitizer`, `Element.setHTML` (Chrome 146, Firefox 148, no Safari), plus `setHTMLUnsafe`/`parseHTMLUnsafe` for trusted content. `Sanitizer.getDefaultConfiguration` is absent in this build, so construct an instance. Trusted Types is now Baseline (83/148/26). Neither changes the §10 boundary (sandboxed widgets, escaped HTML), but `setHTML` is a strictly better primitive than string-escaping for any future path that inserts AI-authored rich content into host cards.

## 6. On-device AI: the section that matters most for Balaur

Verified status from developer.chrome.com and BCD, July 2026:

| API | Status | Chrome | Engines |
|---|---|---|---|
| Summarizer | **Stable** | 138+ | Chrome only, Gemini Nano, geo-restricted |
| Translator | **Stable** | 138+ | Chrome only, Gemini Nano, geo-restricted |
| Language Detector | **Stable** | 138+ | Chrome only, Gemini Nano, geo-restricted |
| Prompt API (`ai.languageModel`) | Stable for **extensions only**; origin trial for web pages | 138+ | Chrome, Gemini Nano |
| Writer / Rewriter | Origin trial | 138+ era | Chrome |
| Proofreader | Origin trial (new) | recent | Chrome |
| WebMCP (`navigator.modelContext`) | Origin trial from Chrome 149; local flag `chrome://flags/#enable-webmcp-testing` | 149+ | Chrome |
| WebNN (`navigator.ml`) | Origin trial since 112; enabled on Windows/ChromeOS/Android; **absent on Linux and macOS desktop** | 112+ (OT) | Chrome |

All of these require a downloaded Gemini Nano model and degrade gracefully when `self.ai` is undefined or `ai.languageModel.capabilities()` reports unavailable.

Fit with Balaur's §10 architecture: the AI operators are already provider-abstracted text nodes with edge-derived context and a confirm-before-apply operation pipeline. A `built-in` provider slot would:

1. Feature-detect `self.ai?.languageModel` (Prompt API origin trial) for free-form operators, and `ai.summarizer`/`ai.translator` (stable) for the summarization/translation operators specifically.
2. Keep provider keys in `sessionStorage` exactly as today; the built-in path simply has no key, which strengthens the local-first story (no network, no key management, works offline once the model is downloaded).
3. Fall back to the configured HTTP provider when the model is unavailable, with the same allowlisted-operation validation applied to both outputs.

Constraints to respect: origin-trial enrollment is required for the Prompt API on the GitHub Pages origin (Summarizer/Translator/LanguageDetector need no enrollment), model availability is device- and region-dependent, and none of this exists in Firefox or Safari, so the provider selector must treat built-in as one optional entry, never the default assumption. WebMCP is worth a grill in 2027 at the earliest: an app-controlled agent UI surface is philosophically aligned with "AI output never mutates the host without confirmation", but the API is pre-standardization.

## 7. Negative results (checked, and not there)

| Item | Finding |
|---|---|
| Service Worker Static Routing API | Removed after Chrome 121; absent in 150. Do not use. |
| Canvas2D layers (`beginLayer`) | Removed after Chrome 123. |
| `CompressionStream('zstd')` | Unsupported in Chrome 150; gzip/deflate only. |
| Scroll snap events (`scrollsnapchanged`) | Absent in Chrome 150. |
| `SharedArrayBuffer` | Requires COOP/COEP; GitHub Pages cannot provide them. This re-confirms the AGENTS.md §7 position on OPFS SQLite Wasm. |
| `random()`, `hanging-punctuation` | Safari-only. |
| `fit-content()`, `color-contrast()`, masonry, `:closed`, Layout/Animation Worklets | Not shipped in Chromium. |
| `inverted-colors` media query | Not in Chrome (Firefox/Safari only). |
| Decorators, `import defer`, `import source`, AsyncContext, ShadowRealm | Not implemented; stages 2 to 3. |
| `Set.getOrInsert` | Absent while the Map variants shipped. |
| `popover="hint"` | Ships in Chrome 151, one release ahead; detect, do not assume. |
| Wasm GC JS API (`WebAssembly.Struct`) | Not exposed in Chrome 150 (moot for Balaur; recorded for completeness). |

## 8. Balaur adoption map

| Subsystem | Opportunity | Feature | Chrome | Effort | Caveat |
|---|---|---|---|---|---|
| Canvas engine | Floating menus/inspector anchoring | Anchor positioning | 125 | M | Baseline 2025; safe with `@supports` |
| Canvas engine | Typed portal transitions | VT `types` plus `:active-view-transition-type()` | 125 | S | Replaces body-attribute styling |
| Canvas engine | Off-viewport card skip | `content-visibility` plus `contain-intrinsic-size` | 85 | M | Cards are absolutely positioned; measure scroll geometry after enabling |
| Shell | Dialog close handling | `closedby`, invokers, `CloseWatcher` | 126-135 | S | Safari preview only for CloseWatcher |
| Shell | Auto-growing inputs | `field-sizing: content` | 123 | S | Keep JS autosize fallback for Firefox <152 |
| Shell | Theme shade derivation | `color-mix`, `light-dark()`, relative colors | 111-123 | M | Baseline; token refactor |
| Today | Render budget | LoAF plus `scheduler.yield` | 123/129 | S | Chromium-only observability; yield needs fallback |
| Storage | Multi-tab coherence | BroadcastChannel plus Web Locks | 54/69 | M | Baseline everywhere |
| Storage | Backup compression | `CompressionStream` gzip (no zstd) | 80 | S | Optional; bundles are small |
| Dates | New date logic | Temporal with helper fallback | 144 | M | Safari preview only |
| AI | Keyless local provider | Summarizer/Translator stable; Prompt API OT | 138+ | L | Origin trial plus Nano availability |
| AI | Agent UI surface | WebMCP | 149 (OT) | watch | Pre-standardization |
| Security | Rich-content insertion | Sanitizer plus `setHTML` | 146 | S | No Safari; keep escaping as the floor |
| Offline | SW strategy | Keep current design (static routing retracted) | n/a | none | `APP_SHELL` drift discipline only |

Effort: S under half a day, M a day or two, L a short project.

## 9. Cross-browser strategy

Balaur deploys to GitHub Pages for all browsers, so the rule stays progressive enhancement: Baseline features (popover, `:has`, anchor positioning, View Transitions, container queries, `color-mix`, dialog, inert, Set methods, iterator helpers) can be used unconditionally; Chromium-only items (Paint API, `corner-shape`, `if()`, `calc-size`, `overlay`, scroll-state queries, LoAF, Storage Buckets, EditContext, EyeDropper, built-in AI, WebNN) go behind `@supports`/feature detection with an equivalent behavior path; Safari-gap items (`overlay`, CloseWatcher, Sanitizer, Document PiP, `using`) need the old-path fallback to remain functional, not merely present. The existing `prefers-reduced-motion`/`prefers-reduced-transparency`/`prefers-contrast` guards in `app.js` and `styles/motion.css` are the template to copy.

## 10. Sources (accessed 2026-07-24)

- Live feature detection, HeadlessChrome/150.0.0.0: [`web-platform-2026-07-probe.json`](web-platform-2026-07-probe.json)
- Chrome stable version: https://versionhistory.googleapis.com/v1/chrome/platforms/linux/channels/stable/versions
- MDN browser-compat-data 8.0.8 (npm snapshot, queried locally)
- TC39 proposals: https://github.com/tc39/proposals (README plus finished-proposals.md)
- Chrome Status API: https://chromestatus.com/api/v0/features (Sanitizer enabled by default; WebMCP, Prompt API, WebNN proposed/OT)
- Built-in AI status: https://developer.chrome.com/docs/ai/built-in-apis (Summarizer/Translator/Language Detector stable at 138; Prompt API stable for extensions, OT for web; Proofreader OT)
- WebMCP: https://developer.chrome.com/docs/ai/webmcp (origin trial from Chrome 149, flag `enable-webmcp-testing`)
