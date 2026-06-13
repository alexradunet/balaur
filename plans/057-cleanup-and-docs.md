# Plan 057: Cleanup & docs — finish the card-first program (Phase 7)

> **Executor instructions**: Follow this plan step by step. Run every Verify and
> confirm before moving on. On a STOP condition, stop and report. When done,
> update the `057` row in `plans/readme.md`. Execute with
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`.
>
> **Drift check (run first)**: `git diff --stat 06361b0..HEAD -- internal/web web/templates DESIGN.md`
> Authored at `06361b0` (Phase 6 / plan 056 merged). Spec:
> `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md`.
> If `internal/web/chat.go`, `internal/web/web.go`, `web/templates/home.html`,
> `web/templates/cards.html`, `web/templates/layout.html`, or `DESIGN.md` changed
> since `06361b0`, compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P2 (Phase 7 — the finale)
- **Effort**: S–M
- **Risk**: LOW (dead-code removal + comment/doc fixes; no behavior change)
- **Depends on**: plans/050–056 (all DONE/merged)
- **Category**: direction (card-first "kill the pages", Phase 8 of 8 — final)
- **Planned at**: commit `06361b0`, 2026-06-13

## Why this matters

All feature pages are retired; the UI is **boards + focus-capable cards + a dock
chat that swaps conversations**. This plan closes the program: remove code
orphaned by the retirements, fix comments left stale by the HTMX→Datastar and
page→card migrations, give the docs a single holistic "card-first IA" pass, and
verify the whole program is coherent (no page route or page template survives).

## Current state (verified at `06361b0`)

- **Routes**: the only navigable GET routes left are `/` (`boardHome` → `/boards`),
  `/boards`, `/boards/{id}`, `/focus/{type}`, `/ui/cards`(palette), `/_/`(engine
  room), `/static/*`; everything else is `/ui/*` fragment/write routes. **No
  feature page route remains.**
- **Topbar** (`layout.html` `{{define "topbar"}}`): brand→`/`, `Boards`→`/boards`,
  `Settings`→`/focus/settings`, `Engine room`→`/_/`, theme toggle. **Already free
  of page links** — keep, just confirm.
- **Dead code** (orphaned by the Phase-4 head-chat-page retirement):
  - `renderError` (`internal/web/chat.go:110-121`) — has **no caller** (grep:
    only its own def + comment).
  - `execFragment` (`internal/web/chat.go:32-39`) — called **only** by
    `renderError`. Once `renderError` is gone, `execFragment` is dead too.
  - (`focus_page` is NOT dead — `focusPage` renders it via `h.render(e,
    "focus_page", …)` at `focus.go:151`. Do not remove it.)
- **Stale comments** (migration cruft, behavior is correct):
  - `internal/web/web.go:1`: `// Package web serves Balaur's HTMX interface…`
    (HTMX retired in 035f0e9 — it's Datastar now).
  - `web/templates/home.html:142`: `…basm.js calls showModal() after the HTMX
    swap…` (now triggered via Datastar `sse.ExecuteScript`).
  - `web/templates/cards.html:243`: `…Interactions stay htmx during coexistence;
    the board switch re-processes #main so the lazy partials bind.` (no
    coexistence; everything is Datastar/server-rendered).
- **Docs**: `DESIGN.md`, `README.md`, `internal/self/knowledge.md` were updated
  incrementally per phase; they need one holistic pass so the IA reads as
  card-first end-to-end (not page-by-page edits).

## Commands you will need
```bash
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
# Program completeness — no feature page route/template should remain:
grep -rnE 'h\.render\(e, "(tasks|journal|day|life|heads|head-chat|knowledge|settings|profile|models)\.html"' internal/web
grep -rniE 'htmx|hx-get|hx-post|hx-trigger|hx-swap' internal/web web/templates
```

## Scope

**In:** remove the dead `renderError` + `execFragment`; fix the 3 stale HTMX
comments (and any others the grep finds); confirm the topbar is final; a holistic
card-first IA pass over `DESIGN.md` (+ `README.md`/`internal/self/knowledge.md` if
they still describe pages); a final program-completeness verification; mark the
program complete in `plans/readme.md`.

**Out:** any behavior change; new features; touching working cards/focuses/dock.

## Git workflow
Branch `feature/card-first-kill-pages` (synced to `main` @ `06361b0`). Commit
after each green step.

## Steps

### Step A: remove the dead `renderError` + `execFragment`

**File:** `internal/web/chat.go` — delete both functions (and their doc comments):
- `execFragment` (`:32-39`)
- `renderError` (`:110-121`)

**Before deleting, re-confirm no callers** (must both print nothing but the defs):
```bash
grep -rn '\.renderError(\|\.execFragment(' internal/web --include='*.go'
```
If anything other than `renderError`'s internal call to `execFragment` shows up,
STOP — there is a live caller; report it.

After deletion, remove any import left unused (likely `"io"`, if `execFragment`
was its only user in `chat.go` — check `go build`). Confirm `messageView` is still
used elsewhere (it is, widely) — do not touch it.

**Verify:** `go build ./... && go vet ./... && go test ./internal/web/` → ok.
**Commit:** `git add internal/web/chat.go && git commit -m "refactor(web): drop dead renderError/execFragment (head-chat gateway retired)"`

### Step B: fix the stale HTMX comments

- `internal/web/web.go:1`: change `// Package web serves Balaur's HTMX interface:
  server-rendered html/template` → `// Package web serves Balaur's Datastar
  interface: server-rendered html/template` (keep the rest of the line/comment).
- `web/templates/home.html:142`: change `basm.js calls showModal() after the HTMX
  swap.` → `basm.js calls showModal() after the Datastar patch.` (or "after the
  model modal is patched in").
- `web/templates/cards.html:243`: the `ucard_knowledge_manage` comment about
  "Interactions stay htmx during coexistence; the board switch re-processes #main
  so the lazy partials bind." is obsolete — rewrite to describe the current
  reality (the manage card's actions are Datastar `@post`s that self-patch
  `#kcard-{id}`; no htmx, no coexistence), or trim it to the still-true part.
- Run `grep -rniE 'htmx|hx-get|hx-post|hx-trigger|hx-swap' internal/web web/templates`
  and fix any OTHER stale mention you find (comments only — there should be no
  live `hx-*` attributes). If a live `hx-*` attribute exists, STOP and report (it
  would mean an un-migrated surface).

**Verify:** `go build ./... && go test ./internal/web/` → ok;
`grep -rniE 'htmx|hx-(get|post|trigger|swap)' internal/web web/templates` → only
benign/none.
**Commit:** `git add internal/web/web.go web/templates/home.html web/templates/cards.html && git commit -m "docs: retire stale HTMX comments (Datastar everywhere now)"`

### Step C: confirm the topbar is final (likely no change)

Read `web/templates/layout.html` `{{define "topbar"}}`. It should be: brand→`/`,
`Boards`→`/boards`, `Settings`→`/focus/settings`, `Engine room`→`/_/`, theme
toggle — no feature-page links. This is the intended final nav (features are
reached via boards, the `/ui/cards` palette/launcher, or chat; Settings keeps a
convenience shortcut to its focus). **If it already matches, make no change** and
note it. If any retired-page link survives, re-point it to its `/focus/*` or
remove it.

**Verify:** `grep -nE 'href="/(tasks|journal|day|memory|skills|life|heads|models|settings|profile)("|/)' web/templates/layout.html | grep -v '/focus/'` → nothing.
(No commit if no change.)

### Step D: holistic card-first IA pass over the docs

Read the UI/architecture sections of `DESIGN.md` (and skim `README.md`,
`internal/self/knowledge.md`). The per-phase edits made them *locally* correct but
possibly *globally* uneven. Make ONE coherent pass so the IA reads as card-first
end to end. The single source of truth to convey:

- The UI is **boards + cards + a persistent dock chat** — there are **no feature
  pages**.
- A **card** is a typed, parameterized, server-rendered HATEOAS resource
  (`/ui/cards/{type}`) that renders at two sizes: a **tile** on a board, and a
  full-canvas **focus** (`/focus/{type}`) reached by expanding it — the focus is
  what the old page was.
- **Boards** are owner-composed dashboards of cards (drag/resize/persisted); the
  `/ui/cards` palette is the **launcher** (open any feature in focus); chat can
  also compose cards/boards (`card_show`/`board_compose`).
- The **dock chat** is the single persistent chat; it **swaps conversations**
  (master ↔ a head's branch) without leaving the dock.
- Retired routes (`/tasks`, `/journal`, `/day`, `/memory`, `/skills`, `/life`,
  `/heads`, `/heads/{id}/chat`, `/models`, `/settings`, `/profile`) **302 →
  `/boards`**; their write endpoints (`/ui/*`) live on, now driving card focuses.

Fix any remaining prose that still describes a feature as a "page". Keep edits
tight and truthful — do not invent UI that doesn't exist.

**Verify:** `grep -rniE '/(tasks|journal|memory|skills|life|heads|models|settings|profile) page|the .* page' DESIGN.md README.md internal/self/knowledge.md` →
review each hit; none should describe a live feature *page*. `go test ./internal/self/...`
(knowledge.md is embedded) → ok.
**Commit:** `git add DESIGN.md README.md internal/self/knowledge.md && git commit -m "docs: holistic card-first IA pass — boards + cards + dock, no feature pages"`

### Step E: final program verification + mark complete

Run the whole-program checks and confirm:
```
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
git diff --check
# No feature page render survives:
grep -rnE 'h\.render\(e, "(tasks|journal|day|life|heads|head-chat|knowledge|settings|profile|models)\.html"' internal/web   # → nothing
# The deleted page templates are gone:
ls web/templates/ | grep -E '^(tasks|journal|day|life|heads|head-chat|knowledge|settings)\.html$'                          # → nothing
```

Update the `057` row in `plans/readme.md` → DONE, and add a one-line
**program-complete** note to the Eighth-cycle section (e.g. "Plans 050–057 DONE
& merged — the card-first 'kill the pages' program is complete: UI is boards +
cards + dock chat, every feature is a card focus, no feature pages remain").

**Commit:** `git add plans/readme.md && git commit -m "docs(plans): 057 done — card-first program complete (050–057)"`

## Test plan
- **No regression**: `go test ./...` green after each step (dead-code removal +
  comment/doc edits change no behavior).
- **Completeness greps** (Step E): no feature page render, no page template, no
  live `hx-*` attribute.
- **Docs truth**: knowledge.md is embedded → `internal/self` tests pass.

## Done criteria
- [ ] `renderError` + `execFragment` removed; no caller existed; build/vet clean.
- [ ] The 3 stale HTMX comments fixed; no live `hx-*` attribute anywhere; only
      Datastar.
- [ ] Topbar confirmed final (boards + settings-focus + engine room + theme; no
      feature-page links).
- [ ] `DESIGN.md` (+ README/knowledge.md) describe the card-first IA coherently;
      no live feature is called a "page".
- [ ] Step E completeness greps return nothing; `go test ./...`, vet, `gofmt -l`
      (empty), CGO-free build clean; `git diff --check` clean.
- [ ] `plans/readme.md` 057 → DONE + program-complete note.

## STOP conditions
- A live caller of `renderError`/`execFragment` exists → do not delete; report it.
- A live `hx-*` attribute (not a comment) exists in a template → an un-migrated
  surface survived; STOP and report.
- A feature page route or `h.render(…"X.html")` for a retired page still exists →
  the program isn't actually complete; STOP and report which one.

## Maintenance notes
- Two known low-priority dock follow-ups from Phase 4 remain (NOT in this plan,
  tracked in memory): a silent 403 if a head expires mid-dock-chat; back-to-main
  mid-stream can drop head-styled tokens into master `#chat`. They are UX
  edge-cases, not correctness bugs — leave for a future pass.
- After this plan, the `feature/card-first-kill-pages` branch == `main`; the
  program (050–057) is complete.
