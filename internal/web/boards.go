package web

// boards.go — owner-composed dashboards of typed cards (plan 029+032).
// Routes registered in web.go. Layout persisted per board via drag/resize (plan 032).

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/cards"
)

// boardCard is one slot in a board's cards array. It mirrors cards.Card
// exactly in JSON shape; using the leaf type directly would create a view
// concern in the cards package, so we keep this local alias.
type boardCard = cards.Card

// boardView is the template data for the full boards page and fragments.
type boardView struct {
	Boards  []*boardRecord // all boards, sorted by .Sort
	Current *boardRecord   // the active board (nil on redirect)
	Specs   []cards.Spec   // for the "add a card" palette
}

// boardRecord is the view-model for one board row.
type boardRecord struct {
	ID      string
	Name    string
	Sort    int
	Cards   []boardCardView
	FreeLay bool // true when at least one card has an explicit position stored
}

// boardCardView is one rendered slot in the grid.
type boardCardView struct {
	Type      string
	W         int               // grid column span (from registry spec or card layout)
	H         int               // grid row span (from registry spec or card layout)
	X         int               // 0-based column start (0 = flow mode)
	Y         int               // 0-based row start (0 = flow mode)
	X1        int               // X+1 for CSS grid-column start (precomputed)
	Y1        int               // Y+1 for CSS grid-row start (precomputed)
	HasPos    bool              // true when explicit position was stored (free layout mode)
	Query     string            // URL-encoded query string, e.g. "?status=open&limit=8"
	FocusHref string            // /focus/{type}?{params}&from={boardID}
	Params    map[string]string // raw params, for server-rendering Body
	Idx       int               // position in the cards array (for remove route)
	Body      template.HTML     // server-rendered card HTML (filled by renderBoardCards)
}

// boardCardViewsOf converts a []boardCard to []boardCardView, resolving each
// card's grid dimensions. If at least one card has an explicit position stored
// (X>0 || Y>0 in its card record), all cards use free-layout positioning.
// Legacy boards (no explicit position on any card) render with the old
// flow-layout (grid-column: span W, no grid-row), so existing boards look
// unchanged until the first drag. The second return value signals free mode.
func boardCardViewsOf(bcs []boardCard, boardID string) ([]boardCardView, bool) {
	// Determine whether this board is in free-layout mode.
	// A board is free if any card has X>0 or Y>0 or an explicit W or H.
	freeLay := false
	for _, bc := range bcs {
		if bc.X > 0 || bc.Y > 0 || bc.W > 0 || bc.H > 0 {
			freeLay = true
			break
		}
	}

	out := make([]boardCardView, 0, len(bcs))
	for i, bc := range bcs {
		specW, specH := 4, 16
		if spec, ok := cards.Get(bc.Type); ok {
			specW = spec.W
			if spec.H > 0 {
				specH = spec.H
			}
		}

		// Resolve W and H: card value if >0 else spec default.
		w := specW
		if bc.W > 0 {
			w = bc.W
		}
		h := specH
		if bc.H > 0 {
			h = bc.H
		}

		var q string
		if len(bc.Params) > 0 {
			vals := url.Values{}
			for k, v := range bc.Params {
				vals.Set(k, v)
			}
			q = "?" + vals.Encode()
		}

		fparams := url.Values{}
		for k, v := range bc.Params {
			fparams.Set(k, v)
		}
		fparams.Set("from", boardID)
		focusHref := "/focus/" + bc.Type + "?" + fparams.Encode()

		hasPos := freeLay // all slots use free mode if any one card has explicit pos
		out = append(out, boardCardView{
			Type:      bc.Type,
			W:         w,
			H:         h,
			X:         bc.X,
			Y:         bc.Y,
			X1:        bc.X + 1,
			Y1:        bc.Y + 1,
			HasPos:    hasPos,
			Query:     q,
			FocusHref: focusHref,
			Params:    bc.Params,
			Idx:       i,
		})
	}
	return out, freeLay
}

