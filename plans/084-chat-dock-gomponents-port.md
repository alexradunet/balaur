# Plan 084: Port the companion chat dock shell to a chat.Dock gomponents organism and retire the legacy chat_dock template

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 084 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/web/home.go internal/web/focus.go internal/web/web.go internal/web/models.go internal/web/heads.go internal/web/chatstream.go internal/web/chat.go web/templates/home.html internal/web/assets/static/basm.css internal/ui/chat internal/feature/storybook internal/self/knowledge.md` — if any in-scope file changed since this plan was written, compare the "Current state" excerpts to the live code; on mismatch, STOP and report.

## Status
- **Priority**: P1
- **Effort**: L
- **Risk**: MED-HIGH
- **Depends on**: plans/076-*.md (chat-width tokens `--w-chat-home`/`--w-chat-overlay` — NOTE: these tokens are ABSENT in basm.css at HEAD `12a2ff5`; 076 is a soft/historical dependency, so this plan defines them itself in Step 6 per the Canonical decisions), plans/080-*.md (chat organisms gain trailing variadic attrs); soft dep plans/085-*.md
- **Category**: architecture
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
The single most-used surface — the companion chat dock — is the LAST chat surface still injected as legacy `html/template` via `g.Raw`. `home.go` and `focus.go` both execute the `chat_dock` template into a string and hand it to `shell.Page`'s `Dock` slot. The dock's proportion logic (rail width, the 940px overlay column, the 1800px home column) lives as three hand-synced CSS blocks keyed on `#dock` / `.dock-full #dock` / `html.home #dock` instead of one variant-driven component — exactly the duplication that produced the 1800-vs-940 width drift plan 076 had to reconcile. While the dock is a template, the storybook (the project's source of truth per AGENTS.md) is NOT the source of truth for the home screen. This plan ports the dock *shell* and its switchers into a `chat.Dock` organism (composing the already-gomponents `chat.Message`, `chat.ToolRow`, `ui.Composer`), adds storybook stories, switches `home.go`/`focus.go` to render the node directly, and deletes the dead template — while preserving every live-stream SSE selector id byte-for-byte.

## Current state

### What is ALREADY gomponents (do NOT re-port — reuse)
The SPEC predates recent work; confirm by reading. The chat MESSAGES and the COMPOSER are already gomponents:
- `internal/web/chatstream.go:96-103,109-111,158-160,236-238` — the live stream renders `chat.Message{...}` and `chat.ToolRow{...}` (CSS classes `cmsg`, `cmsg-balaur`, `cmsg-user`, `msg msg-tool`), NOT the legacy `.msg` `chat-messages.html` fragments.
- `internal/web/recap.go:218-235` (`chatBodyHTML`) — page-load history + the hearth greeting render through `chat.Message` too (`h.renderMessages`). Stored in `homeData.ChatBodyHTML template.HTML` (models.go:40).
- `internal/web/home.go:28-43` (`composerNode`/`composerHTML`) — the live input is `ui.Composer{PostURL:"/ui/chat", ID:"chat-draft", ...}`, rendered in Go and stored in `homeData.ComposerHTML` (models.go:39). The `chat_dock` template merely emits `{{.ComposerHTML}}` and `{{.ChatBodyHTML}}` into the shell.

### What is STILL legacy template (the actual port target) — `web/templates/home.html`
The whole file is dock chrome. `chat_dock` (home.html:6-33) renders:
```
<div class="dock-grip" title="drag to resize" aria-hidden="true"></div>
<header class="dock-head"><button class="dock-btn dock-full-btn" type="button" onclick="basmToggleDockFull()" ...></button></header>
{{if .HasRecap}}<div id="recap" class="recap-zone" data-on:intersect__once="@get('/ui/recap/bands')"><p class="recap-hint">◇ further back…</p></div>{{end}}
<div id="dock-convo">{{template "dock_convo" .}}</div>
<div id="nudge-poll" data-signals:nudgeSince="{{.NowMillis}}" data-signals:dockMaster="true" data-signals:streaming="false" data-on:interval__duration.30s="$dockMaster && @get('/ui/chat/nudges?since='+$nudgeSince)"></div>
{{.ComposerHTML}}
<dialog id="model-modal" aria-labelledby="model-modal-title"></dialog>
```
- `dock_convo` (home.html:40-42): `<section class="chat" id="chat" aria-live="polite">{{.ChatBodyHTML}}</section>`.
- `chat_bar` (home.html:44-50): `<div class="chatbar chatbar-slim" id="chatbar" {{if not .ChatReady}}data-on:interval__duration.2s="@get('/ui/chatbar')"{{end}}>` then `{{template "head_switcher" .}}{{template "model_switcher" .}}`.
- `model_switcher` (home.html:52-73): `<section class="model-switcher" aria-label="Model">` — kicker, `.model-current`, manage link, not-ready block, `.chatbar-profile-link` with the soul avatar.
- `head_switcher` (home.html:78-98): `<section class="head-switcher" id="head-switcher" aria-label="Head">` — kicker, `.head-switcher-current`, and a `{{range .HeadChoices}}` of `<form data-on:submit__prevent="@post('/ui/heads/active', {contentType:'form'})">` buttons gated by `data-attr:disabled="$streaming"`.

