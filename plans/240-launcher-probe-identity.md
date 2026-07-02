# Plan 240: Make the single-instance launcher probe verify it found Balaur, not just any TCP listener

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/launch/launch.go internal/launch/launch_test.go main.go .tours/19-bootstrapping.tour`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

A bare `balaur` invocation (double-click, no terminal) checks a lock file for
an already-running instance and, if the recorded address answers a **bare TCP
dial**, prints "already running", opens the browser at that address, and exits
**without starting Balaur**. The lock file (`.balaur-launcher.json`) is
deliberately never removed on shutdown — the probe is the liveness authority —
and the default port is a stable `8099`. So after Balaur stops, ANY local
process that later listens on 8099 (a dev server, another app) makes every
subsequent bare launch open a foreign service in the browser and silently
refuse to start Balaur. The no-terminal owner has no recovery except
hand-deleting a hidden lock file. Additionally, the probe accepts the lock's
address verbatim without checking it is loopback, contradicting the package's
own documented invariant ("the package never constructs a non-loopback
address") — a tampered lock file could make the launcher open an off-box URL.
This plan makes the probe (a) refuse non-loopback lock addresses and (b)
require an HTTP 200 from PocketBase's unauthenticated `/api/health` endpoint
before treating the instance as alive, while preserving every existing
fail-open branch. It also replaces a tautological test that asserts a prefix
on a string the test itself just built.

## Current state

### Files

- `internal/launch/launch.go` — the no-args loopback launcher helpers;
  contains `RunningInstance` (lines 172–192), the bare-TCP probe to fix.
- `internal/launch/launch_test.go` — the launcher tests; contains the
  unfalsifiable `TestSelectPortAddressIsLoopback` (lines 175–187) and the
  `RunningInstance` table (lines 189–304), several of which must convert
  from plain TCP listeners to HTTP test servers.
- `main.go` — the launcher entry (lines 52–95); its comment at lines 54–58
  says "TCP probe" and must be reworded to "HTTP health probe".
- `.tours/19-bootstrapping.tour` — has one line-anchored step into
  `internal/launch/launch.go` at line 82 (the `DefaultPort` comment block);
  adding an import to `launch.go` shifts it by one.

### The package invariant (launch.go:1–7)

```go
// Package launch holds the no-args loopback launcher: the smallest slice that
// lets a non-developer start Balaur without a shell. A bare `balaur` invocation
// (no subcommand, no flags) defaults the data dir to the XDG data dir, finds a
// free loopback port, and opens the browser — then hands control to the existing
// `serve` path by rewriting argv (see main.go). Every helper here is pure or
// trivially testable; the package never constructs a non-loopback address.
package launch
```

### The buggy probe (launch.go:172–192)

```go
// RunningInstance reads the single-instance lock for dataDir and probes the
// recorded address. It returns (addr, true) ONLY when the address is present
// in the lock AND a TCP connection succeeds (instance is live). Every error —
// missing file, unreadable, malformed JSON, empty addr, probe timeout — returns
// ("", false) so the caller proceeds to start a new server (fail-open).
func RunningInstance(dataDir string) (addr string, alive bool) {
	data, err := os.ReadFile(lockPath(dataDir))
	if err != nil {
		return "", false // missing or unreadable — proceed to start
	}
	var lock instanceLock
	if err := json.Unmarshal(data, &lock); err != nil || lock.Addr == "" {
		return "", false // malformed — proceed to start
	}
	conn, err := net.DialTimeout("tcp", lock.Addr, 300*time.Millisecond)
	if err != nil {
		return "", false // stale (crashed or stopped) — proceed to start
	}
	conn.Close()
	return lock.Addr, true
}
```

Two defects: the dial succeeds against ANY TCP listener (identity is never
checked), and `lock.Addr` is used verbatim (never checked to be loopback).

### The stable default port that makes collisions likely (launch.go:77–83)

```go
// DefaultPort is the stable loopback port a no-args launch tries first, so the
// URL (http://127.0.0.1:8099/) is bookmarkable instead of changing every boot.
// ...
const DefaultPort = 8099
```

### The caller that exits without starting (main.go:52–66)

```go
	var isFirstRun bool
	if launch.IsLauncherInvocation(os.Args[1:]) {
		// Single-instance guard (plan 232): if another server is already running
		// on the same data dir, open it and exit rather than starting a second
		// one. FAIL-OPEN: any error from RunningInstance → proceed to start.
		// Stale locks (crashed/stopped instance) are handled by the TCP probe —
		// no response means stale, and we proceed to start normally.
		if addr, alive := launch.RunningInstance(launch.DataDir()); alive {
			url := "http://" + addr + "/"
			fmt.Fprintf(os.Stderr, "Balaur is already running — opening %s\n", url)
			if err := launch.OpenBrowser(url); err != nil {
				fmt.Fprintf(os.Stderr, "(could not open browser: %v — open %s manually)\n", err, url)
			}
			return
		}
```

### The identity endpoint: PocketBase v0.39.3 serves `/api/health` unauthenticated

Verified in the module cache,
`~/go/pkg/mod/github.com/pocketbase/pocketbase@v0.39.3/apis/health.go:11–26`:

```go
// bindHealthApi registers the health api endpoint.
func bindHealthApi(app core.App, rg *router.RouterGroup[*core.RequestEvent]) {
	subGroup := rg.Group("/health")
	subGroup.GET("", healthCheck)
}

// healthCheck returns a 200 OK response if the server is healthy.
func healthCheck(e *core.RequestEvent) error {
	...
	Code:    http.StatusOK,
	Message: "API is healthy.",
```

It returns 200 to everyone; superuser auth only adds extra `Data` fields.
Balaur's own middleware does not gate it either —
`internal/web/web.go:64–68`:

```go
func guardLocalUI(e *core.RequestEvent) error {
	p := e.Request.URL.Path
	if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/_") {
		return e.Next()
	}
```

So `GET http://127.0.0.1:<port>/api/health` → HTTP 200 is a reliable
"this is a PocketBase-family server" identity check, and a foreign dev
server will typically 404, refuse, or redirect — all treated as stale.

### The tautological test (launch_test.go:175–187)

```go
func TestSelectPortAddressIsLoopback(t *testing.T) {
	port, err := SelectPort()
	if err != nil {
		t.Fatalf("SelectPort() error = %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("constructed addr %q is not loopback", addr)
	}
	if strings.Contains(addr, "0.0.0.0") {
		t.Errorf("constructed addr %q exposes all interfaces", addr)
	}
}
```

It asserts a prefix on a string the test itself just built with
`fmt.Sprintf("127.0.0.1:%d", port)` — it can never fail, so the package's
one named security property is effectively untested.

### Existing tests that assume "any TCP listener = alive" (must be converted)

- `TestRunningInstance_Live` (launch_test.go:190–210) — `net.Listen` plain
  TCP listener, expects `alive=true`.
- `TestRunningInstance_RoundTrip` (launch_test.go:259–276) — same pattern.
- `TestRunningInstance_DifferentDataDirs` (launch_test.go:280–304) — same
  pattern for `dataDir1`.

These three will FAIL under the new probe until converted to an HTTP test
server that answers 200 on `/api/health`. `TestRunningInstance_Stale`
(213–230), `_Missing` (233–239), and `_Malformed` (242–256) stay valid as-is.

### The tour anchor that shifts

`.tours/19-bootstrapping.tour:13–14`:

```json
      "file": "internal/launch/launch.go",
      "line": 82,
```

Line 82 sits inside the `DefaultPort` comment block. Adding one import line
(`"net/http"`) to `launch.go` shifts it to 83. `tours_test.go` only fails on
missing files/out-of-range lines, but repo convention is: when a change
shifts an anchored line, fix the anchor in the same commit.

### Repo conventions that apply here

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code. (`RunningInstance` returns `("", false)` instead of errors —
  keep that contract; it is the documented fail-open design.)
- No global mutable state; no `fmt.Print*` in service code (`main.go`'s
  pre-`New()` stderr prints are the one allowed exception, already noted in
  its comments — do not add new ones).
- Tests: standard `testing` package, table-driven where it helps; no
  `time.Sleep`-based synchronization. This package's tests need no
  PocketBase app — plain `net`/`httptest` suffices, as the existing file
  shows.
- `internal/self/knowledge.md` describes the launcher at lines 421–427
  ("boots a loopback UI on the XDG data dir, prefers a stable default port
  (8099 …)") without mentioning probe mechanics — this change does NOT alter
  the described capability, so knowledge.md is intentionally not touched.
- KISS: smallest correct change; no new config knobs, no retry loops.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Format check | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Targeted tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/launch/ -count=1` | ok, exit 0 |
| One test | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/launch/ -run TestRunningInstance -count=1 -v` | all subtests PASS |
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0 |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` for test runs. `make test` exports it
automatically but is cached; the `-count=1` form above is the gate.)

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before Step 1 — it
  covers this repo's error-handling, testing, and modern-stdlib idioms.

## Scope

**In scope** (the only files you may modify):

- `internal/launch/launch.go`
- `internal/launch/launch_test.go`
- `main.go` (comment rewording only, lines 54–58; keep the line count
  identical so the file's tour anchors at main.go:47/75/126/136/226 stay put)
- `.tours/19-bootstrapping.tour` (one anchor line-number bump)
- `plans/README.md` (status row only)

**Out of scope** (do NOT touch, even though they look related):

- Lock-file cleanup on shutdown — deliberately skipped per the settled plan-232
  design (the probe is the liveness authority); this plan's probe fix
  neutralizes the stale-lock hazard without adding shutdown paths.
- `SelectPort` / `FreeLoopbackPort` / `OpenBrowser` / `waitForListener`
  semantics — `waitForListener`'s bare TCP dial is fine: it waits for OUR OWN
  just-started server, so identity is already known.
- `docs/first-run-design.md` — documentation truth-sync is handled by a
  separate docs plan (see Maintenance notes), not here.
- `internal/self/knowledge.md` — the described capability (bare launch reuses
  a running instance) is unchanged; only probe fidelity improves.
- `internal/web/web.go` — read-only reference for the `/api/` middleware
  bypass; nothing to change there.

## Git workflow

- The executor runs in an isolated git worktree branched from `origin/main`;
  branch name: `advisor/240-launcher-probe-identity`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`), e.g. `fix(launch): probe /api/health identity, reject non-loopback lock addr`.
- Commit per logical unit with explicit pathspecs (the main checkout is
  shared by parallel agents — stage only your own files, e.g.
  `git add internal/launch/launch.go internal/launch/launch_test.go`).
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Harden `RunningInstance` — loopback check + HTTP identity probe

In `internal/launch/launch.go`:

1. Add `"net/http"` to the import block (after `"net"`).
2. Replace the body of `RunningInstance` (and its doc comment) so it:
   - keeps the existing read/unmarshal fail-open branches verbatim;
   - rejects a non-loopback `lock.Addr` BEFORE any network activity
     (this ordering is load-bearing: a tampered lock file must never
     trigger an off-box request);
   - probes `http://<addr>/api/health` with a ~500ms client timeout and
     redirects disabled, and reports alive ONLY on HTTP 200.

Target shape:

```go
// RunningInstance reads the single-instance lock for dataDir and probes the
// recorded address. It returns (addr, true) ONLY when the lock's address is
// loopback AND GET http://<addr>/api/health answers HTTP 200 — PocketBase's
// unauthenticated health endpoint, so a foreign process that grabbed the port
// after Balaur stopped (the lock is deliberately not removed on shutdown) is
// never mistaken for a live instance. Every other outcome — missing file,
// unreadable, malformed JSON, empty or non-loopback addr, connection refused,
// timeout, non-200, redirect — returns ("", false) so the caller proceeds to
// start a new server (fail-open).
func RunningInstance(dataDir string) (addr string, alive bool) {
	data, err := os.ReadFile(lockPath(dataDir))
	if err != nil {
		return "", false // missing or unreadable — proceed to start
	}
	var lock instanceLock
	if err := json.Unmarshal(data, &lock); err != nil || lock.Addr == "" {
		return "", false // malformed — proceed to start
	}
	host, _, err := net.SplitHostPort(lock.Addr)
	if err != nil {
		return "", false // malformed addr — proceed to start
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		// The lock is only ever written with 127.0.0.1:<port> (see main.go);
		// anything else is tampering or corruption. Never probe or open it —
		// the package never constructs a non-loopback address.
		return "", false
	}
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
		// A redirect is not Balaur's health endpoint — do not follow it.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("http://" + lock.Addr + "/api/health")
	if err != nil {
		return "", false // stale (crashed or stopped) — proceed to start
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false // something else answered on the port — treat as stale
	}
	return lock.Addr, true
}
```

Notes:
- `net.ParseIP("localhost")` returns nil → fail-open. That is correct: the
  lock writer (`main.go:74` builds `fmt.Sprintf("127.0.0.1:%d", port)`) never
  writes a hostname.
- Do not add retries, config knobs, or logging — `RunningInstance` runs
  pre-`pocketbase.New()`, before `app.Logger()` exists, and its contract is
  silent fail-open.

**Verify**: `CGO_ENABLED=0 go build ./... && go vet ./internal/launch/` → exit 0.
(Existing tests `TestRunningInstance_Live`, `_RoundTrip`,
`_DifferentDataDirs` are EXPECTED to fail right now — Step 3 converts them.
Do not "fix" the code to make them pass.)

### Step 2: Reword the `main.go` comment (same line count)

In `main.go`, replace lines 57–58 of the plan-232 comment:

```go
		// Stale locks (crashed/stopped instance) are handled by the TCP probe —
		// no response means stale, and we proceed to start normally.
```

with (still exactly two lines, so the tour anchors into `main.go` do not
shift):

```go
		// Stale locks (crashed/stopped instance) are handled by the HTTP health
		// probe — anything but a 200 from /api/health means stale; we start.
```

**Verify**: `gofmt -l .` → empty; `git diff --stat main.go` → shows
`main.go | 4 ++--` (2 insertions, 2 deletions — comment-only, line count
unchanged).

### Step 3: Convert the "alive" tests to a real health endpoint

In `internal/launch/launch_test.go`:

1. Add imports `"net/http"` and `"net/http/httptest"` (`"strings"` is
   already imported).
2. Add one helper:

```go
// healthServer starts a loopback HTTP server answering 200 on /api/health —
// the identity signature RunningInstance probes for — and returns its
// host:port address.
func healthServer(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return strings.TrimPrefix(ts.URL, "http://")
}
```

3. In `TestRunningInstance_Live`, `TestRunningInstance_RoundTrip`, and
   `TestRunningInstance_DifferentDataDirs`: delete the `net.Listen` /
   `defer l.Close()` / `addr := l.Addr().String()` setup and replace it with
   `addr := healthServer(t)`. Keep every assertion unchanged. Update each
   test's doc comment to say the lock points at "a live Balaur-shaped health
   endpoint" rather than "a live listener".
4. Leave `TestRunningInstance_Stale`, `_Missing`, `_Malformed` untouched
   (they assert fail-open branches that still exist).

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/launch/ -run TestRunningInstance -count=1 -v`
→ all `TestRunningInstance_*` PASS.

### Step 4: Add the identity- and loopback-rejection tests

Append to `internal/launch/launch_test.go`:

```go
// TestRunningInstance_ForeignTCPListener: a plain TCP listener on the lock
// addr (some other process that grabbed the port after Balaur stopped — the
// lock is deliberately not removed on shutdown) must NOT count as alive: the
// probe requires an HTTP 200 from /api/health, not just an accepted dial.
func TestRunningInstance_ForeignTCPListener(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Close() // accepts TCP, speaks no HTTP
		}
	}()

	dataDir := filepath.Join(t.TempDir(), "pb_data")
	if err := WriteInstanceLock(dataDir, l.Addr().String()); err != nil {
		t.Fatalf("WriteInstanceLock: %v", err)
	}
	if _, alive := RunningInstance(dataDir); alive {
		t.Fatal("RunningInstance: want alive=false for a non-HTTP TCP listener")
	}
}

