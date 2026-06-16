# UI redesign — focus-body ports (handoff plans)

Self-contained plans to finish the **Domain Navigator** web-UI redesign, written
so a fresh conversation (with zero context from the session that wrote them) can
execute one plan end-to-end. Each plan is independent; pick one, read it, do it.

**Stamped against commit `884d692`** (branch `improve/ui-domain-pages`). If the
tree has drifted far from that, re-read the cited files before trusting excerpts.

> **Why a subdirectory and not `plans/0NN-…`?** The repo's top-level `plans/`
> uses sequential project-plan numbers (…073, 074) and a parallel effort
> (`improve/ollama-to-kronk`, the Ollama→Kronk migration) is actively adding to
> it. Keeping these plans in `plans/ui-domain-pages/` with their own numbering
> avoids colliding on plan numbers or the shared `plans/readme.md` index.

---

## Where the redesign stands

The web UI was migrated from legacy `html/template` pages to a **companion-first
Domain Navigator** (see `internal/self/knowledge.md` "Surfaces"). Done already:

- **`/` is Home** — a full-screen companion chat on `internal/ui/shell.Page`
  (`HTMLClass:"home"`). `internal/web/home.go`.
- **Top-nav topbar** — `shell.Topbar` carries the domain nav
  (Quests/Knowledge/Life/Journal/Heads + Settings); no side rail.
- **Chat uses the storybook components** — `chat.Message`/`chat.ToolRow` panels
  (page-load history *and* the live SSE stream) + a functional `ui.Composer`.
- **All `/focus/{type}` pages render on `shell.Page`** (the new topbar + the
  persistent dock). `internal/web/focus.go` `focusPage` full-load → `shell.Page`;
  the Datastar `@get` branch still patches `#main`.
- **The lifelog focus body is ported to a gomponents component** — this is the
  **exemplar** these plans follow (commit `884d692`):
  `internal/feature/lifecards/lifelogfocus.go`.

**What's left = these plans.** The remaining `/focus/*` *bodies* are still legacy
`html/template`. Each plan ports one body to a gomponents component in its owning
feature package, filling the `CardSize.Focus` seam.

---

## The shared port recipe (read this once; every plan assumes it)

**Read the exemplar first** — open these and mimic them exactly:
`internal/feature/lifecards/lifelogfocus.go` (the ported component + builder),
`internal/feature/lifecards/lifelog.go` (`registerLifelog` size dispatch),
`internal/web/cards.go` (`cardSizeInto` + `cardFocusHTML`), and
`internal/web/focus.go` (`focusBodyHTML` + the `cardFocusHTML` fallback).

The seam (added in `884d692`, do NOT re-add it):

1. `internal/web/focus.go` `focusBodyHTML` already routes any type **not** in its
   bespoke `switch` through `h.cardFocusHTML(typ, params)`, which calls the
   feature renderer with `ui.Focus`. To port a body you **delete that type's
   `case` arm** so it falls through to the seam.
2. `internal/web/cards.go` already has `cardFocusHTML` → `cardSizeInto(…, ui.Focus)`.
   `cardInto` (Tile) is unchanged — **do not touch the four Tile callers**
   (board grid, chat embeds, `/ui/cards` endpoint, `cardHTML`).
3. In the feature package, the card's `register…` fn dispatches on size:
   ```go
   ui.RegisterCard("<type>", func(size ui.CardSize, params map[string]string) (g.Node, error) {
       if size == ui.Focus {
           return <Type>Focus(build<Type>Focus(app, params)), nil
       }
       return <Type>Card(build<Type>(app)), nil   // existing tile
   })
   ```

Per-plan, you then:

