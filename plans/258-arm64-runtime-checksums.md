# Plan 258: Pin linux/arm64 runtime checksums so arm64 installs are fail-closed like amd64

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/kronk/ AGENTS.md`
> (`internal/kronk/` covers both the in-scope files and `librt.go`, whose
> excerpts below are read-only context.) If any of these files changed since
> this plan was written, compare the "Current state" excerpts against the
> live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: direction
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Balaur's owner-initiated llama.cpp runtime install verifies every downloaded
native library against an embedded sha256 manifest (`internal/kronk/runtime_sums.json`)
before the engine ever `dlopen`s it — but only for `linux/amd64`. The
`linux/arm64` cpu and vulkan entries are empty-string placeholders, and the
verifier deliberately **skips** empty hashes, so an arm64 owner (Raspberry
Pi 5, Graviton — plausible sovereign boxes) downloads a native library over
the network and loads it into the process **unverified**. That is exactly the
class of risk the checksum gate exists to close. `AGENTS.md` documents this
as a deliberate v1 deferral; this plan closes it before packaging widens the
audience. After this lands, every Linux triple the install surface offers is
fail-closed, and a committed opt-in test can regenerate/audit the manifest
whenever the pinned runtime version is bumped.

## Current state

### The manifest — `internal/kronk/runtime_sums.json` (entire file)

`internal/kronk/runtime_sums.json:1-18`:

```json
{
  "b9664": {
    "linux/amd64/cpu": {
      "libllama.so": "552f47357c0cebb6887e983b9b0ee079a75e0b18dd0ae570a1d79f6181f2c43a"
    },
    "linux/amd64/vulkan": {
      "libllama.so": "552f47357c0cebb6887e983b9b0ee079a75e0b18dd0ae570a1d79f6181f2c43a",
      "libggml-vulkan.so": "2277c57ed00b6d3d46c8aa9a4b86fb057c55703d8d64455870f6b055f7589a35"
    },
    "linux/arm64/cpu": {
      "libllama.so": ""
    },
    "linux/arm64/vulkan": {
      "libllama.so": "",
      "libggml-vulkan.so": ""
    }
  }
}
```

The four empty strings are the placeholders this plan fills.

### The verifier — `internal/kronk/librt.go` (read-only in this plan)

The pinned version, `internal/kronk/librt.go:17-19`:

```go
// runtimeVersion is the pinned llama.cpp release installed by InstallRuntime.
// Changing this is a reviewed code change — it aligns with what DownloadFor pins.
const runtimeVersion = "b9664"
```

`InstallRuntime` always installs the **host** triple, `internal/kronk/librt.go:32-36`:

```go
func InstallRuntime(ctx context.Context, processor string, log libs.Logger) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if !libs.IsSupported(goarch, goos, processor) {
		return fmt.Errorf("runtime %s/%s/%s is not a supported build", goarch, goos, processor)
	}
```

The real download path stages via the SDK's triple-aware `DownloadFor`,
`internal/kronk/librt.go:60-66`:

```go
	lib, err := libs.New(libs.WithLibPath(staging))
	if err != nil {
		return fmt.Errorf("resolving lib root: %w", err)
	}
	if _, err := lib.DownloadFor(ctx, log, goarch, goos, processor, runtimeVersion); err != nil {
		return fmt.Errorf("installing %s runtime: %w", processor, err)
	}
