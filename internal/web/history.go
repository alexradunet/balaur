package web

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

// history.go renders chat transcripts for page-load reloads: the messageView
// payload, renderMessages (transcript -> chat.* components), chatBodyHTML (the
// #chat body: history or the hearth greeting), and messageViews (records ->
// payloads, decoding the persisted tool-result markers). The live SSE renderer
// lives next door in chatstream.go and renders the same chat.* components, so
// reloaded and streamed turns match. Moved out of recap.go (plan 203).

// messageView is one chat message's template payload (history + day expand).
type messageView struct {
	Role            string
	Tool            string
	Content         string
	Origin          string // agent-initiated marker: "nudge" | "briefing"; "" = chat
	CardURL         string // inline card embed endpoint (legacy; kept for the lazy-mount tests)
	CardBody        g.Node // server-rendered inline card, embedded directly (proposals/inline only)
	ArtifactTitle   string // non-empty for uicard/cluster artifacts (drives the chip label)
	ArtifactIcon    string // /static/icons stem for the chip ("" = none)
	ArtifactType    string // non-empty for single-card artifacts (drives the clickable chip URL)
	ArtifactQuery   string // raw query string for the single-card re-open URL
	SoulAvatarURL   string // resolved soul avatar URL (same for all views in one call)
	BalaurAvatarURL string // resolved Balaur head avatar URL
	OwnerName       string // display name for the "You" label
	WhoLabel        string // assistant display name ("Balaur", or the active head's name)
	Args            string // tool-call arguments (pretty JSON) for the collapsed fold on reload

	// Datastar streaming fields (master chat dock). BubbleID/BodyID give a
	// streamed element a stable id so the SSE handler can morph it in place;
	// Pending marks the live "thinking" state on an assistant bubble.
	BubbleID string
	BodyID   string
	Pending  bool
}

// renderMessages renders a chat transcript via the storybook components
// (chat.Message speech panels + chat.ToolRow trail) — the single source of chat
// markup for page-load history, the Home greeting, and day-recap expansion. The
// live stream (chatstream.go) renders the same components, so history and
// streamed turns match.
func (h *handlers) renderMessages(views []messageView) g.Node {
	nodes := make([]g.Node, 0, len(views))
	for _, mv := range views {
		switch mv.Role {
		case "user":
			nodes = append(nodes, chat.Message(chat.MessageProps{
				Role: "user", AvatarSrc: mv.SoulAvatarURL, Who: mv.OwnerName, Content: mv.Content,
			}))
		case "tool":
			// Re-open chip rides inside the tool card body (matching the live
			// stream) so a tool call reads as one consistent Balaur turn.
			var chip g.Node
			switch {
			case mv.ArtifactType != "": // single card → clickable re-open chip
				chip = h.chipNode(mv.ArtifactType, mv.ArtifactQuery)
			case mv.ArtifactTitle != "": // cluster → non-clickable chip
				chip = clusterChipNode(mv.ArtifactTitle)
			}
			nodes = append(nodes, chat.ToolRow(chat.ToolRowProps{
				Tool: mv.Tool, Icon: toolIconFile(mv.Tool), Who: mv.WhoLabel,
				AvatarSrc: mv.BalaurAvatarURL, Content: mv.Content, Args: mv.Args, Chip: chip,
			}))
			if mv.CardBody != nil { // proposal → framed as a Balaur card turn below
				nodes = append(nodes, chat.CardTurn(chat.CardTurnProps{
					Who: mv.WhoLabel, AvatarSrc: mv.BalaurAvatarURL, Card: mv.CardBody,
				}))
			}
		default: // assistant
			nodes = append(nodes, chat.Message(chat.MessageProps{
				Role: "balaur", AvatarSrc: mv.BalaurAvatarURL, Who: mv.WhoLabel, Origin: mv.Origin, Content: mv.Content,
			}))
		}
	}
	return g.Group(nodes)
}

