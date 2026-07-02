# Plan 256: SPIKE — design and thinly prototype the reference messenger bridge

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- docs/superpowers/specs internal/bridge internal/cli internal/web/messenger.go .tours/10-the-cli-api.tour .tours/00-orientation.tour plans/README.md`
> If any of these paths changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. (`internal/web/messenger.go` is
> out of scope to MODIFY, but its request/response contract is load-bearing
> for this plan — a drift there is a STOP.)

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: direction
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Balaur has a live, token-gated, guard-protected messenger endpoint —
`POST /api/messenger/turn` in `internal/web/messenger.go` — plus an owner
Settings control for its token, and a synchronous `{"reply": "..."}` response
designed for a poll→POST→send loop. But **no bridge process exists anywhere in
the repo**, so the shipped endpoint delivers zero end-to-end value: nothing
polls a messaging platform, nothing POSTs to the endpoint, nothing delivers a
reply to the owner's phone. The Phase-0 feasibility spike
(`docs/superpowers/specs/2026-06-30-messenger-gateway-spike.md`) green-lit the
gateway and explicitly named "a reference bridge implementation (polling mode,
single transport) as a separate, owner-run process — not embedded in Balaur"
as follow-up work. `PRODUCT.md` frames "More gateways, same pipeline" as a
product bet. This plan is that follow-up — **as a SPIKE**: its deliverables
are (1) a design note settling transport, packaging, config/consent, and
failure semantics; (2) a ~200-line stdlib-only prototype of the recommended
happy path, honestly labelled experimental; (3) an open-questions list for the
owner. It deliberately does NOT ship a polished feature, does not touch the
endpoint, and does not promote messenger support out of "roadmap" in any
user-facing copy.

## Current state

### The endpoint the bridge will talk to

`internal/web/messenger.go` — `POST /api/messenger/turn`, the only messenger
surface that exists. Its contract (verify these bytes before coding):

Consent gate + token auth, `internal/web/messenger.go:64-79`:

```go
	//    explicitly sets a token in owner_settings.messenger_token.
	tok := store.GetOwnerSetting(h.app, "messenger_token", "")
	if tok == "" {
		return e.JSON(http.StatusForbidden, map[string]string{"error": "messenger gateway is not enabled"})
	}

	// 3. Token auth — constant-time comparison; the header value is never logged.
	authHeader := e.Request.Header.Get("Authorization")
	const prefix = "Bearer "
	var provided string
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		provided = authHeader[len(prefix):]
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(tok)) != 1 {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
```

Busy signal (cross-surface in-flight guard), `internal/web/messenger.go:83-87`:

```go
	end, ok := turn.TryBegin()
	if !ok {
		return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "busy"})
	}
	defer end()
```

Request body and success response, `internal/web/messenger.go:90-93` and
`internal/web/messenger.go:107-112`:

```go
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || body.Message == "" {
```

```go
	res, runErr := turn.Run(ctx, h.app, client, body.Message, func(agent.Event) {})
	if runErr != nil {
		h.app.Logger().Warn("messenger: turn failed", "error", runErr)
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "turn failed"})
	}
	return e.JSON(http.StatusOK, map[string]string{"reply": res.Reply})
```

So the bridge's wire contract with Balaur is:
`POST <balaur-url>/api/messenger/turn` with `Authorization: Bearer <token>` and
body `{"message":"..."}` → `200 {"reply":"..."}` on success, `403` when the
gateway is disabled, `401` on bad token, `429 {"error":"busy"}` while another
turn is in flight, `503` when no model is active, `500` on a failed turn.
There is also a host check (`messenger.go:55-61`) — the bridge must send a
loopback `Host`, which it gets for free by POSTing to `http://127.0.0.1:...`.

The owner sets the token in the product Settings UI
(`internal/web/messenger_settings.go:16-21`):

```go
func (h *handlers) saveMessengerToken(e *core.RequestEvent) error {
	token := strings.TrimSpace(e.Request.FormValue("messenger_token"))
	// Never log the token value — it is a secret.
	if err := store.SetOwnerSetting(h.app, "messenger_token", token); err != nil {
```

and the Settings card reads it back
(`internal/feature/settingscards/settingsfocus_capabilities.go:32` /
`:87`): `MessengerToken string // current messenger_token; empty = gateway disabled`.

### The design doc this spike extends — READ IT FIRST

`docs/superpowers/specs/2026-06-30-messenger-gateway-spike.md` is the Phase-0
feasibility + threat-model finding. The executor MUST read it in full before
writing anything. The load-bearing passages:

The bridge shape (`2026-06-30-messenger-gateway-spike.md:167-171`):

