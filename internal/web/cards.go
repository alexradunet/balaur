package web

// cards.go — typed card registry handlers (plan 028).
// GET /ui/cards/{type}?params → one rendered card fragment.
// GET /ui/cards            → palette: HTML index of all card specs.
//
// Card tiles are rendered by feature-owned gomponents renderers (see
// internal/feature/*, registered via feature.RegisterAll); this file keeps only
// the shared dispatch (cardInto/cardHTML), the chat embeds (uicardBody/
// proposalBody), and the palette.

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	hh "maragu.dev/gomponents/html" // aliased: the handler receiver is named h

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

// queryToMap converts url.Values to a flat map[string]string (first value
// wins). Only used by the card handlers; complex multi-value params are out
// of scope for the card registry.
func queryToMap(q url.Values) map[string]string {
	m := make(map[string]string, len(q))
	for k, vs := range q {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	return m
}

// cardPaletteNode renders the GET /ui/cards palette: the human/agent index of
// every registered card spec.
func cardPaletteNode(specs []cards.Spec) g.Node {
	rows := make([]g.Node, 0, len(specs))
	for _, s := range specs {
		var params g.Node = g.Text("")
		if len(s.Params) > 0 {
			items := make([]g.Node, 0, len(s.Params))
			for _, p := range s.Params {
				var req g.Node = g.Text("")
				if p.Required {
					req = g.Group([]g.Node{g.Text(" "), ui.Tag(g.Text("required"))})
				}
				enum := ""
				if len(p.Enum) > 0 {
					enum = " [" + strings.Join(p.Enum, ", ") + "]"
				}
				items = append(items, hh.Li(
					hh.Code(g.Text(p.Name)), req,
					g.Text(enum+" — "+p.Doc),
				))
			}
			params = hh.Ul(hh.Class("ucard-params"), g.Group(items))
		}
		rows = append(rows, hh.Li(hh.Class("ucard-row"),
			hh.Span(hh.Class("ucard-title"),
				hh.Img(hh.Class("tool-icon"), hh.Src("/static/icons/"+s.Icon+".png"), hh.Alt("")),
				hh.Code(g.Text(s.Type)), g.Text(" — "+s.Label),
			),
			hh.Span(hh.Class("kcard-meta"), g.Text(fmt.Sprintf("w=%d", s.W))),
			params,
		))
	}
	return hh.Section(hh.Class("k-section ucard-palette"),
		hh.H2(hh.Class("k-heading"), g.Text("Card palette")),
		hh.Ul(hh.Class("ucard-list"), g.Group(rows)),
	)
}

// uiCardPalette handles GET /ui/cards — the palette listing all card specs.
func (h *handlers) uiCardPalette(e *core.RequestEvent) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return cardPaletteNode(cards.All()).Render(e.Response)
}

// uiCard handles GET /ui/cards/{type}?params — one rendered card fragment.
func (h *handlers) uiCard(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	if _, ok := cards.Get(typ); !ok {
		return e.NotFoundError("no such card type", nil)
	}

	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		// Validation error: render a card-note-error strip, HTTP 200.
		e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		_ = ui.ErrorStripID("ucard-"+typ, err.Error()).Render(e.Response)
		return nil
	}

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.cardInto(e.Response, typ, params)
}

// cardInto renders one card of the given (already-validated) type into w. It is
// the single dispatch shared by the HTTP endpoint (w = e.Response) and the
// board grid (w = an in-process buffer, via cardHTML) — so a card is rendered
// the same way whether it's lazily fetched or server-rendered inline.
func (h *handlers) cardInto(w io.Writer, typ string, params map[string]string) error {
	return h.cardSizeInto(w, typ, params, ui.Tile)
}

// cardSizeInto renders one card at the given size. cardInto passes ui.Tile (the
// /ui/cards endpoint, refreshCard, and show_cards clusters); a single in-chat
// artifact passes ui.Focus via cardFocusHTML/uicardBody. A feature renderer that
// ignores the size argument (most do) renders its tile in both cases — only cards
// with a real Focus branch (quests/memory/skills/settings/journal/day/lifelog)
// differ, showing their full interactive surface.
func (h *handlers) cardSizeInto(w io.Writer, typ string, params map[string]string, size ui.CardSize) error {
	if fn, ok := ui.LookupCard(typ); ok {
		node, err := fn(size, params)
		if err != nil {
			return err
		}
		return node.Render(w)
	}
	// Every card type is now served by a feature-owned gomponents renderer
	// (registered via feature.RegisterAll). An unregistered type is a bug or a
	// hand-edited board; surface it rather than rendering a stale tile.
	return fmt.Errorf("unhandled card type %q", typ)
}

