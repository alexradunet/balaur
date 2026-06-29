# Plan 204: Split `internal/knowledge/knowledge.go` (637 LOC) — extract `edit.go` (parked-edit envelope) and `search.go` (search surfaces)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/knowledge/knowledge.go`
> If the file changed since this plan was written, compare the "Current state"
> line references against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (pure same-package relocation)
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

`internal/knowledge/knowledge.go` is 637 LOC — past the repo's ~500-LOC
decompose threshold (`AGENTS.md`: "Treat a package past ~500 lines as a smell to
decompose"). It fuses five concerns: hydration, the proposal lifecycle, the
**parked-edit envelope**, management listing, and **two search surfaces**. The
package already demonstrates the target pattern — `cache.go` and `context.go`
split out cleanly — but `knowledge.go` itself didn't. Extracting the parked-edit
envelope into `edit.go` and the search trio into `search.go` drops the core file
to ~370 LOC, each file readable on its own. This is a same-package move: no
caller changes, no API changes.

## Current state

`internal/knowledge/knowledge.go` symbol map (line ranges as of commit `07fb4d6`):

| Lines    | Symbol(s)                                              | Destination   |
|----------|-------------------------------------------------------|---------------|
| 1–34     | package doc + imports                                  | knowledge.go (adjust) |
| 36–105   | `Kind`, status consts, `clampImportance`, `Hydrate`/`hydrate`/`hydrateAll` | knowledge.go (stays) |
| 107–159  | `MemoryProposal`/`ProposeMemory`, `SkillProposal`/`ProposeSkill` | knowledge.go (stays) |
| 161–194  | `validTransitions`, `Transition`                      | knowledge.go (stays) |
| 196–239  | `UpdateFields`                                         | knowledge.go (stays) |
| **241–386** | `pendingEditKey`, `ProposeEdit`, `PendingEdit`, `PendingEdits`, `ApplyEdit`, `DeclineEdit`, `clearPendingEdit` | **→ edit.go** |
| 388–449  | `Touch`, `ListByStatus`, `FilterActive`, `matchesQuery` | knowledge.go (stays) |
| **451–594** | `searchActiveNodes`, `SearchActive`, `SearchAllActive` | **→ search.go** |
| 596–637  | `UpfrontMemories`, `ActiveSkills`, `LoadSkill`         | knowledge.go (stays) |

The exemplar split files already in the package: `internal/knowledge/cache.go`
(holds `loadContextCache`, `invalidateContextCache`, `copyForRead`) and
`internal/knowledge/context.go`. Match their style: `package knowledge`, a short
top-of-file comment naming the concern, then the symbols.

### Critical: `matchesQuery` STAYS in knowledge.go

`matchesQuery` (lines 438–449) is used by BOTH `FilterActive` (line 424, in the
management code that stays) AND the `SearchActive` fallback (line 533, in the
code that moves to search.go). Leave `matchesQuery` in `knowledge.go` — same
package, so `search.go` calls it freely. Moving it would create a
management→search-file dependency for no benefit.

### Import bookkeeping

Current imports (lines 20–34):

```go
import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/search"
	"github.com/alexradunet/balaur/internal/store"
)
```

- **edit.go needs**: `fmt`, `time`, `core`, `nodes`, `store`.
  (`ProposeEdit` uses `time.Now()`; all use `fmt.Errorf`, `nodes.Props`,
  `store.Audit`, `*core.Record`.) It does NOT need `slices`, `sort`, `strconv`,
  `strings`, `dbx`, or `search`.
- **search.go needs**: `sort`, `strings`, `core`, `nodes`, `search`.
  (`searchActiveNodes` uses `search.Index`/`search.StoreKey` + `sort`;
  `SearchActive`/`SearchAllActive` use `nodes`, `strings`, `sort`, and the
  package-local `matchesQuery`.) It does NOT need `fmt`, `time`, `strconv`,
  `dbx`, `store`.
- **knowledge.go after both moves loses ONLY `search`** (it was used only by
  `searchActiveNodes`). It KEEPS `sort` (used by `FilterActive` at line 431),
  `dbx` (used by `LoadSkill` at line 630), `strconv` (`UpdateFields`), `time`
  (`Touch`), `slices` (`Transition`), `fmt`, `strings`, `core`, `nodes`, `store`.

## Commands you will need

| Purpose   | Command                                   | Expected            |
|-----------|-------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`            | exit 0              |
| Vet       | `go vet ./...`                            | exit 0              |
| Test pkg  | `go test ./internal/knowledge/...`        | PASS                |
| Full test | `go test ./...`                           | all pass            |
| gofmt     | `gofmt -l internal/knowledge`             | prints nothing      |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/knowledge/knowledge.go` (remove the two extracted blocks; drop the `search` import)
- `internal/knowledge/edit.go` (create; parked-edit envelope)
- `internal/knowledge/search.go` (create; search trio)
- `.tours/09-recall-and-search.tour` (repoint the 2 search-symbol anchors — see Step 4)
- `.tours/06-memory-and-self-evolution.tour` (repoint the 1 `SearchActive` anchor — see Step 4)

> **Why the tours are in scope:** two `.tours/` files anchor symbols this plan
> MOVES to `search.go`. `tours_test` (part of `go test ./...`, run by the
> pre-commit hook) fails on an out-of-range or moved anchor, so the commit is
> blocked until they're repointed. This is the AGENTS.md rule: *fix the tour in
> the same commit when a change breaks a tour anchor.* Affected anchors:
> - `09-recall-and-search.tour`: `searchActiveNodes` (currently `knowledge.go:463`) and `SearchActive` (currently `knowledge.go:509`) → both move to `search.go`.
> - `06-memory-and-self-evolution.tour`: `SearchActive` (currently `knowledge.go:509`) → moves to `search.go`.
> The OTHER `06` anchors (`knowledge.go:1` package doc, `:167` `Transition`, `:198`
> `UpdateFields`) STAY in `knowledge.go` and are ABOVE both removed blocks, so
> they do NOT shift — do not touch them.

**Out of scope** (do NOT touch):
- `matchesQuery` — stays in `knowledge.go`.
- `internal/knowledge/cache.go`, `context.go` — unchanged (they are the
  exemplars, not part of this move).
- Any caller of the moved functions — they're same-package or external callers
  of unchanged exported symbols; no call site changes.
- The behavior of any moved function — verbatim move only.

## Git workflow

- Branch: `advisor/204-knowledge-file-split`
- Conventional-commit subject, e.g. `refactor(knowledge): split edit.go + search.go out of knowledge.go`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Extract `edit.go`

Create `internal/knowledge/edit.go`:

```go
package knowledge

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
)

// edit.go is the parked-edit envelope: a model proposes a change to an ACTIVE
// memory/skill by parking it in the node's props (ProposeEdit) without touching
// the approved content; only the owner approves (ApplyEdit), declines
// (DeclineEdit), or the review queue lists pending edits (PendingEdits). The
// consent boundary in the package doc holds here — the model proposes, it never
// applies. Split out of knowledge.go (plan 204).

// ... move lines 241–386 verbatim: pendingEditKey, ProposeEdit, PendingEdit,
//     PendingEdits, ApplyEdit, DeclineEdit, clearPendingEdit ...
```

Move the bodies exactly as they appear. Note `ApplyEdit` calls `Transition` and
`UpdateFields` (which stay in `knowledge.go`) — same package, fine.

**Verify**: `grep -n "func ProposeEdit\|func ApplyEdit\|func clearPendingEdit\|pendingEditKey" internal/knowledge/edit.go` → present.

### Step 2: Extract `search.go`

Create `internal/knowledge/search.go`:

```go
package knowledge

import (
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/search"
)

// search.go is the active-knowledge search surface: searchActiveNodes is the
// shared FTS5-or-substring-fallback skeleton; SearchActive is the memory-scoped
// recall path (hydrates memory aliases); SearchAllActive is the cross-type
// surface returning raw active node records. The consent status=active filter
// lives in each `keep`/fallback. Split out of knowledge.go (plan 204).

// ... move lines 451–594 verbatim: searchActiveNodes, SearchActive, SearchAllActive ...
```

These call the package-local `matchesQuery` (stays in knowledge.go) and
`nodes.ListByTypeStatus`, `hydrate`/`hydrateAll` (stay in knowledge.go) — all
same package.

**Verify**: `grep -n "func searchActiveNodes\|func SearchActive\|func SearchAllActive" internal/knowledge/search.go` → 3 matches.

### Step 3: Remove the moved blocks from `knowledge.go` and drop `search` import

- Delete lines 241–386 (the parked-edit block) and lines 451–594 (the search
  block) from `internal/knowledge/knowledge.go`.
- Remove `"github.com/alexradunet/balaur/internal/search"` from its import block
  (now orphaned). Keep everything else.
- Confirm `matchesQuery` (was 438–449) is STILL in `knowledge.go`.

**Verify**:
- `grep -n "func matchesQuery" internal/knowledge/knowledge.go` → one match (stayed)
- `grep -n "func ProposeEdit\|func searchActiveNodes" internal/knowledge/knowledge.go` → no matches (moved out)
- `grep -n "internal/search" internal/knowledge/knowledge.go` → no matches
- `gofmt -l internal/knowledge` → prints nothing

### Step 4: Repoint the search-symbol tour anchors to `search.go`

After Step 2 (`search.go` created) and Step 3 (`knowledge.go` trimmed),
`searchActiveNodes` and `SearchActive` live in `search.go`. Three tour anchors
still point at them in `knowledge.go` — repoint each (BOTH `file` AND `line`) or
`tours_test` fails the pre-commit hook.

First get their new lines in `search.go`:
```
grep -n "^func searchActiveNodes\|^func SearchActive(" internal/knowledge/search.go
```
Call these LINE_SAN (searchActiveNodes) and LINE_SA (SearchActive).

Then in each tour JSON, find the anchor object and change `"file"` to
`internal/knowledge/search.go` and `"line"` to the new value:
- `.tours/09-recall-and-search.tour`: anchor `"line": 463` (searchActiveNodes) → `search.go` / LINE_SAN
- `.tours/09-recall-and-search.tour`: anchor `"line": 509` (SearchActive) → `search.go` / LINE_SA
- `.tours/06-memory-and-self-evolution.tour`: anchor `"line": 509` (SearchActive) → `search.go` / LINE_SA

Do NOT touch the other `06` anchors (`knowledge.go:1` package doc, `:167`
`Transition`, `:198` `UpdateFields`) — they stay in `knowledge.go`, above both
removed blocks, so their lines are unchanged.

**Verify**:
- `grep -c '"file": "internal/knowledge/knowledge.go"' .tours/09-recall-and-search.tour` → 0
- `grep -c '"file": "internal/knowledge/search.go"' .tours/09-recall-and-search.tour` → 2
- `grep -c '"file": "internal/knowledge/knowledge.go"' .tours/06-memory-and-self-evolution.tour` → 3
- `grep -c '"file": "internal/knowledge/search.go"' .tours/06-memory-and-self-evolution.tour` → 1
- `go test . -run TestTours` → PASS (the anchor gate)

### Step 5: Build and test

**Verify**:
- `go build ./internal/knowledge/...` → exit 0 (proves each file's imports match
  its uses)
- `go vet ./internal/knowledge/...` → exit 0
- `go test ./internal/knowledge/...` → PASS
- `wc -l internal/knowledge/knowledge.go` → ~370 lines (down from 637)
- `go test ./...` → all pass

## Test plan

- No new tests — verbatim relocation. The existing `knowledge_test.go` and
  `cache_test.go` cover proposals, transitions, edits, and search; all must pass
  unchanged.
- If `go vet`/build is green and tests pass, the move is proven behavior-neutral.
- Verification: `go test ./internal/knowledge/...` → PASS with the same test
  count as before.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `internal/knowledge/edit.go` and `internal/knowledge/search.go` exist
- [ ] `grep -n "func matchesQuery" internal/knowledge/knowledge.go` returns one match
- [ ] `grep -n "internal/search" internal/knowledge/knowledge.go` returns no matches
- [ ] `gofmt -l internal/knowledge` prints nothing
- [ ] `grep -c '"file": "internal/knowledge/knowledge.go"' .tours/09-recall-and-search.tour` returns 0; the same grep on `.tours/06-memory-and-self-evolution.tour` returns 3
- [ ] `go test . -run TestTours` passes (the moved search anchors now resolve in `search.go`)
- [ ] `go test ./...` exits 0
- [ ] Only `knowledge.go` modified, `edit.go`/`search.go` created, and the 2 tour files repointed (`git status` shows exactly these 5 paths)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- After the moves, `knowledge.go` still references `search.` (then `search` is
  used outside `searchActiveNodes`; keep the import and report) — OR `edit.go`/
  `search.go` reference a symbol that turns out to be unexported in a file you
  didn't expect.
- The build reports `matchesQuery` undefined from `search.go` — it means
  `matchesQuery` was accidentally moved; it must stay in `knowledge.go`.
- A build error can't be resolved by adjusting only the three files' import
  sets — the line ranges drifted; re-confirm against the drift check.

## Maintenance notes

- After this, `internal/knowledge` is one-concern-per-file: `knowledge.go`
  (hydration + proposals + transitions + management), `edit.go` (parked-edit
  envelope), `search.go` (search surfaces), `cache.go`/`context.go` (the context
  cache). Keep new knowledge logic in the matching file.
- Reviewer: confirm `matchesQuery` stayed put and the moved blocks are verbatim
  (diff should be pure relocation + the one `search` import drop).
- If a future change adds a third search surface, it goes in `search.go`.
