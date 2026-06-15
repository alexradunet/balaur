# Plan 065: kill the Ollama pull cancel-vs-restart race and cover the whole failure surface

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 1f8f55e..HEAD -- internal/ollama/manager.go internal/ollama/manager_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1 (data-race / user-visible state corruption on an ordinary owner action)
- **Effort**: S–M
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug + tests
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

`internal/ollama/manager.go` runs Balaur's single background `ollama pull` as a
state machine guarded by one mutex. It has a lost-update race: `Cancel()`
optimistically clears `progress.Active` **without waiting** for the in-flight
`runPull` goroutine to exit (Ollama's streaming pull cancel is not
instantaneous). The very next owner action — `Pull("b")` — is then accepted
(its only guard is `if m.progress.Active`), reassigns `m.cancel`/`m.progress`,
and launches a **second** `runPull`. When the first, cancelled goroutine
finally returns, its completion block unconditionally writes
`m.progress.Active = false` and `m.progress.Err = "pull cancelled"` — clobbering
the second pull's live snapshot. The UI then shows the running pull as
cancelled while its progress callbacks keep writing `BytesDone`/`BytesTotal`:
transient, self-healing, but visibly wrong. Trigger is ordinary — the web
gateway exposes `modelPullCancel` (`h.ollama.Cancel`) and `modelPull`
(`h.ollama.Pull`) as two distinct owner endpoints, with `modelPullProgress`
polling `Snapshot()` (`internal/web/models.go:246,275,287`).

Bounded-impact fact (state it in the fix comment): the cancelled goroutine
takes the `err != nil` branch, so `cb` stays `nil` and `onDone` is **not**
fired — there is no false model activation, only the wrong snapshot.

