# Plan 230: First-run onboarding — turn the silently-discarded `IsFirstRun` signal into a guided "set up your companion" banner on Home

> **Phase-1 build of plan 226's recommended slice.** The Phase-0 gap map
> (`docs/superpowers/specs/2026-06-30-standalone-executable-gap-map.md`, Row 9)
> argued this is the highest-leverage north-star slice: it converts the two
> asset gaps (no runtime, no model) from a silent first-message error into a
> guided step, using machinery that already exists. ~50–80 lines.
>
> **Drift check (run first)**:
> `git diff --stat 41abe4f..HEAD -- main.go internal/web/web.go internal/web/home.go internal/feature/settingscards/settingsfocus_models.go`
> If any changed since this plan was written, compare the "Current state"
> excerpts to the live code before editing; on a mismatch, STOP.

## Status
- **Priority**: P3 (direction / north-star)
- **Effort**: S–M (~50–80 lines + a story + a test)
- **Risk**: LOW (informational banner; never gates chat)
- **Depends on**: none (builds on shipped launcher plan 190 + Models page install/download flows plans 086/087)
- **Category**: direction / onboarding
- **Planned at**: commit `41abe4f`, 2026-06-30

## Why this matters

The bare-`balaur` launcher (plan 190) already self-serves a loopback web UI. But
a non-technical owner who double-clicks a fresh download lands on a chat box that
**errors on the first message** — the llama.cpp runtime and a model are both
owner-supplied and never fetched on boot. The install/download machinery is fully
built and owner-accessible from the Models page (`installRuntime` /
`downloadOfficialModel`), but nothing *routes the owner to it on first boot*.

The first-run signal already exists and is **thrown away**:
```go
// main.go:53
_ = launch.IsFirstRun(launch.DataDir())   // computed, immediately discarded ("Reserved for Phase 2")
```
This plan spends that signal: on first run, show a prominent, dismissible
onboarding banner on Home that points the owner at model setup. The chatbar
already self-heals (it polls every 2s and unlocks once a model is active), so
only the *start* signal is missing — the *done* signal is already wired.

## Design (informational banner, NOT a gate)

On the Home page, when **first-run AND no active model**, render a guided
onboarding banner (reusing the design system) above/within the chat shell:
> "Welcome — let's get your companion running. Install the inference engine, then
> download a starter model." with a button/link to the Models setup
> (`/ui/show/settings?section=models`).

It is **informational, never gating**: the owner can dismiss it and still use the
app. This makes the worst-case false positive (IsFirstRun fires on a re-download
onto a machine that already has a data dir) harmless — a stale banner, dismissed.

> **Why a banner, not an HTTP redirect.** `/ui/show/{type}` (web.go:241) is a
> Datastar *fragment* endpoint, not a full-page navigation target — redirecting
> `GET /` to it would render a bare fragment without the page shell. A banner in
> the Home shell is the idiomatic, low-risk mechanism. (Focusing/auto-opening the
> Models panel is OPTIONAL polish — only if it's a clean one-liner in the existing
> Datastar idiom; otherwise leave it to the banner's link. Do NOT reshape the root
> handler for it — see STOP conditions.)

## Current state (verified at `41abe4f`)

- **The discarded signal** — `main.go:53`: `_ = launch.IsFirstRun(launch.DataDir())`.
  `launch.IsFirstRun` (internal/launch/launch.go) is a cheap `os.Stat`-based check
  that is only meaningful **at boot, before data is written** — so it MUST be
  captured at startup and stashed, not recomputed on-demand in the handler.
- **Web wiring** — `internal/web/web.go:138` `func Register(se *core.ServeEvent) error`;
  the `handlers` struct is at `web.go:250`; the root route is
  `web.go:182` `se.Router.GET("/", h.root)`. `web.Register(se)` is called from
  `main.go:88` inside `OnServe`. Register's signature is fixed by the PocketBase
  hook shape — thread the flag via `app.Store()` (set it in `main` before/at
  OnServe; read it in the handler), NOT by changing Register's parameters.
- **The Home handler already knows the no-model state** — `internal/web/home.go`
  renders "No model is ready yet." + a "Set up a model →" link + a 2s self-refresh
  poll (gap map cites home.go ~228–272, ~235). So the banner condition reuses an
  existing model-readiness check; do not invent a new one — find how `home.go`
  decides "no model ready" and reuse it.
- **Settings/models panel** — shown via `web.go:241` `se.Router.GET("/ui/show/{type}", h.uiShow)`
  (type=`settings`, `?section=models`); the view builder is
  `internal/feature/settingscards/settingsfocus_models.go` (`BuildModelsPanelView`).
- **Reusable components already in the design system** — `internal/ui/alert.go`
  (an Alert atom) and `internal/ui/nudgebanner.go` (a NudgeBanner). REUSE or extend
  one of these for the onboarding banner; do NOT hand-roll new markup.

## Follow the `ui-development` skill (this is UI work)
- Check the **storybook** first (`/storybook`, `internal/feature/storybook`) and
  reuse/extend an existing banner/alert component rather than hand-rolling.
- **Add or update the component's story** in the same change (storybook is the
  source of truth).
- Use typed `gomponents`, `h "maragu.dev/gomponents/html"`; user/model text via
  escaping `g.Text`. Match the Hearthwood/Basm design language + tokens.

## Commands you will need
| Purpose   | Command                                              | Expected |
|-----------|------------------------------------------------------|----------|
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0   |
| Vet       | `go vet ./...`                                        | exit 0   |
| Test pkg  | `go test ./internal/web/... ./internal/ui/... ./internal/feature/... -count=1` | PASS |
| Full test | `go test ./... -count=1`                             | all pass |
| gofmt     | `gofmt -l internal/web internal/ui internal/feature main.go` | nothing |

