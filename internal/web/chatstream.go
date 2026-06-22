package web

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/ui/chat"
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

// renderNode renders a gomponents node to a string; empty on error (the caller
// owns a live stream and cannot un-send bytes).
func (s *chatStream) renderNode(n g.Node) string {
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		s.h.app.Logger().Warn("chat node render failed", "err", err)
		return ""
	}
	return b.String()
}

// appendNode appends a rendered component as the last child of #chat.
func (s *chatStream) appendNode(n g.Node) {
	_ = s.sse.PatchElements(s.renderNode(n),
		datastar.WithSelectorID("chat"), datastar.WithModeAppend())
}

// morphNode replaces an element in place by the id on the node's root.
func (s *chatStream) morphNode(n g.Node) {
	_ = s.sse.PatchElements(s.renderNode(n))
}

// balaurBubble builds a Balaur chat.Message bubble for the stream — pending
// while empty, finalized once buf has text. ID/BodyID make it a morph target.
func (s *chatStream) balaurBubble(content string, pending bool) g.Node {
	return chat.Message(chat.MessageProps{
		Role: "balaur", AvatarSrc: s.balaURL, Who: s.who,
		ID: s.bubbleID, BodyID: s.bodyID, Content: content, Pending: pending,
	})
}

// toolCard builds the current tool turn's speech panel (chat.ToolRow): pending
// while the tool runs, finalized with its result + optional artifact chip once
// it returns. Same id/avatar/name as the head's spoken turns so a tool call
// reads as a consistent Balaur turn ("{who} · Tool").
func (s *chatStream) toolCard(content string, chip g.Node, pending bool) g.Node {
	return chat.ToolRow(chat.ToolRowProps{
		Tool: s.toolName, Icon: toolIconFile(s.toolName), Who: s.who, AvatarSrc: s.balaURL,
		ID: s.toolID, BodyID: s.toolBody, Content: content, Chip: chip, Pending: pending,
	})
}

// start clears any stale choice panels and the composer signal, echoes the
// owner's message, and opens the first assistant bubble.
func (s *chatStream) start(userMsg string) {
	_ = s.sse.RemoveElement(".choices")
	_ = s.sse.MarshalAndPatchSignals(chatSignals{Message: ""})
	s.appendNode(chat.Message(chat.MessageProps{
		Role: "user", AvatarSrc: s.soulURL, Who: s.ownerName, Content: userMsg,
	}))
	s.openBubble()
	// Mark the turn in-flight so the branch "← back to main" control disables
	// while tokens stream (otherwise a swap leaves the open head stream
	// appending head-styled tokens into the now-master #chat).
	_ = s.sse.MarshalAndPatchSignals(struct {
		Streaming bool `json:"streaming"`
	}{true})
}

// openBubble appends a fresh pending assistant bubble and points the stream at
// its body. A new bubble is opened after every tool, mirroring the prior
// open/close-around-tools shape.
func (s *chatStream) openBubble() {
	s.bubbleN++
	s.bubbleID = fmt.Sprintf("balaur-%s-%d", s.base, s.bubbleN)
	s.bodyID = s.bubbleID + "-body"
	s.buf.Reset()
	s.appendNode(s.balaurBubble("", true))
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
		s.morphNode(s.balaurBubble(s.buf.String(), false))
	}
	s.bubbleID, s.bodyID = "", ""
}

// emit adapts one agent.Event into Datastar patches.
func (s *chatStream) emit(ev agent.Event) {
	switch ev.Kind {
	case "text":
		s.buf.WriteString(ev.Text)
		s.morphNode(chat.MessageBody(s.bodyID, s.buf.String()))
	case "tool_start":
		s.finalizeBubble()
		s.toolN++
		s.toolName = ev.Tool
		s.toolID = fmt.Sprintf("tool-%s-%d", s.base, s.toolN)
		s.toolBody = s.toolID + "-body"
		s.appendNode(s.toolCard("", nil, true)) // pending "running…" until the result lands
	case "tool_result":
		s.handleToolResult(ev)
		s.openBubble()
	case "error":
		if s.buf.Len() > 0 {
			s.buf.WriteString(" — ")
		}
		s.buf.WriteString("the thread snapped: " + s.h.chatErrText(ev.Err))
		s.morphNode(chat.MessageBody(s.bodyID, s.buf.String()))
	}
}

