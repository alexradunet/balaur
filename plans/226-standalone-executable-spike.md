# Plan 226 (DIR-01): North-star spike — what stands between today's binary and a non-technical owner's double-click?

> **Direction bet / north-star audit.** This is the most strategic and least
> code-prescriptive of the six. Its Phase 0 deliverable is an honest gap map +
> a recommended next slice, NOT an implementation. Balaur's north star is "the
> single standalone executable for a non-technical owner" (PRODUCT.md); this
> plan measures the distance to it and turns it into a backlog, so the gap is
> tracked instead of assumed-closed.

## Status

- **Priority**: P3 (direction / strategic)
- **Effort**: Phase 0 audit = M; the slices it spawns = varies
- **Risk**: LOW (the spike); the slices it identifies carry their own risk
- **Depends on**: none
- **Category**: direction / north-star
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

The product promise is a single file a non-technical owner runs and *it just
works*. The no-args launcher (plan 190) gets impressively close — a bare
`balaur` already self-serves:

```go
// main.go:41-46 — bare argv → serve on a free loopback port + open browser
if launch.IsLauncherInvocation(os.Args[1:]) {
	...
	port, err := launch.SelectPort()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	os.Args = append(os.Args[:1], "serve", "--http", addr, "--dir", launch.DataDir())
	fmt.Fprintf(os.Stderr, "Balaur is starting — open %s in your browser.\n", url)
	go func() { launch.OpenAfterReady(addr) ... }()
}
```

But "double-click and it works" still has unmet preconditions that a *technical*
operator silently satisfies today. The biggest: **the binary cannot do its core
job — local inference — out of the box.** Two runtime assets are owner-supplied
and the engine *never fetches them on boot*:

```go
// internal/kronk/presets.go:8-13 — the llama.cpp lib is NOT bundled, NOT auto-fetched
// LibPath returns the llama.cpp library root (BALAUR_LIB_PATH). Empty hands
// resolution to Kronk's own default root. ... Slice 1 never downloads it — a
// missing library surfaces as a plain error at first inference.
func LibPath() string { return os.Getenv("BALAUR_LIB_PATH") }
```

```go
// internal/kronk/presets.go:27-29 — the model file is owner-supplied too
// ModelsDir returns the directory downloaded GGUF model files live in ...
```

So today's honest reality: a developer with `BALAUR_*` env vars and a model on
disk gets a working binary; a non-technical owner double-clicking a fresh
download gets a web UI that **errors on the first message**. That's the gap
between "ships as a binary" (true) and "is the standalone product" (not yet).
This plan refuses to let that gap stay invisible.

## Phase 0 — the gap map (do this now)

Produce a written audit, one page, of every precondition between a fresh download
and a working first conversation, each tagged **met / partial / unmet**. Seed it
with what's already verified:

| Precondition | State at `ef9f2df` | Evidence |
|---|---|---|
| Launch with no terminal/args | **met** | `main.go:41` launcher (plan 190) |
| Pick a free port, open browser | **met** | `launch.SelectPort`/`OpenAfterReady` |
| Data dir auto-created (XDG) | **met** | `launch.DataDir()` |
| Schema present | **met** | Go migrations, `Automigrate:false` (main.go:77-80) |
| **llama.cpp runtime present** | **unmet** | not bundled, not auto-fetched (`presets.go:8-13`); owner-initiated install ships (plan 087) but is a Models-page action, not first-boot |
| **A model file present** | **unmet** | owner-supplied; download ships (plan 086) but is owner-initiated post-boot |
| First-run onboarding | **partial** | `launch.IsFirstRun` exists but is explicitly `_ =`-discarded, "Reserved for Phase 2" (`main.go:48-53`) |
| Works offline after setup | **met** (once assets present) | engine is in-process, CGO-free |

Then for each **unmet/partial** row, write the smallest slice that would close it
and its cost/risk. Known anchors for those slices:

- **First-run onboarding** — the seam already exists (`launch.IsFirstRun`,
  `main.go:48-53`), deliberately inert. The slice: on first run, route the owner
  to a guided setup (pick/download a runtime + model) instead of a chat box that
  errors. This is the highest-leverage unmet item — it converts the two asset
  gaps from "silent error" into "guided step."
- **Runtime/model on first boot** — the machinery exists (plan 086 download, 087
  install into `LibRoot()` = `~/.local/share/balaur/kronk/lib`); the gap is that
  nothing *drives* it before the first turn. Decide: bundle vs guided-fetch vs
  fail-with-instructions. (Bundling fights the "single small file" goal — the
  full engine is already ~+33MB per AGENTS.md; a bundled model is hundreds of MB.
  Guided-fetch is likely the right bet, but that's the spike's job to recommend.)

### Decision gate

Recommend the **one** next slice with the highest "distance-to-north-star
closed per unit risk." The likely answer is *first-run onboarding that drives the
existing download/install machinery* — but let the gap map argue it, don't
assume it.

## Current state (verified at `ef9f2df`)

- Launcher: `main.go:40-73`; `internal/launch/launch.go`
  (`IsLauncherInvocation`, `SelectPort`, `DataDir`, `IsFirstRun`,
  `OpenAfterReady`).
- Runtime/model resolution: `internal/kronk/presets.go`
  (`LibPath` :13, `Processor` :20, `ModelsDir` :30, `LibRoot` :45).
- Shipped-but-owner-initiated: model download (plan 086), runtime install (plan
  087) — see AGENTS.md "Known limitations."

## Done criteria (Phase 0)

- [ ] A written gap map (met/partial/unmet) committed under
      `docs/superpowers/specs/`, covering fresh-download → first working turn.
- [ ] Each unmet/partial row has a proposed slice with cost + risk.
- [ ] A single recommended next slice, argued from the map.
- [ ] No product code changed in this phase.

## STOP conditions

- This is a spike — if it turns into implementing onboarding, stop and split that
  into its own owner-approved plan.
- Do not recommend bundling a model into the binary without explicitly weighing
  it against the "single small file" promise and the ~+33MB engine cost already
  accepted — that trade-off must be argued, not defaulted.

## Notes

- The honest framing for any user-facing copy until the unmet rows close: Balaur
  *ships* as one binary; it is not yet *zero-setup* for a non-technical owner.
  Keep that distinction out of marketing claims (overlaps plan 214 doc-truth).