// TestRunningInstance_ForeignHTTPServer: an HTTP server that is not Balaur
// (404s or redirects /api/health) is treated as stale — fail-open to start.
func TestRunningInstance_ForeignHTTPServer(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
	}{
		{"404 on health", http.NotFoundHandler()},
		{"redirects health", http.RedirectHandler("http://127.0.0.1:1/", http.StatusFound)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(tt.handler)
			t.Cleanup(ts.Close)
			dataDir := filepath.Join(t.TempDir(), "pb_data")
			if err := WriteInstanceLock(dataDir, strings.TrimPrefix(ts.URL, "http://")); err != nil {
				t.Fatalf("WriteInstanceLock: %v", err)
			}
			if _, alive := RunningInstance(dataDir); alive {
				t.Fatal("RunningInstance: want alive=false for a foreign HTTP server")
			}
		})
	}
}

// TestRunningInstance_NonLoopbackAddr: a lock pointing off-box must never be
// reported alive (and must never be probed) — the launcher's invariant is
// that it never constructs, opens, or trusts a non-loopback address.
func TestRunningInstance_NonLoopbackAddr(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "pb_data")
	if err := WriteInstanceLock(dataDir, "10.0.0.5:8099"); err != nil {
		t.Fatalf("WriteInstanceLock: %v", err)
	}
	if _, alive := RunningInstance(dataDir); alive {
		t.Fatal("RunningInstance: want alive=false for a non-loopback lock addr")
	}
}
```

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/launch/ -run 'TestRunningInstance_Foreign|TestRunningInstance_NonLoopback' -count=1 -v`
→ 3 tests (4 subtests total) PASS.
Also: `TestRunningInstance_NonLoopbackAddr` must complete in well under a
second (`-v` prints per-test times) — if it takes ~500ms+, the loopback check
is running AFTER the probe; fix the ordering in `RunningInstance`.

