# Card-first: kill the pages — design spec

**Date:** 2026-06-13
**Status:** Approved (brainstorm) — ready for implementation planning
**Predecessors:** `plans/028-typed-card-registry.md`, `plans/029-boards.md`, `plans/030-agent-ui-tools.md`, the HTMX→Datastar migration.

## 1. Goal

Collapse Balaur's UI to **one world: boards + cards + dock-chat.** Today the app
has two parallel surfaces — a card/board world *and* a set of legacy full-page
routes (`/tasks`, `/memory`, `/journal`, `/day`, `/life`, `/heads`,
`/heads/{id}/chat`, `/models`, `/settings`, `/profile`, recap). This program
promotes each feature into a card complete enough that its page can be deleted,
then deletes the page. End state: there are no feature pages — only boards of
cards, and a persistent dock chat.

**Success criterion:** every feature currently reachable as a page is reachable
as a card (on a board, via the launcher, or via chat), with full write parity;
the legacy page routes and their templates are gone; navigation is board tabs +
dock chat only.

## 2. What already exists (do not rebuild)

This is a *finishing* program, not a rewrite. Already shipped:

- **Typed card registry** (`internal/cards/cards.go`): 11 card types
  (today, quests, calendar, timeline, journal, measure, lines, memory, skills,
  heads, habits), each an addressable server resource at
  `GET /ui/cards/{type}?params`, server-rendered from PocketBase. Zero `internal/web`
  imports so agent tools can validate without a cycle.
- **Boards** (`internal/web/boards.go`): owner-composed dashboards persisted in
  the `boards` collection as a JSON `cards` array with per-card `x/y/w/h`.
  Create/rename/delete boards, add cards from a palette, drag/resize on a 12-col
  grid — all saved (`POST /ui/boards/{id}/layout`).
- **Persistent dock chat**: switching boards patches only `#main`
  (`isDatastarRequest` → `sse.PatchElements(WithSelectorID("main"))` →
  `ReplaceURL`); the dock and its live SSE chat are never touched, so the
  conversation survives navigation.
- **Agent composition tools** (`internal/tools/ui.go`): `card_show`,
  `board_compose`, `board_add_card` — chat can drop a card inline or build a
  whole dashboard.
- **Interactive `manage` mode** on quests/memory/skills/heads cards: self-targeting
  fragments (`#tcard-{id}`, `#head-{id}`, …) that transition state in place via
  existing POST endpoints.
- **Single-user, local-first**: owner-scoped at the PocketBase rule boundary
  (`@request.auth.collectionName = 'users'`), localhost-guarded (`guardLocalUI`).
  Boards already *are* the saved-layout store; no schema work needed.

## 3. Decisions (from brainstorm)

1. **Goal:** kill the pages — unify on boards + cards + chat.
2. **Full surface = card expand/focus.** A card is one component that renders at
   two sizes; "page" is just a card at full size. (Not feature-boards, not
   bigger-tiles.)
