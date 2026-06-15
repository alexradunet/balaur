# Gomponents Atomic Design System + Storybook-as-Product-Surface

**Status:** Design approved (pending written-spec review) · **Date:** 2026-06-15

## Context

Balaur's web UI is mid-migration from server-rendered `html/template` to
gomponents. The card *tiles* already render via per-feature gomponents
(`internal/feature/*cards`), but the page shell, chat, focus bodies, and ~40
`ExecuteTemplate` callers remain on `html/template` under the top-level `web/`
package.

Separately, a complete design system — **"Hearthwood"**, pixel-art 16-bit
Romanian-fairy-tale RPG — was authored in **Claude Design** and exported to
`Balaur_Design/`. The export is plain-JS React (`React.createElement`, no JSX)
over **vanilla CSS**: 16 canonical components in `_ds/.../_ds_bundle.js`, more
in batch files (`Primitives`, `Utilities`, `DataDisplay`, `Domain`, `Screens`,
`Composer`, `ChatExplore`), a machine-readable `_ds_manifest.json`, and tokens
via `light-dark()` custom properties.

This spec finishes the gomponents migration *and* adopts the export as a typed
Go atomic design system, displayed as a **storybook page that becomes the
product surface at `/`**.

### Key reframe: there is no CSS re-skin

`web/static/basm.css` at HEAD is **already** the export's Hearthwood — tokens,
typography, materials (`.parch`/`.wood`/`.ornate`), components, and pages are
effectively byte-identical to the export's `tokens/*.css` + `basm/*.css`, and
all 6 fonts, 13 icons, 36 avatars, crest, and logo already live in
`web/static/`. So "embed Hearthwood, replace the old CSS" is a non-task. The
CSS work shrinks to: fix inherited bugs, and *add* rules for atoms the export
ships inline-only.

## Goal

Rebuild the web UI as a typed gomponents atomic design system ported faithfully
from the Claude Design export, with the storybook gallery as the product
surface at `/`, the functional screens at `/focus/{type}`, and the legacy
`html/template` machinery + top-level `web/` package removed.

## Non-goals

- No Node build step, no React at runtime, no MCP, no global mutable state
  (AGENTS.md invariants stand).
- No multi-human/multi-user work.
- Not porting the export's demo choreography (the `setTimeout` scripts and
  in-world parsers `matchChoice`/`propose`/`logEntry`/`capture`): we port the
  *rendering*, not the scripted demo behavior. Real data + transitions come
  from PocketBase collections and `internal/turn`.

## Locked decisions

1. **Cut boards entirely.** Remove the owner-composed dashboard: `/boards`
   routes/handlers, `web/templates/boards.html`, `web/static/board.js`, the
   `board_compose` + `board_add_card` agent tools, `cards.ValidateCards`, and
   the boards collection (via a drop migration). The card *registry/dispatch*
   survives (see Architecture).
2. **Split composer** (KISS): `MessageDraft` + `DialogueChoices` + `ChatBar` as
   separate components, not the export's single morphing ledge.
3. **Relocate + delete top-level `web/`.** Move `web/static/` →
   `internal/web/assets/` with its own `embed.FS`; delete `web/templates/` and
   the `web` package. Assets keep their `/static/...` URLs (no other change).
4. **Full component library.** Port every export component, including those no
   screen uses (GuardianCard, Skeleton, Breadcrumb, Pagination, FolkBand) — the
   storybook is the consumer, so a complete catalog is the deliverable. This
   deliberately overrides YAGNI for this one surface.

### Defaults proceeding without further questions

- **Drop TypewriterText** as server logic for v1 (render final text; an
  optional reveal effect can live in `basm.js` later).
- **Accept the modern-browser floor** for `light-dark()`/`color-mix()` (2024+
  evergreen, no `@supports` fallback) — acceptable for a localhost-first
  personal app.
- **Datastar action contract is form-submit**:
  `data.On("submit", url, data.ModifierPrevent)` on a `<form>`, mirroring the
  verified `taskcards.transitionPost` pattern — **not** a React-style onClick
  carrying an `@post(...)` string.
- **Orphaned boards data**: add a drop migration (a personal pre-1.0 app; the
  owner is the developer). No export path.

## Target architecture

### Package layout (extend, don't rebuild)

