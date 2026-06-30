# Standalone-executable gap map — plan 226 Phase 0

> Spike deliverable. No product code changed here.
> North star (PRODUCT.md:36-38): "download one file, run it, get a working
> companion — no terminal, no env file, no model hunt."
> Verified against the tree at the time of writing (branch
> `advisor/226-standalone-gap-map`, rebased on main including plans 212–224).

## 1. Precondition table

Each row covers one thing a non-technical owner needs between "fresh binary
download" and "first working turn." Evidence anchors are file:line of the
claim verified in live code.

| # | Precondition | State | Evidence |
|---|---|---|---|
| 1 | Launch with no terminal or args | **met** | `main.go:47` — `launch.IsLauncherInvocation` fires on empty argv, rewrites to `serve --http 127.0.0.1:<port> --dir <DataDir()>` |
| 2 | Pick a free port; prefer stable bookmarkable URL | **met** | `internal/launch/launch.go:90-96` — `SelectPort()` tries 8099 first, falls back to kernel-assigned free port |
| 3 | Open browser after server is ready | **met** | `internal/launch/launch.go:146-156` — `OpenAfterReady()` polls the loopback listener (100 ms tick, 15 s deadline), then `OpenBrowser` |
| 4 | Data dir auto-created (XDG) | **met** | `internal/launch/launch.go:27-36` — `DataDir()` returns `~/.local/share/balaur/pb_data`; PocketBase creates it on `app.Start()` |
| 5 | Schema present on first boot | **met** | `main.go:77-80` — Go migrations in `migrations/` registered with `Automigrate: false`; run at every `app.Start()` |
| 6 | Product UI reachable without PocketBase admin setup | **met** | `internal/web/web.go:53-68` — `guardLocalUI` routes `/` to Balaur's own handlers; PocketBase's admin setup page is at `/_/` and does not gate `/` |
| 7 | llama.cpp runtime (libllama.so) present | **unmet** | `internal/kronk/engine.go:8-9` — "the engine never downloads anything on boot"; `internal/kronk/runtime.go:11-26` — `RuntimeInstalled()` is a stat-only check; install machinery exists at `internal/web/models_install.go:209-267` but is owner-initiated from the Models page |
| 8 | A model file (GGUF) present and active | **unmet** | `internal/kronk/presets.go:30-38` — `ModelsDir()` is `~/.local/share/balaur/models` but nothing lives there on a fresh download; `internal/turn/models.go:186-188` — `Active()` returns `"no active model is available"` |
| 9 | First-run onboarding routes owner to setup | **partial** | `internal/launch/launch.go:44-46` — `IsFirstRun()` exists (stat, never downloads); `main.go:48-53` — signal computed but immediately discarded (`_ = launch.IsFirstRun(...)`) with comment "Reserved for Phase 2 onboarding"; `internal/web/home.go:262-272` — chatbar shows "No model is ready yet." + "Set up a model →" link + 2 s self-refresh poll, but the owner must discover and navigate the Models page themselves |
| 10 | Works offline after initial setup | **met** (once assets present) | `internal/kronk/engine.go:1-10` — in-process, CGO-free; all inference is local after the one-time download |
| 11 | OS packaging (double-click artifact) | **unmet** | `docs/first-run-design.md:118-143` — macOS `.app`, Linux `.desktop`, Windows installer all sketched but not built; explicitly out-of-repo per AGENTS.md; no release-pipeline infrastructure exists |
| 12 | Single-instance guard | **unmet** | `docs/first-run-design.md:141-143` — "Not built; needs a lockfile or a fixed-port probe"; a second double-click starts a second server on a second port |
| 13 | linux/arm64 runtime checksum verified | **partial** | AGENTS.md — "linux/arm64 stays placeholder (out of v1 scope — those installs download unverified until hashes are added)"; `internal/kronk/runtime.go:24` — libllama.so stat is linux-specific; comment "non-linux is plan 087" |

## 2. Closing slices for each unmet / partial row

### Row 7 — llama.cpp runtime not present

**Gap:** On a fresh machine the runtime `.so` is absent. The engine boots
lazily so the binary does not crash, but the first turn fails with
`"initializing local inference runtime (cpu): ..."` (`internal/kronk/engine.go:82-84`).

**Smallest closing slice:** Drive the existing `installRuntime()` handler
(`internal/web/models_install.go:209-267`) from the first-run flow (row 9 below).
No new install logic needed — the slot already exists and is tested.

**Do not:** auto-install the runtime silently on boot without owner awareness.
The install is ~33 MB and requires network; a silent first-boot download violates
the Consent pillar and surprises owners on metered connections.

**Cost:** small — ~20 lines routing + passing a `firstRun` flag to the web
layer. The hard part (download, checksum, SSE progress) is already built.

**Risk:** low. The only new code is the routing condition; the machinery is
exercised by existing tests (`internal/web/handlers_test.go`).

