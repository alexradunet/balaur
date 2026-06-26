# Plan 205: Split `internal/web/models.go` (550 LOC) into the dock view-model (`home.go`), the model-install SSE orchestrators (`models_install.go`), and a slim `models.go` (model selection + cloud consent)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/web/models.go internal/web/home.go`
> If either file changed since this plan was written, compare the "Current
> state" line references against the live code before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: LOW (mechanical, but the largest relocation in this batch)
- **Depends on**: none. Independent of #203 even though both touch `package web`
  (different files/symbols). Best done AFTER #200/#203/#205-precursors land so
  there's less churn, but no hard code dependency.
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

`internal/web/models.go` (550 LOC) fuses **four unrelated flows** into one file
whose name advertises only one of them:

1. the **chat-dock view-model** (`homeData`/`homeData()`/`chatbar`/
   `patchChatbar`/`refreshDockChrome` + the `homeData`/`headChoice` structs),
2. **model selection + cloud consent** (`selectModel`/`saveCloudModel`/
   `saveCloudPreset`/`confirmCloudModel`/`deleteCloudModel`/`cloudConsentDialog`
   + `modelsPanel`/`setProcessor`),
3. a **GGUF-download SSE orchestrator** (`downloadOfficialModel`/`cancelDownload`),
4. a **runtime-install SSE orchestrator** (`installRuntime`),

with two `app.Store()` cancel-func sidecars and shared single-flight helpers
(`claimInFlight`, `formatProgress`, `humanBytes`) underneath. To edit how the
dock's model label refreshes you scroll past 100+ lines of HuggingFace download
streaming; to trace a download you filter out dock chrome and consent dialogs.
The file also crosses the repo's ~500-LOC decompose threshold, and its *name*
hides that it owns the home page's view-model.

This is a pure same-package relocation — **no new types, no new abstractions, no
behavior change**. Do **NOT** wrap the two SSE handlers behind a shared
"progress orchestrator" interface; the ~15 lines of single-flight boilerplate is
not worth a single-impl seam.

## Current state

`internal/web/models.go` symbol map (line ranges as of commit `07fb4d6`):

| Lines    | Symbol(s)                                                                 | Destination          |
|----------|---------------------------------------------------------------------------|----------------------|
| 27–53    | `type homeData struct`, `type headChoice struct`                          | **→ home.go**        |
| 55–85    | `func (h *handlers) homeData()`                                           | **→ home.go**        |
| 87–97    | `func (h *handlers) chatbar`                                              | **→ home.go**        |
| 99–112   | `func (h *handlers) patchChatbar`                                         | **→ home.go**        |
| 114–127  | `func (h *handlers) refreshDockChrome`                                    | **→ home.go**        |
| 129–142  | `func (h *handlers) modelsPanel`                                          | models.go (stays)    |
| 144–166  | `func (h *handlers) setProcessor`                                         | models.go (stays)    |
| 168–187  | `downloadStoreKey`, `kronkOfficialByKey`, `claimInFlight`                 | **→ models_install.go** |
| 189–300  | `func (h *handlers) downloadOfficialModel`                                | **→ models_install.go** |
| 302–311  | `func (h *handlers) cancelDownload`                                       | **→ models_install.go** |
| 313–337  | `formatProgress`, `humanBytes`                                           | **→ models_install.go** |
| 339–398  | `selectModel`, `cloudAckKey`, `cloudConsentDialog`                        | models.go (stays)    |
| 400–482  | `saveCloudModel`, `saveCloudPreset`, `confirmCloudModel`, `deleteCloudModel` | models.go (stays) |
| 484–550  | `runtimeInstallStoreKey`, `kronkInstallRuntime`, `installRuntime`         | **→ models_install.go** |

The renderers for the dock view-model (`chatBarNode`, `composerNode`,
`modelSwitcherNode`, `headSwitcherNode`) already live in
`internal/web/home.go` — that's where the view-model belongs. `home.go`'s
existing imports (lines 8–18):

```go
import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
	"github.com/alexradunet/balaur/internal/ui/shell"
)
```

