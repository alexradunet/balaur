package web

// messenger.go — POST /api/messenger/turn
//
// A consent-gated, token-authed endpoint a local bridge can POST a message to
// and receive a reply from, using the same turn pipeline the web and CLI
// gateways use. Ships NO chat-app client; the bridge is the owner's own
// process.
//
// Security model (all four constraints from PRODUCT.md / plan 231):
//  1. Host check is a DNS-rebinding defense, NOT network-layer loopback
//     isolation. isAllowedHost(e.Request.Host) trusts a spoofable Host header,
//     exactly as guardLocalUI guards /ui/* (guardLocalUI's middleware bypasses
//     /api/*, so the handler calls the shared helper inline). On a box that
//     binds 0.0.0.0 (e.g. the prod NetBird mesh), a mesh peer can reach this
//     endpoint with Host: localhost — so the Bearer token (the consent gate +
//     auth, constraint 2 below) is the PRIMARY, effective access control, not
//     the host check. True loopback isolation would require binding the
//     listener to 127.0.0.1, which is outside this handler's control.
//  2. Consent-gated / fail-closed — DISABLED unless owner_settings key
//     "messenger_token" is non-empty. No token → 403, no turn run. The owner
//     sets the token in Settings → Capabilities (POST
//     /ui/settings/messenger-token, messenger_settings.go); the PocketBase
//     admin engine room remains a fallback.
//  3. No third-party routing — only accepts a local POST and returns a reply.
//     No platform API is called anywhere in this file.
//  4. No secrets in output/logs — the token is never logged; errors are
//     sanitized.
//  5. Failed auth is logged (remote addr only) and throttled: after 5
//     consecutive bad tokens the endpoint answers 429 for a 30s cooldown.
//
// In-flight guard: turn.TryBegin (internal/turn) is a process-wide mutex —
// within the serve process, web and messenger turns are serialized. It does
// NOT reach across processes: a separate `balaur chat` process on the same
// data dir takes its own copy of the mutex and is not serialized against a
// running server (known limitation — see AGENTS.md).

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)

// Brute-force friction on the token check (v1 scale: one owner, one
// bridge). After messengerMaxFailures consecutive bad tokens the endpoint
// answers 429 until messengerCooldown passes; any successful auth resets
// the counter. This is deliberate friction, NOT real rate limiting — no
// per-IP tracking, no persistence across restarts.
const (
	messengerMaxFailures = 5
	messengerCooldown    = 30 * time.Second
)

// authThrottle holds the failure counter. It lives on the handlers struct
// (one instance per serve, shared across requests), so all access is
// mutex-guarded. The zero value is ready to use.
type authThrottle struct {
	mu       sync.Mutex
	failures int
	lastFail time.Time
	now      func() time.Time // test seam; nil means time.Now
}

func (t *authThrottle) clock() time.Time {
	if t.now != nil {
		return t.now()
	}
	return time.Now()
}

// allow reports whether an auth attempt may proceed. Cooldown expiry
// resets the counter so one stale failure cannot re-lock the endpoint.
func (t *authThrottle) allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.failures < messengerMaxFailures {
		return true
	}
	if t.clock().Sub(t.lastFail) >= messengerCooldown {
		t.failures = 0
		return true
	}
	return false
}

func (t *authThrottle) fail() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failures++
	t.lastFail = t.clock()
}

func (t *authThrottle) success() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failures = 0
}

// messengerTurn handles POST /api/messenger/turn.
func (h *handlers) messengerTurn(e *core.RequestEvent) error {
	// 1. Host check (DNS-rebinding defense, not loopback isolation). guardLocalUI
	//    bypasses /api/*, so we call the shared isAllowedHost helper directly.
	//    The Host header is spoofable, so on a 0.0.0.0-binding box this does not
	//    confine the endpoint to the local machine — the Bearer token below is
	//    the effective access control.
	host := e.Request.Host
	if hh, _, err := net.SplitHostPort(host); err == nil {
		host = hh
	}
	if !isAllowedHost(host) {
		return e.ForbiddenError("host not allowed", nil)
	}

	// 2. Consent gate (fail-closed). Feature is disabled until the owner
	//    explicitly sets a token in owner_settings.messenger_token.
	tok := store.GetOwnerSetting(h.app, "messenger_token", "")
	if tok == "" {
		return e.JSON(http.StatusForbidden, map[string]string{"error": "messenger gateway is not enabled"})
	}

	// 3a. Brute-force friction: after messengerMaxFailures consecutive bad
	//     tokens, reject with 429 until messengerCooldown passes. The body
	//     differs from the turn guard's "busy" so callers can tell them apart.
	if !h.messengerThrottle.allow() {
		e.Response.Header().Set("Retry-After", strconv.Itoa(int(messengerCooldown/time.Second)))
		return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many failed auth attempts"})
	}

	// 3. Token auth — constant-time comparison; the header value is never logged.
	authHeader := e.Request.Header.Get("Authorization")
	const prefix = "Bearer "
	var provided string
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		provided = authHeader[len(prefix):]
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(tok)) != 1 {
		h.messengerThrottle.fail()
		h.app.Logger().Warn("messenger: auth failed", "remote", e.Request.RemoteAddr)
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	h.messengerThrottle.success()

	// 4. In-flight guard — one turn at a time on the master conversation
	//    within this process (shared with the web gateway via turn.TryBegin).
	end, ok := turn.TryBegin()
	if !ok {
		return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "busy"})
	}
	defer end()

	// 5. Parse body.
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || body.Message == "" {
		return e.BadRequestError("message is required", nil)
	}

	// Resolve the active model client — mirror web/chat.go's h.clients.Active.
	client, err := h.clients.Active(h.app)
	if err != nil {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "no active model"})
	}

	// Run the turn with a generous timeout; the bridge handles async delivery.
	ctx, cancel := context.WithTimeout(e.Request.Context(), 10*time.Minute)
	defer cancel()

	res, runErr := turn.Run(ctx, h.app, client, body.Message, func(agent.Event) {})
	if runErr != nil {
		h.app.Logger().Warn("messenger: turn failed", "error", runErr)
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "turn failed"})
	}
	return e.JSON(http.StatusOK, map[string]string{"reply": res.Reply})
}
