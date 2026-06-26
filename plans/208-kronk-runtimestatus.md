# Plan 208: Add `kronk.RuntimeStatus` and drop the raw `ardanlabs/kronk` SDK import from `internal/feature/settingscards`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/kronk/librt.go internal/feature/settingscards/settingsfocus_models.go`
> If either file changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, treat it as a
> STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

`internal/feature/settingscards/settingsfocus_models.go` imports
`github.com/ardanlabs/kronk/sdk/tools/libs` directly — the **only** raw kronk-SDK
import anywhere outside `internal/kronk`. A UI view-builder thus depends on the
SDK's library-layout contract (`libs.IsSupported`, `libs.ReadVersionFile`, plus
the host triple math). A kronk upgrade that changes that layout would ripple
into a `feature/` package instead of being contained in `internal/kronk`, which
is supposed to be the single seam over the dlopen engine.

The fix: add `kronk.RuntimeStatus(processor) (supported bool, version string)`
inside `internal/kronk` (where `libs` is already imported, in `librt.go`),
computing the host triple itself. `settingscards` then calls it and maps the two
scalars to its own `modelcards.Status*` constants. **Return facts (two scalars),
not a presentation string** — UI vocabulary stays in the UI.

## Current state

### The leak — `settingsfocus_models.go`

Import (line 13): `"github.com/ardanlabs/kronk/sdk/tools/libs"`.

The runtime-section loop (lines 104–125) is the only user:

```go
// Build the runtime section: cpu and vulkan rows.
goos := runtime.GOOS
goarch := runtime.GOARCH
for _, proc := range []string{"cpu", "vulkan"} {
	rv := modelcards.RuntimeView{
		Processor:       proc,
		NeedsHostLoader: proc == "vulkan",
	}
	if !libs.IsSupported(goarch, goos, proc) {
		rv.Status = modelcards.StatusUnsupported
	} else {
		dir := kronk.InstallDirFor(kronk.LibRoot(), goarch, goos, proc)
		vt, err := libs.ReadVersionFile(dir)
		if err == nil && vt.Version != "" {
			rv.Status = modelcards.StatusInstalled
			rv.Version = vt.Version
		} else {
			rv.Status = modelcards.StatusAvailable
		}
	}
	view.RuntimeSection = append(view.RuntimeSection, rv)
}
```

`runtime` (lines 105–106) and `libs` (lines 112, 116) are used ONLY in this
block. `runtime.GOOS`/`GOARCH` also flow into the now-removed `libs`/
`InstallDirFor` calls. After the change, `settingscards` no longer needs
`runtime` or `libs`. (`os`, `path/filepath`, `kronk` stay — used elsewhere in the
file, e.g. `filepath.Join`/`os.Stat`/`kronk.ModelsDir` in the official-catalog
loop, and `kronk.EstimateVRAM`, `kronk.RuntimeInstalled`, `kronk.Processor`,
`kronk.FromStore`.)

### The kronk file that already wraps `libs` — `internal/kronk/librt.go`

It imports `runtime` and `libs`, and already exposes `InstallDirFor` and
`RuntimeVersion`. The existing usage of the SDK version-file read is the same one
`settingscards` duplicates:

```go
// librt.go already does:
func InstallDirFor(root, arch, goos, processor string) string {
	return filepath.Join(root, goos, arch, processor)
}
// and InstallRuntime() calls libs.IsSupported(goarch, goos, processor) with
// goos := runtime.GOOS; goarch := runtime.GOARCH.
```

`libs.ReadVersionFile(dir)` returns a value with a `.Version` field (as used at
`settingsfocus_models.go:116-119`).

### The `modelcards.Status*` constants the UI maps to

`StatusUnsupported`, `StatusInstalled`, `StatusAvailable` (from
`internal/feature/modelcards`) — already used in the loop above; unchanged.

## Commands you will need

