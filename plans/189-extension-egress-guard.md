# Plan 189: Default-deny cloud-metadata & link-local egress from extension `balaur.http` (owner opt-out)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- internal/ext/vm.go internal/ext/ext.go internal/ext/ext_test.go internal/store/owner_settings.go internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

A balaur-extension's JS handler can call `balaur.http({url})` with **any**
URL — including `169.254.169.254` (the cloud instance-metadata endpoint),
`127.0.0.1`, and link-local addresses — and the full response body + headers
are handed back to the handler. That is a clean exfiltration channel for a
cloud-credential source. The extension trust model is owner-approval of
reviewed, sha256-pinned JS, so this is **defense-in-depth, not an open hole**;
and reaching *local* services is a documented by-design decision (a personal
box talks to its own services). **But on this deployment the dev box IS the
live prod VPS, where `169.254.169.254` returns real cloud credentials.** This
plan *hardens* the existing decision rather than reversing it: it
**default-denies** the cloud-metadata + link-local ranges at the dialer, while
leaving an **owner opt-out** (`ext_local_egress`) so an owner who genuinely
needs local egress can re-enable it. Loopback (`127.0.0.1`, the address
`httptest` binds) is intentionally **not** in the default-deny set, so existing
extension behavior and tests stay green.

## Current state

### `internal/ext/vm.go` — the bare client and the dial site

The HTTP client is a **package-level var** with no egress filtering — only
redirect-following is disabled (vm.go:34–42):

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

`invoke()` is the entry that runs a handler. **It does NOT receive `app`** —
it has only `ctx` (vm.go:124):

```go
func invoke(ctx context.Context, src, name, tool, argsJSON string) (out string, err error) {
```

`newVM()` builds the runtime and wires the http binding (vm.go:60, 85–86):

```go
func newVM(ctx context.Context, src, name string, withHTTP bool) (*goja.Runtime, []captured, error) {
```
```go
	if withHTTP {
		_ = balaur.Set("http", httpBinding(ctx, vm))
```

`httpBinding()` builds the request and issues it through the package var; note
the by-design comment about local addresses (vm.go:198–253, the dial at 234):

```go
// httpBinding implements balaur.http({url, method?, headers?, body?}) →
// {status, body, headers}. Errors throw as JS exceptions so handlers can
// try/catch. Local addresses are deliberately reachable: a personal box
// talks to its own services; the audit log carries every invocation.
func httpBinding(ctx context.Context, vm *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		opts, _ := call.Argument(0).Export().(map[string]any)
		...
		resp, err := extHTTPClient.Do(req)
		if err != nil {
			panic(vm.NewGoError(err))
		}
```

The JS-error convention for a Go-side failure inside the binding is
`panic(vm.NewGoError(err))` (vm.go:225, 236, 241); validation failures use
`panic(vm.NewTypeError("…"))`. A denied dial must surface as one of these —
**never** a process crash. `invoke()`'s top-level `defer recover()` (vm.go:125)
turns any panic into a returned error, and `httpBinding` runs synchronously
inside the handler call, so a `NewGoError` panic from the dialer becomes a
normal returned `error` the JS `try/catch` can also see.

`net` is **not** currently imported in `vm.go`. The current imports are:
`context`, `encoding/json`, `fmt`, `io`, `net/http`, `strings`, `time`, and
`github.com/dop251/goja`.

### `internal/ext/ext.go` — where `app` IS in scope

`app core.App` reaches `extTool` (ext.go:130), whose `Execute` closure calls
`invoke` (ext.go:138–142):

```go
	return agent.Tool{
		Spec: agent.ToolSpecOf(toolName, def.Description+" (balaur-extension: "+extName+")", params),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			res, err := invoke(ctx, src, extName, toolName, argsJSON)
			store.Audit(app, "extensions", "ext.invoke", extName+"/"+toolName, err == nil, nil)
			return res, err
		},
	}
```

This is the threading seam: read the opt-out flag here (where `app` exists) and
pass the resulting `bool` down into `invoke`. `ext.go` already imports
`github.com/alexradunet/balaur/internal/store`.

