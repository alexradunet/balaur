# Plan 120: Make `SetOwnerSetting` safe under concurrent writes to the same key

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat ce2ba72..HEAD -- internal/store/owner_settings.go internal/store/owner_settings_test.go migrations/1750300000_owner_settings.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Why this matters

`store.SetOwnerSetting` is a classic check-then-act upsert: it looks up the row
for a key, and if absent creates a new record, then saves. The `owner_settings`
collection has a UNIQUE index on `key` (`migrations/1750300000_owner_settings.go`).
When two requests write the **same** key concurrently — e.g. two panel-state
saves, or a panel save racing a profile save on the same key — both can miss the
lookup, both create a new record, and the second `Save` fails on the UNIQUE
constraint. Today that surfaces as an HTTP 500 (profile handlers) or a silently
dropped write (panel handlers `_ =` the error). This is the only genuine
concurrency bug the cleanup audit found. The fix makes the upsert converge under
contention without adding global state.

## Current state

`internal/store/owner_settings.go:28-41`:
```go
// SetOwnerSetting upserts a key/value pair in owner_settings.
func SetOwnerSetting(app core.App, key, value string) error {
	col, err := app.FindCollectionByNameOrId("owner_settings")
	if err != nil {
		return err
	}
	rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("key", key)
	}
	rec.Set("value", value)
	return app.Save(rec)
}
```

The UNIQUE index, `migrations/1750300000_owner_settings.go` (around line 26):
`col.AddIndex("idx_owner_settings_key", true, "key", "")` (the `true` = unique).

Concurrent callers (all reachable from HTTP handlers, inherently concurrent):
- `internal/web/profile.go` — `saveName`, `setSoulAvatarFromProfile`,
  `setBalaurAvatarPref` (return 500 on error).
- `internal/web/panel.go` — `panelClose` (`:156`), `uiPanelCollapse` (`:169`),
  `uiPanelWidth` (`:186`) (swallow the error with `_ =`).

Why one retry is provably sufficient for the same-key race: the only failure is
**insert-vs-insert** (two `NewRecord`s for a missing key). After the first
`Save` wins, the row exists; a retry's `FindFirstRecordByData` then finds it and
performs an UPDATE on that row (same id → no UNIQUE conflict). Different keys map
to different rows and never collide. So a single retry closes the window.

Repo conventions: errors wrapped with `fmt.Errorf("…: %w", err)`, return early,
no panics (`AGENTS.md`). No global mutable state — pass `core.App` explicitly;
do NOT introduce a package-level `sync.Mutex`. The store test harness is
`storetest.NewApp(t)` (see `internal/store/owner_settings_test.go` and
`internal/storetest/storetest.go`).

## Commands you will need

| Purpose   | Command                                       | Expected on success |
|-----------|-----------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                | exit 0              |
| Vet       | `go vet ./...`                                | exit 0              |
| Format    | `gofmt -l internal/store/owner_settings.go internal/store/owner_settings_test.go` | empty |
| Test (store)| `go test ./internal/store/...`              | `ok`                |
| Test (race) | `CGO_ENABLED=1 go test -race -run TestSetOwnerSettingConcurrent ./internal/store/` | `ok` |
| Full tests | `go test ./...`                              | all `ok`            |
| Diff hygiene | `git diff --check`                         | no output           |

(The `-race` run needs CGO; the normal suite and build stay CGO-free. In a
TLS-intercepting sandbox, Go commands may need a GOPROXY shim; GOSUMDB stays on.)

## Scope

**In scope**:
- `internal/store/owner_settings.go` (the `SetOwnerSetting` function only)
- `internal/store/owner_settings_test.go` (add the new test)
- Optionally `internal/web/panel.go` (Step 3 — log the swallowed errors)

**Out of scope** (do NOT touch):
- `migrations/1750300000_owner_settings.go` — the UNIQUE index is correct and is
  what makes the fix necessary; never weaken or drop it (frozen migration).
- `GetOwnerSetting` and the avatar rosters in the same file — unaffected.
- The profile handlers' 500-on-error behavior — correct; leave it.
- Any change that adds a package-level mutex or other global state.

## Git workflow

- Land on `main`; if dispatched, base off `origin/main`. Conventional-commit
  subject, e.g. `fix(store): converge SetOwnerSetting upsert under concurrent same-key writes`. Commit/push only when the operator instructs.

## Steps

### Step 1: Make the upsert retry once on a conflicting insert