`internal/ui` is a verified no-web-import leaf documented as the
atomic-primitive home; `internal/feature/*cards` already follow
`buildXxx → View → Component(g.Node)`. We extend these rather than invent a
parallel system.

| Package | Layer | Contents |
|---|---|---|
| `internal/ui` *(extend)* | **atoms + shared helpers** | One file per group, each <500 lines: `button.go`, `tag.go`, `pip.go`, `badge.go`, `toggle.go`, `avatar.go`, `icon.go`, `card.go` (Card/Stitch/FolkBand), `field.go` (TextField/Select), `tabs.go`, `toast.go`, `alert.go`, `tooltip.go`, `skeleton.go`, `nav.go` (Breadcrumb/Pagination), `list.go`, `empty.go`, `dialog.go`. Keep existing `registry.go`, `components.go`, `text.go`, `spark.go`. Imports only gomponents + `pocketbase/core`. |
| `internal/ui/chat` *(new)* | **chat organisms** | `message.go` (Message/frameMessage + shared portrait helper), `toolrow.go`, `dialogue.go` (DialogueChoices), `composer.go` (MessageDraft + ChatBar), `conversation.go` (the `#chat` scroll container that holds the message list — the SSE patch target). Imports `internal/ui` only. |
| `internal/ui/shell` *(new)* | **template (page shell)** | `shell.go`: `Page(PageProps)` — the one place that emits `<html>`, ports `layout.html` (page_head + no-flash theme script + Topbar + `#main` + `#dock`). `Topbar(TopbarProps)`, `Canvas(maxWidth, …)`. Imports `internal/ui` + `internal/ui/chat`. |
| `internal/feature/*cards` *(extend)* | **domain organisms/pages** | Existing packages compose the new atoms instead of hand-rolled class strings + duplicated pip loops. New `recapcards` (RecapCard, NudgeBanner) replaces the recap/nudge templates. Profile + avatar picker **relocated down** from the `web` gateway (see below). |
| `internal/feature/storybook` *(new)* | **fixtures + gallery** | `fixtures.go` (pure `View` structs, never PocketBase) + the section gallery builder. Rendered by a web handler, **not** registered as a card. |
| `internal/web/assets` *(new)* | **embedded assets** | `embed.go` with `//go:embed static`; holds the relocated `static/` (basm.css, fonts, icons, avatars, crest, logo). Served via the existing `apis.Static` at `/static/...`. |
| `internal/web` *(thin out)* | **gateway** | `GET /` → storybook handler composing `shell.Page`; delete boards; delete the `tmpl`/`funcs`/`render` machinery only after every `ExecuteTemplate` caller is ported. |

The layering law holds: `internal/ui*` and `internal/feature/*` never import
`internal/web`.

### The card registry survives (boards cut, dispatch kept)

`internal/cards` (the `Spec` registry: `Get`, `Validate`, `All`),
`internal/ui/registry.go` (`RegisterCard`/`LookupCard`), and
`internal/web/cards.go` (`cardInto`/`cardHTML`/`uiCard`/`uiCardPalette`) are
**not** board-only. They remain the server-rendered card dispatch used by:

- `/focus/{type}` — the functional screens (the primary navigable surface
  after boards),
- `card_show` agent tool — the model embeds live cards inline in chat,
- SSE card re-patch after task/knowledge transitions.

Only `cards.ValidateCards` (board-composer-only) is deleted. `cards.Spec`'s
grid `W`/`H` fields become vestigial but harmless; leave them until a later
cleanup unless trivially removable.

### Two surfaces

- **`/` — the storybook.** A plain `web` handler renders `shell.Page` with the
  gallery as its body and the live chat dock. Because it goes through the real
  `shell.Page` + `basm.css`, it can never drift from production. It is **not** a
  `cards.Spec` tile (a full page has no grid span/params and must not pollute
  the registry exposed to `card_show`).
- **`/focus/{type}` — functional screens.** Unchanged dispatch; bodies ported
  from their focus templates to gomponents during the migration.

## Component translation (summary)

The full per-component mapping (55 components catalogued; Go symbol, package,
CSS handling, Datastar/data needs, effort, reconciled existing renderer) is
enumerated in the implementation plan. Categories and notable points:

- **Atoms** (`internal/ui`): Button, Card, Tag, Stitch, FolkBand, Pips, Avatar,
  Icon, Badge, Toggle, Skeleton, Sparkline (extend existing). Most are
  class-only against existing `basm.css`. `ui.Pips` collapses the byte-identical
  `memoryPips`/`recordPips` duplication (the Phase-1 proof point).