> **The bridge runs on-box.** The owner runs a small bridge process on the same
> box (or a trusted device on the NetBird mesh). The bridge polls the messaging
> platform (polling mode, not accepting inbound webhooks from the platform's
> servers) and POSTs messages to `127.0.0.1:8090/api/messenger/turn`. The
> platform's servers never reach Balaur directly.

The follow-up mandate this plan executes
(`2026-06-30-messenger-gateway-spike.md:257-260`):

> - A reference bridge implementation (polling mode, single transport) as a
>   separate, owner-run process — not embedded in Balaur.
> - Update `internal/self/knowledge.md` only when the bridge ships; "future
>   messengers" stays roadmap until then.

The disqualified designs — the design note must not recommend any of these
(`2026-06-30-messenger-gateway-spike.md:205-218`, abbreviated):

> - **Balaur exposes a public webhook endpoint** receiving messages directly
>   from the platform's servers: violates "Not an internet-exposed service."
> - **Turn content is sent to a platform's cloud** as message payloads (…)
> - **The bridge is a cloud function** intermediating between platform and
>   Balaur: the data path leaves the owner's box infrastructure.
> - **Auto-enabled by default** without an explicit owner confirmation (…)

The privacy fact that drives the consent copy (spike §3.2 data table,
`2026-06-30-messenger-gateway-spike.md:191`): the owner's message text and
Balaur's reply DO transit the messaging platform's servers — "the messaging
platform sees the message before it leaves the phone".

Product framing (`PRODUCT.md:108-109` and `PRODUCT.md:152-154`):

> - **Not an internet-exposed service.** Loopback-first; reaching it remotely is
>   the owner's explicit, deliberate act, never a default.

> - **More gateways, same pipeline.** A messenger or CLI surface that adapts the
>   one shared turn pipeline (`internal/turn`) rather than re-implementing it —
>   meeting the owner where they already are without forking behavior.

### Where a subcommand would live

`internal/cli/cli.go:52-76` is the single CLI registration point:

```go
// Register mounts the Balaur CLI on the root command.
func Register(app core.App, root *cobra.Command) {
	root.AddCommand(
		chatCmd(app),
		taskCmd(app),
		...
		doctorCmd(app),
		seedCmd(app),
	)
}
```

Two constraints on adding a `bridge` command here:

1. **Do NOT wrap it in `run(app, kind, body)`** (`internal/cli/cli.go:85-103`).
   That wrapper applies migrations (`app.RunAllMigrations()`, `cli.go:94`) and
   emits a one-shot JSON envelope — both wrong for a long-running process that
   deliberately does not use the database. The bridge command gets its own
   `RunE`. On failure it should call `failJSON(cmd, err)`
   (`internal/cli/cli.go:119-123`) so the process exits non-zero through the
   existing `cli.ExitCode()` path (`main.go:144` — `os.Exit(cli.ExitCode())`),
   since PocketBase's `Execute` discards `RunE` errors (see the comment at
   `cli.go:41-46`). Note this deviates from the package doc's "Every command
   prints one JSON envelope on stdout" (`cli.go:4`) — the bridge command's own
   doc comment must state the deviation and why (long-running process, output
   is a structured log stream, not one envelope).

2. **The no-args launcher cannot fire.** `main.go:53` gates the launcher on
   `launch.IsLauncherInvocation(os.Args[1:])`, and
   `internal/launch/launch.go:56-58` is:

   ```go
   func IsLauncherInvocation(args []string) bool {
   	return len(args) == 0
   }
   ```

   Any argument (`bridge`) ⇒ hands off. **No `main.go` restructuring is needed
   or allowed** — `balaur bridge telegram` flows through cobra like every
   existing CLI verb.

3. **PocketBase bootstraps before `RunE` runs.** In the pinned dependency
   (`~/go/pkg/mod/github.com/pocketbase/pocketbase@v0.39.3/pocketbase.go:180-181`):

   ```go
   	if !pb.skipBootstrap() {
   		if err := pb.Bootstrap(); err != nil {
   ```

   `skipBootstrap` skips only for `-h/--help/-v/--version`, unknown commands,
   and the default help/version commands (`pocketbase.go:251-267`). So running
   `balaur bridge telegram` opens (and, on a fresh dir, creates) the data
   directory as a process-level side effect. The bridge CODE must still read
   zero records and zero owner settings — its config comes from flags and env
   only — and the design note must record this bootstrap side effect as an
   accepted spike limitation. (`--dir` is a root PocketBase flag and works on
   any subcommand; use it in manual checks to avoid touching the real
   `pb_data/`.)

### The docs that must NOT change

`internal/self/knowledge.md:70-73` currently says:

```
- Gateways adapt, they never re-implement. The web UI (internal/web) and
  the CLI (internal/cli) both run internal/turn and only render its
  events in their medium. Future gateways (messengers) follow the same
  rule.
```

