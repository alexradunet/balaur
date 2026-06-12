# Plan 031: The candle — an immersive /journal writing page (free-hand + guided)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 9c77f42..HEAD -- internal/web web/templates web/static/basm.css internal/self/knowledge.md`
> Plans 025–030 being DONE is expected drift. Anything else touching the day
> journal handlers → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: LOW (new page over an existing domain path; one opt-in LLM call)
- **Depends on**: plans/025-hearthwood-visual-foundation.md
- **Category**: direction
- **Planned at**: commit `9c77f42`, 2026-06-12

## Why this matters

The Hearthwood design includes a dedicated writing mode — "the candle": a
calm, centered page with the crest glowing above a large parchment textarea,
two modes (free-hand, or guided by one prompt line from Balaur), and the day's
entries beneath. Journaling already exists as a domain (the `journal_write`
tool in chat and the form on `/day/{date}`), but it has no room of its own.
This plan gives it one, reusing the existing write path end-to-end.

## Current state

- **Existing journal surface**:
  - `POST /ui/day/{date}/journal` → `h.dayJournalWrite` and
    `POST /ui/day/journal/{id}/drop` → `h.dayJournalDrop`
    (registered in `internal/web/web.go:197-198`), rendering the
    `day_journal` fragment defined in `web/templates/day.html` (journal list +
    entry form, ids `day-journal`, classes `.journal-entry`, `.journal-meta`,
    `.journal-text`, `.journal-form` — all already Hearthwood-styled by 025's
    `pages.css` section).
  - The chat tool `journal_write` in `internal/tools/journal.go`.
  - Find where both persist (grep `journal` in `internal/web/*.go` and
    `internal/tools/journal.go`) — they share a domain write path; the new
    page MUST call the same one (verbatim capture, page-side removal only, per
    DESIGN.md's honesty ledger).
- **Voice + design constraints** (DESIGN.md): warm, no exclamation marks; the
  crest (`/static/crest.png`) ships borderless; activity/glow is CSS
  (`basm-glow` keyframes exist in basm.css); `prefers-reduced-motion`
  respected; mythic copy as seasoning only.
- **LLM use rule** (AGENTS.md): "Deterministic, offline, free behavior is the
  default. LLM/network paths are opt-in." Guided mode satisfies this: the
  model call happens only when the owner clicks "guided". The pattern for a
  small model-composed line with a deterministic fallback already exists in
  the task nudger (`internal/tasks`, "model-composed in the companion voice
  with a deterministic fallback" — find it with
  `grep -rn "fallback" internal/tasks/*.go`) — mirror its shape: resolve the
  active client, one short completion, on any error use the canned line.
- **Page idioms**: full pages embed `page_head` + `topbar`
  (`web/templates/life.html` is a clean exemplar); the chat home shows the
  crest with the `.hearth-crest` class (home.html:99).
- **Topbar nav** (`web/templates/layout.html:23-30`): Tasks · Life · Memory ·
  Heads · Settings (+ Boards after plan 029).

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./...`                  | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`    | clean               |

## Scope

**In scope**:
- `internal/web/journal.go`, `internal/web/journal_test.go` (create)
- `web/templates/journal.html` (create)
- `internal/web/web.go` (routes)
- `web/templates/layout.html` (nav link "Journal")
- `web/static/basm.css` (ONLY a small `## Candle` appendix: centered column,
  crest glow keyframe reuse, mode toggle; ~30 lines)
- `internal/self/knowledge.md`, `DESIGN.md` honesty ledger

**Out of scope**:
- The journal domain/write path itself — reuse, don't reshape.
- Storing a "guided/free" mode tag on entries (would need a schema change;
  deferred — entries are entries).
- Ambient audio from the mockup (rejected: against the quiet-craft principles).
- `/day/{date}` page changes.

## Git workflow

- Branch: `advisor/031-journal-candle`
- Commit style: `feat(journal): the candle — immersive /journal page`

## Steps

### Step 1: Routes + handlers

In `internal/web/web.go` register; implement in `internal/web/journal.go`:

```
GET  /journal            → full candle page (today's entries + form)
POST /ui/journal         → write entry for today; re-render the entries+form fragment
GET  /ui/journal/prompt  → guided mode: one prompt line (HTML fragment)
```

- `POST /ui/journal` resolves "today" server-side (use the same local-day
  resolution the day pages / briefing use — grep how `dayPage` parses/derives
  dates and how "once per local day" is computed in the briefing) and calls
  the SAME write path as `dayJournalWrite`. Re-renders a `journal_candle_body`
  fragment (entries list + cleared form). Entry removal: reuse the existing
  `POST /ui/day/journal/{id}/drop` endpoint with an `hx-target` of the candle
  fragment? — No: that handler re-renders the day-page fragment. Simplest
  correct: the candle page's entry list links each entry to its `/day/{date}`
  page for management, and offers no inline remove. (Honesty-ledger rule —
  "page-side removal only" — remains satisfied via the day page.)
- `GET /ui/journal/prompt`: model-composed single line in the companion voice
  ("What did the day ask of you?" energy — never saccharine), via the active
  `llm.Client`; system-style instruction ~2 sentences; clip to ~140 chars; on
  ANY error or no-model, return the deterministic fallback line
  `"Write what the day left behind. I am listening."`. Returns
  `<p class="candle-prompt">…</p>`.

**Verify**: `go test ./internal/web/...` → ok (tests below).

### Step 2: Template `web/templates/journal.html`

Full page: `page_head` + `topbar`, then a narrow centered column
(`.candle-page`, max-width ~640px):

1. The crest, borderless, with a soft breathing glow
   (`.candle-crest` — reuse the `basm-glow` keyframe on box-shadow is wrong
   for a transparent image; instead a CSS `filter: drop-shadow(0 0 18px …)`
   pulse, gold-toned, defined in the basm.css appendix, disabled under
   `prefers-reduced-motion`).
2. Mode toggle — two `.k-tab` buttons, "free hand" (default, active) and
   "guided": guided is a button with
   `hx-get="/ui/journal/prompt" hx-target="#candle-prompt" hx-swap="innerHTML"`;
   free-hand clears the prompt container (an `hx-on:click` that empties it, or
   a server round-trip — pick the one consistent with the no-JS-state rule:
   a tiny `hx-on:click="document.getElementById('candle-prompt').innerHTML=''"`
   is acceptable inline glue, matching existing inline `hx-on::after-request`
   usage in home.html).
3. `<div id="candle-prompt"></div>`
4. The write form — large textarea (`.journal-form` styling already exists;
   give it `rows="8"`), submit button "Keep it" (`.btn-primary`), posting to
   `/ui/journal`, target the `journal_candle_body` fragment id.
5. Today's entries (`.journal-entry` markup identical to day.html's), each
   wrapped in a link or with a small mono link "→ this day" to `/day/{date}`.

