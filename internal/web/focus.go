package web

// focus.go — a single card expanded to the full canvas (plan 050). The "page"
// in Balaur's card-first UI is just a card at full size. GET /focus/{type} is
// dual-mode like boardsPage: a Datastar @get patches only #main (the dock and
// its live chat persist); a direct browser load renders the whole shell.

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/cards"
)

// focusView is the template data for focus_main / focus_page.
//
// MainClass exists only because focus_page reuses the shared shell_open partial
// (layout.html), which reads {{with .MainClass}}; Go templates error on a
// missing field, so the field must be present even though focus leaves it "".
type focusView struct {
	Title     string        // document title (full-load only; shell adds "· Balaur")
	Dock      homeData      // companion dock (full-load only)
	MainClass string        // shell_open hook; always "" for focus
	Type      string        // card type
	Label     string        // card label, for the focus header
	Body      template.HTML // server-rendered focus card body
	BackHref  string        // where "← Back" returns
}

// focusBackHref returns the board to return to. A focus opened from a board
// carries ?from={boardID}; one opened from the launcher has none, so we fall
// back to /boards (which redirects to the first board).
func focusBackHref(from string) string {
	if from == "" {
		return "/boards"
	}
	return "/boards/" + from
}

// focusParams validates the card params and, for cards that have a richer
// interactive view, defaults the focus surface to mode=manage. Per-feature
// phases (051+) replace this generic focus body with a bespoke full view.
func focusParams(typ string, q url.Values) (map[string]string, error) {
	params, err := cards.Validate(typ, queryToMap(q))
	if err != nil {
		return nil, err
	}
	if cards.HasManage(typ) && params["mode"] == "" {
		params["mode"] = "manage"
	}
	return params, nil
}

// focusCanonicalQuery drops the transient "from" key from the reflected URL.
func focusCanonicalQuery(q url.Values) string {
	c := url.Values{}
	for k, vs := range q {
		if k == "from" || len(vs) == 0 {
			continue
		}
		c[k] = vs
	}
	return c.Encode()
}

// focusPage handles GET /focus/{type}?params[&from={boardID}].
func (h *handlers) focusPage(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	spec, ok := cards.Get(typ)
	if !ok {
		return e.NotFoundError("no such card type", nil)
	}

	q := e.Request.URL.Query()
	params, err := focusParams(typ, q)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}

	view := focusView{
		Type:     typ,
		Label:    spec.Label,
		Body:     h.cardHTML(typ, params),
		BackHref: focusBackHref(q.Get("from")),
	}

	if isDatastarRequest(e) {
		sse := datastar.NewSSE(e.Response, e.Request)
		var b strings.Builder
		if err := h.tmpl.ExecuteTemplate(&b, "focus_main", view); err != nil {
			return e.InternalServerError("rendering focus", err)
		}
		if err := sse.PatchElements(b.String(),
			datastar.WithSelectorID("main"), datastar.WithModeInner()); err != nil {
			return nil // client gone
		}
		canonical := "/focus/" + typ
		if qs := focusCanonicalQuery(q); qs != "" {
			canonical += "?" + qs
		}
		if u, err := url.Parse(canonical); err == nil {
			_ = sse.ReplaceURL(*u)
		}
		_ = sse.ExecuteScript(fmt.Sprintf("document.title=%q", spec.Label+" · Balaur"))
		return nil
	}

	dock, err := h.dockData()
	if err != nil {
		return e.InternalServerError("loading companion dock", err)
	}
	view.Title = spec.Label
	view.Dock = dock
	return h.render(e, "focus_page", view)
}
