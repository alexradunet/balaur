# Plan 186: Split settingsfocus.go into one file per settings section

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 12a48bf..HEAD -- internal/feature/settingscards/`
> If `settingsfocus.go` changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

`internal/feature/settingscards/settingsfocus.go` is 619 lines — the largest
non-fixture/non-seed file in the feature tree, past AGENTS.md's "treat a package
past ~500 lines as a smell to decompose" threshold (the rule applies to files in
practice; this is one file holding five independent settings sections). It
bundles the Profile, Models, Heads, Nudge, and Capabilities sections into one
unit, so editing any one section means scrolling an unrelated 600-line file. It
also imports both `headscards` and `modelcards` to re-render those features
inside the settings card, making it the cross-feature settings hub — a file you
must touch often.

This is a **pure mechanical move**: cut each section's view-models, builder, and
render function into its own file in the *same package*. Zero behavior change.
After it lands, each settings section is a ~100–280-line file you can read in one
screen, and the aggregate dispatcher is a slim ~90-line file.

## Current state

All code below lives in **one file**: `internal/feature/settingscards/settingsfocus.go`
(619 lines, package `settingscards`). The split keeps the same package, so every
symbol stays visible to every other file in the package — no export changes, no
caller changes.

### The package today

```
internal/feature/settingscards/
  register.go               (23 lines)  Register/Unregister + init()
  settings.go               (49 lines)  SettingsCard() static tile + registerSettings()
  settingsfocus.go          (619 lines) ← THE FILE TO SPLIT
  settingscards_test.go     (10 lines)  TestNoWebImports
  settings_test.go          (28 lines)
  settingsfocus_test.go     (198 lines) section render contracts
```

### The import block (settingsfocus.go:15-38, VERBATIM)

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/ardanlabs/kronk/sdk/tools/libs"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/self"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/ui"
)
```

### Symbol → destination-file map (verified by reading every line)

| Symbol | Lines today | Destination file |
|--------|-------------|------------------|
| `ProfileAvatarOption` (struct) | 60-66 | `settingsfocus_profile.go` |
| `ProfileView` (struct) | 68-75 | `settingsfocus_profile.go` |
| `BuildProfile` | 121-130 | `settingsfocus_profile.go` |
| `buildAvatarOptions` (unexported) | 132-153 | `settingsfocus_profile.go` |
| `buildBalaurHeadOptions` (unexported) | 155-170 | `settingsfocus_profile.go` |
| `ProfileIdentityCard` | 391-424 | `settingsfocus_profile.go` |
| `ProfileSoulSection` | 426-462 | `settingsfocus_profile.go` |
| `ProfileBalaurSection` | 464-499 | `settingsfocus_profile.go` |
| `cloudPresetViews` (unexported) | 40-54 | `settingsfocus_models.go` |
| `BuildModelsPanelView` | 172-282 | `settingsfocus_models.go` |
| `ExamplePanelView` | 368-385 | `settingsfocus_models.go` |
| `NudgeView` (struct) | 87-91 | `settingsfocus_nudges.go` |
| `BuildNudge` | 314-324 | `settingsfocus_nudges.go` |
| `NudgeSection` | 526-568 | `settingsfocus_nudges.go` |
| `CapabilitiesView` (struct) | 93-103 | `settingsfocus_capabilities.go` |
| `GateView` (struct) | 105-109 | `settingsfocus_capabilities.go` |
| `ExtStatusView` (struct) | 111-115 | `settingsfocus_capabilities.go` |
| `BuildCapabilities` | 326-366 | `settingsfocus_capabilities.go` |
| `CapabilitiesSection` | 570-619 | `settingsfocus_capabilities.go` |
| `SettingsFocusView` (struct) | 77-85 | stays in `settingsfocus.go` |
| `BuildSettingsFocus` | 284-312 | stays in `settingsfocus.go` |
| `SettingsFocus` (dispatcher) | 501-524 | stays in `settingsfocus.go` |