- **Molecules** (`internal/ui`): TextField, Select, Tooltip (CSS-only hover),
  Toast, Breadcrumb, Pagination, ListItem, **Tabs** (load-bearing — Tasks/Memory
  tab filters; needs new `.tabs`/`.tab` CSS), SectionLabel, ScreenTitle.
- **Organisms** (`internal/ui` / `internal/feature`): Alert, Dialog (prefer
  native `<dialog>`), EmptyState, List, StatCard (lifecards), RecapCard +
  NudgeBanner (new `recapcards`), GuardianCard (new, deferred surface but built
  for the catalog), KnowledgeCard/TaskCard (reconcile existing).
- **Chat organisms** (`internal/ui/chat`): Message/frameMessage, ToolRow,
  DialogueChoices, MessageDraft, ChatBar, Conversation container.
- **Shell/templates** (`internal/ui/shell`): Topbar, Page, Canvas.
- **Screens** (`web` handlers + feature focus bodies): ChatScreen,
  TasksScreen, MemoryScreen, LifeScreen, ProfileScreen — ported as `/focus/*`
  bodies + the chat dock, composing the organisms above with real data.

**Explicitly not ported as server logic** (marked export-only): TypewriterText
(client sugar), and the demo parsers `matchChoice`/`propose`/`logEntry`/
`capture` + `FULL`/`DEFAULT_TOOLS` constants (storybook fixtures / demo
choreography). The DialogueChoices **server** dispatch reuses the existing chat
choice handler path (`chatstream.go`), not a re-implemented parser.

### Styling rule

Every atom emits a **class**; repeated inline-style objects from the export are
lifted into `basm.css` rules that consume existing tokens. Inline `style` from
Go is reserved for genuine per-instance custom properties only (`--avatar-size`,
`--portrait-size`/`--chat-gutter` on the chat wrapper, the profile grid). The
export's hardcoded hex (`#1c0d04`, `#06120f`) maps to existing tokens
(`--ember`/`--good`/`--teal`), not preserved as literals. `basm.css` stays the
single styling source of truth all class names reference.

## CSS & assets strategy

1. **Fixes (Phase 0).** Define `--indigo-deep` (referenced at ~`basm.css:515`
   and `:695` for the owner-portrait keyline, never defined); map the four
   undefined Section-7 fallback tokens (`--line`→`--parch-edge`/`--outline-2`,
   `--accent`→`--gold`, `--border`→`--parch-edge`, `--parchment`→`--surface-2`).
   Optionally drop the unused Work Sans `@font-face` and the legacy
   `--shadow-hard` alias.
2. **New rules** for inline-only atoms with no class today: `.toggle`/
   `.toggle-knob`, `.badge`/`.badge-<tone>`, `.toast`/`.toast-<tone>`,
   `.alert`/`.alert-<tone>`, `.tooltip` (CSS hover), `.skeleton` + `@keyframes
   sk-sheen`, `.list`/`.list-item`, `.tabs`/`.tab`, `.section-label`,
   `.screen-title`, `.breadcrumb`, `.pager`. Tokenized, no raw hex.
3. **Relocate + serve.** Move `web/static/` → `internal/web/assets/static/`;
   `internal/web/assets/embed.go` does `//go:embed static`; keep the existing
   `apis.Static("/static/{path...}")` so URLs are unchanged. `basm.css` is one
   concatenated file with `@font-face` pointing at `/static/fonts/*` — no
   `@import` chain, no url() rewrite. Delete the top-level `web` package once
   `templates/` is gone.
4. **Theming.** Single-source `light-dark()` + `color-scheme`; force a mode via
   `html.light`/`.dark`. Port the no-flash inline script into `shell.Page`'s
   `<head>` verbatim; keep `basm.js basmToggleTheme()` + `localStorage`. The
   toggle stays a tiny client behavior (no Datastar round-trip).

## Cut boards — precise plan

**Delete:** `internal/web/boards.go` (~691 lines) + `boards_test.go` (~588);
`web/templates/boards.html`; `web/static/board.js`;
`migrations/1750740000_boards.go` + test (or supersede with a drop migration —
see below); `boardComposeTool` + `boardAddCardTool` in `internal/tools/ui.go`
(~296 lines, keep `cardShowTool`) and their `ui_test.go` cases;
`cards.ValidateCards`; the 8 `/boards*` route registrations in `web.go`.