### Step 5: Rewrite `TestSelectPortAddressIsLoopback` to assert an observed property

Replace the whole function (launch_test.go:175–187) with:

```go
// TestSelectPortAddressIsLoopback asserts the OBSERVED kernel address of the
// selected port bound on 127.0.0.1 is loopback — not a prefix of a string the
// test built itself.
func TestSelectPortAddressIsLoopback(t *testing.T) {
	port, err := SelectPort()
	if err != nil {
		t.Fatalf("SelectPort() error = %v", err)
	}
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		// TOCTOU: something grabbed the port between SelectPort closing it and
		// this bind — same accepted window the launcher itself lives with.
		t.Skipf("port %d taken between SelectPort and re-bind: %v", port, err)
	}
	defer l.Close()
	tcpAddr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected addr type %T", l.Addr())
	}
	if !tcpAddr.IP.IsLoopback() {
		t.Errorf("bound addr %v is not loopback", tcpAddr)
	}
}
```

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/launch/ -run TestSelectPortAddressIsLoopback -count=1 -v`
→ PASS (or SKIP with the TOCTOU message, which is acceptable).

### Step 6: Fix the tour anchor and run the full gates

1. In `.tours/19-bootstrapping.tour`, step 19.2 anchors
   `"file": "internal/launch/launch.go", "line": 82`. After Step 1 added one
   import line, the `DefaultPort` comment block moved down by one. Confirm
   with `grep -n "DefaultPort is the stable" internal/launch/launch.go`
   (expect line 78 after the change) and bump the tour's `"line": 82` to
   `"line": 83`. If your diff added a different number of lines above the old
   line 82, set the anchor to old-82 + that delta instead.
2. Run the full gate suite (all commands from the table):
   - `gofmt -l .` → empty
   - `go vet ./...` → exit 0
   - `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output
   - `CGO_ENABLED=0 go build ./...` → exit 0
   - `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
   - `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0

