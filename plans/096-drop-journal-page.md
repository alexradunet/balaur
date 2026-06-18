# Plan 096: Drop the journal page (the `journal` card) — journaling lives on in chat + the day card

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat e533c5a..HEAD -- internal/feature/journalcards/ internal/web/journal.go internal/web/web.go internal/web/home.go internal/cards/cards.go internal/cards/cards_test.go internal/feature/storybook/ internal/self/knowledge.md DESIGN.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **Sandbox note**: in a TLS-intercepting sandbox (Hyperagent), Go commands
> need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. GOSUMDB stays on.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none (independent of 095; both touch `home.go`/`cards_test.go`/`stories_navigation.go` in different regions — see Maintenance notes)
- **Category**: direction (UX) / tech-debt
- **Planned at**: commit `e533c5a`, 2026-06-18

## Why this matters

The owner decided (2026-06-18) to **drop the journal page** — the dedicated
`journal` card artifact (the "candle": a free/guided tab strip + write form +
today's entries). It is the last sidebar surface still carrying its own
in-artifact navigation (the `free / guided` tab strip), and rather than de-nav
it (the treatment plans 092–095 applied elsewhere) the owner wants it removed.

**This loses no journaling capability**, because journaling already lives in two
other places that stay:

1. The **chat** `journal_write` tool (`internal/tools/journal.go`) — the agent
   keeps the owner's reflections verbatim.
2. The **`day` card** (`internal/feature/journalcards/day.go` +
   `dayfocus.go`) — a full day surface that *renders* the day's journal entries
   **and has its own write form** (`POST /ui/day/{date}/journal`) and entry-drop.
   It is reachable independent of the journal card: the **calendar** card links
   every day cell to `/ui/show/day?date=…` (`taskcards/calendar.go`), and recap
   day cards deep-link too.

So the `entries` data, the chat verb, the day view, and the calendar→day path
all survive. Only the standalone journal *page/card* and its write/guided
endpoints go away.

## Current state

What gets **removed** vs **kept**:

**Remove (the journal page):**
- `internal/feature/journalcards/journal.go` — the `journal` card: `JournalCard`
  (tile), `JournalView`, `JournalEntry`, `buildJournal`, `journalParamLine`,
  `journalBody`, `journalEntryNode`, and `registerJournal`.
- `internal/feature/journalcards/journalfocus.go` — the candle: `JournalFocus`,
  `JournalCandleBody`, `BuildJournalFocus`, `JournalFocusView`,
  `JournalEntryView`, and the local `dayStart` helper.
- `internal/web/journal.go` — `journalWrite`, `journalPrompt`,
  `composeJournalPrompt`, `renderCandleBody`, and the `candlePromptFallback` /
  `candlePromptTimeout` consts.
- Routes `POST /ui/journal` + `GET /ui/journal/prompt` (`web.go:209-210`).
- The `journal` spec in `internal/cards/cards.go` (lines 87–96).
- The `journalfocusStory` (storybook) and its registration.
- Journal-card/route tests + the journal storybook + the journal nav fixture item.

**Keep (do NOT touch):**
- `internal/feature/journalcards/day.go`, `dayfocus.go` — the `day` card
  (including `dayStartOf`, `DayJournal`, the day write/drop UI).
- `internal/web/day.go` — `dayJournalWrite`, `dayJournalDrop`, `renderDayJournal`,
  and routes `POST /ui/day/{date}/journal`, `POST /ui/day/journal/{id}/drop`.
- `internal/tools/journal.go` — the `journal_write` chat tool.
- `internal/life/journal.go` — the data layer (`kind="journal"`).
- `internal/cli/life.go` — the CLI `journal` command.
- The `entries` collection and all journal records.

### `dayStart` is independent — deletion is clean

`journalfocus.go` defines a local `dayStart` (line ~56). `day.go` and
`dayfocus.go` use their **own** `dayStartOf` (day.go line ~32). Confirmed:
`grep -rn 'dayStart\b' internal/feature/journalcards/*.go` shows `dayStart` only
in `journalfocus.go`; everything else uses `dayStartOf`. So deleting
`journalfocus.go` removes `dayStart` without breaking the day card.

### `register.go` today (`journalcards/register.go:17-27`)

```go
func Register(app core.App) {
	registerJournal(app)
	registerDay(app)
}

func Unregister() {
	ui.UnregisterCard("journal")
	ui.UnregisterCard("day")
}
```

Remove the `registerJournal(app)` line and the `ui.UnregisterCard("journal")`
line; keep the `day` lines.

### The `journal` spec today (`cards.go:87-96`)

```go
		{
			Type:  "journal",
			Label: "Journal",
			Icon:  "quill",
			W:     4,
			H:     18,
			Params: []ParamSpec{
				{Name: "limit", Doc: "number of recent journal entries to show (default 5, max 50)"},
			},
		},
```

### `cards_test.go` references to fix

- `allTypes` (line 13–16) lists `"journal"` — remove it.
- `TestValidateNumericClamping` (line ~97) has a sub-test `typ: "journal"` with
  `limit: "3"` → re-point to another card that has a `limit` param, e.g.
  `"memory"` (memory has `limit`); keep `wantValue: "3"`.
- `TestHasManage` (line ~217) lists `"journal"` in the **not-manage** set —
  remove it (the type no longer exists; `HasManage("journal")` would still
  return false, but the type is gone from the registry so leaving it is dead).

### `web/journal_test.go` — what to delete vs keep

- **Delete**: `TestJournalArtifact` (the `/ui/show/journal` summon),
  `TestJournalWrite` (`POST /ui/journal`), `TestJournalPrompt`
  (`GET /ui/journal/prompt`) — all reference removed routes/consts.
- **Keep** (retarget comments): `TestJournalCandleIntegration` — it writes via
  `life.JournalWrite` (kept) and asserts the **day** view shows the entry; it
  does NOT call `POST /ui/journal`. Update its name/comment to drop "candle".
  And `TestJournalAndDayRoutesRetired` — it asserts the retired `/journal` +
  `/day/...` page routes 302 (the catch-all); still valid. Update its comment
  ("the journal/day card artifacts live on" → "the day card artifact lives on;
  the journal card was removed in plan 096").
- After deleting the three tests, prune now-unused imports (`strings`,
  `llmtest`) — `go build`/`go vet` will tell you which.

### `web.go` routes today (`web.go:209-210`)

```go
	se.Router.POST("/ui/journal", h.journalWrite)
	se.Router.GET("/ui/journal/prompt", h.journalPrompt)
```

Remove both lines. The `/ui/day/...` routes immediately below stay.

### Storybook references

- `internal/feature/storybook/story.go:98` — `journalfocusStory(),` in the
  story list. Remove that line.
- `internal/feature/storybook/stories_cards.go:277-304` — the entire
  `journalfocusStory()` func. Remove it.
- `internal/feature/storybook/stories_navigation.go:150` — the sidebar fixture
  has `item("Journal", "journal", "quill", false),`. Remove that one item line.

### Docs to update

- `DESIGN.md` line ~92: `carries the six domains (Quests, Knowledge, Life,
  Journal, Heads, Settings)` — remove `Journal, ` (and adjust the count word).
- `DESIGN.md` line ~167: `typed card registry — 14 parameterized … card
  resources … today, quests, calendar, timeline, journal, day, …, settings` —
  remove `journal, ` and change `14` → `13`.
- `DESIGN.md` line ~169: the clause `the candle (the journal card's artifact at
  /ui/show/journal): immersive writing surface — … entries shared with the day
  card ·` — delete this clause.
- `internal/self/knowledge.md` ~135–141: the paragraph beginning `The candle
  (the journal card's writing surface, artifact at /ui/show/journal): …` —
  delete the whole paragraph. (The day-card paragraph just below already states
  that the day shows the owner's journal entries.)
- `internal/self/knowledge.md` ~152: in the card-registry list, remove
  `journal (recent entries, limit param), `.
- **Do NOT** remove the `journal_write` line (~92) or the CLI `journal` entry
  (~127) — those are the chat tool and CLI command, both kept.

### Repo conventions to match

- Card types self-register from feature packages; the registry source of truth
  is `internal/cards/cards.go` (no web imports). Removing a type = remove its
  spec + its `register*`/`Unregister` lines.
- `.k-tabs` / `.k-tab` CSS (`basm.css`) and `internal/ui/tabs.go` are a SHARED
  atom — leave them; the journal card was only one consumer.

## Commands you will need

| Purpose    | Command                                             | Expected on success |
|------------|-----------------------------------------------------|---------------------|
| Build      | `CGO_ENABLED=0 go build ./...`                      | exit 0              |
| Vet        | `go vet ./...`                                      | exit 0              |
| Test (pkg) | `go test ./internal/web/... ./internal/feature/... ./internal/cards/...` | all pass |
| Test (all) | `go test ./...`                                     | all pass            |
| Format     | `gofmt -l internal/`                                | no output           |
| Diff check | `git diff --check`                                  | no output           |

## Scope

**In scope** (modify/delete only these):
- Delete: `internal/feature/journalcards/journal.go`, `internal/feature/journalcards/journalfocus.go`, `internal/feature/journalcards/journalfocus_test.go`, `internal/web/journal.go`.
- Edit: `internal/feature/journalcards/register.go`, `internal/web/web.go`, `internal/web/home.go`, `internal/cards/cards.go`, `internal/cards/cards_test.go`, `internal/web/journal_test.go`, `internal/feature/storybook/story.go`, `internal/feature/storybook/stories_cards.go`, `internal/feature/storybook/stories_navigation.go`, `internal/self/knowledge.md`, `DESIGN.md`.

**Out of scope** (do NOT touch):
- `internal/feature/journalcards/day.go`, `dayfocus.go`, `day_test.go`, `dayfocus_test.go` — the day card stays.
- `internal/web/day.go` and the `/ui/day/...` routes — stay.
- `internal/tools/journal.go`, `internal/life/journal.go`, `internal/cli/life.go` — chat tool, data layer, CLI command all stay.
- The `journal_write` tool-group plumbing (`internal/heads/heads.go`, `internal/turn/tools.go`, `internal/web/web.go:44` icon mapping) — these are about the chat tool, not the card.
- Plan 095 surfaces (knowledge).

## Git workflow

- Branch: `improve/096-drop-journal-page`.
- Commit per logical unit; conventional-commit style, e.g.
  `feat(web): drop the journal page (journaling stays in chat + the day card)`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Delete the journal card + candle source files

`git rm` (or delete) `internal/feature/journalcards/journal.go` and
`internal/feature/journalcards/journalfocus.go`.

In `internal/feature/journalcards/register.go`, remove `registerJournal(app)`
from `Register` and `ui.UnregisterCard("journal")` from `Unregister` (keep the
`day` lines).

**Verify**: `grep -rn 'registerJournal\|JournalFocus\|JournalCandleBody\|BuildJournalFocus' internal/feature/journalcards/` → no output (the `day` files never referenced them). Build comes in Step 3.

### Step 2: Delete the journal web handlers + routes

`git rm` (or delete) `internal/web/journal.go`. In `internal/web/web.go`, remove
the two routes `POST /ui/journal` and `GET /ui/journal/prompt` (lines ~209–210).

**Verify**: `grep -rn 'journalWrite\|journalPrompt\|composeJournalPrompt\|renderCandleBody\|candlePromptFallback' internal/web/` → only matches inside `internal/web/journal_test.go` (cleaned in Step 5), nothing in non-test code.

### Step 3: Remove the `journal` card spec + sidebar item

1. `internal/cards/cards.go` — delete the `journal` spec block (lines ~87–96).
2. `internal/web/home.go` — in `domainSidebar()`, remove the
   `item("Journal", "journal", "quill"),` line from the Domains section.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0. `grep -rn '"journal"' internal/cards/cards.go internal/web/home.go` → no output.

### Step 4: Fix the cards registry tests

In `internal/cards/cards_test.go`:
1. Remove `"journal"` from the `allTypes` slice.
2. In `TestValidateNumericClamping`, change the `typ: "journal"` sub-test to
   `typ: "memory"` (keep `params: {"limit": "3"}`, `key: "limit"`,
   `wantValue: "3"`).
3. In `TestHasManage`, remove `"journal"` from the not-manage list.

**Verify**: `go test ./internal/cards/...` → all pass (`TestAll` now expects one
fewer spec and `cards.All()` returns one fewer — they must match).

### Step 5: Prune the journal web tests

In `internal/web/journal_test.go`:
- Delete `TestJournalArtifact`, `TestJournalWrite`, `TestJournalPrompt`.
- Keep `TestJournalCandleIntegration` (consider renaming to
  `TestDayCardReflectsJournalEntry`; update its comment to drop "candle"/"POST
  /ui/journal" — it uses `life.JournalWrite` + the day view).
- Keep `TestJournalAndDayRoutesRetired`; update its comment to: the day card
  artifact lives on; the journal card was removed in plan 096.
- Remove now-unused imports (`strings`, `llmtest`, and anything else the build
  flags). Do NOT delete the shared `seedScriptedModel`/`seedFailingModel`
  helpers — they live in other `_test.go` files and serve other tests.

Also delete `internal/feature/journalcards/journalfocus_test.go` (it tests the
removed `JournalFocus`/`JournalCandleBody`).

**Verify**: `go test ./internal/web/... ./internal/feature/journalcards/...` → all pass.

### Step 6: Remove the journal storybook story + nav fixture item

1. `internal/feature/storybook/story.go` — remove the `journalfocusStory(),`
   line (~98) from the story list.
2. `internal/feature/storybook/stories_cards.go` — remove the entire
   `journalfocusStory()` function (~277–304).
3. `internal/feature/storybook/stories_navigation.go` — remove the
   `item("Journal", "journal", "quill", false),` line (~150) from the sidebar
   fixture.

**Verify**: `grep -rn 'journalfocusStory\|"Journal"' internal/feature/storybook/` → no output. `go test ./internal/feature/storybook/...` → all pass.

### Step 7: Update the docs (DESIGN.md + self/knowledge.md)

Apply the doc edits listed under "Docs to update" in Current state:
- `DESIGN.md`: remove `Journal` from the domain list (~92); remove `journal, `
  from the card-type list and change `14` → `13` (~167); delete the candle
  clause (~169).
- `internal/self/knowledge.md`: delete the candle paragraph (~135–141); remove
  the `journal (recent entries, limit param), ` item from the card list (~152).
- Leave the `journal_write` (chat tool) and CLI `journal` mentions intact.

If plan 095 has already landed (the sidebar is Domains/Knowledge/Settings
groups), make the DESIGN.md domain parenthetical match the real rail; otherwise
just remove `Journal`. Full IA-prose reconciliation beyond removing journal is a
separate docs pass — do not expand scope.

**Verify**: `grep -rin 'candle\|/ui/show/journal' DESIGN.md internal/self/knowledge.md` → no output. `grep -n '13 ' DESIGN.md` → the card-count line shows 13.

### Step 8: Full gates + completeness greps

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `gofmt -l internal/` → no output
- `git diff --check` → no output
- `grep -rn 'ui/show/journal\|/ui/journal\b\|"journal".*quill\|registerJournal\|JournalCard\|JournalFocus\|JournalCandleBody' internal/ --include=*.go | grep -v _test` → no output (the journal CARD is gone from production code).
- Sanity that the day card still works: `go test ./internal/web/... -run 'Day|Journal'` → all pass (the day write/integration tests stay green).

## Test plan

- Delete the three journal card/route tests + the journal focus component test.
- Keep the day-reflects-journal integration test (proves journaling still shows
  on the day card) and the retired-routes test.
- Pattern: the kept tests already use `tests.ApiScenario` / direct
  `life.JournalWrite` — no new test scaffolding needed.
- Optional (nice): add a sub-test to `internal/web/show_test.go` or
  `page_error_test.go` that `GET /ui/show/journal` → 404 (the type is gone) —
  model it on the existing `GET /ui/show/bogus → 404` case. This guards against
  silent re-registration.
- Verification: `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l internal/` prints nothing
- [ ] `internal/feature/journalcards/journal.go`, `journalfocus.go`, `journalfocus_test.go`, and `internal/web/journal.go` no longer exist (`ls` errors / `git status` shows deletions)
- [ ] `grep -rn 'ui/show/journal\|/ui/journal\b\|registerJournal\|JournalFocus' internal/ --include=*.go | grep -v _test` returns nothing
- [ ] `grep -rn '"journal"' internal/cards/cards.go internal/web/home.go` returns nothing
- [ ] `cards.All()` returns 13 specs (`go test ./internal/cards/...` passes with the updated `allTypes`)
- [ ] The `day` card still works: `journal_write` tool, `internal/web/day.go`, and `/ui/day/...` routes are untouched (`grep -rn 'dayJournalWrite' internal/web/` still present)
- [ ] `git status` shows only in-scope files modified/deleted
- [ ] `plans/readme.md` status row for 096 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The files/excerpts above don't match the live code (drift since `e533c5a`).
- Deleting `journalfocus.go` breaks the build because something OTHER than the
  removed journal code uses `dayStart` or `BuildJournalFocus`/`JournalCandleBody`
  (it should not — `day.go`/`dayfocus.go`/`web/day.go` use `dayStartOf` and
  `DayJournal`; if the build complains, STOP — an unexpected coupling exists).
- Removing the `journal` spec breaks a test that hard-codes the card count
  somewhere other than `cards_test.go`'s `allTypes` (grep `len(cards.All` and
  `13`/`14` across tests before assuming).
- A verification fails twice after a reasonable fix attempt.
- You find a production (non-test, non-doc) caller of the journal card that the
  scope list did not anticipate.

## Maintenance notes

- Journaling after this: the chat `journal_write` tool keeps entries; the owner
  views/writes/drops them on the **day** card (`/ui/show/day?date=…`), reached
  from the calendar card and recap day cards. No journal data is lost.
- The retired standalone page routes `/journal` + `/day/...` still 302 → `/`
  (the router catch-all) — that behavior is unrelated to this change and the
  `TestJournalAndDayRoutesRetired` test still guards it.
- The `journal` tool group (`internal/heads/heads.go` `Groups`) and the Coach
  persona's `journal` group reference the **chat tool**, not the card — left
  intact so heads can still journal.
- Reviewer should scrutinize: (1) no production code still references the
  `journal` card type; (2) the day card's write/drop forms and the
  `journal_write` tool are untouched; (3) `cards.All()` count and `allTypes`
  agree (13).
- Sibling plan 095 (knowledge categories) shares `home.go` (different sidebar
  region: 095 splits Knowledge into a group; 096 drops the Journal item from
  Domains), `cards_test.go` (different test funcs), and
  `stories_navigation.go` (the same sidebar fixture). Land 095 then 096, or
  reconcile the fixture + Domains list at merge.
- Deferred (not in scope): a fuller DESIGN.md IA-prose reconciliation (the
  "domains" parenthetical predates the 092 Settings-group split); do it in a
  dedicated docs pass.