The fix is a generation token so a superseded goroutine becomes a no-op. While
here, we fold in the downgraded perf item (PERFORM-03): the control calls
(List/Heartbeat/Delete and `cachedTags`' cold fetch) share an `http.Client`
with no timeout, so a hung daemon can wedge a board-render request forever. We
bound those with a short per-call `context` — **without** touching the Pull
path, which must be able to run for minutes. This also closes test-coverage
gaps TESTCO-02/03/04: there is no test for cancel-mid-pull, pull-error,
the race itself, delete-invalidates-cache, or the cachedTags error path.

## Current state

`internal/ollama/manager.go` (as of `1f8f55e`), package `ollama`. Module path is
`github.com/alexradunet/balaur`. The `Manager` struct and the single-slot pull
machine:

```go
// internal/ollama/manager.go:39-50
type Manager struct {
	mu sync.Mutex

	// single-slot pull
	cancel   context.CancelFunc
	progress PullSnapshot
	onDone   func(tag string)

	// tags cache (board-render hot path)
	tagsCache   []Model
	tagsCacheAt time.Time
}
```

```go
// internal/ollama/manager.go:55-57 — every control + pull call goes through this.
func (m *Manager) apiClient() *api.Client {
	return api.NewClient(&url.URL{Scheme: "http", Host: Host()}, &http.Client{})
}
```

`Pull` guards ONLY on `progress.Active` (the race's open door):

```go
// internal/ollama/manager.go:75-87
func (m *Manager) Pull(tag string, onDone func(tag string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress.Active {
		return fmt.Errorf("a model pull is already in progress")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.progress = PullSnapshot{Active: true, URL: tag, Dest: tag}
	m.onDone = onDone
	go m.runPull(ctx, tag)
	return nil
}
```

`runPull` mutates shared state with NO token identifying which pull it owns —
both the progress callback (lines 94-97) and the completion block (101-117)
write `m.progress` unconditionally:

```go
// internal/ollama/manager.go:89-121
func (m *Manager) runPull(ctx context.Context, tag string) {
	err := m.apiClient().Pull(ctx, &api.PullRequest{Model: tag}, func(p api.ProgressResponse) error {
		if p.Total <= 0 {
			return nil // status-only line (e.g. "success"); keep the last byte counts
		}
		m.mu.Lock()
		m.progress.BytesDone = p.Completed
		m.progress.BytesTotal = p.Total
		m.mu.Unlock()
		return nil
	})
	var cb func(string)
	m.mu.Lock()
	m.progress.Active = false
	if err != nil {
		// A cancelled pull surfaces a raw context/transport error; keep the
		// friendly "pull cancelled" message Cancel() set instead of clobbering it.
		if ctx.Err() != nil {
			m.progress.Err = "pull cancelled"
		} else {
			m.progress.Err = err.Error()
		}
	} else {
		m.progress.Done = true
		m.progress.Err = ""
		m.tagsCacheAt = time.Time{} // a new model exists; force a refetch
		cb = m.onDone
	}
	m.mu.Unlock()
	if cb != nil {
		cb(tag)
	}
}
```

`Cancel` clears `Active` optimistically and does NOT wait for `runPull`:

```go
// internal/ollama/manager.go:123-135
func (m *Manager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.progress.Active {
		m.progress.Active = false
		m.progress.Err = "pull cancelled"
	}
}
```

The control + cache calls that should get a bounded timeout (all currently use a
plain `context.Background()` with the no-timeout `http.Client`):

```go
// internal/ollama/manager.go:61-71  (List path)
func (m *Manager) fetchModels(ctx context.Context) ([]Model, error) {
	resp, err := m.apiClient().List(ctx)
	...
}

// internal/ollama/manager.go:147-149  (Reachable)
func (m *Manager) Reachable(ctx context.Context) bool {
	return m.apiClient().Heartbeat(ctx) == nil
}

// internal/ollama/manager.go:156-175  (cachedTags cold fetch)
func (m *Manager) cachedTags() ([]Model, error) {
	... // TTL fast-path elided
	models, err := m.fetchModels(context.Background())  // <- cold fetch, no timeout
	...
}

// internal/ollama/manager.go:190-196  (Delete)
func (m *Manager) Delete(tag string) error {
	if err := m.apiClient().Delete(context.Background(), &api.DeleteRequest{Model: tag}); err != nil {
		return err
	}
	m.invalidateTags()
	return nil
}
```

`PullSnapshot` / `Model` field names (lines 14-33) are consumed by web
templates via `Snapshot()` and `gguf_progress`; **do not rename them here**
(plan 070 renames the gguf-derived names separately).

### Conventions that apply

- Errors are values: `fmt.Errorf("doing x: %w", err)`, return early, no panics
  in library code (this is library code — `Manager`).
- `gofmt` is law; `go vet ./...` clean. A `PostToolUse` hook reformats on edit,
  but you must still confirm `gofmt -l .` prints nothing.
- Tests: standard `testing`, table-driven where it helps, **no assertion
  frameworks**, and they never hit a real Ollama daemon. The ollama tests point
  at an `httptest.Server` via `t.Setenv("BALAUR_OLLAMA_HOST", host)` where `host`
  is `host:port` with **no scheme** — use the existing
  `hostFromURL(u) = strings.TrimPrefix(u, "http://")` helper
  (`internal/ollama/manager_test.go:14`).
- Existing tests to model after, in `internal/ollama/manager_test.go`:
  `TestPullSnapshotProgress` (newline-delimited JSON pull stream + `onDone`
  channel + `Snapshot()` assert), `TestPullRejectsSecondConcurrent` (a `block`
  channel server that hangs until released), `TestReachable`,
  `TestCachedTagsHitsServerOnceWithinTTL` (`atomic.Int32` server-hit counter,
  `invalidateTags`). Read all four before writing new tests.
- The `-race` detector requires CGO; the repo's `make race` is
  `CGO_ENABLED=1 go test -race ./...`. Run `CGO_ENABLED=1 go test -race ./internal/ollama`
  for this package specifically.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Drift | `git diff --stat 1f8f55e..HEAD -- internal/ollama/manager.go internal/ollama/manager_test.go` | empty |
| Vet | `go vet ./internal/ollama/` | exit 0 |
| Package tests | `go test ./internal/ollama/` | all pass (incl. 5 new) |
| **Race tests** | `CGO_ENABLED=1 go test -race ./internal/ollama/` | exit 0, no `DATA RACE` |
| All tests | `go test ./...` | all pass |
| CGO-free build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| gofmt | `gofmt -l .` | prints nothing |
| Whitespace | `git diff --check` | no output |

## Scope

**In scope** (modify only these):
- `internal/ollama/manager.go` — add the `gen` token; make `runPull` a no-op when
  superseded; bound the control/cache calls with a short timeout.
- `internal/ollama/manager_test.go` — add the five tests below.

**Out of scope** (do NOT touch, even though related):
- `internal/web/models.go` — handlers (`modelPull`/`modelPullCancel`/
  `modelPullProgress`) stay exactly as-is; the fix is entirely inside the
  manager, callers are unchanged.
- `internal/ollama/client.go`, `presets.go`, `e2e_test.go`, `presets_test.go`,
  `client_test.go`, and every other package.
- `PullSnapshot` and `Model` **field names** — plan 070 renames the gguf-derived
  names. You may not rename a field here; you may only add a non-exported field
  to `Manager`.
- The Pull path's timeout: do **NOT** put a `Timeout` on the `http.Client` that
  `apiClient()` returns and `runPull` shares — a real pull can take minutes; a
  client-level timeout would abort long pulls. Only the control calls get a
  bounded `context`.

## Git workflow

- Branch: `improve/065-ollama-manager-hardening`
- One commit (or one per logical unit: fix, then tests); conventional-commit
  style, e.g. `fix(ollama): end pull cancel-vs-restart race with a generation token`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add the generation token to `Manager`

Add a non-exported counter to the struct (do not rename anything):

```go
type Manager struct {
	mu sync.Mutex

	// single-slot pull
	gen      int // bumped on every Pull/Cancel; runPull writes only when it still owns m.gen
	cancel   context.CancelFunc
	progress PullSnapshot
	onDone   func(tag string)

	// tags cache (board-render hot path)
	tagsCache   []Model
	tagsCacheAt time.Time
}
```

**Verify**: `go build ./internal/ollama/` → exit 0.

### Step 2: Bump `gen` in `Pull` and capture it for the goroutine

In `Pull`, increment `m.gen` under the held lock and pass the captured value to
`runPull` so the goroutine knows which pull it owns:

```go
func (m *Manager) Pull(tag string, onDone func(tag string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress.Active {
		return fmt.Errorf("a model pull is already in progress")
	}
	m.gen++
	gen := m.gen
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.progress = PullSnapshot{Active: true, URL: tag, Dest: tag}
	m.onDone = onDone
	go m.runPull(ctx, gen, tag)
	return nil
}
```

**Verify**: nothing yet (compile fails until Step 4 updates `runPull`); proceed.

### Step 3: Bump `gen` in `Cancel`

So that an in-flight goroutine is immediately superseded the moment Cancel runs,
even before a new Pull starts:

```go
func (m *Manager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.progress.Active {
		m.progress.Active = false
		m.progress.Err = "pull cancelled"
	}
	m.gen++ // supersede the goroutine we just cancelled so its late completion is a no-op
}
```

Keep the existing mutex discipline; `Cancel` already holds `m.mu` for its whole
body.

**Verify**: nothing yet; proceed to Step 4.

### Step 4: Make `runPull` a no-op when superseded

`runPull` takes the captured `gen` and guards **both** write sites with
`m.gen == gen`. A superseded goroutine returns without touching shared state.

```go
func (m *Manager) runPull(ctx context.Context, gen int, tag string) {
	err := m.apiClient().Pull(ctx, &api.PullRequest{Model: tag}, func(p api.ProgressResponse) error {
		if p.Total <= 0 {
			return nil // status-only line (e.g. "success"); keep the last byte counts
		}
		m.mu.Lock()
		if m.gen == gen { // only the owning pull may write progress
			m.progress.BytesDone = p.Completed
			m.progress.BytesTotal = p.Total
		}
		m.mu.Unlock()
		return nil
	})
	var cb func(string)
	m.mu.Lock()
	// If a Cancel or a newer Pull bumped m.gen, this goroutine has been
	// superseded: its late completion must not clobber the live snapshot.
	// (A cancelled pull takes the err!=nil branch, so cb stays nil — onDone is
	// never spuriously fired; the only bug was the stale snapshot write.)
	if m.gen != gen {
		m.mu.Unlock()
		return
	}
	m.progress.Active = false
	if err != nil {
		// A cancelled pull surfaces a raw context/transport error; keep the
		// friendly "pull cancelled" message Cancel() set instead of clobbering it.
		if ctx.Err() != nil {
			m.progress.Err = "pull cancelled"
		} else {
			m.progress.Err = err.Error()
		}
	} else {
		m.progress.Done = true
		m.progress.Err = ""
		m.tagsCacheAt = time.Time{} // a new model exists; force a refetch
		cb = m.onDone
	}
	m.mu.Unlock()
	if cb != nil {
		cb(tag)
	}
}
```

Note: when a same-`gen` Cancel happened (Cancel bumps `gen`), the goroutine is
already superseded and returns early — Cancel itself already set
`Active=false`/`Err="pull cancelled"`, so the snapshot stays correct.

**Verify**: `go build ./internal/ollama/` → exit 0; `go vet ./internal/ollama/`
→ exit 0.

### Step 5: Bound the control/cache calls with a short timeout (PERFORM-03)

A hung daemon must not wedge a board-render request forever. Apply a short
`context.WithTimeout` to the control calls only — Heartbeat, List (cold fetch),
and Delete. **Do not** touch the Pull path. Add a package constant near
`tagsTTL`:

```go
// controlTimeout bounds the non-streaming control calls (List/Heartbeat/Delete)
// so a hung Ollama daemon can't wedge a board-render or readiness check. It is
// deliberately NOT applied to Pull: a real pull streams for minutes, so it stays
// on a cancellable background context with no deadline.
const controlTimeout = 5 * time.Second
```

Then wrap each control call site. `cachedTags`' cold fetch:

```go
	ctx, cancel := context.WithTimeout(context.Background(), controlTimeout)
	defer cancel()
	models, err := m.fetchModels(ctx)
	if err != nil {
		return nil, err // do not cache errors
	}
```

`Delete`:

```go
func (m *Manager) Delete(tag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), controlTimeout)
	defer cancel()
	if err := m.apiClient().Delete(ctx, &api.DeleteRequest{Model: tag}); err != nil {
		return err
	}
	m.invalidateTags()
	return nil
}
```

`Reachable` already takes a `ctx` from its caller — leave its signature alone;
do **not** add an internal timeout there (the caller owns the deadline). If you
prefer symmetry, you may leave `Reachable` untouched; the brief only requires
List/Delete/cachedTags to be bounded.

**Verify**: `go build ./internal/ollama/` → exit 0; `go vet ./internal/ollama/`
→ exit 0; `gofmt -l internal/ollama/manager.go` prints nothing.

### Step 6: Add the five tests

Append to `internal/ollama/manager_test.go` (the file already imports
`context`, `net/http`, `net/http/httptest`, `strings`, `sync`, `sync/atomic`,
`testing`, `time` — add none unless your test needs one). Each test sets
`t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))` and uses a fresh
`m := &Manager{}`.

**(a) `TestCancelMidPull`** — Pull against a server that blocks (an unbuffered
`block` channel the handler reads from, released via `defer close(block)`), so
the pull never completes on its own. Then call `m.Cancel()`. Assert the
snapshot reflects cancellation and `onDone` never fires:

```go
func TestCancelMidPull(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hold the pull open until released
	}))
	defer srv.Close()
	defer close(block)
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	called := make(chan struct{}, 1)
	if err := m.Pull("a", func(string) { called <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	m.Cancel()
	snap := m.Snapshot()
	if snap.Active {
		t.Fatalf("Active still true after Cancel: %+v", snap)
	}
	if snap.Err != "pull cancelled" {
		t.Fatalf("Err = %q, want \"pull cancelled\"", snap.Err)
	}
	select {
	case <-called:
		t.Fatal("onDone fired on a cancelled pull")
	default:
	}
}
```

**(b) `TestPullError`** — server returns an error line / HTTP 500 so the pull
fails. Wait for the pull to settle (poll `Snapshot()` until `!Active`, bounded by
a deadline — do not sleep blindly), then assert `Err` is set, `Active` is false,
and `onDone` never fired. The ollama client treats a non-2xx or an
`{"error":...}` line as an error:

```go
func TestPullError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"boom"}` + "\n"))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	called := make(chan struct{}, 1)
	if err := m.Pull("a", func(string) { called <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	snap := waitPullSettled(t, m) // helper below
	if snap.Active {
		t.Fatalf("Active still true: %+v", snap)
	}
	if snap.Err == "" {
		t.Fatal("expected a non-empty Err on pull failure")
	}
	if snap.Done {
		t.Fatalf("Done true on a failed pull: %+v", snap)
	}
	select {
	case <-called:
		t.Fatal("onDone fired on a failed pull")
	default:
	}
}
```

Add a small polling helper next to the tests (no `time.Sleep` busy-spin in the
test body; bounded by a deadline so a hang fails loudly):

```go
func waitPullSettled(t *testing.T, m *Manager) PullSnapshot {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if snap := m.Snapshot(); !snap.Active {
			return snap
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("pull did not settle within 3s")
	return PullSnapshot{}
}
```

**(c) `TestCancelThenPullRaceDoesNotClobber`** — THE regression net for the
gen-token fix. Pull tag `a` against a blocking server, `Cancel()`, immediately
`Pull` tag `b` (which must be accepted because Cancel cleared `Active`), then
release the first goroutine and let it return late. Assert the second pull's
snapshot is NOT flipped to cancelled/`Done` by the first goroutine, and the
first pull's `onDone` never fires. Use TWO releasable gates so you control when
each server handler returns:

```go
func TestCancelThenPullRaceDoesNotClobber(t *testing.T) {
	// One gate per in-flight request; the test releases them by tag order.
	gates := make(chan chan struct{}, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g := make(chan struct{})
		gates <- g
		<-g // hold until released
		// Emit a terminal "success" so a non-cancelled pull can finish.
		w.Write([]byte(`{"status":"success"}` + "\n"))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}

	firstDone := make(chan struct{}, 1)
	if err := m.Pull("a", func(string) { firstDone <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	g1 := <-gates // first request is now in flight and parked on g1

	m.Cancel() // clears Active, bumps gen; g1's goroutine is now superseded

	if err := m.Pull("b", nil); err != nil {
		t.Fatalf("second Pull rejected: %v", err)
	}
	g2 := <-gates // second request in flight

	close(g1) // let the FIRST (superseded) goroutine return and try to write

	// Give the superseded goroutine time to run its completion block.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-firstDone:
			t.Fatal("cancelled pull fired onDone")
		default:
		}
		snap := m.Snapshot()
		if snap.URL != "b" {
			t.Fatalf("second pull's snapshot was clobbered: %+v", snap)
		}
		if !snap.Active || snap.Done || snap.Err != "" {
			t.Fatalf("second pull flipped by superseded goroutine: %+v", snap)
		}
		time.Sleep(5 * time.Millisecond)
	}

	close(g2) // let the second pull finish so the goroutine exits cleanly
	snap := waitPullSettled(t, m)
	if !snap.Done || snap.URL != "b" {
		t.Fatalf("second pull did not complete cleanly: %+v", snap)
	}
}
```

If the ollama client API ignores a streamed `{"status":"success"}` for the
terminal callback and never returns, fall back to releasing `g2` first and
asserting the second pull reaches `Done` with `URL == "b"`; the load-bearing
assertion is that the superseded first goroutine never set `Done`/`"pull
cancelled"` on the `b` snapshot and never called `onDone`. Do not weaken that
assertion.

**(d) `TestDeleteInvalidatesTagsCache`** — `List()` once (one server hit,
cached), `Delete(...)`, then `List()` again → a SECOND server hit (cache was
invalidated). Model the counter after `TestCachedTagsHitsServerOnceWithinTTL`.
The handler must serve both `/api/tags` (List) and `/api/delete` (Delete);
branch on `r.URL.Path`:

```go
func TestDeleteInvalidatesTagsCache(t *testing.T) {
	var listHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/tags"):
			atomic.AddInt32(&listHits, 1)
			w.Write([]byte(`{"models":[{"name":"gemma4:e4b","size":1}]}`))
		default: // delete
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if _, err := m.List(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.List(); err != nil { // within TTL: still 1 hit
		t.Fatal(err)
	}
	if c := atomic.LoadInt32(&listHits); c != 1 {
		t.Fatalf("List hits before delete = %d, want 1", c)
	}
	if err := m.Delete("gemma4:e4b"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.List(); err != nil {
		t.Fatal(err)
	}
	if c := atomic.LoadInt32(&listHits); c != 2 {
		t.Fatalf("List hits after delete = %d, want 2 (cache invalidated)", c)
	}
}
```

Confirm the path suffix by checking what the ollama client requests: if `/tags`
doesn't match, adjust the `switch` to whatever path the List call uses (the
Delete is a DELETE/POST to a different path). If unsure, log `r.Method` +
`r.URL.Path` in the handler during a first run.

**(e) `TestCachedTagsErrorPathDoesNotCache`** — a server that returns HTTP 500
makes `IsPulled` return an error and **not** cache; flip the server to success
and the next call still hits it and succeeds:

```go
func TestCachedTagsErrorPathDoesNotCache(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"models":[{"name":"gemma4:e4b","size":1}]}`))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if _, err := m.IsPulled("gemma4:e4b"); err == nil {
		t.Fatal("expected an error when the server fails")
	}
	fail.Store(false)
	ok, err := m.IsPulled("gemma4:e4b")
	if err != nil {
		t.Fatalf("second IsPulled errored: %v", err)
	}
	if !ok {
		t.Fatal("model should be present after server recovered")
	}
	if c := atomic.LoadInt32(&hits); c != 2 {
		t.Fatalf("server hit %d times, want 2 (error path did not cache)", c)
	}
}
```

**Verify**: `go test ./internal/ollama/` → all pass (existing 4 + new 5).

### Step 7: Full verification incl. the race detector

```
go vet ./internal/ollama/
go test ./internal/ollama/
CGO_ENABLED=1 go test -race ./internal/ollama/
go test ./...
CGO_ENABLED=0 go build ./...
gofmt -l .
git diff --check
```

**Verify**: vet clean; package tests pass; **`-race` reports no `DATA RACE`**;
whole-tree tests pass; CGO-free build exits 0; `gofmt -l .` prints nothing;
`git diff --check` prints nothing.

## Test plan

- New tests, all in `internal/ollama/manager_test.go`, closing TESTCO-02/03/04:
  - `TestCancelMidPull` — Cancel sets `Active=false`/`Err="pull cancelled"` and
    `onDone` does not fire (blocking server).
  - `TestPullError` — a failing server sets `Err`, leaves `Active=false`/
    `Done=false`, and `onDone` does not fire.
  - `TestCancelThenPullRaceDoesNotClobber` — **the regression net**: a
    superseded first goroutine must not flip the second pull's snapshot or call
    its `onDone`. This test must FAIL on the pre-fix code and PASS after the
    gen-token change.
  - `TestDeleteInvalidatesTagsCache` — Delete forces the next List to refetch.
  - `TestCachedTagsErrorPathDoesNotCache` — an error fetch is not cached.
- Structural pattern: model the blocking-server tests after
  `TestPullRejectsSecondConcurrent`, the streamed-progress + `onDone` flow after
  `TestPullSnapshotProgress`, and the server-hit counters after
  `TestCachedTagsHitsServerOnceWithinTTL`.
- Sanity check the fix actually addresses the race: optionally `git stash` the
  `manager.go` change and confirm `TestCancelThenPullRaceDoesNotClobber` fails
  (or flakes under `-race`) on the old code, then restore. This is a check, not
  a deliverable.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `Manager` has a `gen int` field; `Pull` and `Cancel` each `m.gen++` under
      the held lock; `runPull` captures `gen` and writes `m.progress` only when
      `m.gen == gen` at BOTH the progress callback and the completion block.
- [ ] No `Timeout` is set on the `http.Client` in `apiClient()`; the Pull path
      still runs on a deadline-free `context.Background()` (long pulls unbroken).
- [ ] `List`/`Delete`/`cachedTags` cold fetch use a `context.WithTimeout(..., controlTimeout)`.
- [ ] `PullSnapshot` and `Model` field names are unchanged (`git diff` shows no
      field renames).
- [ ] `internal/web/models.go` is unmodified (`git status --porcelain` lists only
      the two in-scope files).
- [ ] `go test ./internal/ollama/` passes with the 5 new tests present.
- [ ] `CGO_ENABLED=1 go test -race ./internal/ollama/` exits 0 with no `DATA RACE`.
- [ ] `go vet ./internal/ollama/` exits 0; `go test ./...` passes;
      `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `gofmt -l .` prints nothing; `git diff --check` prints nothing.
