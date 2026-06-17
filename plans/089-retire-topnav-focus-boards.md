# Plan 089: Retire the top-nav, the `/focus/{type}` pages, and boards — drop the boards collection — and rewrite DESIGN.md + knowledge.md to the single-page chat+sidebar IA

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 089 row in `plans/readme.md` (add it if absent, matching the existing column format). Do NOT push or open a PR.
>
> **Drift check (run first)**:
> `git diff --stat 3136bad..HEAD -- internal/ui/shell/shell.go internal/ui/shell/shell_test.go internal/web/web.go internal/web/focus.go internal/web/boards.go internal/web/focus_test.go internal/web/boards_test.go internal/web/handlers_test.go internal/tools/ui.go internal/tools/ui_test.go internal/turn/tools.go internal/feature/storybook/stories_navigation.go internal/feature/storybook/story.go migrations/1750740000_boards.go web/templates/focus.html web/templates/boards.html web/templates/layout.html DESIGN.md internal/self/knowledge.md`
> If any changed since this plan was written, compare the "Current state" excerpts below to the live code; on mismatch, STOP and report which excerpt drifted.
>
> **Dependency gate (CRITICAL — read before any deletion)**: This plan **depends on plan 088** (the single-page chat shell + left domain sidebar + the deterministic `/ui/show` artifact-injection endpoint). 088 builds the replacement surface; 089 removes the old surfaces it replaces. **If plan 088 has NOT landed yet, STOP** — confirm `grep -rn "func.*Sidebar" internal/ui/shell/` shows the NEW interactive domain sidebar (not just the storybook-only `Sidebar`/`SidebarItem` from `sidebar.go`) AND a new artifact-injection route exists in `internal/web/web.go` (e.g. `/ui/show`). Removing the top-nav / `/focus` before 088 ships leaves the product with **no navigation at all**. Do not proceed on 089 until 088 is merged.

## Status
- **Priority**: P1
- **Effort**: L
- **Risk**: MED–HIGH (deletes the entire old navigation + a PocketBase collection; blast radius spans `internal/web`, `internal/ui/shell`, `internal/tools`, `internal/turn`, `migrations`, two docs, and dangling `/focus` links in `internal/feature/*cards`)
- **Depends on**: plan **088** (single-page chat shell + left domain sidebar + the deterministic `/ui/show` artifact endpoint — the replacement for everything this plan removes). Land 088 first.
- **Category**: feature (owner-requested full replacement) + tech-debt removal + docs
- **Planned at**: commit `3136bad`, 2026-06-17

## Why this matters
The owner has LOCKED a single-page redesign: one page = a **left domain sidebar + the chat as the only primary surface**; domain content arrives as **artifacts rendered into the conversation stream** (deterministic sidebar click, or a conversational tool call), both converging on the same card registry. Plan 088 builds that shell and the deterministic injection path. This plan (089) **retires everything the new surface replaces** so two parallel navigation systems do not coexist:

1. **The top-nav** (`shell.Topbar` + its desktop nav, off-canvas drawer, and burger) — superseded by the left domain sidebar.
2. **The `/focus/{type}` pages** — the "a page is a card at full size" mechanism. In the new IA a domain's content is an artifact injected into chat, not a full-canvas page. `/focus/{type}` and its dual-mode handler go away.
3. **Boards** — owner-composed dashboards of typed cards at `/boards`, plus the `board_compose`/`board_add_card` agent tools and the `boards` PocketBase collection. The owner's decision is explicit: **"drop the table completely, I don't care"** — do NOT preserve board data, do NOT recreate a board surface. Ad-hoc card clusters (plan 090) replace the "compose several cards" use case, in chat, without a separate board surface.

