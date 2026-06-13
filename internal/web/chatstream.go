package web

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/tools"
)

// chatSignals is the Datastar signal payload the composer sends with @post.
type chatSignals struct {
	Message string `json:"message"`
}

// readChatMessage pulls the owner's turn from a Datastar @post (JSON signals)
// or a plain form post (choices fallback / no-JS / tests). It must not read the
// body twice, so it only consults Datastar when the request is JSON.
func readChatMessage(e *core.RequestEvent) string {
	if strings.Contains(e.Request.Header.Get("Content-Type"), "application/json") {
		var sig chatSignals
		if err := datastar.ReadSignals(e.Request, &sig); err == nil {
			return strings.TrimSpace(sig.Message)
		}
	}
	return strings.TrimSpace(e.Request.FormValue("message"))
}

// chatStream renders one streamed turn as Datastar SSE patches. The agent loop
// (internal/turn) stays the single source of truth — this only adapts its
// agent.Event stream into datastar-patch-elements events: it appends complete
// elements to #chat and morphs their bodies by id as text accumulates. The
// fragment markup lives once in chat-messages.html.
type chatStream struct {
	sse       *datastar.ServerSentEventGenerator
	h         *handlers
	base      string // per-turn id prefix (unique element ids)
	balaURL   string
	who       string
	soulURL   string
	ownerName string

	bubbleN  int
	toolN    int
	bubbleID string // current assistant bubble root id ("" = none open)
	bodyID   string // current assistant bubble body id (morph target)
	toolName string // tool of the open tool row
	toolID   string // open tool row root id
	toolBody string // open tool row body id
	buf      strings.Builder
}

// newChatStream upgrades the response to a Datastar SSE connection and prepares
// a streamed turn for the given avatar/name.
func (h *handlers) newChatStream(e *core.RequestEvent, balaURL, who, soulURL, ownerName string) *chatStream {
	return &chatStream{
		sse:       datastar.NewSSE(e.Response, e.Request),
		h:         h,
		base:      newNonce(),
		balaURL:   balaURL,
		who:       who,
		soulURL:   soulURL,
		ownerName: ownerName,
	}
}

// render executes a named fragment to a buffer; empty string on error (the
// caller owns a live stream and cannot un-send bytes).
func (s *chatStream) render(name string, mv messageView) string {
	var b strings.Builder
	if err := s.h.tmpl.ExecuteTemplate(&b, name, mv); err != nil {
		s.h.app.Logger().Warn("chat fragment render failed", "fragment", name, "err", err)
		return ""
	}
	return b.String()
}

// appendChat appends a fragment as the last child of #chat.
func (s *chatStream) appendChat(name string, mv messageView) {
	_ = s.sse.PatchElements(s.render(name, mv),
		datastar.WithSelectorID("chat"), datastar.WithModeAppend())
}

// morph replaces an element in place by the id on the fragment's root.
func (s *chatStream) morph(name string, mv messageView) {
	_ = s.sse.PatchElements(s.render(name, mv))
}

// start clears any stale choice panels and the composer signal, echoes the
// owner's message, and opens the first assistant bubble.
func (s *chatStream) start(userMsg string) {
	_ = s.sse.RemoveElement(".choices")
	_ = s.sse.MarshalAndPatchSignals(chatSignals{Message: ""})
	s.appendChat("chat-msg-user", messageView{
		SoulAvatarURL: s.soulURL, OwnerName: s.ownerName, Content: userMsg,
	})
	s.openBubble()
}

// openBubble appends a fresh pending assistant bubble and points the stream at
// its body. A new bubble is opened after every tool, mirroring the prior
// open/close-around-tools shape.
func (s *chatStream) openBubble() {
	s.bubbleN++
	s.bubbleID = fmt.Sprintf("balaur-%s-%d", s.base, s.bubbleN)
	s.bodyID = s.bubbleID + "-body"
	s.buf.Reset()
	s.appendChat("chat-balaur-bubble", messageView{
		BalaurAvatarURL: s.balaURL, WhoLabel: s.who,
		BubbleID: s.bubbleID, BodyID: s.bodyID, Pending: true,
	})
}

