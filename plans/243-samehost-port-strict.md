# Plan 243: Make the Origin-fallback host comparison port-sensitive in guardLocalUI

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, add a status row for this plan to
> `plans/README.md` (no row for 243 exists yet): append
> `| 243 | ... | ... |` to the `| Plan | Builds | Status |` table under the
> "## Phase-1 & follow-up builds" heading, after the row for 234 — unless a
> reviewer dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/web/web.go internal/web/handlers_test.go .tours/07-the-web-gateway.tour`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

`guardLocalUI` in `internal/web/web.go` is Balaur's CSRF defence for
state-changing requests. Its primary check (the browser-set `Sec-Fetch-Site`
header) is sound, but its fallback — used by legacy browsers/clients that send
an `Origin` header without fetch metadata — compares hosts with `sameHost`,
which **strips the port from both sides** before comparing. That means a page
served from `http://localhost:3000` (any local dev server, another local app,
a malicious page a local process serves) passes the fallback check for POSTs
to Balaur on `localhost:8090` — including `POST /ui/settings/messenger-token`
(`internal/web/web.go:227`), which would let a cross-port page enable the
remote messenger gateway with an attacker-known token. Modern browsers are
protected upstream by the `Sec-Fetch-Site` default-reject, so this is
defense-in-depth for the fallback path only — but legitimate same-origin
requests always carry the full matching `host:port`, so the port-stripping is
a pure, benefit-free weakening. The fix: compare `host:port` to `host:port`,
normalizing scheme-default ports (http→80, https→443) so `http://localhost`
still matches request host `localhost:80` but never `localhost:8090`.

## Current state

Files and roles:

- `internal/web/web.go` — the Datastar web gateway; `guardLocalUI` (the
  host/CSRF guard bound before all routes), `isAllowedHost`, and `sameHost`
  all live here.
- `internal/web/handlers_test.go` — gateway HTTP tests using PocketBase's
  `tests.ApiScenario`; `TestOriginGuard` (line 419) covers the guard.
- `.tours/07-the-web-gateway.tour` — CodeTour anchoring `internal/web/web.go`
  lines 64 (`guardLocalUI`) and 138 (`Register`); this plan's edits shift
  line 138.

The guard's fallback branch — `sameHost` is called in exactly one place
(`internal/web/web.go:97`):

```go
// internal/web/web.go:76-103 (at 077318a)
	if e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead {
		// Sec-Fetch-Site is browser-set and unspoofable: a cross-site page
		// cannot forge it. When present it is authoritative — only same-origin
		// and none (top-level user navigation) are trusted.
		switch e.Request.Header.Get("Sec-Fetch-Site") {
		case "same-origin", "none":
			return e.Next()
		case "":
			// No fetch-metadata (curl, CLI, older clients): fall through to the
			// Origin check below.
		default:
			return e.ForbiddenError("cross-site request rejected", nil)
		}
		// Origin: null is attacker-influenced (opaque/sandboxed origins emit it),
		// so it is a rejection, not a trusted-absent. A truly absent Origin
		// (curl, CLI) still passes.
		if origin := e.Request.Header.Get("Origin"); origin != "" {
			if origin == "null" {
				return e.ForbiddenError("cross-origin request rejected", nil)
			}
			u, err := url.Parse(origin)
			if err != nil || !sameHost(u.Host, e.Request.Host) {
				return e.ForbiddenError("cross-origin request rejected", nil)
			}
		}
	}
	return e.Next()
}
```

The defective comparison — ports are stripped from BOTH sides:

```go
// internal/web/web.go:124-135 (at 077318a)
func sameHost(origin, request string) bool {
	// Strip ports for comparison
	origHost := origin
	if h, _, err := net.SplitHostPort(origin); err == nil {
		origHost = h
	}
	reqHost := request
	if h, _, err := net.SplitHostPort(request); err == nil {
		reqHost = h
	}
	return origHost == reqHost
}
```

The guard's doc comment describes the fallback (this sentence changes too):

```go
// internal/web/web.go:59-63 (at 077318a)
//   - On state-changing methods: the browser-set, unspoofable Sec-Fetch-Site
//     header is authoritative when present (only same-origin/none pass);
//     otherwise an Origin header, when present, must match the request Host,
//     and the attacker-influenced value "null" (opaque/sandboxed origins) is a
//     rejection. Absent both headers (curl, CLI, same-origin GET) passes.
```

The loopback/allow-list check stays untouched (out of scope, quoted so you
recognize the boundary):

