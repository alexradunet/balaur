# Plan 010: Security hardening bundle — hide api_key from REST, gate the dev-seed route, pin extension HTTP, cap Scoped reads

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- migrations/ internal/web/web.go internal/web/dev_seed.go internal/ext/vm.go internal/heads/scoped.go internal/store/llm_settings.go`
> On drift, re-verify the excerpts below.

## Status

- **Priority**: P2
- **Effort**: S (four independent small changes; land as one branch, four commits)
- **Risk**: LOW
- **Depends on**: none (plan 002 also adds a migration — coordinate numbering: this plan uses the next free timestamp AFTER plan 002's `1750700000`)
- **Category**: security
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/25

## Why this matters

Four defense-in-depth gaps, each small, none individually critical for a
loopback single-owner box, together cheap to close:

A. **`api_key` returned by the PocketBase REST API.** The field is a plain
   `TextField` (`migrations/1750200000_llm_model_config.go:23`) on
   `llm_providers` with owner rules — so `GET /api/collections/llm_providers/records`
   with the owner's `users` token returns stored API keys verbatim. Balaur's
   own Go path redacts (`internal/store/llm_settings.go:77` sets
   `cfg.APIKey = ""`), and the README promises "API keys are never rendered
   back into the UI or audit log" — the REST surface contradicts the spirit
   of that promise. PocketBase fields support `Hidden: true` (excluded from
   API responses; Go-side `record.GetString` is unaffected — verified in
   pocketbase v0.39.3 `core/field.go:72-73,91-95`).

B. **Dev-seed route registered unconditionally.**
   `POST /ui/dev/seed-recaps` exists in every production binary
   (`internal/web/web.go:112`); only the HANDLER checks
   `BALAUR_DEV_SEED=1` (`internal/web/dev_seed.go:48-50`, returning 404).
   Registration-time gating removes the route from production entirely —
   smaller surface, same dev experience.

C. **Extension HTTP follows redirects with the shared default client.**
   `internal/ext/vm.go:222` uses `http.DefaultClient` — up to 10 redirect
   hops, attacker-controllable targets, and extension-set headers
   (including `Authorization`) forwarded along the chain. Local-address
   reach is a DOCUMENTED design decision (vm.go:188-189 comment) and stays;
   redirect-following is incidental, not decided. A dedicated client with
   redirects disabled makes approved-extension behavior deterministic:
   what the owner read is what runs.

D. **`Scoped.Records` passes `limit` through unbounded.**
   `internal/heads/scoped.go:64-68`: `limit=0` returns EVERY record of a
   granted collection in one call. Today only owner-driven flows construct
   heads, so impact is theoretical — but the sub-head chat route (roadmap)
   will put model-driven calls behind this exact method. Cap it now, before
   it has callers that depend on unbounded reads.

## Current state

- A: `migrations/1750200000_llm_model_config.go:18-30` — `llm_providers`
  fields incl. `&core.TextField{Name: "api_key", Max: 10000}`.
  Go-side readers of the raw key: `internal/store/llm_settings.go`
  (`configForModel` at :225 reads it for client construction;
  `ListLLMModels` redacts at :77).
- B: `internal/web/web.go:112`:

```go
	se.Router.POST("/ui/dev/seed-recaps", h.seedRecaps)