NOTE: `chat_bar` is currently NOT referenced by `chat_dock` (the dock no longer embeds the chatbar — read home.html: `chat_dock` does not `{{template "chat_bar"}}`). `chat_bar` is still used as a standalone SSE patch target by `patchChatbar` (models.go:111-128, patches `#chatbar`) and `chatbar` (models.go:95-105). `head_switcher` is patched standalone by `setActiveHead` (heads.go:30-33, patches `#head-switcher`). So `chat_bar`/`model_switcher`/`head_switcher` are LIVE SSE fragments even though the dock chrome no longer mounts them. CONFIRM whether `#chatbar` still appears in any rendered page before assuming you can drop it (grep: `grep -rn 'chatbar\|chat_bar' internal/web web/templates`). If `#chatbar` is mounted nowhere on a page, the 2s poll has no DOM anchor — that is a pre-existing question, NOT this plan's to resolve; preserve current behavior exactly.

### The three width CSS blocks to collapse (basm.css; total file 3218 lines)
- Rail (default `#dock`): `basm.css:2364-2410` — `#dock { position:fixed; ... width: var(--sidebar-w); ... }`, `#dock .chat { padding:12px 12px 6px; --portrait-size:40px; --chat-gutter:54px; }`.
- Overlay (`.dock-full #dock`): `basm.css:2441-2454` —
```
.dock-full #dock { left: 0; width: auto; z-index: 50; border-left: 0; box-shadow: none; }
.dock-full #dock .chat, .dock-full #dock .msg-draft, .dock-full #dock .chatbar, .dock-full #dock .dock-head, .dock-full #dock .recap-zone, .dock-full #dock .recap-band { width: auto; max-width: 940px; margin-left: auto; margin-right: auto; }
.dock-full #dock .chat { --portrait-size: 56px; --chat-gutter: 84px; padding: 20px 24px 8px; }
```
- Home (`html.home #dock`): `basm.css:3170-3218` (end of file) —
```
html.home #dock { left: 0; width: auto; z-index: 50; border-left: 0; box-shadow: none; }
html.home #dock .chat, html.home #dock .msg-draft, html.home #dock .chatbar, html.home #dock .recap-zone, html.home #dock .recap-band { width: 100%; max-width: 1800px; margin-left: auto; margin-right: auto; }
html.home #dock .chat { --portrait-size: 56px; --chat-gutter: 84px; padding: 20px 24px 8px; }
...
#dock-convo { flex: 1 1 auto; min-height: 0; display: flex; flex-direction: column; }
#dock .composer { flex: 0 0 auto; margin: 8px 10px 10px; }
html.home #dock .composer { width: 100%; max-width: 1800px; margin-left: auto; margin-right: auto; }
```
GUARDRAIL: the 1800px home column is a DELIBERATE recent decision (commit 12a2ff5). Do NOT shrink it. Tokenize the three widths as `--w-chat-home:1800px`, `--w-chat-overlay:940px`, rail = `var(--sidebar-w)` (plan 076 should have added the first two to the `:root` layout block; if absent, add them per the Canonical decisions). The overlay/home divergence is intentional and must stay commented.

### The mount point in shell
- `internal/ui/shell/shell.go:44` — `h.Aside(h.ID("dock"), p.Dock)`. `PageProps.Dock g.Node` (shell.go:25). Home passes `HTMLClass:"home"` (home.go:68); focus passes no class (focus.go:168-173). The `home`/`dock-full` class on `<html>` is what currently selects the width block — preserve that selection mechanism (the no-flash script reads `basm-dock-full` and `basm-theme`; shell.go:14).

### Datastar contract for atoms/organisms (the conventions to match)
- gomponents idioms: `import g "maragu.dev/gomponents"` + `h "maragu.dev/gomponents/html"` (QUALIFIED `h.`, never dot-import). Arbitrary/SVG via `g.El`. Components are `func Name(p NameProps, attrs ...g.Node) g.Node` with a small Props struct; ATOMS take trailing variadic `...g.Node` so callers pass Datastar attrs through (exemplar: `internal/ui/button.go`, and `ui.Composer(p, attrs...)` at composer.go:58).
- Dependency direction is ONE-WAY: `internal/feature/* -> internal/ui`; `internal/ui` (incl. `internal/ui/chat`) must NEVER import `internal/feature/*`. The dock's feature-owned parts (recap zone content, the switchers' data, the composer) come IN as pre-rendered `g.Node` slots from the caller — same dependency-injection `ui.Composer.Decision` uses (composer.go:31-40, "the caller renders the card and hands it in").
- Datastar typed helper: `import data "maragu.dev/gomponents-datastar"`; `data.On("submit", "...")`, `data.Get(...)`. The legacy template uses raw `g.Attr("data-on:submit", ...)` strings in some places (composer.go:110); either is acceptable but match the surrounding file's choice.

