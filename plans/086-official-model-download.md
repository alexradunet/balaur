# Plan 086: One-click "Get our official model" — in-app GGUF download + verify + auto-activate

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 086 row in plans/readme.md (add it if absent, matching the existing column format).
>
> **Drift check (run first)**:
> `git diff --stat 3136bad..HEAD -- internal/web/models.go internal/web/web.go internal/store/llm_settings.go internal/kronk/presets.go internal/feature/modelcards internal/feature/settingscards internal/feature/storybook`
> If any of these changed since this plan was written, compare the "Current state" excerpts below to the live code; on mismatch, STOP and report.
>
> **Reconciliation note (re-anchored 1f463bb → 3136bad, 2026-06-17)**: commit `3136bad` ("home gutter, quest-log legibility, Heads+palette into Settings") reshaped `settingscards/settingsfocus.go` but left `BuildModelsPanelView` (`:118-143`) and `knowledge.md:18` byte-identical, so every excerpt below still holds. One useful addition landed: `settingscards.ExamplePanelView()` at `settingsfocus.go:175` returns a populated `modelcards.PanelView` for storybook/tests — Step 10 may extend it (or a sibling) for the new download states rather than hand-building fixtures.

## Status
- **Priority**: P1
- **Effort**: M–L
- **Risk**: MED
- **Depends on**: none. (Soft pairing: plan **087** resolves the "runtime not installed" sub-state this plan stubs with an Alert — see Step 8. 086 ships standalone and lower-risk; 087 upgrades the precheck into a one-click runtime install. On a box that has **no** llama.cpp runtime yet — e.g. the owner's current box — a model downloaded by this plan will not actually run until 087 lands or `BALAUR_LIB_PATH` points at a real runtime. This plan is honest about that; it does not claim chat works without a runtime.)
- **Category**: feature (owner-requested)
- **Planned at**: commit `1f463bb`, re-anchored to `3136bad`, 2026-06-17

## Why this matters
Today the only way to get a model onto a Balaur box is to **find a GGUF on Hugging Face yourself, download it by hand, and paste an absolute path** into the Models panel (`internal/feature/modelcards/panel.go:52-71` — the `installForm` with the `path`/`embed_path` fields). The owner asked for the opposite: *"a simple way to download our official model and use it right away."* This was actually shipped once before — plan **023** ("GGUF download manager: background, progress, cancel, delete") — but plan **074** removed the whole Ollama/remote path and collapsed model install back to "paste a path," so the download affordance is gone in the Kronk world. This plan re-introduces it, Kronk-native: a single **"Get our official model"** button that streams a pinned GGUF down with a live progress bar, verifies its sha256, and activates it through the **exact** atomic path the manual install already uses (`store.SaveLocalModel` → `store.SetActiveLLMModel`), so the card flips to **"In use"** with zero path typing.

This is deliberately the **owner-initiated** download AGENTS.md said was "deferred work" — it is *not* a boot-time download (the engine still never fetches anything on `OnServe`), so it stays inside the local-first rule: the owner clicks, the fetch is audited, nothing happens unprompted.

## Background facts (confirmed against the live tree + the vendored SDK)
- **The install/activate path is two store calls.** `internal/web/models.go:162-183` (`installModel`) validates an absolute `.gguf` path, `os.Stat`s it, then calls `store.SaveLocalModel(app, path, embed)` and `store.SetActiveLLMModel(app, id, "owner")`, audits `llm.model.install`, and re-renders via `h.modelsPanel(e, "")`. Reuse these verbatim — do **not** invent a parallel registration path.
- **`SaveLocalModel` is idempotent.** `internal/store/llm_settings.go:98-113` → `findOrCreateLLMModel` (`:191-223`) de-dupes by `(provider, chat_model)` and skips the write when unchanged (plan 067), so re-installing the same file is a safe no-op upsert.
- **A model shows "In use" purely because the file exists.** `internal/turn/models.go` `isLocalFile` (`:16-22`) gates on `filepath.IsAbs && .gguf`; `availableChoices` (`:107-142`) `os.Stat`s the file → present = `StatusAvailable`, active = `StatusActive`. So once the `.gguf` exists at an absolute path and is set active, `BuildModelsPanelView` yields `StatusActive` and the card renders **"In use"** (`internal/feature/modelcards/modelcard.go:59-70`) with no extra work.
- **The panel is one SSE patch target.** Every model action ends in `sse.PatchElements(..., datastar.WithSelectorID("models-panel"), datastar.WithModeOuter())` (`internal/web/models.go:145-157`). A long-lived download handler can morph a smaller fragment by id mid-stream, then patch the whole panel on completion.
- **There is a streaming-SSE exemplar.** `internal/web/chatstream.go` opens `datastar.NewSSE(e.Response, e.Request)` and morphs a single element by id repeatedly during a long operation — copy that shape for the progress meter.
- **go-getter is already in the graph** (`go.mod:` `github.com/hashicorp/go-getter v1.8.6 // indirect`, pulled transitively via kronk/yzma), and the vendored yzma package **already exposes a model downloader**: `github.com/hybridgroup/yzma/pkg/download.GetModelWithProgress(url, dest, progress)` (uses go-getter under the hood). See "Decision: downloader" below — we recommend a small hand-rolled net/http downloader over this, for resume + streaming-checksum + cancel control, but the SDK function is the zero-new-code fallback.
- **The XDG default in the README is aspirational.** `internal/kronk/presets.go:10` `LibPath()` returns `os.Getenv("BALAUR_LIB_PATH")` — **empty when unset**; the README's `~/.local/share/balaur/kronk/lib` "default" is not in the code. Do not assume model/lib dirs share a parent that is actually defined. This plan defines its own concrete `ModelsDir()` default (Step 2).

## Current state (confirmed excerpts)

**`internal/web/models.go:159-183`** — the path this plan extends, NOT replaces:
```go
159 // installModel registers a local GGUF model by absolute path and makes it active.
160 // The file must already be on this box (owner-initiated downloads are a later
161 // slice). It patches #models-panel.
162 func (h *handlers) installModel(e *core.RequestEvent) error {
163 	path := strings.TrimSpace(e.Request.FormValue("path"))
...
174 	id, err := store.SaveLocalModel(h.app, path, embed)
...
178 	if err := store.SetActiveLLMModel(h.app, id, "owner"); err != nil {
...
181 	store.Audit(h.app, "owner", "llm.model.install", path, true, nil)
182 	return h.modelsPanel(e, "")
183 }
```

**`internal/web/web.go:204-205`** — where the new routes mount:
```go
204 	se.Router.POST("/ui/model/select", h.selectModel)
205 	se.Router.POST("/ui/model/install", h.installModel)
```

**`internal/feature/modelcards/modelcard.go:16-31`** — the status enum + view model to extend:
```go
17 const (
18 	StatusActive    = "active"    // currently the active model (in use)
19 	StatusAvailable = "available" // GGUF present on disk; selectable
20 	StatusMissing   = "missing"   // GGUF file not found / not yet installed
21 )
24 type ModelView struct {
25 	ID     string
26 	Name   string
27 	Detail string
28 	Kind   string
29 	Status string
30 	VRAM   string
31 }
```

**`internal/feature/modelcards/panel.go:46,52-71`** — the panel body + the existing manual form (the CTA sits alongside it):
```go
46 	kids = append(kids, installForm())
...
52 func installForm() g.Node {
53 	return h.Section(h.Class("k-section"),
54 		ui.SectionLabel(ui.SectionLabelProps{Text: "Add a local model"}),
55 		h.Form(h.Class("card model-install-form"),
56 			data.On("submit", "@post('/ui/model/install', {contentType:'form'})", data.ModifierPrevent),
...
```

**`internal/feature/settingscards/settingsfocus.go:118-143`** — `BuildModelsPanelView`, the single source of truth for the panel view (add CTA + in-flight state here, not in the handler):
```go
118 func BuildModelsPanelView(app core.App, errMsg string) (modelcards.PanelView, error) {
119 	choices, _, err := turn.ModelChoices(app)
...
123 	view := modelcards.PanelView{Processor: kronk.Processor(), Error: errMsg}
124 	for _, c := range choices { ... }
142 	return view, nil
143 }
```

**`internal/kronk/presets.go:1-22`** — the env-getter shape to mirror for `ModelsDir()`:
```go
10 func LibPath() string { return os.Getenv("BALAUR_LIB_PATH") }
17 func Processor() string {
18 	if p := os.Getenv("BALAUR_PROCESSOR"); p != "" { return p }
21 	return "cpu"
22 }
```

**`internal/store/llm_settings.go:98-113`** — the atomic registration call to reuse:
```go
98 func SaveLocalModel(app core.App, path, embedPath string) (string, error) {
...
106 	model, err := findOrCreateLLMModel(app, provider.Id, "Local "+filepath.Base(path), path, embedPath, true)
...
112 	return model.Id, nil
113 }
```

## Decision: the downloader (recommended vs fallback)
Two viable mechanisms; **pick the recommended unless the owner says otherwise**, and record the choice in the plan's commit message.

- **RECOMMENDED — a small hand-rolled `net/http` downloader (~120 lines, stdlib only, CGO-free).** Gives exact control over the three things a multi-GB download with a good UX needs and that go-getter abstracts away: (1) **HTTP `Range` resume** into a `*.part` file (`Range: bytes=<existing>-`; on `206` append, on `200` truncate-restart), (2) a **streaming sha256** over the bytes as they land (`io.TeeReader(resp.Body, sha256.New())` — no second read pass), (3) a **throttled progress callback** (~250 ms) wired to the SSE loop with **`ctx` cancellation**. This is "copy 120 lines rather than widen a dependency" (AGENTS.md SUCKLESS) and keeps go-getter indirect.
- **FALLBACK — `download.GetModelWithProgress(url, dest, progress)`** from the already-vendored `github.com/hybridgroup/yzma/pkg/download`. Zero new code, but go-getter's HTTP getter gives **no robust Range-resume** and **no streaming checksum** (you'd verify after a full re-download), and its progress tracker is shaped for stdout. Acceptable only if the owner wants the absolute-smallest diff and accepts no-resume.

