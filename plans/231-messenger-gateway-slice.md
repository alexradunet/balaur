# Plan 231: Messenger gateway slice — a loopback-only, consent-gated `POST /api/messenger/turn` over `turn.Run`

> **Phase-1 build of plan 228's Bet A.** The Phase-0 spike
> (`docs/superpowers/specs/2026-06-30-messenger-gateway-spike.md`) verified the
> gateway line holds and designed a constraint-respecting slice. This builds the
> smallest **transport-agnostic** version: an owner-only loopback endpoint a
> local bridge can POST a message to and get the reply. It deliberately ships NO
> chat-app bridge (that's the owner's own process) — it proves the doctrine and
> gives a usable, safe surface.
>
> **Drift check (run first)**:
> `git diff --stat e8766b1..HEAD -- internal/web/web.go internal/web/chat.go internal/cli/chat.go internal/turn/turn.go internal/store/owner_settings.go`
> On any change, compare excerpts to live code before editing; on mismatch, STOP.

## Status
- **Priority**: P3 (direction / reach)
- **Effort**: M
- **Risk**: HIGH (a new network-adjacent surface; the build must stay inside the
  `loopback-first`, `not-internet-exposed`, `consent-gated` non-goals)
- **Depends on**: none (builds on `turn.Run`, `guardLocalUI`, owner settings)
- **Category**: direction / reach
- **Planned at**: commit `e8766b1`, 2026-06-30

## Why this matters (and the hard constraints)

AGENTS.md doctrine: *"Every surface that carries an owner turn calls the shared
pipeline in `internal/turn` and only renders its events in its own medium."* The
Phase-0 spike audited the two existing gateways (`internal/web/chat.go`,
`internal/cli/chat.go`) and found the line holds — neither re-implements turn
behavior. This slice proves it with a third, differently-shaped surface AND gives
the owner a scriptable message endpoint.

**The non-negotiable constraints (PRODUCT.md, enforced by the spike's threat
model). The build is disqualified if it violates any:**
1. **Loopback-first.** The endpoint binds no new non-loopback listener; it lives
   on the existing PocketBase router behind `guardLocalUI` (host check). Remote
   reach is the owner's own tunnel (NetBird mesh), never a public listener.
2. **Consent-gated / opt-in / fail-closed.** The endpoint is **disabled unless the
   owner explicitly enables it** by setting a token owner-setting. No token set →
   the endpoint refuses (it does not run a turn). Same bar as the OS-tools and
   cloud-model opt-ins.
3. **No turns through a third party.** This slice contains NO chat-app client; it
   never calls a platform API. It only accepts a local POST and returns a reply.
4. **No secrets in output/logs.** The token is never logged; errors are sanitized.

> If, while building, the only way to make this work crosses any of the four
> constraints, **STOP and report (Bet B / defer)** — that is an acceptable,
> doctrine-honest outcome. Do NOT bind a public port, do NOT add a platform client.

## The reference gateways to mirror (read them first)

`internal/cli/chat.go` is the closest analog — a **buffered (non-streaming)**
gateway over `turn.Run`:
```go
// cli/chat.go:59-110 (the shape to mirror)
client, err := chatClients(app)              // resolve the active model client
ctx, cancel := context.WithTimeout(...)      // a deadline
events := []toolEvent{}; emit := func(ev agent.Event) { /* buffer tool rounds */ }
res, runErr := turn.Run(ctx, app, client, message, emit)   // THE one shared pipeline
return map[string]any{"reply": res.Reply, ...}             // buffered reply out
```
`internal/web/chat.go` is the web gateway — mirror **its** client resolution (it
runs inside OnServe where the Kronk engine is already registered, unlike the CLI
which builds one). Use the same active-client resolution the web chatbar uses
(`turn.ClientSource{Engine: kronk.FromStore(app)}.Active(app)` or whatever
`web/chat.go` calls — copy it exactly; do not invent a new resolution path).

`turn.Run` signature (`internal/turn/turn.go:69`):
```go
func Run(ctx context.Context, app core.App, client llm.Client, userText string, emit func(agent.Event)) (Result, error)
```
`res.Reply` is the assistant's reply string. **Do NOT change `turn.Run`'s
signature** (spike STOP condition).

