# Plan 083: Delete the orphaned focus-body templates and reconcile the competing /boards page shell

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 083 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/web/settings.go internal/web/tasks.go internal/web/day.go internal/web/life.go internal/web/knowledge.go internal/web/models.go internal/web/boards.go internal/web/web.go internal/web/focus.go internal/web/templates_test.go internal/web/boards_test.go internal/web/focus_test.go web/templates/` — if any in-scope file changed since this plan was written, compare the "Current state" excerpts to the live code; on mismatch, STOP.

## Status
- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: tech-debt/architecture
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
Five `*FocusHTML` Go functions in `internal/web` (≈170 lines) and the six `html/template`
focus-body files they execute have NO live caller — the live focus path goes through
`focus.go:focusBodyHTML → cards.go:cardFocusHTML → cardSizeInto(ui.Focus)`, which renders the
feature-owned gomponents focus renderers (`journalcards.DayFocus`, `taskcards.QuestsFocus`,
`lifecards.LifelogFocus`, `knowledgecards.KnowledgeFocus`, `settingscards.SettingsFocus`).
The only thing keeping the dead Go + dead templates alive is four pinning tests in
`templates_test.go` that still `ExecuteTemplate` the legacy defines. Separately, `GET /boards`
still renders its OWN full `<html>` document through the legacy `layout.html` topbar (the OLD
Boards/Settings/Engine-room nav), while every other surface renders through `shell.Page`'s
domain rail (`internal/ui/shell/shell.go`) — which has no "Boards" link at all. Boards is no
longer top-level nav. Retiring the legacy page shell and the dead focus templates removes a
whole parallel shell and ≈300 lines of fossil, leaving ONE page shell and ONE focus path.

## Current state

### The 5 orphaned focus-body funcs (confirmed zero live callers — grep returns only their own defs + comments)
- `internal/web/settings.go:27` `func (h *handlers) settingsFocusHTML(params map[string]string) template.HTML` — executes `settings_body`. Comment at `settings.go:21-26` already labels it "Dead since plan 05". Its only collaborator `settingsData` (`settings.go:15-19`) and `h.modelsData()` (`models.go:134-140`, comment "Dead since plan 05 — called only by settingsFocusHTML") die with it.
- `internal/web/tasks.go:155` `func (h *handlers) questsFocusHTML() template.HTML` — executes `tasks_list`. Comment at `tasks.go:150-154` labels it dead.
- `internal/web/day.go:121` `func (h *handlers) dayFocusHTML(params map[string]string) template.HTML` — executes `day_focus`.
- `internal/web/life.go:80` `func (h *handlers) lifelogFocusHTML() template.HTML` — executes `life_body`.
- `internal/web/knowledge.go:62` `func (h *handlers) knowledgeFocusHTML(kind knowledge.Kind) template.HTML` — executes `knowledge_body`.

The live focus path (keep, do NOT touch behavior):
```
internal/web/focus.go:80   func (h *handlers) focusBodyHTML(typ string, params map[string]string) template.HTML {
internal/web/focus.go:81       return h.cardFocusHTML(typ, params)
internal/web/cards.go:101  func (h *handlers) cardFocusHTML(typ string, params map[string]string) template.HTML {
internal/web/cards.go:110      if err := h.cardSizeInto(&b, typ, cleaned, ui.Focus); err != nil {
```

### The 4 pinning tests (line numbers confirmed; drifted from the SPEC's leads)
- `internal/web/templates_test.go:45` `TestModelsPageAndCleanChatbarRender` — the **only** part to change is lines `106-114`: it builds `settingsData{Section:"models", Models: models}` and `ExecuteTemplate(&b,"settings_body", settingsModels)` asserting `MODELS-PANEL-MARKER`. (The chatbar/composer parts above stay.)
- `internal/web/templates_test.go:121` `TestQuestsFocusListRenders` — `ExecuteTemplate(&b,"tasks_list", map[string]any{"QuestLog": ql})` (line 126), asserts `id="quest-rail"`, `id="quest-detail"`.
- `internal/web/templates_test.go:139` `TestLifeBodyRenders` — `ExecuteTemplate(&b,"life_body", data)` (line 154) + empty case (line 165), asserts `weight`, `82.5`, `polyline`, `gratitude`, `streak 5`, `life-grid`, and empty → `yours to invent`.
- `internal/web/templates_test.go:176` `TestDayPageRenders` — `ExecuteTemplate(&b,"day_focus", data)` (line 189) + today case (line 206).

### The gomponents focus renderers the tests will repoint to (confirmed exported names/signatures)
- `taskcards.BuildQuestsFocus(app core.App) QuestsFocusView` (`internal/feature/taskcards/questsfocus.go:33`) and `taskcards.QuestsFocus(v QuestsFocusView) g.Node` (`questsfocus.go:171`); rail-only `taskcards.QuestRail(v) g.Node` (`questsfocus.go:109`). NOTE: a test app is needed for `Build*` (they take `core.App`); the existing render tests use `tests.ApiScenario` (see `boards_test.go`/`focus_test.go`), not the template-only `parseTemplates` helper. See Test plan for the pattern.
- `journalcards.BuildDayFocus(app core.App, params map[string]string) DayFocusView` (`internal/feature/journalcards/dayfocus.go:42`) and `journalcards.DayFocus(v DayFocusView) g.Node` (`dayfocus.go:176`).
- `lifecards.LifelogFocus(v LifelogFocusView) g.Node` (`internal/feature/lifecards/lifelogfocus.go:86`); `buildLifelogFocus` is unexported, so build a `LifelogFocusView` literal in the test (it is a plain struct, `lifelogfocus.go:31-34`, with its doc comment at 29-30).
- `knowledgecards.KnowledgeFocus(v KnowledgeFocusView) g.Node` (`internal/feature/knowledgecards/knowledgefocus.go:58`).
- `settingscards.BuildSettingsFocus(app core.App, params map[string]string) (SettingsFocusView, error)` (`internal/feature/settingscards/settingsfocus.go:142`) and `settingscards.SettingsFocus(v SettingsFocusView) g.Node` (`settingsfocus.go:289`).

These all already have storybook stories + render coverage via `TestUiCardAllTypesRender` and `TestAllStoriesRender`, so the focus output is ALREADY tested by the live path. The cleanest reconciliation is to make the named legacy tests assert the live gomponents output (or, where a `Build*` needs an app, fold the assertion into the existing `focus_test.go` ApiScenario coverage — see Test plan).

### The /boards page shell (the central decision)
`web.go:237-238` serves both `/boards` routes live:
```
internal/web/web.go:237   se.Router.GET("/boards", h.boardsIndex)
internal/web/web.go:238   se.Router.GET("/boards/{id}", h.boardsPage)
```
`boardsPage` renders its own full document on a normal load:
```
internal/web/boards.go:356   return h.render(e, "boards.html", boardPageData{
internal/web/boards.go:357       boardView: bv,
internal/web/boards.go:358       Title:     current.Name + " · Balaur",
```
`web/templates/boards.html:1-20` is a hand-rolled `<!DOCTYPE html>…` document that pulls the legacy `layout.html` defines `page_head` (boards.html:4) and `topbar` (boards.html:9). That `topbar` is the OLD nav (`web/templates/layout.html:21-38`): `<a href="/boards">Boards</a>`, `Settings`, `Engine room`. The LIVE nav is `shell.Topbar` (`internal/ui/shell/shell.go:74-100`) — a domain rail (Quests / Knowledge / Life / Journal / Heads / Settings) with NO Boards link. Home (`home.go:66`) and the focus page (`focus.go:168`) both render via `shell.Page`. `focus_test.go:TestFocusFullLoad` asserts the focus full-load topbar is the gomponents shell (`class="topbar"` + `aria-current="page"`) and `NotExpectedContent: {">Boards<","Engine room"}` — i.e. the legacy topbar is already retired everywhere except `/boards`.

**Boards is NOT reachable from the live nav.** The only inbound links to `/boards*` are: typing the URL, and the never-rendered `focusBackHref` (`focus.go:70`, returns `/boards/…`; the `BackHref` field is set into `focusView` but NOT emitted by `focus_main` — `focus_test.go` NotExpectedContent confirms `focus-back`/`← Back` are absent).

### The coupling that constrains the decision (CRITICAL — read before deciding)
The board WRITE endpoints (which the SPEC says MUST keep working) render template defines that live INSIDE `boards.html`:
- `boardsRename` (`boards.go:434`) renders `board_header`.
- `boardsCardAdd` (`boards.go:537`) and `boardsCardRemove` (`boards.go:599`) render `board_grid`.
- `boardsPage`'s Datastar branch (`boards.go:335`) renders `board_main`.
`boards_test.go` exercises both the page (`TestBoardsDefaultsCreated`, `TestBoardsPageRenders`) AND the write endpoints (`TestBoardsCreate`, `TestBoardsRename`, `TestBoardsCardAddValid`, `TestBoardsCardRemove`, `TestBoardsLayoutHappyPath`, …). `board_grid`/`board_header`/`board_add`/`board_main` are defined ONLY in `boards.html` (grep). So `boards.html` CANNOT be deleted wholesale without breaking the kept write endpoints + their passing tests.

Consequence: this plan **retires the PAGE shell only** — it does NOT delete `boards.html` and does NOT delete `layout.html`. See "The decision" below.

### Template define → file map (confirmed)
| define | file | live consumers after this plan |
| `settings_body` | `web/templates/settings-focus.html` (sole define) | none → DELETE FILE |
| `tasks_list` + `quest_rail` | `web/templates/quests-focus.html` (`quest_rail` used only by `tasks_list`) | none → DELETE FILE |
| `day_focus` + `day_journal` | `web/templates/day-focus.html` (`day_journal` used only by `day_focus`) | none → DELETE FILE |
| `life_body` | `web/templates/lifelog-focus.html` (sole define) | none → DELETE FILE |
| `knowledge_body` | `web/templates/knowledge-focus.html` (sole define) | none → DELETE FILE |
| `board_main`,`board_header`,`board_grid`,`board_add` | `web/templates/boards.html` (+ a full `<html>` doc) | write endpoints render ONLY `board_header` (`boards.go:434`) and `board_grid` (`boards.go:537`,`599`) → KEEP those two defines. `board_main` is rendered only by `boardsPage`'s Datastar branch (`boards.go:335`, removed in Step 5) → DELETE define. `board_add` is referenced ONLY by `board_main` (`boards.html:43`) → it is ORPHANED once `board_main` goes; keeping it is harmless dead template content (a follow-up cleanup), do NOT claim a write endpoint renders it |
| `page_head`,`topbar`,`shell_open`,`shell_close` | `web/templates/layout.html` | still pulled by boards.html + focus.html's `focus_page` → KEEP file |

### Design constraints to honor (DESIGN.md / AGENTS.md)
- "Gateways adapt; they never re-implement." One page shell is the goal — but only collapse it where it is genuinely orphaned. Do NOT introduce a second shell.
- Do not change the rendered focus output for the owner: the gomponents focus path stays byte-identical (it is already the live path).
- Surgical changes: every deleted line must trace to "dead, zero live callers". Pre-existing dead code that is OUT of this plan's targets stays.

## Commands you will need
| Purpose | Command | Expected |
| Drift baseline | `git diff --stat 12a2ff5..HEAD -- <in-scope paths>` | empty (no drift) |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Web tests | `go test ./internal/web/...` | ok |
| Storybook render | `go test ./internal/feature/storybook/...` | ok |
| Format | `gofmt -l internal/web/*.go internal/web/*_test.go` | empty output |
| Whitespace | `git diff --check` | no output |
| Dead-func grep | `grep -rn "settingsFocusHTML\|questsFocusHTML\|dayFocusHTML\|lifelogFocusHTML\|knowledgeFocusHTML\|modelsData" internal/web/` | only comment-free zero matches after deletion (see Done) |
| Dead-define grep | `grep -rn '"settings_body"\|"tasks_list"\|"day_focus"\|"life_body"\|"knowledge_body"' internal/web/ web/templates/` | empty after template deletion |

If `go` commands fail with "certificate signed by unknown authority" you are in the Hyperagent sandbox — apply the GOPROXY shim per `docs/hyperagent-sandbox.md` (GOSUMDB stays on). Otherwise proceed normally.

## Scope
**In scope** (only files you may modify):
- `internal/web/settings.go` — delete `settingsFocusHTML` + `settingsData`.
- `internal/web/models.go` — delete `modelsData` (dead with `settingsFocusHTML`).
- `internal/web/tasks.go` — delete `questsFocusHTML`.
- `internal/web/day.go` — delete `dayFocusHTML`.
- `internal/web/life.go` — delete `lifelogFocusHTML`.
- `internal/web/knowledge.go` — delete `knowledgeFocusHTML`.
- `internal/web/templates_test.go` — repoint the 4 pinning tests off the legacy defines.
- `internal/web/web.go` — change `GET /boards` and `GET /boards/{id}` to redirect to `/`.
- `internal/web/boards.go` — replace the `boardsIndex`/`boardsPage` page-render path with a redirect; KEEP all write-endpoint handlers and helpers.
- `internal/web/boards_test.go` — update ALL FOUR GET-page render tests (`TestBoardsDefaultsCreated`, `TestBoardsPageRenders`, `TestBoardsLegacyFlowRender` at line 452, `TestBoardsFreeLayoutRender` at line 475) for the new redirect behavior; KEEP the write-endpoint tests.
- `web/templates/settings-focus.html`, `web/templates/quests-focus.html`, `web/templates/day-focus.html`, `web/templates/lifelog-focus.html`, `web/templates/knowledge-focus.html` — DELETE (proven orphaned).
- `web/templates/boards.html` — DELETE the full `<html>` document wrapper (lines 1-20) AND the `board_main` define (orphaned once Step 5 drops its only render call). KEEP `board_header` and `board_grid` (rendered by the live write endpoints). `board_add` is referenced only by `board_main`, so it becomes orphaned — keeping it is harmless dead template content (follow-up cleanup), it is NOT rendered by any write endpoint.

**Out of scope** (do NOT touch, reason per item):
- The `/ui/boards/*` write endpoints and the card registry (`cards.go`, `cards` package) — SPEC says keep working; the board defines they render stay.
- `web/templates/layout.html` — still pulled in by the kept boards.html defines' callers and by `focus.html`'s `focus_page` define; deleting it is a SEPARATE decision (see Maintenance notes). Removing it now would break template parse.
- `web/templates/focus.html` (`focus_main`/`focus_page`) — `focus_main` is the LIVE focus inner; leave both.
- `focusBackHref` / `safeBoardID` / the `BackHref` field in `focus.go`, and `focus_test.go:TestFocusBackHrefRejectsUnsafe` — pre-existing, already-tested code; the URL it builds (`/boards/…`) is now a redirect target, which is harmless. Do NOT remove (it would break the test). Note it under Maintenance.
- `internal/web/assets/static/board.js` — referenced only by the deleted boards.html `<head>`; it becomes an orphaned static asset. Leave it (deleting embedded assets is a separate cleanup); note under Maintenance.
- The `chat.Dock` gomponents port — that is plan 084.
- Live focus rendering behavior — must stay byte-identical (it is already the live path).

## Git workflow
Branch `improve/083-retire-legacy-focus-templates`. Conventional commits, e.g.
`refactor(web): repoint focus tests to gomponents renderers`,
`chore(web): delete orphaned focus-body funcs + templates`,
`refactor(web): redirect retired /boards page to home`. Do NOT push or open a PR unless told.

## Steps

### Step 1: Confirm zero drift and zero live callers
Run the drift check from the header. Then prove the 5 funcs + `modelsData` have no live caller:
```
grep -rn "settingsFocusHTML\|questsFocusHTML\|dayFocusHTML\|lifelogFocusHTML\|knowledgeFocusHTML" internal/web/ web/
grep -rn "h.modelsData\|\.modelsData(" internal/web/
```
Every hit must be either the function's own `func` line or a comment/doc line (no call site). If ANY hit is a real call (e.g. `h.dayFocusHTML(...)` outside a comment), STOP — the func is live; report it.
**Verify**: the only `func (h *handlers) <name>FocusHTML` definitions appear, plus comments. No `h.<name>FocusHTML(` call sites. → proceed.

### Step 2: Repoint the 4 pinning tests to the live gomponents output (build stays green)
Edit `internal/web/templates_test.go`. The legacy `ExecuteTemplate` calls must stop referencing the soon-deleted defines. Replace each as follows.

(a) `TestModelsPageAndCleanChatbarRender` — replace ONLY the trailing `settings_body` block (lines ~104-114, from the `// The settings shell…` comment through the `MODELS-PANEL-MARKER` assertion) with an assertion against the gomponents settings focus. Use an app-backed render via `settingscards.SettingsFocus(settingscards.BuildSettingsFocus(...))`. Because `BuildSettingsFocus` needs a `core.App`, prefer to DELETE this trailing block here and let the settings models focus be covered by the existing focus ApiScenario (the models panel render is already covered by `modelcards` story render + `settingscards` tests). Keep the chatbar/composer assertions (lines 45-103) untouched. If you delete the block, also remove the now-unused `models`/`settingsModels`/`modelsPageData` locals so the test compiles.

(b) `TestQuestsFocusListRenders` (line 121) — replace the body so it renders the gomponents quests focus rail rather than `tasks_list`. The rail render `taskcards.QuestRail(taskcards.BuildQuestsFocus(app))` needs an app, so convert this test to an `ApiScenario` against `GET /focus/quests` (the live path) asserting the SAME markers (`id="quest-rail"`, `id="quest-detail"`). Model it on `focus_test.go:TestFocusFullLoad` (uses `tests.ApiScenario` + `newWebApp`).

(c) `TestLifeBodyRenders` (line 139) — replace `ExecuteTemplate(life_body)` with a direct gomponents render: build a `lifecards.LifelogFocusView` literal (it is exported, plain struct) and render `lifecards.LifelogFocus(v).Render(&b)`; assert the same content markers that still exist in the gomponents output (`weight`, `82.5`, `polyline`, `gratitude`, `streak`, and the grid class). VERIFY the exact class names/markers by reading `internal/feature/lifecards/lifelogfocus.go` before writing assertions — the legacy `life-grid` / `yours to invent` strings may differ in the gomponents port; assert what the port actually emits, not the legacy strings.

(d) `TestDayPageRenders` (line 176) — replace `ExecuteTemplate(day_focus)` with `journalcards.DayFocus(...)`. `BuildDayFocus` needs an app, so convert this to an `ApiScenario` against `GET /focus/day?date=2026-06-09` (the live path), asserting the day sections the gomponents `DayFocus` emits. Read `internal/feature/journalcards/dayfocus.go` to confirm the actual markers (the prev/next deep-link form `/focus/day?date=…`, the "transcript" expander, the empty-state strings) before writing assertions.

Whichever conversion you choose for (a)/(b)/(d), the simplest correct option is to fold the assertion into an `ApiScenario` mirroring `focus_test.go`. Add the imports the new code needs (`journalcards`, `taskcards`, `lifecards`, `tests`, `_ "…/migrations"`) and drop imports that become unused (e.g. `html/template`, `turn`) — `gofmt`/`go build` will flag leftovers.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./internal/web/...` → all pass (tests now assert the live gomponents output, with the legacy defines STILL present, so this step is independently green).

### Step 3: Delete the 5 orphaned focus funcs + the dead `modelsData`/`settingsData`
In each file remove only the named symbol(s) and their doc comments:
- `settings.go`: delete `settingsData` (15-19) and `settingsFocusHTML` (21-50). If the file's only remaining `import` becomes unused (`html/template`, `strings`), remove it.
- `models.go`: delete `modelsData` (130-140). Keep `renderModelsPanel`/`modelsPanel`/`patchChatbar`.
- `tasks.go`: delete `questsFocusHTML` (150-166). Keep `buildQuestLog`/`loadQuestLogRecs` (still used by the gomponents path? — VERIFY with grep; if `buildQuestLog` has no remaining caller after the func is gone, it is orphaned too — but only delete it if grep proves zero callers, else leave and note).
- `day.go`: delete `dayFocusHTML` (118-140). Keep `buildDay` only if still referenced; grep first.
- `life.go`: delete `lifelogFocusHTML` (77-91). Keep `lifeOverview` only if still referenced; grep first.
- `knowledge.go`: delete `knowledgeFocusHTML` (58-75). Keep `skillsData`/`memoryData` only if still referenced; grep first.

For each "Keep only if referenced" helper, run `grep -rn "<helper>" internal/web/` — if the ONLY remaining references are inside the func you just deleted, delete the helper too in the same step; otherwise leave it. Do NOT delete a helper that any live code still calls.

NOTE (orphan cascade): deleting the 5 focus funcs leaves these helpers/types with no remaining live caller (confirmed by grep at `12a2ff5` — each is called ONLY from inside a func deleted in this plan, or from the test repointed in Step 2): `buildProfileData` (`profile.go:19`, sole caller was `settingsFocusHTML` at `settings.go:42`), `lifeOverview` + `numericValues` (`life.go`), `buildDay` (`day.go`), `loadQuestLogRecs` + `buildQuestLog` (`tasks.go` — `buildQuestLog`'s only non-deleted caller is `templates_test.go:124`, which Step 2 repoints), `skillsData` + `memoryData` + `memoryCategories` (`knowledge.go`). Go does NOT flag unused methods, package-level funcs, types, or vars as build errors — only unused IMPORTS and unused LOCAL vars break the build. So these orphans will NOT trip `go build`/`go vet`; per the surgical-changes rule, you may delete a helper in the same step IF grep proves it has zero callers, but leaving it as harmless dead code is also acceptable — do not hunt for a compiler error that won't appear.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 (an unused import or now-unreferenced helper that the compiler flags must be cleaned up here); `go vet ./...` → exit 0; `go test ./internal/web/...` → all pass.

### Step 4: Delete the 5 now-orphaned focus-body templates
Confirm zero references FIRST, then delete:
```
grep -rn '"settings_body"\|"tasks_list"\|"quest_rail"\|"day_focus"\|"day_journal"\|"life_body"\|"knowledge_body"' internal/web/ web/templates/
```
Every remaining hit must be inside the to-be-deleted files themselves (e.g. `quest_rail` referenced by `tasks_list` inside `quests-focus.html`; `day_journal` referenced by `day_focus` inside `day-focus.html`). If a define is referenced from OUTSIDE its own file, STOP — it is still live.
Then `git rm` the five files:
`web/templates/settings-focus.html`, `web/templates/quests-focus.html`, `web/templates/day-focus.html`, `web/templates/lifelog-focus.html`, `web/templates/knowledge-focus.html`.
**Verify**: `go test ./internal/web/...` → all pass (`TestTemplatesParse` re-parses `templates/*.html`; a dangling `{{template "x"}}` to a now-missing define fails parse, so a green run proves nothing else referenced them). `grep -rn '"settings_body"\|"tasks_list"\|"day_focus"\|"life_body"\|"knowledge_body"' internal/web/ web/templates/` → empty.

### Step 5: Retire the /boards PAGE shell (redirect both routes to `/`)
This is the central decision — **DECISION: retire the legacy /boards PAGE (not nav-reachable, renders a second shell), keep all `/ui/boards/*` write endpoints and the board card registry.** Implement:

1. In `web.go:237-238`, leave the routes registered but point the page handlers at a redirect. Simplest: change `boardsIndex`/`boardsPage` to redirect.
   - `boards.go:boardsIndex` (272-282): replace the body with `return e.Redirect(http.StatusFound, "/")`. (Drop the now-unused `ensureDefaultBoards`/`loadBoards` calls inside it — but KEEP the functions; the write endpoints use `loadBoards`, and `ensureDefaultBoards` is used by `boardsPage` and tests. VERIFY callers with grep before removing any call.)
   - `boards.go:boardsPage` (296-361): replace the FULL-LOAD render branch (the `dock, err := h.dockData()` block at 352-360 that calls `h.render(e,"boards.html",…)`) with `return e.Redirect(http.StatusFound, "/")`. KEEP the Datastar `@get` branch IF and only if a live caller still issues a Datastar `@get` to `/boards/{id}` — board-tab switching came from the deleted boards.html nav, so after the page is gone NOTHING issues that fetch. Therefore replace the WHOLE `boardsPage` body with a single redirect to `/` (both ds and non-ds). This drops the `board_main` render at 335 — that define becomes orphaned (handle in Step 6).
2. Delete the full `<html>` document wrapper in `web/templates/boards.html` (lines 1-20, from `<!DOCTYPE html>` through `</html>`). KEEP everything from `{{define "board_main"}}` onward — wait: `board_main` is now orphaned (only `boardsPage`'s deleted Datastar branch rendered it). Remove the `board_main` define too. KEEP `board_header` and `board_grid` — these ARE rendered by the live write endpoints (`boards.go:434` renders `board_header`; `boards.go:537`/`599` render `board_grid`). `board_add` is referenced ONLY by `board_main` (`boards.html:43`), so once `board_main` is removed `board_add` has zero callers — it is NOT rendered by any write endpoint. Keeping `board_add` is harmless (`ParseFS` accepts a define with no caller, no parse/build break); leave it as orphaned dead template content noted for a follow-up cleanup, and do NOT try to "verify board_add is rendered by a write endpoint" (it is not — that check would falsely trip a STOP). After this edit `boards.html` is a defines-only partial file (no top-level `<html>`), which `ParseFS` handles fine (it is template content, not a standalone page).

Update `internal/web/boards_test.go`:
- `TestBoardsDefaultsCreated` (line 18) and the second-GET case: `GET /boards` now `302`-redirects to `/` and does NOT seed (seeding moved out of the redirect). Change `ExpectedStatus` to `302` (or whatever `http.StatusFound` resolves to in the scenario) and assert the `Location: /` redirect; drop the "seeds 4 boards" assertion. If seeding-on-first-visit is still desired, note that defaults now seed lazily from the FIRST write endpoint that calls `loadBoards`/`ensureDefaultBoards` — confirm which write endpoints seed, and if NONE do, leave a Maintenance note that default-board seeding is no longer auto-triggered (acceptable: boards is retired from nav).
- `TestBoardsPageRenders` (line 66): `GET /boards/{id}` now redirects to `/`. Change to assert the redirect (status + `Location: /`), drop the `board-grid`/`ucard-today` body assertions.
- `TestBoardsLegacyFlowRender` (line 452) and `TestBoardsFreeLayoutRender` (line 475): these ALSO hit `GET /boards/{id}` and currently set `ExpectedStatus: 200` with `ExpectedContent: ["board-grid"]` / `["board-grid-free","grid-row"]` — they exercise the page-render branch that Step 5 replaces with a redirect, so they WILL FAIL unless converted. Change each `ExpectedStatus` to `302` (`http.StatusFound`) and assert the `Location: /` redirect; drop the `board-grid`/`board-grid-free`/`grid-row` body assertions. (Their board-seeding setup can stay or be trimmed — the redirect does not read the board.)
- KEEP all write-endpoint tests (`TestBoardsCreate`, `TestBoardsRename`, `TestBoardsCardAddValid`, `TestBoardsCardRemove`, `TestBoardsCardRemoveOutOfBounds`, `TestBoardsDeleteLastRefused`, `TestBoardsDeleteWithMultiple`, `TestBoardsLayout*`) — they exercise the kept handlers + the kept `board_header`/`board_grid` defines and must stay green.

**STOP-AND-FLAG**: Because the write endpoints AND a `boards_test.go` page suite both depend on board state, this redirect changes default-seeding timing. If after this step ANY write-endpoint test fails because defaults are no longer seeded, do NOT delete seeding — instead keep `ensureDefaultBoards` being called where those tests expect it, and report the residual ambiguity (retired page vs. live write endpoints) for human confirmation rather than improvising a deeper boards refactor.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./internal/web/...` → all pass; manual: with the app running, `curl -s -o /dev/null -w "%{http_code} %{redirect_url}\n" http://127.0.0.1:8090/boards` → `302 http://127.0.0.1:8090/` (or `/`).

### Step 6: Clean up symbols orphaned by Step 5
Grep for now-unreferenced board symbols and remove only those with zero live callers:
```
grep -rn '"board_main"' internal/web/ web/templates/   # must be empty after removing the define + its render call
```
Also confirm `boardPageData` (type) and `h.render` (`web.go:254`) — after Step 5 their only callers (`boards.go:335`/`356`) are gone, so both are now dead. NOTE: Go does NOT report an unused type or unused method as a build error, so the compiler will NOT flag `boardPageData` or `h.render` — do not hunt for a build failure that won't appear. Use grep to confirm zero callers, then either delete them (surgical, since they are genuinely dead) or leave them as harmless dead code and note it; only unused IMPORTS will break the build. If the compiler flags an unused import in `boards.go` (e.g. `datastar` if the Datastar branch is fully gone — CHECK: the write endpoints still use `datastar`, so it likely stays), clean it.
**Verify**: `go vet ./...` → exit 0; `go test ./...` → all pass; `grep -rn '"board_main"' web/templates/ internal/web/` → empty.

### Step 7: Full validation + format
**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `go test ./internal/feature/storybook/...` → ok (no story touched, but proves the focus renderers still render)
- `gofmt -l internal/web/*.go internal/web/*_test.go` → empty
- `git diff --check` → no output

## Test plan
- No NEW component → no new storybook story (the gomponents focus renderers already have stories + render coverage via `TestUiCardAllTypesRender` and `TestAllStoriesRender`).
- The 4 repointed tests (`templates_test.go`) now assert the LIVE gomponents focus output instead of the dead legacy defines. The ApiScenario pattern to copy is `internal/web/focus_test.go:TestFocusFullLoad` (`tests.ApiScenario` + `TestAppFactory: newWebApp`, `ExpectedContent`/`NotExpectedContent`). For pure-struct renderers (`lifecards.LifelogFocus`) a direct `node.Render(&b)` against a hand-built view (no app) is simpler — mirror `TestSparkPointsScaling`'s direct-call style.
- The 2 boards PAGE tests (`boards_test.go`) become redirect assertions; the write-endpoint tests are unchanged and must stay green.
- Verification command for the whole change: `go test ./...`.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` all pass.
- [ ] `go test ./internal/feature/storybook/...` ok.
- [ ] `grep -rn "settingsFocusHTML\|questsFocusHTML\|dayFocusHTML\|lifelogFocusHTML\|knowledgeFocusHTML\|h.modelsData" internal/web/` returns no function-definition or call hits (only stray comments removed too — ideally empty).
- [ ] `grep -rn '"settings_body"\|"tasks_list"\|"day_focus"\|"life_body"\|"knowledge_body"' internal/web/ web/templates/` → empty.
- [ ] `grep -rn '"board_main"' internal/web/ web/templates/` → empty.
- [ ] The 5 template files (`settings-focus.html`, `quests-focus.html`, `day-focus.html`, `lifelog-focus.html`, `knowledge-focus.html`) no longer exist.
- [ ] `boards.html` has no `<!DOCTYPE>`/`<html>` wrapper; `board_header`/`board_grid`/`board_add` defines remain.
- [ ] `gofmt -l internal/web/*.go internal/web/*_test.go` empty; `git diff --check` no output.
- [ ] Only in-scope files changed (`git status` shows nothing outside the Scope list, except the `plans/readme.md` index row).
- [ ] The 083 row in plans/readme.md is updated (add the row if it is not present yet, matching the existing column format).
- [ ] VISUAL (both modes): with the app running, `GET /boards` → 302 to `/`; open `/focus/quests`, `/focus/day`, `/focus/lifelog`, `/focus/memory`, `/focus/settings?section=models` in BOTH `theme-hearthwood dark` and `theme-hearthwood light` (force via `document.documentElement.className='theme-hearthwood dark'`/`light`) and confirm each focus body renders identically to before this change (it uses the unchanged gomponents path). Check `<=920px` width still lays out.

## STOP conditions
- Drift: the header `git diff --stat 12a2ff5..HEAD` shows an in-scope file changed and its live code no longer matches a "Current state" excerpt → STOP, report the drift.
- A "dead" focus func or `modelsData` turns out to have a live caller (Step 1 grep shows a real call site) → do NOT delete it; report which symbol and where.
- Deleting a template breaks parse because a still-live define (outside the deleted file) referenced it (Step 4 `TestTemplatesParse`/`go test` fails) → STOP, restore the file, report the unexpected reference.
- The boards decision proves genuinely two-sided in practice — i.e. retiring the page breaks a write-endpoint test via default-board seeding timing (Step 5 STOP-AND-FLAG) → implement the redirect, KEEP seeding where the tests expect it, and flag the residual ambiguity for human confirmation; do NOT refactor the boards write path further in this plan.
- Any Verify command fails twice after a fix attempt → STOP and report the command + output.
- A change requires editing a file outside the Scope list (e.g. `layout.html`, `focus.html`, `board.js`, a `cards`/feature package) → STOP; that is a sign the decomposition is wrong or a follow-up plan is needed.

## Maintenance notes
- **layout.html is intentionally NOT deleted.** After this plan it is reachable only via `boards.html` (the kept defines do not pull `topbar`/`page_head` anymore once the `<html>` wrapper is gone — VERIFY) and via `focus.html`'s never-rendered `focus_page` define. A follow-up plan can: (a) delete the unused `focus_page` define from `focus.html`, then (b) once `boards.html` no longer references `page_head`/`topbar`, delete `layout.html` and migrate the board write-endpoint defines (`board_header`/`board_grid`/`board_add`) to gomponents — collapsing to the single `shell.Page`. That is a larger, separate change; do NOT attempt it here.
- **`board.js`** (`internal/web/assets/static/board.js`) is now an orphaned static asset (only the deleted boards.html `<head>` referenced it). Removing embedded assets is a separate cleanup; left in place deliberately.
- **`focusBackHref`/`safeBoardID`/`BackHref`** in `focus.go` now build `/boards/…` URLs that 302 to `/`. They are pre-existing, still tested (`focus_test.go:TestFocusBackHrefRejectsUnsafe`), and harmless. A future plan that fully retires boards should remove them together with their test.
- **Default-board seeding** (`ensureDefaultBoards`) no longer runs on a page visit (the page is gone). If the board write endpoints are ever surfaced again, seeding must be re-triggered explicitly. Documented as accepted (boards is retired from nav).
- A reviewer should scrutinize: (1) that NO focus output changed for the owner (the gomponents path is untouched); (2) that every kept board write endpoint still renders its define from the trimmed `boards.html`; (3) that the repointed tests assert the ACTUAL gomponents markers (read the renderers), not stale legacy strings.
- **Plan-082 coordination**: this plan redirects `/boards`. If plan 082 carries an "R3" item about the legacy `/boards` page shell, that item is now moot — note it when 082 is written/executed.