4. **Build the component** `XxxFocus(view) g.Node` in the feature package — a
   **byte-faithful port** of the legacy `web/templates/xxx-focus.html` markup.
   Preserve **every CSS class and element id** (the served `basm.css` and the
   existing handlers/SSE depend on them). Reuse existing `ui.*` atoms / feature
   record-cards where their emitted classes already match; **hand-emit** the rest
   (sections, headings, bespoke widgets) rather than swapping in an atom that
   emits different classes. (The lifelog port hand-emits the `.spark` SVG for
   exactly this reason — `ui.Sparkline` would emit a different class tree.)
5. **Build the view-model + `buildXxxFocus(app, params)`** in the feature
   package, mirroring the web-side builder. **Feature packages must never import
   `internal/web`** (layering law) — copy any tiny web-only helper locally (the
   lifelog port copied `clip`, a 5-line `clipText`).
6. **Preserve the ACTION CONTRACT** — the single biggest risk. The ported markup
   must emit the **same** form `@post`/`@get` endpoints, the **same** form field
   names, and the **same** element ids that the existing handlers patch back over
   SSE. Each plan lists its contract explicitly; if you change an id or endpoint,
   the buttons silently no-op. Datastar attrs in gomponents:
   `g.Attr("data-on:submit", "@post('/ui/…', {contentType:'form'})")`,
   `g.Attr("data-on:click__prevent", "@get('/…')")`, `g.Attr("data-bind:q")`,
   `g.Attr("data-signals:q", "''")`.
7. **Storybook story** — add `xxxfocusStory()` to
   `internal/feature/storybook/stories_cards.go` and register it in
   `internal/feature/storybook/story.go`'s `stories` slice (insert it in the
   Cards cluster, **not** at the end — the Ollama branch appends there, so a
   mid-list insert auto-merges). + a contract **test** in the feature package
   asserting the preserved classes/ids (see `lifelogfocus_test.go`).
