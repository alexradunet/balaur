# Plan 180: Reject `Origin: null` and honor `Sec-Fetch-Site` in the `/ui/*` CSRF guard

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- internal/web/web.go internal/web/handlers_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

`guardLocalUI` in `internal/web/web.go` is the *only* CSRF defense on Balaur's
state-changing `/ui/*` POSTs — there is no anti-CSRF token. On a non-GET/HEAD
request it only rejects a cross-site `Origin` when the header is *present and
not the literal string `null`*. That leaves two bypasses: a request with
`Origin: null` (the value a browser sends from a sandboxed iframe or an opaque
origin) and a request with *no* `Origin` at all both sail through, because
`isAllowedHost` returns `true` for any loopback host. A malicious page the
owner visits can therefore POST to `http://127.0.0.1:8090/ui/chat`,
`/ui/model/select`, `/ui/model/cloud/delete`, `/ui/day/.../journal`, etc. from
a sandboxed iframe whose form submits with `Origin: null`, driving
owner-trusted actions without consent. The attack is blind and loopback-bounded
(it can't read the response), but it still *mutates state without the owner's
intent* — off Balaur's Consent pillar.

After this plan: `Origin: null` is treated as a rejection on state-changing
methods (it is attacker-influenced, not trusted-absent), and the guard
additionally honors the browser-set, unspoofable `Sec-Fetch-Site` header.
Requests that legitimately send neither header — the CLI, `curl`, the
same-process test harness — still pass, so nothing real breaks.

## Current state

### The guard — `internal/web/web.go`

The package doc, guard, and helpers (verified live at planned-at SHA):

```go
53	// guardLocalUI rejects browser-driven cross-site requests to Balaur's own
54	// surfaces. Two checks, both scoped to Balaur paths (PocketBase's /api and
55	// /_ keep their own auth):
56	//   - Host must be a loopback address (DNS-rebinding defence). Owners who
57	//     deliberately serve on a LAN name can allow it via BALAUR_ALLOWED_HOSTS
58	//     (comma-separated host[:port] values).
59	//   - On state-changing methods, an Origin header, when present, must match
60	//     the request Host (cross-site form/fetch POST defence). Absent Origin
61	//     (curl, CLI, same-origin GET) passes.
62	func guardLocalUI(e *core.RequestEvent) error {
63		p := e.Request.URL.Path
64		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/_") {
65			return e.Next()
66		}
67		host := e.Request.Host
68		if h, _, err := net.SplitHostPort(host); err == nil {
69			host = h
70		}
71		if !isAllowedHost(host) {
72			return e.ForbiddenError("host not allowed", nil)
73		}
74		if e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead {
75			if origin := e.Request.Header.Get("Origin"); origin != "" && origin != "null" {
76				u, err := url.Parse(origin)
77				if err != nil || !sameHost(u.Host, e.Request.Host) {
78				return e.ForbiddenError("cross-origin request rejected", nil)
79			}
80		}
81	}
82	return e.Next()
83}
```

(Lines 77–82 above are reproduced with the file's real tab indentation; do not
re-indent the surrounding block when you edit — only change what the steps say.)

`sameHost` (used by the guard) and `isAllowedHost`, live at the planned-at SHA:

```go
85	func isAllowedHost(host string) bool {
86		if host == "localhost" {
87			return true
88		}
89		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
90			return true
91		}
92		allowed := os.Getenv("BALAUR_ALLOWED_HOSTS")
93		if allowed == "" {
94			return false
95		}
96		for h := range strings.SplitSeq(allowed, ",") {
97			if strings.TrimSpace(h) == host {
98				return true
99			}
100		}
101		return false
102	}
103	
104	func sameHost(origin, request string) bool {
105		// Strip ports for comparison
106		origHost := origin
107		if h, _, err := net.SplitHostPort(origin); err == nil {
108			origHost = h
109		}
110		reqHost := request
111		if h, _, err := net.SplitHostPort(request); err == nil {
112			reqHost = h
113		}
114		return origHost == reqHost
115	}
```

The guard is bound first in `Register` (line 125): `se.Router.BindFunc(guardLocalUI)`.

The state-changing `/ui/*` POST routes this guard protects (registered in
`Register`, lines 165+) include `POST /ui/chat`, `POST /ui/model/select`,
`POST /ui/model/cloud/delete`, `POST /ui/day/{date}/journal`,
`POST /ui/panel/collapse`, and ~30 more.

### How Balaur's own UI issues state-changing requests (confirmed)

This was confirmed before writing the plan — it is the load-bearing assumption
that the new rule must not break the real UI:

- The UI drives writes through **same-origin browser `fetch`** — both Datastar's
  `@post(...)` (the vendored `internal/web/assets/static/datastar.js`) and a
  couple of hand-rolled calls in `internal/web/assets/static/basm.js`, e.g.:

  ```js
  191	  fetch('/ui/panel/collapse', {
  192	    method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  193	    body: 'on=' + (on ? '1' : '0'),
  194	  });
  ```

  These target **relative, same-origin URLs** (`/ui/...`). For a same-origin
  `fetch`/form POST a browser sends `Origin: <page origin>` (matching Host, never
  the string `null`) **and** the browser-set request-metadata header
  `Sec-Fetch-Site: same-origin`. Neither `datastar.js` nor `basm.js` sets
  `Sec-Fetch-Site` itself — that header is browser-controlled and cannot be
  forged by a cross-site page. So under the new rule the real UI passes via the
  same-origin `Origin` match (already passing today) and additionally via
  `Sec-Fetch-Site: same-origin`.

- The **CLI / `curl` / same-process test harness** send *neither* `Origin` nor
  `Sec-Fetch-Site`. They must keep passing — do **not** blanket-reject an absent
  `Origin`.

### The test harness (model new tests after this)

`internal/web/handlers_test.go` already contains the guard's only test,
`TestOriginGuard` (live at the planned-at SHA, lines 419–432):

```go
419	func TestOriginGuard(t *testing.T) {
420		// Origin guard is bound at the start of Register. This test verifies
421		// it allows localhost (the test app default) and doesn't break normal requests.
422		scenario := tests.ApiScenario{
423			Name:            "origin guard allows localhost",
424			Method:          "GET",
425			URL:             "/",
426			Headers:         map[string]string{"Host": "localhost"},
427			ExpectedStatus:  200,
428			ExpectedContent: []string{"app-shell"},
429		}
430		scenario.TestAppFactory = newWebApp
431		scenario.Test(t)
432	}
```

`newWebApp` (lines 29–52) is the factory. It sets
`t.Setenv("BALAUR_ALLOWED_HOSTS", "example.com")` because `httptest.NewRequest`
(used by `tests.ApiScenario`) defaults the request **Host to `example.com`**.
So in tests `e.Request.Host == "example.com"`, and a *same-origin* `Origin`
header in a test is `http://example.com`.

`tests.ApiScenario` applies its `Headers` map verbatim onto the request via
`req.Header.Set(k, strings.TrimSpace(v))` (PocketBase v0.39.3
`tests/api.go:234-236`), so a test can set `Origin` and `Sec-Fetch-Site`
directly through the `Headers` map. The scenario also sets a default
`content-type: application/json` first, which a `Headers` entry overrides — an
existing POST scenario (`TestChatChoices`, lines 64–76) overrides it with
`Content-Type: application/x-www-form-urlencoded` and posts a form body. Model
the new POST scenarios after that one.

A simple POST target that needs no model and no special body is
`POST /ui/panel/collapse` (registered at `web.go:225`, handler
`h.uiPanelCollapse`). Use it for the guard tests so a 403 from the *guard* is
the only interesting outcome and a 200 means the guard let the request reach
the handler. (Its handler returns 200 on a well-formed `on=...` form body; if
in doubt about its exact success body, assert only `ExpectedStatus` and do not
assert response content for the allow cases.)

### Conventions that apply here (with exemplars)

- **Errors are values / structured logging**: the guard already returns
  `e.ForbiddenError("...", nil)` rather than panicking — keep that style for any
  new rejection. No new logging is needed (none exists in the guard today).
- **Tests are standard `testing`, table-driven, no assertion frameworks, no
  `time.Sleep`** — see `TestOriginGuard` and `TestChatChoices` above; a
  `[]tests.ApiScenario` slice iterated with `t.Run(sc.Name, ...)` is the
  idiomatic shape. Use `storetest`/`tests.NewTestApp` indirectly through the
  existing `newWebApp` factory; do not stand up a new app harness.
- **gofmt is law** (a PostToolUse hook + CI gofmt gate enforce it); keep tabs,
  run `gofmt -l .` before declaring done.
- **KISS / suckless**: this is a ~6-line change in the guard plus a small
  helper if it reads cleaner. Do not introduce a CSRF-token system, a config
  knob, or new packages.

## Commands you will need

| Purpose      | Command                            | Expected on success           |
|--------------|------------------------------------|-------------------------------|
| Drift check  | `git diff --stat 12a48bf..HEAD -- internal/web/web.go internal/web/handlers_test.go` | empty, or you reconcile excerpts |
| Build        | `CGO_ENABLED=0 go build ./...`     | exit 0, no output             |
| Test (pkg)   | `go test ./internal/web/`          | `ok ... internal/web`         |
| Test (all)   | `go test ./...`                    | all packages `ok`             |
| Vet          | `go vet ./...`                     | exit 0, no output             |
| Fmt check    | `gofmt -l .`                       | empty output                  |
| Diff check   | `git diff --check`                 | empty output                  |

## Suggested executor toolkit

- Invoke the `go-standards` skill before editing `web.go` to apply the repo's
  Go idioms (error wrapping, the testing conventions above).

## Scope

**In scope** (the only files you should modify):

- `internal/web/web.go` — change `guardLocalUI` (and optionally add one small
  unexported helper next to it).
- `internal/web/handlers_test.go` — extend `TestOriginGuard` / add a new
  table-driven test for the guard.

**Out of scope** (do NOT touch, even though they look related):

- A full anti-CSRF-token system — overkill for a loopback-only personal service;
  explicitly deferred.
- PocketBase's `/api/*` and `/_*` surfaces — already skipped at `web.go:64`,
  and they manage their own auth.
- `isAllowedHost` and its loopback / `BALAUR_ALLOWED_HOSTS` policy — unchanged.
- The static JS (`basm.js`, `datastar.js`) — read-only here; the browser sets
  `Sec-Fetch-Site` automatically, so no client change is needed.
- `internal/self/knowledge.md` — the guard is not described there (a grep for
  "guard/CSRF/Origin/loopback" finds nothing), so this change does not alter a
  documented capability; do not add a new section for it.

## Git workflow

- Branch: an executor worktree off `origin/main` (e.g. `advisor/180-harden-csrf-ui`).
- Single commit; conventional-commit subject, e.g.
  `fix(web): reject Origin: null and honor Sec-Fetch-Site in /ui CSRF guard`.
- Do NOT push or open a PR unless the operator instructed it. (Balaur lands on
  `main`; the operator says when.)

## Steps

### Step 1: Confirm the assumption before editing

Re-confirm the UI does not depend on `Origin: null` or a cross-site
`Sec-Fetch-Site`. The relevant client fetches target relative `/ui/*` URLs
(see `internal/web/assets/static/basm.js:191` and Datastar `@post` usage), which
the browser issues same-origin. If you find any UI code path that POSTs to
`/ui/*` from a sandboxed/opaque-origin context or that would send
`Sec-Fetch-Site: cross-site`, **stop** (see STOP conditions).

**Verify**: `grep -n "fetch('/ui" internal/web/assets/static/basm.js` →
shows only relative same-origin `/ui/...` POSTs (no absolute cross-origin URL).

### Step 2: Tighten `guardLocalUI` in `internal/web/web.go`

Replace the state-changing-method block (lines 74–81) so it implements three
rules, in this order:

1. **`Sec-Fetch-Site`, when present, is authoritative.** If the header is
   present and its value is neither `same-origin` nor `none`, reject. (Browsers
   set `none` for user-initiated top-level navigations and `same-origin` for
   same-site fetches; `cross-site` / `same-site` from a state-changing request
   is a cross-site attempt.) When `Sec-Fetch-Site` is present and is
   `same-origin` or `none`, the request is trusted same-origin — allow it
   without the Origin check.
2. **`Origin: null` is a rejection.** If `Sec-Fetch-Site` is absent, fall back to
   the Origin check, but treat the literal value `null` as a cross-site reject
   (it is attacker-influenced — opaque/sandboxed origins send it), not as
   trusted-absent.
3. **Absent `Origin` (and absent `Sec-Fetch-Site`) still passes** — curl, CLI,
   the same-process test harness send neither and must not be rejected.

Target shape for the block (keep the file's tab indentation; the surrounding
`if e.Request.Method != ...GET && ... != ...HEAD {` and the closing `}` /
`return e.Next()` stay):

```go
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
```

Also update the guard's doc comment (lines 59–61) so it no longer claims only
"an Origin header, when present, must match" — describe the new
`Sec-Fetch-Site` + `Origin: null` behavior in one or two lines. Keep it short.

Do not change `isAllowedHost` or `sameHost`.

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `gofmt -l internal/web/web.go` → empty.

### Step 3: Extend the guard tests in `internal/web/handlers_test.go`

Keep the existing `TestOriginGuard` GET allow-case, and add a table-driven set
of POST cases against `POST /ui/panel/collapse` covering exactly the matrix
below. Use `tests.ApiScenario` with `TestAppFactory: newWebApp`,
`Method: "POST"`, `URL: "/ui/panel/collapse"`,
`Body: strings.NewReader("on=1")`, and
`Headers` including `Content-Type: application/x-www-form-urlencoded` plus the
per-case CSRF headers. Remember the test Host is `example.com` (so a same-origin
Origin is `http://example.com`).

| Case | Headers (besides Content-Type) | Expected status |
|------|-------------------------------|-----------------|
| `Origin: null` rejected | `Origin: null` | 403 |
| cross-site fetch-metadata rejected | `Sec-Fetch-Site: cross-site` | 403 |
| cross-origin Origin rejected (regression of existing rule) | `Origin: http://evil.example` | 403 |
| same-origin Origin allowed | `Origin: http://example.com` | 200 |
| same-origin fetch-metadata allowed | `Sec-Fetch-Site: same-origin` | 200 |
| CLI/curl (no CSRF headers) allowed — harness regression guard | *(none)* | 200 |

For the allow cases assert only `ExpectedStatus: 200` (do not assert response
body — the point is the guard let the request reach the handler). For the 403
cases assert `ExpectedStatus: 403`. Iterate the slice with
`t.Run(sc.Name, func(t *testing.T) { sc.Test(t) })`, mirroring the structure of
`TestChatChoices`.

Note on the existing reject value: the current code already returns 403 for a
cross-origin `Origin`; `e.ForbiddenError` yields HTTP 403, so all reject rows
assert `403`.

**Verify**: `go test ./internal/web/ -run 'TestOriginGuard' -v` →
all sub-tests `PASS`, including the four new POST cases and the two allow cases.

### Step 4: Full package + repo gates

**Verify**:
- `go test ./internal/web/` → `ok ... internal/web`.
- `go vet ./...` → exit 0.
- `gofmt -l .` → empty.
- `git diff --check` → empty.
- `go test ./...` → all packages `ok` (green full suite before any push).

## Test plan

- **File**: `internal/web/handlers_test.go` (extend, do not create a new file).
- **New cases** (all in/adjacent to `TestOriginGuard`, table-driven against
  `POST /ui/panel/collapse`):
  - `Origin: null` → 403 (the core bug this plan fixes).
  - `Sec-Fetch-Site: cross-site` → 403 (new defense).
  - `Origin: http://evil.example` → 403 (existing rule still holds).
  - `Origin: http://example.com` (same-origin) → 200 (real-UI path 1).
  - `Sec-Fetch-Site: same-origin` → 200 (real-UI path 2).
  - no CSRF headers → 200 (CLI/curl/harness regression guard — must not break).
- **Structural pattern**: model the POST scenarios after `TestChatChoices`
  (lines 64–76) and the existing `TestOriginGuard` (lines 419–432).
- **Verification**: `go test ./internal/web/` → all pass, including the 6 new
  guard cases.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./internal/web/` passes, including the new `Origin: null` → 403,
      `Sec-Fetch-Site: cross-site` → 403, same-origin → 200, and no-header → 200
      cases.
- [ ] `go test ./...` is green (full suite).
- [ ] `go vet ./...` exits 0.
- [ ] `gofmt -l .` prints nothing.
- [ ] `git diff --check` prints nothing.
- [ ] `git status` shows only `internal/web/web.go` and
      `internal/web/handlers_test.go` modified (no files outside scope).
- [ ] `plans/README.md` status row for plan 180 updated (unless a reviewer owns
      the index).

## STOP conditions

Stop and report back (do not improvise) if:

- The code at `internal/web/web.go:62-83` or `internal/web/handlers_test.go:419-432`
  does not match the "Current state" excerpts (the codebase drifted since this
  plan was written).
- **You find Balaur's own UI actually issues a state-changing `/ui/*` request
  with `Origin: null` or `Sec-Fetch-Site: cross-site`** (e.g. an `@post` from an
  iframed/sandboxed surface, or a cross-origin absolute URL). That would mean
  this fix breaks the real UI, and the correct design would instead be a
  *positive header-allow* (a header cross-site forms cannot set) — report it
  rather than shipping a guard that 403s the UI.
- The same-origin allow case (`Origin: http://example.com` or
  `Sec-Fetch-Site: same-origin` → 200) does NOT return 200 — that means the new
  rule is over-rejecting and would break the UI; do not "fix" it by widening to
  also allow `null`. Report.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching a file outside the in-scope list.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **Reviewer should scrutinize** that the *order* of the two checks is right:
  `Sec-Fetch-Site` present-and-trusted short-circuits to allow; only an absent
  `Sec-Fetch-Site` falls through to the `Origin` check; and absent-both still
  passes (CLI/harness). A regression that blanket-rejects absent `Origin` would
  silently break the CLI gateway and the test harness — the no-header → 200 test
  is the guard against that.
- **Interactions**: if a future change adds a Content-Security-Policy or moves
  any UI surface to an iframe/cross-origin embed, revisit this guard — an iframed
  surface posting to `/ui/*` would send `Sec-Fetch-Site: same-origin` only when
  same-origin; a genuinely cross-origin embed would (correctly) be rejected, so
  that embed would need its own design.
- **Deferred**: a real anti-CSRF token (double-submit or per-session) is out of
  scope here. It is the right move only if Balaur ever serves `/ui/*` beyond
  loopback to clients that strip `Sec-Fetch-Site`; until then the
  fetch-metadata + Origin guard is sufficient for a loopback-first service.
- This change does not alter `internal/self/knowledge.md` (the guard is not
  described there); if a future refactor surfaces the CSRF posture in the
  self-knowledge doc, keep it in sync in the same commit.