**Bundling trade-off:** bundling the runtime in the binary would eliminate the
download step. The engine already adds ~33 MB (accepted cost, AGENTS.md). A
bundled runtime `.so` would add roughly the same again, making the download
~66+ MB. More importantly, a model (hundreds of MB) cannot be bundled either, so
the owner still needs a download step for the model. Bundling the runtime saves
one click but doubles the download size for every subsequent release, forces a
binary rebuild every time the llama.cpp version changes, and fights the "single
small file" promise. Guided-fetch is the right call: the runtime is a one-time
install into `LibRoot()` and survives binary upgrades.

---

### Row 8 — No model file present

**Gap:** Even after the runtime is installed, no GGUF file lives in
`ModelsDir()` on a fresh machine. `Active()` returns "no active model is
available" and every turn fails.

**Smallest closing slice:** As part of the first-run flow (row 9), after
runtime install completes, pre-select the recommended "medium" curated model
and offer a single "Download Balaur's recommended model" button. The
`downloadOfficialModel()` handler (`internal/web/models_install.go:58-162`)
already handles this end-to-end: SHA256 verification, resume-safe `.part`
files, SSE progress, DB registration, and activation.

**Cost:** small — the guided flow is a UI frame around existing handlers; no
new download logic.

**Risk:** low-medium. The download is network-dependent and ranges from ~1 GB
(small quantization) to ~8 GB (large). The handler already surfaces errors and
supports cancellation. The risk is a poor experience on slow or
quota-constrained connections — mitigated by the existing progress meter and the
option to use a smaller curated model first.

---

### Row 9 — First-run onboarding (partial → met)

**Gap:** `IsFirstRun` is computed and immediately discarded at `main.go:53`.
The owner lands on the chat home with "No model is ready yet." and a "Set up a
model →" link — accurate but passive. A non-technical owner may not connect
"set up a model" with "I also need to install a runtime first."

**Smallest closing slice (the highest-leverage item — see §3):**

1. Pass the `isFirstRun` bool from `main.go:53` into `web.Register` (e.g., as
   a field on `handlers` or a one-time `app.Store()` flag set before
   `OnServe`).
2. On the first `GET /`, if `isFirstRun`, redirect to
   `/ui/show/settings?section=models` with a `?firstrun=1` query param.
3. The models panel reads `firstrun=1` and renders a lightweight banner:
   "Welcome — let's get your companion running. Install the inference engine,
   then download a starter model." Two existing buttons do the work; the banner
   just sets context.
4. The chatbar already polls every 2 s (`internal/web/home.go:235`) and
   unlocks automatically once a model is active. No extra wiring needed for
   the "done" signal.

**Cost:** small. ~50-80 lines total: a flag threaded through web.Register,
a redirect in the root handler, and a conditional banner in the models panel.

**Risk:** low. The worst-case false positive (IsFirstRun fires on a
re-download to a new machine that already has an existing data dir) is handled
by the banner being informational, not gating — the owner can dismiss it and
proceed to chat. The IsFirstRun check is cheap (one os.Stat) and already
present; no new filesystem operations.

---

### Row 11 — OS packaging (double-click artifact)

**Gap:** The binary has no OS packaging. On macOS it will be quarantined by
Gatekeeper; on Windows, blocked by SmartScreen. On Linux it needs a `.desktop`
file to appear in application launchers. Without packaging, "double-click"
requires the owner to chmod+x, move to a known path, and clear OS security
warnings — steps that disqualify the non-technical owner.

**Smallest closing slice:**

- **Linux:** a `.desktop` file + icon in a release archive; AppImage or Flatpak
  wrapping the binary for single-file distribution. Low signing cost; no notarization.
- **macOS:** an unsigned `.app` bundle in a `.dmg`. Works for technically
  comfortable users who can right-click → Open to bypass Gatekeeper. Full
  notarization requires an Apple Developer account (~$99/yr) and CI integration
  with the signing keychain.
- **Windows:** a Chocolatey/Scoop package or a basic installer (NSIS/WiX).
  Authenticode signing avoids SmartScreen ($200-400/yr EV certificate or a
  Microsoft-enrolled publisher certificate).

**Cost:** medium-to-high. This is release-pipeline work, not in-repo source.
It requires code-signing infrastructure, CI secrets for private keys, and
ongoing certificate maintenance. Per AGENTS.md: "keep host operating-system
setup outside this repository."

**Risk:** medium. An unsigned macOS build will be blocked by Gatekeeper for
most users. An unsigned Windows installer will trigger SmartScreen. Partial
mitigations exist (right-click → Open on macOS, "More info → Run anyway" on
Windows) but require technical savvy — exactly what the north-star owner lacks.
Linux is the lowest-risk target for the first packaging pass.

---

### Row 12 — Single-instance guard

**Gap:** A second launch (second double-click) starts a second server on a
different port. The owner ends up with two Balaur instances; browser tabs may
be split across them; data writes are not coordinated.

**Smallest closing slice:** On a bare-argv launch (in `internal/launch`),
before `SelectPort()`:
1. Try to acquire a per-user lockfile at
   `~/.local/share/balaur/balaur.lock` (or `<DataDir()>/../balaur.lock`).
