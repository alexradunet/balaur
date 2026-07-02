# Messenger bridge design — transport, packaging, consent, failure semantics

**Plan 256 spike** · 2026-07-02
**Status:** DECIDED — prototype shipped as experimental

---

## Context & constraints

Balaur ships a live, token-gated, guard-protected messenger endpoint —
`POST /api/messenger/turn` in `internal/web/messenger.go` — plus an owner
Settings control for its token (`internal/web/messenger_settings.go`) and a
synchronous `{"reply": "..."}` response designed for a poll→POST→send loop.
Its verified wire contract: `POST <balaur-url>/api/messenger/turn` with
header `Authorization: Bearer <messenger_token>` and JSON body
`{"message": "..."}`; on success `200 {"reply": "..."}`; `403
{"error":"messenger gateway is not enabled"}` when `owner_settings.
messenger_token` is empty; `401 {"error":"unauthorized"}` on a bad token;
`429 {"error":"busy"}` while another turn is in flight
(`turn.TryBegin`/`messenger.go:83-87`); `503` when no model is active; `500`
on a failed turn. What is missing is a bridge process: nothing polls a
messaging platform, nothing POSTs to this endpoint, nothing delivers a reply
to the owner's phone. The Phase-0 feasibility spike
(`docs/superpowers/specs/2026-06-30-messenger-gateway-spike.md`) green-lit
the gateway and named "a reference bridge implementation (polling mode,
single transport) as a separate, owner-run process — not embedded in
Balaur" as the follow-up. This document settles that follow-up's design
decisions; the accompanying `internal/bridge` package is the thin
prototype.

Four designs are disqualified as hard constraints, carried forward from the
Phase-0 spike (`2026-06-30-messenger-gateway-spike.md:205-218`):

- **Balaur exposes a public webhook endpoint** receiving messages directly
  from the platform's servers — violates "Not an internet-exposed service"
  (`PRODUCT.md:108-109`). Balaur binds no new non-loopback listener; this
  bridge, and any future one, must only ever poll outward.
- **Turn content is sent to a platform's cloud** as message payloads (e.g.
  forwarding conversation summaries or memories to a bot API for context) —
  violates sovereignty and "a turn never silently leaves the box."
- **The bridge is a cloud function** intermediating between the platform and
  Balaur — the data path would leave the owner's box infrastructure
  entirely; the bridge must run on-box (or on a trusted NetBird-mesh
  device) and talk to Balaur over loopback.
- **Auto-enabled by default** without explicit owner confirmation — the
  messenger gateway is already fail-closed (empty `messenger_token` ⇒ 403);
  the bridge adds its own fail-closed gate (the sender allowlist, below) and
  is never started implicitly.

## Transport decision: Telegram Bot API long-polling

**Recommendation: Telegram Bot API `getUpdates` long-polling.**

Telegram pros: `getUpdates` is pure outbound polling — the bridge reaches
out to `api.telegram.org`, Telegram never reaches Balaur or the bridge, so
this is loopback-first compliant per the Phase-0 spike's §3.3 verdict ("the
bridge polls (not receives) platform messages"). It is free, requires no
platform approval process, and is a two-endpoint JSON-over-HTTPS API
(`getUpdates` to receive, `sendMessage` to reply) fully implementable with
stdlib `net/http` and `encoding/json` alone — no SDK, no new `go.mod`
dependency. The owner's phone already has a Telegram client installed or a
one-tap install away, so there is no bridge-side pairing UX to build.

