package web

import (
	"crypto/rand"
	"encoding/hex"
	"html"
	"io"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/turn"
)

// ping is the Datastar bootstrap proof (plan 050, Phase 0): a button with
// data-on-click="@get('/ui/ping')" patches #ping to "pong" over SSE. Removed
// once the chat stream proves the same path end-to-end in Phase 1.
func (h *handlers) ping(e *core.RequestEvent) error {
	sse := datastar.NewSSE(e.Response, e.Request)
	return sse.PatchElements(`<span id="ping">pong</span>`, datastar.WithSelectorID("ping"))
}

// choicesView is the template payload for the chat-choices fragment.
type choicesView struct {
	Prompt        string
	Nonce         string // unique per render, used for element IDs
	Choices       []tools.Choice
	SoulAvatarURL string
	OwnerName     string
}

// chatSignals is the Datastar signal payload the composer and the dialogue-
// choice buttons post: {"message": "..."} (read via datastar.ReadSignals,
// replacing the old form FormValue).
type chatSignals struct {
	Message string `json:"message"`
}

// newNonce generates a random 8-byte hex string for unique element IDs.
func newNonce() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// execFragment executes a named template fragment to w, silently ignoring
// errors — the caller already owns the live stream and cannot un-write bytes.
// Still used by the head-chat gateway (headsmgmt.go), which remains on HTMX.
func (h *handlers) execFragment(w io.Writer, name string, data messageView) {
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.app.Logger().Warn("chat fragment render failed", "fragment", name, "err", err)
	}
}

// fragment renders a named template fragment to a string for Datastar
// PatchElements. Render errors are logged, not fatal — a malformed fragment
// must not tear down the live SSE stream.
func (h *handlers) fragment(name string, data any) string {
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, name, data); err != nil {
		h.app.Logger().Warn("chat fragment render failed", "fragment", name, "err", err)
	}
	return b.String()
}

// chat handles one owner turn. The web layer is a gateway: it adapts the shared
// turn pipeline (internal/turn) to Datastar SSE patches. PatchElements morphs by
// id, so each assistant reply is a self-contained bubble appended into #chat;
// streamed tokens are morphed into its body (inner mode). turn.Run and the
// fragment markup are unchanged — only the delivery idiom moves from HTMX
// chunked-append swaps to SSE patches. "Gateways adapt; they never re-implement."
func (h *handlers) chat(e *core.RequestEvent) error {
	var sig chatSignals
	if err := datastar.ReadSignals(e.Request, &sig); err != nil {
		return e.BadRequestError("invalid signals", err)
	}
	msg := strings.TrimSpace(sig.Message)
	if msg == "" {
		return e.BadRequestError("empty message", nil)
	}

	soulURL := store.SoulAvatarURL(h.app)
	balaURL := store.BalaurAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)

	sse := datastar.NewSSE(e.Response, e.Request)
	appendChat := func(htmlStr string) {
		sse.PatchElements(htmlStr, datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	}
	scroll := func() { sse.ExecuteScript("window.balaurScrollToLatest&&balaurScrollToLatest()") }

	// Echo the owner's turn — there is no client-side optimistic clone anymore.
	appendChat(h.fragment("chat-msg-user", messageView{
		SoulAvatarURL: soulURL, OwnerName: ownerName, Content: msg,
	}))

	client, err := h.clients.Active(h.app)
	if err != nil {
		appendChat(h.fragment("chat-msg-balaur", messageView{
			BalaurAvatarURL: balaURL, WhoLabel: "Balaur", Content: err.Error(),
		}))
		scroll()
		return nil
	}

	// Streaming-bubble state. Each bubble is self-contained with a stable id;
	// after a tool row we open a fresh bubble (mirrors the old open-after-tool
	// behavior). buf accumulates the current bubble's text for inner morphs.
	var (
		buf         strings.Builder
		bubbleID    string
		bodyID      string
		hasText     bool
		pendingTool string
	)
	startBubble := func() {
		buf.Reset()
		hasText = false
		nonce := newNonce()
		bubbleID = "bubble-" + nonce
		bodyID = bubbleID + "-body"
		appendChat(h.fragment("chat-balaur-bubble", messageView{
			Nonce: nonce, BalaurAvatarURL: balaURL, WhoLabel: "Balaur",
		}))
	}
	// dropEmptyBubble removes a bubble that never received text — it would be
	// stuck on the "thinking" placeholder. Called before a tool row and at end.
	dropEmptyBubble := func() {
		if bubbleID != "" && !hasText {
			sse.RemoveElementByID(bubbleID)
			bubbleID = ""
		}
	}
	streamBody := func(extra string) {
		sse.PatchElements(html.EscapeString(buf.String())+extra,
			datastar.WithSelectorID(bodyID), datastar.WithModeInner())
	}

	startBubble()
	scroll()

	emitEv := func(ev agent.Event) {
		switch ev.Kind {
		case "text":
			hasText = true
			buf.WriteString(ev.Text)
			streamBody("")
		case "tool_start":
			pendingTool = ev.Tool
			dropEmptyBubble()
		case "tool_result":
			// Consumer order matches history rendering: uicard → choices →
			// proposal → plain.
			if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
				appendChat(h.fragment("chat-msg-tool", messageView{
					Tool: pendingTool, Content: rest, CardURL: "/ui/cards/" + typ + "?" + query,
				}))
				startBubble()
				scroll()
				break
			}
			// ParseChoices before ParseProposal — choices ride a tool_result.
			if prompt, choices, _, ok := tools.ParseChoices(ev.Text); ok {
				appendChat(h.fragment("chat-msg-tool", messageView{
					Tool: pendingTool, Content: "choices offered",
				}))
				appendChat(h.fragment("chat-choices", choicesView{
					Prompt: prompt, Nonce: newNonce(), Choices: choices,
					SoulAvatarURL: soulURL, OwnerName: ownerName,
				}))
				startBubble()
				scroll()
				break
			}
			kind, id, rest, ok := tools.ParseProposal(ev.Text)
			mv := messageView{Tool: pendingTool}
			if ok {
				mv.Content = rest
				mv.CardURL = cardURL(kind, id)
			} else {
				mv.Content = clipText(ev.Text, 2000)
			}
			appendChat(h.fragment("chat-msg-tool", mv))
			startBubble()
			scroll()
		case "error":
			hasText = true
			streamBody(`<span class="thinking">the thread snapped: ` +
				html.EscapeString(ev.Err.Error()) + `</span>`)
		}
	}

	res, runErr := turn.Run(e.Request.Context(), h.app, client, msg, emitEv)

	dropEmptyBubble()
	if res.CheckNote != "" {
		appendChat(h.fragment("chat-msg-balaur", messageView{
			BalaurAvatarURL: balaURL, WhoLabel: "Balaur", Origin: "check", Content: res.CheckNote,
		}))
	}
	// Calm the avatars once the turn is done: drop the thinking/working glow.
	sse.ExecuteScript(`document.querySelectorAll('#chat .balaur-avatar[data-state]').forEach(function(a){a.removeAttribute('data-state')})`)
	scroll()
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

// renderError writes a Balaur error bubble as a plain HTML fragment. Still used
// by the head-chat gateway (headsmgmt.go), which remains on HTMX this phase.
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
