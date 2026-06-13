# Plan 042: Web hardening bundle — escape card errors, drop example.com from the host guard, add response headers, cap card param sizes

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat dd9e60b..HEAD -- internal/web/cards.go internal/web/web.go internal/cards/cards.go internal/web/handlers_test.go internal/web/cards_test.go internal/cards/cards_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S–M
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `dd9e60b`, 2026-06-12

## Why this matters

Balaur is a loopback-first personal web app, but four small gaps weaken its
browser-facing defences. (1) `/ui/cards/{type}` echoes card-validation errors
into HTML **unescaped**, and those errors embed the raw user/model-supplied
parameter value — a crafted URL or a model-composed card parameter can inject
markup into the owner's page. (2) The DNS-rebinding host guard hardcodes
`example.com` as an allowed Host — a test convenience living in production
code; an attacker who can answer DNS for `example.com` on a hostile network
(plain-HTTP page → rebind to 127.0.0.1) defeats the guard's whole purpose.
(3) No hardening response headers (`X-Content-Type-Options`,
`X-Frame-Options`, `Referrer-Policy`) are set. (4) Free-string card params
(`kind`, `query`, `month`) have no length cap, so a model tool-call can bloat
board records without bound.

## Current state

Relevant files:

- `internal/web/cards.go` — `/ui/cards/{type}` handler; the unescaped error
  write is at line 73.
- `internal/web/web.go` — router registration and `guardLocalUI` /
  `isAllowedHost` (the `example.com` allowance is line 126).
- `internal/cards/cards.go` — the typed card registry; `Validate` is the
  param-cleaning chokepoint (string params pass through uncapped at line 277).
- `internal/web/knowledge.go:169-174` — the **exemplar** for escaped error
  strips (`cardError`).
- `internal/web/handlers_test.go` — `newWebApp(t)` test factory used by all
  web tests (PocketBase `tests.TestApp` + `Register`).

`internal/web/cards.go:68-75` today:

```go
	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		// Validation error: render a card-note-error strip, HTTP 200.
		e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		fmt.Fprintf(e.Response, `<div class="card-note card-note-error" id="ucard-%s">%s</div>`, typ, err.Error())
		return nil
	}
```

The error string can embed a raw param value — `internal/cards/cards.go:259-263`:

```go
		// Enum check.
		if len(ps.Enum) > 0 {
			if !enumContains(ps.Enum, v) {
				return nil, fmt.Errorf("param %q must be one of [%s], got %q",
					ps.Name, strings.Join(ps.Enum, ", "), v)
			}
		}
```

(`%q` escapes Go-string style; `<` and `>` pass through untouched.)

The exemplar pattern, `internal/web/knowledge.go:169-174`:

```go
func (h *handlers) cardError(e *core.RequestEvent, err error) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.WriteHeader(http.StatusUnprocessableEntity)
	fmt.Fprintf(e.Response, `<div class="card-note card-note-error">%s</div>`, html.EscapeString(err.Error()))
	return nil
}
```

`internal/web/web.go:125-142` today (`example.com` at line 126):

```go
func isAllowedHost(host string) bool {
	if host == "localhost" || host == "example.com" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	allowed := os.Getenv("BALAUR_ALLOWED_HOSTS")
	if allowed == "" {
		return false
	}
	...
```

`example.com` exists only because PocketBase's `ApiScenario`/`httptest`
requests default to `Host: example.com`. No production caller needs it.
The web tests do **not** call `t.Parallel()` (verified), so `t.Setenv` is
safe in the shared factory.

The guard is bound router-wide at `internal/web/web.go:166-167`:

```go
	// Bind the origin/host guard first, before any route registration.
	se.Router.BindFunc(guardLocalUI)
```

`guardLocalUI` skips PocketBase's own surfaces (`/api/`, `/_`) at
`web.go:103-106` — the new headers middleware must skip the same paths
(PocketBase manages its own headers).

`internal/cards/cards.go:266-277` (numeric params are clamped; everything
else passes through unchanged):