Telegram cons, stated honestly: the owner's message text and Balaur's reply
both transit Telegram's servers before/after crossing the loopback boundary
(Phase-0 spike §3.2 data table — "the messaging platform sees the message
before it leaves the phone"). The bot token is itself a credential capable
of reading those messages and must be handled as a secret with the same
care as the messenger token. Telegram is a US/UAE-jurisdiction platform, not
an EU-sovereign one — this is a knowing sovereignty trade that the
informed-consent copy (below) must surface plainly, distinct from Balaur's
EU-only cloud-model catalog policy (`internal/llm/presets.go`), which does
not apply here because Telegram is a messaging transport, not an inference
provider.

Matrix (the alternative considered): self-hostable, which would be a
genuine sovereignty win over Telegram. But it carries substantially more
protocol surface to implement correctly — `/sync` long-polling with
opaque since-tokens, room state, device/session management — and any
meaningful end-to-end encryption story requires olm/megolm cryptography
that is not stdlib-feasible; pulling in a Matrix SDK would violate the
spike's no-new-dependency rule ("prefer copying 30 lines over importing
3,000"). Verdict: Matrix is a credible second transport for a future
graduation slice, not the first reference implementation.

## Packaging decision: same binary, own process

**Recommendation: `balaur bridge telegram` as a subcommand of the one
`balaur` executable, run by the owner as its own separate OS process.**

The Phase-0 spike's follow-up line says the bridge is "a separate,
owner-run process — not embedded in Balaur"
(`2026-06-30-messenger-gateway-spike.md:257-258`). What matters there is
**process separation**, not binary separation: the bridge is never
registered in `OnServe`, holds no turn state, runs its own polling loop in
its own `os.Exec`'d process, dies independently of `serve`, and can be
killed without touching the companion server. A second compiled binary
would instead break two explicit product commitments — AGENTS.md's "Balaur
ships as a standalone executable named `balaur`" and PRODUCT.md's
single-standalone-executable bet — for no compensating benefit, since cobra
subcommands already give full process independence.

Accepted limitation: because `balaur bridge telegram` rides PocketBase's
root cobra command, PocketBase bootstraps (opens or creates the data
directory) before `RunE` runs
(`pocketbase@v0.39.3/pocketbase.go:180-181` — `skipBootstrap` only skips
for `-h`/`--help`/`-v`/`--version`/unknown/default commands). This is a
process-level side effect the bridge subcommand cannot opt out of without
restructuring `main.go`, which is explicitly out of scope for this spike.
The bridge's own code, however, reads zero PocketBase records and zero
owner settings — its configuration arrives entirely via CLI flags and
environment variables (`internal/cli/bridge.go` builds a `bridge.Config`
and hands it to `bridge.Run`, which imports no PocketBase package). The
data-directory-open side effect is a spike-scope accepted cost, not a
design flaw to fix here.

## Config & consent

Two credentials, two different lifecycles:

- **Balaur side**: `owner_settings.messenger_token`, set by the owner in
  Settings → Capabilities (`internal/web/messenger_settings.go`) —
  unchanged by this plan. This is the pre-shared secret that authenticates
  the bridge process to Balaur's endpoint.
- **Bridge side**: `BALAUR_TELEGRAM_BOT_TOKEN` and `BALAUR_MESSENGER_TOKEN`
  environment variables — deliberately env, not flags, because flag values
  leak into `ps aux` output and shell history in a way env vars set via a
  process manager or `.env`-sourcing wrapper do not. Two flags round out
  the config: `--balaur-url` (string, default `http://127.0.0.1:8090`) and
  `--allow-chat` (repeatable `int64`, REQUIRED — see the allowlist section
  below). Rationale for keeping the bridge DB-free even though the same
  binary could technically read `owner_settings`: the bridge process is a
  separate, less-trusted execution context (it talks to an external
  platform), and keeping it credential-scoped to exactly two secrets passed
  explicitly is easier to audit than letting it open the database.

The informed-consent copy, verbatim (printed once to stderr at startup, and
included in `--help`):

> EXPERIMENTAL: message text you send to the bot, and Balaur's replies,
> transit Telegram's servers. Do not use this bridge for content that must
> never leave your infrastructure.

Grounded directly in the Phase-0 spike's §3.2 data-boundary table.

## Sender authentication: owner-chat-id allowlist, fail-closed

This is the critical security property of the bridge. Anyone who discovers
the bot's public handle can send it a message — Telegram has no built-in
notion of "only my owner may talk to this bot." The Bearer token in the
Balaur wire contract authenticates the *bridge process* to *Balaur*; it says
nothing about who is allowed to talk to the *bridge*. Without a separate
gate, a stranger who finds the bot could inject arbitrary turns into the
owner's single master conversation.

The design therefore requires, and `internal/bridge` implements:

- The bridge refuses to start with an empty `AllowedChatIDs` — `bridge.Run`
  returns a non-nil error before making any HTTP call, fail-closed by
  construction (mirrors the messenger endpoint's own fail-closed pattern:
  no token configured ⇒ 403, no bridge allowlist configured ⇒ refuse to
  run).
- Every incoming update whose `message.chat.id` is not in the allowlist is
  dropped without ever contacting Balaur — no turn is spent, no owner data
  is touched, and the reject happens strictly before the `/api/messenger/
  turn` POST.
- The rejected chat id (never the message text) is logged at `Info` level,
  so on first contact from a not-yet-configured bot the owner can read
  their own numeric chat id out of the bridge's log stream and add it to
  `--allow-chat` — a workable bootstrap path without ever building a
  pairing UI.

## Failure & delivery semantics

- **Long-poll timeout**: `getUpdates` is called with `timeout≈50s` (the
  Telegram-recommended long-poll window, configurable via `--poll-timeout`,
  default `50s`). On any poll HTTP or decode error, the bridge backs off
  exponentially (base 1s, doubling, capped at 60s) and keeps looping — it
  never gives up permanently on a transient network blip.
- **On Balaur `429 busy`** (an in-flight turn already running,
  `messenger.go:83-87`): the bridge retries the *same* message up to 5
  times with exponential backoff from the same base, then gives up on that
  attempt and sends the owner a plain `"Balaur is busy — try again in a
  moment."` reply so the owner is never left wondering whether the message
  arrived.
- **Delivery semantics, stated honestly**: within a single bridge process
  run, each Telegram update triggers at most one turn — the offset always
  advances after handling an update, whether that handling succeeded or
  not. Across a bridge crash/restart, Telegram will redeliver any update
  not yet acknowledged by an advanced `offset`, so a message whose turn ran
  just before a crash (before the next `getUpdates` call could persist the
  advanced offset) can trigger a second turn on restart. Net: **at-least-
  once delivery across restarts, at-most-once within a single run**. This
  is accepted for the spike; a persisted `update_id` high-water mark
  (surviving process restarts) is the graduation fix.
- **Reply delivery (`sendMessage`) failure**: one retry, then log at `Warn`
  and move on — the turn itself is already durably persisted in Balaur's
  conversation, so the owner can always see the reply in the web UI even if
  the Telegram-side delivery silently failed.

## Open questions for the owner

- **Group-chat behavior**: the spike only ever handles allowlisted private
  chats. Should a future version ever accept a group chat (e.g. a family
  group), and if so, how does the allowlist model apply — per-chat or
  per-sender-within-chat?
- **Multiple allowed devices/accounts**: `--allow-chat` already accepts a
  list, but there is no design yet for how the owner would use one bot from
  two of their own devices/accounts versus accidentally allowlisting a
  second person.
- **Streaming replies**: should a graduated bridge use Telegram's
  `editMessageText` to progressively stream tokens into one message (matching
  the web UI's live streaming feel), instead of the spike's one-final-message
  delivery?
- **Media/attachments**: entirely out of spike scope — images, voice notes,
  and documents are all silently ignored (`message.text` empty ⇒ skipped).
- **Settings-page bridge liveness**: should Settings show whether a bridge
  process is currently polling (e.g. a last-seen-heartbeat record), or does
  that cross back into "the bridge writes to the DB," which this spike
  deliberately avoids?
- **Exactly-once-ish delivery**: does the graduation version persist the
  `update_id` high-water mark to disk (closing the at-least-once-across-
  restarts window described above), and if so, where — a flag-provided
  state file, or does that require the bridge to become DB-aware after all?
- **Telegram data-retention implications**: what does Telegram's own data
  retention policy imply for the consent copy — should the informed-consent
  message mention how long Telegram itself retains message content on its
  servers?
