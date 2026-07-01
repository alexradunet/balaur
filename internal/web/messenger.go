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
//     sets the token via the PocketBase admin engine room; a settings-UI
//     toggle is a natural follow-up.
//  3. No third-party routing — only accepts a local POST and returns a reply.
//     No platform API is called anywhere in this file.
//  4. No secrets in output/logs — the token is never logged; errors are
//     sanitized.
//
// In-flight guard: turn.TryBegin (internal/turn) provides a cross-surface
// guard — web, CLI, and messenger all acquire it before running a turn so the
// master conversation is never written concurrently from any surface.

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)

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

	// 4. Cross-surface in-flight guard — one turn at a time on the master
	//    conversation (shared with the web and CLI gateways via turn.TryBegin).
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
