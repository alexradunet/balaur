# Plan 233: Messenger-token settings field — let the owner set/rotate `messenger_token` from the Settings page (not the admin engine room)

> **Follow-up (b) of plan 231.** The messenger gateway (`POST /api/messenger/turn`)
> is consent-gated on `owner_settings.messenger_token`, but today the owner can
> only set that key via the PocketBase admin engine room (`/_/`). This adds an
> owner-facing field on the Settings page so enabling the feature is a first-class
> product action, matching how every other owner setting is edited.
>
> **Drift check (run first)**:
> `git diff --stat <BASE>..HEAD -- internal/feature/settingscards/settingsfocus_capabilities.go internal/feature/settingscards/settingsfocus_profile.go internal/web/web.go`
> (BASE = the commit this plan is dispatched against; the reviewer sets it.) On a
> mismatch with the excerpts below, compare before editing; on real drift, STOP.

## Status
- **Priority**: P4 (polish on a power-user feature — the messenger endpoint needs an owner-run bridge to matter)
- **Effort**: S–M
- **Risk**: LOW–MEDIUM (handles a secret in the owner-only UI; must not log it)
- **Depends on**: plan 231 (the endpoint + `messenger_token` key) — merged
- **Category**: direction / usability
- **Planned at**: dispatched against current `main` after 231/232

## Why this matters

Plan 231 made the messenger endpoint **fail-closed on `messenger_token`** — good
security, but the only way to SET the token today is the PocketBase admin dashboard
(`/_/`), which AGENTS.md calls "the superuser engine room, never the product
surface." So enabling a shipped feature currently requires leaving the product.
This closes that: a Settings field to set/rotate the token, so enabling the
messenger is a normal, consent-explicit owner action.

## The owner-facing settings-form pattern to mirror (read first)

`internal/feature/settingscards/settingsfocus_profile.go` is the exemplar — an
owner field that saves via a Datastar form POST:
```go
// settingsfocus_profile.go:95-99 — the "profile-name-form"
h.Class("profile-name-form"),
data.On("submit", "@post('/ui/profile/name', {contentType:'form'})", data.ModifierPrevent),
h.Input( ... h.Name("name") ... ),
```
Find its **POST handler** (grep `"/ui/profile/name"` in `internal/web/`) — it reads
the form value and persists it (via `store.SetOwnerSetting` or a profile write).
Mirror that handler shape for the token.

`internal/feature/settingscards/settingsfocus_capabilities.go` is the
**security/capabilities section** (`os_access`, `recap`, `nudge`, `briefing` — see
its `:61` loop). The messenger token is a security-adjacent enable, so this section
is its natural home (a text field + save, alongside the os_access control).

Owner settings are read/written with `store.GetOwnerSetting` / `store.SetOwnerSetting`.

## Design

Add a "Messenger gateway" control to the capabilities/security settings section:
- A field to set/rotate `owner_settings.messenger_token`, saved via a new POST
  handler (mirror `/ui/profile/name`): e.g. `POST /ui/settings/messenger-token`
  reads the form value and `store.SetOwnerSetting(app, "messenger_token", value)`.
