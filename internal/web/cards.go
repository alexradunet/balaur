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
	"html"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/ui"
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

// uiCardPalette handles GET /ui/cards — the palette listing all card specs.
func (h *handlers) uiCardPalette(e *core.RequestEvent) error {
	specs := cards.All()
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "ucard_palette", specs); err != nil {
		return e.InternalServerError("rendering card palette", err)
	}
	return nil
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
		fmt.Fprintf(e.Response, `<div class="card-note card-note-error" id="ucard-%s">%s</div>`, typ, html.EscapeString(err.Error()))
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
	if fn, ok := ui.LookupCard(typ); ok {
		node, err := fn(ui.Tile, params)
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

// cardHTML server-renders one card to HTML for inline embedding in a board grid.
// It validates the stored params (defending against hand-edited board JSON) and
// renders the same error strip the HTTP endpoint uses on failure, so a single
// bad card never blanks the whole board.
func (h *handlers) cardHTML(typ string, params map[string]string) template.HTML {
	if _, ok := cards.Get(typ); !ok {
		return cardErrorStrip("no such card type: " + typ)
	}
	cleaned, err := cards.Validate(typ, params)
	if err != nil {
		return cardErrorStrip(err.Error())
	}
	var b strings.Builder
	if err := h.cardInto(&b, typ, cleaned); err != nil {
		h.app.Logger().Warn("board card render failed", "type", typ, "err", err)
		return cardErrorStrip("could not render this card")
	}
	return template.HTML(b.String())
}

// cardErrorStrip is the inline card-error fragment (no id — several cards of the
// same type may coexist on a board, and the slot already scopes it).
func cardErrorStrip(msg string) template.HTML {
	return template.HTML(`<div class="card-note card-note-error">` + html.EscapeString(msg) + `</div>`)
}

// uicardBody server-renders a registry card ("/ui/cards/{type}?query") for inline
// chat embeds — so the chat stream and reloaded history carry the card directly,
// with no lazy htmx mount.
func (h *handlers) uicardBody(typ, query string) template.HTML {
	vals, _ := url.ParseQuery(query)
	return h.cardHTML(typ, queryToMap(vals))
}

// proposalBody server-renders an approval/proposal card (a task, or a knowledge
// record) for inline chat embeds. Returns "" when the record can't be loaded, so
// the tool row degrades to plain text rather than a broken card.
func (h *handlers) proposalBody(kind, id string) template.HTML {
	if kind == "tasks" {
		rec, err := h.app.FindRecordById("tasks", id)
		if err != nil {
			return ""
		}
		s, err := h.taskCardHTML(rec)
		if err != nil {
			return ""
		}
		return template.HTML(s)
	}
	rec, err := h.app.FindRecordById(kind, id) // collection name == kind ("memories"/"skills")
	if err != nil {
		return ""
	}
	s, err := h.renderCardHTML(knowledge.Kind(kind), rec)
	if err != nil {
		return ""
	}
	return template.HTML(s)
}