// handleToolResult mirrors the prior consumer order: uicard → choices →
// proposal → plain. The open tool row's body is morphed with the result.
// uicard and cluster artifacts route to the panel + chip; proposals/choices/
// refresh stay inline in the transcript.
func (s *chatStream) handleToolResult(ev agent.Event) {
	if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
		s.endArtifactCard(rest, typ, query)
		return
	}
	if prompt, choices, _, ok := tools.ParseChoices(ev.Text); ok {
		s.endTool("choices offered", "")
		s.appendChoices(prompt, choices)
		return
	}
	if kind, id, rest, ok := tools.ParseProposal(ev.Text); ok {
		s.endTool(rest, s.h.proposalBody(kind, id))
		return
	}
	if types, rest, ok := tools.ParseRefresh(ev.Text); ok {
		s.endTool(clipText(rest, 2000), "")
		for _, typ := range types {
			s.refreshCard(typ)
		}
		return
	}
	if title, cs, rest, ok := tools.ParseArtifact(ev.Text); ok {
		s.endArtifactCluster(rest, title, cs)
		return
	}
	s.endTool(clipText(ev.Text, 2000), "")
}

// endTool morphs the open tool row with its result and, when a card is
// attached, appends it inline (proposals stay in the transcript). Artifact
// callers use endArtifactCard / endArtifactCluster instead.
func (s *chatStream) endTool(content string, card template.HTML) {
	s.morphNode(s.toolCard(content, nil, false))
	if card == "" {
		return
	}
	// Only proposals reach here with a card; they stay inline in the transcript.
	s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), g.Raw(string(card))))
}

// endArtifactCard morphs the tool card (its body carrying the re-open chip),
// then routes the single card to the panel (single-active). Mirrors the owner
// door (show.go) so live and reload agree; the chip lives inside the tool turn
// (plan: tool-call consistency) rather than as a loose #chat sibling.
func (s *chatStream) endArtifactCard(content, typ, query string) {
	s.morphNode(s.toolCard(content, s.h.chipNode(typ, query), false))
	_ = store.SetOwnerSetting(s.h.app, panelActiveKey, showURL(typ, query))
	s.morphNode(s.h.panelNode(typ, query)) // morph #panel-inner
}

// endArtifactCluster routes an agent cluster to the panel; the non-clickable
// chip (clusters have no deterministic re-open URL — plan 090) rides inside the
// tool card body. Clusters do not update panel_active (no restore URL).
func (s *chatStream) endArtifactCluster(content, title string, cs []cards.Card) {
	s.morphNode(s.toolCard(content, clusterChipNode(title), false))
	s.morphNode(s.h.panelClusterNode(title, cs))
}

// refreshCard re-renders one registry card from live data and morphs it in
// place by its ucard-{type} root id. Selector-less PatchElements is a blind
// patch-if-present: it updates the card if it's on screen and silently no-ops
// otherwise (an off-board or focus-view owner is unaffected). nil params render
// the card's default — safe for non-parameterized cards (today).
func (s *chatStream) refreshCard(typ string) {
	_ = s.sse.PatchElements(string(s.h.cardHTML(typ, nil)))
}

// appendChoices appends a live dialogue-choice panel.
func (s *chatStream) appendChoices(prompt string, choices []tools.Choice) {
	items := make([]chat.ChoiceItem, len(choices))
	for i, c := range choices {
		items[i] = chat.ChoiceItem{Label: c.Label, Hint: c.Hint}
	}
	node := chat.Choices(chat.ChoicesProps{
		Prompt: prompt, Nonce: newNonce(), OwnerName: s.ownerName,
		SoulAvatarSrc: s.soulURL, Choices: items,
	})
	_ = s.sse.PatchElements(s.renderNode(node), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
}

// note appends a standalone assistant message (e.g. the honesty check note).
func (s *chatStream) note(origin, content string) {
	s.appendNode(chat.Message(chat.MessageProps{
		Role: "balaur", AvatarSrc: s.balaURL, Who: s.who, Origin: origin, Content: content,
	}))
}

// finish closes the last open bubble and clears the streaming signal.
func (s *chatStream) finish() {
	s.finalizeBubble()
	_ = s.sse.MarshalAndPatchSignals(struct {
		Streaming bool `json:"streaming"`
	}{false})
}