There is **no Heads section file** — Heads has no view-model, builder, or render
func of its own in this file. The aggregate just delegates: the struct field
`Heads headscards.HeadsView`, the builder call `headscards.BuildHeads(app)`, and
the render `headscards.HeadsCard(v.Heads)` all reference the `headscards`
package. So Heads stays in the slim `settingsfocus.go`. **Do not create
`settingsfocus_heads.go`** — there is nothing to put in it. (The brief listed
five files; reality has four section files + the aggregate. Record this in your
done-report.)

### Two unexported helpers — used by exactly one section each (confirmed)

`cloudPresetViews` (line 43) is called only by `BuildModelsPanelView` (line 182)
and `ExamplePanelView` (line 373) → both go to `settingsfocus_models.go`, so the
helper goes there too. `buildAvatarOptions` / `buildBalaurHeadOptions` are called
only by `BuildProfile` → all three go to `settingsfocus_profile.go`. No
unexported state is shared across sections, so the split needs no signature
changes.

### Which imports each new file needs (verified by grepping every usage)

This is the load-bearing detail of the whole plan. Use exactly these import sets
— `goimports`/`gofmt` will NOT add a missing import, it only formats, and an
unused import is a compile error. Get these right and the build is green on the
first try.

**`settingsfocus_profile.go`** — uses `core` (BuildProfile/buildAvatarOptions
take `core.App`), `store` (OwnerName/GetOwnerSetting/SoulAvatars/BalaurHeads),
`g`, `data`, `h`:
```go
import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/store"
)
```

**`settingsfocus_models.go`** — uses `fmt`, `os`, `filepath`, `runtime`, `core`,
`libs`, `modelcards`, `kronk`, `llm`, `store`, `turn` (does NOT use `g`, `data`,
`h`, `ui`, `self`, `strings`, `time` — it builds view-models, not markup; the
markup is `modelcards.Panel`, which lives in another package):
```go
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pocketbase/pocketbase/core"

	"github.com/ardanlabs/kronk/sdk/tools/libs"

	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)
```

**`settingsfocus_nudges.go`** — uses `time` (BuildNudge parses RFC3339), `core`,
`store`, `g`, `data`, `h`:
```go
import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/store"
)
```

**`settingsfocus_capabilities.go`** — uses `fmt` (Sprintf), `strings` (TrimSpace),
`core`, `self`, `turn`, `g`, `h`, `ui` (NOT `data` — the capability roster has no
forms/@post; NOT `store`):
```go
import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/self"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/ui"
)
```

**`settingsfocus.go` (the remaining slim file)** — keeps `SettingsFocusView`,
`BuildSettingsFocus`, `SettingsFocus`. `BuildSettingsFocus` calls
`BuildModelsPanelView` / `headscards.BuildHeads` / `BuildNudge` /
`BuildCapabilities` / `BuildProfile`; the struct embeds `modelcards.PanelView`
and `headscards.HeadsView`; `SettingsFocus` calls `modelcards.Panel` and
`headscards.HeadsCard`. So it uses `core`, `g`, `h`, `headscards`, `modelcards`
(NOT `data`, `store`, `fmt`, `strings`, `time`, `os`, `filepath`, `runtime`,
`libs`, `kronk`, `llm`, `self`, `turn`, `ui`):
```go
import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/modelcards"
)
```

### Callers that must keep compiling unchanged (do NOT touch these)

Every external reference is to an **exported** symbol; none touches the
unexported helpers. Because the split stays in-package, all of these resolve
exactly as before:

- `internal/web/profile.go:22,24,43,45,64,66` — `BuildProfile`,
  `ProfileIdentityCard`, `ProfileSoulSection`, `ProfileBalaurSection`
- `internal/web/models.go:130,241,515` and `internal/web/models_test.go:133,154,178`
  — `BuildModelsPanelView`
- `internal/web/nudge.go:24` — `NudgeSection`, `BuildNudge`
- `internal/web/settings_gomponents_test.go:17` — `Register`
- `internal/feature/storybook/stories_cards.go:107-135,728-754` — `NudgeSection`,
  `NudgeView`, `CapabilitiesSection`, `CapabilitiesView`, `GateView`,
  `ExtStatusView`, `SettingsFocusView`, `ProfileView`, `ProfileAvatarOption`,
  `ExamplePanelView`, `SettingsFocus`
