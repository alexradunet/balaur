# Plan 203: Extract the shared chat-history renderer from `web/recap.go` into `web/history.go`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/web/recap.go`
> If the file changed since this plan was written, compare the "Current state"
> line references against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (pure same-package relocation; independent of #205 even
  though both touch `package web` — they touch different files/symbols)
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

`internal/web/recap.go` (404 LOC) is named for the recap telescope, but it also
holds the **core chat-history renderer** — `messageView`, `renderMessages`,
`chatBodyHTML`, `messageViews` — which is shared infrastructure imported by the
home page, the dock, task nudges, and compaction. Anyone looking for "how chat
history renders" has to discover it by following `homeData` into a file named
`recap.go`. This is pure misfiling, not coupling: the fix is a same-package
relocation into a new `web/history.go` next to `web/chatstream.go` (the live SSE
renderer), so the reload renderer and the live renderer sit together. No
behavior changes, no caller changes (same package).

## Current state

`internal/web/recap.go` mixes two concerns:

**Recap telescope (STAYS in recap.go):** `recapView`, `bandView`, `recapCard`,
`bandHeading`, `recapExpand`, `chronicleView`, `chronicleBody`,
`chronicleBandsNode`, `recapCardsNode`.

**Chat-history renderer (MOVES to history.go):**

- `type messageView struct` — lines 129–153 (the chat message template payload;
  note the `CardBody g.Node` field).
- `func (h *handlers) renderMessages(views []messageView) g.Node` — lines 155–194.
- `func (h *handlers) chatBodyHTML(d homeData) g.Node` — lines 196–229.
- `func (h *handlers) messageViews(recs []*core.Record) []messageView` — lines 231–306.

These four symbols are used from elsewhere in `package web` (e.g.
`internal/web/tasks.go:270` `h.renderMessages(h.messageViews(recs))`,
`recap.go:106` and `:206`/`:216`, plus `homeData.History []messageView` and the
SSE stream). Because the move stays within `package web`, **no call site
changes** — only the file the definitions live in.

### Import bookkeeping (the one non-trivial part)

The current `internal/web/recap.go` import block (lines 3–20):

```go
import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/ui/chat"
)
```

After the four symbols move out:

- **`recap.go` loses** (these are used ONLY by the moved symbols): `encoding/json`
  (only `messageViews` uses `json.Unmarshal`), `tools` (only `messageViews` uses
  `tools.Parse*`), and `chat` i.e. `internal/ui/chat` (only `renderMessages` and
  `chatBodyHTML` use `chat.*`).
- **`recap.go` keeps**: `fmt`, `strconv`, `strings`, `time`, `core`, `datastar`,
  `g`, `h` (`html`), `conversation`, `recap`, `store`.
- **`history.go` needs**: `encoding/json`, `core`, `g` (`maragu.dev/gomponents`),
  `store`, `tools`, `chat` (`internal/ui/chat`). It does **not** need `h`
  (`html`), `fmt`, `time`, `strconv`, `strings`, `datastar`, `conversation`, or
  `recap`.

Verify these claims before finalizing imports:
`grep -n "json\.\|tools\.\|chat\." internal/web/recap.go` should, after the move,
return nothing in `recap.go`.

## Commands you will need

| Purpose   | Command                                   | Expected            |
|-----------|-------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`            | exit 0              |
| Vet       | `go vet ./...`                            | exit 0              |
| Test pkg  | `go test ./internal/web/...`              | PASS                |
| Full test | `go test ./...`                           | all pass            |
| gofmt     | `gofmt -l internal/web`                   | prints nothing      |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/web/recap.go` (remove the four chat-history symbols + their now-unused imports)
- `internal/web/history.go` (create; receive the four symbols + their imports)

**Out of scope** (do NOT touch):
- Any call site of `messageView`/`renderMessages`/`chatBodyHTML`/`messageViews`
  — they are package-internal and need no change.
- `internal/web/chatstream.go` — the live renderer stays as-is (history.go just
  sits next to it).
- The recap telescope symbols staying in `recap.go`.

## Git workflow

- Branch: `advisor/203-web-history-renderer-split`
- Conventional-commit subject, e.g. `refactor(web): split chat-history renderer into history.go`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Create `internal/web/history.go`