### THE SSE selector ids the live stream depends on (MUST be preserved byte-for-byte)
The new markup MUST keep these exact ids/classes/structure or the live stream breaks silently:
- `id="chat"` on `<section class="chat" aria-live="polite">` — `chatstream.go:87,231` append to it (`WithSelectorID("chat")`); `tasks.go:413` (nudge poll) appends to it; `recap.go` reads it indirectly. Tests assert `id="chat"` (home_test.go:28-ish, handlers_test.go:207).
- `id="dock-convo"` — the flex wrapper around `#chat` (CSS `#dock-convo` at basm.css:3208 makes the composer pin to the bottom). focus.go patches `#main` only, never `#dock-convo`, so the dock survives in-app nav — keep the id.
- `id="chat-draft"` — the `ui.Composer` root; `patchChatbar` (models.go:125) and the disabled→enabled re-render target (`WithSelectorID("chat-draft")`). Provided by `composerNode` (home.go:33).
- `id="recap"` (`class="recap-zone"`) — `recap.go:95` patches `#recap` inner with bands (`WithSelectorID("recap")`); the `data-on:intersect__once="@get('/ui/recap/bands')"` sentinel must stay.
- `id="nudge-poll"` — carries `data-signals:nudgeSince`, `data-signals:dockMaster`, `data-signals:streaming` and the 30s interval poll. The `streaming` signal is read by `chatstream.go:116-118,244-246` (`MarshalAndPatchSignals{Streaming}`) and the head-switcher's `data-attr:disabled="$streaming"`. Keep the signal seeds.
- `id="model-modal"` (`<dialog>`) — opened by basm.js after a model-panel swap. Keep it.
- `id="chatbar"` — patched by `patchChatbar` (models.go:117) + the 2s `@get('/ui/chatbar')` poll while not ready. (See the NOTE above about whether it is currently mounted in the dock.)
- `id="head-switcher"` — patched by `setActiveHead` (heads.go:33).
- Choice ids `choices-{nonce}` and the `.choices` class — `chatstream.go:107` `RemoveElement(".choices")`; choices are rendered by the `chat-choices` template (chat-messages.html:100-121) which this plan does NOT touch (it is still executed by `chatstream.go:227` and `messageViews`/reload path). Leave `chat-messages.html` alone.
- Per-turn bubble/tool ids `balaur-{base}-{n}` / `tool-{base}-{n}` (+`-body`, +`-card`) — generated in Go (chatstream.go:126-160), NOT in any template. The `chat.Message`/`chat.ToolRow` organisms already carry them via `ID`/`BodyID`. Untouched by this plan.

### Existing storybook chat stories (the pattern to extend)
- `internal/feature/storybook/stories_chat.go` — `chatmessageStory()` (id `chatmessage`, `OnDark:true`), `chattoolrowStory()` (id `chattoolrow`, `OnDark:true`), `composerStory()` (id `composer`, `OnDock:true`). Each returns a `Story{ID,Group:"Chat",Title,Wide,OnDock|OnDark,Blurb,Variants:[]Variant{{name,node}},Props,Dos,Donts}`.
- `internal/feature/storybook/story.go:54-83` — the `stories` slice registers each builder; `chatmessageStory(), chattoolrowStory(), composerStory()` are at lines 81-83. `OnDock bool` (story.go:46-50) renders the variant on the always-dark wood dock (`--chrome`), which is the right stage for the dock chrome.

## Commands you will need
| Purpose | Command | Expected |
| Drift check | `git diff --stat 12a2ff5..HEAD -- internal/web internal/ui/chat internal/feature/storybook web/templates internal/web/assets/static/basm.css internal/self/knowledge.md` | only files this plan touches, if any |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Test (web) | `go test ./internal/web/...` | ok |
| Storybook render | `go test ./internal/feature/storybook/...` | ok |
| Format | `gofmt -l internal/ui/chat internal/web internal/feature/storybook` | empty output |
| Whitespace | `git diff --check` | no output |
| Selector-id grep | `grep -rn 'id="chat"\|dock-convo\|chat-draft\|id="recap"\|nudge-poll\|model-modal\|head-switcher\|chatbar' internal/web web/templates` | every id still present somewhere |
| Run app (visual) | app may already serve on 127.0.0.1:8090; else `go run . serve --http=127.0.0.1:8090` | serves |
| Streaming round-trip | in the app, send a chat message; watch SSE patches | bubble appends, body morphs token-by-token, finalizes |