**Verify**: every command above matches its expected output.

## Test plan

- **File**: `internal/launch/launch_test.go` (extend; model new tests after
  the existing `TestRunningInstance_*` functions in the same file — plain
  `testing`, `t.TempDir()`, `t.Cleanup`, no PocketBase app, no `time.Sleep`).
- **Converted** (bug this plan fixes, positive path):
  `TestRunningInstance_Live`, `_RoundTrip`, `_DifferentDataDirs` — lock →
  httptest server answering 200 on `/api/health` → `alive=true`, addr
  returned.
- **New — the regression this plan exists for**:
  `TestRunningInstance_ForeignTCPListener` — plain TCP accept-and-close
  listener → `alive=false`.
- **New — foreign HTTP**: `TestRunningInstance_ForeignHTTPServer` — 404 and
  302 on `/api/health` → `alive=false` (both subtests).
- **New — invariant**: `TestRunningInstance_NonLoopbackAddr` — lock addr
  `10.0.0.5:8099` → `alive=false`, fast (no probe attempted).
- **Rewritten**: `TestSelectPortAddressIsLoopback` — binds the selected port
  and asserts `listener.Addr().(*net.TCPAddr).IP.IsLoopback()`.
- **Unchanged and must stay green**: `TestRunningInstance_Stale`, `_Missing`,
  `_Malformed`, plus all other tests in the file.