Leaving these in place would make Balaur lie about itself (the docs already drift) and would ship two competing nav systems. The honesty docs (`DESIGN.md` §3 the explicit "honesty ledger", and `internal/self/knowledge.md` — the running binary's self-description) MUST be rewritten in the same change to describe the single-page chat+sidebar+in-conversation-artifacts IA.

## What this plan does NOT remove (kept on purpose — verified)
- **`shell.Sidebar` / `SidebarSection` / `SidebarItem` / `SidebarPage`** in `internal/ui/shell/sidebar.go` — the GENERIC reusable left rail used ONLY by the storybook (`internal/web/storybook.go:sidebarFor`). 088 builds the product sidebar; 089 must not touch the storybook's. Keep `sidebar.go` and the storybook entirely.
- **`card_show` tool + `MarkUICard`/`ParseUICard` + the `cards` registry + `cardHTML`/`uicardBody`** — the in-conversation artifact machinery the new IA is built on. Keep.
- **The `/ui/cards` palette + `GET /ui/cards/{type}`** (`uiCardPalette`/`uiCard`) — the parameterized card resources. Keep (cards render in chat artifacts).
- **The settings card focus write endpoints** (`/ui/profile/*`, `/ui/heads/*`, `/ui/model/*`, `/ui/journal*`, `/ui/day/*`, `/ui/tasks/*`, `/ui/knowledge/*`, `/ui/recap/*`) — Keep. They are SSE patch endpoints, not pages.

## Current state (VERBATIM excerpts — confirmed by reading the files at `3136bad`)

### 1. The top-nav lives entirely in `internal/ui/shell/shell.go`
`Page` mounts the topbar + drawer, with a `with-sidebar` wrapper around `#main` and a `#dock` aside (`internal/ui/shell/shell.go:28-53`):
```go
28 // Page renders the full <html> document for one Balaur page.
29 func Page(p PageProps) g.Node {
...
39 	h.Body(
40 		h.A(h.Class("skip-link"), h.Href("#main"), g.Text("Skip to content")),
41 		Topbar(p.Active),
42 		h.Div(h.Class("with-sidebar"),
43 			h.Main(h.ID("main"), p.Body),
44 		),
45 		h.Aside(h.ID("dock"), p.Dock),
46 		topnavDrawer(p.Active),
47 	),
```
`Topbar` (`shell.go:83-105`), `topnavDrawer` (`shell.go:113-121`), `topbarLinks` (`shell.go:126-134`, the five `/focus/*` links), and `navLink` (`shell.go:136-143`) are the topbar-only helpers:
```go
126 func topbarLinks(active string) []g.Node {
127 	return []g.Node{
128 		navLink("/focus/quests", "Quests", "quests", active),
129 		navLink("/focus/memory", "Knowledge", "knowledge", active),
130 		navLink("/focus/lifelog", "Life", "life", active),
131 		navLink("/focus/journal", "Journal", "journal", active),
132 		navLink("/focus/settings", "Settings", "settings", active),
133 	}
134 }
```

### 2. `shell.Page` has three live callers — ALL of which 088 should be moving onto its new shell
```
internal/web/home.go:69       page := shell.Page(shell.PageProps{   (Home — 088's surface)
internal/web/focus.go:178     page := shell.Page(shell.PageProps{   (focus full-load — DELETED here)
internal/web/web.go:265       page := shell.Page(shell.PageProps{   (renderPageError — the in-app error page)
```
`renderPageError` (`internal/web/web.go:261-278`) wraps a `ui.EmptyState` in `shell.Page` so a handler failure stays in-app. **Decision (Step 1):** this plan keeps `shell.Page` ALIVE but strips the topbar/drawer out of it (so `renderPageError` and any 088 fallback still render a valid `<html>` shell), UNLESS plan 088 already replaced `shell.Page`'s body with the sidebar shell — in which case `renderPageError` and `home.go` already call 088's shell and `shell.Page` may be deletable. **Read `shell.go` and `home.go` at execution time and pick the branch that matches what 088 left** (see Step 1 STOP).

### 3. `shell.Topbar` is referenced by a storybook story
`internal/feature/storybook/stories_navigation.go:119`:
```go
119 			{"quests active", h.Div(h.Style("position:relative"), shell.Topbar("quests"))},
```
…inside `topbarStory()` (`stories_navigation.go:109-135`), which is registered in `internal/feature/storybook/story.go:100`:
```go
100 	topbarStory(),
```
Deleting `shell.Topbar` breaks the build of `stories_navigation.go` AND `TestAllStoriesRender`. The story must be removed in the same change (Step 1).

### 4. `shell_test.go` pins the topbar markup
`internal/ui/shell/shell_test.go` asserts topbar links and the drawer:
```go
37 		`<a href="/focus/quests" aria-current="page">Quests</a>`,
38 		`<a href="/focus/journal">Journal</a>`,
39 		`<a href="/focus/settings">Settings</a>`,
...
89 		`class="topnav-burger"`,
...
93 		`id="topnav-drawer"`,
```
`TestPage` (`:12`), `TestTopbarDrawer` (`:74`), `TestPageHTMLClass` (`:108`). These must be migrated (Step 1).

### 5. The `/focus/{type}` route + handler
`internal/web/web.go:237`:
```go
237 	se.Router.GET("/focus/{type}", h.focusPage)
```
`internal/web/focus.go` is the whole handler (190 lines): `focusActiveKey` (`:30-44`), `focusView` (`:51-59`), `safeBoardID` (`:65`), `focusBackHref` (`:71-76`), `focusBodyHTML` (`:81-83`), `focusParams` (`:88-97`), `focusCanonicalQuery` (`:100-109`), `focusPage` (`:112-189`). `focusPage`'s full-load branch (`:163-188`) renders `shell.Page` + the `focus_main` template; its Datastar branch (`:138-157`) patches `#main`.

`internal/web/focus_test.go` is the test file (327 lines, all `/focus/*` scenarios + `focusBackHref`/`focusCanonicalQuery` unit tests).

`web/templates/focus.html` defines `focus_main` + `focus_page` (the only two defines in the file):
```
7  {{define "focus_main"}}
16 {{define "focus_page"}}{{template "shell_open" .}}{{template "focus_main" .}}{{template "shell_close" .}}{{end}}
```

### 6. `cardFocusHTML` + `focusParams`/`HasManage` become orphaned once `focus.go` is gone
`internal/web/cards.go:100` `cardFocusHTML` is called ONLY from `focus.go:82` (grep-confirmed: its other hits are its own def + comments). `cards.HasManage` (`internal/cards/cards.go:206`) is called ONLY from `focus.go:93`. `ui.Focus` (the `CardSize` Focus value, `internal/ui/registry.go`) is used ONLY by `cardSizeInto` via `cardFocusHTML`. After `focus.go` is deleted these are dead. **Go does NOT flag unused package funcs/methods/consts** — only unused imports and unused locals break the build. So deleting `cardFocusHTML` is OPTIONAL surgical cleanup (Step 3 decides); `cards.HasManage` and `ui.Focus` are in OTHER packages and out of this plan's tight scope — leave them with a Maintenance note (a feature card may still implement a Focus branch; 088/090 may reuse a "full size" render).

### 7. The board routes + handler + template
`internal/web/web.go:238-246`:
```go
238 	// Boards — owner-composed dashboards of typed cards (plan 029).
239 	se.Router.GET("/boards", h.boardsIndex)
240 	se.Router.GET("/boards/{id}", h.boardsPage)
241 	se.Router.POST("/ui/boards", h.boardsCreate)
242 	se.Router.POST("/ui/boards/{id}/rename", h.boardsRename)
243 	se.Router.POST("/ui/boards/{id}/delete", h.boardsDelete)
244 	se.Router.POST("/ui/boards/{id}/cards/add", h.boardsCardAdd)
245 	se.Router.POST("/ui/boards/{id}/cards/{idx}/remove", h.boardsCardRemove)
246 	se.Router.POST("/ui/boards/{id}/layout", h.boardsLayout)
```
`internal/web/boards.go` is the whole handler (610 lines): `boardCard` alias, `boardView`/`boardRecord`/`boardCardView` types, `boardCardViewsOf`, `boardCardsOf`, `boardRecordOf`, `renderBoardCards`, `loadBoards`, `ensureDefaultBoards`, and the eight handlers above. (After plan 083, `boardsIndex`/`boardsPage` already just `e.Redirect(http.StatusFound, "/")` — `boards.go:273-280`.)

`web/templates/boards.html` after plan 083 has NO `<html>` wrapper — it defines ONLY `board_header`, `board_grid`, `board_add` (confirmed: lines `1`, `24`, `57`). `board_header` is rendered by `boardsRename` (`boards.go:353`); `board_grid` by `boardsCardAdd`/`boardsCardRemove` (`boards.go:456`,`518`). `board_add` is already orphaned (083 noted it).

`internal/web/boards_test.go` is the test file (595 lines).

### 8. The board agent tools live in `internal/tools/ui.go`
```go
60 // UITools returns the card_show, board_compose, and board_add_card tools.
61 func UITools(app core.App) []agent.Tool {
62 	return []agent.Tool{cardShowTool(app), boardComposeTool(app), boardAddCardTool(app)}
63 }
```
`boardComposeTool` (`ui.go:126-220`) and `boardAddCardTool` (`ui.go:222-343`) both read/write the `boards` collection and audit `board_compose`/`board_add_card`. `cardShowTool` (`ui.go:65-124`) stays.

`UITools` is called from BOTH `internal/turn/tools.go:27` (`Tools`) and `:65` (`ToolsForHead`). After trimming, `UITools` returns ONLY `card_show`; the call sites do not change.

`internal/tools/ui_test.go` tests `cardShowTool` directly (`:131,151,171,189` — KEEP) and the board tools via `UITools` + `findTool` (`TestBoardComposeCreatesRecord` `:203`, `TestBoardComposeWritesAuditLog` `:233`, and more below — DELETE the board-tool tests).

### 9. The boards migration (mirror this for the Down)
`migrations/1750740000_boards.go`:
```go
16 func boardsUp(app core.App) error {
17 	owner := types.Pointer(ruleOwner)
18 	boards := core.NewBaseCollection("boards")
19 	setOwnerRules(boards, owner)
20 	boards.Fields.Add(
21 		&core.TextField{Name: "name", Required: true, Max: 80},
22 		&core.JSONField{Name: "cards"},
23 		&core.NumberField{Name: "sort"},
24 		&core.AutodateField{Name: "created", OnCreate: true},
25 		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
26 	)
27 	boards.AddIndex("idx_boards_sort", false, "sort", "")
28 	return app.Save(boards)
29 }
```
`ruleOwner` (`migrations/1749600000_init.go:19`) and `setOwnerRules` (`:148`) are shared package helpers — available to the new migration (same `migrations` package). `migrations/1750740000_boards_test.go` (`TestBoardsCollectionExists`) asserts the collection EXISTS — it must be DELETED (the collection is gone). The latest migration timestamp on disk is `1750840000` (`migrations/1750840000_kronk_local_models.go`); **`1750850000` is free** (confirmed: no `1750850000_*` file exists; `timestamp_uniqueness_test.go` enforces unique strictly-increasing prefixes).

### 10. The two docs (rewrite targets)
`DESIGN.md:89-105` (the IA paragraph) currently describes the topbar domain rail + `/focus/{type}` + boards + `302 → /`, and `:169-183` describes the typed-card registry + the candle/`/focus/journal` + boards + the three on-the-spot tools. `internal/self/knowledge.md:103-134` (Surfaces), `:136-150` (the per-domain focus prose), `:169-203` (Boards + the three agent UI tools), `:209` (`/focus/settings?section=models`). All three describe the to-be-removed top-nav + `/focus` + boards and MUST be rewritten to the single-page chat+sidebar+artifacts IA established by 088 (and the ad-hoc clusters of 090, framed as "from chat"). The `card_show` tool description and the `cards` registry prose STAY.

### 11. BLAST RADIUS — dangling `/focus` and `/boards` inbound links (MUST be re-pointed or this plan ships dead links)
After `/focus/{type}` and `/boards` are gone, every inbound link to them dangles. Grep-confirmed inbound references at `3136bad`:

**In `internal/feature/*cards/` (card footers/links rendered INTO chat artifacts — the new primary surface):**
```
internal/feature/journalcards/dayfocus.go:181,269   Href("/focus/day?date="+…)
internal/feature/journalcards/journal.go:72         A(Href("/focus/day?date="+v.TodayDate), …)
internal/feature/journalcards/day.go:96             A(Href("/focus/day?date="+v.Date), …)
internal/feature/journalcards/journalfocus.go:81    h.Href("/focus/day?date="+e.Date)
internal/feature/knowledgecards/skills.go:121,139,226   Href("/focus/skills")
internal/feature/knowledgecards/memory.go:59,76,185     h.Href("/focus/memory")
internal/feature/lifecards/lines.go:71              Href("/focus/lifelog")
internal/feature/lifecards/lifelog.go:125           Href("/focus/lifelog")
internal/feature/lifecards/measure.go:89            Href("/focus/lifelog")
internal/feature/lifecards/habits.go:60             Href("/focus/lifelog")
internal/feature/taskcards/calendar.go:126          Href("/focus/calendar")
internal/feature/taskcards/calendar.go:164          A(Class("cal-daylink"), Href("/focus/day?date="+cell.Date), …)
internal/feature/taskcards/today.go:72              Href("/focus/quests")
internal/feature/taskcards/quests.go:31,70          Href("/focus/quests")
internal/feature/taskcards/timeline.go:89           Href("/focus/timeline")
internal/feature/settingscards/settings.go:23-28    Href("/focus/settings?section=…") + Href("/focus/settings")
internal/feature/settingscards/settingsfocus.go:339 href := "/focus/settings?section=" + section
```
**In templates served at page load:**
```
web/templates/home.html:24,29,37   /focus/settings?section=models|profile (the chatbar model/avatar links)
web/templates/profile.html:4        (comment only — /focus/settings?section=profile)
web/templates/journal-focus.html:35 href="/focus/day?date={{.Date}}"
web/templates/recap-cards.html:12   href="/focus/day?date={{.Date}}"
web/templates/layout.html:25-26    /boards + /focus/settings (the LEGACY topbar — only pulled by boards.html, which is being trimmed)
```
**THE OPEN QUESTION (resolve with 088's design before executing):** these footer links were the way a tile said "open the full surface." In the single-page IA there is no `/focus` page; the equivalent is "inject the full artifact into chat" — i.e. the `GET /ui/show/{type}` deterministic endpoint plan 088 introduces (the card TYPE is a path segment; any other params ride the query string, e.g. `/ui/show/day?date=…`). **Decision (Step 5):** re-point each `/focus/{type}?params` link to the 088 deterministic artifact-injection endpoint (a Datastar `@get('/ui/show/{type}?…')` that appends the artifact to `#chat`), NOT a hard navigation. The EXACT endpoint name/shape is owned by 088 — read 088's route (`internal/web/web.go`, `GET /ui/show/{type}`) and its `SidebarItem` action (`Action: "@get('/ui/show/<type>')"`) and mirror that exact contract. If 088 did NOT already re-point these (i.e. they still say `/focus`), this plan re-points them; if 088 already did, this plan only verifies none remain. **STOP if 088's endpoint contract is ambiguous** — do not invent one; report and ask. (The Done-criteria `grep -rn "/focus/" internal/ web/` is the completeness gate — it catches `Href(...)`, `href=...`, AND `data-on:click__prevent="@get('/focus/...')"` forms; re-point until it returns empty.)

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Drift check | see header | excerpts match |
| 088 landed? | `grep -rn "func.*Sidebar" internal/ui/shell/; grep -n "/ui/show" internal/web/web.go` | the NEW product sidebar + the artifact endpoint exist |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Storybook render | `go test ./internal/feature/storybook/...` | ok (`TestAllStoriesRender`) |
| Migration tests | `go test ./migrations/...` | ok (uniqueness + no boards-exists test) |
| Format | `gofmt -l .` | empty |
| Whitespace | `git diff --check` | no output |
| No `/focus` render path | `grep -rn "/focus/" internal/ web/ --include=*.go --include=*.html` | empty (excl. plans/docs/comments-of-history) |
| No `/boards` route/render | `grep -rn "/boards\|boardsIndex\|boardsPage\|board_compose\|board_add_card\|boardComposeTool\|boardAddCardTool" internal/ web/ --include=*.go --include=*.html` | empty |
| Boards collection gone | a migration test asserting `FindCollectionByNameOrId("boards")` errors after up | pass |

Sandbox note: a TLS-intercepting Hyperagent sandbox needs the GOPROXY shim (`docs/hyperagent-sandbox.md`; GOSUMDB stays on).

## Scope
**In scope** (the only files you may modify/delete):
- `internal/ui/shell/shell.go` — delete `Topbar`, `topbarLinks`, `navLink`, `topnavDrawer`; rework or delete `Page` (Step 1 decision); delete `focusActiveKey`-style topbar plumbing if present.
- `internal/ui/shell/shell_test.go` — migrate/delete the topbar+drawer assertions.
- `internal/feature/storybook/stories_navigation.go` + `internal/feature/storybook/story.go` — remove `topbarStory()` and its registration.
- `internal/web/web.go` — remove the `/focus/{type}` route and the eight `/boards*` routes; update `renderPageError` if Step 1 changes `shell.Page`.
- `internal/web/focus.go` — DELETE the file.
- `internal/web/focus_test.go` — DELETE the file.
- `internal/web/boards.go` — DELETE the file.
- `internal/web/boards_test.go` — DELETE the file.
- `internal/web/handlers_test.go` — fix the comments/assertions that reference `/focus`, `/boards`, and "302 → /boards" (now 302 → `/`); the heads-redirect + retired-route scenarios stay but their `/boards` wording must become `/`.
- `internal/web/cards.go` — OPTIONAL: delete `cardFocusHTML` if Step 3 confirms zero callers (surgical).
- `internal/tools/ui.go` — delete `boardComposeTool` + `boardAddCardTool`; trim `UITools` to `[]agent.Tool{cardShowTool(app)}`; fix the `UITools` doc comment.
- `internal/tools/ui_test.go` — delete the board-tool tests (and `findTool` if it becomes unused); keep all `cardShowTool`/`MarkUICard`/`ParseUICard` tests.
- `web/templates/focus.html` — DELETE the file.
- `web/templates/boards.html` — DELETE the file.
- `web/templates/layout.html` — DELETE the legacy `topbar` define (`:21-38`) and the legacy `shell_open`/`shell_close` (`:40-56`) IF they have no remaining caller after focus.html + boards.html are gone (Step 6 grep-gates this; KEEP `page_head` if still used). VERIFY before deleting.
- `migrations/1750850000_drop_boards.go` — NEW: Up deletes the `boards` collection; Down recreates it (mirror `1750740000_boards.go`).
- `migrations/1750740000_boards_test.go` — DELETE (asserts the collection exists; it no longer does).
- `migrations/1750850000_drop_boards_test.go` — NEW: assert the collection is gone after up (and round-trips on down).
- The `internal/feature/*cards/` + `web/templates/*.html` `/focus`/`/boards` inbound links listed in §11 — re-point to 088's deterministic artifact endpoint (Step 5).
- `DESIGN.md` — rewrite §3 IA paragraph + the typed-card/boards prose to the single-page IA.
- `internal/self/knowledge.md` — rewrite the Surfaces + per-domain + Boards + agent-UI-tools prose.

**Out of scope** (do NOT touch):
- `internal/ui/shell/sidebar.go` (`Sidebar`/`SidebarItem`/`SidebarPage`) and `internal/web/storybook.go` — the storybook's separate rail. Untouched.
- `cards` package, `card_show` tool, `MarkUICard`/`ParseUICard`, `cardHTML`/`uicardBody`, `/ui/cards*` routes — the artifact machinery. Untouched.
- `cards.HasManage` (`internal/cards/cards.go`) and `ui.Focus` / `CardSize` (`internal/ui/registry.go`) — orphaned by `focus.go`'s deletion but in other packages; leave + Maintenance note.
- `internal/web/assets/static/basm.css` — the obsolete `.topbar`/`.topnav-*`/`.board-*`/`.focus`/`.with-sidebar` rules become dead CSS. Layout/CSS for the new shell is plan 088's; CSS hygiene is plan 091. Do NOT delete CSS here (note it). EXCEPTION: only if 088 left a rule this plan's markup deletion makes parse-broken (it won't — CSS is not parsed against markup).
- `internal/web/assets/static/board.js` — orphaned static asset (noted by plan 083). Leave + note.
- `AGENTS.md` / `README.md` — touch only a provably-false line; otherwise leave (the orchestrator/owner owns the README IA rewrite). README has `/focus/*` mentions (§ `README.md:86,93,98,178`); flag them in your report, do not rewrite README in this plan unless the owner scoped it in.
- Plan 090's `ArtifactMarker`/`show_cards`/bare-`tasks` cluster — that is 090; do not add it here.

## Git workflow
- Branch `improve/089-retire-topnav-focus-boards` off `main` (or stacked on 088's branch if not yet merged).
- Conventional commits, e.g.:
  - `refactor(ui): retire the top-nav (Topbar + drawer) from the page shell`
  - `refactor(web): remove the /focus/{type} pages`
  - `refactor(web,tools): remove boards (routes, handler, board_* tools)`
  - `feat(migrations): drop the boards collection`
  - `refactor(cards): re-point /focus tile links to the in-chat artifact endpoint`
  - `docs: rewrite DESIGN.md + knowledge.md to the single-page chat+sidebar IA`
- Do NOT push or open a PR unless explicitly told.

## Steps

### Step 1: Retire the top-nav from the shell + migrate its tests + drop the topbar story
1. **Branch off the 088 reality.** Read `internal/ui/shell/shell.go` and `internal/web/home.go` as they exist post-088. Decide:
   - **Branch A — 088 already replaced `shell.Page`'s body with the sidebar shell:** then `Topbar`/`topbarLinks`/`navLink`/`topnavDrawer` are already unreferenced by `Page`. Delete those four funcs; keep `Page`/`PageProps` as 088 left them. `home.go` and `renderPageError` already use the new shell — no change there.
   - **Branch B — 088 introduced a NEW shell func and left `shell.Page` as the old topbar shell:** then delete `Topbar`/`topbarLinks`/`navLink`/`topnavDrawer` AND delete `shell.Page`/`PageProps` if `home.go` + `renderPageError` no longer call it (grep `shell.Page` — see §2). If `renderPageError` still calls `shell.Page`, re-point it to 088's shell instead (the in-app error page should sit in the SAME single-page shell). **STOP** and report if neither branch matches (088 left an unexpected shape).
2. Remove the `shell.Topbar("quests")` variant: delete `topbarStory()` from `internal/feature/storybook/stories_navigation.go` and its registration line `topbarStory(),` from `internal/feature/storybook/story.go:100`. Drop the now-unused `shell` import from `stories_navigation.go` if it becomes unused (gofmt/build will flag).
3. Migrate `internal/ui/shell/shell_test.go`: delete `TestTopbarDrawer` and the topbar-link assertions inside `TestPage` (`:37-65`); keep `TestPageHTMLClass` IF `HTMLClass` survives 088 (read 088's `PageProps`). If `shell.Page` was deleted in Branch B, delete the whole `shell_test.go` (or replace it with whatever 088's shell test covers — but 088 owns that test; don't duplicate).
**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./internal/ui/shell/... ./internal/feature/storybook/...` → pass; `grep -rn "Topbar\|topnav-drawer\|topnav-burger\|topbarLinks\|navLink(" internal/ui/ internal/feature/storybook/` → empty.

### Step 2: Remove the `/focus/{type}` route + handler + template + test
1. In `internal/web/web.go` delete line `237` (`se.Router.GET("/focus/{type}", h.focusPage)`) and the comment above it that references the typed card registry's focus surface if it only refers to `/focus`.
2. `git rm internal/web/focus.go` and `git rm internal/web/focus_test.go`.
3. `git rm web/templates/focus.html`.
**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 (an unused import surfaced by deleting `focus.go` must be cleaned where it lives); `grep -rn "focusPage\|focus_main\|focus_page\|focusBackHref\|focusActiveKey\|focusParams\|focusBodyHTML" internal/ web/ --include=*.go --include=*.html` → empty; `go test ./internal/web/...` → pass (note: tests still referencing `/focus` are fixed in Step 4/5).

### Step 3: Decide the fate of `cardFocusHTML`
Run `grep -rn "cardFocusHTML" internal/`. With `focus.go` gone its only caller is removed.
- **Surgical option (recommended):** delete `cardFocusHTML` (`internal/web/cards.go:97-114`). If that leaves `cardSizeInto`'s `ui.Focus` branch with no caller, that is fine — `cardSizeInto` stays (it is the shared size dispatch used by `cardHTML` at `ui.Tile`). Do NOT delete `cardSizeInto`.
- If deleting `cardFocusHTML` reveals an unused import in `cards.go`, clean it.
- Leave `cards.HasManage` and `ui.Focus` (other packages, out of scope) — Maintenance note.
**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0.

### Step 4: Remove boards — routes, handler, tools, template, tests
1. In `internal/web/web.go` delete the boards block (`:238-246`, the comment + all eight routes).
2. `git rm internal/web/boards.go` and `git rm internal/web/boards_test.go`.
3. `git rm web/templates/boards.html`.
4. In `internal/tools/ui.go`: delete `boardComposeTool` (`:126-220`) and `boardAddCardTool` (`:222-343`); change `UITools` to `return []agent.Tool{cardShowTool(app)}` and fix its doc comment (`:60`) to "returns the card_show tool." Drop the `store` import if it becomes unused (board tools used `store.Audit`; `cardShowTool` does not — gofmt/build will flag).
5. In `internal/tools/ui_test.go`: delete every board-tool test (`TestBoardComposeCreatesRecord`, `TestBoardComposeWritesAuditLog`, and all board_* tests after `:201`) and delete the `findTool` helper IF it is now unused (grep). Keep all `cardShowTool`/`MarkUICard`/`ParseUICard` tests.
6. `internal/turn/tools.go` needs NO change — it calls `tools.UITools(app)` (`:27`,`:65`), which still returns a valid (now single-tool) slice. Confirm by grep that `turn` does not name `board_compose`/`board_add_card` directly (it does not).
**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0; `go test ./internal/tools/... ./internal/turn/...` → pass; `grep -rn "board_compose\|board_add_card\|boardComposeTool\|boardAddCardTool\|boardsIndex\|boardsPage\|loadBoards\|ensureDefaultBoards" internal/ web/ --include=*.go --include=*.html` → empty.

### Step 5: Re-point the dangling `/focus` and `/boards` inbound links (§11)
**Read 088's deterministic artifact endpoint contract first** (its route in `web.go` + the `SidebarItem` action it wires). The target is "inject this domain's artifact into the chat stream," NOT a page navigation.
1. For each `internal/feature/*cards/` link in §11: replace `Href("/focus/{type}?params")` with the 088 artifact action — mirror 088's `SidebarItem` exactly: an `<a>` carrying `data.On("click", "@get('/ui/show/{type}?…')", data.ModifierPrevent)` (the card type is the PATH segment; remaining params ride the query string) plus the same URL as the `href` no-JS fallback. Keep the visible label ("all quests →", "open the day →", etc.).
   - `/focus/day?date={date}` → `@get('/ui/show/day?date={date}')`.
   - `/focus/skills` → `@get('/ui/show/skills')`; `/focus/memory` → `@get('/ui/show/memory')`.
   - `/focus/lifelog` → `@get('/ui/show/lifelog')`; `/focus/quests` → `@get('/ui/show/quests')`; `/focus/calendar` → `@get('/ui/show/calendar')`; `/focus/timeline` → `@get('/ui/show/timeline')`.
   - `/focus/settings?section=…` (settings.go, settingsfocus.go) → `@get('/ui/show/settings?section=…')`.
2. For templates: `web/templates/home.html:24,29,37` (settings model/profile links) and `web/templates/journal-focus.html:35`, `web/templates/recap-cards.html:12` (day links) → the same artifact endpoint. `web/templates/profile.html:4` is a comment — update or leave (cosmetic).
3. **If 088 ALREADY re-pointed all of these** (grep `/focus` in `internal/feature` + `web/templates` returns empty), this step is just the verify.
4. **STOP** if 088's endpoint contract is ambiguous or absent — report; do not invent an endpoint.
**Verify**: `grep -rn "/focus/" internal/ web/ --include=*.go --include=*.html` → empty (only `plans/`, `DESIGN.md`/`knowledge.md` history mentions remain, handled in Step 8); `go test ./internal/feature/... ./internal/web/...` → pass; `go test ./internal/feature/storybook/...` → pass (re-pointed card footers still render).

### Step 6: Trim/delete `web/templates/layout.html` legacy defines (grep-gated)
After focus.html + boards.html are gone, grep what still references `layout.html`'s defines:
```
grep -rn '"topbar"\|"shell_open"\|"shell_close"\|"page_head"' internal/ web/templates/
```
- `topbar` (`layout.html:21-38`) — the LEGACY template topbar; its only puller was the (now-gone) full-doc wrappers. If grep shows zero callers, delete the define.
- `shell_open`/`shell_close` (`:40-56`) — pulled only by the deleted `focus.html:focus_page`. If zero callers, delete them.
- `page_head` (`:8-19`) — check callers; if zero, the whole `layout.html` can be `git rm`'d. If something still pulls `page_head`, keep the file with only the live defines.
**STOP** if a define still has a live caller you did not expect — report it rather than deleting and breaking template parse (`TestTemplatesParse` would catch it, but stop first).
**Verify**: `go test ./internal/web/...` → pass (`TestTemplatesParse` re-parses `templates/*.html`; a dangling `{{template "x"}}` fails parse, so green proves no dangling reference).

### Step 7: Drop the boards collection (migration) + its test
1. Create `migrations/1750850000_drop_boards.go`:
```go
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// drop boards (plan 089) — the boards surface and the board_compose/board_add_card
// tools were retired in favor of the single-page chat+sidebar + in-conversation
// artifacts. Owner decision: drop the table completely; do not preserve data.
func init() {
	m.Register(dropBoardsUp, dropBoardsDown)
}

func dropBoardsUp(app core.App) error {
	col, err := app.FindCollectionByNameOrId("boards")
	if err != nil {
		return nil // already gone — idempotent
	}
	return app.Delete(col)
}

// dropBoardsDown recreates the boards collection (schema mirrored from
// 1750740000_boards.go) for reversibility — data is NOT restored.
func dropBoardsDown(app core.App) error {
	owner := types.Pointer(ruleOwner)
	boards := core.NewBaseCollection("boards")
	setOwnerRules(boards, owner)
	boards.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 80},
		&core.JSONField{Name: "cards"},
		&core.NumberField{Name: "sort"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	boards.AddIndex("idx_boards_sort", false, "sort", "")
	return app.Save(boards)
}
```
(`ruleOwner`/`setOwnerRules` are the same-package helpers from `1749600000_init.go` — confirm they are still exported within the package via build.)
2. `git rm migrations/1750740000_boards_test.go` (its `TestBoardsCollectionExists` now fails by design).
3. Create `migrations/1750850000_drop_boards_test.go` asserting that after the migrations run the `boards` collection is ABSENT. Use the `storetest`/`storetest.NewApp(t)` helper the deleted test used (`migrations/1750740000_boards_test.go` imported `github.com/alexradunet/balaur/internal/storetest`):
```go
package migrations_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestBoardsCollectionDropped(t *testing.T) {
	app := storetest.NewApp(t) // runs all registered migrations
	if _, err := app.FindCollectionByNameOrId("boards"); err == nil {
		t.Fatal("boards collection still exists after drop migration")
	}
}
```
**Verify**: `go test ./migrations/...` → pass (uniqueness test still green — `1750850000` is unique; the new drop test passes; the old boards-exists test is gone). `go test ./internal/web/... ./internal/tools/...` → pass (no surviving code reads the `boards` collection — confirmed by Step 4 grep).

### Step 8: Rewrite DESIGN.md + internal/self/knowledge.md to the single-page IA
Precedents: plan **085** (DESIGN.md honesty-ledger refresh — match its terse `·`-separated voice, no emoji, no hype) and plan **057** (the card-first finale doc pass). Keep the warm/plain voice (DESIGN.md §2).

1. **DESIGN.md §3 IA paragraph (`:89-105`)** — rewrite from "Home + domain rail topbar + `/focus/{type}` + boards" to:
   > Single page. `/` is the only product surface: a **left domain sidebar** (Quests, Knowledge, Life, Journal, Heads, Settings) beside the **companion chat**, which is the only primary content area. There are no feature pages and no top-nav. Domain content arrives as **artifacts rendered into the conversation stream**, two ways that converge on one card registry: (1) **deterministic** — clicking a sidebar domain injects that domain's artifact (a card / card-cluster) into the chat instantly, no model call; (2) **conversational** — a natural-language request makes the agent call `card_show` (or `show_cards`, plan 090) to render the same artifact. A summoned artifact is a permanent, live transcript entry: it survives reload and re-renders from current data each time (the `card_show` marker contract). A *card* is a typed, parameterized, server-rendered resource (`/ui/cards/{type}`); the agent composes from the registry only — it cannot author markup. A **head switcher** in the sidebar changes the active persona without leaving or forking the conversation.
   - Delete the "Legacy flat routes … 302 → /" sentence and the `/focus/{type}`/`/boards`/`board_compose`/`board_add_card` mentions (`:93-105`, `:100-101`).
2. **DESIGN.md typed-card/boards prose (`:169-183`)** — keep the typed-card-registry paragraph (14 types, `/ui/cards/{type}`) but drop "the composition unit for boards", the candle's `/focus/journal` phrasing (say "the journal artifact" / "in chat"), and DELETE the entire boards paragraph (`:176-183`) + the "on-the-spot UI — card_show / board_compose / board_add_card" sentence; replace with "on-the-spot UI — `card_show` embeds a typed card inline in chat; `show_cards` (plan 090) hands the agent N cards as one cluster artifact — both compose from the typed registry only."
3. **internal/self/knowledge.md Surfaces (`:103-134`)** — rewrite: `/` is the single page (left domain sidebar + companion chat); the sidebar is the nav (no topbar, no side-rail-vs-dock split); domains inject artifacts into chat (deterministic click via the 088 endpoint, or `card_show`); there are no `/focus/{type}` pages and no `/boards`. Keep the dock/chat-component prose (chat.Message/ToolRow/Dock, `/ui/chat`) and the CLI/PocketBase-dashboard sentences. Fix `:209` (`/focus/settings?section=models`) → the settings artifact / Models surface.
4. **internal/self/knowledge.md per-domain focus prose (`:136-150`)** — reframe "the quests card's focus at /focus/quests" etc. as "the quests artifact" / "the day artifact"; drop the `/focus/...` URLs.
5. **internal/self/knowledge.md Boards + agent-UI-tools (`:169-203`)** — DELETE the Boards paragraph (`:169-181`) and the `board_compose`/`board_add_card` bullets (`:192-200`); keep the `card_show` bullet; add a `show_cards` note marked "(plan 090)". Fix the "The registry vocabulary for both tools" sentence (now one tool, soon two).
6. Cross-check the three docs agree (DESIGN.md ↔ AGENTS.md ↔ knowledge.md). AGENTS.md already says "no MCP / artifacts as cards" but mentions boards in the `board_compose` context only via plan-history — if AGENTS.md has a CURRENT-tense boards claim, flag it (do not rewrite AGENTS.md beyond a provably-false line).
**Verify**:
- `grep -niE "board_compose|board_add_card|/boards|/focus/" DESIGN.md internal/self/knowledge.md` → no CURRENT-tense match (only plan-history parentheticals allowed).
- `grep -niE "single page|domain sidebar|artifact|inject" DESIGN.md internal/self/knowledge.md` → matches the new IA.
- `go test ./...` → still green (docs-only edits change no behavior).

### Step 9: Full validation gate + index
Run the Done-criteria gate. Update the 089 row in `plans/readme.md` (do NOT touch the rest of the index — the advisor owns it; add only the 089 row if absent).

## Test plan
- **shell + storybook**: `go test ./internal/ui/shell/... ./internal/feature/storybook/...` — the migrated `shell_test.go` (no topbar assertions) and `TestAllStoriesRender` (no `topbarStory`) pass.
- **web**: `go test ./internal/web/...` — `focus_test.go`/`boards_test.go` are gone; `handlers_test.go` retired-route scenarios now assert 302 → `/` (the catch-all home redirect in `home.go:root`), no `/focus`/`/boards` content assertions remain.
- **tools**: `go test ./internal/tools/...` — `cardShowTool` tests pass; no board-tool tests; `UITools` returns one tool.
- **migrations**: `go test ./migrations/...` — `TestMigrationTimestampsAreUnique` green (`1750850000` unique), `TestBoardsCollectionDropped` green, old `TestBoardsCollectionExists` removed.
- **Whole suite**: `go test ./...`.
- **No new storybook story** is added (this plan only REMOVES `topbarStory`; the new sidebar/shell story belongs to 088). Confirm `go test ./internal/feature/storybook/...` stays green.
- **Manual (after 088, app running)**: load `/` → only the left domain sidebar + chat; click each domain → its artifact appends to `#chat` (deterministic) and survives reload; no topbar, no `/focus/*` page, `curl -s -o /dev/null -w "%{http_code} %{redirect_url}\n" http://127.0.0.1:8090/focus/quests` and `…/boards` → 302 → `/` (caught by the `root` catch-all); confirm in BOTH `theme-hearthwood dark` and `light`.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0.
- [ ] `go test ./...` → all pass (incl. `TestAllStoriesRender`, `TestMigrationTimestampsAreUnique`, `TestBoardsCollectionDropped`).
- [ ] `grep -rn "Topbar\|topbarLinks\|topnav-drawer\|topnav-burger\|navLink(" internal/ui/ internal/feature/storybook/` → empty.
- [ ] `grep -rn "/focus/" internal/ web/ --include=*.go --include=*.html` → empty (re-pointed to 088's artifact endpoint; only `plans/` + docs history remain).
- [ ] `grep -rn "/boards\|boardsIndex\|boardsPage\|board_compose\|board_add_card\|boardComposeTool\|boardAddCardTool\|loadBoards\|ensureDefaultBoards\|focusPage" internal/ web/ --include=*.go --include=*.html` → empty.
- [ ] `internal/web/focus.go`, `internal/web/focus_test.go`, `internal/web/boards.go`, `internal/web/boards_test.go`, `web/templates/focus.html`, `web/templates/boards.html`, `migrations/1750740000_boards_test.go` no longer exist.
- [ ] `migrations/1750850000_drop_boards.go` exists; Up deletes the `boards` collection (idempotent), Down recreates the schema; `FindCollectionByNameOrId("boards")` errors after migrations run.
- [ ] `UITools` returns exactly `[card_show]`; `internal/turn/tools.go` unchanged and building.
- [ ] DESIGN.md §3 and `internal/self/knowledge.md` describe the single-page chat+sidebar+in-conversation-artifacts IA with no current-tense `/focus`, `/boards`, `board_compose`, or `board_add_card` claim.
- [ ] `gofmt -l .` empty; `git diff --check` no output.
- [ ] Only in-scope files changed (plus the `plans/readme.md` 089 row).
- [ ] The 089 row in `plans/readme.md` updated.

## STOP conditions
- **088 not landed**: the dependency gate (header) fails — STOP. Removing the top-nav/`/focus`/boards before 088 ships leaves no navigation.
- **088's artifact endpoint contract is ambiguous/absent** (Step 5) — STOP; do not invent the `/ui/show` shape. Report and ask.
- **A "dead" symbol turns out live**: `cardFocusHTML`, a board helper, or a `layout.html` define still has a real caller after the grep — STOP, do not delete; report.
- **Migration timestamp clash**: a `1750850000_*` file already exists, or `TestMigrationTimestampsAreUnique` fails — STOP; pick the next free strictly-increasing prefix and report.
- **Drift**: a cited file changed since `3136bad` and an excerpt no longer matches — STOP and report which excerpt drifted.
- **Re-pointing `/focus` links would require a 088 endpoint that does not exist** — STOP; the inbound links are load-bearing (they render in chat artifacts). Do not leave dead links and do not hard-navigate to a deleted page.
- **A Verify fails twice** after a fix attempt — STOP and report the command + output.

## Maintenance notes
- **Dead CSS (deferred to plan 091)**: deleting the topbar/focus/board markup leaves `.topbar`, `.topnav-*`, `.board-grid`/`.board-slot*`, `.focus`/`.focus-header`/`.focus-body`, and the `.with-sidebar`/`#main`/`#dock` layout rules in `internal/web/assets/static/basm.css` orphaned. The new single-page layout/CSS is plan 088's; CSS hygiene (delete dead rules, measure caps, reduced-motion, a11y) is plan 091. Do not delete CSS in 089.
- **Orphaned static asset**: `internal/web/assets/static/board.js` (already flagged by plan 083) is fully dead after 089. Removing embedded assets is a separate cleanup.
- **`cards.HasManage` + `ui.Focus`/`CardSize`** lose their only caller (`focus.go`) but live in other packages; left in place. If 090's bare-`tasks` cluster or 088's "full artifact" render never re-uses the Focus size, a later plan can remove the `Focus` `CardSize` value and `HasManage` together.
- **README.md** still has `/focus/*` mentions (`:86,93,98,178`); they are user-facing copy and out of this plan's tight scope. Flag for an owner-scoped README pass.
- **Three-document agreement** (DESIGN.md / AGENTS.md / knowledge.md) is a standing contract (DESIGN.md §3 "update it the moment shape changes"). 090 (clusters) and 091 (sidebar chrome) must keep all three in sync as they land.
- **Reversibility of the drop**: the Down migration recreates the `boards` schema but NOT its data — per the owner's explicit "drop completely, I don't care." If boards data ever needs recovery, it must come from a pre-089 backup, not the migration.