**Replace:** `GET /` (currently `boardHome` → `/boards`) → the storybook
handler.

**Drop migration:** add a new migration with a strictly-increasing prefix
(after `1750740000`) that drops the `boards` collection. Migration timestamp
prefixes must be unique and increasing (AGENTS.md).

**Update in the same change:** `internal/self/knowledge.md` (remove the Boards
section + `board_compose`/`board_add_card` + default boards + drag/resize;
describe the post-boards model: `/` storybook, `/focus/{type}` screens); tours
`08-hateoas-cards-and-boards` (keep card-registry steps, drop board steps),
`00-orientation`, `07-the-web-gateway` (remove `/boards`, update `/`).

## Interactivity model

**Datastar contract:** form-submit only — `data.On("submit", url,
data.ModifierPrevent)` with a real URL, per `taskcards.transitionPost`. Two
Datastar packages coexist: `gomponents-datastar` (`data.*` node attrs) and
`datastar-go` (SSE patches); follow the existing pattern exactly.

**Storybook interactivity — three honest tiers:**
- **(A) Pure-CSS** (button press bevel, tag rune, gold notch, Tooltip hover,
  `basm-glow`, thinking-dots, theme toggle, Skeleton sheen) — just works; shown
  statically, live in the browser.
- **(B) Static variants** for state machines — render every visual STATE as a
  separate instance (Avatar idle vs thinking, Message normal vs pending, Toggle
  on/off, Dialog open inline). 100% of the look, zero JS.
- **(C) Live Datastar** for genuinely interactive organisms — the chat dock at
  the bottom is the **real** ChatBar posting to `/ui/chat`; optionally one live
  DialogueChoices/action wired to a storybook-only echo endpoint that SSE-patches
  a confirmation. Hotkeys/autosize/typing-sounds are isolated progressive
  enhancement in `basm.js`, documented as "optional client sugar".

**Chat SSE ID stability (hard gate).** `chatstream.go` patches stable IDs
(`#chat`, `#tcard-<id>`, the `chat` selector). Before repointing the handler to
the gomponents renderers, golden-string tests must assert the gomponents
Message/ToolRow/Choices fragments emit **identical** root IDs and class names.
This is a Phase-4 entry criterion, not a mitigation.

## Storybook design

One `web` handler (`storybookHome`) renders `shell.Page{Title:"Balaur",
Active:"storybook", Body: gallery, Dock: live chat}`. The gallery is a sequence
of `<section>` blocks mirroring the manifest buckets: **Atoms, Molecules,
Organisms** (Chat / Knowledge / Tasks / Life / Domain), **Brand** (16 heads, 16
souls, crest, glyphs), **Colors** (token swatches), **Type** (font tokens),
**Materials/Spacing** (`.parch`/`.wood`/`.ornate`/bevels/pips). Each story
renders the real Go component with a caption (Go symbol + prop set) and shows
variants side by side from plain Go fixture literals. **Fixtures never touch
PocketBase** — the storybook renders on an empty DB.

## Phased plan (thin slice first)

Each phase ends with: `go vet ./...`, `go test ./...`,
`CGO_ENABLED=0 go build ./...`, `git diff --check`.

- **Phase 0 — CSS fixes + asset relocation (no behavior change).** First
  **verify the load-bearing assumption**: diff `web/static/basm.css` against the
  export's `tokens/*.css` + `basm/*.css`; if they have drifted, reconcile toward
  the export (this spec assumes they match — if a real re-skin surfaces here, it
  is added Phase-0 scope, not a separate effort). Then fix `--indigo-deep` +
  Section-7 fallback tokens; add the new inline-atom rules + `@keyframes`.
  Relocate `web/static/` → `internal/web/assets/`, update the top-level
  `web/embed.go` to embed only `templates` (until it too is deleted in Phase 6),
  repoint the static route, app renders unchanged.
- **Phase 1 — Shell + one atom, live (thin end-to-end slice).** Port
  `shell.Page` + `Topbar` + `Canvas` (no-flash script + theme toggle wired),
  render a single `ui.Button` at `/storybook`. Proves shell → atom → CSS →
  served page end-to-end.