- `internal/feature/settingscards/settingsfocus_test.go` — `ProfileView`,
  `ProfileIdentityCard`, `ProfileSoulSection`, `ProfileBalaurSection`,
  `SettingsFocusView`, `ProfileAvatarOption`, `ExamplePanelView`, `SettingsFocus`
- `internal/feature/settingscards/settings.go:41,45` —
  `BuildSettingsFocus`, `SettingsFocus`

### Repo conventions that apply

- **gofmt is law** — a PostToolUse hook + CI gofmt gate reject unformatted files.
  Run `gofmt -w` on every new file before verifying. Match the existing import
  grouping you see in this file: stdlib block, then a blank line, then
  third-party (`pocketbase`, `gomponents`), then (for `modelcards`/`kronk`)
  another group, then the `internal/...` block. When in doubt, run
  `gofmt -w <file>` and accept its grouping.
- **This is a verbatim move.** Cut each function/struct *exactly* as written —
  same body, same comments, same blank lines, same markup, same symbol names. Do
  not "improve", rename, reorder fields, or reflow. The storybook coverage test
  asserts byte-identical rendered output; any markup change fails it.
- **No new dependency, no behavior change, no migration, no `knowledge.md`
  update.** This is a file move within one package; the architecture description
  in `internal/self/knowledge.md` does not name this file and does not change.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build all | `CGO_ENABLED=0 go build ./...` | exit 0, no output |
| Test the package | `go test ./internal/feature/settingscards/` | `ok ...` |
| Test storybook (coverage) | `go test ./internal/feature/storybook/` | `ok ...` |
| Test callers | `go test ./internal/web/` | `ok ...` |
| Test all | `go test ./...` | all `ok`/`no test files` |
| Vet | `go vet ./...` | exit 0, no output |
| gofmt check | `gofmt -l .` | **empty output** (no files listed) |
| Diff whitespace check | `git diff --check` | exit 0, no output |
| Line counts | `wc -l internal/feature/settingscards/*.go` | no file > ~300 |

## Suggested executor toolkit

- Invoke the `go-standards` skill if available — it covers the gofmt/vet
  tool surface and the import-alias convention (`h "maragu.dev/gomponents/html"`,
  `g "maragu.dev/gomponents"`, `data "maragu.dev/gomponents-datastar"`) this
  plan relies on.

## Scope

**In scope** (the only files you create or modify):
- `internal/feature/settingscards/settingsfocus.go` (shrink to the aggregate)
- `internal/feature/settingscards/settingsfocus_profile.go` (create)
- `internal/feature/settingscards/settingsfocus_models.go` (create)
- `internal/feature/settingscards/settingsfocus_nudges.go` (create)
- `internal/feature/settingscards/settingsfocus_capabilities.go` (create)

**Out of scope** (do NOT touch, even though they reference these symbols):
- `internal/web/profile.go`, `internal/web/models.go`, `internal/web/nudge.go`,
  and their tests — they call exported symbols that do not move packages; they
  must keep compiling unchanged. If you find yourself editing one, you have
  changed a signature — STOP.
- `internal/feature/storybook/stories_cards.go` and the storybook coverage test
  — the rendered output must stay byte-identical.
- `internal/feature/settingscards/settings.go`, `register.go`, and any test file
  in the package — no edits needed.
- Any markup, class, element id, Datastar attribute, struct field, or function
  signature. This plan moves code; it does not change a single character of any
  function body.
- Do NOT create `settingsfocus_heads.go` — there is no Heads-only code to move
  (see "Current state").
- `internal/self/knowledge.md` — unchanged by a same-package file move.

## Git workflow

- Branch: `advisor/186-decompose-settingsfocus` off `origin/main` (executors run
  in a worktree off `origin/main`).
- Single commit is fine — this is one logical move. Conventional-commit subject,
  e.g. `refactor(settingscards): split settingsfocus.go into one file per section`.
  Match the log style (`git log --oneline -5`).
