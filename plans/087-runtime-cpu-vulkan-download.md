# Plan 087: Owner-initiated llama.cpp runtime install — both CPU and Vulkan, into the lib dir the engine loads from

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 087 row in plans/readme.md (add it if absent, matching the existing column format).
>
> **Drift check (run first)**: `git rev-parse HEAD` should be at or after `e4c6394` (plan 086 merged). Confirm the vendored SDK is still `github.com/ardanlabs/kronk v1.28.0` and `github.com/hybridgroup/yzma v1.17.1` in `go.mod` — the `libs`/`download` API anchors below are pinned to those versions. Read the "Current state" files live before editing; the line numbers below predate 086 and have shifted.
>
> ## ⚠️ Reconciliation — PLAN 086 HAS LANDED (re-anchored 1f463bb → e4c6394)
> Plan 086 (one-click official-model download) is **merged to main** and several primitives this plan said it would "reuse from 086" now **exist** — use them, don't recreate:
> - **`kronk.RuntimeInstalled()` already exists** (`internal/kronk/runtime.go`) and currently calls `resolveLibDir(LibPath(), Processor())` then stats `libllama.so`. **Step 1's `LibRoot()` change MUST also update `RuntimeInstalled` to use `LibRoot()`** (it's a third caller alongside the engine and the new installer — all three must agree).
> - **`kronk.ModelsDir()` exists** in `presets.go` (the env-getter shape to mirror for `LibRoot()`).
> - **The SSE-progress + single-in-flight + audit pattern exists** in `internal/web/models.go`: `downloadOfficialModel` uses `const downloadStoreKey = "modeldownload.cancel"`, guards via `h.app.Store().GetOk(...)`, removes via `h.app.Store().Remove(...)` in a `defer` (NOT `Set(...,nil)` — GetOk is presence-based; this was a bug fixed in 086), streams `datastar.NewSSE`, morphs a card by id, and audits `store.Audit(h.app,"owner","llm.model.download",...)`. **Mirror this exactly** for `installRuntime` (use a distinct store key e.g. `"runtimedownload.cancel"` and audit action `llm.runtime.install`).
> - **`modelcards` already has** `StatusDownloading`, `Progress`/`ProgressLabel`, a progress meter, a Cancel form, and an official-model CTA (`ShowOfficialCTA`/`OfficialCTAName`/`OfficialCTAMeta`/`RuntimeMissing` on `PanelView`); the runtime section adds alongside these.
> - **`settingscards.BuildModelsPanelView`** already renders the `RuntimeMissing` Alert (086 Step 8) when `kronk.RuntimeInstalled()` is false — this plan's Step 6 turns that Alert into the actual cpu/vulkan **Install** actions.
> - Routes `/ui/model/download` + `/ui/model/download/cancel` are mounted near `/ui/model/install` in `web.go` — add `/ui/runtime/install` beside them.
> - The official-model download proves the end-to-end UX; this plan makes the model actually *runnable* by installing the native runtime it needs.

## Status
- **Priority**: P1
- **Effort**: L
- **Risk**: MED–HIGH
- **Depends on**: plan **086** (reuses its SSE-progress loop, single-in-flight `app.Store()` flag, audit pattern, and the `RuntimeInstalled()` helper; and it *resolves* the runtime-missing Alert that 086 stubs). Land 086 first.
- **Category**: feature (owner-requested)
- **Planned at**: commit `1f463bb`, re-anchored to `e4c6394` (post-086-merge), 2026-06-17

## Why this matters
The owner asked to *"embed both cpu and vulkan runtimes"* so the official model can be used **right away** on a fresh box. The honest engineering reading of "embed both" here is **"make both runtimes first-class and installable without leaving the app"**, *not* "compile the native libraries into the Go binary." Two hard facts drive that:

1. **Compiling them in is the wrong trade.** `libggml-vulkan.so` on this box is **50 MB** (it embeds SPIR-V compute shaders); `go:embed` is compile-time, so embedding cpu+vulkan would push the binary from ~111 MB to **~168 MB for every user** — including the GPU-less majority — with no way to opt out, and it would force a per-OS/arch build matrix because one Linux binary can't carry every platform's `.so`. That breaks the "one lean CGO-free binary" posture (AGENTS.md, plan 074's accepted-bloat ledger) for negative value.
2. **The capability is already vendored.** The Kronk SDK (`github.com/ardanlabs/kronk@v1.28.0/sdk/tools/libs`) ships a complete library installer — `Libs.DownloadFor(arch, os, processor, version)` fetches a precompiled llama.cpp bundle and writes it to **exactly** `<root>/<os>/<arch>/<processor>/` with a `version.json`, which is **exactly** what Balaur's `resolveLibDir` already loads from (`internal/kronk/engine.go:80-100`). "Both runtimes" = call it twice (cpu + vulkan) into the same root. The Go binary is untouched; CGO stays off; the fetch is **owner-initiated and audited**, so the "engine never downloads on **boot**" rule holds.

So this plan keeps the binary flat and adds a Models-panel control that installs the cpu and/or vulkan runtime for the host into the lib root, verifies it, and lets `BALAUR_PROCESSOR=vulkan` light up the owner's AMD 780M with no manual lib staging.

## The decisive correctness pin (read before coding)
**The download target MUST be the exact directory the engine loads from.** There is a real trap here:

- `internal/kronk/presets.go:10` — `LibPath()` returns `os.Getenv("BALAUR_LIB_PATH")`, which is **empty when unset**. The README's "default `~/.local/share/balaur/kronk/lib`" is **not in the code**.
- `internal/kronk/engine.go:83-100` — `resolveLibDir(root, processor)` calls `libs.New(libs.WithProcessor(p))` (with `WithLibPath(root)` **only when `root != ""`**). When `BALAUR_LIB_PATH` is unset, the SDK's own default root is used: `defaults.BaseDir("")` → `$HOME/.kronk` → libraries root `$HOME/.kronk/libraries`, per-triple `$HOME/.kronk/libraries/<os>/<arch>/<processor>/`.

If the install writes to one root and the engine loads from another, the owner downloads a runtime and the engine still reports "not installed." **Fix: introduce one shared resolver** (`kronk.LibRoot()`) used by *both* the installer and `resolveLibDir`, and pass that same root into `libs.New(WithLibPath(root))` on both sides. Pick a Balaur-owned default (recommend `~/.local/share/balaur/kronk/lib`, matching the README so the docs stop lying) so the install dir is predictable and not buried under `$HOME/.kronk`.

## Background facts (confirmed against the vendored SDK @ kronk v1.28.0)
- **`Libs.DownloadFor(ctx, log, arch, os, processor, version)`** (`sdk/tools/libs/libs.go:466-489`) installs a specific triple into `installPathFor(root, ...)` = `<root>/<os>/<arch>/<processor>/`, downloads via yzma into a `temp/` subdir, swaps it in atomically (`swapTempForLibAt`), and writes `version.json` (`writeVersionFile`). `version == ""` ⇒ the SDK's pinned `defaultVersion = "b9664"` (`libs.go:29`). It errors if the Libs is **read-only** (a user-supplied non-empty dir without `version.json`, `libs.go:746-770`) — so construct the Libs against the **root**, not a pre-populated user dir.
- **`Libs.Download(ctx, log)`** (`libs.go:281-347`) installs/updates only the **active** triple (host arch/os + the Libs' processor), with a network-availability check and the version matrix. Good for "install the runtime I'm configured to use"; `DownloadFor` is what installs the **other** variant too.
- **`SupportedCombinations()` / `IsSupported(arch,os,proc)`** (`sdk/tools/libs/combinations.go:54-75`) enumerate buildable triples. Confirmed present for v1: `{amd64, linux, cpu}` (`combinations.go:21`) and `{amd64, linux, vulkan}` (`:25`). Use `IsSupported` to validate before any install and to scope the UI to what this plan ships.
- **No checksum.** The SDK/yzma download path does **not** verify a hash — `downloadInto` (`libs.go:427-456`) extracts the tarball and writes `version.json`, nothing more. yzma fetches from `github.com/ggml-org/llama.cpp/releases` (and `hybridgroup/llama-cpp-builder` for some triples) over go-getter (`yzma/pkg/download/download.go:142-422`). **This plan must add Balaur-owned post-extract sha256 verification** (Step 4) — that is the main net-new work and the main risk.
- **`download.LibraryName(os)`** → `libllama.so` (linux) / `llama.dll` / `libllama.dylib` (`yzma/pkg/download/download.go:533-550`). Presence of this file is the "installed" signal `RuntimeInstalled()` (plan 086 Step 3) already checks.
- **Vulkan still needs a host loader.** The downloaded vulkan bundle provides `libggml-vulkan.so` but dlopen's the host `libvulkan.so.1` + the GPU ICD (mesa/RADV) at runtime. No download can ship those — they are host setup. The owner's box already has working RADV (verified: `vulkaninfo` reports "AMD Radeon 780M Graphics (RADV PHOENIX)"), so once the vulkan bundle is installed, `BALAUR_PROCESSOR=vulkan` should work. The UI must say this plainly and **fall back to CPU, never panic**, when the loader/ICD is absent (mirror the missing-lib error posture in `engine.go:73-75`).

## Current state (confirmed excerpts)

**`internal/kronk/engine.go:80-100`** — the resolver the install must agree with:
```go
80 // resolveLibDir returns the directory yzma should dlopen the llama.cpp library
81 // from for the given processor. A root containing version.json is honored as-is;
82 // otherwise the per-triple variant dir <root>/<os>/<arch>/<processor>/ is used.
83 func resolveLibDir(root, processor string) (string, error) {
...
87 	p, err := download.ParseProcessor(processor)
...
91 	opts := []libs.Option{libs.WithProcessor(p)}
92 	if root != "" {
93 		opts = append(opts, libs.WithLibPath(root))
94 	}
95 	lib, err := libs.New(opts...)
...
99 	return lib.LibsPath(), nil
100 }
```

**`internal/kronk/presets.go:5-22`** — `LibPath()` (env, empty when unset) + `Processor()`:
```go
10 func LibPath() string { return os.Getenv("BALAUR_LIB_PATH") }
17 func Processor() string {
18 	if p := os.Getenv("BALAUR_PROCESSOR"); p != "" { return p }
21 	return "cpu"
22 }
```

**Vendored** `…/kronk@v1.28.0/sdk/tools/libs/libs.go:466-489` (`DownloadFor`) and `…/combinations.go:19-26` (the supported linux triples) — the install primitive + the matrix. Do not copy these into the repo; call them.

## Decision: progress granularity (recommended vs simpler)
`DownloadFor` does not expose a byte-level progress callback to the caller — internally it builds its own progress reader that **logs** lines like `"download-libraries: Downloading … 12 MB of 84 MB (5.20 MB/s)"` through the `applog.Logger` you pass (`libs.go:434-440`).
- **RECOMMENDED (simple, KISS)**: pass a Balaur `Logger` that forwards those progress lines to the SSE meter as a coarse status (variant name + "12 / 84 MB"). Runtime bundles are tens-of-MB (cpu) to ~100 MB (vulkan) — far smaller than the multi-GB model — so coarse progress is fine.
- **Alternative (finer, more code)**: bypass the SDK and call `download.GetWithProgress(arch, os, processor, version, destTemp, tracker)` with your own `getter.ProgressTracker`, then write `version.json` yourself in the SDK's format and atomic-swap. Only do this if the owner wants a precise determinate bar; it duplicates the SDK's temp-swap/version-file logic. Default to RECOMMENDED.

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Drift check | see header | excerpts match; SDK pins intact |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test | `go test ./...` | all pass |
| Runtime-install tests | `go test ./internal/kronk/...` | ok |
| Route registered | `grep -n '/ui/runtime/install' internal/web/web.go` | present |
| Resolver agreement | a unit test asserting `LibRoot()`-based install dir == `resolveLibDir(LibPath(),proc)` parent triple | pass |
| Whitespace | `git diff --check` | no output |

Sandbox note: a TLS-intercepting Hyperagent sandbox needs the GOPROXY shim (`docs/hyperagent-sandbox.md`). **Also**: the runtime install makes a real outbound fetch to GitHub releases — in a network-restricted sandbox the *live* install will fail; the unit tests must not depend on real network (inject the installer seam, Step 5).

## Scope
**In scope**:
- `internal/kronk/presets.go` (or a new `internal/kronk/librt.go`) — `LibRoot()` shared resolver + a thin `InstallRuntime(ctx, processor, logger) error` wrapper over `libs.DownloadFor`, scoped to host arch/os, validated by `IsSupported`.
- `internal/kronk/engine.go` — make `resolveLibDir` consume `LibRoot()` so install and load agree (the **one** behavioral change to the load path; keep it minimal and covered by the agreement test).
- **NEW** a small Balaur-owned checksum manifest: `internal/kronk/runtime_sums.json` (or a `.go` map) embedded via `go:embed`, keyed by `(version, os, arch, processor) → {filename: sha256}`, plus a `verifyInstall(dir, triple)` that checks the key `.so` files post-extract.
- `internal/feature/modelcards` + `internal/feature/settingscards` — a "Local AI runtime" section: per-variant (cpu / vulkan) status (installed / available / unsupported), an **Install** action, and host-loader messaging for vulkan.
- `internal/web/models.go` — `installRuntime(e)` SSE handler (reusing 086's progress/in-flight/audit primitives; audit action `llm.runtime.install`).
- `internal/web/web.go` — `POST /ui/runtime/install` (form field `processor=cpu|vulkan`).
- `internal/feature/storybook/stories_settings.go` — runtime-section stories (installed / installing / vulkan-needs-host-loader / unsupported).
- Docs same commit: `README.md` (env table note + that the runtime is installable in-app), `internal/self/knowledge.md` (the `:18` "never downloads" nuance), AGENTS.md (the deferred-runtime-download line now ships), and an MPL-2.0/MIT redistribution note (see Licensing).

**Out of scope** (do NOT touch / build):
- macOS (`darwin/metal`, `darwin/cpu`) and Windows runtimes — each is a separate test + checksum-manifest obligation. Scope v1 to **linux/amd64 {cpu, vulkan}** (the owner's box); document the rest as "supply the lib manually via `BALAUR_LIB_PATH`."
- CUDA / ROCm variants — out of scope (no NVIDIA/ROCm target this cycle).
- Auto-detecting/auto-installing on boot — forbidden by the "no boot download" rule. Install is always an explicit owner click.
- Re-hosting/mirroring the tarballs on a Balaur server — keep fetching from the upstream releases the SDK already targets (see Licensing).
- Uninstall/version-management UI for runtimes (`libs.Remove`/`List` exist) — deferred.

## Git workflow
- Branch `improve/087-runtime-cpu-vulkan-download` off `main` (or stacked on 086's branch if not yet merged).
- Conventional commits, e.g.:
  - `feat(kronk): single LibRoot() resolver shared by install + load`
  - `feat(kronk): owner-initiated cpu/vulkan runtime install via SDK libs`
  - `feat(kronk): post-extract sha256 verification of installed runtimes`
  - `feat(web): Local AI runtime section — install cpu/vulkan with progress`
  - `docs: runtime is installable in-app; MPL/MIT redistribution note`
- Do NOT push or open a PR unless explicitly told.

## Steps

### Step 1: One shared lib-root resolver (`LibRoot()`)
Add to `internal/kronk/presets.go` (or `librt.go`):
```go
// LibRoot returns the llama.cpp libraries ROOT that holds the per-triple variant
// dirs (<root>/<os>/<arch>/<processor>/). BALAUR_LIB_PATH wins; empty defaults to
// the Balaur-owned XDG dir ~/.local/share/balaur/kronk/lib. BOTH the runtime
// installer and resolveLibDir use this so the install target and the load source
// are always the same directory.
func LibRoot() string {
	if p := LibPath(); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "lib"
	}
	return filepath.Join(home, ".local", "share", "balaur", "kronk", "lib")
}
```
Then change `internal/kronk/engine.go:resolveLibDir` so it is **always** called with `LibRoot()` as the root (e.g. callers pass `LibRoot()` instead of `LibPath()`, and `resolveLibDir` always `append`s `libs.WithLibPath(root)` since root is now never empty). This is the single behavioral change to the load path; keep the diff tiny. **Verify**: `go build ./internal/kronk/...`; add the agreement unit test now (Step 8) and run it.

### Step 2: `InstallRuntime` wrapper over the SDK
Add (e.g. `librt.go`):
```go
// InstallRuntime downloads the precompiled llama.cpp bundle for the host
// (runtime.GOOS/GOARCH) and the given processor ("cpu"|"vulkan") into LibRoot(),
// then verifies it (Step 4). It is owner-initiated only — never called on boot.
func InstallRuntime(ctx context.Context, processor string, log libs.Logger) error {
	goos, err := defaults.OS("")     // runtime.GOOS, validated
	goarch, err2 := defaults.Arch("") // runtime.GOARCH, validated
	// ... handle err/err2 ...
	if !libs.IsSupported(goarch.String(), goos.String(), processor) {
		return fmt.Errorf("runtime %s/%s/%s is not a supported build", goarch, goos, processor)
	}
	lib, err := libs.New(libs.WithLibPath(LibRoot()))
	if err != nil { return fmt.Errorf("resolving lib root: %w", err) }
	if _, err := lib.DownloadFor(ctx, log, goarch.String(), goos.String(), processor, ""); err != nil {
		return fmt.Errorf("installing %s runtime: %w", processor, err)
	}
	return verifyInstall(installDirFor(LibRoot(), goarch.String(), goos.String(), processor), version, goos.String(), goarch.String(), processor)
}
```
(Read the actual version that landed from the written `version.json` — `libs.ReadVersionFile(dir)` — and pass it to `verifyInstall`; don't hardcode `b9664` and let it drift.) Construct the Libs against `LibRoot()` (a root, not a populated read-only dir) so `DownloadFor` is not refused as read-only. **Verify**: `go build ./internal/kronk/...`.

### Step 3: "Both runtimes" = install cpu and vulkan
`InstallRuntime` installs **one** variant; "embed both" means the owner can install both into the same root so switching `BALAUR_PROCESSOR=vulkan` needs no second trip. The UI (Step 6) offers cpu and vulkan as separate Install actions; installing both writes `<root>/linux/amd64/cpu/` and `<root>/linux/amd64/vulkan/`, each with its own `version.json`. No code beyond calling `InstallRuntime` twice (sequentially — the single-in-flight flag from 086 prevents concurrent installs clobbering the shared `temp/`).

### Step 4: Post-extract checksum verification (the net-new safety work)
The SDK does **not** verify downloads, so add Balaur-owned verification:
- Create `internal/kronk/runtime_sums.json` embedded with `go:embed`, shaped:
  ```json
  { "b9664": { "linux/amd64/cpu":    { "libllama.so": "<sha256>", "libggml-base.so": "<sha256>", ... },
               "linux/amd64/vulkan": { "libllama.so": "<sha256>", "libggml-vulkan.so": "<sha256>", ... } } }
  ```
- `verifyInstall(dir, version, os, arch, proc)`: look up the `(version, os/arch/proc)` entry; for each listed file, sha256 the file in `dir` and compare. **Missing manifest entry for the installed version ⇒ STOP/return an error**, not a silent pass (fail-closed). On mismatch: delete the variant dir (so a corrupt install can't load) and return an error the handler surfaces.
- **Decision (confirm before merge)**: the executor installs cpu + vulkan once on a clean box, captures the real sha256s of at least `libllama.so` + the per-variant `libggml-*.so`, and pastes them into the manifest for `b9664` (the SDK's current `defaultVersion`). Pin to that version explicitly via `libs.WithVersion("b9664")` in `InstallRuntime` so the manifest and the download can't drift apart; bumping the runtime version is then a deliberate manifest+pin change (mirrors plan 086's model-hash discipline).

> Lighter alternative if the owner declines manifest maintenance: verify only that `libllama.so` exists and the engine can `kronk.Init` against the dir (a load smoke-test) rather than full hashes — weaker (no integrity guarantee), so the manifest is recommended. Record whichever is chosen.

### Step 5: Web handler (`internal/web/models.go`)
Add `installRuntime(e *core.RequestEvent) error`, an SSE handler reusing **plan 086's** primitives:
- `processor := e.Request.FormValue("processor")` (`cpu`|`vulkan`); reject anything else.
- Open `datastar.NewSSE`, set the single-in-flight flag on `h.app.Store()`, render the runtime section in an "installing <variant>" state, patch `#models-panel`.
- Call `kronk.InstallRuntime(ctx, processor, sseLogger)` where `sseLogger` forwards SDK progress log lines to a `#runtime-dl-progress` morph (recommended granularity).
- On success: re-render via `BuildModelsPanelView` (now `RuntimeInstalled()` is true) and patch `#models-panel`; audit `llm.runtime.install` (`target=processor`, `detail={version}`, `allowed=true`). On failure (network, checksum, unsupported): audit `allowed=false` with the reason and render a `ui.Alert` — never panic, app stays up.
- Clear the in-flight flag in a `defer`.
**Verify**: `go build ./...`; `go vet ./...`.

### Step 6: Runtime section in the panel (view + builder)
In `internal/feature/modelcards` add a `RuntimeView` (per-variant: `Processor`, `Status` ∈ installed|available|installing|unsupported, `Version`, `NeedsHostLoader bool`) and render a **"Local AI runtime"** section above the models grid in `panel.go`: cpu and vulkan rows, each with an Install action (`@post('/ui/runtime/install')` with a hidden `processor` field, mirroring `actionForm` at `modelcard.go:75-82`), the installed version when present, and for vulkan a one-line note — *"Uses your GPU via Vulkan. Needs the host Vulkan loader + driver (e.g. mesa); falls back to CPU if absent."* Populate it in `settingscards.BuildModelsPanelView` from `kronk.LibRoot()` + `libs.ReadVersionFile` per variant + `libs.IsSupported(host…, proc)` (mark unsupported variants disabled). Keep `BuildModelsPanelView` the single source of truth. **Verify**: `go build ./...`; `go test ./internal/feature/storybook/...`.

### Step 7: Routes
In `internal/web/web.go` beside the model routes:
```go
	se.Router.POST("/ui/runtime/install", h.installRuntime)
```
**Verify**: `grep -n '/ui/runtime/install' internal/web/web.go` → present; `go build ./...`.

### Step 8: The resolver-agreement test (the correctness guard)
Add a `kronk` package test asserting the install dir and the load dir are identical:
- `installDirFor(LibRoot(), "amd64", "linux", "vulkan")` (the dir `DownloadFor` writes) **==** `resolveLibDir(LibRoot(), "vulkan")` (the dir the engine loads). Run with `BALAUR_LIB_PATH` set (to a `t.TempDir()`) and unset (default `LibRoot()`), via an injected `HOME`/env seam — never mutate the real home. This is the regression net for the "downloads to one place, loads from another" trap. **Verify**: `go test ./internal/kronk/...` → the agreement test passes both ways.

### Step 9: Storybook + docs
- Stories in `internal/feature/storybook/stories_settings.go`: runtime section in installed / installing / vulkan-needs-host-loader / unsupported states.
- `README.md`: note the runtime is installable from the Models page; clarify the `BALAUR_LIB_PATH` default now actually resolves to `~/.local/share/balaur/kronk/lib` (Step 1) — fix the doc/code mismatch.
- `internal/self/knowledge.md`: the inference section ("never downloads on boot" stays true; the owner can now install the runtime in-app).
- **Licensing note** (AGENTS.md safety / plan 074 §licensing style): the runtime tarballs are llama.cpp + ggml, **MIT** — permissive, but redistribution requires shipping their copyright/license notice. Since this plan fetches from the **upstream** `ggml-org/llama.cpp` (and `hybridgroup/llama-cpp-builder`) releases the SDK already targets and does **not** re-host them, Balaur is not a redistributor — record this explicitly so a future "mirror them on our server" decision knows it takes on the MIT attribution duty. go-getter stays indirect (MPL-2.0, already accepted in plan 074).
**Verify**: `grep -n 'runtime' README.md` shows the new note.

### Step 10: Full verification + index
Run the Done-criteria gate. Update the 087 row in `plans/readme.md`.

## Test plan
- **`internal/kronk`**: (1) the resolver-agreement test (Step 8) — the headline guard; (2) `verifyInstall` table tests (matching manifest → ok; mismatch → variant dir deleted + error; missing manifest entry → error) using `t.TempDir()` and fixture `.so` files with known hashes; (3) `IsSupported` gating (vulkan supported on linux/amd64; an unsupported triple → error from `InstallRuntime` **without** a network call). **Inject the SDK install** behind a seam (a func var) so these never hit the real network.
- **Web handler**: a `handlers_test.go` case faking the installer seam asserting success flips the runtime row to "installed" and re-renders, and that an install error renders an Alert (no panic).
- **Storybook**: `go test ./internal/feature/storybook/...`.
- **Manual (network-permitting, on the owner's box)**: from the Models panel, Install cpu → confirm `~/.local/share/balaur/kronk/lib/linux/amd64/cpu/libllama.so` appears with a `version.json`; Install vulkan → `…/vulkan/libggml-vulkan.so` appears; set `BALAUR_PROCESSOR=vulkan`, run `go run . serve`, send a chat turn after a model is present (plan 086) → confirm it loads on the GPU (or falls back to CPU with a clear message if the host Vulkan loader is missing). Inspect `audit_log` for `llm.runtime.install` rows.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0.
- [ ] `go test ./...` → all pass, incl. the **resolver-agreement test** and the `verifyInstall` tests.
- [ ] `grep -n '/ui/runtime/install' internal/web/web.go` → present.
- [ ] One `LibRoot()` resolver feeds **both** the installer and `resolveLibDir` (the agreement test proves the dirs match, with `BALAUR_LIB_PATH` set and unset).
- [ ] Installing cpu and vulkan writes `<LibRoot>/linux/amd64/{cpu,vulkan}/` each with `libllama.so` + `version.json`; `kronk.RuntimeInstalled()` then returns true.
- [ ] Every installed runtime is sha256-verified against the embedded manifest before it is allowed to load; a mismatch or missing-manifest-entry fails closed (variant dir removed, error surfaced, audited `allowed=false`).
- [ ] The vulkan row states the host-loader requirement; a missing loader/ICD falls back to CPU and never panics.
- [ ] `darwin`/`windows`/`cuda`/`rocm` are **not** offered (shown unsupported or omitted); README documents the manual `BALAUR_LIB_PATH` path for them.
- [ ] `git diff --check` → no output; docs (README runtime note + `BALAUR_LIB_PATH` default fix, knowledge.md, AGENTS.md, licensing note) updated same commit.
- [ ] `plans/readme.md` 087 row updated.

## STOP conditions
- **Drift**: a cited file or the SDK version pin changed since `1f463bb` and an excerpt/anchor no longer matches — STOP and report (the `libs`/`download` API is version-specific).
- **Install dir ≠ load dir**: the resolver-agreement test fails — STOP; this is the core correctness invariant. Do not ship a runtime installer that writes where the engine won't look.
- **No real checksums**: the manifest still has placeholders for the version being installed — STOP; capture real sha256s (Step 4 decision) or the verify gate is theater.
- **Tempted to embed the `.so` in the binary** — STOP; that is the rejected approach (50 MB×variants of unavoidable bloat). This plan delivers "both runtimes" via install-on-demand.
- **Tempted to add a boot-time / `OnServe` auto-install** — STOP; installs are owner-click only.
- **macOS/Windows/CUDA/ROCm creeping into scope** — STOP; v1 is linux/amd64 cpu+vulkan.
- **A Verify fails twice** after a fix attempt — STOP and report the command + output.

## Maintenance notes
- **Version pin discipline**: the runtime version is pinned (`b9664`) in *both* the download call and the checksum manifest. Bumping it is a deliberate two-place change (re-capture hashes). Never let `DownloadFor`'s implicit "latest" path run unpinned — it would fetch a build the manifest can't verify.
- **The `resolveLibDir` change is the one load-path edit** — a reviewer should confirm it only swaps the root source (`LibPath()` → `LibRoot()`) and doesn't alter the `libs.New` resolution semantics; the agreement test is the proof.
- **Deferred (record, don't build)**: macOS-metal + Windows runtimes (each = a build target + manifest entries + a test), runtime uninstall/version UI (`libs.Remove`/`List`), CUDA/ROCm, and a self-hosted tarball mirror (which would add the MIT attribution duty). 
- **Vulkan host dependency is permanent** — no install can ship `libvulkan.so.1`/the ICD; the UI messaging is load-bearing, keep it honest. Pairs with plan 086's runtime-missing Alert, which this plan upgrades into the Install action.
- Together with **086**, this completes the owner's ask: a fresh linux/amd64 box goes from nothing → one click installs the runtime → one click downloads the official model → chat works, CPU by default or Vulkan on the 780M, with the binary still flat and CGO off.
