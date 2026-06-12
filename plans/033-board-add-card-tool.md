# Plan 033: `board_add_card` — amend an existing board from chat

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. Commit on branch
> `advisor/033-board-add-card`. SKIP updating `plans/readme.md` (reviewer
> maintains it). Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 90bc397..HEAD -- internal/tools internal/turn internal/cards internal/self/knowledge.md`
> Plan 032 may be landing concurrently in another worktree (it extends
> `cards.Card` with optional layout fields x/y/w/h — your code must not
> assume Card has ONLY Type+Params; treat extra fields as opaque). Any other
> drift → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P3 · **Effort**: S · **Risk**: LOW
- **Depends on**: 030 (DONE, merged); 032 lands in parallel (no file overlap)
- **Category**: direction · **Planned at**: commit `90bc397`, 2026-06-12

## Why this matters

`board_compose` (plan 030) creates whole boards, but "add my weight to the
trip board" — amending an existing board — still requires the page. This tool
closes that gap with the same typed-composition discipline.

## Current state

- `internal/tools/ui.go` holds `card_show`, `board_compose`, the
  `UICardMarker` trio, and `UITools(app)` (registered in
  `internal/turn/tools.go:27`). `board_compose` already: validates via
  `cards.ValidateCards`, saves a `boards` record (fields `name` text,
  `cards` json, `sort` number), audits via
  `store.Audit(app, headID, actor, action, target, allowed, detail)` with
  actor `"agent"`, returns plain text with the `/boards/{id}` link.
  **Read its Execute closure first and mirror its conventions exactly.**
- `internal/tools/ui_test.go` has the temp-dir-app test pattern for
  `board_compose` (record + audit assertions). Model the new tests on it.
- Validation errors return model-facing text results, never Go errors.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |

## Scope

**In scope**: `internal/tools/ui.go`, `internal/tools/ui_test.go`,
`internal/self/knowledge.md`, `DESIGN.md` ledger.
**Out of scope**: everything else — no web changes (the board page already
re-renders from the record), no new routes, no registry changes,
no `internal/web`.

## Git workflow

Branch `advisor/033-board-add-card`; commit
`feat(tools): board_add_card — amend an existing board from chat`.

## Steps

### Step 1: The tool

Add `boardAddCardTool(app)` to `internal/tools/ui.go` and append it in
`UITools`. Spec: name `board_add_card`, description "Add one typed card to an
existing board (e.g. 'add my weight to the trip board'). Use board_compose to
create a new board instead." Args:
- `board` (string, required) — board name or id. Resolution: exact id match
  first; else case-insensitive exact name match; else case-insensitive
  substring match. Zero matches → return text listing existing board names
  ("no board matches %q — boards: …"). More than one match → return text
  naming the ambiguity. Never a Go error.
- `type` (string, required) + `params` (object, optional) — validate via
  `cards.Validate` exactly like `card_show`; bad input → model-facing text.

Execute: load the board record, decode its `cards` json into `[]cards.Card`
(preserving any fields you don't understand — decode into `cards.Card`, which
may carry layout fields if plan 032 landed; appending a new entry never
touches existing entries), append `{Type, Params}`, re-encode, save. Audit:
`store.Audit` with action `board_add_card`, target the board id, detail
including the card type. Return plain text:
`"added <label> to <board name> — /boards/<id>"`.

**Verify**: `go test ./internal/tools/...` → ok.

### Step 2: Docs

knowledge.md: extend the agent-UI-tools sentence with board_add_card.
DESIGN.md "True today" on-the-spot UI clause: append "; `board_add_card`
amends an existing board".

**Verify**: `grep -n "board_add_card" internal/self/knowledge.md DESIGN.md` → ≥1 each.

## Test plan

`internal/tools/ui_test.go`, table-driven where natural: happy path (record
gains one entry, others byte-identical, audit row written); resolve by name
case-insensitive; resolve by substring; ambiguous name → text result listing;
unknown board → text result listing boards; invalid card type → text result;
existing entries with layout fields survive append unchanged (seed a card
entry whose json includes `"x":2,"w":4` and assert those keys survive —
decode-append-encode must not strip them).

## Done criteria

- [ ] Tool registered inside `UITools` (no new registration line in turn)
- [ ] All resolution/validation failures are text results, never Go errors
      (tests prove)
- [ ] Audit row asserted; layout-preservation test passes
- [ ] All gates clean; only the four in-scope files changed (`git status`)

## STOP conditions

- `board_compose`'s conventions in the landed code differ materially from
  this plan's description.
- Preserving unknown json fields through decode/append/encode is impossible
  with `cards.Card` (would force a `map[string]any` detour) — report rather
  than silently dropping fields.

## Maintenance notes

- If plan 032 lands first, `cards.Card` already has X/Y/W/H — the
  preservation test then exercises real fields. If 032 lands second, the
  test's seeded `"x"` key is still preserved only if Card has the fields;
  coordinate at merge: whichever lands second re-runs the other's tests.
- Reviewer: the substring matcher must not panic on empty `board` arg.