Create the file with the package clause, the import set listed above, and the
four symbols moved **verbatim** from `recap.go` (the `messageView` struct,
`renderMessages`, `chatBodyHTML`, `messageViews`). Add a short package-level
comment at the top explaining the file's role:

```go
package web

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

// history.go renders chat transcripts for page-load reloads: the messageView
// payload, renderMessages (transcript → chat.* components), chatBodyHTML (the
// #chat body: history or the hearth greeting), and messageViews (records →
// payloads, decoding the persisted tool-result markers). The live SSE renderer
// lives next door in chatstream.go and renders the same chat.* components, so
// reloaded and streamed turns match. Moved out of recap.go (plan 203).

// ... the four symbols, copied verbatim ...
```

Copy the bodies exactly as they appear in `recap.go` lines 129–306. Do not
change any logic.

**Verify**: file exists; symbols present:
`grep -n "func (h \*handlers) renderMessages\|func (h \*handlers) chatBodyHTML\|func (h \*handlers) messageViews\|type messageView struct" internal/web/history.go` → 4 matches.

### Step 2: Remove the four symbols from `recap.go` and fix its imports

Delete lines 129–306 from `internal/web/recap.go` (the `// messageView is one
chat message's...` comment through the end of `messageViews`). Then remove the
now-unused imports from `recap.go`: `encoding/json`, `tools`
(`internal/tools`), and `chat` (`internal/ui/chat`).

**Verify**:
- `grep -n "json\.\|tools\.\| chat\.\|chat\.Message\|chat\.ToolRow\|chat\.CompactNote" internal/web/recap.go` → no matches
- `grep -n "encoding/json\|internal/tools\|internal/ui/chat" internal/web/recap.go` → no matches
- `gofmt -l internal/web` → prints nothing

### Step 3: Build and test

**Verify**:
- `go build ./internal/web/...` → exit 0 (this proves both files have exactly
  the imports they use — a missing or extra import fails the build)
- `go vet ./internal/web/...` → exit 0
- `go test ./internal/web/...` → PASS
- `go test ./...` → all pass

## Test plan

- No new tests. This is a pure relocation within one package; the existing
  `internal/web` tests (history rendering, dock, nudges, recap expand) cover all
  four symbols and must pass unchanged.
- If a test file imports nothing new and still compiles, that is the proof the
  move was behavior-preserving.
- Verification: `go test ./internal/web/...` → PASS with the same test count as
  before.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `internal/web/history.go` exists and defines `messageView`,
      `renderMessages`, `chatBodyHTML`, `messageViews`
- [ ] `grep -n "type messageView struct\|func (h \*handlers) renderMessages" internal/web/recap.go` returns no matches
- [ ] `grep -n "encoding/json\|internal/tools\|internal/ui/chat" internal/web/recap.go` returns no matches
- [ ] `gofmt -l internal/web` prints nothing
- [ ] `go test ./...` exits 0
- [ ] Only `internal/web/recap.go` and `internal/web/history.go` are modified/created (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- After deleting the four symbols, `recap.go` still references `json.`, `tools.`,
  or `chat.` somewhere — then one of those imports is NOT exclusive to the moved
  code; keep it and report.
- Any of the four symbols turns out to be referenced by a file OUTSIDE
  `package web` (it shouldn't be — they're unexported) — `grep -rn
  "renderMessages\|messageViews\|chatBodyHTML\|messageView" internal/ --include=*.go | grep -v internal/web`
  returns matches. Report rather than proceed.
- The build fails with an "imported and not used" or "undefined" error you can't
  resolve by adjusting only the import sets of the two files — the symbol set
  moved is wrong; report.

## Maintenance notes

- After this, the reload chat renderer (`history.go`) and the live SSE renderer
  (`chatstream.go`) are filename-adjacent — keep them in sync when changing chat
  markup (both render `chat.*` components).
- `recap.go` is now just the telescope. If it's still large, that's a separate
  concern — do not fold history rendering back in.
- Reviewer: confirm the four symbols are byte-identical to before (a verbatim
  move), and that `recap.go`'s remaining imports are exactly the kept set listed
  above.
