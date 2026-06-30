# Messenger gateway spike — feasibility + threat-model finding

**Plan 228, Phase 0** · 2026-06-30  
**Status:** COMPLETE — Bet A

---

## 1. Gateway-line audit

### 1.1 The shared pipeline

`internal/turn/turn.go:69`

```go
func Run(ctx context.Context, app core.App, client llm.Client, userText string, emit func(agent.Event)) (Result, error)
```

Everything below the line runs once, identically, for every surface: persist
the user message, assemble context (system prompt + head flavor + now-line +
today block + knowledge block + summary + recent turns), run the agent loop
with streaming events to `emit`, apply the honesty check with one self-repair
pass, persist all assistant/tool rounds, touch used memories. The `Result`
carries the final reply text, all appended messages, the honesty-check note,
and the memories that were used.

### 1.2 Web gateway — `internal/web/chat.go`

**Claimed at `ef9f2df`: verified against current code (L28–61).**

Above the gateway line — what the web handler owns exclusively:

| Concern | Location | Notes |
|---|---|---|
| Message extraction + empty-body 400 | `chat.go:29–31` | `readChatMessage` reads JSON signals or form fallback |
| Presentation metadata load | `chat.go:34–38` | owner name, soul avatar, head name+avatar |
| LLM client resolution | `chat.go:41–48` | `h.clients.Active()`; error surfaced as a styled chat note, not an HTTP error |
| SSE upgrade + stream lifecycle | `chat.go:39`, `chatstream.go:66–76` | `newChatStream` wraps `datastar.NewSSE` |
| User message echo + streaming signal | `chatstream.go:122–135` | `cs.start()` appends the owner bubble, sets `streaming=true` |
| `emit` adapter → Datastar patches | `chatstream.go:168–195` | text tokens morph bubble body; tool_start/tool_result open/close tool cards; error event writes into bubble |
| Tool result routing | `chatstream.go:201–233` | UI card → panel, choices → inline panel, proposal → chip+card, refresh → card morph, error redaction |
| Honesty check note rendering | `chat.go:54–56` | appends styled `"check"` origin chat message |
| Error sanitization | `chat.go:79–108` | `chatErrText` redacts provider URLs; `chatToolErrText` logs raw + redacts paths/URLs |
| Stream finalization | `chatstream.go:314–319` | finalizes last bubble, sets `streaming=false` |
| CSRF + host guard | `web.go:64–103` | `guardLocalUI` middleware, applied globally before routes |

Nothing in this list is turn behavior. The handler's only call into the shared
pipeline is `turn.Run(e.Request.Context(), h.app, client, msg, cs.emit)` at
`chat.go:52`.

### 1.3 CLI gateway — `internal/cli/chat.go`

**Claimed at `ef9f2df`: verified against current code (L45–113).**

Above the gateway line:

| Concern | Location | Notes |
|---|---|---|
| Cobra command + timeout flag | `chat.go:58–59` | `--timeout` default 5m |
| CLI-specific Kronk engine init | `chat.go:26–31` | CLI runs outside OnServe; creates or reuses engine from store |
| LLM client resolution | `chat.go:63` | same `turn.ClientSource` pattern |
| Context + deadline | `chat.go:64–65` | `context.WithTimeout` from the flag |
| `emit` adapter → `[]toolEvent` slice | `chat.go:69–88` | accumulates tool_start/tool_result pairs; drops live-refresh marker (CLI has no UI to patch); extracts proposal kind+id |
| JSON serialisation of result | `chat.go:95–110` | reply, tools, verify claims, check_note, used_memories, messages_appended |

Only call into the pipeline: `turn.Run(ctx, app, client, args[0], emit)` at
`chat.go:90`.

### 1.4 `internal/kronk/runtime.go` — not a turn gateway

`runtime.go` (L1–26) exports `RuntimeInstalled()` and
`RuntimeInstalledFor(processor)`. Both are stat-checks only (they `os.Stat`
for `libllama.so`); they never call `turn.Run` and carry no conversation
logic. This is engine-presence plumbing, not a gateway.

### 1.5 Messenger-needed behaviors not yet below the line

Every item below is an above-the-line, legitimately gateway-specific concern —
not a duplicate of turn behavior.