## Current state (verified at `e8766b1`)
- `guardLocalUI` — `internal/web/web.go` (~L105–122): the host-check guard that
  enforces loopback / `BALAUR_ALLOWED_HOSTS`. Apply it to the messenger route the
  same way the `/ui/*` routes use it (confirm the exact application idiom in
  `web.go`'s route registration around L180–245).
- Route registration — `se.Router.GET/POST(...)` in `web.Register` (`web.go:138`).
- Owner settings — `store.GetOwnerSetting(app, key, default)` /
  `store.SetOwnerSetting` (`internal/store/owner_settings.go:43`) is the
  consent-ledger read/write for a singleton key (UNIQUE on `key`).
- Buffered gateway exemplar — `internal/cli/chat.go` (above).

## The slice (what to build)

A new `internal/web/messenger.go` registering **`POST /api/messenger/turn`**, wired
in `web.Register` behind `guardLocalUI`. The handler:

1. **Loopback guard.** Reuse `guardLocalUI` (host check) so a non-loopback request
   is rejected before any logic — exactly as the UI routes are guarded.
2. **Consent gate (fail-closed).** Read the token: `tok := store.GetOwnerSetting(app, "messenger_token", "")`.
   If `tok == ""`, the feature is **disabled** → return `404`/`403` with a plain
   "messenger gateway is not enabled" message, and run NO turn. (Setting the token
   is the owner's explicit enable; a settings-UI toggle is a future enhancement —
   for v1 the owner sets `owner_settings.messenger_token` via the admin engine
   room. Note this in the handler doc + your report.)
3. **Token auth.** Compare a request header (e.g. `Authorization: Bearer <token>`
   or `X-Balaur-Token`) to `tok` with a **constant-time compare**
   (`crypto/subtle.ConstantTimeCompare`). Missing/mismatch → `401`. Never log the
   token or the header.
4. **In-flight guard (gateway-local).** A package-level `sync.Mutex` (or
   `TryLock`): if a messenger turn is already running, return `429` "busy — a turn
   is already in progress" instead of starting a concurrent turn on the master
   conversation. (Scope: this guards the MESSENGER surface against self-collision.
   A cross-surface guard — web+messenger at once — is a known, pre-existing gap
   that also exists between web and CLI today; call it out in the maintenance
   notes as the spike's broader follow-up, do NOT push a guard into `turn.Run` in
   this slice.)
5. **Run the turn (buffered).** Parse the inbound message (JSON body, e.g.
   `{"message": "..."}`), resolve the active client (mirror `web/chat.go`), apply a
   timeout context, build a buffering `emit` (you may discard tool events or
   include a compact summary — the reply is what matters), call
   `turn.Run(ctx, app, client, message, emit)`, and return `{"reply": res.Reply}`
   as JSON. On a turn error, return a sanitized `500` (no internal paths/secrets).
6. **No streaming.** Synchronous request → reply. The bridge handles async delivery
   on its side. (The spike's async note: a goroutine wrapper is only needed for a
   true push transport, which this slice does not build.)

## Commands you will need
| Purpose   | Command                                              | Expected |
|-----------|------------------------------------------------------|----------|
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0   |
| Vet       | `go vet ./...`                                        | exit 0   |
| Test pkg  | `go test ./internal/web/... -count=1`                | PASS     |
| Full test | `go test ./... -count=1`                             | all pass |
| gofmt     | `gofmt -l internal/web`                              | nothing  |

> Prefix with `TMPDIR=/home/alex/.cache/go-tmp`, use `-count=1`. Commit in the
> FOREGROUND (hook runs `make lint`). Do NOT run `make vulncheck` (RAM-OOMs).

## Scope
**In scope**:
- `internal/web/messenger.go` (new — the handler + the gateway-local mutex).
- `internal/web/web.go` (register the route behind `guardLocalUI`).
- `internal/web/messenger_test.go` (new — the security + behavior tests).

**Out of scope** (do NOT touch):
- `internal/turn/turn.go` — do NOT change `turn.Run`'s signature or push the
  in-flight guard into it.
- The web chat gateway (`web/chat.go`) / CLI gateway — read as reference only.
- Any chat-app client / platform API — NOT in this slice.
- `internal/self/knowledge.md` / DESIGN / README — until this ships as a real
  capability, "future messengers" stays roadmap; a doc update is a SEPARATE
  follow-up (note it in your report, do not edit docs here).

## Git workflow
- Branch: `advisor/231-messenger-gateway-slice`
- Subject e.g. `feat(web): loopback-only consent-gated messenger turn endpoint`
- Do NOT push.

## Steps

### Step 1: The handler + route (loopback + consent gate + auth + mutex + turn)
Build `internal/web/messenger.go` per "The slice" above and register
`POST /api/messenger/turn` in `web.Register` behind `guardLocalUI`. Mirror
`cli/chat.go` for the buffered `turn.Run` call and `web/chat.go` for client
resolution.
**Verify**: `go build ./internal/web/... && go vet ./internal/web/...` → exit 0; `gofmt -l internal/web` → clean.

### Step 2: Tests (the security boundary is the deliverable)
Add `internal/web/messenger_test.go` (mirror the `newWebApp(t)` + router pattern in
`internal/web/nudges_poll_test.go` / `graph_test.go`; fake `llm.Client` via the
existing web test fake):
- **Disabled by default**: no `messenger_token` set → POST returns 403/404 and NO
  message is appended to the master conversation (assert the conversation is
  unchanged).
- **Auth required**: token set, but request has no/incorrect token header → `401`,
  no turn run.
- **Happy path**: token set + correct header + `{"message":"hi"}` → `200` with a
  `reply` (fake client), and the turn was persisted (master conversation grew).
- **Loopback guard**: a request with a non-loopback `Host` (and not in
  `BALAUR_ALLOWED_HOSTS`) → rejected (mirror `TestGuardRejectsNonLoopbackHost`).
- **In-flight guard** (best-effort): with a fake client that blocks until signaled,
  fire a second request while the first is mid-turn → second returns `429`. If a
  deterministic concurrency test proves too flaky, assert the mutex path exists and
  document why the concurrency case is covered structurally — do NOT ship without
  at least the disabled/auth/happy/loopback tests.
**Verify**: `go test ./internal/web/ -run Messenger -count=1 -v` → PASS.

### Step 3: Full verification
- `gofmt -l internal/web` → nothing
- `go vet ./...` → exit 0
- `go test ./... -count=1` → all pass

## Test plan
The security tests ARE the feature: **disabled-by-default**, **auth-required**, and
**loopback-only** prove the three consent/exposure constraints; the happy path
proves the gateway line (it calls `turn.Run`, persists a real turn). The in-flight
guard test proves the new surface won't self-collide.

## Done criteria — ALL must hold
- [ ] `POST /api/messenger/turn` exists, behind `guardLocalUI`, in `web.Register`.
- [ ] Disabled (no turn run, no message persisted) when `messenger_token` is unset.
- [ ] Requires a constant-time-compared token header when enabled; mismatch → 401.
- [ ] Happy path runs `turn.Run` and returns `{"reply": ...}`; the turn is persisted.
- [ ] Non-loopback host rejected (guardLocalUI).
- [ ] A gateway-local in-flight guard returns 429 on a concurrent messenger turn.
- [ ] `grep -n "turn.Run(" internal/web/messenger.go` → exactly one call (it ADAPTS, never re-implements the loop).
- [ ] `turn.Run`'s signature is unchanged; no platform/chat-app client added; token never logged.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l internal/web` clean; `go test ./... -count=1` green.
- [ ] `plans/README.md` status row updated.

## STOP conditions
- If making the endpoint work requires binding a non-loopback listener, routing
  through a third party, or enabling it without an explicit owner token → STOP and
  report Bet B (defer); do not cross the non-goal.
- If `guardLocalUI` cannot be applied to a `POST /api/*` route the way it is applied
  to `/ui/*` (e.g. it is UI-path-specific) → report the obstacle; the loopback guard
  is non-negotiable, find the correct application or STOP.
- If a real turn cannot be run from the messenger handler without changing
  `turn.Run`'s contract → STOP (that's a finding for a separate plan).

## Maintenance notes
- This proves the gateway line holds for a third, push-shaped surface, and gives the
  owner a scriptable loopback endpoint a local bridge can use. It ships NO chat-app
  bridge by design.
- **Known limitation (the spike's broader finding):** the in-flight guard here is
  messenger-local; a cross-surface guard (so web + messenger + CLI never run
  concurrent turns on the master conversation) is a deliberate follow-up — it would
  live below the gateway line and must be designed not to change web/CLI behavior.
- A settings-UI toggle to set/rotate `messenger_token` (instead of the admin engine
  room) is a natural follow-up.
- Doc-truth follow-up: only once a real bridge ships does `internal/self/knowledge.md`
  move "future messengers" from roadmap to capability — NOT in this slice.
- Reviewer: scrutinize the four constraints (loopback, consent/fail-closed, no
  third party, no secret leak) and the constant-time token compare. This is the
  surface most able to quietly betray the product's non-goals.