The rest of this plan assumes the RECOMMENDED path; where the fallback changes a step it is noted.

## The official-model pin (decision to confirm before merge)
"Our official model" is a **single code-level pin**, git-auditable and diff-reviewed — never runtime config. The owner asked for *"at least Qwen3.5 or Gemma 4"* — both shipped in 2026 and both have real GGUFs.

Recommended default for the owner's box (26 GB RAM, AMD Radeon 780M iGPU, x86-64): **`gemma-4-E4B-it-Q4_K_M.gguf`** from **`ggml-org/gemma-4-E4B-it-GGUF`** (~5.34 GB, **Apache-2.0** — Gemma 4 is the first Gemma under Apache-2.0, not the old custom Gemma Terms). Why this is the best *official* pin specifically:
- **First-party.** `ggml-org` is the llama.cpp team itself — the GGUF is canonical, well-mirrored, and served from HF `resolve/main` URLs that honor HTTP `Range` (so resume works). Pinning the official model to the same org as the runtime (plan 087) is the cleanest supply story.
- **Sized right for an iGPU box.** E4B (~4.5B effective) is fast first-token on CPU and offloadable to the 780M via `BALAUR_PROCESSOR=vulkan`; ~5.3 GB on disk, ~6–7 GB working set at 4096 ctx — comfortable in 26 GB.
- **Modern + permissive.** Gemma 4 (Apr 2026), Apache-2.0, multimodal-capable base.