// cardHTMLAt server-renders one card at the given size to a g.Node for inline
// embedding, with validate + error-strip discipline. cardHTML/cardFocusHTML are
// thin wrappers choosing the size and the log context.
func (h *handlers) cardHTMLAt(typ string, params map[string]string, size ui.CardSize, logMsg string) g.Node {
	if _, ok := cards.Get(typ); !ok {
		return cardErrorStrip("no such card type: " + typ)
	}
	cleaned, err := cards.Validate(typ, params)
	if err != nil {
		return cardErrorStrip(err.Error())
	}
	var b strings.Builder
	if err := h.cardSizeInto(&b, typ, cleaned, size); err != nil {
		h.app.Logger().Warn(logMsg, "type", typ, "err", err)
		return cardErrorStrip("could not render this card")
	}
	return g.Raw(b.String())
}

// cardHTML server-renders one card to a g.Node for inline embedding in a board grid.
// It validates the stored params (defending against hand-edited board JSON) and
// renders the same error strip the HTTP endpoint uses on failure, so a single
// bad card never blanks the whole board.
func (h *handlers) cardHTML(typ string, params map[string]string) g.Node {
	return h.cardHTMLAt(typ, params, ui.Tile, "board card render failed")
}

// cardErrorStrip is the inline card-error fragment (no id — several cards of the
// same type may coexist on a board, and the slot already scopes it).
func cardErrorStrip(msg string) g.Node {
	return ui.ErrorStrip(msg)
}

// uicardBody server-renders a single registry card as an in-chat artifact, at
// ui.Focus — the FULL interactive surface (manager / rail / write-form), not a
// summary tile. In the single-page UI the chat IS the canvas (there is no board),
// so a summoned domain artifact is its full working surface; size-agnostic cards
// render identically to their tile. Used by BOTH the deterministic door
// (/ui/show, via messageViews) and the agent's card_show, and re-rendered on
// reload through this same path — so the live append and the reload stay
// consistent (the #1 invariant from plan 088).
func (h *handlers) uicardBody(typ, query string) g.Node {
	vals, _ := url.ParseQuery(query)
	return h.cardFocusHTML(typ, queryToMap(vals))
}

// cardFocusHTML server-renders one card at ui.Focus (its full surface), with the
// same validate + error-strip discipline as cardHTML. Restored for the in-chat
// artifact path after plan 089 removed the /focus pages — a sidebar domain click
// or card_show now shows the real manager, not a dead summary.
func (h *handlers) cardFocusHTML(typ string, params map[string]string) g.Node {
	return h.cardHTMLAt(typ, params, ui.Focus, "focus card render failed")
}

// artifactBody server-renders a hand-picked cluster of cards as a chat.Cluster.
// Used by panelClusterNode (the panel body for show_cards); each card is
// rendered via cardHTML (validated + error-stripped). The panel head owns the
// title so the cluster is untitled here (no duplicate heading).
func (h *handlers) artifactBody(title string, cs []cards.Card) g.Node {
	nodes := make([]g.Node, 0, len(cs))
	for _, c := range cs {
		nodes = append(nodes, h.cardHTML(c.Type, c.Params))
	}
	return chat.Cluster(chat.ClusterProps{Cards: nodes})
}

// proposalBody server-renders an approval/proposal card (a task, or a knowledge
// record) for inline chat embeds. Returns nil when the record can't be loaded, so
// the tool row degrades to plain text rather than a broken card.
func (h *handlers) proposalBody(kind, id string) g.Node {
	if kind == "tasks" {
		rec, err := h.app.FindRecordById("tasks", id)
		if err != nil {
			return nil
		}
		s, err := h.taskCardHTML(rec)
		if err != nil {
			return nil
		}
		return g.Raw(s)
	}
	// kind is the "nodes" collection; the record's type ("memory"/"skill")
	// selects which knowledge card to render.
	rec, err := h.app.FindRecordById(kind, id)
	if err != nil {
		return nil
	}
	k := knowledge.Kind(rec.GetString("type"))
	knowledge.Hydrate(k, rec) // alias node fields → legacy names the card reads
	s, err := h.renderCardHTML(k, rec)
	if err != nil {
		return nil
	}
	return g.Raw(s)
}