```go
// internal/web/web.go:105-122 (at 077318a)
func isAllowedHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	allowed := os.Getenv("BALAUR_ALLOWED_HOSTS")
	if allowed == "" {
		return false
	}
	for h := range strings.SplitSeq(allowed, ",") {
		if strings.TrimSpace(h) == host {
			return true
		}
	}
	return false
}
```

Existing test coverage. `newWebApp` allows the `httptest` default host:

```go
// internal/web/handlers_test.go:29-33 (at 077318a)
func newWebApp(t testing.TB) *tests.TestApp {
	t.Helper()
	// httptest requests default to Host "example.com"; allow it for tests
	// only — production allows loopback + BALAUR_ALLOWED_HOSTS.
	t.Setenv("BALAUR_ALLOWED_HOSTS", "example.com")
```

`TestOriginGuard`'s table cases all POST to a fixed relative URL (so the
request host is the `httptest.NewRequest` default `example.com`, portless):

```go
// internal/web/handlers_test.go:461-471 (at 077318a) — two of six cases
		{
			Name:            "cross-origin Origin rejected",
			Headers:         csrf(map[string]string{"Origin": "http://evil.example"}),
			ExpectedStatus:  403,
			ExpectedContent: []string{"Cross-origin request rejected."},
		},
		{
			Name:           "same-origin Origin allowed",
			Headers:        csrf(map[string]string{"Origin": "http://example.com"}),
			ExpectedStatus: 204,
		},
```

```go
// internal/web/handlers_test.go:483-489 (at 077318a)
	for _, sc := range cases {
		sc.Method = "POST"
		sc.URL = "/ui/panel/collapse"
		sc.Body = strings.NewReader("on=1")
		sc.TestAppFactory = newWebApp
		t.Run(sc.Name, func(t *testing.T) { sc.Test(t) })
	}
```

Mechanism you will rely on in Step 3: PocketBase's `tests.ApiScenario` builds
its request with `httptest.NewRequest(scenario.Method, scenario.URL, ...)`
(verified in `pocketbase@v0.39.3/tests/api.go:228`) and does NOT special-case
a `Host` header — so a **relative** URL yields `req.Host == "example.com"`,
and an **absolute** URL like `http://example.com:8090/ui/panel/collapse`
yields `req.Host == "example.com:8090"` (standard `httptest.NewRequest`
behavior: an absolute target sets the request Host from the URL).
`guardLocalUI` strips the port from `e.Request.Host` (`web.go:69-72`) before
calling `isAllowedHost` (which itself does no port handling), so
`example.com:8090` still passes the host allow-list in tests.

Repo conventions that apply here:

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in library
  code (not much error plumbing here — `sameHost` stays a pure predicate).
- Tests: standard `testing` package, table-driven where it helps; no assertion
  frameworks; gateway HTTP behavior uses `tests.ApiScenario` with
  `TestAppFactory: newWebApp` — model new cases on the existing
  `TestOriginGuard` cases quoted above.
- No global mutable state; no `fmt.Print*` in service code.
- `.tours/` are maintained artifacts: `tours_test.go` only catches missing
  files/out-of-range lines, but the convention is to fix shifted anchors in
  the same change — Step 4 does this.