- Do NOT push or open a PR unless the operator instructed it. This repo lands on
  `main`; the operator says when.

## Steps

The strategy: create the four section files first (each compiles on its own once
its imports are right), then delete the moved code from `settingsfocus.go` and
fix its import block last. Between steps the tree may not build — that is
expected; the gate is the final build at Step 6. Build only after Step 6.

### Step 1: Create `settingsfocus_profile.go`

Create `internal/feature/settingscards/settingsfocus_profile.go` with
`package settingscards`, the profile import block from "Current state", and then,
**cut verbatim** from `settingsfocus.go`, these symbols in this order:
`ProfileAvatarOption` (60-66), `ProfileView` (68-75), `BuildProfile` (121-130),
`buildAvatarOptions` (132-153), `buildBalaurHeadOptions` (155-170),
`ProfileIdentityCard` (391-424), `ProfileSoulSection` (426-462),
`ProfileBalaurSection` (464-499). Keep each function's doc comment with it. Add a
one-line file header comment, e.g.
`// settingsfocus_profile.go — the Profile settings section: identity card, soul`
`// avatar, and Balaur head pickers. Split out of settingsfocus.go (plan 186).`

**Verify**: `gofmt -l internal/feature/settingscards/settingsfocus_profile.go`
→ empty (file is gofmt-clean). Do not build yet.

### Step 2: Create `settingsfocus_models.go`

Create the file with `package settingscards`, the models import block from
"Current state", and **cut verbatim**: `cloudPresetViews` (40-54),
`BuildModelsPanelView` (172-282), `ExamplePanelView` (368-385). Add a file header
comment. (This file has no markup — it builds `modelcards.PanelView` view-models;
that is why it imports neither `g`/`h`/`data` nor `ui`.)

**Verify**: `gofmt -l internal/feature/settingscards/settingsfocus_models.go`
→ empty.

### Step 3: Create `settingsfocus_nudges.go`

Create the file with `package settingscards`, the nudges import block, and **cut
verbatim**: `NudgeView` (87-91), `BuildNudge` (314-324), `NudgeSection`
(526-568). Add a file header comment.

**Verify**: `gofmt -l internal/feature/settingscards/settingsfocus_nudges.go`
→ empty.

### Step 4: Create `settingsfocus_capabilities.go`

Create the file with `package settingscards`, the capabilities import block, and
**cut verbatim**: `CapabilitiesView` (93-103), `GateView` (105-109),
`ExtStatusView` (111-115), `BuildCapabilities` (326-366), `CapabilitiesSection`
(570-619). Add a file header comment.

**Verify**: `gofmt -l internal/feature/settingscards/settingsfocus_capabilities.go`
→ empty.

### Step 5: Shrink `settingsfocus.go` to the aggregate

Delete from `settingsfocus.go` every symbol moved in Steps 1–4 (everything except
`SettingsFocusView`, `BuildSettingsFocus`, and `SettingsFocus`). Replace the
import block (lines 15-38) with the slim aggregate import block from "Current
state" (only `core`, `g`, `h`, `headscards`, `modelcards`). Keep the existing
top-of-file package + doc comment, or trim its body comment so it describes only
the aggregate dispatcher (optional — do not over-edit). After this step the file
should hold exactly three symbols.

**Verify**: `gofmt -l internal/feature/settingscards/settingsfocus.go` → empty.
Then run the gate in Step 6.

### Step 6: Build, format, and test the whole tree

Run, in order:

1. `gofmt -w internal/feature/settingscards/` — normalize all five files.
2. `gofmt -l .` → **empty** (nothing left unformatted).
3. `CGO_ENABLED=0 go build ./...` → exit 0. **If this fails with "imported and
   not used" or "undefined", an import set is wrong** — re-check the per-file
   import blocks in "Current state" against the symbols you actually moved into
   that file, then re-run. This is the most likely failure and the import blocks
   above are exact.
4. `go vet ./...` → exit 0.
5. `go test ./internal/feature/settingscards/ ./internal/feature/storybook/ ./internal/web/`
   → all `ok`.