Rewrite `SetOwnerSetting` so the find+save runs as a closure that is retried once
if the first attempt fails (the retry's find sees the row the winner created):

```go
// SetOwnerSetting upserts a key/value pair in owner_settings. The collection
// has a UNIQUE index on key, so two concurrent writers that both miss the
// initial lookup would otherwise collide on insert; on a failed save we retry
// once, by which point the row exists and the retry updates it.
func SetOwnerSetting(app core.App, key, value string) error {
	col, err := app.FindCollectionByNameOrId("owner_settings")
	if err != nil {
		return err
	}
	save := func() error {
		rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
		if err != nil {
			rec = core.NewRecord(col)
			rec.Set("key", key)
		}
		rec.Set("value", value)
		return app.Save(rec)
	}
	if err := save(); err != nil {
		// A concurrent insert may have created the row between our lookup and
		// save (UNIQUE on key). Retry once: the row now exists, so we update it.
		if err := save(); err != nil {
			return fmt.Errorf("set owner setting %q: %w", key, err)
		}
	}
	return nil
}
```
Add `"fmt"` to the imports if not already present (the current file imports only
`core`; add `fmt`).

**Verify**: `go build ./internal/store/` exit 0; `gofmt -l internal/store/owner_settings.go` empty.

### Step 2: Add a concurrency regression test

Append to `internal/store/owner_settings_test.go` a test that hammers the same
key from many goroutines and asserts no error and exactly one row. Model the app
setup on the existing `TestLegacySoulAvatarAliases` (`storetest.NewApp(t)`).

```go
func TestSetOwnerSettingConcurrent(t *testing.T) {
	app := storetest.NewApp(t)
	const n = 24
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- SetOwnerSetting(app, "panel_active", fmt.Sprintf("/ui/show/quests?n=%d", i))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent SetOwnerSetting: %v", err)
		}
	}
	// Exactly one row for the key — the UNIQUE index plus our converge logic.
	recs, err := app.FindRecordsByFilter("owner_settings", "key = 'panel_active'", "", 0, 0)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("want exactly one owner_settings row for the key, got %d", len(recs))
	}
}
```
Add `"sync"` and `"fmt"` to the test file's imports as needed.

**Verify**:
- `go test ./internal/store/` → `ok`.
- `CGO_ENABLED=1 go test -race -run TestSetOwnerSettingConcurrent ./internal/store/` → `ok` (no data race, no constraint error).

> NOTE on the test as a regression guard: depending on PocketBase's internal
> write serialization, this test may not always fail against the OLD code on
> every machine. If, before applying Step 1, you run it and it PASSES against
> the old `SetOwnerSetting`, that is acceptable — keep the test (it still guards
> the converge behavior and the single-row invariant) and proceed. Do NOT
> contort the test to force a failure.

### Step 3 (optional, low-priority): stop silently dropping panel-save errors

In `internal/web/panel.go`, the three panel handlers `panelClose`,
`uiPanelCollapse`, `uiPanelWidth` discard the error with `_ =`. Keep them
non-fatal (panel state is cosmetic and the client already applied it
optimistically), but log on failure instead of dropping silently, e.g.:
```go
if err := store.SetOwnerSetting(h.app, panelCollapsedKey, on); err != nil {
	h.app.Logger().Warn("persisting panel state failed", "key", panelCollapsedKey, "err", err)
}
```
Apply the same pattern to all three. Do not change their HTTP responses.

**Verify**: `go build ./internal/web/` exit 0; `go test ./internal/web/...` `ok`.

(If you skip Step 3, say so in your status update — the root-cause fix in Step 1
is what resolves the bug; Step 3 is observability polish.)

### Step 4: Full build, vet, test

Run build, `go vet ./...`, and full `go test ./...`.

**Verify**: all green.

## Test plan

- New test: `TestSetOwnerSettingConcurrent` in `internal/store/owner_settings_test.go`
  — N concurrent same-key writes converge with no error and exactly one row.
- Pattern to follow: `TestLegacySoulAvatarAliases` (same file) for app setup.
- Verification: `go test ./internal/store/` and the `-race` run both pass; full
  suite stays green.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `SetOwnerSetting` retries the save once on failure (`grep -c "save()" internal/store/owner_settings.go` ≥ 2)
- [ ] `TestSetOwnerSettingConcurrent` exists and passes
- [ ] `CGO_ENABLED=1 go test -race -run TestSetOwnerSettingConcurrent ./internal/store/` → `ok`
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` all `ok`
- [ ] `migrations/1750300000_owner_settings.go` is unchanged (`git diff --stat` shows it untouched)
- [ ] `git diff --check` → no output
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back if:

- The `SetOwnerSetting` excerpt doesn't match the live code (drift).
- The UNIQUE index in the migration is not present as described (then the race
  analysis is wrong; report before changing anything).
- The `-race` test reports a data race INSIDE PocketBase rather than in this code
  (means the fix needs a different approach — report, don't add a mutex blindly).
- Any verification fails twice after a reasonable fix attempt.

## Maintenance notes

- The single retry is provably enough for same-key insert races (after one
  conflict the row exists and subsequent saves are updates). If a future change
  makes `owner_settings` rows deletable concurrently with writes, revisit (a
  delete between the conflict and the retry could reopen the window).
- A reviewer should confirm no package-level mutex/global was introduced and the
  UNIQUE migration is untouched.
- The deeper `internal/turn` error-path coverage gap and audit-log redaction are
  tracked separately (not part of this plan).
