# Plan 043: Stop swallowing board cards JSON errors — corrupt data must abort, not erase

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat dd9e60b..HEAD -- internal/web/boards.go internal/web/boards_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `dd9e60b`, 2026-06-12

## Why this matters

Boards store their card list as a JSON string in the `boards` collection's
`cards` field. Every handler that reads it discards the `json.Unmarshal`
error, and every mutating handler discards the `json.Marshal` error on
write-back. The dangerous path: if a board's `cards` JSON is ever corrupt
(hand-edited via the PocketBase dashboard, a partial write, a future bug),
`boardsCardAdd` unmarshals into an **empty slice**, appends the new card,
and saves — silently replacing the owner's whole board with one card. The
owner composed that board by hand or via agent tools; erasing it without an
error is data loss.

## Current state

One file owns all of this: `internal/web/boards.go` (530 lines) — board
CRUD handlers + the `boardRecord` view-model builder. Its JSON sites
(line numbers at `dd9e60b`):

| Line | Function | Today |
|------|----------|-------|
| 120  | `boardRecordOf` (read path, page render) | `_ = json.Unmarshal(...)` |
| 206  | `ensureDefaultBoards` | `raw, err := json.Marshal(d.cards)` — **already checked**; this is the in-file exemplar |
| 387  | `boardsCardAdd` | `_ = json.Unmarshal(...)` → append → save (the data-loss path) |
| 395  | `boardsCardAdd` | `raw, _ := json.Marshal(bcs)` |
| 431  | `boardsCardRemove` | `_ = json.Unmarshal(...)` (bounds check downstream limits damage) |
| 438  | `boardsCardRemove` | `raw, _ := json.Marshal(bcs)` |
| 496  | `boardsLayout` | `_ = json.Unmarshal(...)` (entry-count check downstream limits damage) |
| 522  | `boardsLayout` | `rawCards, _ := json.Marshal(cleaned)` |

Excerpt — `internal/web/boards.go:118-124` (read path):

```go
func boardRecordOf(rec *core.Record) *boardRecord {
	var bcs []boardCard
	_ = json.Unmarshal([]byte(rec.GetString("cards")), &bcs)
	views, freeLay := boardCardViewsOf(bcs)
	return &boardRecord{
		ID:      rec.Id,
		Name:    rec.GetString("name"),
```

Excerpt — `internal/web/boards.go:386-399` (the data-loss path in
`boardsCardAdd`; `rec` was loaded by `FindRecordById` a few lines up):

```go
	var bcs []boardCard
	_ = json.Unmarshal([]byte(rec.GetString("cards")), &bcs)

	newCard := boardCard{Type: typ}
	if len(cleaned) > 0 {
		newCard.Params = cleaned
	}
	bcs = append(bcs, newCard)

	raw, _ := json.Marshal(bcs)
	rec.Set("cards", string(raw))
	if err := h.app.Save(rec); err != nil {
		return e.InternalServerError("saving board", err)
	}
```

Error-response conventions in this file (match them):

- Bad input → `return e.BadRequestError("message", err)`
- Server-side failure → `return e.InternalServerError("message", err)`

An **empty** `cards` field is normal (new board): `json.Unmarshal` of `""`
returns an error ("unexpected end of JSON input") — so blank must be
treated as "no cards", NOT as corruption. This is the one subtlety.

Repo conventions: standard Go; wrap errors `fmt.Errorf("doing x: %w", err)`;
no panics; comments explain constraints only. Tests for this file live in
`internal/web/boards_test.go` (477 lines) — model new tests on its existing
handler tests (they use `newWebApp` from `handlers_test.go` and seed
`boards` records directly via `core.NewRecord` + `app.Save`).

## Commands you will need

| Purpose   | Command                        | Expected on success |
|-----------|--------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...` | exit 0              |
| Tests     | `go test ./internal/web/`      | ok                  |
| All tests | `go test ./...`                | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`  | silent / empty      |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify):
- `internal/web/boards.go`
- `internal/web/boards_test.go`

**Out of scope** (do NOT touch, even though they look related):
- `internal/cards/cards.go` — validation layer is correct; nothing changes.
- `internal/tools/ui.go` (`board_compose`/`board_add_card` agent tools) —
  they construct fresh card lists and never read back corrupt JSON; leave
  them alone.
- The `boards` collection schema / migrations.
- `web/static/board.js`.

## Git workflow

- Branch: `advisor/043-board-json-errors`
- Conventional commit, e.g. `fix(boards): surface cards JSON corruption instead of silently erasing boards`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: One decode helper

Add near `boardRecordOf` in `internal/web/boards.go`:

```go
// boardCardsOf decodes the record's cards JSON. A blank field is an empty
// board; anything else that fails to parse is corruption the caller must
// surface — proceeding would overwrite the owner's composition.
func boardCardsOf(rec *core.Record) ([]boardCard, error) {
	raw := rec.GetString("cards")
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var bcs []boardCard
	if err := json.Unmarshal([]byte(raw), &bcs); err != nil {
		return nil, fmt.Errorf("decoding board %s cards: %w", rec.Id, err)
	}
	return bcs, nil
}
```

(If `strings` or `fmt` is not yet imported in boards.go, add it.)

**Verify**: `go build ./internal/web/` → exit 0.

### Step 2: Mutating handlers abort on corruption

In `boardsCardAdd` (line 386), `boardsCardRemove` (line 430), and
`boardsLayout` (line 495), replace the two-line
`var bcs []boardCard` + `_ = json.Unmarshal(...)` with:

```go
	bcs, err := boardCardsOf(rec)
	if err != nil {
		return e.InternalServerError("board cards are corrupted; fix the record in the dashboard", err)
	}
```

Watch shadowing: each function already has an `err` in scope from the
record fetch — use `=` not `:=` where needed so the build stays clean.

**Verify**: `go build ./internal/web/` → exit 0.

### Step 3: Check the Marshal write-backs

At lines 395, 438, 522, replace `raw, _ := json.Marshal(...)` with an
error check returning `e.InternalServerError("encoding board cards", err)`.
(Marshal of these plain structs can't realistically fail; the check is
cheap honesty and matches `ensureDefaultBoards` at line 206.)

**Verify**: `go build ./internal/web/` → exit 0.

### Step 4: Read path degrades loudly-but-gracefully

`boardRecordOf` renders pages; a corrupt board should still render (empty)
rather than 500 the whole boards index — but not silently. `boardRecordOf`
has no logger access (it's a free function taking only `rec`). Change its
body to use the helper and ignore the error **with a comment**, and surface
the error where there IS context — in `loadBoards` (line 132), which has
`h.app`. Concretely:

- `boardRecordOf`: `bcs, _ := boardCardsOf(rec)` with comment
  `// corrupt cards render as an empty board; loadBoards logs it`.
- In `loadBoards`'s loop over records, before building the view: call
  `boardCardsOf(rec)` and on error
  `h.app.Logger().Warn("board cards corrupted", "board", rec.Id, "err", err)`.
  (One extra decode per board per page; boards are few — fine. Do NOT
  restructure `boardRecordOf`'s signature.)

**Verify**: `go test ./internal/web/` → existing board tests still pass.

### Step 5: Regression tests

In `internal/web/boards_test.go` add:

1. `TestBoardCardAddRejectsCorruptCards` — seed a board record with
   `cards` set to `"{not json"`, POST `/ui/boards/{id}/cards/add` (copy the
   form/request shape from the existing add test), assert HTTP 500 and
   that the stored record's `cards` field is **unchanged** (re-fetch,
   compare string).
2. `TestBoardCardsOfBlankIsEmpty` — unit test: record with `cards` `""` →
   `nil, nil`; with valid JSON → round-trips; with `"{not json"` → error.

**Verify**: `go test ./internal/web/ -run TestBoard` → ok, including the
two new tests.

### Step 6: Full gate

**Verify**: `gofmt -l .` → empty; `go vet ./...` → silent;
`go test ./...` → ok; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

- New tests per Step 5 (corruption aborts + helper truth table).
- Pattern: existing handler tests in `internal/web/boards_test.go`.
- Verification: `go test ./internal/web/` → all pass, 2 new tests included.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n '_ = json.Unmarshal' internal/web/boards.go` → no matches
- [ ] `grep -n ', _ := json.Marshal\|, _ = json.Marshal' internal/web/boards.go` → no matches
- [ ] `grep -c 'boardCardsOf' internal/web/boards.go` → ≥ 5 (helper + 4+ call sites)
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l .` empty, `go vet ./...` silent, `CGO_ENABLED=0 go build ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The line numbers/excerpts above don't match (drift).
- Any existing test seeds a board with deliberately invalid `cards` JSON
  and expects success (would mean blank-vs-corrupt semantics differ from
  this plan's assumption).
- You find a code path that **writes** non-JSON into `cards` (that's a
  separate, bigger bug — report it, don't fix it here).

## Maintenance notes

- Future board features (duplication, import) must use `boardCardsOf`,
  never raw `json.Unmarshal`, to inherit the blank-vs-corrupt distinction.
- Reviewer should scrutinize: `err` shadowing in the three mutating
  handlers, and that the corrupt-add test re-fetches the record (proving
  no overwrite happened).
- Deferred: a `balaur doctor` check for corrupt board JSON could reuse
  `boardCardsOf` if ever wanted.