6. `git diff --check` → no output.

**Verify**: all six commands succeed.

## Test plan

No new tests. This is a move; the existing tests are the regression guard:

- `internal/feature/settingscards/settingsfocus_test.go` — render-contract tests
  for `ProfileIdentityCard`, `ProfileSoulSection`, `ProfileBalaurSection`, and
  `SettingsFocus` across sections. These assert exact ids/classes/Datastar
  attributes; if any moved markup changed, they fail.
- `internal/feature/storybook/` coverage test — asserts the storybook renders
  every component; the rendered HTML must be byte-identical.
- `internal/web/` tests (`models_test.go`, `profile.go` handlers,
  `settings_gomponents_test.go`) — exercise the callers; they fail if a signature
  moved or a symbol disappeared.

Verification: `go test ./...` → all pass, **same count as before the change**
(no tests added, none removed). If any settingscards/storybook/web test that
passed at `12a48bf` now fails, you changed behavior — STOP.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` prints nothing
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./internal/feature/settingscards/` passes
- [ ] `go test ./internal/feature/storybook/` passes
- [ ] `go test ./internal/web/` passes
- [ ] `go test ./...` passes (full suite green before any push)
- [ ] `git diff --check` exits 0
- [ ] `wc -l internal/feature/settingscards/settingsfocus*.go` shows no file
      over ~300 lines (the slim `settingsfocus.go` ≈ 90; `settingsfocus_models.go`
      ≈ 140; `settingsfocus_profile.go` ≈ 180; the rest smaller)
- [ ] No file outside the in-scope list is modified (`git status` shows only the
      five settingscards files: one modified, four new)
- [ ] `git diff` for `settingsfocus.go` (and `git show` of the new files) is
      moves only — no edited function bodies, no renamed symbols, no markup change
- [ ] `plans/README.md` status row for plan 186 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `settingsfocus.go` changed since `12a48bf` and the
  "Current state" excerpts (the import block, the symbol line ranges) no longer
  match the live file. Adapt only after reporting.
- **A clean split would require changing a function signature** because two
  sections share unexported state. This plan's analysis found NONE — the only
  unexported helpers (`cloudPresetViews`, `buildAvatarOptions`,
  `buildBalaurHeadOptions`) are each used by exactly one section. If reality
  differs, do the minimal cohesive split: keep the genuinely-coupled pair in one
  file, and report exactly which symbols you grouped and why.
- `go build` still fails after you have double-checked the per-file import blocks
  against the moved symbols twice. (Do not start deleting imports by trial and
  error beyond what "Current state" specifies — report the exact compiler error.)
- Any settingscards/storybook/web test that passed at `12a48bf` now fails —
  that means the move was not verbatim. Diff the failing render against the
  original and report.
- You find yourself needing to edit any out-of-scope file
  (`internal/web/*`, `stories_cards.go`, `settings.go`, `register.go`,
  `knowledge.md`) — that means a symbol changed shape; STOP.

## Maintenance notes

For the reviewer and the next maintainer:

- **What a reviewer should scrutinize**: that the diff is moves only. The fastest
  check is `git show <new-file>` should be familiar code, and the deletions from
  `settingsfocus.go` should equal the additions across the four new files (modulo
  the per-file `package`/import headers). Confirm no function body, class, id, or
  Datastar attribute string changed.
- **Why there is no `settingsfocus_heads.go`**: Heads has no section-local
  view-model/builder/render in this package — it delegates wholly to
  `headscards`. The Heads wiring (struct field, `BuildHeads`, `HeadsCard`) lives
  in the aggregate `settingsfocus.go` alongside the dispatcher, which is correct.
- **Future work this unblocks**: when a new settings section is added (e.g.
  Appearance — `settings.go:31` already links a `section=appearance` that
  currently falls back to profile in `BuildSettingsFocus`), it gets its own
  `settingsfocus_<name>.go` file plus a `case` in the two switches in
  `settingsfocus.go`. The aggregate is the only file a new section touches beyond
  its own.
- **No deferred work.** This is complete as a unit.