// boardCardsOf decodes the record's cards JSON. A blank field (including
// PocketBase's JSON-encoded empty string "\"\"") is an empty board; anything
// else that fails to parse is corruption the caller must surface — proceeding
// would overwrite the owner's composition.
func boardCardsOf(rec *core.Record) ([]boardCard, error) {
	raw := rec.GetString("cards")
	trimmed := strings.TrimSpace(raw)
	// PocketBase stores an empty JSON field as the literal "" (a JSON-encoded
	// empty string). An empty or blank raw value also means no cards.
	if trimmed == "" || trimmed == `""` {
		return nil, nil
	}
	var bcs []boardCard
	if err := json.Unmarshal([]byte(raw), &bcs); err != nil {
		return nil, fmt.Errorf("decoding board %s cards: %w", rec.Id, err)
	}
	return bcs, nil
}

// boardRecordOf builds a boardRecord from a PocketBase record.
func boardRecordOf(rec *core.Record) *boardRecord {
	// corrupt cards render as an empty board; loadBoards logs it.
	bcs, _ := boardCardsOf(rec)
	views, freeLay := boardCardViewsOf(bcs, rec.Id)
	return &boardRecord{
		ID:      rec.Id,
		Name:    rec.GetString("name"),
		Sort:    int(rec.GetFloat("sort")),
		Cards:   views,
		FreeLay: freeLay,
	}
}

// renderBoardCards server-renders each card's body into the board record so the
// grid template can emit it inline — no lazy-load, no per-card round-trip. Done
// only for the board actually being shown (never for the whole list), so the
// cost stays proportional to one board. A nil board is a no-op.
func (h *handlers) renderBoardCards(b *boardRecord) {
	if b == nil {
		return
	}
	for i := range b.Cards {
		b.Cards[i].Body = h.cardHTML(b.Cards[i].Type, b.Cards[i].Params)
	}
}

