package web

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

// The recap telescope UI: the chat page tops out with a "further back"
// sentinel that lazily loads summary bands (days → weeks → months →
// quarters → years). Each band card expands one level down via /expand.

// recapView is one summary card's template payload.
type recapView struct {
	Type     string
	Label    string
	Content  string
	Start    string // Unix seconds for expand requests (URL-safe)
	Date     string // day cards: YYYY-MM-DD link to the day page
	HasChild bool
	Missing  bool // period in range but not summarised yet
}

// bandView is one telescope band (a heading over a row of recap cards).
type bandView struct {
	Heading string
	Cards   []recapView
}

func (h *handlers) recapCard(p recap.Period, rec *core.Record) recapView {
	v := recapView{
		Type:     p.Type,
		Label:    recap.Label(p),
		Start:    fmt.Sprintf("%d", p.Start.Unix()),
		HasChild: p.Type != "day",
	}
	if p.Type == "day" {
		v.Date = p.Start.Format("2006-01-02")
	}
	if rec != nil {
		v.Content = rec.GetString("content")
	} else {
		v.Missing = true
	}
	return v
}

func bandHeading(periodType string) string {
	switch periodType {
	case "day":
		return "Earlier this week"
	case "week":
		return "Past weeks"
	case "month":
		return "Past months"
	case "quarter":
		return "Past quarters"
	default:
		return "Past years"
	}
}

// recapExpand renders one period's children (or its raw day transcript).
func (h *handlers) recapExpand(e *core.RequestEvent) error {
	master, err := conversation.Master(h.app)
	if err != nil {
		return e.InternalServerError("master conversation", err)
	}
	periodType := e.Request.URL.Query().Get("type")
	switch periodType {
	case "day", "week", "month", "quarter", "year":
	default:
		return e.BadRequestError("bad period type", nil)
	}
	unix, err := strconv.ParseInt(e.Request.URL.Query().Get("start"), 10, 64)
	if err != nil {
		return e.BadRequestError("bad start", err)
	}
	p := recap.Containing(periodType, time.Unix(unix, 0).In(store.OwnerLocation(h.app)))

	// The expand patches the card's own children container, whose id is built
	// from the (validated) period type + start — the same id the template emits.
	targetID := fmt.Sprintf("recap-children-%s-%d", periodType, unix)
	sse := datastar.NewSSE(e.Response, e.Request)
	var b strings.Builder

	// Days expand to their preserved transcript; everything else to child cards.
	if p.Type == "day" {
		msgs, err := conversation.MessagesBetween(h.app, master.Id, p.Start, p.End)
		if err != nil {
			return e.InternalServerError("loading day", err)
		}
		b.WriteString(renderNodeHTML(h.renderMessages(h.messageViews(msgs))))
	} else {
		var cards []recapView
		for _, child := range recap.Children(p) {
			if rec := recap.Find(h.app, master.Id, child); rec != nil {
				cards = append(cards, h.recapCard(child, rec))
			}
		}
		if len(cards) == 0 {
			b.WriteString(`<p class="k-empty">Nothing recorded in this stretch.</p>`)
		} else {
			b.WriteString(renderNodeHTML(recapCardsNode(cards)))
		}
	}
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID(targetID), datastar.WithModeInner())
	return nil
}

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

// chronicleView loads the telescope bands for the master conversation, oldest last.
func (h *handlers) chronicleView() []bandView {
	master, err := conversation.Master(h.app)
	if err != nil {
		return nil
	}
	oldest, ok := conversation.OldestMessageTime(h.app, master.Id)
	if !ok {
		return nil
	}
	loc := store.OwnerLocation(h.app)
	oldest = oldest.In(loc)
	var view []bandView
	for _, band := range recap.Bands(time.Now().In(loc), oldest) {
		bv := bandView{Heading: bandHeading(band.Type)}
		for _, p := range band.Periods {
			card := h.recapCard(p, recap.Find(h.app, master.Id, p))
			if card.Missing {
				continue
			}
			bv.Cards = append(bv.Cards, card)
		}
		if len(bv.Cards) > 0 {
			view = append(view, bv)
		}
	}
	return view
}

// chronicleBody is the Chronicle page body: telescope top-down (newest band first), or an empty state.
func (hh *handlers) chronicleBody() g.Node {
	return chronicleBandsNode(hh.chronicleView())
}

// chronicleBandsNode renders Chronicle bands top-down (newest first), or the empty state.
func chronicleBandsNode(view []bandView) g.Node {
	if len(view) == 0 {
		return h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"), g.Text("Chronicle")),
			h.P(h.Class("k-sub"), g.Text("No history yet. As days pass and recaps are kept, your past appears here — days, then weeks, months, and years.")))
	}
	bands := make([]g.Node, 0, len(view)*2)
	for _, b := range view {
		bands = append(bands,
			h.Section(h.Class("recap-band"),
				h.H2(h.Class("recap-heading"),
					h.Span(h.Class("recap-rune"), g.Text("◇")), g.Text(" "+b.Heading)),
				recapCardsNode(b.Cards)),
			h.Div(h.Class("stitch")))
	}
	return h.Div(h.Class("chronicle-focus"), g.Group(bands))
}

// recapCardsNode renders a row of recap cards. The card head+body OPEN the
// period/day node (day → /ui/show/day, coarser → /ui/show/period); a secondary
// button still peeks inline (day transcript / child cards). The inline expand
// renders into .recap-children, which sits OUTSIDE the clickable open-zone so
// an expanded transcript never re-triggers navigation.
func recapCardsNode(cards []recapView) g.Node {
	items := make([]g.Node, 0, len(cards))
	for _, c := range cards {
		// Node URL: coarser lenses open the synthesised period node; days open
		// the day node. @get morphs the panel (a plain nav would render raw SSE).
		nodeURL := "/ui/show/day?date=" + c.Date
		expandType := "day"
		expandLabel := "transcript"
		if c.HasChild {
			nodeURL = "/ui/show/period?type=" + c.Type + "&start=" + c.Start
			expandType = c.Type
			expandLabel = "open"
		}
		openNode := "@get('" + nodeURL + "'); basmOpenPanel()"

		// Label is a real anchor (keyboard/AT focusable) to the node; stop
		// propagation so it doesn't also fire the open-zone's click.
		label := h.A(h.Class("recap-label"), h.Href(nodeURL),
			g.Attr("data-on:click__prevent", "evt.stopPropagation(); "+openNode),
			g.Text(c.Label))
		// Secondary inline peek; stopPropagation so it expands without navigating.
		expand := h.Button(h.Class("recap-expand"), h.Type("button"),
			g.Attr("data-on:click", "evt.stopPropagation(); el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type="+expandType+"&start="+c.Start+"')"),
			g.Text(expandLabel))

		items = append(items, h.Article(h.Class("recap-card recap-"+c.Type),
			h.Div(h.Class("recap-open-zone"), g.Attr("data-on:click", openNode),
				h.Header(h.Class("recap-head"), label, expand),
				h.P(h.Class("recap-body"), g.Text(c.Content)),
			),
			h.Div(h.Class("recap-children"), h.ID("recap-children-"+c.Type+"-"+c.Start)),
		))
	}
	return g.Group(items)
}