- **Verification**:
  `TMPDIR=$HOME/.cache/go-tmp go test ./internal/launch/ -count=1 -v`
  → all pass; output lists `TestRunningInstance_ForeignTCPListener`,
  `TestRunningInstance_ForeignHTTPServer`, `TestRunningInstance_NonLoopbackAddr`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0; `gofmt -l .` prints nothing
- [ ] `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` prints nothing, exits 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` exits 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/launch/ -count=1 -v 2>&1 | grep -c 'PASS: TestRunningInstance_Foreign\|PASS: TestRunningInstance_NonLoopback'` ≥ 3 (the new tests exist and pass; use `--- PASS:` lines)
- [ ] `grep -c "DialTimeout" internal/launch/launch.go` prints `1` (only `waitForListener` still dials raw TCP; `RunningInstance` no longer does)
- [ ] `grep -n "api/health" internal/launch/launch.go` shows the probe URL in `RunningInstance`
- [ ] `grep -n "IsLoopback" internal/launch/launch.go` shows the loopback gate in `RunningInstance`
- [ ] `grep -n "TCP probe" main.go` prints nothing (comment reworded)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `git status --porcelain` lists ONLY: `internal/launch/launch.go`, `internal/launch/launch_test.go`, `main.go`, `.tours/19-bootstrapping.tour`, `plans/README.md`
- [ ] `plans/README.md` status row for plan 240 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows changes to any in-scope file, or the "Current state"
  excerpts do not match the live code (e.g. `RunningInstance` no longer sits
  at launch.go:172–192, or `main.go:59` no longer calls it).