- `internal/self/knowledge.md` update is **NOT needed**: this hardens an
  internal comparison inside an existing guard; no user-visible architecture
  or capability changes (the guard's described behavior — "Origin must match
  the request Host" — becomes more true, not different).
- KISS/YAGNI: smallest correct change — do not add config knobs, do not touch
  the `Sec-Fetch-Site` branch, do not generalize beyond http/https.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0 |
| Targeted tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestSameHost|TestOriginGuard' -count=1` | ok, exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours lint (after Step 4) | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |

Note: the host `/tmp` is a small tmpfs — the Go linker OOMs there, hence the
`TMPDIR=$HOME/.cache/go-tmp` prefix on every `go test`.

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before Step 1 for the
  repo's Go idioms (error style, table-driven tests, modern stdlib).

## Scope

**In scope** (the only files you should modify):

- `internal/web/web.go` — `sameHost` (rewrite), its one call site at line 97,
  and the guard doc-comment sentence at lines 61–63.
- `internal/web/handlers_test.go` — new `TestSameHost` unit table; two new
  `TestOriginGuard` cases; the case-loop URL default.
- `.tours/07-the-web-gateway.tour` — update the line anchor for `Register`
  (and `guardLocalUI` if it shifts).

**Out of scope** (do NOT touch, even though they look related):

- `isAllowedHost` (`internal/web/web.go:105-122`) — the DNS-rebinding /
  allow-list check deliberately compares bare hosts (its env var is documented
  as "no port" in `docs/netbird.md`); changing it breaks LAN/NetBird setups.
- The `Sec-Fetch-Site` primary check (`web.go:80-88`) — already correct.
- `internal/web/messenger.go` / `messenger_settings.go` — the messenger
  endpoint's own Host check uses `isAllowedHost`, not `sameHost`.
- `docs/netbird.md`, `internal/self/knowledge.md` — no behavior they describe
  changes.

## Git workflow

- You run in an isolated git worktree branched from `origin/main`; branch
  name: `advisor/243-samehost-port-strict`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`);
  one commit per logical unit. Suggested single commit:
  `fix(web): compare Origin host:port in guardLocalUI fallback`.
- Stage only your own files with explicit pathspecs
  (`git add internal/web/web.go internal/web/handlers_test.go .tours/07-the-web-gateway.tour`)
  — the main checkout is shared by parallel agents.
- **NEVER push**; the reviewer merges.

## Steps

### Step 1: Rewrite `sameHost` to compare host:port with scheme-default normalization

In `internal/web/web.go`, replace the `sameHost` function (currently lines
124–135, quoted in "Current state") with a port-sensitive version. The caller
at line 97 already has the parsed `*url.URL` in hand — change the signature to
take it so the origin's scheme is available for default-port normalization.
Target shape:

```go
// sameHost reports whether the Origin URL and the request Host name the same
// host:port. A missing port normalizes to the origin scheme's default
// (http→80, https→443) — browsers omit default ports from both Origin and
// Host — so "http://localhost" matches request host "localhost:80", but a
// page on localhost:3000 never matches a request to localhost:8090.
func sameHost(origin *url.URL, request string) bool {
	oh, op := splitHostOptionalPort(origin.Host)
	rh, rp := splitHostOptionalPort(request)
	def := schemeDefaultPort(origin.Scheme)
	if op == "" {
		op = def
	}
	if rp == "" {
		rp = def
	}
	return oh == rh && op != "" && op == rp
}

// splitHostOptionalPort splits host[:port], tolerating a missing port
// (net.SplitHostPort errors on that). Brackets around a bare IPv6 literal
// are stripped so "[::1]" and "[::1]:80" compare on the same host form.
func splitHostOptionalPort(hostport string) (host, port string) {
	if h, p, err := net.SplitHostPort(hostport); err == nil {
		return h, p
	}
	return strings.Trim(hostport, "[]"), ""
}

// schemeDefaultPort returns the default port for http/https origins; other
// schemes get none, so a portless non-http(s) origin never matches.
func schemeDefaultPort(scheme string) string {
	switch scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	}
	return ""
}
```

Notes:

- `url.Parse` lowercases the scheme, so no case folding is needed in
  `schemeDefaultPort`.
- The `op != ""` term makes non-http(s) origins without an explicit port fail
  closed instead of matching a portless request host.

Update the single call site (`internal/web/web.go:97`) from:

```go
			if err != nil || !sameHost(u.Host, e.Request.Host) {
```

to:

```go
			if err != nil || !sameHost(u, e.Request.Host) {
```

Update the guard doc comment (`web.go:61`): change

```go
//     otherwise an Origin header, when present, must match the request Host,
```

to

```go
//     otherwise an Origin header, when present, must match the request Host —
//     port included (scheme-default ports normalized: http→80, https→443) —
```

Keep everything else in `guardLocalUI` and all of `isAllowedHost` byte-for-byte
unchanged.

**Verify**: `CGO_ENABLED=0 go build ./... && go vet ./... && gofmt -l .`
→ exit 0, no vet errors, empty gofmt output.

**Verify**: `grep -c "Strip ports for comparison" internal/web/web.go` → `0`.

### Step 2: Add a table-driven `TestSameHost` unit test

In `internal/web/handlers_test.go` (package `web`, so unexported `sameHost` is
reachable; `net/url` is already imported), add after `TestOriginGuard`:

```go
// TestSameHost covers the port-sensitive Origin↔Host comparison used by the
// guardLocalUI legacy-browser fallback (no Sec-Fetch-Site header).
func TestSameHost(t *testing.T) {
	cases := []struct {
		name   string
		origin string // full Origin header value
		host   string // request Host
		want   bool
	}{
		{"cross-port rejected", "http://localhost:3000", "localhost:8090", false},
		{"same host:port allowed", "http://localhost:8090", "localhost:8090", true},
		{"http default port matches :80", "http://localhost", "localhost:80", true},
		{"https default port matches :443", "https://balaur", "balaur:443", true},
		{"no port both sides allowed", "http://example.com", "example.com", true},
		{"https origin vs :80 host rejected", "https://balaur", "balaur:80", false},
		{"different host rejected", "http://evil.example", "localhost:8090", false},
		{"ipv6 same host:port allowed", "http://[::1]:8090", "[::1]:8090", true},
		{"ipv6 bare both sides allowed", "http://[::1]", "[::1]", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.origin)
			if err != nil {
				t.Fatalf("parse origin %q: %v", tc.origin, err)
			}
			if got := sameHost(u, tc.host); got != tc.want {
				t.Errorf("sameHost(%q, %q) = %v; want %v", tc.origin, tc.host, got, tc.want)
			}
		})
	}
}
```

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run TestSameHost -count=1 -v`
→ all 9 subtests PASS.