2. If the lock is held: read the port from the lockfile, probe
   `127.0.0.1:<port>`, and if the server is live, open the browser to the
   existing instance and exit.
3. If the lockfile is stale (held but port not listening), overwrite it and
   proceed normally.

**Cost:** small. ~40 lines in `internal/launch`.

**Risk:** low. Stale lockfile handling is the main edge case; a port-liveness
probe (a TCP connect with a short timeout) resolves it.

---

### Row 13 — linux/arm64 runtime checksum unverified

**Gap:** `runtime_sums.json` has placeholder checksums for linux/arm64. An
arm64 owner (Raspberry Pi 5, Apple Silicon in a Linux VM, AWS Graviton) gets an
unverified install — the fail-closed checksum gate is bypassed.

**Smallest closing slice:** Build or cross-compile the arm64 `.so` on a real
arm64 machine, record its SHA256, and add it to `runtime_sums.json`. Also
confirm the `libllama.so` stat check in `RuntimeInstalledFor` works on
arm64 (the filename is the same; the SONAME may differ — verify at test time).

**Cost:** small in code (~2 lines in runtime_sums.json), medium in
infrastructure (access to an arm64 machine in CI to produce and verify the hash).

**Risk:** low once the hash is pinned. The current risk is that arm64 owners
install an unverified binary artifact — a supply-chain concern that warrants
prioritizing this before marketing to arm64 users.

## 3. Recommended next slice

**First-run onboarding routing the owner to the guided setup (Row 9 slice,
which drives Rows 7 and 8).**

### Argument from the gap map

The table has two kinds of gaps: infrastructure gaps (rows 11, 12, 13 — OS
packaging, single-instance guard, arm64 checksums) and UX gaps (rows 7, 8, 9 —
runtime, model, onboarding). The infrastructure gaps are real but are either
out-of-repo (packaging) or small standalone items (single-instance guard,
arm64). None of them affect what happens once the owner has the binary running
on a supported platform.

Rows 7 and 8 are what actually prevent a first working turn. But their
machinery is already built — `installRuntime()` and `downloadOfficialModel()`
are complete, tested, and accessible from the Models page. The owner just has to
find and use them.

Row 9 is the seam that connects the existing machinery to the owner's first
boot. The `IsFirstRun` bool is already computed at `main.go:53` and silently
discarded. Wiring it into a first-run redirect costs ~50-80 lines and closes
the "owner stranded at a chatbox with no path to setup" gap without implementing
new infrastructure.

### Why not bundle or auto-fetch instead?

- **Bundling the runtime:** rejected above (doubles download size per release,
  ties binary to a specific llama.cpp build, does not solve the model gap).
- **Auto-fetching runtime + model on first boot:** silently downloading hundreds
  of MB without owner consent violates the Consent pillar and surprises metered
  connections. The guided flow asks once, downloads with a visible progress
  meter, and gives the owner a cancel button.
- **OS packaging first:** necessary for a true "double-click" experience but
  is out-of-repo release work. It does not help owners who already have the
  binary and can run it from a terminal (the current beachhead). Packaging
  should land after onboarding works, not before.

### Scope contract

This slice is UI-only: a routing condition, a one-time flag, and a banner in
the existing models panel. It explicitly does NOT:

- Change the download/install machinery (already built).
- Change `IsFirstRun` semantics (already correct).
- Add new persistence (the flag is in-memory for the process lifetime; the data
  dir's existence is the durable signal).
- Add account/auth setup (V1 is one owner, no product auth).

### Success criterion (for a follow-on executor)

A fresh binary + fresh `~/.local/share/balaur/pb_data` (no prior data dir) +
bare invocation → browser opens → owner lands on the Models section with a
visible "Welcome, let's get your companion running" banner → one click installs
the cpu runtime → one click downloads the recommended model → chatbar unlocks →
first turn works. All within the running browser window, no terminal interaction
after the initial double-click.

---

## Known limitations not addressed in this gap map

- **macOS and Windows runtime detection:** `RuntimeInstalledFor` checks for
  `libllama.so` (Linux only; `internal/kronk/runtime.go:24`). The check for
  macOS (`.dylib`) and Windows (`.dll`) is deferred to plan 087. Until then the
  Models page cannot correctly report runtime status on those platforms.
- **Cloud fallback for no-runtime scenarios:** an owner without a model or
  runtime could use a cloud provider (Mistral via the openai-compatible client)
  as a bridge. This is already architecturally possible (the cloud path is
  built) but is not surfaced in the first-run flow. Adding a "use cloud while
  local model downloads" option is a follow-on to the onboarding slice above,
  not a prerequisite for it.
- **Restart after runtime install:** the llama.cpp library is dlopen'd once per
  process. After `installRuntime()` completes, the owner needs to restart Balaur
  for the newly installed runtime to load. The UI should communicate this
  clearly; it is a UX concern in the onboarding slice, not a new gap.
