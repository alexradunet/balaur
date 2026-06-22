package web

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/pocketbase/pocketbase/core"

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
