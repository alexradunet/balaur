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

const (
	// avatarBalaurHTML is the Balaur avatar span — constant because the URL
	// never varies. data-kind lets basm.js drive the animation state.
	avatarBalaurHTML = `<span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" aria-hidden="true">` +
		`<img src="/static/avatars/balaur.png" alt="" decoding="async"></span>`

	assistantOpenHTML = `<div class="msg msg-balaur msg-with-avatar">` + avatarBalaurHTML +
		`<div class="msg-main"><div class="who">Balaur</div><div class="body">`

	messageCloseHTML = `</div></div></div>`
)

// soulAvatarHTML builds the soul avatar span for the current owner preference.
// Unlike the Balaur avatar, the URL is dynamic (soul-male / soul-female) so it
// cannot be a package-level constant.
func soulAvatarHTML(avatarURL string) string {
	return `<span class="balaur-avatar balaur-avatar-soul" data-kind="soul" aria-hidden="true">` +
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

	// Resolve the soul avatar URL once per turn so the streaming fragment
	// and the template path stay in sync with the owner's picker preference.
	soulHTML := soulAvatarHTML(store.SoulAvatarURL(h.app))

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
				`<div class="msg-main"><div class="who">You</div><div class="body">%s</div></div></div>`,
			soulHTML, html.EscapeString(msg))
	}
	fmt.Fprint(w, assistantOpenHTML)
	flush()

	emitEv := func(ev agent.Event) {
		switch ev.Kind {
		case "text":
			fmt.Fprint(w, html.EscapeString(ev.Text))
			flush()
		case "tool_start":
			// Bare teal glyph matches the .tool-icon CSS class and the
			// {{toolIcon}} template helper used by chat-messages.html.
			fmt.Fprintf(w,
				messageCloseHTML+
					`<div class="msg msg-tool"><div class="who">`+
					`<span class="tool-icon" aria-hidden="true">%s</span>tool · %s`+
					`</div><div class="body">`,
				toolGlyph(ev.Tool), html.EscapeString(ev.Tool))
			flush()
		case "tool_result":
			h.writeToolResult(w, ev.Text)
			fmt.Fprint(w, `</div></div>`+assistantOpenHTML)
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
			avatarBalaurHTML, html.EscapeString(res.CheckNote))
	}
	flush()
	_ = runErr // already surfaced in-stream; the fragment stays well-formed
	return nil
}

// writeToolResult renders a tool result row. Marked results render as live
// cards instead of raw text (the Hyperagent card pattern).
func (h *handlers) writeToolResult(w http.ResponseWriter, text string) {
	kind, id, rest, ok := tools.ParseProposal(text)
	if !ok {
		fmt.Fprint(w, html.EscapeString(clipText(text, 2000)))
		return
	}
	fmt.Fprint(w, html.EscapeString(rest))
	// Close the tool row and inject the live card fetched by HTMX, so the
	// card in chat is the same template the dedicated pages use.
	fmt.Fprintf(w,
		`</div></div><div class="k-inline" hx-get="%s" hx-trigger="load" hx-swap="innerHTML"></div>`+
			`<div class="msg msg-tool" hidden><div class="body">`,
		html.EscapeString(cardURL(kind, id)))
}

// cardURL maps a marker kind to its card endpoint.
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
	fmt.Fprintf(e.Response,
		`<div class="msg msg-balaur msg-with-avatar">%s`+
			`<div class="msg-main"><div class="who">Balaur</div><div class="body">`+
			`<span class="thinking">%s</span></div></div></div>`,
		avatarBalaurHTML, html.EscapeString(err.Error()))
	return nil
}
