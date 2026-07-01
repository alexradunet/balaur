package web

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

// newNonce generates a random 8-byte hex string for unique element IDs.
func newNonce() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// chat handles one user turn in the master conversation. The web layer is a
// gateway: it adapts the shared turn pipeline (internal/turn) into a Datastar
// SSE stream of element patches. The streaming lifecycle lives in chatStream
// (chatstream.go).
func (h *handlers) chat(e *core.RequestEvent) error {
	msg := readChatMessage(e)
	if msg == "" {
		return e.BadRequestError("empty message", nil)
	}

	// Cross-surface in-flight guard: exactly one turn runs on the master
	// conversation at a time (web + CLI + messenger). Acquire before any
	// medium setup so a busy response never paints a user bubble or opens
	// the stream. TryLock is intentional — at v1 a second concurrent turn
	// is always a race, never a work queue.
	end, ok := turn.TryBegin()
	if !ok {
		// Open a minimal SSE connection just to deliver the toast; no #chat
		// mutation, no user bubble.
		sse := datastar.NewSSE(e.Response, e.Request)
		emitToast(sse, "warn", "One message is still being answered — try again in a moment.")
		return nil
	}
	defer end()

	soulURL := store.SoulAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)
	head := heads.Active(h.app)
	balaURL := store.BalaurAvatarURLForKey(h.app, head.Avatar)

	cs := h.newChatStream(e, balaURL, head.Name, soulURL, ownerName)

	client, err := h.clients.Active(h.app)
	if err != nil {
		cs.appendNode(chat.Message(chat.MessageProps{
			Role: "user", AvatarSrc: soulURL, Who: ownerName, Content: msg,
		}))
		_ = cs.sse.MarshalAndPatchSignals(chatSignals{Message: ""})
		cs.note("", h.chatErrText(err))
		return nil
	}

	cs.start(msg)
	res, runErr := turn.Run(e.Request.Context(), h.app, client, msg, cs.emit)
	cs.finish()
	if res.CheckNote != "" {
		cs.note("check", res.CheckNote)
	}
	if runErr != nil {
		h.app.Logger().Warn("chat: turn failed", "error", runErr)
	}
	return nil
}

func clipText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Truncate on a rune boundary so multi-byte tool output never renders a
	// broken replacement char in the morphed row.
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// chatErrText returns an owner-facing chat error, replacing anything that looks
// like a provider endpoint with a generic line (AGENTS.md: do not leak private
// URLs) and logging the raw detail. Shared by the chat gateway and the stream.
func (h *handlers) chatErrText(err error) string {
	h.app.Logger().Warn("chat: surfaced error", "error", err)
	if strings.Contains(err.Error(), "://") {
		return "the model is unreachable — check the active provider in Settings"
	}
	return err.Error()
}

// reURL and rePath redact provider endpoints and absolute filesystem paths from
// text that may leave the box. A failed tool is formatted "error: <detail>" by
// the agent loop (internal/agent) and rendered verbatim into the transcript;
// without this a failing OS or HTTP tool could surface a private path, token, or
// provider URL to the owner (AGENTS.md: sanitize before displaying/persisting).
var (
	reURL  = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.-]*://\S+`)
	rePath = regexp.MustCompile(`(?:/[A-Za-z0-9._-]+){2,}/?`)
)

func redactSensitive(s string) string {
	s = reURL.ReplaceAllString(s, "[link]")
	return rePath.ReplaceAllString(s, "[path]")
}

// chatToolErrText sanitizes a failed tool's "error: <detail>" message for the
// transcript: it logs the raw detail and redacts URLs/paths. The model's own
// copy of the tool result (internal/agent) is unaffected — this only shapes the
// owner-facing tool row, so useful error structure survives while secrets don't.
func (h *handlers) chatToolErrText(detail string) string {
	h.app.Logger().Warn("chat: tool error", "detail", detail)
	return redactSensitive(detail)
}