## Scope
**In scope** (only files you may modify):
- `internal/ui/chat/dock.go` (NEW) — the `chat.Dock` organism (shell + variant). Possibly `internal/ui/chat/switcher.go` (NEW) for `chat.ModelSwitcher`/`chat.HeadSwitcher` if you decide the switchers belong as organisms (see Step 2 decision gate).
- `internal/feature/storybook/stories_chat.go` + `internal/feature/storybook/story.go` — add story builders and register them.
- `internal/web/home.go`, `internal/web/focus.go` — render `chat.Dock(...)` instead of `g.Raw(chat_dock)`.
- `internal/web/models.go`, `internal/web/heads.go` — IF the switchers move to gomponents, repoint `patchChatbar`/`setActiveHead` to render the organism instead of `ExecuteTemplate`.
- `internal/web/assets/static/basm.css` — collapse the three dock width blocks to one variant-driven rule set using the width tokens; add tokens to `:root` if plan 076 did not.
- `web/templates/home.html` — delete `chat_dock`/`dock_convo` (and `chat_bar`/`model_switcher`/`head_switcher` ONLY if their SSE patch sites are repointed) once the gomponents path is proven.
- `internal/self/knowledge.md` — update the Surfaces paragraph (lines 103-122): the dock chrome is no longer legacy `g.Raw`.
- `*_test.go` in `internal/web` — update assertions that referenced the template (templates_test.go, home_test.go, handlers_test.go) to the new render path.

**Out of scope** (do NOT touch):
- `internal/turn/*` — gateway-only change; the loop is the single source of truth (AGENTS.md "Gateways adapt; they never re-implement").
- `internal/ui/chat/message.go`, `internal/ui/chat/toolrow.go` — the message/tool organisms already work and the stream depends on their exact output; reuse, do not modify.
- `internal/ui/composer.go` — the composer is done; reuse via `composerNode`.
- `web/templates/chat-messages.html` — still executed by `chatstream.go` (`chat-choices`) and the choices reload path; the bubble open/close fragments are dead-but-untouched here. Leave it; retiring it is a SEPARATE plan.
- The streaming LIFECYCLE in `chatstream.go` (append/morph/finalize) — only the selector ids matter; do not change behavior.
- `internal/web/recap.go` rendering, `internal/web/tasks.go` nudge poll — they target `#chat`/`#recap` which you keep.

## Git workflow
Branch `improve/084-chat-dock-gomponents-port`. Conventional commits, one per slice (e.g. `feat(ui): add chat.Dock organism + stories`, `refactor(web): render home dock via chat.Dock`, `refactor(web): render focus dock via chat.Dock`, `refactor(css): collapse dock width blocks to variant tokens`, `chore(web): retire legacy chat_dock template`). Do NOT push or open a PR unless told. If `go mod download` fails with a TLS/certificate error in the sandbox, apply the GOPROXY shim per `docs/hyperagent-sandbox.md` (GOSUMDB stays on).

## Steps

### Step 1: Confirm the current behavior and enumerate every selector id (no code change)
Read `web/templates/home.html`, `internal/web/chatstream.go`, `internal/web/models.go`, `internal/web/heads.go`, `internal/web/recap.go`, and run the selector-id grep. Write down (for your own reference) the exact set of ids the new markup must reproduce: `chat`, `dock-convo`, `chat-draft`, `recap`, `nudge-poll`, `model-modal`, `chatbar`, `head-switcher`, plus the `.choices` class and the `streaming`/`nudgeSince`/`dockMaster`/`dockMaster` signal seeds. Confirm the rail/overlay/home CSS blocks at the cited line numbers.
**Verify**: `grep -rn 'id="chat"\|dock-convo\|chat-draft\|id="recap"\|nudge-poll\|model-modal\|head-switcher' internal/web web/templates` lists all ids -> proceed. If any cited line number is off by more than a few lines or an excerpt differs materially, STOP and report the drift.

### Step 2: Decision gate — how far to port the switchers
The dock chrome splits into (a) the shell (grip, head, recap-zone, dock-convo wrapper, nudge-poll, composer slot, model-modal dialog) and (b) the switchers (`model_switcher`, `head_switcher`) which are LIVE SSE patch targets (`#chatbar` via `patchChatbar`, `#head-switcher` via `setActiveHead`). Porting the switchers to gomponents means repointing those two handlers from `ExecuteTemplate` to `node.Render`. Choose the SMALLEST slice that lands cleanly:
- **Preferred**: port the shell now as `chat.Dock` with feature-injected `g.Node` slots (`Recap`, `Convo`, `Composer`, `Switchers`); leave `model_switcher`/`head_switcher` as templates for THIS plan (their `#chatbar`/`#head-switcher` patch sites are unaffected because the dock just receives a pre-rendered `Switchers g.Node`). This is the Pareto slice: it kills the `chat_dock`/`dock_convo` `g.Raw` injection without touching the switcher SSE contract.
- **Optional follow-up (same plan, separate commit, only if the shell slice is green and small)**: port the switchers to `chat.ModelSwitcher`/`chat.HeadSwitcher` and repoint `patchChatbar`/`setActiveHead`. If this would push the diff past a reviewable size, DEFER it to a follow-up plan and note it in Maintenance notes — do NOT big-bang.