### `internal/store/owner_settings.go` — how owner flags are read

A string-valued setting with a default (owner_settings.go:29–41):

```go
// GetOwnerSetting returns the value of a key from the owner_settings
// collection. Returns defaultVal if the key is not found or any error occurs.
func GetOwnerSetting(app core.App, key, defaultVal string) string {
	rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
	if err != nil {
		return defaultVal
	}
	v := rec.GetString("value")
	if v == "" {
		return defaultVal
	}
	return v
}
```

**Boolean opt-in/opt-out flags are read by string comparison — no new helper,
no migration.** Exemplar (`internal/tasks/nudge.go:32`):

```go
	if store.GetOwnerSetting(app, "nudge_enabled", "1") == "0" {
```

So a default-OFF opt-out flag is read as:
`store.GetOwnerSetting(app, "ext_local_egress", "") == "1"` (default `""` ⇒
flag off ⇒ guard ON). The `owner_settings` collection already exists from the
baseline migration; **no new migration is required** — a key materializes on
first write, and an unread key falls through to the default. Do **not** add a
migration.

### `internal/self/knowledge.md` — the running binary's self-description

The extensions paragraph the executor must keep honest (knowledge.md:223–229):

```
- Extensions: propose_extension submits a balaur-extension — one
  JavaScript file in pb_extensions/ that registers new tools via
  balaur.registerTool({name, description, parameters, handler}); handlers
  may call balaur.http. An extension's tools join your registry only
  while the owner has approved exactly that file content (sha256-pinned);
  any change re-proposes it, and every invocation is audited. Extensions
  add verbs, not privileges — no filesystem, no shell, no npm.
```

### Test harness facts (`internal/ext/ext_test.go`)

- Tests build a PocketBase-backed app with `storetest.NewApp(t)` (boots the
  full migration chain), isolate the extensions dir with `setupDir(t)`
  (`t.Setenv("BALAUR_EXT_DIR", …)`), `write(t, app, name, src)` the JS,
  `Sync(app)`, `Approve(app, name)`, then `Tools(app, map[string]bool{})` and
  `ts[0].Execute(ctx, argsJSON)`.
- `TestHTTPBindingWorksInsideHandlers` (ext_test.go:224) fetches an
  `httptest.NewServer`, which binds to **`127.0.0.1`** (loopback, **not**
  link-local). Because the default-deny set excludes loopback, this test stays
  green unchanged.
- A flag is flipped on in a test with
  `store.SetOwnerSetting(app, "ext_local_egress", "1")` (the setter at
  owner_settings.go:47). `ext_test.go` does not yet import `internal/store`;
  add that import when you write the opt-out test.

## Commands you will need

| Purpose        | Command                          | Expected on success          |
|----------------|----------------------------------|------------------------------|
| Build (CGO-off)| `CGO_ENABLED=0 go build ./...`   | exit 0, no output            |
| Test (ext pkg) | `go test ./internal/ext/`        | `ok  …/internal/ext`         |
| Test (all)     | `go test ./...`                  | all packages `ok`            |
| Vet            | `go vet ./...`                   | exit 0, no output            |
| Fmt check      | `gofmt -l .`                     | empty output                 |
| Diff check     | `git diff --check`              | exit 0, no output            |

(Per the repo: gofmt is law — a PostToolUse hook + CI gate enforce it; `go vet`,
`staticcheck`, `govulncheck` also gate CI. Tests are standard `testing`,
table-driven where it helps, **no** assertion frameworks, **no** `time.Sleep`,
`t.TempDir()` for I/O, `storetest.NewApp(t)` for a PB-backed app.)

## Suggested executor toolkit

- Invoke the `go-standards` skill if available — it covers the error-wrapping
  (`fmt.Errorf("…: %w", err)`), structured-logging, and testing idioms this
  plan relies on.

## Scope

**In scope** (the only files you should modify):
- `internal/ext/vm.go` — add a default-deny dialer + opt-out plumbing on the
  http client; thread the opt-out bool through `invoke → newVM → httpBinding`.
- `internal/ext/ext.go` — read `ext_local_egress` in `extTool` and pass the
  bool into `invoke`.