Per the spike's settled decision (quoted above,
`2026-06-30-messenger-gateway-spike.md:259-260`), knowledge.md is updated
"only when the bridge ships; 'future messengers' stays roadmap until then".
An experimental spike prototype is not "shipped" — **do not touch
knowledge.md in this plan**, and do not add messenger claims to `README.md`
or `PRODUCT.md`.

### Tour anchors that WILL shift

`.tours/` files are maintained artifacts (`tours_test.go` fails the suite on
missing files / out-of-range lines; the repo convention additionally requires
fixing anchors your change shifts and prose your change falsifies).

- `.tours/10-the-cli-api.tour` anchors `internal/cli/cli.go` at lines 1, 53,
  84 (step 10.4, `run`), 105 (step 10.3, `envelope`), and 47 (step 10.5,
  `exitCode`). Adding one `bridgeCmd(),` line inside `Register`'s
  `AddCommand` list shifts everything after it by +1: anchors 84→85 and
  105→106 must be bumped (anchors 1, 47, 53 are before the insertion and stay).
  Step 10.2's description also enumerates the command roster and says
  "`Register` mounts 19 top-level commands", and the tour-level `description`
  (line 4 of the tour file) repeats the count: "the full command roster (19
  commands across chat, tasks, knowledge, life, export, and infra)". Both
  counts must be updated together.
- `.tours/00-orientation.tour` step at line 64 enumerates the roster in prose:
  "The commands registered in `cli.Register` are `chat`, `task`, …, `doctor`,
  `seed`." (Note: this list — and tour 10.2's — already omits `restore`,
  which IS registered at `cli.go:72`; that staleness predates this plan.)

### Repo conventions that bind this plan (quoted so you don't guess)

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code.
- No global mutable state; pass `core.App`/config explicitly. Structured
  logging only (slog key/value); no `fmt.Print*` in service code. The bridge
  runs as its own process outside the serve app, so pass a `*slog.Logger`
  explicitly into `bridge.Run` (construct it in the cobra command with
  `slog.New(slog.NewTextHandler(os.Stderr, nil))`).
- Tests: std `testing` package, table-driven where it helps; no
  `time.Sleep`-based synchronization (use channels / injected tiny backoff
  durations). The bridge tests here need NO PocketBase app and NO
  `llm.Client` — pure `net/http/httptest` fakes.
- "Sanitize errors and tool output so they do not leak private paths, tokens,
  or vault content" — the Telegram bot token is embedded in every Bot API URL
  path (`/bot<TOKEN>/getUpdates`), so **never log a full Telegram URL**, and
  never log message text (it is the owner's private content).
- Every new dependency must justify itself; "prefer copying 30 lines over
  importing 3,000" — this prototype is **stdlib only**, `go.mod` must not
  change.
- KISS/YAGNI: smallest correct change; this is a spike, resist polish.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all packages `ok` |
| Targeted bridge tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/bridge/ -run TestBridge -count=1 -v` | all pass |
| Vet | `go vet ./...` | exit 0, no output |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours lint (after Step 5) | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | `ok` |

(The `TMPDIR` override is required: the host `/tmp` is a small tmpfs and the
Go linker OOMs there. `make test` exports it automatically but is cached; the
`-count=1` uncached form above is the gate.)

## Suggested executor toolkit

- Invoke the `go-standards` skill (if available) before writing
  `internal/bridge` — it covers this repo's error-wrapping, slog, and testing
  idioms.
- Read fully before starting:
  `docs/superpowers/specs/2026-06-30-messenger-gateway-spike.md` (the whole
  file), `internal/web/messenger.go`, `PRODUCT.md:100-158`.

## Scope

**In scope** (the only files you should create/modify):

- `docs/superpowers/specs/<execution-date>-messenger-bridge-design.md`
  (create — use TODAY's date as the prefix, e.g. `2026-07-05-…`, matching the
  existing `YYYY-MM-DD-slug.md` convention in that directory)
- `internal/bridge/bridge.go` (create)
- `internal/bridge/bridge_test.go` (create)
- `internal/cli/bridge.go` (create)
- `internal/cli/cli.go` (one added registration line inside `Register`)
- `.tours/10-the-cli-api.tour` (anchor bumps + roster prose, Step 5)
- `.tours/00-orientation.tour` (roster prose, Step 5)
- `plans/README.md` (status row for this plan only)

**Out of scope** (do NOT touch, even though they look related):

- `internal/web/messenger.go`, `internal/web/messenger_test.go`,
  `internal/web/messenger_settings.go`, `internal/web/web.go` — the endpoint
  is shipped and reviewed; this plan only consumes its contract.
- `internal/self/knowledge.md`, `README.md`, `PRODUCT.md`, `AGENTS.md` — the
  spike's settled decision: messenger support "stays roadmap" until the bridge
  graduates from spike (`2026-06-30-messenger-gateway-spike.md:259-260`).
- `main.go`, `internal/launch/` — the launcher/argv path must not be
  restructured (STOP condition if it seems necessary).
- `internal/feature/settingscards/` — no bridge status UI in the spike.
- `go.mod`, `go.sum` — no new dependencies, stdlib only.
- Any webhook mode, any Matrix (or other non-recommended transport)
  implementation, anything default-on.

## Git workflow

- Work in an isolated git worktree branched from `origin/main`; branch name
  `advisor/256-messenger-bridge-spike`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`).
  Commit per logical unit, e.g.:
  - `docs: messenger bridge design note (plan 256 spike)`
  - `feat(bridge): experimental telegram reference bridge + balaur bridge telegram`
  - `chore(tours): repoint cli.go anchors + command roster for bridge cmd`
