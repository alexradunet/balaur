# gomponents + feature modules + reactive cards — design spec

**Date:** 2026-06-14
**Status:** Approved (brainstorm) — ready for implementation planning
**Predecessors:** `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md` (card-first), the HTMX→Datastar migration.
**Supersedes / amends:** the card-first boards model (user-editable drag/resize boards are retired); §4.6 "cross-card propagation out of scope" is partially reopened **for agent actions only** (§4.6 below).

## 1. Goal

Make Balaur's card UI **simple, type-safe, and modular** without losing its
interactive feel, and let the **agent's actions reflect in the UI on the spot**.
Three threads, one program:

1. **Renderer:** replace stringly-typed `html/template` with **gomponents**
   (pure-Go, compile-checked HTML components), keeping **Datastar** as the
   interactivity layer.
2. **Architecture + UX:** split the flat `package web` into **per-feature Go
   packages** (logic + UI co-located), and make a page a **developer-composed
   set of cards** — dropping user drag/resize/free-form board editing entirely.
3. **Reactive + generative UI:** when the agent mutates state (e.g. completes a
   task) the affected on-screen cards **refresh live** in the open chat stream;
   keep the existing `card_show` catalog as the "AI responds with hypermedia"
   path (no Genkit, no model-authored HTML).

**Success criterion:** every UI surface renders through typed gomponents
components owned by a feature package; there is no user-facing board editor or
drag/resize; an agent task mutation visibly updates the on-screen card(s) within
the same turn; `go build` stays tool-free and CGO-free; the whole thing is
smaller than what it replaces.

## 2. What already exists (do not rebuild)

Grounded in the codebase as of this spec:

- **Datastar SSE spine.** Handlers render an HTML fragment to a string and call
  `sse.PatchElements(html, WithSelectorID(id), WithMode*)`; the chat turn holds
  **one** `datastar.NewSSE` stream open for the whole turn, synchronously, with
  the agent loop emitting inline (`internal/web/chatstream.go`,
  `internal/agent/agent.go:107-112`). This spine is renderer-agnostic and stays.
- **Typed card registry.** `internal/cards` (Spec/ParamSpec/Card/`Validate`/
  `All`/`Get`/`HasManage`) is a stdlib-only leaf with **no `internal/web`
  import**, so `internal/tools` validates agent-composed cards without a cycle.
  This boundary is load-bearing and is preserved.
- **Generative-UI catalog (already the industry-correct pattern).**
  `internal/tools/ui.go` exposes `card_show`/`board_compose`/`board_add_card`;
  the model emits a **structured tool call** (type + params), `cards.Validate`
  is the firewall, `cardHTML`/`cardInto` server-renders, and the result is
  appended inline to `#chat` (`chat-inline-card`). This is the same shape as
  Vercel `streamUI` / Thesys C1 / Google A2UI's Component Firewall — the
  approach all of them endorse over raw model HTML. **Part A of the request is
  already met by this.**
- **Stable card ids.** Every card root carries `id="ucard-{type}"`
  (`web/templates/cards.html`), and a selector-less `PatchElements` morph
  (`chatstream.go:90`) patches by root id and **no-ops if absent** — the exact
  "blind patch-if-present" primitive the reactive feature needs.
- **Self-targeting fragment writes (HATEOAS).** Inline forms `@post` to
  endpoints that re-render and patch their own fragment by id (e.g.
  `taskTransition` → `#tcard-{id}`/`#urow-{src}-{id}`).
- **Single-user, local-first, server-authoritative.** Owner-scoped at the
  PocketBase rule boundary, localhost-guarded, no browser→PocketBase REST. The
  code already encodes the no-LLM-HTML rule: `internal/web/journal.go:66` —
  *"Escape text — never `template.HTML` from an LLM response."*

## 3. Decisions (from brainstorm + research)

1. **Keep Datastar.** Its load-bearing strength is server-driven, multi-element
   token-streaming chat; no single alternative replicates it, and switching
   means assembling htmx + SSE-ext + idiomorph + Alpine (≈24KB, two mental
   models) — a regression against KISS/grug/suckless. *Maintenance, not a
   decision: bump vendored `datastar.js` 1.0.2 → match the Go SDK 1.2.2.*
2. **gomponents over templ and over keeping `html/template`.** gomponents is
   pure Go (no codegen step — preserves the tool-free, CGO-free single-binary
   build and the automated merge flow), and `gomponents-datastar` **types the
   Datastar attribute layer** (`data.On`/`data.Bind`/`data.Signals` + typed
   modifiers) — the actual bug surface, which templ leaves opaque. templ's only
   edge (HTML-shaped markup) is a solo-dev taste preference that doesn't pay for
   the first codegen dependency. *Caveat that survives: gomponents-datastar
   types the attribute **structure**, not the JS **expression strings** inside —
   keep `data.On` expressions trivial (signal flips / `@post(...)`); real logic
   lives in Go.*