- **Enable/disable is implicit in the token** (matches 231's fail-closed model): a
  non-empty token = enabled; clearing the field (empty value) = disabled. Make that
  explicit in the field's help text ("Set a token to enable the local messenger
  endpoint; clear it to disable.").
- **Secret handling (owner-only UI, loopback):** the Settings page is server-rendered
  and stays on-box, so it MAY show the current token so the owner can copy it into
  their bridge config. Acceptable UX options (executor's call, follow ui-development):
  (i) show the current token in a readonly/copyable field + a "rotate" action that
  generates a new random token, or (ii) a masked "set new token" write-only field +
  a "generate" button that reveals the new value once. Either is fine; do NOT log
  the token anywhere and do NOT render it outside this owner settings view.
- **A "generate" affordance is nice-to-have**, not required — a strong random token
  (e.g. base64 of crypto/rand bytes) beats an owner typing a weak one. If cheap,
  add it; if it complicates the slice, a plain text field the owner fills is fine.

## Follow the `ui-development` skill
- Check the storybook; reuse existing settings-card atoms/inputs (the profile/nudges
  forms use `h.Input` + the `data.On("submit", "@post(...)")` idiom — mirror it).
- Add/update the capabilities settings story so the new control is catalogued.
- Typed gomponents; user text via escaping `g.Text`.

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test | `go test ./internal/web/... ./internal/feature/settingscards/... -count=1` | PASS |
| Full | `go test ./... -count=1` | all pass |
| gofmt | `gofmt -l internal/web internal/feature/settingscards` | nothing |

> Prefix with `TMPDIR=/home/alex/.cache/go-tmp`, `-count=1`. Commit FOREGROUND. No `make vulncheck`.

## Scope
**In scope**:
- `internal/feature/settingscards/settingsfocus_capabilities.go` — add the token control (+ its story).
- `internal/web/*.go` — a new POST handler that writes `messenger_token` via `store.SetOwnerSetting`, registered in `web.Register` (mirror the `/ui/profile/name` handler + registration).
- The matching `_test.go` (settingscards render + the web save handler).

**Out of scope** (do NOT touch):
- `internal/web/messenger.go` — the endpoint is done; this only sets the key it reads.
- The four security constraints of 231 — unchanged.
- Any other settings section beyond adding this one control.

## Git workflow
- Branch: `advisor/233-messenger-token-settings-ui`
- Subject e.g. `feat(settings): owner field to set/rotate the messenger gateway token`
- Do NOT push.

## Steps

### Step 1: The save handler
Add a POST handler (mirror `/ui/profile/name`) that reads the form value and calls
`store.SetOwnerSetting(h.app, "messenger_token", value)`. Register it in
`web.Register`. Never log the value. (guardLocalUI already guards `/ui/*` — put the
handler under `/ui/…`, NOT `/api/…`, so it inherits the UI host+origin guard.)
**Verify**: `go build ./internal/web/... && go vet ./internal/web/...` → exit 0.

### Step 2: The settings control
Add the "Messenger gateway" field to `settingsfocus_capabilities.go` (mirror the
profile-name form), reading the current token via `store.GetOwnerSetting` to
pre-fill/show state, posting to the Step-1 handler. Add its storybook story.
**Verify**: `go test ./internal/feature/settingscards/... -count=1` → PASS.

### Step 3: Tests
- **Save handler**: POST a token → `store.GetOwnerSetting(app,"messenger_token","")`
  returns it; POST empty → cleared (disables the endpoint). Assert the value is NOT
  in any log.
- **Render**: the capabilities settings view renders the messenger control (present
  in the HTML); if it shows the current token, assert it renders only in this
  owner view.
- (Optional) enabling round-trip: set token via the handler, then a messenger POST
  with that token succeeds (only if cheap to wire against the existing messenger test helpers; otherwise the messenger tests already cover the endpoint).
**Verify**: `go test ./internal/web/... ./internal/feature/settingscards/... -count=1` → PASS.

### Step 4: Full verification
- `gofmt -l internal/web internal/feature/settingscards` → nothing
- `go vet ./...` → exit 0
- `go test ./... -count=1` → all pass

## Done criteria — ALL must hold
- [ ] A Settings (capabilities/security) control sets/rotates `messenger_token`; a new POST handler persists it via `store.SetOwnerSetting`.
- [ ] Handler lives under `/ui/…` (inherits guardLocalUI), NOT `/api/…`.
- [ ] Empty value clears the token (disables the endpoint), matching 231's fail-closed model; help text says so.
- [ ] The token is never logged; it renders only in the owner settings view.
- [ ] Storybook story added for the new control.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l` clean; `go test ./... -count=1` green.
- [ ] `plans/README.md` status row updated.

## STOP conditions
- If the settings section has no clean place for a text field + save (only toggles),
  report the obstacle rather than building a whole new settings section — a minimal
  new "Messenger" sub-section is acceptable, but flag it.
- If persisting the token would require anything beyond `store.SetOwnerSetting`
  (e.g. a schema change), STOP and report — 231 already reads the plain owner-setting.

## Maintenance notes
- This makes 231 usable without the admin engine room. A real chat-app bridge is
  still the owner's own process (out of scope by design).
- Reviewer: confirm the token is never logged and renders only in the owner-only
  settings view; confirm empty-clears-disables matches `messenger.go`'s
  `tok == ""` fail-closed check.