Record your choice in the branch's first commit message. The steps below assume the Preferred slice for the shell and treat switcher porting as the optional tail (Step 7).
**Verify**: no command — this is a planning decision. Proceed to Step 3.

### Step 3: Build the `chat.Dock` organism with a variant prop (no behavior change yet)
Create `internal/ui/chat/dock.go`. Mirror the message/toolrow file header (`package chat`, qualified `h.`). Define:
```go
// DockVariant selects the dock's width/density: "rail" (the right-rail
// sidebar), "overlay" (.dock-full full-screen over content), "home" (the
// full-screen companion chat home — the widest column). The variant maps to a
// CSS width token, collapsing the three hand-synced #dock width blocks.
type DockVariant string
const (
    DockRail    DockVariant = "rail"
    DockOverlay DockVariant = "overlay"
    DockHome    DockVariant = "home"
)

type DockProps struct {
    Variant   DockVariant
    HasRecap  bool
    NowMillis int64    // nudge-poll cursor seed
    Convo     g.Node   // #chat section content (history/greeting) — caller-rendered
    Composer  g.Node   // ui.Composer node — caller-rendered (feature-injected)
    Switchers g.Node   // the chatbar/switcher node — caller-rendered (may be nil)
}
```
`Dock(p DockProps, attrs ...g.Node) g.Node` renders the EXACT structure of `chat_dock`/`dock_convo` (same ids, classes, order):
1. `.dock-grip` div (title/aria-hidden);
2. `.dock-head` header with the `.dock-btn.dock-full-btn` button (`onclick="basmToggleDockFull()"`, `aria-label`);
3. if `p.HasRecap`: `#recap.recap-zone` with `data-on:intersect__once="@get('/ui/recap/bands')"` and the `.recap-hint` child;
4. `#dock-convo` wrapping `<section class="chat" id="chat" aria-live="polite">{p.Convo}</section>` (this is the `dock_convo` content — inline it, the dock IS the only caller);
5. `#nudge-poll` with the three `data-signals:*` seeds (use `p.NowMillis` for `nudgeSince`) and the 30s interval poll attr;
6. `p.Composer` (the `ui.Composer` node);
7. `<dialog id="model-modal" aria-labelledby="model-modal-title">`.
Add a variant class on the ROOT (e.g. `dock-v-rail`/`dock-v-overlay`/`dock-v-home`) so the CSS can key width off the component, NOT off `html.home`/`.dock-full` (Step 6 ties this together). Keep the existing `html`/`dock-full` class selection working too (do NOT remove the no-flash class toggles) — the variant class is additive and authoritative for width; the `html.home`/`.dock-full` classes still drive z-index/overlay behavior. Pass `attrs...` onto the root after the variant class.
NOTE: the root here is the CONTENT of `<aside id="dock">` (shell.go:44 already provides the `<aside id="dock">`), so `chat.Dock` must NOT emit its own `#dock` wrapper — it emits the dock's children, matching what `chat_dock` does today (the template emits children of `#dock`, not `#dock` itself). Put the variant class on the `<aside id="dock">` instead — which means the variant must be applied at the shell mount, not inside `chat.Dock`. RESOLVE this: either (a) add a `DockClass string` to `shell.PageProps` and set it from the caller, or (b) keep the variant class on a wrapper div that `chat.Dock` emits as its single root child of `#dock`. Choose (a) if `shell.PageProps` is in scope and minimal; otherwise (b). Document the choice in the commit. (Out-of-scope guard: editing `internal/ui/shell/shell.go` is acceptable for this one additive field — add it to In scope if you pick (a).)
**Verify**: `CGO_ENABLED=0 go build ./...` -> exit 0; `go vet ./...` -> exit 0.

### Step 4: Add storybook stories for `chat.Dock` (OnDock stage)
In `internal/feature/storybook/stories_chat.go`, add `chatdockStory()` returning a `Story{ID:"chatdock", Group:"Chat", Title:"Dock", Wide:true, OnDock:true, ...}` with one Variant per `DockVariant` ("rail", "overlay", "home"). For each, pass a small fixture `Convo` (a couple of `chat.Message` panels), a `ui.Composer{...}` (static draft, no PostURL), and a nil or simple `Switchers`. Fill `Blurb`/`Props`/`Dos`/`Donts` matching the design intent (the dock is the persistent chat column; rail vs overlay vs home are the three width contexts). Register `chatdockStory()` in the `stories` slice in `internal/feature/storybook/story.go` next to `composerStory()` (around line 83).
**Verify**: `go test ./internal/feature/storybook/...` -> ok (TestAllStoriesRender renders the new variants); `go test ./...` checks tours_test.go is still green.

