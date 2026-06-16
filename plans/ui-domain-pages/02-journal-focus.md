# 02 — Port the Journal focus body (the candle) to a gomponents component

> **Read `plans/ui-domain-pages/README.md` first** (shared recipe, conventions,
> conflict map, verification). **Exemplar**:
> `internal/feature/lifecards/lifelogfocus.go` + `lifelog.go` +
> `internal/web/{cards.go,focus.go}`. Stamped against `884d692`.

## Context / why

`/focus/journal` (the "candle" — free-hand / guided journaling + today's
history) still renders from `web/templates/journal-focus.html` via
`(*handlers).journalFocusHTML`. Port it to a gomponents component in
`internal/feature/journalcards`, filling the `CardSize.Focus` seam (same shape as
the lifelog port in `884d692`). Smallest of the remaining bodies.

Wrinkle (same as quests' rail): the `journal_candle_body` fragment is
**re-rendered by a handler** — `journalWrite` (POST `/ui/journal`) calls
`renderCandleBody`, which patches `#journal-candle-body`. So the body section must
become a **shared component** used by both the focus body and `renderCandleBody`.

## Current state (read these)

`web/templates/journal-focus.html`:

- `journal_focus`: `<div class="candle-focus">` → a `<div class="k-tabs"
  role="tablist">` with two `<button class="k-tab[ k-tab-active]">` (free / guided)
  → `<div id="candle-prompt"></div>` → `{journal_candle_body}`.
  - **free** button `data-on:click`: `document.getElementById('candle-prompt').innerHTML='';el.parentElement.querySelectorAll('.k-tab').forEach(b=>b.classList.remove('k-tab-active'));el.classList.add('k-tab-active')`
  - **guided** button `data-on:click`: `el.parentElement.querySelectorAll('.k-tab').forEach(b=>b.classList.remove('k-tab-active'));el.classList.add('k-tab-active');@get('/ui/journal/prompt')`
- `journal_candle_body`: `<div id="journal-candle-body">` → `<form class="journal-form" data-on:submit__prevent="@post('/ui/journal', {contentType:'form'})"><textarea name="text" rows="8" placeholder="What stays with you from this day?"></textarea><button class="btn btn-primary btn-sm" type="submit">Keep it</button></form>` → `{if .Journal}<div class="journal-list"><article class="journal-entry"><div class="journal-meta"><span class="tl-time">{Time}</span><a class="btn btn-ghost btn-sm" href="/focus/day?date={Date}">→ this day</a></div><p class="journal-text">{Text}</p></article>…</div>{end}`.

Handlers/builders (`internal/web/journal.go`, read for current line numbers):
`journalFocusHTML` (renders `journal_focus`), `buildCandleData(time.Now())`
(today-only), `journalWrite` (POST `/ui/journal`, field `text`, empty→400),
`renderCandleBody` (renders `journal_candle_body`, patches `#journal-candle-body`
**outer**), `journalPrompt` (GET `/ui/journal/prompt`, patches `#candle-prompt`
**inner**). View-model: `candleData{… Today string; Journal []candleJournalView}`,
`candleJournalView{ID, Time, Text, Date string}`.

## Action contract — preserve byte-for-byte

| Trigger | Endpoint | Fields | SSE target & mode |
|---|---|---|---|
| "Keep it" submit | `@post('/ui/journal', {contentType:'form'})` | `text` (textarea) | `#journal-candle-body` **outer** |
| "guided" tab | `@get('/ui/journal/prompt')` | — | `#candle-prompt` **inner** |
| "free" tab | (no request) inline JS only | — | clears `#candle-prompt`, toggles `.k-tab-active` |

Element ids `#journal-candle-body` and `#candle-prompt` are load-bearing.
**Reproduce the two `data-on:click` inline-JS strings verbatim** (they mutate
`.k-tab-active` and the prompt div) — emit them with `g.Attr("data-on:click", "…")`.
The `textarea name="text"` and the form's `@post`+`{contentType:'form'}` are the
write contract.

## Scope

**In scope:** `internal/feature/journalcards/` (new `journalfocus.go` + the
`registerJournal` size dispatch), `internal/web/journal.go` (point
`renderCandleBody` at the shared component; drop `journalFocusHTML` if dead),
`internal/web/focus.go` (drop `case "journal"`), `web/templates/journal-focus.html`
(retire defines once unused), storybook.

**Out of scope (do NOT touch):** `journalPrompt` and its LLM call
(`h.clients.Active` — Ollama-adjacent, leave it; it just patches `#candle-prompt`
with raw HTML the component's empty `<div id="candle-prompt">` receives), the
`/ui/day/*` handlers, the README conflict files. Journal is **today-only** — do
not add date parameterization.

## Steps

1. `internal/feature/journalcards/journalfocus.go`: view-model
   `JournalFocusView{Journal []JournalEntryView}` (`JournalEntryView{ID, Time,
   Text, Date string}`) + `buildJournalFocus(app)` mirroring
   `web.buildCandleData(time.Now())` (reuse the `internal/life` journal read; no
   `internal/web` import).
2. **`JournalCandleBody(v JournalFocusView) g.Node`** — port of
   `journal_candle_body`: `<div id="journal-candle-body">` + the `journal-form`
   (`@post('/ui/journal', {contentType:'form'})`, `textarea name="text"`) + the
   `journal-list` of `journal-entry` rows (omit the list when empty). Preserve
   `tl-time`, `journal-meta`, `journal-text`, and the `→ this day` link
   `href="/focus/day?date={Date}"`.
3. **`JournalFocus(v JournalFocusView) g.Node`** — port of `journal_focus`:
   `<div class="candle-focus">` + the `k-tabs` (the two buttons with the verbatim
   inline-JS `data-on:click`) + `<div id="candle-prompt">` (empty) +
   `JournalCandleBody(v)`.
4. `registerJournal` (in `journalcards`): `ui.Focus` →
   `JournalFocus(buildJournalFocus(app))`, else the existing journal tile.
5. `internal/web/focus.go`: delete `case "journal": return h.journalFocusHTML()`.
6. `internal/web/journal.go` `renderCandleBody`: replace
   `ExecuteTemplate(&b, "journal_candle_body", …)` with
   `journalcards.JournalCandleBody(journalcards.BuildJournalFocus(h.app))` →
   keep `PatchElements(…, WithSelectorID("journal-candle-body"), WithModeOuter())`.
   (Export `buildJournalFocus`/`JournalCandleBody` as needed.) This keeps the
   write/re-render markup identical to the initial body.
7. Retire `journalFocusHTML` + the `journal_focus`/`journal_candle_body` defines
   **after** step 6 — first grep `*_test.go` for those names (e.g.
   `TestFocusJournalShowsCandle` in `internal/web/focus_test.go` asserts
   `@post('/ui/journal'`, `/ui/journal/prompt`, `id="journal-candle-body"` against
   the *route*, so it stays green via the ported body; but if any test
   `ExecuteTemplate`s the defines directly, leave them as dead code + a TODO).
8. Storybook `journalfocusStory()` (variants: with entries, empty) + register
   mid-Cards-cluster; + `journalfocus_test.go` asserting `candle-focus`, `k-tabs`,
   `id="candle-prompt"`, `id="journal-candle-body"`, the `@post('/ui/journal'`
   form, `journal-entry`.

## Done criteria

- `CGO_ENABLED=0 go build ./...` → 0; `…go test ./internal/feature/journalcards/...
  ./internal/web/... ./internal/feature/storybook/...` → ok;
  `gofmt -l internal/feature/journalcards internal/web internal/feature/storybook`
  → empty; `git diff --check` clean.
- `internal/web/focus_test.go::TestFocusJournalShowsCandle` still passes
  (`@post('/ui/journal'`, `/ui/journal/prompt`, `id="journal-candle-body"`).
- Live: `curl -s 127.0.0.1:PORT/focus/journal` contains `candle-focus`,
  `id="candle-prompt"`, `id="journal-candle-body"`, `name="text"`, plus the new
  topbar + `id="dock"`. Manually: write an entry → `#journal-candle-body` updates;
  click "guided" → a prompt appears in `#candle-prompt`.

## Test plan

`internal/feature/journalcards/journalfocus_test.go` (mirror `lifelogfocus_test.go`):
assert the class/id contract + the write form `@post`/`name="text"` + the tab
buttons' inline JS substrings, on a populated and an empty `JournalFocusView`.
Re-run `TestFocusJournalShowsCandle`.

## Maintenance note

The candle body has one renderer (`JournalCandleBody`) shared by the focus body
and `renderCandleBody`. `journalPrompt` (guided prompt, LLM) is intentionally
left as-is — it patches the empty `#candle-prompt`. Watch in review that
`#journal-candle-body`/`#candle-prompt` and the tab-toggle JS stay exact.

## Escape hatches

- If `renderCandleBody`'s data differs from `buildJournalFocus` (e.g. it re-reads
  with a different window), reconcile to one builder — don't fork the markup.
- If `journalPrompt`'s LLM call (`h.clients.Active`) has been refactored by the
  Ollama→Kronk work and won't compile, that's their surface — STOP and report;
  don't "fix" it here.
</content>