| Behavior | Currently where | Classification |
|---|---|---|
| Rate limiting / in-flight guard | Absent everywhere | **Gap** — no concurrency guard at `turn.Run` level; two simultaneous inbound messages could race on `conversation.Append`. The web gateway relies on the browser not sending a second POST while `streaming=true`; a messenger ignores that signal. Must be addressed in the follow-up plan (gateway-level mutex or per-conversation queue). |
| Owner identity / auth | `guardLocalUI` (host + CSRF) | Legitimately gateway-specific; a messenger gateway needs its own auth token check |
| Async / fire-and-forget delivery | Absent; web holds the SSE connection open | Legitimately gateway-specific (see §2) |
| Message routing / thread IDs | Absent below the line | Legitimately gateway-specific; adapter maps platform IDs to the one master conversation |
| Outbound push to platform API | Absent | Legitimately gateway-specific |

Nothing in this list requires re-implementing behavior that lives in
`turn.Run`. Every gap is either "add a thin gateway-side mechanism" or "write
a new above-the-line adapter."

---

## 2. Async-fit assessment

`turn.Run` is synchronous: it blocks until the full turn completes and returns
a `Result`. A local model can take 30–120 seconds. A messenger surface is
fire-and-forget inbound (the platform's webhook or the local bridge's POST
expects a fast 200) and async outbound (the reply is delivered later, to the
platform API).

**Does `emit + Result` suffice without changing `turn.Run`'s contract?**

Yes. The adapter pattern is:

```
inbound POST → gateway returns 200 immediately
              → goroutine: turn.Run(backgroundCtx, app, client, text, emit)
                  → on return: deliver Result.Reply via outbound API call
```

Specifics:

- **Non-streaming delivery**: pass `nil` for `emit` (turn.Run substitutes a
  no-op internally, `turn.go:70–72`). Block in the goroutine; deliver
  `Result.Reply` when done. Simple, correct.

- **Streaming delivery** (for platforms that support incremental message
  edits, e.g., Telegram's `editMessageText`): `emit` accumulates `"text"`
  events and triggers a platform update per chunk. The `emit` contract is
  already designed for this — it is a plain `func(agent.Event)` callback with
  no channel or transport coupling.

- **Context lifetime**: the web gateway uses `e.Request.Context()`, which
  cancels if the client disconnects. The messenger goroutine must use a
  context tied to the app's lifecycle, not the inbound webhook request — a
  `context.WithCancel` derived from the PocketBase serve context, cancelled on
  terminate. This is an adapter concern, not a `turn.Run` concern.

- **Error handling**: `turn.Run` returns `(Result, error)` where `Result` is
  populated even on partial error. The adapter logs the error and delivers
  whatever `Result.Reply` is non-empty, or a short "something went wrong"
  message — exactly what the web gateway does (it surfaces errors as chat
  notes, not HTTP errors, `chat.go:57–59`).

No change to `turn.Run`'s signature or behavior is needed. The async wrapper
is a thin goroutine around an unchanged call.

---

## 3. Threat model

### 3.1 Constraint sources

- **PRODUCT.md (L108–109):** "Not an internet-exposed service. Loopback-first;
  reaching it remotely is the owner's explicit, deliberate act, never a
  default."
- **PRODUCT.md (L103–107):** "Not a cloud-model router by default … a turn
  never silently leaves the box."
- **NetBird ACL reality** (project memory): Balaur ports are gated by
  NetBird `table ip netbird` ACL rules; a mesh device reaching port 8090
  requires an explicit dashboard policy. The platform's servers cannot reach
  the box via this path.

### 3.2 The constraint-respecting design

The key insight from the plan: Balaur stays loopback-first; the owner
provides their own tunnel. Concretely:

**Listener binding.** No new port, no new non-loopback listener. The
messenger endpoint (`/api/messenger/turn`, for example) is registered on the
same PocketBase router as all other routes. `guardLocalUI` enforces that the
`Host` header is loopback or an owner-`BALAUR_ALLOWED_HOSTS` entry
(`web.go:105–122`). Any request reaching this endpoint arrived via the
loopback interface.

**The bridge runs on-box.** The owner runs a small bridge process on the same
box (or a trusted device on the NetBird mesh). The bridge polls the messaging
platform (polling mode, not accepting inbound webhooks from the platform's
servers) and POSTs messages to `127.0.0.1:8090/api/messenger/turn`. The
platform's servers never reach Balaur directly.

**Authentication.** Two layers:
1. `guardLocalUI` host check: requests not from loopback (or
   `BALAUR_ALLOWED_HOSTS`) are rejected before any route logic runs.
2. Pre-shared token: the gateway endpoint checks a short-lived or
   owner-configured token stored as an owner setting (same pattern as
   `store.GetOwnerSetting`). This prevents an untrusted local process from
   injecting turns without the owner's knowledge.

**Access from the owner's phone.** The owner reaches the bridge or Balaur
itself via the NetBird mesh overlay (the existing deployment pattern, per the
project memory). NetBird enforces E2E encryption between devices and the
box's ACL rules (`table ip netbird`) gate which ports are reachable from
which devices. The platform's cloud never sees the Balaur endpoint.

**What private data crosses which boundary:**

| Data | Path | Boundary |
|---|---|---|
| Owner's message text | Phone → NetBird E2E → box loopback | Stays on owner-controlled infrastructure; the messaging platform sees the message before it leaves the phone |
| Turn content (conversation body) | Never leaves the box | Box-local only |
| Balaur's reply | Box loopback → NetBird E2E → phone → platform | The owner chooses what to send to the platform; Balaur never calls the platform API directly |
| LLM inference | In-process (Kronk, local model) | Stays on-box unless the owner has opted in to a remote provider |

### 3.3 Explicit loopback-first verdict

**This design is loopback-first compliant.** Balaur binds no new non-loopback
listener. The bridge polls (not receives) platform messages. The platform
never reaches Balaur. Remote access from the owner's device is via an
owner-operated mesh (NetBird), not a public port. The owner's explicit,
deliberate act (running the bridge, configuring the token, setting up the mesh
policy) is the access path — not a default.

### 3.4 What would fail the threat model (STOP conditions)

The following designs are disqualified by PRODUCT.md and the plan's STOP
conditions:

- **Balaur exposes a public webhook endpoint** receiving messages directly
  from the platform's servers: violates "Not an internet-exposed service."
- **Turn content is sent to a platform's cloud** as message payloads (e.g.,
  forwarding conversation summaries to a bot API for context): violates
  sovereignty and the "a turn never silently leaves the box" rule.