3. **Feature packages.** Split the flat `package web` so each feature owns its
   logic + UI + write routes; this is the real parallel-dev win and is the same
   Go refactor regardless of renderer.
4. **Static card-composed pages; drop drag/free-form.** Cards are reusable
   components a developer composes into pages. No user drag/move/resize/board
   editor. "Like xtiles" = the visual card aesthetic, not an editable canvas.
5. **No Genkit; keep the existing catalog; `blocks` primitive-tree is YAGNI.**
   Genkit has no generative-UI feature and its constrained-output capability is
   already hand-rolled via the tool-arg schema + `cards.Validate`; adopting it
   violates the single-binary ethos. A model-composed primitive tree is correctly
   designed *if ever needed* but is gated on real transcripts showing ≥2 recurring
   layouts no card/board expresses. The higher-value increment is **more/richer
   typed cards + sharper tool descriptions** so the small local model picks well.
6. **No model-authored HTML, ever.** Not SaaS ceremony: untrusted content (tool
   results, memories, future web/email) flows into the model context, so
   indirect prompt-injection-to-UI is a live path even single-user (OWASP LLM05).
   The secure path is also the simpler/already-built one (auto-escaping by
   default). Never pass a model-derived string to `g.Raw`/`template.HTML`.
7. **Reactive agent-action updates (Part B): whole current-page morph.** After
   an agent mutation, re-render the page the user is on and morph its card
   container in one shot (§4.6). Chosen over a per-type refresh map because, in
   the static-page model, a page is a deterministic function of its id — no map
   to maintain, no param-scoped-card gap, and it *closes* the §4.6 problem for
   agent actions instead of reopening it partially.

## 4. Architecture

### 4.1 Package layering (the import-direction law)

Preserve the existing "`internal/cards` imports no `internal/web`" rule and
extend it into a one-way stack:

```
Layer 0  internal/cards        registry + Validate (unchanged; stdlib + core only)
Layer 0  internal/ui           shared gomponents primitives + helpers + the
                               Datastar-attr re-exports (clipText, sparkline,
                               error strip, shared chrome) — imports gomponents
                               + gomponents-datastar + core only; NEVER web
Layer 1  internal/{tasks,knowledge,life,heads,recap}   domain libs (unchanged)
Layer 2  internal/feature/<name>   implements its cards/pages/routes; imports
                               internal/ui + internal/cards + its domain +
                               pocketbase; NEVER imports internal/web
Layer 3  internal/tools        imports internal/cards (unchanged direction)
Layer 4  internal/web          thin host: builds deps, mounts each feature's
                               routes, owns the dock + chat stream + the
                               reactive refresh seam, guards/headers
```

**Law:** `web → feature → {ui, cards, domain} → core`, never reversed. A
compile-time assertion (mirroring `internal/cards/cards_test.go`) guards that no
feature/ui package imports `internal/web`.

### 4.2 gomponents replaces html/template

- Components are exported Go funcs returning `g.Node` (e.g.
  `tasks.TodayCard(v TodayView) g.Node`), composing by direct typed calls.
- Datastar attributes use `gomponents-datastar`: `data.On("submit", "@post('/ui/tasks/'+...)" , data.ModifierPrevent)`, `data.Bind`, `data.Signals`, `data.Text`, `data.Attr`, `data.Class`. The **wiring** is compile-checked Go.
- **SSE integration is a one-line swap** — the stream layer is unchanged:
  `var b strings.Builder; node.Render(&b); sse.PatchElements(b.String(), …)`
  replaces `h.tmpl.ExecuteTemplate(&b, name, data)` at every call site.
- **Escaping firewall:** `g.Text` auto-escapes. `g.Raw`/`g.Rawf` are **banned on
  any model-derived or user-derived string** (lint/grep gate in CI). One leak
  voids the firewall (§4.5/§4.7).
- Inbound: where a handler reads posted Datastar signals, use
  `datastar.ReadSignals(r, &typedStruct)` (typed decode).

### 4.3 Feature packages (the split)

Each feature folder owns: `data.go` (view-model builders over its domain),
`components.go` (gomponents card + focus components), `routes.go`
(`func Mount(r, deps)` registering its write endpoints), `*_test.go`. Wiring is
**explicit** in `web` setup (`tasks.Mount(r, deps)`, …) — no `init()` magic, no
registry framework. The dynamic type→component lookup needed by the inline-card
path and pages is one small map built at startup from each feature
(`map[string]CardFunc`, app/deps captured in closures) — it replaces the
`cardInto`/`focusBodyHTML`/`cardHTML` switches.