- `internal/ext/ext_test.go` — add egress tests.
- `internal/self/knowledge.md` — one clause documenting the new default-deny +
  opt-out, in the extensions paragraph.

**Out of scope** (do NOT touch, even though they look related):
- **Removing local reach** — loopback / local egress is **by-design**. You are
  *gating* the dangerous ranges behind an opt-out, not removing reachability.
  Do **not** add loopback (`127.0.0.0/8`, `::1`) to the default-deny set;
  doing so breaks `TestHTTPBindingWorksInsideHandlers`.
- The consent ledger, sha256 pinning, `Sync`/`Approve`/`Disable`, proposal
  flow (`propose.go`) — unrelated to egress.
- Any new migration file. `owner_settings` already exists; the flag needs no
  schema. **Do not** create a migration.
- `internal/store/owner_settings.go` — the existing `GetOwnerSetting` /
  `SetOwnerSetting` are sufficient; do not add a typed getter (YAGNI).
- A Models/Settings UI toggle for the flag — read-only consumption is enough
  for this plan; a UI control is deferred (see Maintenance notes).

## Git workflow

- Executors typically run in a worktree off `origin/main`; land on `main`
  (no PR gate). Branch name if you create one: `advisor/189-extension-egress-guard`.
- One commit is fine for this change. Conventional-commit subject, e.g.:
  `feat(ext): default-deny cloud-metadata & link-local egress from balaur.http (189)`
  (matches the repo's `feat/fix/docs/refactor(scope): …` log style).
- Do NOT push or open a PR unless the operator instructed it. Before any push,
  `go test ./...` must be green.

## Steps

Order the work so the tree builds between steps: define the deny logic first,
thread the flag, then wire the read, then tests, then docs.

### Step 1: Add the default-deny dialer logic in `vm.go`

Add `"net"` to the imports. Write a small pure helper that decides whether a
resolved IP is in a denied range, and a `net.Dialer` whose `Control` hook
enforces it. The `Control` hook receives the **post-resolution** address as
`address` (`host:port`); split it and parse the IP. This catches the resolved
target even when the URL used a hostname that resolves to a denied IP.

Target shape (produce this logic; exact names/comments your judgment, but keep
the deny set EXACTLY these ranges):

```go
// deniedEgressIP reports whether addr (an already-resolved "host:port" from
// the dialer's Control hook) targets a range balaur.http refuses by default:
// the cloud instance-metadata endpoints and link-local space. Loopback is NOT
// denied — reaching local services is by-design (httpBinding's comment); only
// the credential-bearing metadata/link-local ranges are hardened here.
func deniedEgressIP(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false // a hostname slipped through unresolved; let Dial fail/proceed normally
	}
	// IPv4 link-local 169.254.0.0/16 (incl. the canonical metadata IP
	// 169.254.169.254) and IPv6 link-local fe80::/10.
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// IPv6 cloud-metadata address used by some providers.
	if ip.Equal(net.ParseIP("fd00:ec2::254")) {
		return true
	}
	return false
}
```

> Note: `net.IP.IsLinkLocalUnicast()` returns true for the whole
> `169.254.0.0/16` (which includes `169.254.169.254`) and for `fe80::/10`.
> Using it keeps the deny set canonical and avoids hand-rolled CIDR math.

Then build the client with a guarded dialer. Replace the bare package var so
that, **when the guard is active**, the dialer refuses a denied IP with a
`net`-level error (which `httpClient.Do` returns, so `httpBinding` turns it
into `panic(vm.NewGoError(err))` — a JS-visible error, no crash). When the
guard is **off** (owner opt-out), dial normally.

Recommended: keep two clients — one guarded (default), one plain — selected by
the opt-out bool, so the redirect policy is shared. Sketch:

```go
// egressBlockedErr is returned by the guarded dialer when a handler targets a
// default-denied range; it reaches the JS handler as a normal error.
var egressBlockedErr = errors.New("balaur.http: egress to cloud-metadata / link-local addresses is blocked by default (enable the ext_local_egress owner setting to allow local egress)")

func guardedDialer() *net.Dialer {
	d := &net.Dialer{}
	d.Control = func(network, address string, _ syscall.RawConn) error {
		if deniedEgressIP(address) {
			return egressBlockedErr
		}
		return nil
	}
	return d
}

func extHTTPClient(localEgress bool) *http.Client {
	c := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if !localEgress {
		c.Transport = &http.Transport{DialContext: guardedDialer().DialContext}
	}
	return c
}
```

> Importing `syscall` is fine here (this code is host-side, CGO-off — `syscall`
> is pure Go and does not break `CGO_ENABLED=0`). If you prefer to avoid
> `syscall`, use a custom `DialContext` wrapper that resolves then checks before
> dialing instead of `Control`; either is acceptable as long as the resolved IP
> is checked. The `Control` form is preferred because it inspects the actual
> connect target after DNS.

Add `errors` to imports if you use `errors.New`. Remove the old package-level
`var extHTTPClient = &http.Client{…}` (it becomes the function above). Update
its doc comment to state the new default-deny + opt-out behavior.

**Verify**: `CGO_ENABLED=0 go build ./internal/ext/` → exit 0 (compiles; you
will wire the caller in Step 2–3, so a temporary "declared and not used" on the
new function is expected until then — if so, proceed to Step 2 before
re-running build).

### Step 2: Thread the opt-out bool through `invoke → newVM → httpBinding`

`app` is not available at the dial site, so pass a `bool` down from where it is
(Step 3 reads it). Change the signatures:

- `invoke(ctx context.Context, src, name, tool, argsJSON string, localEgress bool)`
- `newVM(ctx context.Context, src, name string, withHTTP, localEgress bool)`
- `httpBinding(ctx context.Context, vm *goja.Runtime, localEgress bool)`

In `newVM`, pass `localEgress` into `httpBinding(ctx, vm, localEgress)` at the
`withHTTP` branch. In `httpBinding`, replace the `extHTTPClient.Do(req)` call
with `extHTTPClient(localEgress).Do(req)` (build the client once at the top of
the returned closure, not per `Do`, to keep it readable — e.g.
`client := extHTTPClient(localEgress)` before the request is issued).

`extract()` (vm.go:109) calls `newVM(ctx, src, name, false)` — update it to
`newVM(context.Background(), src, name, false, false)`. The `withHTTP=false`
path wires the throwing stub, so the egress bool is irrelevant there; pass
`false`.

**Verify**: `CGO_ENABLED=0 go build ./internal/ext/` → exit 0 (now the new
function is used; no unused-symbol error).

### Step 3: Read the opt-out flag in `extTool` (ext.go) and pass it to `invoke`

In `extTool` (ext.go:130), read the flag where `app` is in scope and pass the
bool into `invoke`. Read it **inside the `Execute` closure** (so a toggle takes
effect on the next invocation without re-`Sync`), matching the nudge exemplar:

```go
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			localEgress := store.GetOwnerSetting(app, "ext_local_egress", "") == "1"
			res, err := invoke(ctx, src, extName, toolName, argsJSON, localEgress)
			store.Audit(app, "extensions", "ext.invoke", extName+"/"+toolName, err == nil, nil)
			return res, err
		},
```

Leave the existing `store.Audit(...)` call exactly as-is — audit stays **after**
the invoke result, unchanged. (`ext.go` already imports `internal/store`.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0. Then
`go test ./internal/ext/` → existing tests still `ok` (the 127.0.0.1 httptest
test must still pass — loopback is not denied).

### Step 4: Add egress tests in `ext_test.go`

Add `"github.com/alexradunet/balaur/internal/store"` to the test imports. Add
three tests, modeled structurally on `TestHTTPBindingWorksInsideHandlers`
(ext_test.go:224) — same `setupDir` / `write` / `Sync` / `Approve` /
`Tools` / `Execute` shape:

1. **`TestMetadataEgressDeniedByDefault`** — write an extension whose handler
   does `balaur.http({url: "http://169.254.169.254/latest/meta-data/"})`.
   `Approve`, `Tools`, `Execute(ctx, "{}")`. Assert `err != nil` (the dial was
   refused) **and** that the test does not crash/hang. Optionally assert the
   error text mentions `egress` or `blocked`. The audit row for this invoke
   should record `allowed = false` (the `Execute` returned an error) — you may
   assert that via `app.FindRecordsByFilter("audit_log", "action = 'ext.invoke'", …)`
   if you want extra coverage, mirroring `TestApprovePinsServesAndAudits`.

   > This dial must fail fast at the `Control` hook (before any network I/O),
   > so the test needs no real metadata endpoint and no `time.Sleep`.

2. **`TestMetadataEgressAllowedWithOptOut`** — same extension, but before
   `Execute` call `store.SetOwnerSetting(app, "ext_local_egress", "1")`. Now the
   guard is off, so the dial is **attempted**. Since `169.254.169.254` is not
   reachable in CI, the handler will get a connection/timeout error rather than
   the *blocked* error — so assert that the returned error (if any) is **not**
   the egress-blocked error (i.e. the guard did not refuse it). The cleanest
   deterministic assertion: point the URL at a **closed loopback port** instead
   (e.g. `http://127.0.0.1:0/` or a port you bind-and-close) and assert the
   guard does not block it — OR, better, prove opt-out lets a normal external
   fetch proceed by pointing at the `httptest` server (loopback) and asserting
   success, since loopback already proceeds regardless. **Recommended concrete
   assertion**: with the flag on, a handler fetching a **link-local** URL
   (`http://169.254.169.254/`) returns an error whose text does **not** contain
   the blocked-egress sentinel — confirming the guard was bypassed and the
   failure is an ordinary dial failure. Keep it deterministic and
   `time.Sleep`-free.

3. **`TestExternalFetchStillWorks`** — reuse the `httptest.NewServer` pattern
   from `TestHTTPBindingWorksInsideHandlers` (binds `127.0.0.1`, which is
   loopback, **not** denied) and assert the handler still gets `status 200` and
   the body, with the guard at its **default** (flag unset). This proves the
   default-deny did not regress normal/loopback fetches.

Each test asserts on the **returned error** (and/or output string), never on a
panic escaping — `invoke`'s `defer recover()` guarantees a denied dial returns
as an error, not a crash.

**Verify**: `go test ./internal/ext/` → `ok`, including the 3 new tests. Run
`go test -run 'Egress|ExternalFetch' ./internal/ext/ -v` to see them pass by
name.

### Step 5: Update `internal/self/knowledge.md`

Keep the self-description honest: extend the extensions paragraph
(knowledge.md:223–229) with one clause noting the egress guard. Change the
`handlers may call balaur.http` sentence so it reads (additive, one clause):

```
balaur.registerTool({name, description, parameters, handler}); handlers
may call balaur.http, which by default refuses egress to cloud-metadata
and link-local addresses (169.254.0.0/16, fe80::/10, fd00:ec2::254) —
loopback/local services stay reachable, and an owner can re-enable
link-local egress via the ext_local_egress setting.
```

Match the surrounding prose style (it is wrapped prose, not a list of bullets).
Do not restructure the paragraph; only weave in the clause.

**Verify**: `gofmt -l .` → empty (knowledge.md is not Go, unaffected);
`git diff internal/self/knowledge.md` shows only the added clause.

### Step 6: Full validation sweep

**Verify** (all must pass):
- `gofmt -l .` → empty
- `go vet ./...` → exit 0
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go test ./internal/ext/` → `ok`
- `go test ./...` → all `ok`
- `git diff --check` → exit 0

## Test plan

- New tests in `internal/ext/ext_test.go`:
  - `TestMetadataEgressDeniedByDefault` — handler fetching `169.254.169.254`
    is refused by default; `Execute` returns an error; no crash/hang.
  - `TestMetadataEgressAllowedWithOptOut` — with `ext_local_egress=1` the guard
    no longer refuses the dial (error, if any, is an ordinary dial failure, not
    the blocked-egress sentinel).
  - `TestExternalFetchStillWorks` — a normal (loopback `httptest`) fetch still
    succeeds at the default guard setting — regression guard.
- Structural pattern to copy: `TestHTTPBindingWorksInsideHandlers`
  (ext_test.go:224) for the server+handler+execute shape;
  `TestApprovePinsServesAndAudits` (ext_test.go:83) for audit-row assertions if
  used.
- No assertion framework, no `time.Sleep`. The denied dial fails at the
  `Control` hook (no network wait); the opt-out test asserts on error *content*,
  not timing.
- Verification: `go test ./internal/ext/` → all pass, including the 3 new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `gofmt -l .` prints nothing
- [ ] `go test ./internal/ext/` is `ok`, with `TestMetadataEgressDeniedByDefault`,
      `TestMetadataEgressAllowedWithOptOut`, `TestExternalFetchStillWorks` present
      and passing
- [ ] `go test ./...` all `ok` (no other package regressed)
- [ ] A handler fetching `169.254.169.254` is refused by default and the
      refusal reaches JS as a returned error (not a panic/crash) — proven by
      `TestMetadataEgressDeniedByDefault`
- [ ] `git diff --check` exits 0
- [ ] No files outside the in-scope list are modified (`git status`): only
      `internal/ext/vm.go`, `internal/ext/ext.go`, `internal/ext/ext_test.go`,
      `internal/self/knowledge.md`
- [ ] `internal/self/knowledge.md` extensions paragraph documents the guard
- [ ] `plans/README.md` status row for plan 189 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The "Current state" excerpts do not match the live code (e.g. `httpBinding`
  no longer issues through `extHTTPClient`, or `invoke` already receives `app`).
  The codebase drifted; re-confirm before changing the design.
- **Default-denying all link-local breaks an existing tested extension
  behavior.** Per the brief: if a currently-passing test (or a real bundled
  extension) depends on reaching a `169.254.0.0/16` / `fe80::/10` address,
  **narrow the default-deny to the canonical metadata IP(s) only**
  (`169.254.169.254` and `fd00:ec2::254`) instead of the full link-local
  ranges, and **report the narrowing** in your status update. (Note:
  `TestHTTPBindingWorksInsideHandlers` uses `127.0.0.1` loopback, which is NOT
  link-local — it should NOT trigger this. If it does, something else changed.)
- Adding the `Control` dialer requires `syscall` and that breaks
  `CGO_ENABLED=0 go build` (it should NOT — `syscall` is pure Go). If it
  somehow does, switch to the custom-`DialContext` form that resolves-then-checks
  and report it.
- Any verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (a migration, the
  store package, a UI file) — re-read Scope; the read-only flag consumption
  needs none of these.

## Maintenance notes

For the human/agent who owns this after it lands:

- **The deny set is the security contract.** It denies link-local
  (`169.254.0.0/16` via `IsLinkLocalUnicast`, `fe80::/10`) + the IPv6
  metadata address `fd00:ec2::254`. It intentionally does **not** deny
  loopback or RFC1918 private ranges — local/LAN reach is by-design. If a future
  threat model wants to also block private/loopback ranges, that is a *separate,
  larger* policy decision (it would break local-service extensions) — do it as
  its own plan with its own opt-out, not by quietly widening this set.
- **The flag has no UI yet.** `ext_local_egress` is read-only-consumed here; an
  owner sets it via the PocketBase admin or `SetOwnerSetting`. A Settings/Models
  toggle is a deferred follow-up — when added, reuse the nudge-toggle pattern in
  `internal/feature/settingscards/settingsfocus.go` and write through
  `store.SetOwnerSetting(app, "ext_local_egress", "1"|"0")`.
- **The guard is at the dialer, after DNS** (the `Control` hook sees the
  resolved connect address), so a hostname that resolves to `169.254.169.254`
  is also blocked — verify any future refactor preserves post-resolution
  checking (a check on the raw URL host would miss DNS-rebinding to a denied IP).
- A reviewer should scrutinize: (1) that the denied dial surfaces as a JS error,
  not a panic that escapes `invoke`'s recover; (2) that `extract`'s
  `newVM(..., false, false)` call still loads side-effect-free; (3) that the
  audit row for a blocked invoke records `allowed=false`.
