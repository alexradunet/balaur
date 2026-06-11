package web

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/turn"
)

const messageCloseHTML = `</div></div></div>`

// soulAvatarHTML builds the soul avatar span. Dynamic because the URL varies
// by owner preference (soul-01..soul-16).
func soulAvatarHTML(avatarURL string) string {
	return `<span class="balaur-avatar balaur-avatar-soul" data-kind="soul" aria-hidden="true">` +
		`<img src="` + html.EscapeString(avatarURL) + `" alt="" decoding="async"></span>`
}

// balaurAvatarHTML builds the Balaur avatar span. Dynamic because the owner
// can choose from 16 head personalities (balaur-01..balaur-16).
func balaurAvatarHTML(avatarURL string) string {
	return `<span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" aria-hidden="true">` +
		`<img src="` + html.EscapeString(avatarURL) + `" alt="" decoding="async"></span>`
}

// chat handles one user turn. The web layer is a gateway: it adapts the
// shared turn pipeline (internal/turn) to a streamed chunked response that
// HTMX appends to the chat (hx-swap beforeend). The fragment shape mirrors
// templates/home.html; the behavior lives in turn.Run.
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

	// Resolve per-turn dynamic values — owner preferences from owner_settings.
	soulHTML      := soulAvatarHTML(store.SoulAvatarURL(h.app))
	balaHTML      := balaurAvatarHTML(store.BalaurAvatarURL(h.app))
	ownerName     := store.OwnerName(h.app)
	assistantOpen := `<div class="msg msg-balaur msg-with-avatar">` + balaHTML +
		`<div class="msg-main"><div class="who">Balaur</div><div class="body">`

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
	// replaces only the pending Balaur row. Without JS, keep the old echo path.
	if !clientRendered {
		fmt.Fprintf(w,
			`<div class="msg msg-user msg-with-avatar">%s`+
				`<div class="msg-main"><div class="who">%s</div><div class="body">%s</div></div></div>`,
			soulHTML, html.EscapeString(ownerName), html.EscapeString(msg))
	}
	fmt.Fprint(w, assistantOpen)
	flush()

	emitEv := func(ev agent.Event) {
		switch ev.Kind {
		case "text":
			fmt.Fprint(w, html.EscapeString(ev.Text))
			flush()
		case "tool_start":
			// Bare teal glyph matches the {{toolIcon}} template helper.
			fmt.Fprintf(w,
				messageCloseHTML+
					`<div class="msg msg-tool"><div class="who">`+
					`<span class="tool-icon" aria-hidden="true">%s</span>tool · %s`+
					`</div><div class="body">`,
				toolGlyph(ev.Tool), html.EscapeString(ev.Tool))
			flush()
		case "tool_result":
			h.writeToolResult(w, ev.Text)
			fmt.Fprint(w, `</div></div>`+assistantOpen)
			flush()
		case "error":
			fmt.Fprintf(w,
				`<span class="thinking">the thread snapped: %s</span>`,
				html.EscapeString(ev.Err.Error()))
			flush()
		}
	}

	res, runErr := turn.Run(e.Request.Context(), h.app, client, msg, emitEv)

	fmt.Fprint(w, messageCloseHTML)
	if res.CheckNote != "" {
		fmt.Fprintf(w,
			`<div class="msg msg-balaur msg-with-avatar">%s`+
				`<div class="msg-main"><div class="who">Balaur · check</div><div class="body">%s</div></div></div>`,
			balaHTML, html.EscapeString(res.CheckNote))
	}
	flush()
	_ = runErr
	return nil
}

// writeToolResult renders a tool result row.
func (h *handlers) writeToolResult(w http.ResponseWriter, text string) {
	kind, id, rest, ok := tools.ParseProposal(text)
	if !ok {
		fmt.Fprint(w, html.EscapeString(clipText(text, 2000)))
		return
	}
	fmt.Fprint(w, html.EscapeString(rest))
	fmt.Fprintf(w,
		`</div></div><div class="k-inline" hx-get="%s" hx-trigger="load" hx-swap="innerHTML"></div>`+
			`<div class="msg msg-tool" hidden><div class="body">`,
		html.EscapeString(cardURL(kind, id)))
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
	fmt.Fprintf(e.Response,
		`<div class="msg msg-balaur msg-with-avatar">%s`+
			`<div class="msg-main"><div class="who">Balaur</div><div class="body">`+
			`<span class="thinking">%s</span></div></div></div>`,
		balaurAvatarHTML(balaURL), html.EscapeString(err.Error()))
	return nil
}
