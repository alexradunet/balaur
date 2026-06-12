# Plan 028: Typed card registry — every card a parameterized server resource (HATEOAS foundation)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 9c77f42..HEAD -- internal/web internal/cards web/templates`
> Plans 025–027 being DONE is expected drift. Compare any other in-scope
> change against "Current state"; on mismatch, STOP.

## Status

- **Priority**: P1 (it is the foundation for boards and agent-composed UI)
- **Effort**: L
- **Risk**: MED (broad read surface; no mutations)
- **Depends on**: plans/025-hearthwood-visual-foundation.md (card styling)
- **Category**: direction
- **Planned at**: commit `9c77f42`, 2026-06-12

## Why this matters

The owner's goal: Balaur composes UI on the spot — "show my weight", "set up a
board for the trip" — without the model ever authoring HTML. The mechanism is
HATEOAS: every card type is a **parameterized server resource** at
`GET /ui/cards/{type}?params`, rendered server-side from PocketBase data, with
its affordances (links, forms) baked into the HTML the server returns. Boards
(plan 029) are lists of card references; agent tools (plan 030) compose them.
This plan builds the registry and the card endpoints. The decision is settled:
typed composition only — the model picks types and parameters from a fixed
registry; it never emits markup (owner decision, 2026-06-12).

## Current state

- **Route registration**: `internal/web/web.go:163` `func Register(se
  *core.ServeEvent) error`; routes bound on `se.Router` after the
  `guardLocalUI` BindFunc (web.go:172) — every new `/ui/...` route gets the
  Origin/Host guard for free. Existing card-ish fragments:
  `GET /ui/tasks/{id}/card` (web.go:199), `GET /ui/knowledge/{kind}/{id}/card`
  (web.go:203), `GET /ui/knowledge/{kind}/grid` (web.go:202).
- **Domain data access** (AGENTS.md): domain packages own their PocketBase
  reads — `internal/tasks`, `internal/life`, `internal/knowledge`,
  `internal/conversation`, `internal/heads`, `internal/recap`. The web
  handlers already build view models from them (e.g. `taskView` in
  `internal/web`, sparkline data for `/life`, calendar/timeline data for
  `/tasks` views, journal/day data for `/day/{date}`). **Reuse those query
  paths; do not duplicate queries in a new layer when an existing function can
  be called or cheaply extracted.**
- **Card markup vocabulary** (post-025 Hearthwood, `web/static/basm.css`):
  `.kcard` parchment card with gold notch; `.kcard-kind` mono teal kicker;
  `.k-grid` auto-fill grid; `.h-icon`/`img.tool-icon` pixel icons;
  `.card-note-error` for error strips. An exemplar card template:
  `web/templates/card-task.html` (excerpt):

  ```html
  <article class="kcard tcard tcard-{{.Status}}" id="tcard-{{.ID}}">
    <header class="kcard-head"><span class="kcard-kind">▪ task</span>…</header>
    <h3 class="kcard-title">{{.Title}}</h3>
    …
  </article>
  ```

- **The mockup's card-type vocabulary** (`Balaur_ds/Balaur App.dc.html:169`,
  `CARD_TYPES`): character, today, campaigns, road, quests, calendar,
  timeline, journal, identity, souls, stats, weight, mood, ledger, companion,
  skills, memory, heads — each with an icon and a default grid width (4 or 8
  of a 12-col grid) and height. This plan ships the subset that maps cleanly
  onto shipped Balaur domains (see Step 1); the rest are deferred.
- **Import direction constraint**: `internal/web` imports `internal/tools`
  (web.go uses `tools.ParseProposal`), so `internal/tools` can NEVER import
  `internal/web`. Plan 030's agent tools must validate card types/params —
  therefore the registry's **specs** (names, params, validation) live in a new
  leaf package `internal/cards` with no web imports, and `internal/web` owns
  the HTML rendering against those specs.

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./...`                  | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`    | clean               |

## Scope

**In scope**:
- `internal/cards/cards.go`, `internal/cards/cards_test.go` (create — specs only)
- `internal/web/cards.go`, `internal/web/cards_test.go` (create — handlers/views)
- `internal/web/web.go` (route registration + any small extraction of an
  existing view-model builder into a reusable function within the same package)
- `web/templates/cards.html` (create — one `{{define}}` per card type + shared
  shell)
- `internal/self/knowledge.md`, `DESIGN.md` honesty ledger (one line each)

**Out of scope**:
- Boards (plan 029), agent tools (plan 030).
- Any mutation endpoint — all card endpoints are GET, read-only.
- Restructuring existing pages to consume cards (pages stay as they are; cards
  are a parallel composition surface for boards/chat).
