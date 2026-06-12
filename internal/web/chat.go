package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
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
func (h *handlers) execFragment(w io.Writer, name string, data messageView) {
	_ = h.tmpl.ExecuteTemplate(w, name, data)
}

// execChoicesFragment executes the chat-choices template fragment.
func (h *handlers) execChoicesFragment(w io.Writer, cv choicesView) {
	_ = h.tmpl.ExecuteTemplate(w, "chat-choices", cv)
}

// chat handles one user turn. The web layer is a gateway: it adapts the
// shared turn pipeline (internal/turn) to a streamed chunked response that
// HTMX appends to the chat (hx-swap beforeend). The fragment shape is
// defined once in chat-messages.html and executed here and on page-load.
func (h *handlers) chat(e *core.RequestEvent) error {
	msg := strings.TrimSpace(e.Request.FormValue("message"))
	if msg == "" {
		return e.BadRequestError("empty message", nil)
	}

	client, err := h.clients.Active(h.app)
	if err != nil {
		return h.renderError(e, err)
	}
	clientRendered := e.Request.FormValue("client_rendered") == "1"

	soulURL := store.SoulAvatarURL(h.app)
	balaURL := store.BalaurAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)

	w := e.Response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}

	// When the browser optimistically rendered the user row, this response
	// replaces only the pending Balaur row. Without JS, echo the user row first.
	if !clientRendered {
		h.execFragment(w, "chat-msg-user", messageView{
			SoulAvatarURL: soulURL,
			OwnerName:     ownerName,
			Content:       msg,
		})
	}

	// Open the assistant bubble; token text streams into its open body div.
	h.execFragment(w, "chat-balaur-open", messageView{
		BalaurAvatarURL: balaURL,
		WhoLabel:        "Balaur",
	})
	flush()

	emitEv := func(ev agent.Event) {
		switch ev.Kind {
		case "text":
			fmt.Fprint(w, html.EscapeString(ev.Text))
			flush()
		case "tool_start":
			h.execFragment(w, "chat-balaur-close", messageView{})
			h.execFragment(w, "chat-msg-tool-start", messageView{Tool: ev.Tool})
			flush()
		case "tool_result":
			// Consumer order: uicard → choices → proposal → plain.
			// uicard: re-renders on reload (lazy-fetches live data — safe).
			// choices: inert on reload (no live panel for stale decisions).
			// proposal: renders an approval card on first view and reload.
			if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
				mv := messageView{Content: rest, CardURL: "/ui/cards/" + typ + "?" + query}
				h.execFragment(w, "chat-msg-tool-end", mv)
				h.execFragment(w, "chat-balaur-open", messageView{
					BalaurAvatarURL: balaURL,
					WhoLabel:        "Balaur",
				})
				flush()
				break
			}
			// Check ParseChoices before ParseProposal — choices ride a tool_result.
			if prompt, choices, _, ok := tools.ParseChoices(ev.Text); ok {
				h.execFragment(w, "chat-msg-tool-end", messageView{Content: "choices offered"})
				h.execChoicesFragment(w, choicesView{
					Prompt:        prompt,
					Nonce:         newNonce(),
					Choices:       choices,
					SoulAvatarURL: soulURL,
					OwnerName:     ownerName,
				})
				h.execFragment(w, "chat-balaur-open", messageView{
					BalaurAvatarURL: balaURL,
					WhoLabel:        "Balaur",
				})
				flush()
				break
			}
			kind, id, rest, ok := tools.ParseProposal(ev.Text)
			var mv messageView
			if ok {
				mv = messageView{Content: rest, CardURL: cardURL(kind, id)}
			} else {
				mv = messageView{Content: clipText(ev.Text, 2000)}
			}
			h.execFragment(w, "chat-msg-tool-end", mv)
			h.execFragment(w, "chat-balaur-open", messageView{
				BalaurAvatarURL: balaURL,
				WhoLabel:        "Balaur",
			})
			flush()
		case "error":
			fmt.Fprintf(w,
				`<span class="thinking">the thread snapped: %s</span>`,
				html.EscapeString(ev.Err.Error()))
			flush()
		}
	}

	res, runErr := turn.Run(e.Request.Context(), h.app, client, msg, emitEv)

	h.execFragment(w, "chat-balaur-close", messageView{})
	if res.CheckNote != "" {
		h.execFragment(w, "chat-msg-balaur", messageView{
			BalaurAvatarURL: balaURL,
			WhoLabel:        "Balaur",
			Origin:          "check",
			Content:         res.CheckNote,
		})
	}
	flush()
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
