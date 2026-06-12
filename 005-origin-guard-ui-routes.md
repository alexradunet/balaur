# Plan 005: Reject cross-origin browser requests to the Balaur UI (CSRF + DNS-rebinding guard)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- internal/web/web.go internal/web/handlers_test.go`
> On drift, re-verify the excerpts below before proceeding.

## Status

- **Priority**: P1 (highest-severity security finding of the audit)
- **Effort**: S–M
- **Risk**: LOW–MED (a wrong guard can lock the owner out of their own UI — the tests from plan 004 are the safety net)
- **Depends on**: plans/004-web-handler-test-harness.md
- **Category**: security
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/20

## Why this matters

Every state-changing route under `/ui/` is registered with no auth, no CSRF
token, and no Origin/Host validation (`internal/web/web.go:88-122`). Balaur
listens on `127.0.0.1:8090` — but loopback does not protect against the
owner's own browser:

- **Classic CSRF**: any website the owner visits can submit a hidden HTML
  form to `http://127.0.0.1:8090/ui/...`. Plain form POSTs are not subject
  to CORS preflight; the attacker cannot read the response but does not need
  to. Concretely reachable: `POST /ui/model/openai` (point Balaur's LLM
  traffic — which carries memories, the today block, and conversation
  context — at an attacker's endpoint and store an attacker API key),
  `POST /ui/chat` (inject a turn that drives the agent loop and its tools),
  `POST /ui/knowledge/{kind}/{id}/transition` (approve proposed memories/
  skills), task transitions, journal drop.
- **DNS rebinding**: a page on `evil.example` whose DNS flips to `127.0.0.1`
  becomes same-origin with the local Balaur in the browser's eyes; fetches
  then carry `Host: evil.example:8090` and CAN READ responses — the whole
  chat history page. A Host allowlist closes this.

The vendored htmx 2.0.8's `selfRequestsOnly` only constrains htmx-initiated
requests on Balaur's own pages; it does nothing about other sites' forms.

The repo's documented threat model ("loopback-first, do not expose the
port") does not cover either vector — both attacks ride the owner's
browser to a loopback service. AGENTS.md KISS rules favor the ~40-line
middleware below over a token framework.

## Current state

- `internal/web/web.go:78-123` — `Register` mounts everything directly on
  `se.Router` (a `*router.RouterGroup`); excerpt:

```go
	se.Router.GET("/static/{path...}", apis.Static(staticFS, false))
	h := &handlers{app: se.App, tmpl: tmpl}
	se.Router.GET("/", h.home)
	...
	se.Router.POST("/ui/chat", h.chat)
	se.Router.POST("/ui/model/openai", h.saveOpenAIModel)
	...
```

- PocketBase v0.39.3 router middleware:
  `func (group *RouterGroup[T]) BindFunc(middlewareFuncs ...func(e T) error) *RouterGroup[T]`
  (`tools/router/group.go:49`). A middleware bound on `se.Router` sees every
  request, including PocketBase's own `/api/*` and `/_/*` — the guard MUST
  scope itself by path so PB's surfaces keep their own protections
  (they have auth) and `/api` JS/SDK use stays untouched.
- `e.Request` is a standard `*http.Request`; reject with
  `e.ForbiddenError("...", nil)` (pattern: `e.BadRequestError` used at
  `web.go:130`).
- Test harness: `internal/web/handlers_test.go` from plan 004
  (`newWebApp` + `tests.ApiScenario` with a `Headers` map).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Format | `gofmt -l .` | empty |
| Vet | `go vet ./...` | exit 0 |
| Focused tests | `go test ./internal/web/ -run TestOriginGuard -v` | pass |
| Full suite | `go test ./...` | all ok |
| Build | `CGO_ENABLED=0 go build -o /tmp/balaur-test .` | exit 0 |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/web/web.go` (the middleware + its binding)
- `internal/web/handlers_test.go` (guard scenarios)
- `README.md` (one short paragraph documenting the guard + the
  `BALAUR_ALLOWED_HOSTS` escape hatch, in the section that currently warns
  "Do not expose the Balaur port"; README lines ~172-184)

**Out of scope** (do NOT touch):
- PocketBase's `/api/*` and `/_/*` routes — they carry their own auth; the
  guard must SKIP them.
- `GET /static/*` — public assets, harmless.
- Any individual handler — the guard is one middleware, not 24 edits.
- Session/token CSRF frameworks — rejected for KISS; Origin+Host checking
  fully covers the identified vectors for a browser-driven loopback app.

## Git workflow

- Branch: `advisor/005-origin-guard`
- Commit style: `fix(web): reject cross-origin and non-local-host requests to the Balaur UI`. No push/PR unless instructed.

## Steps

### Step 1: Write the guard middleware

In `internal/web/web.go` add:

```go
// guardLocalUI rejects browser-driven cross-site requests to Balaur's own
// surfaces. Two checks, both scoped to Balaur paths (PocketBase's /api and
// /_ keep their own auth):
//   - Host must be a loopback address (DNS-rebinding defence). Owners who
//     deliberately serve on a LAN name can allow it via BALAUR_ALLOWED_HOSTS
//     (comma-separated host[:port] values).
//   - On state-changing methods, an Origin header, when present, must match
//     the request Host (cross-site form/fetch POST defence). Absent Origin
//     (curl, CLI, same-origin GET) passes.
func guardLocalUI(e *core.RequestEvent) error {
	p := e.Request.URL.Path
	if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/_") {
		return e.Next()
	}
	host := e.Request.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if !isAllowedHost(host) {
		return e.ForbiddenError("host not allowed", nil)
	}
	if e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead {
		if origin := e.Request.Header.Get("Origin"); origin != "" && origin != "null" {
			u, err := url.Parse(origin)
			if err != nil || !sameHost(u.Host, e.Request.Host) {
				return e.ForbiddenError("cross-origin request rejected", nil)
			}
		}
	}
	return e.Next()
}
```

with helpers `isAllowedHost` (true for `localhost`, any IP for which
`net.ParseIP(host).IsLoopback()`, or membership in the comma-separated
`BALAUR_ALLOWED_HOSTS` env list) and `sameHost` (compare host:port with the
port-stripped fallback, so `Origin: http://127.0.0.1:8090` matches
`Host: 127.0.0.1:8090`). Reject `Origin: null` (sandboxed/opaque origins —
covered by the `!= "null"` exclusion above being ABSENT: note `null` must be
REJECTED, so the condition treats `"null"` as a mismatch — implement it as:
non-empty Origin that is `"null"` or fails the host match → reject).
Imports to add: `net`, `net/http`, `net/url`, `os`.

Bind it first thing in `Register`, before any route registration:

```go
	se.Router.BindFunc(guardLocalUI)
```

**Verify**: `gofmt -l .` → empty; `go vet ./internal/web/` → exit 0.

### Step 2: Guard scenarios in the harness

Extend `internal/web/handlers_test.go` (plan 004) with `TestOriginGuard`
table cases — all against `POST /ui/chat` with body `message=x` unless
noted:

| Case | Headers | Expect |
|---|---|---|
| same-origin POST | `Origin: http://127.0.0.1:8090`, `Host` default | 200 |
| cross-origin POST | `Origin: https://evil.example` | 403 |
| null origin POST | `Origin: null` | 403 |
| no-Origin POST (curl-style) | none | 200 |
| rebound host GET | `Host: evil.example:8090` on `GET /` | 403 |
| allowed extra host | env `BALAUR_ALLOWED_HOSTS=mybox.lan`, `Host: mybox.lan:8090`, `GET /` | 200 |
| PB API untouched | `GET /api/health`, `Host: evil.example` | NOT 403 from this guard (PB's own status) |

For the env case use `t.Setenv`. Check how `ApiScenario` sets `Host`
(`Headers["Host"]` may not apply to `Request.Host` — if not, set it in a
`BeforeTestFunc`-style hook or construct that one case with the plan-004
factory + a hand-built request; read `tests/api.go` to pick the supported
mechanism).

**Verify**: `go test ./internal/web/ -run TestOriginGuard -v` → all cases pass.

### Step 3: Confirm the existing suite + harness cases still pass

The plan-004 chat/transition scenarios must still pass (they send no Origin
header — the curl-style row above proves the design). Then README: add the
short guard paragraph + `BALAUR_ALLOWED_HOSTS` to the optional env block.

**Verify**: `go test ./...` → all ok; `grep -n "BALAUR_ALLOWED_HOSTS" README.md` → ≥ 1 match.

### Step 4: Manual end-to-end sanity (optional but recommended)

```bash
CGO_ENABLED=0 go build -o /tmp/balaur-test . && /tmp/balaur-test --dir $(mktemp -d) serve &
sleep 2
curl -s -o /dev/null -w '%{http_code}\n' -X POST -H 'Origin: https://evil.example' --data 'message=x' http://127.0.0.1:8090/ui/chat   # expect 403
curl -s -o /dev/null -w '%{http_code}\n' -X POST --data 'message=x' http://127.0.0.1:8090/ui/chat                                    # expect non-403 (likely model-missing error page, 200)
kill %1
```

## Test plan

- New: `TestOriginGuard` table (Step 2) — 7 cases listed above, in
  `internal/web/handlers_test.go`, modeled on plan 004's scenario table.
- Regression: all plan-004 scenarios and the full suite stay green.

## Done criteria

- [ ] `grep -n "BindFunc(guardLocalUI)" internal/web/web.go` → 1 match
- [ ] `go test ./internal/web/ -run TestOriginGuard -v` → ≥ 7 passing cases
- [ ] `go test ./...` exit 0; `gofmt -l .` empty; `go vet ./...` exit 0
- [ ] `CGO_ENABLED=0 go build -o /tmp/balaur-test .` exit 0
- [ ] Changes confined to `internal/web/web.go`, `internal/web/handlers_test.go`, `README.md` (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- Plan 004's harness is not merged yet (no `internal/web/handlers_test.go`)
  — land 004 first; do not inline a second harness.
- `se.Router.BindFunc` does not exist on the ServeEvent router in the
  PocketBase version in `go.mod` — report the actual middleware API you
  find in `tools/router/group.go`.
- The guard breaks HTMX same-origin requests in Step 3 (browsers send
  `Origin` on XHR POSTs — if the same-origin case 403s, your `sameHost`
  comparison is wrong; fix the comparison, and if it still fails, stop).
- You find yourself wanting to add a token/cookie CSRF system — out of
  scope; stop and report why Origin+Host was insufficient.

## Maintenance notes

- If a future gateway serves Balaur behind a reverse proxy with a public
  hostname, `BALAUR_ALLOWED_HOSTS` is the supported knob; a proxy that
  strips/rewrites `Origin` will behave like the curl row (allowed) — at
  that point a real auth layer (PB auth on the UI routes) is the next step,
  not more Origin logic.
- `POST /ui/dev/seed-recaps` remains env-gated AND now origin-guarded;
  plan 010 additionally moves its registration behind the env check.
- Reviewer: confirm the guard runs BEFORE `apis.Static` route matching (it
  is bound on the root group, so yes) and that `/_` prefix skip covers both
  `/_/` UI and superuser API paths.