**Lighter alternative** (clearly Apache-2.0, smaller): `Qwen3.5-4B` Q4_K_M from `bartowski/Qwen_Qwen3.5-4B-GGUF` (~3.01 GB) — faster/smaller, third-party quant. **Higher-quality alternative** if the owner wants more headroom: `ggml-org/gemma-4-26B-A4B-it-GGUF` (MoE, 4B active → fast inference, but ~15–16 GB resident — fits 26 GB, slower load) or `Qwen_Qwen3.5-9B`. Pick one; the default is Gemma 4 E4B.

> **STOP / decision**: the executor must (1) confirm the chosen model with the owner if they want something other than the Gemma 4 E4B default, then (2) download the file once, run `sha256sum` on it, and paste the **real** hash + exact byte size into the pin (Step 1). Do not invent a hash. A wrong/placeholder hash makes every download fail the checksum gate (which is the correct fail-closed behavior, but blocks the feature).

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Drift check | see header | no output / excerpts match |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test | `go test ./...` | all pass |
| Downloader tests | `go test ./internal/kronk/modelget/...` | ok |
| Storybook render | `go test ./internal/feature/storybook/...` | ok |
| Route registered | `grep -n '/ui/model/download' internal/web/web.go` | 2 routes |
| Whitespace | `git diff --check` | no output |
| Run app | `go run . serve --http=127.0.0.1:8090` | serves `/` |