### 4.4 Static card-composed pages (retire the board editor)

- A **page** is a Go function composing card components in a fixed layout,
  served at a route. The default dashboard, plus feature focus pages, are such
  functions.
- **Delete:** `web/static/board.js` (drag/resize), the layout persistence
  (`x/y/w/h` on the `boards` collection cards JSON), and the editor endpoints
  `POST /ui/boards/{id}/{layout,cards/add,cards/{idx}/remove}` and the
  drag/resize grid. Existing default-board seeding becomes static page
  definitions.
- The agent composing a *view* of cards inline in chat (`card_show`,
  `board_compose`) stays — that is a server fragment, unrelated to user drag.
- Each page renders its cards into a container with a **stable id** (e.g.
  `id="page-<name>"` inside `#main`) so the reactive morph (§4.6) can target it.

### 4.5 Generative UI — keep the catalog, no Genkit, no raw HTML

- **Part A is done.** The agent renders a card inline in the conversation via
  the existing `card_show` → `MarkUICard`/`ParseUICard` → `cardHTML` →
  append-to-`#chat` path. No new code; only confirm `card_show` is exposed on the
  master chat tool set and nudge the prompt to use it when helpful.
- **Do not build** `internal/blocks`/`ui_compose` (model-composed primitive
  tree) now — YAGNI, gated on real transcripts (§3.5). When/if built: closed
  `Kind` allowlist, **recursive total** validator with hard depth (~6) and
  node-count (~200) caps that **reject, not truncate**, `button.action` bound to
  a **closed enum of owner-scoped server intents** (never a model URL), `g.Text`
  only, validate-complete-then-patch-once, persist only the cleaned tree.
- **Security model:** the catalog/registry is the boundary — the model names a
  registered type and supplies data, never markup/classes/styles/links/scripts;
  the validator rejects unknowns; the server owns all data access; the spec is a
  dumb, fully-resolved tree. Strict CSP as belt-and-suspenders.

### 4.6 Reactive agent-action UI updates (Part B)

**Mechanism (whole current-page morph):**

1. **Refresh signal from mutating tools.** Mutating agent tools
   (`task_done`/`snooze`/`drop`, and other state-mutators) wrap their existing
   plain-text result in a new NUL-prefixed marker `RefreshMarker` (mirroring
   `UICardMarker`/`ProposalMarker` in `internal/tools/ui.go:26`). The marker
   means "a mutation committed; the current page should refresh." The model still
   reads the plain text after the marker (`ParseRefresh` returns the rest).
   Mutations already commit before the result string is built (`tasks.Done`
   before return), so a re-render sees new state.
2. **Current page is carried as one signal.** The shell binds a `$page` signal
   (current page id); the chat `@post` sends the signal store, read server-side
   via `datastar.ReadSignals(r, &struct{ Page string })` at turn start.
3. **Dispatch seam.** In `chatstream.go` `handleToolResult`, before the plain
   fallthrough: if `ParseRefresh(ev.Text)` is ok, finish the tool row with its
   plain text **and** re-render the current page's card container (the `$page`
   page function) and morph it with a selector-less `PatchElements` by the
   container id. Off-page / focus-view → the container id is absent → silent
   no-op (stateless "patch-if-present"). The user gets both the chat confirmation
   and the live card refresh in the same stream.
4. **Recap/CLI marker hygiene (required).** The new marker MUST be stripped in
   `internal/web/recap.go` (history reload renders the plain text, no patch) and
   `internal/cli/chat.go`, or the NUL marker leaks into reloaded history.

**Why whole-page, not a per-type map:** a page knows each card's params, so
re-rendering it refreshes *every* on-screen card (including param-scoped
calendar/measure/day) correctly, with no hand-maintained `entity→card-types` map
to rot and no reliance on unique `ucard-{type}` ids. It closes §4.6 for agent
actions rather than reopening it 20%.

**Caveat to verify in the plan:** a full-container morph could disturb a card's
*unsaved* client state (e.g. a half-typed inline edit in a "manage" card). The
static-page simplification already removed drag state; the plan must confirm no
on-screen card holds un-serialized transient state a morph would stomp. If one
does, scope the morph to skip cards with open editors, or fall back to a
per-type morph of the non-parameterized tiles (today/quests) for v1.

**Scope (bounded):** reopened **only for agent tool actions**, which already own
a live SSE stream for the whole turn — no new endpoint, socket, goroutine, or
client JS. User-action cross-card propagation remains out of scope (existing
self-targeting fragments still handle the acting card).

### 4.7 Security (unchanged + reaffirmed)

