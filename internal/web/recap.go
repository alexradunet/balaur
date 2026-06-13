package web

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
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

// recapBands renders the whole telescope above the chat history.
func (h *handlers) recapBands(e *core.RequestEvent) error {
	master, err := conversation.Master(h.app)
	if err != nil {
		return e.InternalServerError("master conversation", err)
	}
	oldest, ok := conversation.OldestMessageTime(h.app, master.Id)
	if !ok {
		e.Response.WriteHeader(http.StatusOK)
		return nil // no history, nothing further back
	}
	// Same timezone as generation (store.OwnerLocation).
	loc := store.OwnerLocation(h.app)
	oldest = oldest.In(loc)

	type bandView struct {
		Heading string
		Cards   []recapView
	}
	var view []bandView
	for _, band := range recap.Bands(time.Now().In(loc), oldest) {
		bv := bandView{Heading: bandHeading(band.Type)}
		for _, p := range band.Periods {
			rec := recap.Find(h.app, master.Id, p)
			card := h.recapCard(p, rec)
			if card.Missing {
				continue // quiet or not-yet-summarised periods stay invisible
			}
			bv.Cards = append(bv.Cards, card)
		}
		if len(bv.Cards) > 0 {
			view = append(view, bv)
		}
	}

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "recap-bands.html", view); err != nil {
		return e.InternalServerError("rendering recap", err)
	}
	return nil
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
	unix, err := strconv.ParseInt(e.Request.URL.Query().Get("start"), 10, 64)
	if err != nil {
		return e.BadRequestError("bad start", err)
	}
	p := recap.Containing(periodType, time.Unix(unix, 0).In(store.OwnerLocation(h.app)))

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Days expand to their preserved transcript.
	if p.Type == "day" {
		msgs, err := conversation.MessagesBetween(h.app, master.Id, p.Start, p.End)
		if err != nil {
			return e.InternalServerError("loading day", err)
		}
		return h.tmpl.ExecuteTemplate(e.Response, "chat-messages.html", h.messageViews(msgs))
	}

	// Everything else expands to its child summaries.
	var cards []recapView
	for _, child := range recap.Children(p) {
		if rec := recap.Find(h.app, master.Id, child); rec != nil {
			cards = append(cards, h.recapCard(child, rec))
		}
	}
	if len(cards) == 0 {
		fmt.Fprint(e.Response, `<p class="k-empty">Nothing recorded in this stretch.</p>`)
		return nil
	}
	return h.tmpl.ExecuteTemplate(e.Response, "recap-cards.html", cards)
}

// messageView is one chat message's template payload (history + day expand).
type messageView struct {
	Role            string
	Tool            string
	Content         string
	Origin          string // agent-initiated marker: "nudge" | "briefing"; "" = chat
	CardURL         string // inline card embed endpoint, when the tool result carried one
	SoulAvatarURL   string // resolved soul avatar URL (same for all views in one call)
	BalaurAvatarURL string // resolved Balaur head avatar URL
	OwnerName       string // display name for the "You" label
	WhoLabel        string // assistant display name ("Balaur", or head name for branch chats)
}

func (h *handlers) messageViews(recs []*core.Record) []messageView {
	soulURL := store.SoulAvatarURL(h.app)
	balaurURL := store.BalaurAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)
	out := make([]messageView, 0, len(recs))
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
		// Re-render marked tool results.
		// Consumer order: uicard → choices → proposal → plain.
		// uicard: safe and useful to re-render on reload — it lazy-fetches
		//   current data from the registry, so the card is always live.
		// choices: degrade to inert plain text — no live panel on reload
		//   (avoids resubmitting stale decisions).
		// proposal: renders an approval card on first view and on reload.
		if mv.Role == "tool" {
			if typ, query, rest, ok := tools.ParseUICard(mv.Content); ok {
				mv.CardURL = "/ui/cards/" + typ + "?" + query
				mv.Content = rest
			} else if _, _, modelText, ok := tools.ParseChoices(mv.Content); ok {
				mv.Content = clipText(modelText, 2000)
			} else if kind, id, rest, ok := tools.ParseProposal(mv.Content); ok {
				mv.CardURL, mv.Content = cardURL(kind, id), rest
			}
		}
		if mv.Role == "assistant" && mv.Content == "" {
			continue // tool-call-only turns carry nothing visible
		}
		out = append(out, mv)
	}
	return out
}