Sandbox note: in a TLS-intercepting Hyperagent sandbox, Go commands need the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope
**In scope** (files you create or modify):
- **NEW** `internal/kronk/officialmodel.go` — the pin struct + the one pinned default.
- `internal/kronk/presets.go` — add `ModelsDir()` (and a tiny `RuntimeInstalled()` presence helper, Step 3).
- **NEW** `internal/kronk/modelget/` (`modelget.go` + `modelget_test.go`) — the streaming downloader.
- `internal/feature/modelcards/modelcard.go` + `panel.go` — `StatusDownloading` + progress fields, the meter, the Cancel form, the "Get our official model" CTA.
- `internal/feature/settingscards/settingsfocus.go` — thread the CTA + in-flight state into `BuildModelsPanelView`.
- `internal/web/models.go` — `downloadOfficialModel` (SSE) + `cancelDownload` handlers.
- `internal/web/web.go` — register `POST /ui/model/download` and `/ui/model/download/cancel`.
- `internal/feature/storybook/stories_settings.go` — `StatusDownloading` + download-error stories.
- Docs same-commit (AGENTS.md "self-knowledge is part of the change"): `internal/self/knowledge.md:18`, `README.md` env table, the AGENTS.md "owner-initiated downloads are deferred" line.

**Out of scope** (do NOT touch):
- The **runtime/native-library** download — that is plan **087**. This plan only *detects* runtime presence and shows an Alert when absent (Step 8); it never fetches a `.so`.
- The manual `installForm` / `/ui/model/install` path — leave it working unchanged.
- Multi-model catalog / model browser, delete/uninstall UI, `BALAUR_HF_TOKEN` private-repo gating beyond an optional unlogged header (Step 4) — deferred.
- Any migration — the `llm_*` collections already exist; download state is transient (`app.Store()` sidecar), not a new collection.

## Git workflow
- Branch `improve/086-official-model-download` off `main`.
- Conventional commits, bisectable by layer, e.g.:
  - `feat(kronk): pin the official model + add ModelsDir/RuntimeInstalled`
  - `feat(kronk): streaming GGUF downloader (range-resume, sha256, cancel)`
  - `feat(web): one-click official-model download with live progress`
  - `docs: flip owner-initiated-download deferral; add BALAUR_MODELS_DIR`
- Do NOT push or open a PR unless explicitly told.

## Steps

### Step 1: Pin the official model
Create `internal/kronk/officialmodel.go`:
```go
package kronk

// OfficialModel is Balaur's one curated, owner-installable local model. The URL,
// SHA256, and SizeBytes are a git-auditable pin: changing the official model is a
// reviewed code change, never runtime config. The download verifies SHA256 before
// the file is ever registered, so a stale pin fails closed.
type OfficialModel struct {
	Name      string // display name, e.g. "Gemma 4 E4B"
	URL       string // public HTTPS GET that honors Range (HF resolve URL)
	SHA256    string // lowercase hex; verified over the streamed bytes
	SizeBytes int64  // exact size; drives the pre-flight disk check + progress total
	Quant     string // e.g. "Q4_K_M"
	Params    string // e.g. "7B"
	License   string // e.g. "Apache-2.0"
	FileName  string // canonical .gguf filename (matches the URL's basename)
}

// Official returns the pinned model. (Single entry for v1; a curated list is
// deferred — YAGNI until a second model earns its place.)
func Official() OfficialModel {
	return OfficialModel{
		Name:      "Gemma 4 E4B",
		URL:       "https://huggingface.co/ggml-org/gemma-4-E4B-it-GGUF/resolve/main/gemma-4-E4B-it-Q4_K_M.gguf",
		SHA256:    "REPLACE_WITH_REAL_SHA256", // STOP: compute via sha256sum before merge
		SizeBytes: 0,                          // STOP: paste real byte size (~5.34 GB) before merge
		Quant:     "Q4_K_M",
		Params:    "E4B (~4.5B eff.)",
		License:   "Apache-2.0",
		FileName:  "gemma-4-E4B-it-Q4_K_M.gguf",
	}
}
```
**Verify**: `CGO_ENABLED=0 go build ./internal/kronk/...` → exit 0. (The placeholder hash/size compile fine; they are filled before merge — see the Step 1 STOP.)

