package web

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/turn"
)

// choicesView is the template payload for the chat-choices fragment.
type choicesView struct {
	Prompt        string
	Nonce         string // unique per render, used for element IDs
	Choices       []tools.Choice
	SoulAvatarURL string
	OwnerName     string
}

// newNonce generates a random 8-byte hex string for unique element IDs.
func newNonce() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// execFragment executes a named template fragment to w, silently ignoring
// errors — the caller already owns the live stream and cannot un-write bytes.
// Used by the head-chat gateway, which still streams chunked HTML.
func (h *handlers) execFragment(w io.Writer, name string, data messageView) {
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.app.Logger().Warn("chat fragment render failed", "fragment", name, "err", err)
	}
}

// chat handles one user turn in the master conversation. The web layer is a
// gateway: it adapts the shared turn pipeline (internal/turn) into a Datastar
// SSE stream of element patches. Fragment markup lives in chat-messages.html;
// the streaming lifecycle lives in chatStream (chatstream.go).
func (h *handlers) chat(e *core.RequestEvent) error {
	msg := readChatMessage(e)
	if msg == "" {
		return e.BadRequestError("empty message", nil)
	}

	soulURL := store.SoulAvatarURL(h.app)
	balaURL := store.BalaurAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)

	cs := h.newChatStream(e, balaURL, "Balaur", soulURL, ownerName)

	client, err := h.clients.Active(h.app)
	if err != nil {
		cs.appendChat("chat-msg-user", messageView{
			SoulAvatarURL: soulURL, OwnerName: ownerName, Content: msg,
		})
		_ = cs.sse.MarshalAndPatchSignals(chatSignals{Message: ""})
		cs.note("", err.Error())
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

func cardURL(kind, id string) string {
	if kind == "tasks" {
		return "/ui/tasks/" + id + "/card"
	}
	return "/ui/knowledge/" + kind + "/" + id + "/card"
}

func clipText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// renderError writes a plain assistant error bubble. Used by the head-chat
// gateway (still chunked HTML).
func (h *handlers) renderError(e *core.RequestEvent, err error) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	balaURL := store.BalaurAvatarURL(h.app)
	h.execFragment(e.Response, "chat-msg-balaur", messageView{
		BalaurAvatarURL: balaURL,
		WhoLabel:        "Balaur",
		Content:         err.Error(),
	})
	return nil
}