- New collections or migrations.

## Git workflow

- Branch: `advisor/028-card-registry`
- Commit style: `feat(cards): typed card registry + /ui/cards/{type} endpoints`

## Steps

### Step 1: Specs package `internal/cards`

```go
package cards

type ParamSpec struct {
    Name     string // query/JSON key
    Required bool
    Enum     []string // optional closed set
    Doc      string   // one line, model- and owner-facing
}

type Spec struct {
    Type   string // "today", "quests", …
    Label  string // "Today"
    Icon   string // icon file stem under /static/icons
    W      int    // default grid span (of 12) — from the mockup's CARD_TYPES
    Params []ParamSpec
}

func All() []Spec
func Get(typ string) (Spec, bool)
// Validate checks unknown keys, missing required keys, enum membership.
// Returns a cleaned param map (only known keys) or an error.
func Validate(typ string, params map[string]string) (map[string]string, error)
```

Ship these 10 types (v1 — each maps to a shipped domain):

| Type | Label | Icon | W | Params |
|---|---|---|---|---|
| `today` | Today | scroll | 4 | — (open tasks due/overdue today) |
| `quests` | Quest log | scroll | 8 | `status` enum open/done/all (default open), `limit` (default 10) |
| `calendar` | Calendar | hourglass | 4 | `month` YYYY-MM (default: current) |
| `timeline` | Timeline | hourglass | 8 | `days` (default 14, max 31) |
| `journal` | Journal | quill | 4 | `limit` (default 5) |
| `measure` | Measure | orb | 4 | `kind` REQUIRED (a numeric life-entry kind), `days` (default 90) |
| `lines` | Recent lines | orb | 4 | `kind` REQUIRED (a text life-entry kind), `limit` (default 5) |
| `memory` | Memory | tome | 4 | `query` (optional search), `limit` (default 6) |
| `skills` | Skills | key | 4 | `limit` (default 6) |
| `heads` | Heads | tome | 4 | — |

Numeric params: parse with strconv, clamp to sane bounds (limit ≤ 50,
days ≤ 366), never error on a bad number — fall back to the default
(cards must be forgiving; the agent composes these).
`kind` for measure/lines cannot be enum-validated statically (kinds are
owner-defined) — Validate only checks presence; the renderer handles
"no such kind" as an empty-state card, not an error.

**Verify**: `go test ./internal/cards/...` → ok (tests in test plan).

### Step 2: Card templates `web/templates/cards.html`

A shared shell + one body define per type. Shell pattern:

```html
{{define "ucard_shell_open"}}
<article class="kcard ucard ucard-{{.Type}}">
  <header class="kcard-head">
    <span class="kcard-kind"><img class="tool-icon" src="/static/icons/{{.Icon}}.png" alt="">{{.Label}}</span>
    {{with .ParamLine}}<span class="kcard-meta">{{.}}</span>{{end}}
  </header>
{{end}}
```

(If nested define-composition gets awkward in `html/template`, it is fine to
repeat the header block per card define — clarity beats cleverness here;
match how `card-task.html` is written.)

Card bodies are **compact read views + affordance links** (HATEOAS: each card
links to the resource that owns it):

- `today`/`quests`: list of task rows — title + due line (reuse `taskView`
  fields); each row's Done action is a small form posting to the EXISTING
  `/ui/tasks/{id}/transition` with `hx-target="closest li" hx-swap="delete"`
  (the full re-rendered card response is discarded; deletion of the row is the
  visible effect — simplest correct reuse). Footer: `<a href="/tasks">all quests →</a>`.
- `calendar`: reuse the month-table markup pattern from `tasks_calendar` in
  `web/templates/tasks.html` (extract or re-render compactly; day cells link
  to `/day/{date}`). Footer link `/tasks?view=calendar`.
- `timeline`: compact `tl-items` list from the same data the timeline view
  uses. Footer link `/tasks?view=timeline`.
- `journal`: last N `journal-entry` snippets (clipped to ~200 chars), footer
  link to today's `/day/{date}`.
- `measure`: the `life.html` sparkline pattern (`.spark` SVG polyline +
  `.life-stat`) for the given kind. Footer link `/life`.
- `lines`: `.life-lines` recent text entries for the kind. Footer link `/life`.
- `memory`/`skills`: titles + kind/importance pips of active records (reuse
  field names from `card-memory.html`/`card-skill.html`), each title links to
  the owning page; NOT the full editable cards. Footer link `/memory` or
  `/settings/skills`.