### Step 3: Extend `TestOriginGuard` with cross-port integration cases

Still in `internal/web/handlers_test.go`, inside `TestOriginGuard`:

1. Append two cases to the existing `cases := []tests.ApiScenario{...}` slice
   (after the "no CSRF headers allowed (CLI/curl/harness)" case). They use
   **absolute** URLs so `httptest.NewRequest` sets `req.Host` to
   `example.com:8090` (see the mechanism note in "Current state"):

```go
		{
			Name:            "same-host cross-port Origin rejected",
			URL:             "http://example.com:8090/ui/panel/collapse",
			Headers:         csrf(map[string]string{"Origin": "http://example.com:3000"}),
			ExpectedStatus:  403,
			ExpectedContent: []string{"Cross-origin request rejected."},
		},
		{
			Name:           "same host:port Origin allowed",
			URL:            "http://example.com:8090/ui/panel/collapse",
			Headers:        csrf(map[string]string{"Origin": "http://example.com:8090"}),
			ExpectedStatus: 204,
		},
```

2. Change the case loop so it only fills the URL when a case did not set one
   — replace (quoted at handlers_test.go:483-489 in "Current state"):

```go
	for _, sc := range cases {
		sc.Method = "POST"
		sc.URL = "/ui/panel/collapse"
```

with:

```go
	for _, sc := range cases {
		sc.Method = "POST"
		if sc.URL == "" {
			sc.URL = "/ui/panel/collapse"
		}
```