- **Phase 2 — Full atom + molecule library + first dedupe.** *All* atoms and
  molecules, including the unused-but-catalogued ones (Skeleton, FolkBand,
  Breadcrumb, Pagination) and **Tabs**; refactor `knowledgecards` to `ui.Pips`
  (delete `memoryPips`/`recordPips`) and `taskcards` to `ui.Tag`/`ui.Button`.
  Golden-string tests lock markup for byte-identical refactors; intentional
  markup changes called out explicitly.
- **Phase 3 — Storybook gallery at `/`.** `internal/feature/storybook`
  (fixtures + sections), wire `GET /` → storybook, cut the `/`→`/boards`
  redirect. First user-visible payoff; renders on empty DB.
- **Phase 4 — Chat organisms.** `chat.Message`/`ToolRow`/`DialogueChoices`/
  `MessageDraft`/`ChatBar`/`Conversation`. **Gate:** ID-stability golden tests
  pass, then repoint the chat dock + `chatstream.go`; delete `chat-messages.html`
  + chat partials. Add live + static chat stories.
- **Phase 5 — Domain organisms + screens (all remaining template callers).**
  `recapcards` (replace recap templates + `chatNudges`); StatCard delta +
  week strip; Tasks bucket grouping + tab filter; Memory search + kind tabs;
  **relocate profile + dedupe the two avatar pickers** (gateway `profile.go` +
  `headscards`) into `settingscards` + `ui.AvatarPicker`; port every remaining
  focus body (`day`/`journal`/`knowledge`/`settings`/`quests`/`lifelog`/
  `profile` focuses), `models.html`, `knowledge-grid.html`, `card-*.html`
  fragments. Build the remaining catalog-only organism (GuardianCard; built but
  not wired to a live OS-consent surface). Every component gets a storybook
  entry.
- **Phase 6 — Delete legacy + remove `web/`.** Cut boards (full deletion above)
  + drop migration. Delete `web/templates/` and the `tmpl`/`funcs`/`render`
  machinery once **every** `ExecuteTemplate` caller is ported; delete the
  top-level `web` package. Update `internal/self/knowledge.md` + tours +
  `templates_test.go` in the same commit. Verify the full app (storybook,
  focus screens, chat, settings), `go test ./...`, `CGO_ENABLED=0` build,
  `git diff --check`, `make lint`.

## Testing & verification

- **Golden-string tests** per component (the existing `*_gomponents_test.go`
  pattern) — the primary fidelity oracle, since only two overview screenshots
  exist (`Balaur_Design/screenshots/{bundled,questlog-consolidated}.jpg`); use
  those for eyeball overview only.
- **Chat fragment ID-stability tests** as the Phase-4 gate.
- **Registry invariant test:** every `cards.All()` type has a `ui.LookupCard`
  renderer (prevents "unhandled card type").
- **Storybook-on-empty-DB test:** the gallery renders with no PocketBase
  records (fixtures only).
- Fake `llm.Client`; PocketBase logic via `internal/store` temp-dir apps.

## Risks & mitigations

- **Chat is the deepest SSE port** → ID-stability golden tests as a hard gate;
  port-then-delete in one phase (never two live renderers of one thing).
- **Inline-style lift is lossy** (Badge hex, DayEntry rail, StatCard delta,
  CalendarCell branches) → golden-HTML diffs + eyeball against the two overview
  jpgs; map hex to tokens, don't preserve literals.
- **Inherited CSS bugs copied forward** → Phase 0 fixes them first.
- **~500-line budget** on chat files → the `ui/chat` sub-package split + shared
  private helpers (portrait, `.msg-tool` shell).
- **Template-deletion gap** → Phase 5 explicitly ports *all* remaining
  `ExecuteTemplate` callers (focus bodies, recap, models, settings, profile,
  knowledge-grid, card fragments) so Phase 6 deletes cleanly.
- **Profile/avatar reconciliation** is a gateway→feature relocation + dedupe of
  two pickers, not an in-place extend — sequenced as such in Phase 5.

## Known limitations & deferred work

- TypewriterText reveal deferred (render final text v1).
- Older-browser support for `light-dark()`/`color-mix()` deferred (explicit
  added cost if ever wanted).
- GuardianCard's real OS-consent flow is future work; the component is built
  for the catalog but not wired to a live consent surface.
- `cards.Spec` grid fields (`W`/`H`) become vestigial after boards; full
  removal deferred to avoid widening this change.
- Boards owner data is dropped, not migrated.