- `heads`: name + status tag per active head, linking to `/heads/{id}/chat`.

Empty states: every card renders a `.k-empty` line ("nothing here yet") rather
than collapsing.

**Verify**: `go test ./internal/web/...` → templates parse.

### Step 3: Handlers + route

`internal/web/cards.go`:

```go
// GET /ui/cards/{type}?…  → one rendered card fragment
func (h *handlers) uiCard(e *core.RequestEvent) error {
    typ := e.Request.PathValue("type")
    spec, ok := cards.Get(typ)
    if !ok { return e.NotFoundError("no such card type", nil) }
    params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
    if err != nil { /* render card-note-error fragment, 200 */ }
    switch typ { case "today": …build view, render "ucard_today"… }
}

// GET /ui/cards → the palette: a small HTML index of all specs with labels,
// icons, and param docs (used by the board palette in plan 029 and handy for
// a human exploring with curl).
```

Register both in `Register()` next to the other `/ui/` GETs. For each card's
data needs, call the existing domain/query code; where a page handler inlines
a query you need (e.g. the calendar month build), extract it to an unexported
function in the same file it lives in, then call it from cards.go — extraction
must be a pure move, no behavior change.

**Verify**:
```sh
go run . serve &  # or make run
curl -s "http://127.0.0.1:8090/ui/cards/today" | grep -c "kcard"        # ≥1
curl -s "http://127.0.0.1:8090/ui/cards/measure" | grep -c "card-note"  # ≥1 (missing required kind)
curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:8090/ui/cards/nope"  # 404
```

### Step 4: Docs

- `internal/self/knowledge.md`: one short paragraph — Balaur's UI has typed
  cards at `/ui/cards/{type}`; list the 10 types.
- `DESIGN.md` §3 "True today": "typed card registry — 10 parameterized,
  server-rendered card resources under `/ui/cards/{type}` (the composition
  unit for boards and on-the-spot UI)".

**Verify**: `grep -n "ui/cards" internal/self/knowledge.md DESIGN.md` → ≥1 each.

## Test plan

- `internal/cards/cards_test.go` (table-driven): Get on every type in All();
  Validate rejects unknown type, missing required `kind`, bad enum value;
  Validate drops unknown keys; numeric clamping (limit=999 → 50).
- `internal/web/cards_test.go` (model after `handlers_test.go`, which uses the
  `internal/store` temp-dir app helpers): for each of the 10 types, GET the
  endpoint with valid params on a seeded app → 200 and body contains
  `ucard-{type}`; unknown type → 404; `measure` without kind → 200 +
  `card-note-error`; `today` with one seeded open task → body contains the
  task title and a form posting to `/ui/tasks/{id}/transition`.
- Verification: `go test ./...` → all pass.

## Done criteria

- [ ] `curl /ui/cards/{type}` works for all 10 types (handler tests prove it)
- [ ] `internal/cards` has zero imports of `internal/web` (`go list -deps` or
      just read the imports) and compiles standalone
- [ ] All card endpoints are GET-only and read-only (no `app.Save` in
      `internal/web/cards.go`: `grep -c "Save(" internal/web/cards.go` → 0)
- [ ] `go test ./...`, vet, fmt, CGO-free build clean; `git diff --check` clean
- [ ] knowledge.md + DESIGN.md updated; `plans/readme.md` row updated

## STOP conditions

- A card's data needs cannot be met by calling/extracting existing query code
  and would require new domain logic beyond ~30 lines — report which type;
  shipping 8 of 10 types is better than inventing a parallel query layer.
- The existing page handlers' view-model builders turn out to be entangled
  with request state in a way that resists pure-move extraction.
- You find yourself wanting a JSON API or client-side rendering — wrong
  direction; the contract is HTML fragments only.

## Maintenance notes

- The registry is the single source of card truth: plan 029 reads `All()` for
  the palette and grid spans; plan 030 validates agent args with `Validate()`.
  Adding a card type = one Spec + one template define + one handler case +
  one test row.
- Deferred mockup types: `character`, `identity`, `souls`, `companion`
  (profile-flavored), `campaigns`, `road`, `stats`, `mood`, `weight` (the
  latter three are `measure`/`lines` with a preset kind — add as aliases only
  if the owner asks).
- Reviewer: parameter handling is the attack/robustness surface — confirm
  clamps, the forgiving-fallback rule, and that `query` for memory search
  reuses the existing search path (no new LIKE injection surface; PocketBase
  filter params must be bound, not concatenated).