3. **Rollout = full program up front**, executed phase by phase.
4. **Head branch chat = dock swaps context.** One dock chat, swappable thread
   (master ↔ a head's branch); not a card, not a kept page.

## 4. Architecture

### 4.1 A card becomes a two-size component

Every card type renders at two sizes:

- **tile** — the board version: compact, summary + a few quick actions
  (today's existing summary/manage tiles).
- **focus** — the full-canvas version: everything the old page did, with all
  CRUD inline.

The existing `mode=manage` templates are the seed for **focus** — they already
render self-targeting, interactive rows. Focus is that, grown to full canvas and
given the create/edit forms the page had. The registry `Spec` gains a notion of
"this card has a focus view" and the renderer dispatch gains a focus rendering
per type (mirroring the current `cardInto` switch).

### 4.2 The expand mechanism (reuses the board-switch pattern)

- A card tile carries an expand control (⤢).
- Clicking it fires a Datastar `@get` to an **addressable focus route**:
  `GET /focus/{type}?params` (and a `from` board id for "back").
- The handler is dual-mode via `isDatastarRequest(e)`, exactly like `boardsPage`:
  - **Datastar `@get`:** patch only `#main` with the card in focus mode + a
    "← back to {board}" header; `ReplaceURL` to `/focus/{type}?…` so
    refresh/back/bookmark work; sync `document.title`.
  - **Full browser load:** render the whole shell (topbar + `#main` focus card +
    dock).
- The dock is **never** patched — chat persists across expand/collapse.
- "← back" is a `@get` to the originating board (`/boards/{from}`).

This is byte-for-byte the existing board-switch flow, so no new navigation
primitive is introduced.

### 4.3 Write actions live inside the card (HATEOAS)

Each focus card carries its own controls — `<form>`s with
`data-on:submit__prevent="@post(...)"` that patch the card's own fragment by id.
The card's rendered HTML *is* its API surface.

Most POST endpoints already exist and are simply re-mounted in focus:
`/ui/tasks/{id}/transition`, `/ui/journal`, `/ui/day/{date}/journal`,
`/ui/knowledge/{kind}/{id}/edit`, `/ui/knowledge/{kind}/{id}/transition`,
profile saves, model ops. The net-new writes are a small set of **create** forms:
new task, new memory, new life entry.

### 4.4 The dock swaps conversations (heads)

The dock gains a conversation selector:

- Default active conversation = master (current behavior; nothing changes until
  a head is picked).
- The `heads` card's "open" action sets the dock's active conversation to that
  head's branch: patch `#chat` with the branch history + a "← back to main"
  header.
- `/ui/chat` routes the turn to the active conversation (master or branch).
- Branch conversations already exist in the data model (master + branch
  `conversations`, `conversation.History()`); this adds a **selector**, not a
  conversation system.

### 4.5 The launcher keeps every feature reachable

When pages die, a feature not currently on any board must still be openable. The
existing `/ui/cards` palette (lists every spec) becomes the **launcher**: click a
feature → it opens in focus (`@get('/focus/{type}')`). Chat can also always
`card_show` / `board_compose` it. No feature becomes unreachable.

### 4.6 Live-update policy (explicit)

A card refreshes **itself** after its own action (the existing self-targeting
fragment pattern). **Out of scope for v1:** cross-card propagation (e.g.,
completing a task in `quests` auto-refreshing `today`/`calendar` on the same
board). That is a later nicety via a board-level refresh signal, and is called
out here so the boundary is not a silent omission.

### 4.7 Security (unchanged)

Everything stays server-rendered, owner-scoped at the PocketBase rule boundary,
and localhost-guarded. **No browser→PocketBase REST.** A card is a server
resource. The head/grant model is enforced server-side; client-direct data
access would bypass it and is explicitly rejected.

## 5. Feature map (page → card)

"✓" = already built; "＋" = net-new.

| Today's page | Becomes | Focus (full surface) | Net-new work |
|---|---|---|---|
| `/tasks` | `today`/`quests`/`calendar`/`timeline` ✓ | full task manager | ＋create, ＋edit (transition ✓) |
| `/journal` + `/day/{date}` | `journal` ✓ | write + prompt + history + day navigation | write ✓; ＋history paging |
| `/memory` | `memory` ✓ (manage) | search + approve/edit/archive + add | edit/transition ✓; ＋create |
| `/skills` | `skills` ✓ (manage) | full skills manage | ✓ |
| `/life` | `measure`/`lines` ✓ + ＋`lifelog` | browse kinds + log entries | ＋entry create |
| `/heads` | `heads` ✓ (manage) | manage heads + open branch in dock | ✓ + dock-swap |
| `/heads/{id}/chat` | **dock context-swap** (not a card) | dock shows branch | ＋dock conversation routing |
| `/models` | ＋`models` | picker + GGUF download/progress + providers | ops ✓ (SSE) |
| `/settings` + `/profile` | ＋`settings` (profile folds in) | sections + name/avatars | ✓ |
| recap (`/ui/recap/*`) | `recap` ✓ | telescoping recap | ✓ |

**Net-new cards: three** — `lifelog`, `models`, `settings`. Everything else is
promoting an existing card to focus + mounting controls.

## 6. Phase sequence

Each phase follows the same loop: **card-complete the focus view → verify (Go
tests + browser) → delete the page/routes/templates → grep-confirm nothing
references the dead route.**

- **Phase 0 — Foundations** (shared plumbing, zero pages deleted):
  the `focus` rendering contract + `/focus/{type}` addressable dual-mode route +
  expand/collapse/back chrome; the launcher (palette → focus); the dock
  conversation-selector plumbing (defaults to master — no behavior change yet).
  *This front-loads the only two genuinely-new mechanisms so phases 1–6 never
  touch plumbing.*
- **Phase 1 — Tasks** (beachhead; proves write parity): inline create/edit/
  transition in focus; delete `/tasks` + `tasks.html`.
- **Phase 2 — Journal + Day**: focus journal with write, prompt, history, day
  navigation; delete `/journal`, `/day`, templates.
- **Phase 3 — Knowledge** (memory + skills): focus manage + create; delete
  `/memory`, `/skills`.
- **Phase 4 — Heads + branch chat**: heads focus + dock context-swap; delete
  `/heads`, `/heads/{id}/chat`.
- **Phase 5 — Life**: `lifelog` focus + entry create; delete `/life`.
- **Phase 6 — Settings + Profile + Models**: `settings` and `models` cards;
  delete `/settings`, `/profile`, `/models`.
- **Phase 7 — Cleanup**: strip old topbar nav links, dead handlers/templates,
  finalize launcher, update `DESIGN.md`.

Ordering rationale: Phase 0 isolates new plumbing; Tasks first because it's the
richest write surface (if create/edit/transition-in-focus feels right there,
lighter features are downhill); Heads at Phase 4 so it lands after the focus
pattern is proven three times and is the one that also exercises the dock
selector.

## 7. Out of scope (v1)

- Cross-card live propagation (§4.6).
- Multi-user / per-user scoping (Balaur is single-user by design).
- Browser-side PocketBase REST access (rejected; breaks the grant model).
- New conversation/branch-merge mechanics (we add a dock selector only).

## 8. Testing approach

- Per card promoted to focus: Go handler/template tests (the `*_test.go` pattern
  already in `internal/web/`) for the focus rendering + each write endpoint.
- Browser verification (Playwright/Chrome DevTools) per phase: expand → act →
  card refreshes → collapse → dock intact.
- Deletion safety: after removing a page, grep the tree for the dead route/handler/
  template name and confirm zero references before the phase is "done".

## 9. Open items

None blocking. Detailed per-phase task breakdown to be produced by the
implementation-planning step as numbered `plans/NNN-*.md` files in repo style.