- Stage with explicit pathspecs only (`git add docs/superpowers/specs/<file> internal/bridge/ internal/cli/bridge.go internal/cli/cli.go …`) —
  the main checkout is shared by parallel agents; never `git add -A`.
- Do NOT run `graphify update` — executor worktrees must not pollute
  `graphify-out/`; the reviewer regenerates the graph after merge.
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Write the design note

Create `docs/superpowers/specs/<execution-date>-messenger-bridge-design.md`
(match the header style of `2026-06-30-messenger-gateway-spike.md`: title,
plan reference "Plan 256 spike", date, `**Status:** DECIDED — prototype
shipped as experimental`). It must contain exactly these `##` sections, each
settling its decision with the arguments below (expand them; do not just copy
the bullet):

1. `## Context & constraints` — one paragraph: what exists
   (`POST /api/messenger/turn`, its contract as quoted in this plan's
   "Current state"), what's missing (no bridge), and the four disqualified
   designs from `2026-06-30-messenger-gateway-spike.md:205-218` (public
   webhook, turn content to a platform cloud, cloud-function bridge,
   auto-enabled default) restated as hard constraints.

2. `## Transport decision: Telegram Bot API long-polling` — evaluate
   **Telegram Bot API `getUpdates` long-polling** (recommended) against
   **Matrix** (the alternative). Required content:
   - Telegram pros: pure outbound polling — no inbound webhook, so
     loopback-first compliant per the spike's §3.3 verdict; free; a
     two-endpoint JSON-over-HTTPS API (`getUpdates`, `sendMessage`)
     implementable with stdlib `net/http` alone; the owner's phone already
     has the client.
   - Telegram cons (state honestly): message text and replies transit
     Telegram's servers (spike §3.2); the bot token is a
     credential for reading those messages; Telegram is a US/UAE-jurisdiction
     platform, so this is a knowing sovereignty trade the consent copy must
     surface.
   - Matrix: self-hostable (sovereignty win) but substantially more protocol
     surface (`/sync` tokens, rooms, device management), and meaningful E2E
     requires olm/megolm — not stdlib-feasible, violating the no-new-deps
     rule for a spike. Verdict: second transport candidate, not first.
   - Recommendation: Telegram long-polling for the reference bridge.

3. `## Packaging decision: same binary, own process` — decide
   `balaur bridge telegram` (a subcommand of the ONE `balaur` executable, run
   by the owner as its OWN separate process) versus a second compiled binary.
   Required argument: the spike's follow-up line says the bridge is "a
   separate, owner-run process — not embedded in Balaur"
   (`2026-06-30-messenger-gateway-spike.md:257-258`); what matters there is
   **process separation** (the bridge is not registered in `OnServe`, holds no
   turn state, dies independently, and can be killed without touching the
   companion), not binary separation. A second binary would break AGENTS.md's
   product shape ("Balaur ships as a standalone executable named `balaur`")
   and the PRODUCT.md "single standalone executable" bet. Record the accepted
   limitation: because the subcommand rides PocketBase's root command,
   PocketBase bootstraps (opens/creates the data dir) before `RunE`
   (`pocketbase@v0.39.3/pocketbase.go:180-181`), even though the bridge code
   itself reads no records and no settings. Recommendation: subcommand, own
   process, DB-free bridge code, config via env/flags only.