### Step 2: Add `ModelsDir()` to `internal/kronk/presets.go`
Mirror the `LibPath()`/`Processor()` env-getter shape. Default to a concrete XDG data path (do **not** derive it from `LibPath()`, which is empty when unset):
```go
// ModelsDir returns the directory downloaded GGUF model files live in
// (BALAUR_MODELS_DIR). Empty falls back to the XDG data dir
// ~/.local/share/balaur/models. Like LibPath/Processor this is a lazy getter —
// no module-level global holding a derived path (AGENTS.md).
func ModelsDir() string {
	if d := os.Getenv("BALAUR_MODELS_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "models"
	}
	return filepath.Join(home, ".local", "share", "balaur", "models")
}
```
Add the `path/filepath` import if not present. **Verify**: `go build ./internal/kronk/...`, and `go test ./internal/kronk/...` if a presets test exists.

### Step 3: Add a runtime-presence helper
So handlers can ask "is the native llama.cpp runtime installed?" without forcing `kronk.Init`. Reuse the existing resolution (`resolveLibDir` in `internal/kronk/engine.go:83-100` + the `version.json`/library-file semantics). Add to the `kronk` package (e.g. in `presets.go` or a small `runtime.go`):
```go
// RuntimeInstalled reports whether a usable llama.cpp library is resolvable for
// the active processor — i.e. the variant dir resolves AND contains the platform
// library file (libllama.so on linux). It performs NO dlopen and never inits the
// engine, so it is cheap to call on every panel render.
func RuntimeInstalled() bool {
	dir, err := resolveLibDir(LibPath(), Processor())
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, libraryFileName()))
	return err == nil
}
```
Use `download.LibraryName(runtime.GOOS)` (already importable via the yzma `download` pkg used in `engine.go`) for `libraryFileName()` (`libllama.so`/`llama.dll`/`libllama.dylib`), or hardcode `libllama.so` with a comment that non-linux is plan-087/deferred. **Verify**: `go build ./internal/kronk/...`.

### Step 4: Build the streaming downloader (`internal/kronk/modelget`)
New package `internal/kronk/modelget`, pure Go (CGO stays off). Public surface roughly:
```go
package modelget

type Progress struct {
	Current, Total int64
	BytesPerSec    float64
	Done           bool
}

// Fetch downloads url into destDir as fileName, resuming from a <fileName>.part if
// present (HTTP Range), streaming a sha256 it checks against wantSHA256, then
// atomically renames to the final path. It calls onProgress at most every ~250ms.
// Honors ctx cancellation (a cancel leaves the .part for a later resume). On a
// checksum mismatch it deletes the .part and returns an error (never renames).
func Fetch(ctx context.Context, url, destDir, fileName, wantSHA256 string,
	wantSize int64, token string, onProgress func(Progress)) (finalPath string, err error)
```
Requirements:
- Pre-flight free-space check on `destDir` for `wantSize` + ~10% margin (use `golang.org/x/sys/unix.Statfs` or `syscall.Statfs` behind an OS-agnostic seam — AGENTS.md "OS-agnostic by construction"; non-linux may stub to "skip check" for v1 with a comment). On short space, return a clear error **before** opening the stream.
- Resume: if `<fileName>.part` exists, `stat` it and send `Range: bytes=<size>-`; seed the sha256 by re-reading the existing `.part` bytes first (a resumed checksum must cover the whole file). On `200 OK` (server ignored Range) truncate and restart the hash.
- `token` (optional, from `BALAUR_HF_TOKEN`): set `Authorization: Bearer <token>` only when non-empty; **never** log it or put it in the URL/audit.
- Atomic finish: `fsync` the `.part`, verify the hash, then `os.Rename` to `<fileName>` (same filesystem ⇒ atomic). The final `.gguf` path therefore exists **only** when complete + verified.
- Dedupe: if the final `<fileName>` already exists and its size == `wantSize`, return it immediately (skip the network).