- **The bridge is a cloud function** intermediating between platform and
  Balaur: the data path leaves the owner's box infrastructure.
- **Auto-enabled by default** without an explicit owner confirmation: violates
  the consent pillar (same bar as the cloud model opt-in gate).

---

## 4. Recommendation: Bet A

Green-light a constraint-respecting messenger gateway slice.

**Reasoning:**

1. **The gateway line holds.** Both existing gateways delegate all behavior to
   `turn.Run` without re-implementing any of it. A third gateway of a
   different shape (async, push-driven) confirms the doctrine is structural,
   not aspirational. Every messenger-specific need is a legitimately
   above-the-line adapter concern.

2. **The async wrapper is thin.** `emit + Result` is sufficient. A goroutine
   wrapper with a background context and an outbound delivery call is all that
   is needed. `turn.Run`'s contract is unchanged.

3. **A loopback-first design is achievable.** Polling bridge on-box + loopback
   endpoint + pre-shared token + NetBird mesh for remote access keeps the
   threat model compliant. The platform never reaches Balaur directly.

4. **One prerequisite before implementation:** the concurrent-turn gap (§1.5)
   must be addressed in the follow-up plan. A gateway-level in-flight mutex (or
   a per-conversation turn queue) is necessary before a messenger surface can
   safely accept messages that may arrive while a turn is already running. The
   web gateway avoids this today by relying on the browser's `streaming` signal
   to disable the composer — a messenger cannot rely on that.

**Follow-up plan should include:**
- A per-conversation in-flight guard (mutex or channel) added to the gateway
  layer (not to `turn.Run` itself).
- A `/api/messenger/turn` endpoint registered behind `guardLocalUI` + a
  pre-shared token check.
- A thin async adapter (goroutine + background context).
- Explicit owner opt-in and activation gate (paralleling the cloud model
  confirmation flow).
- A reference bridge implementation (polling mode, single transport) as a
  separate, owner-run process — not embedded in Balaur.
- Update `internal/self/knowledge.md` only when the bridge ships; "future
  messengers" stays roadmap until then.

---

## Verified file:line claims

| Claim | Verified at |
|---|---|
| `turn.Run` signature | `internal/turn/turn.go:69` |
| Web gateway calls `turn.Run` | `internal/web/chat.go:52` |
| CLI gateway calls `turn.Run` | `internal/cli/chat.go:90` |
| `kronk/runtime.go` is engine plumbing only | `internal/kronk/runtime.go:1–26` |
| `guardLocalUI` enforces loopback host | `internal/web/web.go:64–122` |
| `guardLocalUI` bound before routes | `internal/web/web.go:145` |
| `turn.Run` substitutes a no-op nil emit | `internal/turn/turn.go:70–72` |
| PRODUCT.md loopback-first non-goal | `PRODUCT.md:108–109` |
| PRODUCT.md "more gateways, same pipeline" bet | `PRODUCT.md:152–154` |