> **html-alias note**: `home.go` aliases the html package as `h` and uses `h` as
> the method receiver in its methods — these don't collide because Go resolves
> `h` lexically (the receiver shadows the alias inside methods). The four moved
> methods (`homeData`/`chatbar`/`patchChatbar`/`refreshDockChrome`) use the
> receiver `h`, `datastar`, `store`, `turn`, `heads`, `os`, `time` — **not** the
> html alias — so the move is conflict-free. (`models.go`, by contrast, aliases
> html as `hh` for the same reason; that alias travels with `installRuntime` to
> `models_install.go`.)

### Import sets after the split (derived from the symbol uses)

**`home.go`** — ADD to its existing imports: `os`, `time`,
`github.com/starfederation/datastar-go/datastar`,
`github.com/alexradunet/balaur/internal/store`,
`github.com/alexradunet/balaur/internal/turn`,
`github.com/alexradunet/balaur/internal/heads`. (It already has `core` and `g`.)

**`models.go`** (after) — exactly:
```go
import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)
```
(It LOSES `context`, `errors`, `fmt`, `os`, `path/filepath`, `time`, `g`, `hh`,
`heads`, `kronk/modelget`. It KEEPS `kronk` — `setProcessor` calls
`kronk.RuntimeInstalledFor`.)

**`models_install.go`** (new) — exactly:
```go
import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"
	hh "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/kronk/modelget"
	"github.com/alexradunet/balaur/internal/store"
)
```

These are derived, not guessed — but `go build` is the final arbiter. If the
compiler reports an unused or missing import after the move, adjust that file's
import set to match its actual uses.

## Commands you will need

| Purpose   | Command                                   | Expected            |
|-----------|-------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`            | exit 0              |
| Vet       | `go vet ./...`                            | exit 0              |
| Test pkg  | `go test ./internal/web/...`              | PASS                |
| Full test | `go test ./...`                           | all pass            |
| gofmt     | `gofmt -l internal/web`                   | prints nothing      |
| Sizes     | `wc -l internal/web/models.go internal/web/home.go internal/web/models_install.go` | models.go ~150 |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/web/models.go` (remove the moved symbols; trim imports)
- `internal/web/home.go` (receive the dock view-model; add imports)
- `internal/web/models_install.go` (create; receive both SSE orchestrators + helpers)

**Out of scope** (do NOT touch):
- Any caller of the moved symbols — all are package-internal; no call sites change.
- The route registrations (wherever `downloadOfficialModel`/`installRuntime`/
  `selectModel`/etc. are mounted) — same package, unchanged handler identities.
- The two SSE handlers' logic — verbatim move only. Do NOT introduce a shared
  "progress orchestrator" interface.
- `internal/feature/modelcards`, `settingscards`, `kronk`, `kronk/modelget` —
  unchanged.

## Git workflow

- Branch: `advisor/205-web-models-godfile-split`
- Conventional-commit subject, e.g. `refactor(web): split models.go into home view-model + install orchestrators`
- Suggest one commit per step so a bisect can isolate a bad move.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

> Order matters: move the dock view-model first (step 1), then the install
> orchestrators (step 2), so `models.go` shrinks last and you can confirm its
> trimmed import set against what remains.

### Step 1: Move the dock view-model into `home.go`

Cut lines 27–127 from `models.go` (the `homeData` struct through
`refreshDockChrome`) and paste them into `internal/web/home.go` (anywhere
sensible — e.g. just below the import block, above `navDestinations`). Add the
six new imports to `home.go` listed above.

**Verify**:
- `grep -n "type homeData struct\|func (h \*handlers) homeData\|func (h \*handlers) chatbar\|func (h \*handlers) refreshDockChrome" internal/web/home.go` → 4 matches
- `grep -n "type homeData struct\|func (h \*handlers) chatbar" internal/web/models.go` → no matches
- `go build ./internal/web/...` → exit 0
- `go vet ./internal/web/...` → exit 0

### Step 2: Move the SSE orchestrators into `models_install.go`

Create `internal/web/models_install.go` with the import block listed above and a
short top-of-file comment, then move these symbols verbatim from `models.go`:
`downloadStoreKey`, `kronkOfficialByKey`, `claimInFlight`,
`downloadOfficialModel`, `cancelDownload`, `formatProgress`, `humanBytes`,
`runtimeInstallStoreKey`, `kronkInstallRuntime`, `installRuntime`.

Suggested header:

```go
package web

// models_install.go holds the two long-lived model-install SSE orchestrators —
// the curated GGUF download (downloadOfficialModel) and the llama.cpp runtime
// install (installRuntime) — plus their shared single-flight machinery: the two
// app.Store() cancel-func sidecars (downloadStoreKey/runtimeInstallStoreKey),
// claimInFlight, and the progress formatters. Both re-render the models panel
// (modelsPanel, in models.go) when they finish. Split out of models.go (plan 205).
```

**Verify**:
- `grep -n "func (h \*handlers) downloadOfficialModel\|func (h \*handlers) installRuntime\|func claimInFlight\|func formatProgress\|func humanBytes" internal/web/models_install.go` → 5 matches
- `grep -n "downloadOfficialModel\|installRuntime\|claimInFlight" internal/web/models.go` → no matches (definitions gone; modelsPanel/setProcessor stay)
- `go build ./internal/web/...` → exit 0

### Step 3: Trim `models.go` imports and confirm size

`models.go` now contains only `modelsPanel`, `setProcessor`, `selectModel`,
`cloudAckKey`, `cloudConsentDialog`, `saveCloudModel`, `saveCloudPreset`,
`confirmCloudModel`, `deleteCloudModel`. Replace its import block with the
trimmed set listed above. Run `gofmt`.

**Verify**:
- `go build ./internal/web/...` → exit 0 (proves the trimmed imports match uses)
- `go vet ./internal/web/...` → exit 0
- `gofmt -l internal/web` → prints nothing
- `wc -l internal/web/models.go` → ~150 lines

### Step 4: Full verification

**Verify**:
- `go vet ./...` → exit 0
- `go test ./internal/web/...` → PASS
- `go test ./...` → all pass
- `git diff --check` → no whitespace errors

## Test plan

- No new tests. This is a verbatim relocation within `package web`; the existing
  `internal/web` tests (dock/home rendering, model selection, cloud consent,
  download/install SSE — e.g. any `*_test.go` exercising `downloadOfficialModel`
  via the `kronkOfficialByKey`/`kronkInstallRuntime` seams) must pass unchanged.
- The build itself proves the import sets are correct. The test suite proves
  behavior is unchanged.
- Verification: `go test ./internal/web/...` → PASS with the same test count.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `internal/web/models_install.go` exists and defines `downloadOfficialModel`, `installRuntime`, `claimInFlight`
- [ ] `internal/web/home.go` defines `homeData` (struct + method), `chatbar`, `patchChatbar`, `refreshDockChrome`
- [ ] `grep -n "downloadOfficialModel\|installRuntime\|type homeData struct" internal/web/models.go` returns no matches
- [ ] `wc -l internal/web/models.go` reports ≤ ~170 lines
- [ ] `gofmt -l internal/web` prints nothing
- [ ] `go test ./...` exits 0
- [ ] Only `models.go` modified, `home.go` modified, `models_install.go` created (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- After moving the dock view-model, `home.go` fails to build with an `h` symbol
  ambiguity (html alias vs receiver) you can't resolve by confirming the moved
  methods use the receiver `h` and never `h.Class`/`h.Div` — report the exact
  line.
- A symbol you moved turns out to be referenced from OUTSIDE `package web`
  (it shouldn't be — all are unexported): `grep -rn "downloadOfficialModel\|homeData\|claimInFlight" internal/ --include=*.go | grep -v internal/web` returns matches.
- The trimmed import set for any file can't be made to satisfy `go build` by
  adding/removing imports alone — the symbol partition is wrong; re-confirm the
  table against the drift check.
- You feel tempted to introduce a shared interface/struct to "DRY up" the two
  SSE handlers — STOP; that's explicitly out of scope.

## Maintenance notes

- After this, `home.go` owns the home/dock view-model end to end (data + render);
  `models.go` is model selection + cloud consent; `models_install.go` is the two
  background-job orchestrators with their cancel sidecars.
- The two `app.Store()` cancel-func sidecars (`downloadStoreKey`,
  `runtimeInstallStoreKey`) now live in one file — the only place those keys are
  referenced. A future cancel/timeout change touches just `models_install.go`.
- Reviewer: confirm the moves are verbatim (the diff should be pure relocation +
  import-set adjustments), and that `models.go` dropped below the ~500-LOC line.