8. **Retire the legacy** `xxx_body`/`xxx_focus` template define and the now-dead
   web `xxxFocusHTML` handler — **only if** no test still executes them. The
   lifelog port deliberately left `internal/web/life.go` + the `life_body`
   template in place because `internal/web/templates_test.go`'s
   `TestLifeBodyRenders` still uses them (and that file is in the parallel
   branch's churn surface — editing it risks a merge conflict). **Per plan:
   grep for the template name in `*_test.go`; if a test uses it, leave the
   handler/template as dead code and note it for later cleanup.**

### Conventions

- Feature packages dot-import html: `. "maragu.dev/gomponents/html"` and alias
  core gomponents `g "maragu.dev/gomponents"`. Use `g.El("svg", …)`,
  `g.Attr`, `g.Text`, `g.Group`, `g.If`, `g.Raw` from `g`.
- `internal/ui` uses **qualified** `h.` (never dot-import) — but you're editing
  feature packages, which use the dot-import style; match the neighbouring file.
- `gofmt` is law; a PostToolUse hook formats edited Go, but run `gofmt -l` to be
  sure. No `internal/web` import from `internal/feature/*`.

### Verification gates (run all; paste output into the plan's done-check)

```
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./internal/feature/<pkg>/... ./internal/web/... ./internal/feature/storybook/...
gofmt -l internal/feature/<pkg> internal/web internal/feature/storybook   # expect empty
git diff --check
```
Then run the app and content-assert the live route (use `127.0.0.1`, not
`localhost`): build `CGO_ENABLED=0 go build -o /tmp/balaur .`, `/tmp/balaur serve
--http=127.0.0.1:8097`, and `curl -s 127.0.0.1:8097/focus/<type>` — grep for the
preserved classes/ids + the new topbar (`class="topbar"`, the active-domain
`aria-current`) + the dock (`id="dock"`). Optionally screenshot `/storybook/<id>`
in light AND dark (force `document.documentElement.className='theme-hearthwood
dark'` then `…light'`). `make test` / `make build` are the project's wrappers.

---

## Conflict map — the parallel Ollama→Kronk work

A second effort runs on **`improve/ollama-to-kronk`** (removing remote/Ollama,
moving inference in-process to Kronk). **Do not edit its files** — keep each port
to the body's feature package + `focus.go` + the body template. Off-limits:

- `internal/kronk/*`, `internal/turn/models.go`, `internal/store/llm_settings.go`
- `internal/web/models.go` (model handlers), `web/templates/models.html`
- `internal/feature/modelcards/*`, `internal/feature/storybook/stories_settings.go`
- the shared `plans/readme.md` index and `plans/0NN-*` numbering

**Settings (Models) is in their path** → its plan (`05`) ports **Profile only**
and defers Models. Quests / journal / day / knowledge are clean of this surface.

Work on a branch off `improve/ui-domain-pages` (or a worktree). The journal
guided-prompt (`journalPrompt` → `h.clients.Active`) is Ollama-*adjacent* but the
journal *body* port doesn't touch it — leave that handler alone.

---

## Plans

| # | Plan | Body | Effort | Conflict | Depends on |
|---|------|------|--------|----------|------------|
| 01 | `01-quests-focus.md` | quests rail + detail | M | clean | recipe (884d692) |
| 02 | `02-journal-focus.md` | journal candle | S–M | clean (prompt handler untouched) | recipe |
| 03 | `03-day-focus.md` | day-of-life | **L** | clean | recipe |
| 04 | `04-knowledge-focus.md` | memory + skills manager | M–L | clean | recipe |
| 05 | `05-settings-profile-focus.md` | settings **Profile only** | M | **Models BLOCKED on Kronk** | recipe; Models part waits for `improve/ollama-to-kronk` to land |

**Recommended order:** `01` (quests — moderate, reuses `TaskCard`) or `02`
(journal — smallest) first to confirm the recipe on a second body; then `04`
(knowledge — reuses the record cards); `03` (day) last of the clean set (richest
contract surface: path/query form asymmetry, an outer-patch SSE, a bespoke
recap-expander, dual-mode nav, three empty states); `05` Profile any time, Models
after Kronk merges. They're mutually independent otherwise — different files —
so they can also be done in parallel by different conversations.

**Status:** all `TODO`.

---

## Structural backlog (not yet written as plans — ask to flesh these out)

After the bodies, to *finish* the redesign (each touches the Ollama surface or
shared files, so sequence after Kronk lands or coordinate):

1. **`chat.Dock` organism** — port the dock chrome (`home.html` `chat_dock`:
   grip, recap zone, nudge poller, head/model switcher) to a gomponents organism,
   retiring `home.html`. *Touches the model switcher → coordinate with Kronk.*
2. **Cross-domain `@get` navigation** — make topbar domain links patch `#main`
   (no full reload, dock survives). *Touches `shell.Topbar` (`shell.go`), which
   the Ollama branch also edits — merge risk.*
3. **Retire boards** — remove `/boards*` routes + `board.js` + `boards.html`;
   `302 /boards → /`; decide the fate of the `board_compose`/`board_add_card`
   agent tools (`internal/tools/ui.go`) that lose their target. *Touches
   `web.go` routes.*
4. **Canon rewrite + dead-code cleanup** — update `DESIGN.md` +
   `internal/self/knowledge.md` to the Domain Navigator IA; delete the now-dead
   web `lifelogFocusHTML`/`lifeOverview`/`life_body` (and each body's retired
   handler/template) once the parallel-branch tests that reference them are
   reconciled. *Do after Kronk merges so the conflict-file tests can be edited.*

---

## Considered and rejected

- *Delete `internal/web/life.go` now (dead after the lifelog port).* Rejected:
  `templates_test.go::TestLifeBodyRenders` still executes `life_body`, and that
  test file is in the parallel branch's churn — deleting forces a conflict-file
  edit. Deferred to the canon-cleanup backlog item.
- *Continue the top-level `plans/0NN` numbering.* Rejected: collides with the
  Ollama branch's next plan numbers and the shared `plans/readme.md`.
</content>