4. `## Config & consent` — decide where each credential lives:
   - Balaur side: `owner_settings.messenger_token`, set by the owner in
     Settings (`internal/web/messenger_settings.go`) — unchanged.
   - Bridge side: `BALAUR_TELEGRAM_BOT_TOKEN` and `BALAUR_MESSENGER_TOKEN`
     environment variables (env, not flags — flags leak into `ps` output and
     shell history), plus flags `--balaur-url` (default
     `http://127.0.0.1:8090`) and `--allow-chat` (repeatable int64, REQUIRED).
     Rationale: env/flags keep the bridge decoupled from the DB even though
     the same binary could read `owner_settings` — the bridge process
     deliberately does not.
   - The informed-consent copy, verbatim (printed at startup and in `--help`):
     "EXPERIMENTAL: message text you send to the bot, and Balaur's replies,
     transit Telegram's servers. Do not use this bridge for content that must
     never leave your infrastructure." (Grounded in spike §3.2.)

5. `## Sender authentication: owner-chat-id allowlist, fail-closed` — this is
   CRITICAL: anyone who discovers the bot's handle can message it, and the
   Bearer token only authenticates the bridge to Balaur, not the sender to
   the bridge. The design MUST require: the bridge refuses to start with an
   empty allowlist; every update whose `message.chat.id` is not allowlisted
   is dropped without contacting Balaur; the rejected chat id (never the
   text) is logged at Info so the owner can discover their own id on first
   contact and add it.

6. `## Failure & delivery semantics` — decide and record:
   - `getUpdates` long-poll `timeout` ≈ 50s; on poll error, exponential
     backoff (base 1s, cap 60s), keep looping.
   - On Balaur `429 busy` (in-flight turn, `messenger.go:83-87`): bounded
     retries (5 attempts, exponential backoff from the same base) for the
     SAME message, then give up and send the owner a plain "Balaur is busy —
     try again in a moment." reply.
   - Delivery semantics, stated honestly: within one bridge run, each update
     triggers at most one turn (the offset advances after handling,
     success or not). Across a crash/restart, Telegram redelivers updates not
     yet acknowledged by an advanced `offset`, so a message whose turn ran
     just before a crash can run a second turn — **at-least-once across
     restarts, at-most-once within a run**. Accepted for the spike; a
     persisted `update_id` high-water mark is the graduation fix.
   - Reply delivery (`sendMessage`) failure: one retry, then log and move on
     (the turn is already persisted in Balaur; the owner can see it in the
     web UI).

7. `## Open questions for the owner` — at minimum: group-chat behavior
   (spike handles only allowlisted private chats — should groups ever be
   supported?); multiple allowed devices/accounts; streaming replies via
   `editMessageText` vs one final message; media/attachments (out of spike);
   whether the Settings page should show bridge liveness; whether the
   graduation version persists the `update_id` high-water mark to make
   delivery exactly-once-ish; Telegram data-retention implications for the
   consent copy.

**Verify**:
`ls docs/superpowers/specs/ | grep messenger-bridge-design` → one file;
`grep -c '^## ' docs/superpowers/specs/*messenger-bridge-design.md` → `7`.

### Step 2: Implement `internal/bridge` (the prototype core)

Create `internal/bridge/bridge.go` — target ~200 lines, stdlib imports only
(`context`, `encoding/json`, `errors`, `fmt`, `io`, `log/slog`, `net/http`,
`net/url`, `strconv`, `time`). Shape (load-bearing; adjust details, keep the
seams):

```go
// Package bridge is the EXPERIMENTAL reference messenger bridge (plan 256
// spike): an owner-run process that long-polls the Telegram Bot API and
// relays allowlisted messages to a local Balaur's POST /api/messenger/turn.
// It deliberately reads no PocketBase records — all config arrives via
// Config (flags/env in internal/cli/bridge.go). See
// docs/superpowers/specs/<date>-messenger-bridge-design.md.
package bridge

type Config struct {
	BotToken        string        // Telegram bot token (secret — never logged, URLs containing it never logged)
	MessengerToken  string        // Balaur owner_settings.messenger_token value (secret — never logged)
	BalaurURL       string        // e.g. http://127.0.0.1:8090
	TelegramBaseURL string        // default https://api.telegram.org; tests point it at an httptest server
	AllowedChatIDs  []int64       // fail-closed sender allowlist; empty ⇒ Run refuses to start
	PollTimeout     time.Duration // getUpdates long-poll timeout (default 50s)
	RetryBase       time.Duration // backoff base (default 1s; tests use ~1ms)
	HTTP            *http.Client  // default: &http.Client{Timeout: PollTimeout + 10s}
}

// Run polls until ctx is cancelled; cancellation is the graceful shutdown
// path and returns nil.
func Run(ctx context.Context, cfg Config, log *slog.Logger) error
```

Behavior of `Run`:

1. Validate config, return early errors wrapped per convention
   (`fmt.Errorf("bridge config: %w", err)`-style): missing `BotToken`,
   missing `MessengerToken`, missing `BalaurURL`, and — fail-closed — empty
   `AllowedChatIDs` (`errors.New("bridge: --allow-chat is required; refusing to start without a sender allowlist")`).
   Apply defaults for the zero-valued optional fields.