**Tests** (`modelget_test.go`, `httptest` + `t.TempDir()`, table-driven): fresh download → verify → rename; resume from a partial `.part` against a Range-honoring test server; checksum mismatch → no rename, `.part` deleted, error; insufficient-disk pre-flight abort (inject the space check via the seam); `ctx` cancel mid-stream leaves a resumable `.part`; dedupe-by-size short-circuit. **Verify**: `go test ./internal/kronk/modelget/...` → ok.

### Step 5: Extend the view model + render the new states
In `internal/feature/modelcards/modelcard.go`: add `StatusDownloading = "downloading"` to the status enum, and progress fields to `ModelView` (e.g. `Progress int // 0..100`, `ProgressLabel string // "1.9 / 4.7 GB · 41% · 12 MB/s"`). In `modelAction`/`ModelCard`, for `StatusDownloading` render a **determinate progress meter** carrying a **stable id** `h.ID("model-dl-progress")` plus a **Cancel** form that `@post`s `/ui/model/download/cancel` (mirror `actionForm` at `modelcard.go:75-82` — action on the form, hidden fields). In `internal/feature/modelcards/panel.go`: add a **"Get our official model"** CTA section (a `ui.Button` whose form `@post`s `/ui/model/download`, showing the pinned name + size + license) rendered **only when the official model is not yet installed**. Keep `installForm()` exactly as-is below it (manual path stays available). **Verify**: `go build ./...`; `go test ./internal/feature/storybook/...`.

### Step 6: Thread CTA + in-flight state through `BuildModelsPanelView`
Extend `modelcards.PanelView` with the fields the CTA/in-flight card need (e.g. `OfficialAvailable bool`, `Official modelcards.OfficialCTA`, `Downloading *modelcards.ModelView`), and populate them in `settingscards.BuildModelsPanelView` (`settingsfocus.go:118`) from `kronk.Official()` + whether the file already exists at `kronk.ModelsDir()/<FileName>` + the in-flight flag (Step 7). `BuildModelsPanelView` stays the **single source of truth** — the handler must not assemble panel markup itself. **Verify**: `go build ./...`; `go test ./...`.