```

  and `internal/web/dev_seed.go:24-27,48-50`:

```go
func (h *handlers) seedRecaps(e *core.RequestEvent) error {
	if !devSeedEnabled() {
		return e.NotFoundError("not found", nil)
	}
	...
func devSeedEnabled() bool {
	return os.Getenv("BALAUR_DEV_SEED") == "1"
}
```

- C: `internal/ext/vm.go:209-226` — request constructed with a 15s
  per-call context timeout (`httpTimeout`), body read through
  `io.LimitReader(resp.Body, maxHTTPBody)`; the only client is
  `http.DefaultClient.Do(req)`.
- D: `internal/heads/scoped.go:63-69`:

```go
func (s *Scoped) Records(collection, filter, sort string, limit int, params dbx.Params) ([]*core.Record, error) {
	if err := s.allow(collection, "read"); err != nil {
		return nil, err
	}
	return s.app.FindRecordsByFilter(collection, filter, sort, limit, 0, params)
}
```

- Conventions: migrations are append-only; tests use `storetest.NewApp`;
  heads tests live in `internal/heads/heads_test.go` (seeding pattern at
  :17-56); ext tests in `internal/ext/ext_test.go` (e.g.
  `TestHTTPBindingWorksInsideHandlers` at :223 — your model for C's test).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Gates | `gofmt -l .` / `go vet ./...` / `go test ./...` | clean / 0 / ok |
| Focused | `go test ./internal/ext/ ./internal/heads/ ./internal/store/ ./migrations/ -v` | new tests pass |
| Build + fresh box | `CGO_ENABLED=0 go build -o /tmp/balaur-test . && /tmp/balaur-test --dir $(mktemp -d) task list` | exit 0 |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `migrations/<next-free-timestamp>_hide_api_key.go` (create — use
  `1750710000` unless taken)
- `internal/web/web.go` (move ONE route registration behind the env check)
- `internal/ext/vm.go` (dedicated HTTP client)
- `internal/heads/scoped.go` (limit cap)
- Tests: `internal/ext/ext_test.go`, `internal/heads/heads_test.go`,
  `migrations/` test file

**Out of scope** (do NOT touch):
- `internal/web/dev_seed.go` — keep the handler-side check as
  belt-and-braces.
- Blocking local addresses in `balaur.http` — explicitly decided otherwise
  in the code comment; do not "fix" a decision.
- A `users`-collection single-record guard — REJECTED in the audit:
  creating users requires the superuser dashboard already.
- `README.md` — the "never rendered back" sentence becomes TRUE with (A);
  no text change needed.

## Git workflow

- Branch: `advisor/010-hardening-bundle`; one commit per item (A–D), styles:
  `fix(migrations): hide llm_providers.api_key from REST responses`,
  `fix(web): register dev-seed route only when BALAUR_DEV_SEED=1`,
  `fix(ext): dedicated no-redirect HTTP client for balaur.http`,
  `fix(heads): cap Scoped.Records result size`. No push/PR unless instructed.

## Steps

### Step 1 (A): Hide api_key via migration

New migration (pattern: `migrations/1750600000_head_avatar.go`): load
`llm_providers`, find the `api_key` field
(`col.Fields.GetByName("api_key")`), call `SetHidden(true)` on it (the
`core.Field` interface exposes `GetHidden/SetHidden` — pocketbase
`core/field.go:91-95`), save. Down: `SetHidden(false)`, save.
Add a test (same external-package pattern as plan 002's
`migrations/..._test.go`): boot `storetest.NewApp`, fetch the collection,
assert `Fields.GetByName("api_key").GetHidden() == true`. Also assert the
Go read path still works: `store.SaveOpenAIModel(...)` then
`store.ActiveLLMConfig`/`configForModel` returns a non-empty key (read
`llm_settings_test.go` for the existing seeding pattern — extend it there
if more natural).

**Verify**: `go test ./migrations/... ./internal/store/... -v` → new
assertions pass.

### Step 2 (B): Gate the route registration

In `web.go`, wrap line 112:

```go
	if devSeedEnabled() {
		se.Router.POST("/ui/dev/seed-recaps", h.seedRecaps)
	}
```

(`devSeedEnabled` is in the same package — dev_seed.go:48.)

**Verify**: `go build ./internal/web/` → 0. If plan 004's harness exists,
add a scenario: POST `/ui/dev/seed-recaps` without the env → 404.

### Step 3 (C): Dedicated extension HTTP client

In `vm.go`, add a package-level:

```go
// extHTTPClient never follows redirects: an approved extension's reviewed
// code is exactly what runs — a redirect chain must be followed explicitly
// by the handler if it wants to. Local addresses stay deliberately
// reachable (see httpBinding's comment).
var extHTTPClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}
```

and replace `http.DefaultClient.Do(req)` with `extHTTPClient.Do(req)`.
Add a test in `ext_test.go` modeled on
`TestHTTPBindingWorksInsideHandlers`: an `httptest.Server` whose `/a`
301-redirects to `/b`; the extension fetches `/a`; assert the returned
`status` is `301` (not `/b`'s body).

**Verify**: `go test ./internal/ext/ -run TestHTTP -v` → both HTTP tests pass.

### Step 4 (D): Cap Scoped.Records

In `scoped.go`:

```go
const maxScopedRecords = 500