2. Loop: `GET {TelegramBaseURL}/bot{BotToken}/getUpdates?timeout={sec}&offset={n}`.
   Response shape to decode (this Bot API shape is stable and is also what the
   test fakes must emit):
   ```json
   {"ok":true,"result":[{"update_id":123,
     "message":{"message_id":1,"chat":{"id":111,"type":"private"},"text":"hello"}}]}
   ```
   On HTTP/decode error: log at Warn (no URL — it embeds the token; log only
   the operation name and error), exponential backoff (RetryBase doubling,
   cap 60×RetryBase), continue. Honor `ctx.Done()` in every wait (use a
   `time.Timer` + `select`, never bare `time.Sleep`, so shutdown is prompt).
3. Per update, in order: skip if `message` is nil or `text` is empty (advance
   offset). If `chat.id` is not in `AllowedChatIDs`: log
   `log.Info("bridge: rejected sender", "chat_id", id)` — never the text —
   and advance the offset WITHOUT contacting Balaur.
4. For an allowlisted message: `POST {BalaurURL}/api/messenger/turn` with
   header `Authorization: Bearer {MessengerToken}`, JSON body
   `{"message": text}`.
   - `200`: decode `{"reply":"..."}`, deliver via
     `POST {TelegramBaseURL}/bot{BotToken}/sendMessage` with JSON
     `{"chat_id": id, "text": reply}` (one retry on failure, then log Warn
     and move on).
   - `429`: retry the same POST up to 5 times with exponential backoff from
     `RetryBase`; if still busy, `sendMessage` a plain
     `"Balaur is busy — try again in a moment."`.
   - any other non-200: log Warn (status only), `sendMessage` a generic
     `"Something went wrong — check Balaur's logs."`. Never forward raw error
     bodies to Telegram.
5. Advance `offset = update_id + 1` after handling each update, then loop.
   `ctx` cancelled ⇒ return nil.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `gofmt -l .` → empty;
`grep -En "github.com|golang.org/x" internal/bridge/bridge.go` → no matches
(stdlib only).

### Step 3: Tests for the bridge core

Create `internal/bridge/bridge_test.go`. No PocketBase app, no `llm.Client`,
no real network — two `httptest.NewServer` fakes per test:

- **fake Telegram**: a handler that serves
  `/bot<token>/getUpdates` from a scripted queue of update batches (then
  empty batches) and records `/bot<token>/sendMessage` bodies onto a channel.
- **fake Balaur**: a handler for `/api/messenger/turn` that asserts the
  `Authorization: Bearer <token>` header and returns scripted responses.

Point `Config.TelegramBaseURL`/`Config.BalaurURL` at the fakes, set
`RetryBase: time.Millisecond`, `PollTimeout: 0` (or a few ms), and drive
`Run` in a goroutine; synchronize on the sendMessage channel, then cancel the
context and wait for `Run` to return (no `time.Sleep` sync). Tests to write
(names are load-bearing for the Done criteria):

1. `TestBridgeHappyPath` — one allowlisted update; fake Balaur returns
   `200 {"reply":"Hello from Balaur"}`; assert the fake Balaur saw
   `Authorization: Bearer <token>` and body `{"message":"hi"}`; assert
   `sendMessage` received `chat_id` = the sender and `text` =
   `Hello from Balaur`; assert the second `getUpdates` call carried
   `offset=update_id+1`.
2. `TestBridgeBusyRetry` — fake Balaur returns `429 {"error":"busy"}` once,
   then `200`; assert exactly 2 POSTs to Balaur and the reply is delivered.
3. `TestBridgeAllowlistRejects` — update from a non-allowlisted `chat.id`;
   assert the fake Balaur handler was hit **0 times** and the offset still
   advanced.
4. `TestBridgeEmptyAllowlistFails` — `Run` with empty `AllowedChatIDs`
   returns a non-nil error mentioning the allowlist, before any HTTP call
   (fakes hit 0 times).
