# Plan 030: Agent UI tools — Balaur composes cards and boards on the spot (HATEOAS)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 9c77f42..HEAD -- internal/tools internal/turn internal/web internal/cards internal/self/knowledge.md`
> Plans 025–029 being DONE is expected drift (027 in particular touched the
> same chat.go branch and marker file this plan extends — read their final
> shapes first). Anything else → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (agent-triggered record writes; marker protocol growth)
- **Depends on**: plans/028-typed-card-registry.md, plans/029-boards.md (hard);
  plans/027-dialogue-choices.md (soft — shares the marker seam; land 027 first
  to avoid conflicts in chat.go and to inherit its Mark/Parse refactor note)
- **Category**: direction
- **Planned at**: commit `9c77f42`, 2026-06-12

## Why this matters

This is the payoff of the HATEOAS stack: the owner says "show my weight" or
"set up a board for the trip" in chat, and Balaur renders the right card
inline or raises a whole board — by **choosing typed, parameterized server
resources from the registry**, never by writing markup. Two tools:
`card_show` (render one card into the conversation) and `board_compose`
(create a board of cards). The model's entire expressive surface is
`{type, params}` validated by `internal/cards`; the server renders everything.

## Current state

- **Registry** (plan 028): `internal/cards` — `All()`, `Get(typ)`,
  `Validate(typ, params) (map[string]string, error)`; HTML at
  `GET /ui/cards/{type}?params`. `internal/cards` is a leaf package —
  importable from `internal/tools` (which must never import `internal/web`).
- **Boards** (plan 029): `boards` collection
  (`name` text, `cards` json `[{"type":…,"params":{…}}]`, `sort` number);
  page at `/boards/{id}`. Validation helper for card lists lives in
  `internal/web/boards.go` as of 029 — see Step 2 for the required move.
- **Marker seam**: `internal/tools/knowledge.go:28-52` defines
  `ProposalMarker`/`MarkProposal`/`ParseProposal`
  (`"\x00balaur-proposal:" + kind + "/" + id + "\n" + modelText`); plan 027
  added `ChoicesMarker`. The web consumer is the `tool_result` case in
  `internal/web/chat.go:80-93`, which turns a parsed proposal into
  `messageView{Content: rest, CardURL: cardURL(kind, id)}`; the
  `chat-msg-tool-end` fragment then lazy-embeds the card:

  ```html
  {{if .CardURL}}<div class="k-inline" hx-get="{{.CardURL}}" hx-trigger="load" hx-swap="innerHTML"></div>{{end}}
  ```

  and `cardURL` (chat.go:120-125) maps kind→URL:

  ```go
  func cardURL(kind, id string) string {
      if kind == "tasks" { return "/ui/tasks/" + id + "/card" }
      return "/ui/knowledge/" + kind + "/" + id + "/card"
  }
  ```

- **Tool registration**: `internal/turn/tools.go:21-35` appends tool sets in
  `Tools(app)`.