- [ ] `plans/readme.md` status row for 065 updated (unless your reviewer maintains it).

## STOP conditions

Stop and report back (do not improvise) if:

- `manager.go`'s `Pull` (lines ~75-87), `runPull` (~89-121), or `Cancel`
  (~123-135) does not match the "Current state" excerpts — the codebase has
  drifted since this plan was written; reconcile before editing.
- The gen-token change makes any **existing** manager test fail
  (`TestReachable`, `TestPullSnapshotProgress`, `TestPullRejectsSecondConcurrent`,
  `TestCachedTagsHitsServerOnceWithinTTL`). The fix must be additive to the
  happy path — a regression there means the guard is wrong.
- `TestCancelThenPullRaceDoesNotClobber` cannot be made to PASS deterministically
  (it flakes after the fix) — that means the gen guard isn't covering one of the
  two write sites; do not ship a flaky test.
- Bounding the control calls with `controlTimeout` makes
  `TestCachedTagsHitsServerOnceWithinTTL` or `TestDeleteInvalidatesTagsCache`
  flake (5s is too tight for the httptest loopback — it should be plenty; if it
  isn't, something else is wrong).
- The fix appears to require editing any out-of-scope file (e.g. `web/models.go`
  or `presets.go`) — the manager change is meant to be self-contained.

## Maintenance notes

- The single-slot pull model assumes exactly one background pull. If Balaur ever
  allows concurrent pulls (multiple models at once), the `gen` token must become
  per-pull state (a map keyed by an id), not a single counter — revisit then.
- A reviewer should confirm (1) BOTH `runPull` write sites are gen-guarded — the
  progress callback AND the completion block; missing either reopens the race —
  and (2) the Pull path has no client- or context-level deadline (grep
  `apiClient()` usages: only List/Heartbeat/Delete carry a timeout).
- Plan 070 renames the gguf-derived `PullSnapshot`/`Model` fields; this plan
  deliberately leaves those names alone so the two don't collide. If 070 lands
  first, the new tests' struct-field references (`snap.Active`, `snap.URL`,
  `snap.Done`, `snap.Err`) may need the renamed names — adjust accordingly.
- Deferred out of this plan: an internal timeout on `Reachable` (its caller owns
  the deadline today) and any change to web handlers; both stay out of scope.