5. `TestBridgeGracefulShutdown` — cancel the context while polling; `Run`
   returns nil promptly (guard with a `select` on a done channel and
   `t.Fatal` on timeout via `time.After(5 * time.Second)` as the failure
   escape, not as sync).

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/bridge/ -run TestBridge -count=1 -v`
→ 5 tests, all `PASS`.

### Step 4: The `balaur bridge telegram` subcommand

Create `internal/cli/bridge.go` with `bridgeCmd() *cobra.Command` (it takes no
`core.App` — the bridge is DB-free by design; a doc comment must say so and
note the deviation from the package's one-JSON-envelope contract). Shape:

- Parent `bridge` command (`Use: "bridge"`, `Short:` mentions EXPERIMENTAL)
  with one subcommand `telegram`:
  - `Short: "EXPERIMENTAL: relay an allowlisted Telegram chat to this box's Balaur"`.
  - `Long:` includes the verbatim consent sentence from Step 1 §4, the two
    required env vars (`BALAUR_TELEGRAM_BOT_TOKEN`, `BALAUR_MESSENGER_TOKEN`
    — names only, never values), and a pointer to the design note path.
  - Flags: `--balaur-url` (string, default `http://127.0.0.1:8090`),
    `--allow-chat` (`Int64SliceVar`, required — enforce non-empty in `RunE`,
    the fail-closed check also lives in `bridge.Run`), `--poll-timeout`
    (duration, default `50s`).
  - `RunE`: `cmd.SilenceUsage = true`; read the two env vars (error if either
    is empty, via `failJSON(cmd, err)` so the exit code is non-zero); build
    `bridge.Config`; `logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))`;
    print the consent sentence once to stderr via the logger; wrap
    `cmd.Context()` with `signal.NotifyContext(..., os.Interrupt, syscall.SIGTERM)`
    for graceful shutdown; call `bridge.Run(ctx, cfg, logger)`; on non-nil
    error return `failJSON(cmd, err)`.

Then register it: in `internal/cli/cli.go`, add `bridgeCmd(),` as the LAST
entry of the `root.AddCommand(...)` list in `Register` (after
`seedCmd(app),`, `cli.go:74`). One line — nothing else in `cli.go` changes.

**Verify** (run from the worktree root; the scratch `--dir` keeps PocketBase's
bootstrap side effect away from any real data dir):

- `CGO_ENABLED=0 go build ./...` → exit 0.
- `TMPDIR=$HOME/.cache/go-tmp go run . bridge telegram --help 2>&1 | grep -c "EXPERIMENTAL"`
  → `1` or more (note: `--help` skips PocketBase bootstrap entirely).
- `out="$(TMPDIR=$HOME/.cache/go-tmp go run . bridge telegram --allow-chat 1 --dir /tmp/claude-1000/bridge-smoke 2>&1)"; echo "exit=$?"; printf '%s\n' "$out" | head -3`
  → `exit=1` (capture the output first so `$?` is `go run`'s own status —
  piping straight into `head` would report `head`'s exit 0 instead), then a
  JSON error envelope naming the missing env var (name only, no value).

### Step 5: Fix the tour anchors and roster prose your change shifted

In `.tours/10-the-cli-api.tour`:
- Bump the two anchors that now point one line high: the step titled
  "10.3 — envelope: the v1 wire shape" from `"line": 105` to `"line": 106`,
  and "10.4 — run: the command middleware" from `"line": 84` to `"line": 85`.
  (Confirm against the post-edit `internal/cli/cli.go` — if you inserted the
  registration line elsewhere, recompute.)
- In step 10.2's description, update the `AddCommand` code block and the
  "mounts 19 top-level commands" count to match the real roster (with
  `bridgeCmd()`, and note the block also omitted the pre-existing
  `restoreCmd(app)` — include it so the quoted block matches reality: 21
  commands). Add `bridge` under the "Infra / ops" group.
- In the tour-level `description` field (line 4 of the tour file), update the
  same stale count: "the full command roster (19 commands across chat, tasks,
  knowledge, life, export, and infra)" → "(21 commands …)". Leaving it at 19
  would contradict the corrected step 10.2 in the same file.

In `.tours/00-orientation.tour` (the step anchored at `internal/cli`, source
line 64 of the tour file): extend the roster sentence "The commands registered
in `cli.Register` are …" to include `restore` (pre-existing omission — you are
editing this exact sentence anyway, so make it true) and `bridge`.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` →
`ok`; `grep -c '"line": 106' .tours/10-the-cli-api.tour` → `1`;
`grep -c '19 commands\|19 top-level commands' .tours/10-the-cli-api.tour` → `0`
(no stale count survives, in either the tour header or step 10.2).

### Step 6: Full gates + index row

Run, in order:

1. `gofmt -l .` → empty
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
6. `git diff --check` → no output
7. `git status --short` → only the in-scope files
8. `git diff --name-only -- go.mod go.sum internal/self/knowledge.md internal/web` → empty

Then add/update the plan-256 status row in `plans/README.md` (one row, same
table format as the surrounding entries; do not reflow other rows) — unless
the dispatching reviewer said they maintain the index.

## Test plan

- New tests (all in `internal/bridge/bridge_test.go`, all against
  `httptest` fakes of BOTH the Telegram Bot API and the Balaur endpoint —
  no PocketBase, no `llm.Client`, no live network, no live bot token, no
  `time.Sleep` synchronization):
  - `TestBridgeHappyPath` — poll → authed POST → reply delivered → offset advanced.
  - `TestBridgeBusyRetry` — the `429 {"error":"busy"}` retry path
    (mirrors `internal/web/messenger.go:83-87`).
  - `TestBridgeAllowlistRejects` — non-allowlisted sender never reaches Balaur.
  - `TestBridgeEmptyAllowlistFails` — fail-closed startup.
  - `TestBridgeGracefulShutdown` — ctx cancel returns nil promptly.
- Structural pattern to model after: `internal/web/messenger_test.go` (its
  scripted-response + channel-synchronization style, e.g. the
  `messengerBlockingClient` `started`/`release` channels at
  `messenger_test.go:23-49`), except with `httptest.NewServer` instead of the
  PocketBase router.
- No existing tests should change; `internal/web/messenger_test.go` must pass
  untouched.
- Verification: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/bridge/ -run TestBridge -count=1 -v`
  → 5 PASS; then the full gate.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `ls docs/superpowers/specs/ | grep -c 'messenger-bridge-design.md'` → `1`,
      and `grep -c '^## ' docs/superpowers/specs/*messenger-bridge-design.md` → `7`