Server-authoritative, owner-scoped at the PocketBase rule boundary,
localhost-guarded, no browser→PocketBase REST. The **no-model-HTML firewall**
(§4.5/§3.6) is absolute: `g.Text` only on any model/user-derived string; `g.Raw`
forbidden there; agent actionable elements bind to closed server intents, never
model-supplied URLs.

## 5. Feature decomposition

Seven packages (cards grouped by shared domain), mirroring the page→cards map:

| Package | Cards | Notes |
|---|---|---|
| `feature/tasks` | today, quests, calendar, timeline, habits | richest writes; PoC |
| `feature/journal` | journal, day | shares the life store; LLM prompt helper |
| `feature/knowledge` | memory, skills | one `{kind}` pipeline |
| `feature/life` | measure, lines, lifelog | read-mostly + entry create |
| `feature/heads` | heads (personas) | persona switcher + manage card (heads-as-personas is already merged) |
| `feature/settings` | settings (folds profile + models) | largest write surface (GGUF SSE) |
| `internal/web` (host) | — | pages, dock + chat stream, the reactive refresh seam, focus dispatch — stays in `web`, not a feature package |

**Proof-of-concept: `tasks`** — richest write surface, cleanly self-contained
(`tasks`+`store`, no turn/gguf/conversation), exercises five cards through one
package, and is the canonical Part B example (`task_done` → today/quests refresh).

## 6. Rollout — strangler-fig, app green at every step

gomponents and `html/template` coexist (both render to a string into the same
`PatchElements`), so migrate per feature:

- **Phase 0 — Foundations.** Add `gomponents` + `gomponents-datastar` to
  `go.mod`; create `internal/ui` (shared primitives, helpers, error strip,
  Datastar-attr re-exports); add the type→`CardFunc` map with a legacy fallback
  (unmigrated types hit the old `cardInto`); bump vendored `datastar.js` to
  1.2.2. Bind the `$page` signal and add the `ReadSignals` page-context read
  (no behavior change yet). Zero deletions.
- **Phase 1 — `tasks` (PoC).** Port its 5 cards + focus to gomponents in
  `internal/feature/tasks`; move its helpers + write routes; register its
  `CardFunc`s; remove its cases from the legacy switch. **Land Part B here:**
  `RefreshMarker` + the `handleToolResult` seam + whole-page morph + recap/CLI
  strip; `task_done`/`snooze`/`drop` opt in. Verify (Go tests + browser:
  complete a task in chat → today/quests tile updates live).
- **Phases 2–6 — journal → knowledge → life → heads → settings.** Port each
  feature to gomponents + its package; shrink the legacy switch.
- **Phase 7 — Static pages + cleanup.** Replace the board editor with static
  page compositions; delete `board.js`, layout persistence, editor endpoints;
  delete the legacy `cardInto`/`focusBodyHTML` switches and the `html/template`
  set; update `DESIGN.md`.

## 7. Out of scope

- The `internal/blocks` model-composed primitive tree / `ui_compose` (YAGNI,
  gated — §4.5).
- Adopting Genkit / A2UI / AG-UI / Thesys C1 as dependencies.
- User-editable boards / drag / resize / free-form layout (deleted).
- User-action cross-card propagation (only agent actions reopen §4.6).
- Multi-instance/param-scoped reactive refresh beyond the whole-page morph
  (handled by the page morph; per-type is only a fallback).
- Multi-user / per-user scoping.

## 8. Testing approach

- **Per feature:** Go component/render tests (gomponents nodes render to expected
  HTML; replaces the `html/template` parse-guard test) + write-endpoint tests,
  moving with each feature package.
- **Renderer parity:** during migration, assert the gomponents output matches the
  retiring template for each card (snapshot/contains) before deleting the old
  template.
- **Part B:** a turn test that runs `task_done` and asserts the stream contains
  (a) the tool-row text and (b) a `PatchElements` morph of the current page
  container; a browser check (complete a task in chat → tile updates; on a
  different page → no-op).
- **Firewall:** a test that a model-derived string is auto-escaped and that
  `g.Raw` is absent from model-data render paths (grep/lint gate).
- **Deletion safety:** after each phase, grep for the dead switch case / template
  / endpoint and confirm zero references.

## 9. Open items / risks

- **Transient client-state on whole-page morph** (§4.6 caveat) — verify in the
  plan; fall back to per-type morph if a card holds unsaved state.
- **Mixed-renderer drift during migration** — keep model/user-derived values on
  the auto-escaping path in *both* renderers until `html/template` is gone.
- **Local model reliability** — small local Gemma is weak at strict tool/JSON
  conformance; keep the existing `Validate`-then-plain-text-self-correct loop and
  bias the prompt toward `card_show` (catalog) over any future free composition.
- Detailed per-phase task breakdown to be produced by the implementation-planning
  step as numbered `plans/NNN-*.md` files in repo style.
