package web

import (
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
		children := recap.Children(p)
		byPeriod, err := recap.FindMany(h.app, master.Id, children)
		if err != nil {
			return e.InternalServerError("loading summaries", err)
		}
		var cards []recapView
		for _, child := range children {
			if rec := recap.Lookup(byPeriod, child); rec != nil {
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
		byPeriod, err := recap.FindMany(h.app, master.Id, band.Periods)
		if err != nil {
			return nil // same fail-soft contract as the early returns above
		}
		bv := bandView{Heading: bandHeading(band.Type)}
		for _, p := range band.Periods {
			card := h.recapCard(p, recap.Lookup(byPeriod, p))
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
