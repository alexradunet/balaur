package web

// focus.go — a single card expanded to the full canvas (plan 050). The "page"
// in Balaur's card-first UI is just a card at full size. GET /focus/{type} is
// dual-mode like boardsPage: a Datastar @get patches only #main (the dock and
// its live chat persist); a direct browser load renders the whole shell.

import (
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// focusActiveKey maps a card type to its top-level topbar nav key (see
// shell.Topbar) so the active domain rides gold on the focus page. Types that
// are not a top-level domain (today, calendar, timeline, habits, measure, lines)
// return "" — no nav item is marked current.
func focusActiveKey(typ string) string {
	switch typ {
	case "quests":
		return "quests"
	case "memory", "skills":
		return "knowledge"
	case "lifelog":
		return "life"
	case "journal", "day":
		return "journal"
	case "heads":
		return "heads"
	case "settings":
		return "settings"
	}
	return ""
}

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

// safeBoardID accepts only a board-id-shaped value (the charset PocketBase
// record ids use). The focus "from" param is the one attacker-controllable
// input that reaches the Back control; constraining it here keeps the safety
// self-evident instead of relying solely on html/template's JS escaping.
var safeBoardID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)

// focusBackHref returns the board to return to. A focus opened from a board
// carries ?from={boardID}; one opened from the launcher, or with a
// malformed/forged from, falls back to /boards (which redirects to the first
// board).
func focusBackHref(from string) string {
	if !safeBoardID.MatchString(from) {
		return "/boards"
	}
	return "/boards/" + from
}

// focusBodyHTML renders a card's focus body. A few card types have a bespoke,
// richer focus view (the surface of the page they replace); every other type
// falls back to the generic registry render (manage mode where available).
func (h *handlers) focusBodyHTML(typ string, params map[string]string) template.HTML {
	switch typ {
	case "quests":
		return h.questsFocusHTML()
	case "day":
		return h.dayFocusHTML(params)
	case "memory":
		return h.knowledgeFocusHTML(knowledge.Memory)
	case "skills":
		return h.knowledgeFocusHTML(knowledge.Skill)
	case "settings":
		return h.settingsFocusHTML(params)
	}
	// Feature cards that implement the CardSize.Focus branch (e.g. lifelog) render
	// their full-canvas body here; the rest fall back to their Tile render.
	return h.cardFocusHTML(typ, params)
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
		Body:     h.focusBodyHTML(typ, params),
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

	// Full document load: render the new gomponents shell (top-nav topbar +
	// persistent dock) with the focus body in #main — the same shell Home uses.
	// The Datastar @get branch above still patches only #main (focus_main), so
	// the dock and its live chat survive in-app navigation.
	dock, err := h.dockData()
	if err != nil {
		return e.InternalServerError("loading companion dock", err)
	}
	var dockHTML strings.Builder
	if err := h.tmpl.ExecuteTemplate(&dockHTML, "chat_dock", dock); err != nil {
		return e.InternalServerError("rendering companion dock", err)
	}
	var bodyHTML strings.Builder
	if err := h.tmpl.ExecuteTemplate(&bodyHTML, "focus_main", view); err != nil {
		return e.InternalServerError("rendering focus", err)
	}
	page := shell.Page(shell.PageProps{
		Title:  spec.Label,
		Active: focusActiveKey(typ),
		Body:   g.Raw(bodyHTML.String()),
		Dock:   g.Raw(dockHTML.String()),
	})
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(e.Response); err != nil {
		return e.InternalServerError("rendering focus page", err)
	}
	return nil
}