| Purpose   | Command                                                       | Expected         |
|-----------|---------------------------------------------------------------|------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                                | exit 0           |
| Vet       | `go vet ./...`                                                 | exit 0           |
| Test pkg  | `go test ./internal/kronk/... ./internal/feature/settingscards/...` | PASS       |
| Full test | `go test ./...`                                                | all pass         |
| gofmt     | `gofmt -l internal/kronk internal/feature/settingscards`      | prints nothing   |
| Leak check| `grep -rn "ardanlabs/kronk" internal/ --include=*.go \| grep -v internal/kronk` | no matches |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/kronk/librt.go` (add `RuntimeStatus`)
- `internal/feature/settingscards/settingsfocus_models.go` (call it; drop `libs` + `runtime` imports)

**Out of scope** (do NOT touch):
- The rest of `settingsfocus_models.go` (model list, official catalog, processor
  selection) — unchanged.
- `kronk.InstallDirFor`/`RuntimeVersion`/`RuntimeInstalled`/`RuntimeInstalledFor`
  — unchanged (RuntimeStatus is additive).
- `internal/feature/modelcards` — unchanged.

## Git workflow

- Branch: `advisor/208-kronk-runtimestatus`
- Conventional-commit subject, e.g.
  `refactor(kronk): add RuntimeStatus, drop SDK import from settingscards`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `kronk.RuntimeStatus` to `internal/kronk/librt.go`

Add (next to `InstallDirFor`):

```go
// RuntimeStatus reports whether the llama.cpp runtime for processor ("cpu" or
// "vulkan") is supported on this host, and — if installed — its version string.
// It wraps the SDK's libs.IsSupported + version-file read and computes the host
// triple itself, so callers (e.g. the Models settings view) need not import the
// kronk SDK. supported=false means the triple is not a supported build; an empty
// version with supported=true means supported-but-not-installed.
func RuntimeStatus(processor string) (supported bool, version string) {
	goos, goarch := runtime.GOOS, runtime.GOARCH
	if !libs.IsSupported(goarch, goos, processor) {
		return false, ""
	}
	dir := InstallDirFor(LibRoot(), goarch, goos, processor)
	if vt, err := libs.ReadVersionFile(dir); err == nil {
		return true, vt.Version
	}
	return true, ""
}
```

`librt.go` already imports `runtime` and `libs`, so no import changes here.

**Verify**: `go build ./internal/kronk/...` → exit 0

### Step 2: Call it from `settingsfocus_models.go`; drop `libs` + `runtime`

Replace the runtime-section loop (lines 104–125) with:

```go
	// Build the runtime section: cpu and vulkan rows. Status comes from
	// kronk.RuntimeStatus (the kronk seam over the SDK), mapped to UI constants
	// here — UI vocabulary stays in the UI.
	for _, proc := range []string{"cpu", "vulkan"} {
		rv := modelcards.RuntimeView{
			Processor:       proc,
			NeedsHostLoader: proc == "vulkan",
		}
		supported, version := kronk.RuntimeStatus(proc)
		switch {
		case !supported:
			rv.Status = modelcards.StatusUnsupported
		case version != "":
			rv.Status = modelcards.StatusInstalled
			rv.Version = version
		default:
			rv.Status = modelcards.StatusAvailable
		}
		view.RuntimeSection = append(view.RuntimeSection, rv)
	}
```

Then remove the imports now unused in this file: `"github.com/ardanlabs/kronk/sdk/tools/libs"` and `"runtime"`. Run `gofmt`.

**Verify**:
- `grep -n "ardanlabs/kronk\|\"runtime\"\|libs\.\|runtime\.GOOS\|runtime\.GOARCH" internal/feature/settingscards/settingsfocus_models.go` → no matches
- `go build ./internal/feature/settingscards/...` → exit 0
- `go vet ./internal/feature/settingscards/...` → exit 0

### Step 3: Full verification

**Verify**:
- `gofmt -l internal/kronk internal/feature/settingscards` → prints nothing
- `grep -rn "ardanlabs/kronk" internal/ --include=*.go | grep -v internal/kronk` → no matches
- `go vet ./...` → exit 0
- `go test ./internal/kronk/... ./internal/feature/settingscards/...` → PASS
- `go test ./...` → all pass

## Test plan

- Add `TestRuntimeStatus` in `internal/kronk/librt_test.go` (the file exists):
  - For the host's actual cpu triple, `supported` should be true (cpu is
    supported on linux/amd64 and the CI targets). Assert `supported == true`.
  - On a box with no installed runtime, `version` is `""` (supported but not
    installed). This is the default test-box state. Assert `version == ""` when
    no runtime is installed (use a `kronk.LibRoot()` pointing at an empty temp
    dir if the existing tests provide a way to override the root; otherwise
    assert only the `supported` half, which is deterministic).
  - For a deliberately bogus processor (e.g. `"nonsense"`), `supported` is
    false. Assert `supported == false`.
  - Follow the existing `librt_test.go` patterns for any root/seam overrides.
- The settingscards view builder is covered by existing `settingscards`/
  storybook tests (`BuildModelsPanelView`/`ExamplePanelView`); they must pass
  unchanged.
- Verification: `go test ./internal/kronk/...` → PASS including `TestRuntimeStatus`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `grep -rn "func RuntimeStatus" internal/kronk/` returns one match
- [ ] `grep -rn "ardanlabs/kronk" internal/ --include=*.go | grep -v internal/kronk` returns no matches
- [ ] `grep -n "\"runtime\"" internal/feature/settingscards/settingsfocus_models.go` returns no matches
- [ ] `go test ./...` exits 0; `TestRuntimeStatus` exists and passes
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `libs.ReadVersionFile` returns a type whose version field is NOT `.Version`
  (the drift check shows a kronk SDK bump) — adjust `RuntimeStatus` to the real
  field and report.
- `libs.IsSupported` or `kronk.InstallDirFor`/`kronk.LibRoot` no longer exist with
  the signatures shown — report.
- Removing `runtime` from `settingsfocus_models.go` breaks the build because
  `runtime` is used elsewhere in the file you didn't see — keep it and report
  (only `libs` removal is then strictly required).

## Maintenance notes

- After this, `internal/kronk` is the ONLY package importing the kronk SDK. A
  future SDK upgrade's blast radius is contained there.
- `RuntimeStatus` returns two scalars by design — if a caller wants a
  presentation string, it maps the scalars itself (as `settingscards` now does).
  Do not push UI status constants down into `kronk`.
- Reviewer: confirm no `feature/` or `web/` package imports `ardanlabs/kronk`
  after this, and that the runtime-section statuses render the same as before.