Add `<a href="/journal">Journal</a>` to the topbar nav (after Life).

**Verify**: `go test ./internal/web/...` → templates parse; `make run`, open
`/journal`: write an entry → it appears below and on `/day/{today}`; click
guided → a prompt line appears; with no model configured → the fallback line
appears.

### Step 3: Docs

- knowledge.md: the candle page paragraph (free-hand + guided, guided is one
  model line, entries are the same journal records as chat/day pages).
- DESIGN.md "True today": append "the candle (/journal): immersive writing
  page — free-hand or guided by one model-composed prompt line (deterministic
  fallback), entries shared with day pages".

**Verify**: `grep -n "candle" internal/self/knowledge.md DESIGN.md` → ≥1 each.

## Test plan

`internal/web/journal_test.go`, model after `handlers_test.go`:
- GET /journal → 200, contains `candle-page` and the form.
- POST /ui/journal with text → entry persisted (assert via the journal
  collection the day page reads) and response contains the text; empty text →
  400 or error fragment (match `dayJournalWrite`'s behavior — read it first).
- GET /ui/journal/prompt with the fake `llm.Client` returning a line → that
  line; with a failing client → the exact fallback string.
- Entry written via /ui/journal appears in GET /day/{today} (one integration
  assertion proving the shared path).

Verification: `go test ./...` → all pass.

## Done criteria

- [ ] `/journal` page live; nav link present
- [ ] Write path is shared with the day page (integration test proves it; no
      new journal-write query: `grep -c "journal" internal/web/journal.go`
      shows calls into the existing path, not a new collection write)
- [ ] Guided prompt falls back deterministically (asserted by test)
- [ ] Glow respects `prefers-reduced-motion` (CSS appendix includes the guard)
- [ ] `go test ./...`, vet, fmt, CGO-free build clean; `git diff --check` clean
- [ ] knowledge.md + DESIGN.md updated; `plans/readme.md` row updated

## STOP conditions

- The day-journal write path turns out to be inseparable from the day-page
  fragment rendering (i.e. no callable seam without refactoring beyond a pure
  extraction) — report the shape you found.
- The nudger's model-call pattern doesn't exist as described (can't find a
  composed-line-with-fallback exemplar) — write the simplest version but
  flag it.
- Anything requires storing new fields on journal entries.

## Maintenance notes

- If a mode tag (guided/free) is wanted later, it is a schema change on the
  journal collection + a migration — new plan.
- The guided prompt is the only LLM call on the page; if recall/embedding
  lands later (roadmap), guided mode is the natural place to weave "yesterday
  you wrote…" context — keep the handler small so that graft is easy.
- Reviewer: voice check on the fallback line and any copy (no exclamation
  marks, no hype); confirm the prompt line renders as escaped text (model
  output, never `template.HTML`).