// chatBodyHTML renders the #chat body: the conversation history when present,
// otherwise the hearth greeting (the crest + a balaur welcome, or the model
// setup notice). Everything goes through the chat components so the empty state,
// history, and the live stream share one look.
func (h *handlers) chatBodyHTML(d homeData) g.Node {
	// A compact today puts its rolling summary atop the dock, above whatever
	// remains of the live thread (usually nothing — the clean slate).
	if d.CompactSummary != "" {
		nodes := []g.Node{chat.CompactNote(d.CompactSummary)}
		if len(d.History) > 0 {
			nodes = append(nodes, h.renderMessages(d.History))
		} else {
			nodes = append(nodes, chat.Message(chat.MessageProps{
				Role: "balaur", AvatarSrc: d.BalaurAvatarURL, Who: "Balaur",
				Content: "Folded the earlier thread into the note above. Clean slate — where to next?",
			}))
		}
		return g.Group(nodes)
	}
	if len(d.History) > 0 {
		return h.renderMessages(d.History)
	}
	content := "I am here. The hearth is lit and your words stay on this box. What shall we weigh today?"
	if !d.ChatReady {
		content = d.ModelError
		if d.ModelHint != "" {
			content += "\n" + d.ModelHint
		}
	}
	crest := g.El("img", g.Attr("class", "hearth-crest"), g.Attr("src", "/static/crest.png"),
		g.Attr("alt", "The Balaur crest — a three-headed dragon holding a glowing orb and a tome"))
	greeting := chat.Message(chat.MessageProps{Role: "balaur", AvatarSrc: d.BalaurAvatarURL, Who: "Balaur", Content: content})
	return g.Group([]g.Node{crest, greeting})
}

func (h *handlers) messageViews(recs []*core.Record) []messageView {
	soulURL := store.SoulAvatarURL(h.app)
	balaurURL := store.BalaurAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)
	out := make([]messageView, 0, len(recs))
	// Tool-call args ride on the assistant record's tool_payload (one entry per
	// call); the matching tool-result rows follow in order with no persisted
	// call-id link. Queue them per turn so each reloaded tool row shows the same
	// args fold as the live stream.
	var pendingArgs []string
	for _, r := range recs {
		mv := messageView{
			Role:            r.GetString("role"),
			Tool:            r.GetString("tool_name"),
			Content:         r.GetString("content"),
			Origin:          r.GetString("origin"),
			SoulAvatarURL:   soulURL,
			BalaurAvatarURL: balaurURL,
			OwnerName:       ownerName,
			WhoLabel:        "Balaur",
		}
		switch mv.Role {
		case "assistant":
			// Capture even for tool-call-only turns (skipped below) so the
			// queue stays aligned with the tool rows that follow.
			if raw := r.GetString("tool_payload"); raw != "" {
				var calls []struct {
					Args string `json:"args"`
				}
				if json.Unmarshal([]byte(raw), &calls) == nil {
					for _, c := range calls {
						pendingArgs = append(pendingArgs, prettyJSON(c.Args))
					}
				}
			}
		case "tool":
			if len(pendingArgs) > 0 {
				mv.Args, pendingArgs = pendingArgs[0], pendingArgs[1:]
			}
		}
		// Re-render marked tool results.
		// Consumer order: uicard → choices → proposal → refresh → plain.
		// uicard: safe and useful to re-render on reload — it lazy-fetches
		//   current data from the registry, so the card is always live.
		// choices: degrade to inert plain text — no live panel on reload
		//   (avoids resubmitting stale decisions).
		// proposal: renders an approval card on first view and on reload.
		// refresh: drop the live-patch directive — show the plain text only
		//   (there is no card to patch on reload).
		if mv.Role == "tool" {
			if typ, query, rest, ok := tools.ParseUICard(mv.Content); ok {
				// uicard: record coordinates for the re-open chip; artifact lives in the panel.
				mv.Content = rest
				mv.ArtifactType, mv.ArtifactQuery = typ, query
				mv.ArtifactTitle, mv.ArtifactIcon = cardTitleIcon(typ)
			} else if _, _, modelText, ok := tools.ParseChoices(mv.Content); ok {
				mv.Content = clipText(modelText, 2000)
			} else if kind, id, rest, ok := tools.ParseProposal(mv.Content); ok {
				mv.CardBody, mv.Content = h.proposalBody(kind, id), rest
			} else if _, rest, ok := tools.ParseRefresh(mv.Content); ok {
				// Live refresh has no meaning on reload; show the plain text only.
				mv.Content = clipText(rest, 2000)
			} else if title, cs, rest, ok := tools.ParseArtifact(mv.Content); ok {
				// cluster: non-clickable chip; ArtifactType stays "" (no re-open URL).
				mv.Content = rest
				mv.ArtifactTitle = title
				_ = cs // cluster body lives in the panel on live path; on reload just a chip
			}
		}
		if mv.Role == "assistant" && mv.Content == "" {
			continue // tool-call-only turns carry nothing visible
		}
		out = append(out, mv)
	}
	return out
}