Do not modify the six existing cases — in particular
"same-origin Origin allowed" (`Origin: http://example.com` against portless
request host `example.com`) must keep passing: both sides normalize to
`example.com:80` under the new `sameHost`.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run TestOriginGuard -count=1 -v`
→ all 8 subtests PASS (6 pre-existing + 2 new).

### Step 4: Fix the shifted tour anchor in `07-the-web-gateway.tour`

Step 1 grows `web.go` by roughly 18 lines below `guardLocalUI`, shifting
`func Register(` from line 138. `.tours/07-the-web-gateway.tour` step 0
anchors `internal/web/web.go` at `"line": 138` (the `Register` excerpt) and
step 1 at `"line": 64` (`guardLocalUI`).

1. Find the new anchor lines:
   `grep -n "^func Register(" internal/web/web.go` and
   `grep -n "^func guardLocalUI(" internal/web/web.go`.
2. In `.tours/07-the-web-gateway.tour`, set each step's `"line"` value to the
   matching grep result (step with the `Register` code block → the `Register`
   line; step with the `guardLocalUI` code block → the `guardLocalUI` line —
   64 should be unchanged if the doc-comment edit in Step 1 kept the same line
   count; update it if not).
3. The tour prose ("the `Origin` header is the fallback: it must match
   `Host`") remains true — do not rewrite it.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok.

**Verify**: `grep -n '"line"' .tours/07-the-web-gateway.tour | head -2` →
values equal the grep results from sub-step 1.

### Step 5: Full gates and commit

Run, in order:

1. `gofmt -l .` → empty
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0 (this is the
   merge gate; `make test` is cached — use this uncached form)

Then commit with explicit pathspecs:

```
git add internal/web/web.go internal/web/handlers_test.go .tours/07-the-web-gateway.tour
git commit -m "fix(web): compare Origin host:port in guardLocalUI fallback"
```

**Verify**: `git status --porcelain` → only shows untracked/unstaged files you
did not create; nothing of yours outside the three in-scope files is staged or
committed. `git show --stat HEAD` lists exactly the three in-scope files.

## Test plan

- New unit test `TestSameHost` in `internal/web/handlers_test.go` (Step 2),
  table-driven, 9 cases: the vulnerability case
  (`http://localhost:3000` vs `localhost:8090` → false), exact match,
  http→80 and https→443 default-port normalization, portless-both-sides,
  scheme-default mismatch (`https://balaur` vs `balaur:80` → false),
  different host, and two IPv6 forms.
- Two new `tests.ApiScenario` cases in `TestOriginGuard` (Step 3) proving the
  guard end-to-end: cross-port Origin → 403 with
  `"Cross-origin request rejected."`; same host:port Origin → 204. Model on
  the existing cases quoted in "Current state"; factory stays `newWebApp`.
- Regression: the six pre-existing `TestOriginGuard` cases must pass
  unchanged (especially "same-origin Origin allowed" and
  "no CSRF headers allowed").
- Verification:
  `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestSameHost|TestOriginGuard' -count=1`
  → ok, 17 subtests total pass; then the full gate
  `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` prints nothing; `go vet ./...` exits 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` exits 0 with no
      output; `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestSameHost|TestOriginGuard' -count=1 -v`
      → PASS, including subtests `cross-port_rejected` and
      `same-host_cross-port_Origin_rejected`.
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` exits 0.
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` exits 0.
- [ ] `grep -c "Strip ports for comparison" internal/web/web.go` → `0`
      (old port-stripping comment gone).
- [ ] `grep -c "func sameHost(origin \*url.URL" internal/web/web.go` → `1`
      (new signature in place).
- [ ] `git diff 077318a..HEAD --stat -- internal/web/messenger.go internal/web/messenger_settings.go`
      shows no changes from this branch (out-of-scope files untouched).
- [ ] `git show --stat HEAD` lists only files from the in-scope list
      (`plans/README.md` status-row update excepted, per executor
      instructions).
- [ ] `plans/README.md` has a row for 243 in the `| Plan | Builds | Status |`
      table under "## Phase-1 & follow-up builds" — no row exists at plan time;
      add it after the 234 row (unless the reviewer maintains the index).

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows changes to `internal/web/web.go`,
  `internal/web/handlers_test.go`, or `.tours/07-the-web-gateway.tour` since
  `077318a` and the "Current state" excerpts no longer match the live code.
- `grep -rn "sameHost(" internal/ main.go` (run it) returns anything
  beyond the definition in `web.go` and the single call at the Origin-fallback
  site — the plan assumes exactly one caller.
- The pre-existing `TestOriginGuard` case "same-origin Origin allowed" starts
  failing after Step 1 — that would mean the assumption "ApiScenario relative
  URLs yield request host `example.com` (portless)" is false, and the
  normalization needs rethink, not a quick patch.
- The new "same host:port Origin allowed" case returns 403 — that would mean
  the assumption "an absolute ApiScenario URL sets `req.Host` to
  `host:port`" is false for this PocketBase version.
- The fix appears to require touching `isAllowedHost`, the `Sec-Fetch-Site`
  branch, `messenger.go`/`messenger_settings.go`, or any file outside the
  in-scope list.
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

- **Residual (accepted) edge**: when the request Host is portless, the missing
  port is filled from the *origin's* scheme default. A page on
  `https://host` (443) could therefore still match a portless request host for
  a Balaur instance actually serving on 80 behind a proxy on the same
  hostname. This requires an attacker TLS server on the same host name AND a
  legacy browser without `Sec-Fetch-Site` — accepted as out of KISS scope;
  revisit only if Balaur ever officially supports non-loopback TLS serving.
- **Reviewer focus**: confirm `isAllowedHost` is byte-identical (it must stay
  port-insensitive — `BALAUR_ALLOWED_HOSTS` is documented as "no port" in
  `docs/netbird.md:54`), and that the guard's `Sec-Fetch-Site` branch is
  untouched.
- Observed but deliberately not fixed (out of scope): the first
  `TestOriginGuard` scenario sets a `Host` header via `Headers`, but
  PocketBase's `ApiScenario` applies headers with `req.Header.Set`, which does
  not change `req.Host` — that header is inert and the scenario actually runs
  against the default `example.com` host. Harmless; flag it to the owner
  rather than fixing here.
- If a future change adds more callers of `sameHost`, they must pass the
  parsed origin `*url.URL` (the scheme drives default-port normalization) —
  do not reintroduce a string-host-only comparison.