### Step 5: Switch `home.go` to render `chat.Dock` instead of `g.Raw(chat_dock)`
In `internal/web/home.go`, `homePage` (currently home.go:57-77): replace the `h.tmpl.ExecuteTemplate(&dockHTML, "chat_dock", dock)` + `g.Raw(dockHTML.String())` with a `chat.Dock(chat.DockProps{Variant: chat.DockHome, HasRecap: dock.HasRecap, NowMillis: dock.NowMillis, Convo: g.Raw(string(dock.ChatBodyHTML)), Composer: composerNode(dock), Switchers: <nil or rendered>})` node, passed to `shell.Page{...Dock: <that node>}`. (`dock.ChatBodyHTML` is already a rendered `template.HTML`; wrap with `g.Raw(string(...))`. `composerNode(dock)` returns the `ui.Composer` node directly — no need to round-trip through `composerHTML`.) Keep `HTMLClass:"home"` (and set `DockClass`/variant per Step 3 resolution). NOTE: `composerHTML` does NOT become orphaned by this change — `web.go:284` (`data.ComposerHTML = composerHTML(data)` in `dockData`) still calls it, and `web.go` is OUT of scope, so `composerHTML` and home.go's `html/template`/`strings` imports stay referenced; do NOT delete them (the build will fail if you do). After the port, `homeData.ComposerHTML` (set at web.go:284) becomes a dead field that `dockData` still populates — leave it as-is (retiring it is a separate web.go change, out of scope here).
**Verify**: `CGO_ENABLED=0 go build ./...` -> exit 0; `go vet ./...` -> exit 0; `go test ./internal/web/...` -> ok (home_test.go and handlers_test.go assert `id="chat"`, `<aside id="dock">`, `class="dock-grip"` — they must still pass; if an assertion referenced a template-only artifact, update it).
**Verify (LIVE — REQUIRED)**: run the app, open `/`, send a chat message. Confirm the SSE stream appends a user bubble, opens a pending assistant bubble, morphs its body token-by-token, and finalizes. Do this in BOTH `dark` and `light` (set `document.documentElement.className='theme-hearthwood dark'` then `... light'` in devtools). If streaming does NOT round-trip, STOP — a selector id regressed.

### Step 6: Collapse the three dock width CSS blocks to one variant-driven rule set
In `internal/web/assets/static/basm.css`: confirm `--w-chat-home:1800px` and `--w-chat-overlay:940px` exist in the `:root` layout block (plan 076); if not, add them after `--maxw:1080px` (basm.css:157) with a comment that the home/overlay divergence is intentional. Then rewrite the width rules to key off the variant class (`.dock-v-home`, `.dock-v-overlay`, `.dock-v-rail`) instead of `html.home #dock` / `.dock-full #dock`:
- `.dock-v-home .chat, .dock-v-home .composer, ... { max-width: var(--w-chat-home); margin-inline: auto; }`
- `.dock-v-overlay .chat, ... { max-width: var(--w-chat-overlay); margin-inline: auto; }`
- rail stays `width: var(--sidebar-w)` on `#dock` (unchanged).
Keep `#dock` positioning, the `.dock-full`/`html.home` z-index/overlay/`padding-right:0` rules, the `@media (max-width:900px)` overrides, and `#dock-convo`/`#dock .composer` flex pinning. The ONLY consolidation is the three `max-width`/centering blocks (basm.css:2452, 2453, 3191-3197, 3210, 3217) into variant-keyed rules. Append any NEW rule block at the END of basm.css under a `/* ── Section: chat dock variants ─────... */` banner (per the repo CSS convention); leave the existing blocks in place but reduce them to the non-width concerns. Do NOT shrink 1800px. Use only `var(--token)` colors (no raw hex; the dock uses `--chrome`/`--chrome-fg`/`--gold` which are ALWAYS-dark wood tokens — do not swap to ink tokens).
**Verify**: `git diff --check` -> no output; rebuild `CGO_ENABLED=0 go build ./...` -> exit 0.
**Verify (LIVE — REQUIRED)**: app `/` (home variant) — chat column caps at 1800px, centered; open a domain page e.g. `/focus/quests` (rail variant) — dock is the right rail at `--sidebar-w`; toggle full-screen (the `.dock-full-btn`, overlay variant) — column caps at 940px, centered. Check all three in BOTH modes and at viewport <=920px (the dock stacks under per `@media (max-width:900px)`). Measure with devtools that no column shrank below its prior width.