```go
		// Numeric clamping for well-known numeric params — fall back silently
		// on parse errors.
		switch ps.Name {
		case "limit":
			out[ps.Name] = clampInt(v, 1, 50)
			continue
		case "days":
			out[ps.Name] = clampInt(v, 1, 366)
			continue
		}

		out[ps.Name] = v
```

Repo conventions that apply:

- Standard Go, `gofmt` is law; errors wrapped with `fmt.Errorf("doing x: %w", err)`.
- Tests: standard `testing`, table-driven where it helps; web tests go
  through `newWebApp` + PocketBase `tests.ApiScenario` or direct
  `httptest` requests — model new tests on existing ones in
  `internal/web/cards_test.go` and `handlers_test.go`.
- Comments explain intent/constraints, never narrate code.
- **No CSP in this plan**: templates use inline `<script>` blocks
  (`web/templates/home.html:6,128`, `head-chat.html:6,100`) and
  `hx-on:`/`onclick` inline handlers — a CSP would break them. Deferred;
  see Maintenance notes.

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./internal/web/ ./internal/cards/` | ok, all pass |
| All tests | `go test ./...`                  | ok (all ~25 packages) |
| Vet       | `go vet ./...`                   | exit 0, silent      |
| Format    | `gofmt -l .`                     | no output           |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify):
- `internal/web/cards.go`
- `internal/web/web.go`
- `internal/cards/cards.go`
- `internal/web/handlers_test.go` (the `newWebApp` factory + new guard/header tests)
- `internal/web/cards_test.go` (new escape test)
- `internal/cards/cards_test.go` (new cap test)

**Out of scope** (do NOT touch, even though they look related):
- Any Content-Security-Policy header — breaks inline scripts; deferred.
- `internal/web/knowledge.go` — already correct; it is the exemplar.
- `guardLocalUI`'s Origin-check logic and `BALAUR_ALLOWED_HOSTS` parsing —
  only the `example.com` literal goes.
- Template files, `web/static/*`.
- The PocketBase `/api/` and `/_` surfaces.

## Git workflow

- Branch: `advisor/042-web-hardening`
- Conventional commits, e.g. `fix(web): escape card validation errors; drop example.com from host guard`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Escape the card validation error

In `internal/web/cards.go`, change line 73 to escape the error text,
matching the `cardError` exemplar:

```go
		fmt.Fprintf(e.Response, `<div class="card-note card-note-error" id="ucard-%s">%s</div>`, typ, html.EscapeString(err.Error()))
```

Add `"html"` to the import block (keep goimports grouping: stdlib first).

**Verify**: `go build ./internal/web/` → exit 0.

### Step 2: Test the escape

In `internal/web/cards_test.go`, add a test that requests
`/ui/cards/quests?status=<img src=x onerror=x>` through the existing test
harness (follow the request style already used in that file) and asserts:

- response body contains `&lt;img` (escaped),
- response body does NOT contain `<img` (raw).

**Verify**: `go test ./internal/web/ -run TestUiCard` → ok, including the new test.

### Step 3: Remove example.com from the host guard; shim the tests

In `internal/web/web.go:126`, change:

```go
	if host == "localhost" || host == "example.com" {
```

to:

```go
	if host == "localhost" {
```

In `internal/web/handlers_test.go`, inside `newWebApp` (after `t.Helper()`),
add:

```go
	// httptest requests default to Host "example.com"; allow it for tests
	// only — production allows loopback + BALAUR_ALLOWED_HOSTS.
	t.Setenv("BALAUR_ALLOWED_HOSTS", "example.com")
```

Note `newWebApp` takes `testing.TB`; `t.Setenv` exists on `testing.TB` in
this Go version. If the compiler disagrees, change nothing else — STOP and
report.

**Verify**: `go test ./internal/web/` → ok, all pass (every web test routes
through `newWebApp`).

### Step 4: Guard regression test

Add `TestGuardRejectsNonLoopbackHost` in `internal/web/handlers_test.go`:
build a request with `req.Host = "evil.test"` against a `newWebApp` app
(no `BALAUR_ALLOWED_HOSTS` entry for it — note the factory now sets it to
`example.com` only) and assert HTTP 403. Also assert a request with
`req.Host = "127.0.0.1:8090"` passes (status != 403). Model the
request-building on whichever existing test in the file drives requests
most directly.

**Verify**: `go test ./internal/web/ -run TestGuard` → ok.

### Step 5: Hardening headers middleware

In `internal/web/web.go`, add a `BindFunc` immediately after the
`guardLocalUI` binding (line 167):

```go
	// Hardening headers on Balaur's own surfaces. PocketBase's /api and /_
	// manage their own; CSP is deferred — templates still use inline scripts.
	se.Router.BindFunc(func(e *core.RequestEvent) error {
		p := e.Request.URL.Path
		if !strings.HasPrefix(p, "/api/") && !strings.HasPrefix(p, "/_") {
			h := e.Response.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "same-origin")
		}
		return e.Next()
	})
```

Add a test asserting `GET /` carries all three headers.

**Verify**: `go test ./internal/web/` → ok.

### Step 6: Cap free-string card params

In `internal/cards/cards.go`, after the numeric-clamp switch (line 277,
before `out[ps.Name] = v`), truncate long values:

```go
		// Free-string params are model-composable; cap them so a tool call
		// cannot bloat stored board JSON. Truncation, not rejection — cards
		// must stay forgiving.
		if len(v) > maxParamLen {
			v = v[:maxParamLen]
		}
		out[ps.Name] = v
```

with `const maxParamLen = 256` near the top of the file (after the
existing consts/types, with a one-line comment). Note: byte-truncation may
split a UTF-8 rune; that is acceptable here (values are lookup keys, a
broken trailing rune just matches nothing) — do not add rune handling.

Add a table-driven case to `internal/cards/cards_test.go`: a `kind` param
of 1000 chars comes back 256 long; a 10-char value is unchanged.

**Verify**: `go test ./internal/cards/` → ok.

### Step 7: Full gate

**Verify**: `gofmt -l .` → empty; `go vet ./...` → silent;
`go test ./...` → all ok; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

- New: escape test (Step 2), guard rejection + loopback-pass test (Step 4),
  headers-present test (Step 5), param-cap table case (Step 6).
- Pattern: existing tests in `internal/web/cards_test.go` (card requests)
  and `internal/cards/cards_test.go` (Validate tables).
- Verification: `go test ./...` → all pass including the 4+ new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n 'example.com' internal/web/web.go` → no matches
- [ ] `grep -c 'html.EscapeString' internal/web/cards.go` → ≥ 1
- [ ] `grep -n 'X-Content-Type-Options' internal/web/web.go` → 1 match
- [ ] `grep -n 'maxParamLen' internal/cards/cards.go` → ≥ 2 matches
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l .` empty, `go vet ./...` silent, `CGO_ENABLED=0 go build ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The code at the cited lines doesn't match the excerpts above.
- Removing `example.com` breaks tests **outside** `internal/web` (other
  packages should never hit the guard; if one does, the assumption that
  only web tests rely on the httptest default host is false).
- `t.Setenv` is unavailable on `testing.TB` in this toolchain.
- Any existing test asserts the **absence** of one of the new headers.
- You find a template that legitimately frames a Balaur page (would break
  under `X-Frame-Options: DENY`).

## Maintenance notes

- **CSP is deferred**, not rejected: adopting it requires moving the inline
  `<script>` blocks in `home.html`/`head-chat.html` into static files and
  auditing `hx-on:`/`onclick` usage. Record it as future hardening.
- Anyone serving Balaur on a LAN/NetBird name uses `BALAUR_ALLOWED_HOSTS`;
  the guard change does not affect that path (see `docs/netbird.md`).
- New card types with free-string params inherit the 256-byte cap
  automatically via `Validate`.
- Reviewer should scrutinize: the headers middleware path-skip matches
  `guardLocalUI`'s exactly (`/api/`, `/_`).