### Step 7: Web handlers (`internal/web/models.go`)
Add next to `installModel`:
- `downloadOfficialModel(e *core.RequestEvent) error` — a long-lived **SSE** handler:
  1. Open `sse := datastar.NewSSE(e.Response, e.Request)` (as `chatstream.go` does).
  2. Guard a **single-in-flight** flag on `h.app.Store()` (same App-scoped sidecar pattern as `kronk.StoreKey`); a second concurrent POST re-attaches to / reflects the current state rather than starting a second writer on the same `.part`.
  3. Render the panel with the model card in `StatusDownloading` (via `BuildModelsPanelView`) and patch `#models-panel` once up front.
  4. Run `modelget.Fetch(ctx, m.URL, kronk.ModelsDir(), m.FileName, m.SHA256, m.SizeBytes, os.Getenv("BALAUR_HF_TOKEN"), onProgress)`. `onProgress` morphs **only** `#model-dl-progress` (`sse.PatchElements(meterNode)` with the node carrying that id — the default morph-by-id mode, exactly the `chatstream.go` per-bubble morph).
  5. On success: `store.SaveLocalModel(h.app, finalPath, "")` → `store.SetActiveLLMModel(h.app, id, "owner")` → `store.Audit(h.app, "owner", "llm.model.install", finalPath, true, nil)` → `h.modelsPanel(e, "")` so the card flips to **"In use"** (reuse `installModel`'s tail, `models.go:174-182`).
  6. Audit lifecycle edges with a new action `llm.model.download`: start (`target=URL`, `detail={sha256,size}`), cancel (`allowed=false, detail={reason:"cancelled"}`), checksum-fail (`allowed=false, detail={expected,got}`). Clear the in-flight flag in a `defer`.
- `cancelDownload(e *core.RequestEvent) error` — flip the cancel `context`/flag the running `Fetch` selects on; patch the card to a **"paused — resume"** state (the `.part` is kept). 

**Fallback note**: if using `download.GetModelWithProgress`, wire its `getter.ProgressTracker` to the same `onProgress`, drop the resume/streaming-checksum guarantees, and verify sha256 by re-hashing the finished file before `SaveLocalModel`. **Verify**: `go build ./...`; `go vet ./...`.

### Step 8: Lib-presence precheck UX (the seam to plan 087)
Before offering / on completing the download, call `kronk.RuntimeInstalled()` (Step 3). If it returns **false**, do **not** present the downloaded model as runnable: in `BuildModelsPanelView` render a `ui.Alert` (as `panel.go:25` does) — *"The local AI runtime isn't installed yet. Set `BALAUR_LIB_PATH` to a llama.cpp build (see the README env table), or install it from here once that lands (plan 087)."* — and show a **"runtime not installed"** sub-state instead of "In use". The **model download still proceeds** (the GGUF is independent of the `.so` and resumes fine); we just refuse to imply chat works. This is the explicit handoff point: plan 087 replaces this Alert with a one-click runtime install. **Verify**: with `BALAUR_LIB_PATH` unset, the panel shows the runtime Alert, not a false "In use".

### Step 9: Routes
In `internal/web/web.go`, beside `:204-205`:
```go
	se.Router.POST("/ui/model/download", h.downloadOfficialModel)
	se.Router.POST("/ui/model/download/cancel", h.cancelDownload)
```
**Verify**: `grep -n '/ui/model/download' internal/web/web.go` → 2 lines; `go build ./...`.

### Step 10: Storybook stories
In `internal/feature/storybook/stories_settings.go` (existing `modelcardStory`/`modelspanelStory` at `:11-76`): add a `StatusDownloading` variant (with a determinate meter + Cancel) and a **"download error"** panel variant (checksum-mismatch Alert). Add the new props (`Progress`, `ProgressLabel`) to the `Props` table and a Do/Don't ("Show real bytes + speed so a multi-GB download feels alive"; "Don't mark a model 'In use' before its checksum verifies"). **Verify**: `go test ./internal/feature/storybook/...` → ok; eyeball at `/storybook`.

### Step 11: Docs (same commit — AGENTS.md)
- `internal/self/knowledge.md:18` — change *"the engine never downloads them on boot"* to reflect reality: the engine still never downloads on **boot**, but the owner can now download the official **model** in-app. Keep the boot-time guarantee; soften the absolute "never downloads."
- `README.md` env table — add `BALAUR_MODELS_DIR` (default `~/.local/share/balaur/models`) and a one-line note that the Models page can fetch the official model.
- `AGENTS.md` — the "Owner-initiated downloads … are deferred work" line (in the Known-limitations / kronk bullet) now partly ships; update it to "owner-initiated **model** download ships (plan 086); owner-initiated **runtime** download is plan 087." Do **not** overclaim.
**Verify**: `grep -n 'BALAUR_MODELS_DIR' README.md` → present.

### Step 12: Full verification + index
Run the Done-criteria gate. Update the 086 row in `plans/readme.md`.

## Test plan
- **`internal/kronk/modelget`** — the table tests in Step 4 are the core regression net (resume, checksum-mismatch, disk-space, cancel, dedupe) using `httptest` + `t.TempDir()`. No real network, no real model.
- **Web handler** — add a test in `internal/web/handlers_test.go` that fakes the downloader (inject the `modelget.Fetch` func, or point the pin at an `httptest` URL serving a tiny fake `.gguf` with a known sha256) and asserts the success path calls `SaveLocalModel` + `SetActiveLLMModel` and that `BuildModelsPanelView` then reports `StatusActive` ("In use") with **no path input**. Fake the model client per the existing `turn` test seam — tests never hit a real model.
- **Storybook** — `go test ./internal/feature/storybook/...` covers the new stories render.
- **Manual (with a runtime present, i.e. after 087 or with `BALAUR_LIB_PATH` set)**: click "Get our official model", watch `#model-dl-progress` morph live, confirm the card flips to "In use" and a chat turn loads the model; click Cancel mid-download and confirm a resumable state; re-click and confirm it resumes (not restarts). Inspect `audit_log` for `llm.model.download` → `llm.model.install` → `llm.active_model` rows. **With `BALAUR_LIB_PATH` unset**: confirm the runtime-missing Alert shows instead of a false "In use" (Step 8).

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0.
- [ ] `go vet ./...` → exit 0.
- [ ] `go test ./...` → all pass, incl. `internal/kronk/modelget` and the new web-handler test.
- [ ] `go test ./internal/feature/storybook/...` → ok; `/storybook` shows the downloading + error states.
- [ ] `grep -n '/ui/model/download' internal/web/web.go` → exactly 2 routes.
- [ ] The official-model pin has a **real** sha256 + byte size (no `REPLACE_WITH_REAL_SHA256`, no `SizeBytes: 0`).
- [ ] A verified download registers via `SaveLocalModel`+`SetActiveLLMModel` (reused, not reimplemented) and the card reads "In use" with no path typing.
- [ ] With `BALAUR_LIB_PATH` unset, the panel shows the runtime-missing Alert (Step 8), not a false "In use".
- [ ] Cancel leaves a resumable `.part`; a checksum mismatch never renames/activates and is audited `allowed=false`.
- [ ] `BALAUR_HF_TOKEN`, when set, never appears in logs, the URL, or `audit_log`.
- [ ] `git diff --check` → no output.
- [ ] Docs updated same commit: `knowledge.md:18`, README env table (`BALAUR_MODELS_DIR`), AGENTS.md deferral line.
- [ ] `plans/readme.md` 086 row updated.

## STOP conditions
- **Drift**: the header drift check shows any cited file changed since `3136bad` and an excerpt no longer matches — STOP and report.
- **Placeholder hash reaches a build you intend to merge** — STOP; compute the real `sha256sum` of the chosen GGUF first (Step 1 decision).
- **The download wants to write the final `.gguf` before the checksum verifies** — STOP; that violates the fail-closed contract. Only ever rename a verified `.part`.
- **You find yourself adding a migration or a new collection** — STOP; download state is a transient `app.Store()` sidecar, not schema.
- **The runtime (native lib) download starts creeping into scope** — STOP; that is plan 087. This plan only *detects* runtime presence.
- **A Verify fails twice** after a fix attempt — STOP and report the command + output.

## Maintenance notes
- **Plan 087** replaces the Step-8 runtime-missing Alert with a one-click cpu/vulkan runtime install; it deliberately **reuses this plan's SSE-progress + single-in-flight (`app.Store()`) + audit primitives**. Land 086 first so 087 extends rather than re-invents them.
- The pin is **fail-closed by design**: if Hugging Face re-publishes the file under `resolve/main`, the sha256 stops matching and downloads fail until the pin is re-reviewed. Do **not** "fix" a mismatch by dropping the checksum — re-pin deliberately.
- The model is a **re-downloadable cache** under `BALAUR_MODELS_DIR`, never `pb_data/` (AGENTS.md: models out of git/data dir). Deleting it just means a re-download.
- **Deferred (record, don't build)**: a multi-model curated catalog, an uninstall/delete-model UI, private-repo (`BALAUR_HF_TOKEN`) browsing, and background/queued downloads that survive a page close. The first slice is one button, one model, one in-flight download.
- A reviewer should scrutinize: (1) the checksum is streamed and gates the rename (not verified after a blind write); (2) cancel/finish races can't half-activate a model (check the flag once more before rename+activate); (3) `BuildModelsPanelView` remains the only place panel markup is assembled.