### Step 7: Switch `focus.go` to render `chat.Dock` (overlay/rail variant)
In `internal/web/focus.go`, the full-document branch (focus.go:156-173): replace the `chat_dock` `ExecuteTemplate` + `g.Raw` with `chat.Dock(chat.DockProps{Variant: chat.DockRail, ...})` exactly as in Step 5 (focus is NOT home — it uses the rail by default; full-screen overlay is toggled at runtime via the `.dock-full` class, which the variant CSS still honors). Keep the `focus_main` body `g.Raw` as-is (out of scope). Remove `html/template`/`strings` imports from focus.go only if your change orphaned them (focus.go still uses `template.HTML` in `focusView` and `strings.Builder` for `focus_main` — likely still needed; do NOT remove blindly, let the build tell you).
**Verify**: `CGO_ENABLED=0 go build ./...` -> exit 0; `go vet ./...` -> exit 0; `go test ./internal/web/...` -> ok.
**Verify (LIVE — REQUIRED)**: open `/focus/quests` via a direct browser load (full document). The dock renders as the right rail with the live chat; send a message and confirm streaming round-trips. Then click an in-app nav link (Datastar `@get` patching `#main`) and confirm the dock + its chat SURVIVE (focus patches `#main` only). Both modes.

### Step 8: (Optional, per Step 2 decision) port the switchers + retire their templates
ONLY if you chose the optional tail and the diff stays reviewable: add `chat.ModelSwitcher`/`chat.HeadSwitcher` organisms (feature data injected as props/slots; no `internal/feature` import), repoint `patchChatbar` (models.go:111-128) to render `chat.ModelSwitcher` and the `chatbar` wrapper, and `setActiveHead` (heads.go:30-33) to render `chat.HeadSwitcher` with `WithSelectorID("head-switcher")`. Preserve the `#chatbar`/`#head-switcher` ids, the `data-on:interval__duration.2s="@get('/ui/chatbar')"` poll (only when not ready), the `data-attr:disabled="$streaming"` gate, and the `data-on:submit__prevent="@post('/ui/heads/active', {contentType:'form'})"` form contract. Update `TestModelsPageAndCleanChatbarRender`/`TestChatbarPollAndDraft` (templates_test.go:45,335) to assert on the rendered node instead of `ExecuteTemplate("chat_bar")`.
If you DEFER this, skip to Step 9 and keep `chat_bar`/`model_switcher`/`head_switcher` in home.html.
**Verify**: `go test ./internal/web/...` -> ok; LIVE: switch the active head in the dock (or rail) — the switcher re-renders, the next turn uses the new voice; the 2s poll still enables the composer when a model becomes ready.

### Step 9: Retire the dead template defines and update knowledge.md
Delete `chat_dock` and `dock_convo` from `web/templates/home.html` (and `chat_bar`/`model_switcher`/`head_switcher` ONLY if Step 8 was done). If the whole file becomes empty/unreferenced, delete `web/templates/home.html`. Confirm nothing else references the deleted defines: `grep -rn 'chat_dock\|dock_convo\|"chat_bar"\|"model_switcher"\|"head_switcher"' internal/web web/templates` returns nothing for deleted names. Update `internal/self/knowledge.md` Surfaces paragraph (lines 103-122): change the sentence "The surrounding dock chrome (chat_dock shell: grip, recap zone, nudge poller, head/model switcher) is still the legacy template injected via g.Raw, pending a full chat.Dock port." to state the dock chrome now renders via the `chat.Dock` organism (and note if the switchers were/were not ported).
**Verify**: `grep -rn 'chat_dock\|dock_convo' internal/web web/templates` -> empty; `go test ./...` -> all pass (templates_test.go `TestTemplatesParse` must still parse the remaining defines); `CGO_ENABLED=0 go build ./...` -> exit 0.

### Step 10: Final gate
Run the full check set and update the 084 row in `plans/readme.md` (add the row if it is not present yet, matching the existing column format).
**Verify**: `CGO_ENABLED=0 go build ./...` exit 0; `go vet ./...` exit 0; `go test ./...` all pass; `gofmt -l internal/ui/chat internal/web internal/feature/storybook` empty; `git diff --check` no output.