- `GET /api/health` turns out to be gated or absent when you run the launch
  tests (an httptest server is the reference here, but if you find PocketBase
  v0.39.3's `apis/health.go` no longer matches the excerpt above — e.g. the
  dependency was bumped — the identity contract changed). Fallback design if
  so: probe `GET http://<addr>/` and require 200 plus a Balaur marker in the
  body — but FIRST verify what `GET /` actually returns on a running instance
  and report before implementing.
- The httptest-based tests cannot bind loopback in your environment (all
  existing launch tests would already be failing — report, do not work
  around).
- A step's verification fails twice after a reasonable fix attempt —
  especially if `TestRunningInstance_Stale`/`_Missing`/`_Malformed` start
  failing, which would mean a fail-open branch was lost.
- The fix appears to require touching `internal/web`, `docs/`, or
  `internal/self/knowledge.md` — it must not.
- You discover the assumption "the lock file is written only by main.go with
  a `127.0.0.1:<port>` address" is false (i.e. some other code path writes
  `WriteInstanceLock` with a hostname or non-loopback addr).

## Maintenance notes

- **Residual false-positive**: a foreign local server with a catch-all route
  that answers 200 to every path (e.g. an SPA dev server) on 8099 would still
  pass the probe. Accepted as KISS: requiring the response body to contain
  PocketBase's `"API is healthy."` message would pin us to an upstream
  string. Revisit only if it bites a real owner.
- **Reviewer scrutiny points**: (1) the loopback check runs BEFORE any
  network I/O; (2) every pre-existing fail-open branch survives (stale /
  missing / malformed tests unchanged and green); (3) `resp.Body.Close()` is
  called; (4) redirects are not followed; (5) `main.go` diff is comment-only
  with unchanged line count.
- **Interaction with future work**: if the launcher ever probes before
  PocketBase finishes booting (e.g. a supervisor restarting Balaur), the
  500ms timeout treats a slow-booting real instance as stale and a second
  start will fail on the port bind — acceptable today (probe runs only on
  bare human launches), but revisit the timeout if launch orchestration
  changes. If PocketBase is upgraded past v0.39.3, re-verify `/api/health`
  stays unauthenticated (AGENTS.md already flags PB upgrades as repo-wide).
- **Deferred docs**: `docs/first-run-design.md` still describes the
  single-instance guard as a follow-up with a TCP probe; a separate docs
  truth-sync plan (`plans/252-docs-truth-sync-post-230-234.md`) owns bringing
  the design docs in line — do not edit docs here.
- **Deliberate non-fix**: the lock file is still never removed on shutdown.
  That is the settled plan-232 design (the probe is the liveness authority);
  this plan makes that design safe rather than replacing it.