```

The empty-hash skip that makes arm64 installs unverified today,
`internal/kronk/librt.go:129-138` (inside `verifyInstall`):

```go
	key := goos + "/" + arch + "/" + processor
	fileHashes, ok := versionEntry[key]
	if !ok {
		return fmt.Errorf("verifyInstall: no manifest entry for %q @ version %q", key, version)
	}
	for filename, want := range fileHashes {
		if want == "" {
			// Placeholder — filled at merge (Reviewer Note 3).
			continue
		}
```

Unexported helpers the new tests reuse (same package): `runtimeSums`
(`librt.go:111`, the unmarshalled manifest map, keyed
version → "os/arch/proc" → filename → sha256), `verifyInstall`
(`librt.go:124`), `sha256File` (`librt.go:153`), `InstallDirFor`
(`librt.go:85`, returns `<root>/<os>/<arch>/<processor>`).

### Existing tests — `internal/kronk/librt_test.go`

The structural pattern to match, `internal/kronk/librt_test.go:71-90`
(`TestVerifyInstall_Mismatch`): writes a wrong-content `libllama.so` into
`t.TempDir()`, swaps `runtimeSums` for a fake map, expects `verifyInstall`
to error AND delete the dir. `TestVerifyInstall_EmptyHash`
(`librt_test.go:105-116`) locks in the placeholder-skip behavior — it stays
as-is (the skip is still correct behavior for a hypothetical placeholder;
this plan removes the placeholders, it does not change the verifier).

### The exact artifact identity (verified 2026-07-01/02, HTTP HEAD → 302)

Balaur pins `github.com/ardanlabs/kronk v1.28.0` (go.mod:6), which delegates
the download to `github.com/hybridgroup/yzma v1.17.1` (go.mod:8). yzma's URL
selection for the linux triples, from the module cache
(`$GOPATH/pkg/mod/github.com/hybridgroup/yzma@v1.17.1/pkg/download/download.go:144-170`,
function `getDownloadLocationAndFilename`, `case Linux:`):

- `linux/arm64/cpu` → `https://github.com/hybridgroup/llama-cpp-builder/releases/download/b9664/llama-b9664-bin-ubuntu-cpu-arm64.tar.gz`
- `linux/arm64/vulkan` → `https://github.com/hybridgroup/llama-cpp-builder/releases/download/b9664/llama-b9664-bin-ubuntu-vulkan-arm64.tar.gz`
- `linux/amd64/cpu` → `https://github.com/ggml-org/llama.cpp/releases/download/b9664/llama-b9664-bin-ubuntu-x64.tar.gz`
- `linux/amd64/vulkan` → `https://github.com/ggml-org/llama.cpp/releases/download/b9664/llama-b9664-bin-ubuntu-vulkan-x64.tar.gz`

All four URLs returned HTTP 302 (asset exists) on 2026-07-02. Both arm64
tarballs were list-inspected: each has a single top-level `llama-b9664/` dir
(stripped on extraction by yzma's `downloadAndExtractTarGz`), the cpu bundle
contains `libllama.so`, and the vulkan bundle contains `libllama.so` and
`libggml-vulkan.so` — the manifest's file names are correct for arm64.

**Key fact that shapes this plan**: the SDK's `DownloadFor` is triple-aware
("the triple-aware entry point for installing llama.cpp bundles for platforms
other than the active one" — its doc comment in
`kronk@v1.28.0/sdk/tools/libs/libs.go:458-466`), the hashes are of downloaded
file **bytes** (arch-independent), and extraction is pure Go. So the arm64
hashes can be produced on this amd64 dev box by driving the exact same SDK
call `InstallRuntime` uses — no arm64 hardware or CI runner is required. The
amd64 entries double as a built-in cross-check: the tool must reproduce the
already-pinned amd64 hashes exactly, or something is wrong.

**Fallback caveat in yzma**: on a 404, `download.GetWithContext` silently
retries with a *previous* version from `previous.json`
(`yzma@v1.17.1/pkg/download/download.go:376-387`), and the version file is
written with the *requested* version regardless. Step 1's URL existence check
exists precisely so that fallback cannot be what gets hashed.

### `libs.Logger` shape

`libs.Logger` is a function type; the repo's live example is
`internal/web/models_install.go:246`:

```go
	sseLogger := func(_ context.Context, msg string, _ ...any) {
```

so a no-op logger is `func(context.Context, string, ...any) {}`. Do NOT pass
`nil` to `DownloadFor` — the progress callback invokes it during download.

### The AGENTS.md deferral sentence to update

`AGENTS.md:255-257`:

```
  The checksum manifest (`runtime_sums.json`) pins the real b9664 `linux/amd64`
  cpu+vulkan `.so` hashes (verified fail-closed); `linux/arm64` stays placeholder
  (out of v1 scope — those installs download unverified until hashes are added).
```

### What the install surface offers

The Models settings page offers exactly two variants — `cpu` and `vulkan` —
for the host triple only (`internal/feature/settingscards/settingsfocus_models.go:104`
iterates `[]string{"cpu", "vulkan"}`; `InstallRuntime` hardcodes
`runtime.GOOS`/`runtime.GOARCH` as shown above). Because any supported host
could be the live one, the completeness test in Step 5 asserts over the
**whole** embedded manifest, not just the current host's entries.

### Repo conventions that apply here

- Tests: standard `testing` package, table-driven where it helps; no
  assertion frameworks; no `time.Sleep`-based synchronization. Model new
  verify tests after `TestVerifyInstall_Mismatch` (`librt_test.go:71-90`).
- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code. (This plan adds only test code — use `t.Fatalf`/`t.Errorf`.)
- gofmt is law; a PostToolUse hook may auto-format, but run `gofmt -l .` anyway.
- `internal/self/knowledge.md`: **NOT needed** — it never mentions the arm64
  placeholder or the runtime checksum manifest (verified by
  `grep -n "arm64\|runtime_sums" internal/self/knowledge.md` → no relevant hits),
  and the user-visible capability ("owner-initiated, checksum-verified runtime
  install") is unchanged in kind.
- `.tours/`: **no tour fix needed** — no `.tour` file anchors
  `runtime_sums.json`, `librt_test.go`, or `AGENTS.md`
  (`grep -rn '"file": "internal/kronk' .tours/*.tour` anchors only
  `client.go`, `engine.go`, `librt.go`, `officialmodel.go`,
  `modelget/modelget.go` — none modified here).

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted kronk tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/kronk/ -run <Name> -count=1` | pass |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` when running tests. `make test` exports it
automatically but is cached; the `-count=1` uncached form is the gate.)

## Scope

**In scope** (the only files you should modify):

- `internal/kronk/sums_gen_test.go` (create)
- `internal/kronk/runtime_sums.json` (fill the four empty strings)
- `internal/kronk/librt_test.go` (add two tests)
- `AGENTS.md` (the one three-line sentence at lines 255–257)
- `plans/README.md` (status row only)
- `.github/workflows/runtime-hashes.yml` (create ONLY in the Step F fallback)

**Out of scope** (do NOT touch, even though they look related):

- `internal/kronk/librt.go` — the verifier logic (including the empty-hash
  skip) is correct and stays; this plan removes the placeholders, not the skip.
- `internal/kronk/presets.go`, `internal/web/models_install.go`,
  `internal/feature/settingscards/*` — install/UI surfaces are unchanged.
- The `runtimeVersion` constant, `go.mod`/`go.sum` — no version bumps.
- macOS/windows manifest entries — non-Linux triples have no manifest entry
  and therefore already fail closed at `verifyInstall` ("no manifest entry
  for %q"); adding them stays deferred.
- `.github/workflows/ci.yml` — the existing CI pipeline is untouched.

## Git workflow

- Work in an isolated git worktree branched from `origin/main`;
  branch name: `advisor/258-arm64-runtime-checksums`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`);
  commit per logical unit with **explicit pathspecs** — the main checkout is
  shared by parallel agents; stage only your own files. Suggested commits:
  1. `test(kronk): add opt-in TestComputeRuntimeSums manifest audit` —
     `internal/kronk/sums_gen_test.go`
  2. `fix(kronk): pin b9664 linux/arm64 runtime checksums (fail-closed)` —
     `internal/kronk/runtime_sums.json internal/kronk/librt_test.go`
  3. `docs: linux/arm64 runtime hashes pinned — update AGENTS.md deferral note` —
     `AGENTS.md`
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Confirm the four upstream artifacts still exist (no-fallback guard)

yzma silently falls back to a *previous* release on 404 (see "Current state"),
so hashing is only trustworthy when the exact b9664 assets respond. Run:

```sh
for u in \
  "https://github.com/hybridgroup/llama-cpp-builder/releases/download/b9664/llama-b9664-bin-ubuntu-cpu-arm64.tar.gz" \
  "https://github.com/hybridgroup/llama-cpp-builder/releases/download/b9664/llama-b9664-bin-ubuntu-vulkan-arm64.tar.gz" \
  "https://github.com/ggml-org/llama.cpp/releases/download/b9664/llama-b9664-bin-ubuntu-x64.tar.gz" \
  "https://github.com/ggml-org/llama.cpp/releases/download/b9664/llama-b9664-bin-ubuntu-vulkan-x64.tar.gz"; do
  printf '%s -> ' "$u"; curl -sI -o /dev/null -w '%{http_code}\n' "$u"
done
```

**Verify**: every line ends in `302` (or `200`). Any `404` → STOP condition
"artifact missing upstream". Network unreachable / TLS-intercepted → go to
Step F (fallback), then STOP.

### Step 2: Create the opt-in manifest-audit test `internal/kronk/sums_gen_test.go`

Create the file with exactly this content (package `kronk`, so it can use the
unexported `runtimeSums`, `runtimeVersion`, `InstallDirFor`, `sha256File`):

```go
package kronk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardanlabs/kronk/sdk/tools/libs"
)

// TestComputeRuntimeSums downloads every runtime bundle named in the embedded
// checksum manifest (runtime_sums.json) through the exact SDK path
// InstallRuntime uses (libs.New into a fresh staging root + the triple-aware
// DownloadFor — see librt.go:60-66) and logs the sha256 of each
// manifest-named file. Non-empty manifest entries are asserted to match, so a
// green run proves the manifest agrees with the bytes an owner's install
// would fetch today; empty (placeholder) entries only log, which is how new
// hashes are produced when runtimeVersion is bumped.
//
// Opt-in: network + several hundred MB of downloads — deliberately excluded
// from the default suite. Run with:
//
//	BALAUR_COMPUTE_RUNTIME_SUMS=1 TMPDIR=$HOME/.cache/go-tmp \
//	  go test ./internal/kronk/ -run TestComputeRuntimeSums -v -count=1 -timeout 30m
func TestComputeRuntimeSums(t *testing.T) {
	if os.Getenv("BALAUR_COMPUTE_RUNTIME_SUMS") == "" {
		t.Skip("set BALAUR_COMPUTE_RUNTIME_SUMS=1 to download runtime bundles and compute their sha256 sums (network, large)")
	}
	noop := func(context.Context, string, ...any) {}
	for key, fileHashes := range runtimeSums[runtimeVersion] {
		parts := strings.Split(key, "/") // "os/arch/processor" — see verifyInstall
		if len(parts) != 3 {
			t.Fatalf("malformed manifest key %q", key)
		}
		goos, arch, proc := parts[0], parts[1], parts[2]
		staging := t.TempDir()
		lib, err := libs.New(libs.WithLibPath(staging))
		if err != nil {
			t.Fatalf("%s: resolving staging root: %v", key, err)
		}
		if _, err := lib.DownloadFor(t.Context(), noop, arch, goos, proc, runtimeVersion); err != nil {
			t.Fatalf("%s: downloading %s bundle: %v", key, runtimeVersion, err)
		}
		dir := InstallDirFor(staging, arch, goos, proc)
		for filename, want := range fileHashes {
			got, err := sha256File(filepath.Join(dir, filename))
			if err != nil {
				t.Fatalf("%s: hashing %s: %v", key, filename, err)
			}
			t.Logf("%s %s sha256=%s", key, filename, got)
			if want != "" && got != want {
				t.Errorf("%s %s: manifest pins %s but upstream bytes hash to %s", key, filename, want, got)
			}
		}
	}
}
```

**Verify** (compiles, and is a no-op in the default suite):

```sh
go vet ./internal/kronk/ \
&& TMPDIR=$HOME/.cache/go-tmp go test ./internal/kronk/ -run TestComputeRuntimeSums -v -count=1
```

→ vet exits 0; test output shows `--- SKIP: TestComputeRuntimeSums` and `ok`.

### Step 3: Run the audit test to produce the arm64 hashes (amd64 is the cross-check)

```sh
BALAUR_COMPUTE_RUNTIME_SUMS=1 TMPDIR=$HOME/.cache/go-tmp \
  go test ./internal/kronk/ -run TestComputeRuntimeSums -v -count=1 -timeout 30m
```

**Verify**: `PASS`, with six `t.Logf` lines (one per manifest file across the
four triples). The two `linux/amd64/*` entries carry non-empty pins, so a
PASS here **proves the tool reproduces the installer's bytes** — if either
amd64 hash mismatched, the test would fail (that is a STOP condition, not a
reason to re-pin amd64). Record the three logged arm64 values:

- `linux/arm64/cpu` → `libllama.so sha256=<64 hex chars>`
- `linux/arm64/vulkan` → `libllama.so sha256=<64 hex chars>`
- `linux/arm64/vulkan` → `libggml-vulkan.so sha256=<64 hex chars>`

Do NOT assume the arm64 cpu and vulkan `libllama.so` hashes are equal (they
happen to be equal on amd64; the arm64 bundles come from a different builder,
`hybridgroup/llama-cpp-builder`, and may differ).

### Step 4: Fill the four empty strings in `internal/kronk/runtime_sums.json`

Replace each `""` with the corresponding recorded hash, in place — keep the
existing key order, indentation, and line count (18 lines). The file is
`//go:embed`-ed and parsed in `init()` (`librt.go:107-117`), so a JSON typo
panics the whole binary — the re-run below catches that immediately.

**Verify** (manifest now fully asserted against upstream — every `want` is
non-empty, so all six comparisons are enforced):

```sh
grep -c '""' internal/kronk/runtime_sums.json ; \
BALAUR_COMPUTE_RUNTIME_SUMS=1 TMPDIR=$HOME/.cache/go-tmp \
  go test ./internal/kronk/ -run TestComputeRuntimeSums -v -count=1 -timeout 30m
```

→ grep prints `0`; test re-downloads and PASSes with zero `t.Errorf` lines.

### Step 5: Add the completeness + arm64 fail-closed tests to `internal/kronk/librt_test.go`

Append these two tests (imports needed beyond the existing ones: `regexp`):

```go
// TestRuntimeSumsComplete guards against placeholder regressions: every file
// hash in the embedded manifest must be a real sha256. InstallRuntime always
// installs the HOST triple (librt.go), so any manifest entry can be the live
// one on some owner's box — assert over the whole manifest, not just this
// host's triple.
func TestRuntimeSumsComplete(t *testing.T) {
	hex64 := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for version, triples := range runtimeSums {
		for key, files := range triples {
			for filename, hash := range files {
				if !hex64.MatchString(hash) {
					t.Errorf("%s %s %s: hash %q is not a 64-char lowercase sha256 — placeholder entries make installs unverified", version, key, filename, hash)
				}
			}
		}
	}
}

// TestVerifyInstall_ARM64FailClosed proves the pinned arm64 entries reject a
// tampered install, using the REAL embedded manifest (no runtimeSums swap):
// wrong bytes for every manifest-named file must fail verification and delete
// the install dir. Companion to TestVerifyInstall_Mismatch, which covers the
// same mechanics against a fake manifest.
func TestVerifyInstall_ARM64FailClosed(t *testing.T) {
	for _, proc := range []string{"cpu", "vulkan"} {
		t.Run(proc, func(t *testing.T) {
			dir := t.TempDir()
			key := "linux/arm64/" + proc
			files := runtimeSums[runtimeVersion][key]
			if len(files) == 0 {
				t.Fatalf("no embedded manifest entry for %s @ %s", key, runtimeVersion)
			}
			for filename := range files {
				if err := os.WriteFile(filepath.Join(dir, filename), []byte("tampered content"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := verifyInstall(dir, runtimeVersion, "linux", "arm64", proc); err == nil {
				t.Fatal("expected sha256 mismatch error for tampered arm64 install")
			}
			if _, statErr := os.Stat(dir); statErr == nil {
				t.Error("expected install dir to be deleted after mismatch")
			}
		})
	}
}
```

**Verify**:

```sh
TMPDIR=$HOME/.cache/go-tmp go test ./internal/kronk/ -run 'TestRuntimeSumsComplete|TestVerifyInstall' -v -count=1
```

→ all pass, including the two new tests and the four pre-existing
`TestVerifyInstall_*` tests (in particular `TestVerifyInstall_EmptyHash`
still passes — the skip behavior is unchanged; only the shipped manifest no
longer contains placeholders).

### Step 6: Update the AGENTS.md deferral sentence

In `AGENTS.md`, replace lines 255–257 exactly:

```
  The checksum manifest (`runtime_sums.json`) pins the real b9664 `linux/amd64`
  cpu+vulkan `.so` hashes (verified fail-closed); `linux/arm64` stays placeholder
  (out of v1 scope — those installs download unverified until hashes are added).
```

with:

```
  The checksum manifest (`runtime_sums.json`) pins the real b9664 cpu+vulkan
  `.so` hashes for both `linux/amd64` and `linux/arm64` (verified fail-closed);
  non-Linux triples have no manifest entry, so installs there fail closed too.
```

(Same line count; no other AGENTS.md text changes.)

**Verify**: `grep -n "stays placeholder" AGENTS.md` → no output (exit 1).

### Step 7: Full gates, then commit

```sh
gofmt -l . \
&& go vet ./... \
&& go run honnef.co/go/tools/cmd/staticcheck@latest ./... \
&& CGO_ENABLED=0 go build ./... \
&& TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1 \
&& git diff --check
```

**Verify**: gofmt prints nothing; every command exits 0; the full suite is
green (`TestComputeRuntimeSums` shows as skipped — no env var set).

Then commit per the Git workflow section (explicit pathspecs, three commits).

**Verify**: `git status --porcelain` → only in-scope files were touched and
all are committed.

### Step F (FALLBACK ONLY — network to github.com unavailable in your environment)

Only if Step 1 or Step 3 fails because the environment cannot download from
GitHub (e.g. a TLS-intercepting sandbox proxy): the hashes cannot be invented,
so the deliverable becomes the tooling plus a STOP report. Do:

1. Still complete Step 2 (the test file compiles offline — deps are in the
   module cache; verify with `go vet ./internal/kronk/`).
2. Create `.github/workflows/runtime-hashes.yml`:

   ```yaml
   name: runtime-hashes

   # Manually-triggered helper: computes the sha256 sums of the llama.cpp
   # runtime bundles named in internal/kronk/runtime_sums.json, via the exact
   # SDK download path the installer uses. Read the printed hashes from the
   # job log and pin them in runtime_sums.json.
   on:
     workflow_dispatch:

   jobs:
     compute:
       # No arm64 hardware is required — TestComputeRuntimeSums passes explicit
       # triples to the SDK's triple-aware DownloadFor and hashes downloaded
       # bytes. ubuntu-24.04-arm (GitHub's free arm64 runner for public repos)
       # additionally exercises the SDK on a real arm64 host; if that label is
       # unavailable (private repo / label change), ubuntu-latest is equally
       # valid. Manual alternative without Actions: on any box with Go + git,
       # clone the repo and run the command below verbatim.
       runs-on: ubuntu-24.04-arm
       steps:
         - uses: actions/checkout@v4
         - uses: actions/setup-go@v5
           with:
             go-version-file: go.mod
         - name: compute runtime bundle sha256 sums
           env:
             BALAUR_COMPUTE_RUNTIME_SUMS: "1"
           run: go test ./internal/kronk/ -run TestComputeRuntimeSums -v -count=1 -timeout 30m
   ```

3. Commit only `internal/kronk/sums_gen_test.go` and
   `.github/workflows/runtime-hashes.yml`
   (`test(kronk): add opt-in runtime-sums audit + manual hash workflow`),
   run the Step 7 gates, and **STOP**: report that the manifest is still
   placeholder, the owner must trigger the `runtime-hashes` workflow (or run
   the command on any networked box), and Steps 4–6 remain to be executed
   with the printed hashes. Do NOT add the Step 5 tests
   (`TestRuntimeSumsComplete` would fail against placeholders) and do NOT
   touch AGENTS.md in this mode.

## Test plan

- **New tests** (all in `internal/kronk`, std `testing`, no frameworks):
  - `TestComputeRuntimeSums` (`sums_gen_test.go`, opt-in via
    `BALAUR_COMPUTE_RUNTIME_SUMS=1`): downloads every manifest-named bundle
    through the real SDK path and asserts every non-empty manifest hash
    matches upstream bytes; doubles as the hash generator for future
    `runtimeVersion` bumps. Skips (fast, offline) in the default suite.
  - `TestRuntimeSumsComplete` (`librt_test.go`): every hash in the embedded
    manifest is 64-char lowercase hex — the placeholder class of bug can
    never ship silently again.
  - `TestVerifyInstall_ARM64FailClosed` (`librt_test.go`): tampered
    `linux/arm64` cpu and vulkan installs fail verification against the real
    embedded manifest and the install dir is deleted.
- **Structural pattern**: model after `TestVerifyInstall_Mismatch`
  (`internal/kronk/librt_test.go:71-90`).
- **Regression safety**: all four pre-existing `TestVerifyInstall_*` tests
  and `TestInstallRuntime_SeamWorks` must still pass unchanged.
- **Verification**:
  `TMPDIR=$HOME/.cache/go-tmp go test ./internal/kronk/ -count=1 -v` → all
  pass, `TestComputeRuntimeSums` skipped; then the full gate
  `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0.

## Done criteria

Machine-checkable. ALL must hold (fallback Step F instead reports BLOCKED):

- [ ] `grep -c '""' internal/kronk/runtime_sums.json` → `0`
- [ ] `grep -n "stays placeholder" AGENTS.md` → no output (exit 1)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/kronk/ -run 'TestRuntimeSumsComplete|TestVerifyInstall_ARM64FailClosed' -v -count=1` → PASS (both new tests exist and pass)
- [ ] `BALAUR_COMPUTE_RUNTIME_SUMS=1 TMPDIR=$HOME/.cache/go-tmp go test ./internal/kronk/ -run TestComputeRuntimeSums -v -count=1 -timeout 30m` → PASS with all six hashes asserted (run at least once after Step 4)
- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0; `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → exit 0, no output; `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `git status --porcelain` shows no modified files outside the in-scope list
- [ ] `plans/README.md` status row for 258 updated (unless the reviewer maintains the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows changes to `internal/kronk/` or `AGENTS.md` since
  `077318a` and any "Current state" excerpt no longer matches the live code —
  in particular if `runtime_sums.json` already has non-empty arm64 hashes
  (someone landed this independently) or `runtimeVersion` is no longer
  `"b9664"` (every hash in this plan would be for the wrong release).
- Any of the four artifact URLs in Step 1 does not return 302/200: the b9664
  asset is missing upstream. Report WHICH variant — do not hash whatever
  yzma's previous-version fallback serves instead.
- In Step 3, an **amd64** hash mismatches its pinned manifest value. That
  means either the tool does not reproduce the installer's bytes or the
  upstream asset was re-published with different content — both need a human.
  Do NOT "fix" it by re-pinning the amd64 hashes.
- In Step 3, a downloaded arm64 bundle is missing a manifest-named file
  (`sha256File` errors with "no such file") — the manifest's file names would
  be wrong for arm64, which changes the shape of the fix.
- GitHub downloads fail for environmental reasons (no network, TLS-interposed
  proxy): execute Step F, then stop — never hand-write or copy hashes from
  any source other than a run of `TestComputeRuntimeSums`.
- A step's verification fails twice after a reasonable fix attempt, or the
  fix appears to require touching an out-of-scope file (especially
  `internal/kronk/librt.go`).

## Maintenance notes

- **`runtimeVersion` bumps now have a procedure**: change the constant in
  `librt.go`, add the new version's (initially empty) entries to
  `runtime_sums.json`, run `TestComputeRuntimeSums` with the env var to print
  the new hashes, pin them, and re-run — `TestRuntimeSumsComplete` fails the
  suite until every entry is filled. Mention this in the bump's review.
- **Trust model is TOFU**: the pins fix "first verified download" bytes from
  GitHub release assets (arm64 from `hybridgroup/llama-cpp-builder`, amd64
  from `ggml-org/llama.cpp`); there is no upstream signature to chain to. The
  value is tamper-evidence from pin-time onward, identical to the existing
  amd64 posture. A reviewer should independently re-run
  `TestComputeRuntimeSums` once to confirm the pinned arm64 values.
- **What a reviewer should scrutinize**: that the four JSON values are
  exactly the test's logged output (no transcription slips), and that no
  change leaked into `librt.go`.
- **Deferred, deliberately**: darwin/windows manifest entries (those triples
  have no entry, so `verifyInstall` fails closed with "no manifest entry" —
  out of v1 scope, per AGENTS.md); `linux/arm64/cuda` (supported by the SDK
  but not offered by Balaur's install surface, which only offers cpu+vulkan);
  and the Step F workflow file — if Step F was not needed, do not create it
  (a committed opt-in test covers regeneration; YAGNI).
- If a future change makes the install surface offer non-host triples, extend
  `TestVerifyInstall_ARM64FailClosed`'s pattern to the new triples and keep
  `TestRuntimeSumsComplete` manifest-wide.