// loadBoards returns all boards sorted by sort field, then name.
func (h *handlers) loadBoards() ([]*boardRecord, error) {
	recs, err := h.app.FindRecordsByFilter("boards", "1=1", "sort,name", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	out := make([]*boardRecord, 0, len(recs))
	for _, r := range recs {
		if _, err := boardCardsOf(r); err != nil {
			h.app.Logger().Warn("board cards corrupted", "board", r.Id, "err", err)
		}
		out = append(out, boardRecordOf(r))
	}
	return out, nil
}

// ensureDefaultBoards seeds the four canonical boards when the collection is
// empty. Called lazily from the boards page handler — never from a migration.
// If any boards already exist the function is a no-op.
func (h *handlers) ensureDefaultBoards() error {
	count, err := h.app.CountRecords("boards", nil)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []struct {
		name  string
		sort  int
		cards []boardCard
	}{
		{
			name: "Study",
			sort: 0,
			cards: []boardCard{
				{Type: "today"},
				{Type: "quests", Params: map[string]string{"status": "open", "limit": "8"}},
				{Type: "calendar"},
			},
		},
		{
			name: "Quest log",
			sort: 1,
			cards: []boardCard{
				{Type: "quests", Params: map[string]string{"status": "open", "limit": "20"}},
				{Type: "calendar"},
			},
		},
		{
			name: "Self",
			sort: 2,
			cards: []boardCard{
				{Type: "journal", Params: map[string]string{"limit": "5"}},
				{Type: "timeline", Params: map[string]string{"days": "14"}},
			},
		},
		{
			name: "Balaur",
			sort: 3,
			cards: []boardCard{
				{Type: "memory", Params: map[string]string{"limit": "6"}},
				{Type: "skills", Params: map[string]string{"limit": "6"}},
				{Type: "heads"},
			},
		},
	}

	col, err := h.app.FindCollectionByNameOrId("boards")
	if err != nil {
		return err
	}

	for _, d := range defaults {
		if _, err := cards.ValidateCards(d.cards); err != nil {
			return fmt.Errorf("default board %q: %w", d.name, err)
		}
		raw, err := json.Marshal(d.cards)
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		rec.Set("name", d.name)
		rec.Set("sort", d.sort)
		rec.Set("cards", string(raw))
		if err := h.app.Save(rec); err != nil {
			return err
		}
	}
	return nil
}

// -- Page handlers --

// boardsIndex handles GET /boards — retired from nav; redirects home.
func (h *handlers) boardsIndex(e *core.RequestEvent) error {
	return e.Redirect(http.StatusFound, "/")
}

// boardPageData carries a board for the #main canvas plus the companion chat
// for the persistent dock (chat_dock fragment).
type boardPageData struct {
	boardView
	Title string
	Dock  homeData
}

// boardsPage handles GET /boards/{id} — retired from nav; redirects home.
func (h *handlers) boardsPage(e *core.RequestEvent) error {
	return e.Redirect(http.StatusFound, "/")
}

// boardsCreate handles POST /ui/boards — creates a new board.
func (h *handlers) boardsCreate(e *core.RequestEvent) error {
	if err := e.Request.ParseForm(); err != nil {
		return e.BadRequestError("invalid form", err)
	}
	name := strings.TrimSpace(e.Request.FormValue("name"))
	if name == "" {
		return e.BadRequestError("name required", nil)
	}
	if len(name) > 80 {
		return e.BadRequestError("name too long (max 80 characters)", nil)
	}

	col, err := h.app.FindCollectionByNameOrId("boards")
	if err != nil {
		return e.InternalServerError("boards collection", err)
	}

	// Determine next sort value.
	existing, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	nextSort := len(existing)

	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("sort", nextSort)
	rec.Set("cards", "[]")
	if err := h.app.Save(rec); err != nil {
		return e.InternalServerError("saving board", err)
	}
	return e.Redirect(http.StatusFound, "/boards/"+rec.Id)
}

// boardsRename handles POST /ui/boards/{id}/rename.
func (h *handlers) boardsRename(e *core.RequestEvent) error {
	if err := e.Request.ParseForm(); err != nil {
		return e.BadRequestError("invalid form", err)
	}
	name := strings.TrimSpace(e.Request.FormValue("name"))
	if name == "" {
		return e.BadRequestError("name required", nil)
	}
	if len(name) > 80 {
		return e.BadRequestError("name too long (max 80 characters)", nil)
	}

	id := e.Request.PathValue("id")
	rec, err := h.app.FindRecordById("boards", id)
	if err != nil {
		return e.NotFoundError("board not found", nil)
	}
	rec.Set("name", name)
	if err := h.app.Save(rec); err != nil {
		return e.InternalServerError("saving board", err)
	}

	boards, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}

	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "board_header", boardView{
		Boards:  boards,
		Current: current,
		Specs:   cards.All(),
	}); err != nil {
		return e.InternalServerError("rendering board header", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := sse.PatchElements(b.String(),
		datastar.WithSelectorID("board-header"), datastar.WithModeOuter()); err != nil {
		return nil // client gone
	}
	return nil
}

// boardsDelete handles POST /ui/boards/{id}/delete.
func (h *handlers) boardsDelete(e *core.RequestEvent) error {
	boards, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	if len(boards) <= 1 {
		return e.BadRequestError("cannot delete the last board", nil)
	}

	id := e.Request.PathValue("id")
	rec, err := h.app.FindRecordById("boards", id)
	if err != nil {
		return e.NotFoundError("board not found", nil)
	}
	if err := h.app.Delete(rec); err != nil {
		return e.InternalServerError("deleting board", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.Redirect("/boards")
	return nil
}

// boardsCardAdd handles POST /ui/boards/{id}/cards/add.
func (h *handlers) boardsCardAdd(e *core.RequestEvent) error {
	if err := e.Request.ParseForm(); err != nil {
		return e.BadRequestError("invalid form", err)
	}

	id := e.Request.PathValue("id")
	rec, err := h.app.FindRecordById("boards", id)
	if err != nil {
		return e.NotFoundError("board not found", nil)
	}

	typ := strings.TrimSpace(e.Request.FormValue("type"))
	if typ == "" {
		return e.BadRequestError("type required", nil)
	}

	// Collect params from form (skip "type" key).
	params := map[string]string{}
	for k, vs := range e.Request.Form {
		if k == "type" || len(vs) == 0 || vs[0] == "" {
			continue
		}
		params[k] = vs[0]
	}

	cleaned, err := cards.Validate(typ, params)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}

	bcs, err := boardCardsOf(rec)
	if err != nil {
		return e.InternalServerError("board cards are corrupted; fix the record in the dashboard", err)
	}

	newCard := boardCard{Type: typ}
	if len(cleaned) > 0 {
		newCard.Params = cleaned
	}
	bcs = append(bcs, newCard)

	raw, err := json.Marshal(bcs)
	if err != nil {
		return e.InternalServerError("encoding board cards", err)
	}
	rec.Set("cards", string(raw))
	if err := h.app.Save(rec); err != nil {
		return e.InternalServerError("saving board", err)
	}

	boards, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}
	h.renderBoardCards(current)

	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "board_grid", boardView{
		Boards:  boards,
		Current: current,
		Specs:   cards.All(),
	}); err != nil {
		return e.InternalServerError("rendering board grid", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := sse.PatchElements(b.String(),
		datastar.WithSelectorID("board-grid"), datastar.WithModeOuter()); err != nil {
		return nil // client gone
	}
	return nil
}

// boardsCardRemove handles POST /ui/boards/{id}/cards/{idx}/remove.
func (h *handlers) boardsCardRemove(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	idxStr := e.Request.PathValue("idx")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return e.BadRequestError("invalid index", nil)
	}

	rec, err := h.app.FindRecordById("boards", id)
	if err != nil {
		return e.NotFoundError("board not found", nil)
	}

	bcs, err := boardCardsOf(rec)
	if err != nil {
		return e.InternalServerError("board cards are corrupted; fix the record in the dashboard", err)
	}

	if idx < 0 || idx >= len(bcs) {
		return e.BadRequestError("index out of bounds", nil)
	}

	bcs = append(bcs[:idx], bcs[idx+1:]...)
	raw, err := json.Marshal(bcs)
	if err != nil {
		return e.InternalServerError("encoding board cards", err)
	}
	rec.Set("cards", string(raw))
	if err := h.app.Save(rec); err != nil {
		return e.InternalServerError("saving board", err)
	}

	boards, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}
	h.renderBoardCards(current)

	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "board_grid", boardView{
		Boards:  boards,
		Current: current,
		Specs:   cards.All(),
	}); err != nil {
		return e.InternalServerError("rendering board grid", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := sse.PatchElements(b.String(),
		datastar.WithSelectorID("board-grid"), datastar.WithModeOuter()); err != nil {
		return nil // client gone
	}
	return nil
}

// layoutEntry is one slot update sent by the client to POST /ui/boards/{id}/layout.
type layoutEntry struct {
	Idx int `json:"idx"`
	X   int `json:"x"`
	Y   int `json:"y"`
	W   int `json:"w"`
	H   int `json:"h"`
}

// boardsLayout handles POST /ui/boards/{id}/layout.
// Body: form field "layout" = JSON array of layoutEntry.
// Rules:
//   - idx must be in bounds (0..len(cards)-1)
//   - entry count must match board's card count
//   - type/params are NEVER modified — only x/y/w/h are touched
//
// Responds 204 No Content on success (the client already shows the result).
func (h *handlers) boardsLayout(e *core.RequestEvent) error {
	if err := e.Request.ParseForm(); err != nil {
		return e.BadRequestError("invalid form", err)
	}
	id := e.Request.PathValue("id")
	rec, err := h.app.FindRecordById("boards", id)
	if err != nil {
		return e.NotFoundError("board not found", nil)
	}

	rawLayout := e.Request.FormValue("layout")
	if rawLayout == "" {
		return e.BadRequestError("layout field required", nil)
	}
	var entries []layoutEntry
	if err := json.Unmarshal([]byte(rawLayout), &entries); err != nil {
		return e.BadRequestError("invalid layout JSON", err)
	}

	bcs, err := boardCardsOf(rec)
	if err != nil {
		return e.InternalServerError("board cards are corrupted; fix the record in the dashboard", err)
	}

	if len(entries) != len(bcs) {
		return e.BadRequestError("layout entry count mismatch", nil)
	}

	for _, le := range entries {
		if le.Idx < 0 || le.Idx >= len(bcs) {
			return e.BadRequestError("idx out of bounds", nil)
		}
	}

	// Apply x/y/w/h — type and params are never touched.
	for _, le := range entries {
		bcs[le.Idx].X = le.X
		bcs[le.Idx].Y = le.Y
		bcs[le.Idx].W = le.W
		bcs[le.Idx].H = le.H
	}

	// Validate (also clamps layout fields).
	cleaned, err := cards.ValidateCards(bcs)
	if err != nil {
		return e.BadRequestError("invalid cards after layout update", err)
	}

	rawCards, err := json.Marshal(cleaned)
	if err != nil {
		return e.InternalServerError("encoding board cards", err)
	}
	rec.Set("cards", string(rawCards))
	if err := h.app.Save(rec); err != nil {
		return e.InternalServerError("saving board layout", err)
	}

	e.Response.WriteHeader(http.StatusNoContent)
	return nil
}