> Tests/commits must run with `TMPDIR=/home/alex/.cache/go-tmp` and `-count=1`
> (tmpfs `/tmp` OOMs the linker / box has limited RAM). Commit in the FOREGROUND
> (the pre-commit hook runs `make lint`). Do NOT run `make vulncheck` (RAM-OOM).

## Scope
**In scope**:
- `main.go` — capture `isFirstRun` at boot (stop discarding it) and stash it in `app.Store()` before/at OnServe.
- `internal/web/home.go` (or wherever `h.root` builds the Home view) — read the stashed first-run flag + the existing no-model state; render the onboarding banner when both hold.
- A banner component: extend/reuse `internal/ui/alert.go` or `internal/ui/nudgebanner.go` (add a small onboarding variant if needed) — plus its **storybook story**.
- The matching `_test.go` (web + the component/story).

**Out of scope** (do NOT touch):
- `installRuntime` / `downloadOfficialModel` / the Models page install logic — unchanged; the banner only LINKS to it.
- The chatbar self-heal poll (home.go ~235) — already works; do not rewire the "done" signal.
- `launch.IsFirstRun` itself — reuse it; do not change its semantics.
- Any gating of chat — the banner must NEVER block the owner from using the app.

## Git workflow
- Branch: `advisor/230-first-run-onboarding`
- Subject e.g. `feat(web): first-run onboarding banner routing the owner to model setup`
- Do NOT push.

## Steps

### Step 1: Stop discarding `IsFirstRun`; stash it at boot
In `main.go`, replace the `_ =` discard (`main.go:53`) with capturing the bool and
stashing it where the web handler can read it for the process lifetime — e.g.
`app.Store().Set("balaur_first_run", isFirstRun)` set before/within the `OnServe`
registration (use an atomic `app.Store()` set; this is a one-time boot flag).
**Verify**: `go build ./...` → exit 0; `grep -n "IsFirstRun" main.go` → no `_ =` discard remains.

### Step 2: Render the onboarding banner on Home when first-run + no model
In the Home view builder (`h.root` path), read the stashed first-run flag and the
EXISTING no-active-model state (reuse `home.go`'s current check — do not invent a
new one), and when both are true, render the onboarding banner (the reused/extended
design-system component) with a link to `/ui/show/settings?section=models`. The
banner is dismissible and never blocks the composer.
**Verify**: `go build ./internal/web/... && go vet ./internal/web/...` → exit 0; `gofmt -l` clean.

### Step 3: The banner component + its story (ui-development skill)
Reuse `ui.Alert`/`ui.NudgeBanner`, or add a small onboarding variant in
`internal/ui`. Add/update its **storybook story** (`internal/feature/storybook`)
rendering the onboarding state from a fixture. Match the design language.
**Verify**: `go test ./internal/ui/... ./internal/feature/storybook/... -count=1` → PASS.

### Step 4: Tests
- A `web` test: build the Home view with (first-run=true, no active model) → assert
  the onboarding banner + the models-setup link are present; with (first-run=false)
  OR (a model active) → assert the banner is ABSENT. (Model `llm.Client` is faked;
  set the first-run flag via the same `app.Store()` key.)
- The storybook story renders without panicking (existing storybook test pattern).
**Verify**: `go test ./internal/web/... ./internal/ui/... ./internal/feature/... -count=1` → PASS.

### Step 5: Full verification
- `gofmt -l internal/web internal/ui internal/feature main.go` → nothing
- `go vet ./...` → exit 0
- `go test ./... -count=1` → all pass

## Test plan
The decisive test is the **conditional render**: banner present iff (first-run AND
no active model), absent otherwise. That proves it guides a fresh owner without
nagging a set-up one. Plus a storybook story so the component is catalogued. No
browser automation in this plan; the reviewer/owner does the `/verify` browser pass.

## Done criteria — ALL must hold
- [ ] `main.go` no longer discards `IsFirstRun` (`grep -n "_ = launch.IsFirstRun" main.go` → no match); the flag is stashed for the handler.
- [ ] Home renders the onboarding banner ONLY when first-run AND no active model; it is dismissible and never blocks chat.
- [ ] The banner reuses/extends a design-system component (`internal/ui`) and has a storybook story.
- [ ] A web test asserts the banner is present in the first-run+no-model case and absent otherwise.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l` clean.
- [ ] `go test ./... -count=1` exits 0.
- [ ] `plans/README.md` status row updated.

## STOP conditions
- If reusing the first-run signal requires recomputing `IsFirstRun` AFTER the app
  has written data (where it would always return false), STOP — it must be captured
  at boot. Report if the boot ordering makes that awkward.
- If auto-focusing/opening the Models panel from `GET /` requires reshaping `h.root`
  or the Datastar render flow broadly, DROP that polish and ship the banner-only
  slice; report the decision.
- If the no-active-model state cannot be read in the Home builder without a new DB
  round-trip the owner-facing path can't afford, reuse exactly what `home.go`
  already computes and report any obstacle — do not add a second model probe.

## Maintenance notes
- This closes Row 9 of the gap map. Rows 7/8 (runtime + model presence) are now
  reachable via a guided step; Rows 11–13 (OS packaging, single-instance guard,
  arm64 checksums) remain open and are tracked in the gap-map spec.
- Reviewer: confirm the banner NEVER gates chat (informational only), that it
  disappears once a model is active (via the existing self-heal), and that the
  component + story follow the design system. A browser `/verify` pass is worth it
  before the owner relies on it.