- [ ] `grep -En "github.com|golang.org/x" internal/bridge/bridge.go` → no matches
      (stdlib-only prototype)
- [ ] `git diff --name-only -- go.mod go.sum` → empty (no new dependencies)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/bridge/ -run TestBridge -count=1` →
      exit 0 with the 5 named tests present
      (`grep -c "^func TestBridge" internal/bridge/bridge_test.go` → `5`)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go run . bridge telegram --help 2>&1 | grep -c EXPERIMENTAL` → ≥ `1`
- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output;
      `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0 (includes `TestTours`)
- [ ] `git diff --name-only -- internal/self/knowledge.md internal/web README.md PRODUCT.md main.go internal/launch` → empty
- [ ] `git status --short` lists ONLY files from the in-scope list
- [ ] `plans/README.md` status row for plan 256 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The excerpts of `internal/web/messenger.go` in "Current state" no longer
  match the live file — especially the request body shape (`{"message"}`),
  the `200 {"reply"}` response, or the `429 {"error":"busy"}` busy signal.
  The bridge is built against that contract; drift there invalidates the
  design note and the fakes.
- Making `balaur bridge telegram` reachable appears to require editing
  `main.go` or `internal/launch/` (launcher/argv restructuring), or cobra
  rejects the `bridge` command name because PocketBase already reserves it.
- Running the subcommand with a scratch `--dir` does anything beyond
  PocketBase's normal bootstrap (creating the empty data dir / opening the
  DB) — e.g. it applies migrations or writes records. The bridge must be
  DB-free; if avoiding a write requires touching `main.go`, STOP.
- Any test appears to need a live Telegram bot token, a live messenger
  token, or real network access to pass — everything must run against
  `httptest` fakes; if you cannot fake it, STOP (never wire a secret or a
  network call into CI).
- The prototype cannot stay stdlib-only (you find yourself wanting a
  Telegram client library or adding anything to `go.mod`).
- A verification command fails twice after a reasonable fix attempt.

## Maintenance notes

- **This is a spike.** Graduation (a follow-up plan, not this one) is what
  earns: the `internal/self/knowledge.md` and README/PRODUCT copy updates
  (per `2026-06-30-messenger-gateway-spike.md:259-260`), a persisted
  `update_id` high-water mark (fixes the at-least-once-across-restarts
  duplicate-turn window documented in the design note), streaming replies via
  `editMessageText`, a Settings-page bridge-status affordance, and a possible
  Matrix transport. Until then the `--help` text and package doc keep the
  EXPERIMENTAL label.
- **Reviewer scrutiny points**: (1) no log line can ever contain the bot
  token (it is embedded in every Telegram URL path — check every `slog` call
  and every wrapped error for URLs), the messenger token, or message text;
  (2) the allowlist is genuinely fail-closed (empty ⇒ refuse to start;
  rejection happens before the Balaur POST); (3) `internal/bridge` imports no
  PocketBase packages and reads no DB; (4) `go.mod` untouched.
- If `internal/web/messenger.go` later changes its response shape or status
  codes, `internal/bridge` and its fakes must change in the same commit —
  they are two halves of one wire contract with no shared type (deliberately:
  the bridge stays DB- and web-package-free). Consider extracting a tiny
  shared wire-types file only if a third consumer appears (YAGNI until then).
- The tour rosters in `.tours/00-orientation.tour` and
  `.tours/10-the-cli-api.tour` enumerate CLI commands by hand and go stale on
  every `Register` change (they had already dropped `restore` before this
  plan); whoever adds the next command should budget the same prose fix.