## Test plan
- **New**: `chatdockStory()` in stories_chat.go renders 3 variants (rail/overlay/home) — `TestAllStoriesRender` (in storybook package) exercises it; `go test ./internal/feature/storybook/...`.
- **Updated**: `internal/web/home_test.go` `TestHomeFullChat` and `internal/web/handlers_test.go` `TestHandlerHomePage` already assert `<aside id="dock">`, `class="dock-grip"`, `id="chat"`, `<main id="main"></main>`, `<html lang="en" class="home">` — these must STILL pass after the port (the gomponents output must reproduce those exact strings). If `chat.Dock` emits `<section class="chat" id="chat" ...>` and `<div class="dock-grip" ...>` identically, no assertion change is needed; if attribute ORDER differs (gomponents emits attrs in call order), adjust the test's substring (e.g. assert `id="chat"` and `class="chat"` separately rather than a combined `class="chat" id="chat"`). Pattern to follow: the existing substring assertions in home_test.go.
- **Updated (only if Step 8 done)**: `TestModelsPageAndCleanChatbarRender` (templates_test.go:45) and `TestChatbarPollAndDraft` (templates_test.go:335) — repoint from `ExecuteTemplate("chat_bar")` to the new node render; keep the `id="chat-draft"`, `data-on:submit`, `/ui/chat`, disabled-state assertions.
- **Must stay green**: `chatstream_refresh_test.go`, `handlers_test.go` `TestChatHandler` (asserts `cmsg cmsg-user`, `cmsg cmsg-balaur`, `Hello from the fake model` in the SSE stream — proves the stream still patches the right markup), `templates_test.go` `TestChatStreamingBalancedDivs`/`TestChatMsg*PortraitStructure` (these test `chat-messages.html` fragments you did NOT touch).
- **Manual streaming round-trip (REQUIRED, see Steps 5/7)**: NOT done without it. If the live model is unavailable, STOP and report — do not declare done.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` all pass (incl. storybook + tours).
- [ ] `gofmt -l internal/ui/chat internal/web internal/feature/storybook` prints nothing.
- [ ] `git diff --check` prints nothing.
- [ ] `grep -rn 'chat_dock\|dock_convo' internal/web web/templates` returns nothing (template retired).
- [ ] `grep -rn 'g.Raw(dockHTML' internal/web` returns nothing (no more `g.Raw` of the dock template).
- [ ] `grep -rn 'id="chat"\|dock-convo\|chat-draft\|id="recap"\|nudge-poll\|model-modal\|head-switcher' internal/web` shows every live selector id is still produced by the gomponents path.
- [ ] `internal/feature/storybook/stories_chat.go` has `chatdockStory()` and it is registered in story.go.
- [ ] `internal/self/knowledge.md` Surfaces paragraph no longer says the dock chrome is legacy `g.Raw`.
- [ ] Only in-scope files changed (`git diff --stat`).
- [ ] plans/readme.md 084 row updated (add the row if it is not present yet, matching the existing column format).
- [ ] VISUAL (BOTH modes, rail + overlay + home widths, <=920px): home caps at 1800px, overlay at 940px, rail at `--sidebar-w`; dock chrome unchanged; a chat message streams append-then-morph correctly in each.

## STOP conditions
- Any SSE selector id (`chat`, `dock-convo`, `chat-draft`, `recap`, `nudge-poll`, `model-modal`, `chatbar`, `head-switcher`) or the `.choices` class would change, or the `streaming`/`nudgeSince`/`dockMaster` signals would be dropped — the live stream breaks silently. STOP and redesign to preserve it.
- A chat round-trip fails to stream/patch after switching home.go or focus.go (Step 5/7 LIVE verify). STOP — a regression is live.
- The slice exceeds a reviewable size for one commit. STOP, land what is green, defer the rest (especially Step 8) to a follow-up plan; never big-bang.
- The live model is unavailable so the streaming round-trip cannot be tested. STOP and report — do not declare done.
- A drift-check mismatch (the cited line numbers/excerpts no longer match HEAD). STOP and report.
- Any Verify command fails twice after a fix attempt. STOP and report the command + output.

## Maintenance notes
- The `chat.Dock` variant class is the single source of truth for the chat column width — future width changes touch ONE rule set, not three. The 1800px home / 940px overlay divergence is intentional and tokenized (`--w-chat-home`/`--w-chat-overlay`); a reviewer should scrutinize that the home column did NOT shrink (commit 12a2ff5 guardrail).
- If Step 8 was deferred, `chat_bar`/`model_switcher`/`head_switcher` remain in `web/templates/home.html` and `patchChatbar`/`setActiveHead` still `ExecuteTemplate` them. A follow-up plan should port them to `chat.ModelSwitcher`/`chat.HeadSwitcher` and retire those defines, repointing the two SSE patch sites — preserving `#chatbar`/`#head-switcher` ids and the `data-attr:disabled="$streaming"` gate.
- `web/templates/chat-messages.html` is OUT of scope: `chatstream.go` still executes its `chat-choices` define, and the `messageViews` reload path degrades choices to plain text. Retiring it (porting choices to a `chat.Choices` organism) is a separate plan (the SPEC's Choices organism) — do NOT fold it in here.
- A reviewer should diff the rendered `chat.Dock` output against the old `chat_dock` template byte-for-byte (ids, classes, attr presence) — gomponents emits attributes in call order, so attribute ORDER may differ from the template even when the element is equivalent; that is fine as long as ids/classes/Datastar attrs are all present.
- The `<aside id="dock">` wrapper is owned by `shell.go:44`, NOT by `chat.Dock` — keep that boundary; `chat.Dock` renders the children of `#dock`.