// finalizeBubble drops an empty bubble or morphs the open one to its final,
// non-pending state.
func (s *chatStream) finalizeBubble() {
	if s.bubbleID == "" {
		return
	}
	if strings.TrimSpace(s.buf.String()) == "" {
		_ = s.sse.RemoveElementByID(s.bubbleID)
	} else {
		s.morph("chat-balaur-bubble", messageView{
			BalaurAvatarURL: s.balaURL, WhoLabel: s.who,
			BubbleID: s.bubbleID, BodyID: s.bodyID, Content: s.buf.String(),
		})
	}
	s.bubbleID, s.bodyID = "", ""
}

// emit adapts one agent.Event into Datastar patches.
func (s *chatStream) emit(ev agent.Event) {
	switch ev.Kind {
	case "text":
		s.buf.WriteString(ev.Text)
		s.morph("chat-balaur-body", messageView{BodyID: s.bodyID, Content: s.buf.String()})
	case "tool_start":
		s.finalizeBubble()
		s.toolN++
		s.toolName = ev.Tool
		s.toolID = fmt.Sprintf("tool-%s-%d", s.base, s.toolN)
		s.toolBody = s.toolID + "-body"
		s.appendChat("chat-tool-row", messageView{
			Tool: s.toolName, BubbleID: s.toolID, BodyID: s.toolBody,
		})
	case "tool_result":
		s.handleToolResult(ev)
		s.openBubble()
	case "error":
		s.buf.WriteString(" — the thread snapped: " + ev.Err.Error())
		s.morph("chat-balaur-body", messageView{BodyID: s.bodyID, Content: s.buf.String()})
	}
}

// handleToolResult mirrors the prior consumer order: uicard → choices →
// proposal → plain. The open tool row's body is morphed with the result; an
// inline card (when present) is appended and handed to htmx during migration.
func (s *chatStream) handleToolResult(ev agent.Event) {
	if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
		s.endTool(rest, "/ui/cards/"+typ+"?"+query)
		return
	}
	if prompt, choices, _, ok := tools.ParseChoices(ev.Text); ok {
		s.endTool("choices offered", "")
		s.appendChoices(prompt, choices)
		return
	}
	if kind, id, rest, ok := tools.ParseProposal(ev.Text); ok {
		s.endTool(rest, cardURL(kind, id))
		return
	}
	s.endTool(clipText(ev.Text, 2000), "")
}

// endTool morphs the open tool row with its result and, if a card is attached,
// appends the lazy mount and asks htmx to process it (htmx still loads during
// the migration; the card endpoints stay HTML).
func (s *chatStream) endTool(content, card string) {
	s.morph("chat-tool-row", messageView{
		Tool: s.toolName, BubbleID: s.toolID, BodyID: s.toolBody, Content: content,
	})
	if card != "" {
		cardID := s.toolID + "-card"
		s.appendChat("chat-inline-card", messageView{BubbleID: cardID, CardURL: card})
		_ = s.sse.ExecuteScript(fmt.Sprintf(
			"window.htmx&&htmx.process(document.getElementById('%s'))", cardID))
	}
}

// appendChoices appends a live dialogue-choice panel.
func (s *chatStream) appendChoices(prompt string, choices []tools.Choice) {
	cv := choicesView{
		Prompt: prompt, Nonce: newNonce(), Choices: choices,
		SoulAvatarURL: s.soulURL, OwnerName: s.ownerName,
	}
	var b strings.Builder
	if err := s.h.tmpl.ExecuteTemplate(&b, "chat-choices", cv); err != nil {
		s.h.app.Logger().Warn("chat fragment render failed", "fragment", "chat-choices", "err", err)
		return
	}
	_ = s.sse.PatchElements(b.String(), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
}

// note appends a standalone assistant message (e.g. the honesty check note).
func (s *chatStream) note(origin, content string) {
	s.appendChat("chat-msg-balaur", messageView{
		BalaurAvatarURL: s.balaURL, WhoLabel: s.who, Origin: origin, Content: content,
	})
}

// finish closes the last open bubble.
func (s *chatStream) finish() { s.finalizeBubble() }