- **Audit convention** (AGENTS.md): task tools audit every mutation to
  `audit_log` — find the helper they use (`grep -n "audit" internal/tools/tasks.go`)
  and reuse it for board creation. `card_show` mutates nothing → no audit
  record needed (matches `offer_choices`' reasoning in plan 027).
- **Existing inline-embed plumbing is sufficient**: a `card_show` result just
  needs `CardURL=/ui/cards/{type}?{query}` — the `k-inline` fragment and the
  `.chat .k-inline` CSS (gutter-aligned, full message width) already handle
  presentation.

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./...`                  | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`    | clean               |

## Scope

**In scope**:
- `internal/tools/ui.go`, `internal/tools/ui_test.go` (create)
- `internal/cards/cards.go` (add `ValidateCards` — see Step 2)
- `internal/turn/tools.go` (register)
- `internal/web/chat.go` (URL-kind handling in the marker consumer)
- `internal/web/boards.go` (switch to the moved validator; small)
- `internal/self/knowledge.md`, `DESIGN.md` honesty ledger

**Out of scope**:
- New card types, board page changes, template changes (the existing
  `k-inline` embed path carries everything).
- Free-form/model-authored HTML cards — REJECTED by owner decision
  (2026-06-12); if you are tempted, stop.
- Head chats (tool-free by design).

## Git workflow

- Branch: `advisor/030-agent-ui-tools`
- Commit style: `feat(tools): card_show + board_compose — agent-composed UI from the typed registry`

## Steps

### Step 1: Marker for URL-addressed cards

The existing proposal marker carries `kind/id` (record-addressed). Typed cards
are URL-addressed (`type` + query). Extend the seam in
`internal/tools/knowledge.go` — or better, if plan 027's maintenance note was
acted on, the shared marker helper file — with:

```go
const UICardMarker = "\x00balaur-uicard:"

// MarkUICard: marker + type + "?" + url.Values.Encode(params) + "\n" + modelText
func MarkUICard(typ string, params map[string]string, modelText string) string
func ParseUICard(s string) (typ, query, rest string, ok bool)
```

Three marker kinds now exist (proposal, choices, uicard) — per plan 027's
note, collapse the three Mark/Parse pairs into one shared
`markPayload(marker, head, text)` / `parsePayload(marker, s)` internal helper
if the duplication is mechanical (SUCKLESS), keeping the three public APIs.

**Verify**: `go test ./internal/tools/...` → ok (round-trip tests).

### Step 2: Move card-list validation into the leaf package

Plan 029 left `validateBoardCards` in `internal/web/boards.go` and flagged
this move. Add to `internal/cards`:

```go
type Card struct {
    Type   string            `json:"type"`
    Params map[string]string `json:"params,omitempty"`
}
// ValidateCards validates each entry via Validate and returns the cleaned list.
func ValidateCards(cs []Card) ([]Card, error)
```

Update `internal/web/boards.go` to use it (pure move + call-site swap; delete
the old helper).

**Verify**: `go test ./internal/web/... ./internal/cards/...` → ok; behavior
identical (boards tests from 029 still green).

### Step 3: The tools (`internal/tools/ui.go`)

Follow the `agent.Tool` pattern from `internal/tools/knowledge.go:54-66`
(`ToolSpecOf` + `obj(...)` schema helpers + `Execute` closure).

**`card_show`** — "Render a live UI card into the conversation. Choose a type
from the registry; the server renders it from the owner's real data." Args:
`type` (string, required — enumerate the valid types in the description by
ranging over `cards.All()` at registration time, so the model sees the real
vocabulary and each type's params/docs), `params` (object of string values,
optional). Execute: `cards.Validate`; on error return the error text as the
tool result (the model self-corrects — do NOT return a Go error for bad
params); on success return
`MarkUICard(typ, cleaned, "showing the owner the <label> card")`.

**`board_compose`** — "Create a new board of cards for the owner (e.g. 'set
up a board for the trip'). The board appears at /boards." Args: `name`
(string, required, ≤80 chars), `cards` (array of `{type, params}`, required,
1–8 items). Execute: `cards.ValidateCards`; create the `boards` record
directly via the PocketBase DAO (domain write in the tools package — match how
`taskAddTool` in `internal/tools/tasks.go:58` creates task records, including
its audit_log write; reuse the same audit helper with action like
`board_compose`); return
`MarkUICard("__board", map{"id": rec.Id}, "board raised: <name> — find it at /boards/<id>")`…

…actually keep it simpler and more honest: `board_compose` does not need an
inline card embed. Return PLAIN text:
`"board raised: <name> (<n> cards) — /boards/<id>"`. The tool row shows it,
the model narrates it, the owner clicks through. Only `card_show` uses the
marker. (KISS; a board-summary card endpoint can come later if wanted.)

Register both in `internal/turn/tools.go`:
`ts = append(ts, tools.UITools(app)...)`.

**Verify**: `go test ./internal/tools/...` → ok.

### Step 4: Web consumer

In `internal/web/chat.go`'s `tool_result` case, before `ParseProposal`:

```go
if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
    mv = messageView{Content: rest, CardURL: "/ui/cards/" + typ + "?" + query}
}
```

(Adjust to the branch's post-027 shape.) The history path: find where
page-load history applies `ParseProposal` (in `internal/web/web.go`) and add
the same — unlike choices, an embedded card is safe and useful to re-render on
reload (it lazy-fetches current data).

**Verify**: handler test (test plan) + manual: "show me my open quests" →
tool row then a live quests card inline; "set up a board for my health" →
tool row with the /boards link; reload → card re-renders.

### Step 5: Docs

- `internal/self/knowledge.md`: describe both tools and the composition rule
  (typed registry only — Balaur cannot author markup).
- `DESIGN.md` "True today": "on-the-spot UI — `card_show` embeds any typed
  card inline in chat; `board_compose` raises a new board from chat".

**Verify**: `grep -n "card_show\|board_compose" internal/self/knowledge.md DESIGN.md` → ≥1 each.

## Test plan

- `internal/tools/ui_test.go` (table-driven, model after `tasks_test.go` which
  uses temp-dir app helpers): `card_show` happy path returns UICardMarker with
  encoded params; unknown type returns a plain-text error result (not a Go
  error); `board_compose` creates a record with validated cards + an audit_log
  record; 0 and 9 cards rejected; name >80 rejected.
- `internal/web`: chat handler test — fake LLM triggers `card_show`; response
  contains `hx-get="/ui/cards/quests?` inside a `k-inline` div; history reload
  contains it too.
- Verification: `go test ./...` → all pass.

## Done criteria

- [ ] Both tools registered (`grep -n "UITools" internal/turn/tools.go` → 1)
- [ ] `card_show` bad-params returns model-facing text, never a Go error
      (asserted by test)
- [ ] `board_compose` writes audit_log (asserted by test)
- [ ] `internal/tools` does not import `internal/web`
      (`grep -rn "internal/web" internal/tools/` → no matches)
- [ ] `go test ./...`, vet, fmt, CGO-free build clean; `git diff --check` clean
- [ ] knowledge.md + DESIGN.md updated; `plans/readme.md` row updated

## STOP conditions

- 028/029 interfaces differ from "Current state" (read their landed code
  first; this plan was written before they executed).
- The audit helper in tasks.go is not reusable from a new file without
  refactoring beyond a pure move.
- Anything pushes toward the model emitting HTML/CSS or arbitrary URLs —
  rejected direction; the only model-controlled strings that reach a URL are
  `type` (validated against the registry) and `params` (validated +
  URL-encoded). If you find a path where unvalidated model output lands in an
  `hx-get` attribute, stop and report.

## Maintenance notes

- The tool descriptions embed the registry vocabulary at registration time —
  when card types are added (plan 028's maintenance note), the model's
  vocabulary updates for free. Verify this in review (no hardcoded type list
  in the description string).
- The marker protocol now has three kinds; the consumer order in chat.go is
  uicard → choices → proposal → plain. Document that order with a comment at
  the consumer.
- Follow-ups deliberately deferred: a board-summary card type (`__board` embed
  for compose confirmations); `board_add_card` tool for amending existing
  boards ("add my weight to the trip board") — small once this lands; spec on
  request.
- Reviewer: the `hx-get` URL construction in Step 4 — confirm `query` comes
  only from `url.Values.Encode` server-side and is attribute-escaped by
  html/template.