func (s *Scoped) Records(...) ... {
	if err := s.allow(collection, "read"); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > maxScopedRecords {
		limit = maxScopedRecords
	}
	return s.app.FindRecordsByFilter(collection, filter, sort, limit, 0, params)
}
```

Add a test in `heads_test.go` (reuse `newApp`/`seedMemory` and the grant
seeding from `TestScopedDeniesUngrantedAccess`): seed 3 memories, call
`Records("memories", "", "", 0, nil)` with a read grant → 3 rows returned
(cap applies, zero no longer means unbounded-rejected); the cap itself is
asserted by calling with `limit=10_000` and checking the function does not
error (behavioral cap can't be observed with 3 rows — the assertion is the
constant's presence; keep the test simple and grep-verify the constant).

**Verify**: `go test ./internal/heads/ -v` → all pass.

### Step 5: Full gates + fresh box

**Verify**: `gofmt -l .` empty; `go vet ./...` 0; `go test ./...` ok;
fresh-box smoke (`task list`) exit 0 — proves the new migration applies.

## Test plan

- A: migration test (Hidden flag) + store round-trip (key still readable
  Go-side). REST-level proof is optional: if plan 004's harness exists, a
  scenario hitting `/api/collections/llm_providers/records` with a seeded
  owner token asserting the response omits `api_key` is the gold assertion
  — include it if token seeding is straightforward (read pocketbase
  `tests` helpers for auth record token generation; if it costs more than
  ~30 lines, skip and note).
- B: 404 scenario (with harness) or build-only.
- C: redirect test (Step 3).
- D: grant test (Step 4).

## Done criteria

- [ ] `grep -rn "SetHidden(true)" migrations/` → 1 match; migration test passes
- [ ] `grep -n "if devSeedEnabled()" internal/web/web.go` → 1 match
- [ ] `grep -n "extHTTPClient" internal/ext/vm.go` → ≥ 2 matches; redirect test passes
- [ ] `grep -n "maxScopedRecords" internal/heads/scoped.go` → ≥ 2 matches
- [ ] `go test ./...` exit 0; `gofmt -l .` empty; fresh-box smoke exit 0
- [ ] Diff confined to in-scope files (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- `core.Field` lacks `SetHidden` in the resolved pocketbase version — report
  the actual field API (`grep -n "Hidden" $(go env GOMODCACHE)/github.com/pocketbase/pocketbase@v0.39.3/core/field.go`).
- Hiding `api_key` breaks `store.SaveOpenAIModel` round-trips (PocketBase
  validation rejecting hidden-field writes from Go) — report; that would
  force the alternative (a REST `onRecordsListRequest` hook), which needs
  advisor re-scoping.
- A migration timestamp collision with plan 002 — renumber yours upward,
  never reuse.

## Maintenance notes

- (C) means extensions that legitimately need redirects must follow them
  manually (read `status`/`Location` from the response map). If a real
  extension hits this, the right evolution is an explicit
  `{follow_redirects: true}` option on `balaur.http` — opt-in, audited.
- (D)'s cap value (500) is arbitrary-but-sane; when sub-head chat ships,
  revisit against real usage and consider pagination on `Scoped`.
- (A) protects the REST read path only; `pb_data/` on disk remains
  plaintext SQLite — the README's "treat pb_data as secret" stance is
  unchanged (encrypted export stays roadmap).
