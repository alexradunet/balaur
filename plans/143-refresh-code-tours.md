# Plan 143: Refresh the code tours to the current architecture (prerequisite for plan 117)

> **Executor instructions**: Follow this plan tour by tour. Run the verification
> command after each tour and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat d8a8b66..HEAD -- .tours/`
> If a tour changed since this plan was written, reconcile against the per-tour
> worklist before editing.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (docs/content; `tours_test.go` is the gate)
- **Depends on**: none
- **Blocks**: **plan 117** (117 deletes `web/templates/` + `internal/web/templates_test.go`, which two tour anchors point at — this refresh must land FIRST, or 117's deletion breaks `TestTours`)
- **Category**: docs
- **Planned at**: commit `d8a8b66`, 2026-06-22

## Why this matters

`.tours/*.tour` are maintained, guided code walkthroughs, and `tours_test.go`
fails the whole suite if any step's `file` is missing or its `line` is
out-of-range. But `tours_test` does **not** check that the code at the anchor
*matches the prose*, so the tours currently pass tests while being badly stale —
they describe features removed across ~140 plans: **boards, `/focus` pages, htmx,
Ollama, `board_compose`, `gguf.Shared`, the domain rail, and `html/template` as
the render path**. A new contributor following these tours learns a codebase that
no longer exists.

This refresh also UNBLOCKS plan 117 (delete the `html/template` engine). Two tour
anchors land on files 117 deletes:
- **HARD**: `06-memory-and-self-evolution.tour` step 6.9 → `web/templates/card-memory.html` (117 deletes `web/templates/`).
- **HARD**: `07-the-web-gateway.tour` step 7.6 → `internal/web/templates_test.go` (117 deletes the template tests).
Both must be re-anchored off the doomed files HERE, before 117.

This survey was produced by a 12-agent workflow and adversarially verified: every
recommended new anchor below was opened and confirmed at HEAD `d8a8b66` (22
anchors spot-checked, all correct; the 29-migration count, the 14-card count, the
`internal/ollama` removal, and the 6-case `handleToolResult` order all verified).

## Current architecture ground-truth (use this for the rewritten prose)

The tours must reflect THIS. Removed → replacement axis:
- **Ollama → in-process Kronk** (`internal/kronk`: GGUF via yzma/llama.cpp, CGO-free, native lib `dlopen`'d) as the default local path; plus the **opt-in, consent-gated OpenAI-compatible cloud client** (`internal/llm/openai.go`). `internal/ollama` does NOT exist.
- **htmx → Datastar** (SSE hypermedia + client signals; `data-on:click`, `@get`, `@post`). The live UI is Datastar; do not anchor a tour on htmx. `htmx.min.js` is gone (`board.js` still lingers but is not the interaction layer).
- **boards / `board_compose` / `/focus` / domain rail → the single-page chat shell** (`shell.ChatShell`, `internal/ui/shell`) with a single collapsible/resizable **right panel** (`#panel`/`#panel-inner`) driven by `GET /ui/show/{type}` + `POST /ui/panel/collapse|width`, and the composer **`/`-command palette** (`ui.CommandPalette`, `internal/ui/command_palette.go`). Agent UI tools are now **`card_show`** (`internal/tools/ui.go`, one card) + **`show_cards`** (`internal/tools/artifact.go`, a 1–8 card cluster), not `board_compose`. (`shell.Sidebar` survives only as a storybook atom — NOT the product nav.)
- **`html/template` → typed gomponents**: atoms `internal/ui`, chat organisms `internal/ui/chat`, page shell `internal/ui/shell`, domain cards `internal/feature/*cards`, storybook `/storybook`. Cards register via `ui.RegisterCard` across `internal/feature/*cards` and surface via `GET /ui/cards` (palette) + `GET /ui/cards/{type}` + `GET /ui/show/{type}`. (The `html/template` engine in `web.go` is being retired by plan 117 — do NOT describe it as the render path.)
- **`gguf.Shared` → `turn.ClientSource{Engine: kronk.FromStore(app)}`**.
- Assets live in `internal/web/assets/static/` (`basm.css`, `datastar.js`, `basm.js`); there is NO root `web/static/` (root `web/` holds only `templates/` + `embed.go`).
- Migrations: **29** files (not 19). `InitCollections` (migrations/1749600000_init.go:23) creates `heads, conversations, messages, memories, skills, grants, audit_log`; `boards` was added then DROPPED (migrations/1750850000_drop_boards.go).

## Tour file format

Each `.tour` is JSON: `{ "title": ..., "steps": [ { "title", "description" (markdown),
"file", "line", ... } ] }`. You edit the `description` prose, the `file`, and the
`line` of each step. **Keep valid JSON** (escape quotes/newlines in `description`
as the existing steps do). After editing, `go test . -run TestTours` validates
every anchor.

## Per-tour worklist (verdict → actions; all new anchors verified at HEAD)

### `00-orientation.tour` — PARTIAL-REWRITE (6 of 8 steps stale; anchors in-range)
- **0.1**: replace "Ollama / `internal/ollama`" → in-process Kronk; "HTMX web UI" → "Datastar web UI"; fix "local = endpoint Balaur manages" → local = in-process Kronk; OpenAI-compatible is a separate opt-in consent-gated remote client (`internal/llm/openai.go`).
- **0.2**: rewrite the OnServe snippet to match `main.go:35-65` — `registerKronkEngine(se.App)` first, then `if err := web.Register(se); err != nil { return err }`, then `registerRecap/Nudge/Briefing/SearchIndex`; note `OnTerminate` unloads Kronk + closes the search index; relabel the gateway "Datastar".
- **0.3**: "19 migration files" → **29**; `InitCollections` = `heads,conversations,messages,memories,skills,grants,audit_log`; drop the "boards added" claim (dropped in `1750850000_drop_boards.go`).
- **0.4**: delete "no remote provider — local only" (`internal/llm/openai.go` exists); the package is now `llm.go, env.go, openai.go, openai_test.go`; local inference is `internal/kronk`.
- **0.8**: MAJOR — relabel Datastar; REMOVE the `boards.go` + `tasks.go` quest-rail bullets (gone); fix cards prose (now `ui.RegisterCard` across `internal/feature/*cards`, served by `/ui/cards`+`/ui/cards/{type}`+`/ui/show/{type}` into the single right panel); fix asset paths (`internal/web/assets/static/`, no `htmx.min.js`).
- **0.9**: add `seed` to the CLI command list.

### `01-packages-structs-interfaces.tour` — PARTIAL-REWRITE (concepts fine; anchor drift + 1 prose fix)
- Re-anchor the 7 steps: 1.1 `llm.go:8`, 1.2 `llm.go:13`, 1.3 `llm.go:21`, 1.4 `llm.go:35`, 1.5 `llm.go:44`, 1.6 `client.go:36`, 1.7 `engine.go:125`.
- **1.1** prose only: drop "Ollama managed in `internal/ollama`" + "pure HTTP client layer"; the default local path is in-process Kronk implementing `llm.Client`; the package now has 4 files.

### `02-goroutines-channels.tour` — RE-ANCHOR-ONLY (substance healthy, no stale features)
- Re-anchor: 2.1 `engine.go:36`, 2.2 `engine.go:73`, 2.3 `engine.go:125`, 2.4 `client.go:92`, 2.5 `client.go:100`. No prose rewrites (optional: note the bridge now `defer cancel()`s the deadline ctx).

### `03-agent-loop.tour` — PARTIAL-REWRITE (anchors valid, loop unchanged; htmx prose stale)
- **3.2**: replace "emit writes HTML fragments for the HTMX response" → Datastar SSE via `internal/turn` (`turn.Run`, `chat.go:51`).
- **3.4**: `chat.go` now calls `turn.Run(..., cs.emit)` patching Datastar SSE, not `<div>` to a `ResponseWriter`.
- **3.6**: fix the quote — code is `var text strings.Builder` / `text.WriteString(chunk.Content)` + an `if chunk.Reasoning != ""` branch; "HTMX handler can flush" → "Datastar SSE gateway".

### `04-testing-fakes-closures.tour` — RE-ANCHOR-ONLY / near-keep
- **4.2** only: "`openai.go` and `internal/ollama.LocalClient`" → "`internal/llm/openai.go` and `internal/kronk.Client`"; "the real `LocalClient` or `OpenAIClient`" → "the real `kronk.Client` or `OpenAIClient`".

### `06-memory-and-self-evolution.tour` — PARTIAL-REWRITE (core thesis intact; 2 heavy rewrites)
- **6.9 (HARD re-anchor — blocks 117)**: change `file` `web/templates/card-memory.html` → `internal/feature/knowledgecards/memory.go`, `line` → `84` (`func MemoryRecordCard(r MemoryRecord) g.Node`); rewrite prose to typed gomponents (a `MemoryRecord` view-model, no `{{.GetString}}` templating).
- **6.8**: `chatstream.go:184/187` anchors valid (`handleToolResult`) but replace the dead `case "tool_result": …break` switch + HTMX-slot prose with the real **6-way early-return order**: `uicard → choices → proposal → refresh → artifact → plain`; the proposal card is a gomponents node via `proposalBody`; drop `board_compose`/off-board.
- **6.7**: now **FIVE** markers (Proposal / Choices / UICard / Artifact / Refresh); drop `board_compose`.
- Re-anchor the nudges: 6.3 → `159`/`162`, 6.4 → `9`/`12`, 6.5 → `30`/`32`, 6.6 → `231`/`233`, 6.10 → `26`/`31`; fix "Seven tests" → "eleven".

### `07-the-web-gateway.tour` — MAJOR-REWRITE (5 of 7 steps describe removed code; thesis kept)
- Re-anchor: 7.1 `web.go:163` (`Register`), 7.2 `web.go:107` (`guardLocalUI`), 7.3 `chat.go:27`, 7.4 `chatstream.go:159` (`emit`), 7.5 `chatstream.go:187` (`handleToolResult`), 7.7 `tasks.go:179` (`chatNudges`).
- **7.6 (HARD re-anchor — blocks 117)**: it currently anchors at `internal/web/templates_test.go`, which plan 117 deletes. Re-anchor it to a surviving test that exercises the gomponents render path — e.g. `internal/web/handlers_test.go` (or a `*_gomponents_test.go`); reframe the prose from "the template engine is tested" to "handlers render gomponents nodes, asserted in `*_gomponents_test.go`".
- Purge every removed token across the prose: `boards`/`boardHome`, `/focus`, `#quest-rail`, `gguf.Shared` (→ `turn.ClientSource{Engine: kronk.FromStore}`), htmx/OOB, `chat_draft`, `messageView`/`appendChat`.
- **7.5**: document the grown 6-case order and that uicard/cluster route to the single right panel via `/ui/show`, not inline.
- **7.7**: delete the quest-rail half; replace with `taskTransition` row removal (`urow-{src}-{id}` `WithModeRemove`).
- Reframe the tour's top-level description from boards → the single-page chat shell + single right panel + `/`-command palette + storybook.

### `08-hateoas-cards-and-boards.tour` — PARTIAL-REWRITE + RE-THEME (NOT a delete)
- **Retitle**: drop "and boards" → reframe around typed card resources rendered into the single right panel + inline chat embeds.
- Re-anchor: 8.2 → `internal/cards/cards.go:296` (`Validate`), 8.3 → `internal/web/cards.go:88` (`uiCard`), 8.6 → `internal/tools/ui.go:65` (`cardShowTool`).
- Fix counts: "10 card types" → **14** (confirm via the registry in `cards.go`); soften the "12-column / 10px grid" framing (W/H now feed panel sizing).
- Purge: `board_compose`, `/focus`, "switch typ"/`renderCardToday`/template-execution (now `cards.Validate` → `cardInto` → `ui.LookupCard` → feature gomponents), htmx → Datastar. 8.7/8.8 keep anchors, fix only the footnotes (cards parsed in `chatstream.go` + Datastar SSE; choices render via `chat.Choices` gomponents).

### `09-recall-and-search.tour` — KEEP (accurate)
- No changes. (Confirmed `BuildContext` `context.go:32`, `SearchActive` `knowledge.go:233`.) Optional nit: 9.7 `OnTerminate` is in `main.go`'s `app.OnTerminate` block — prose still correct.

### `10-the-cli-api.tour` — RE-ANCHOR-ONLY (architecture accurate)
- Re-anchor: 10.2 → `cli.go:102` (`type envelope`; `emit` at `109`), 10.3 → `cli.go:81` (`func run`). 10.1/10.4/10.5/10.6 keep. (Not a tour fix, just flag: `doctor.go` still prints the retired `/settings/models` route.)

## Commands you will need

| Purpose | Command | Expected |
|---------|---------|----------|
| Tours gate | `go test . -run TestTours` | PASS (all anchors valid) |
| JSON valid | `for f in .tours/*.tour; do jq -e . "$f" >/dev/null || echo "BAD: $f"; done` | no output |
| No doomed anchors | `for f in .tours/*.tour; do jq -r '.steps[]?.file // empty' "$f"; done \| grep -E 'web/templates/\|templates_test.go'` | empty |
| Stale-token scan | `grep -riE 'htmx\|ollama\|board_compose\|gguf.Shared\|/focus\|internal/ollama' .tours/` | only intentional "was retired"-style historical mentions, no CURRENT descriptions |

## Steps

Work tour by tour in the worklist order. After EACH tour's edits:
1. `jq -e . .tours/<that file>.tour >/dev/null` → valid JSON.
2. `go test . -run TestTours` → PASS.

Then the final checks (the four commands above).

## Test plan

- `tours_test.go` (`go test . -run TestTours`) is the machine gate — it must pass
  after every tour and at the end (every `file` exists, every `line` in range).
- There is no automated check that the PROSE is accurate — a reviewer reads each
  refreshed tour against the "Current architecture ground-truth" section. The
  stale-token scan is a backstop, not a proof.

## Done criteria

- [ ] `go test . -run TestTours` PASSES
- [ ] `go test ./...` still passes (no other suite touched)
- [ ] No `.tour` step `file` anchors at `web/templates/` or `internal/web/templates_test.go` (the "No doomed anchors" command is empty) — this is what unblocks 117
- [ ] The stale-token scan shows no token presented as CURRENT behavior (htmx/Ollama/board_compose/gguf.Shared/`/focus`/`internal/ollama`)
- [ ] Each tour edited per its worklist verdict (09 unchanged; 02/04/10 re-anchored; 00/01/03/06/08 partial; 07 major)
- [ ] Only `.tours/*.tour` and `plans/readme.md` modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report if:
- A recommended new anchor doesn't resolve to the described concept (the code
  drifted again since `d8a8b66`) — re-locate by symbol and note it; if you can't,
  STOP.
- `tours_test` fails after a tour and the cause isn't an obvious typo'd line
  number (e.g. the file was deleted) — report.
- Editing a tour seems to require touching source code — it does not; this plan is
  `.tours/`-only.

## Scope

**In scope**: `.tours/*.tour` (the 10 tour files), `plans/readme.md` (status row).
**Out of scope**: all source. The stale SOURCE comments the survey flagged —
`internal/web/agent.go:29`-area "SSE → HTMX fragments", `chat.go`'s template
comment, `doctor.go`'s `/settings/models` — belong to plan 117 / a separate
cleanup, NOT this tour refresh.

## Git workflow

- Branch off `origin/main`: `improve/143-refresh-code-tours`.
- One commit (or one per tour); conventional subject, e.g.
  `docs(tours): refresh code tours to the current architecture`.
- Do NOT push or open a PR.

## Maintenance notes

- After this lands, **plan 117** can delete `web/templates/` + `internal/web/templates_test.go`
  without breaking `TestTours` (the two anchors that pointed there are re-anchored
  here). 117 still owns the source-comment fixes above.
- `tours_test` only validates anchors, not prose — when a future change moves the
  code a tour describes, re-read the tour, don't just chase the line number.
