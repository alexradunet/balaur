package web

// boards.go — owner-composed dashboards of typed cards (plan 029).
// Routes registered in web.go. Layout is server-defined (no drag/resize).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/cards"
)

// boardCard is one slot in a board's cards array.
type boardCard struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}

// boardView is the template data for the full boards page and fragments.
type boardView struct {
	Boards  []*boardRecord // all boards, sorted by .Sort
	Current *boardRecord   // the active board (nil on redirect)
	Specs   []cards.Spec   // for the "add a card" palette
}

// boardRecord is the view-model for one board row.
type boardRecord struct {
	ID    string
	Name  string
	Sort  int
	Cards []boardCardView
}

// boardCardView is one rendered slot in the grid.
type boardCardView struct {
	Type  string
	W     int    // grid column span (from registry spec)
	Query string // URL-encoded query string, e.g. "?status=open&limit=8"
	Idx   int    // position in the cards array (for remove route)
}

// validateBoardCards validates and cleans a slice of boardCards via the
// registry. Used by ensureDefaultBoards, create, and add handlers so the
// validation rule is a single definition.
func validateBoardCards(bcs []boardCard) error {
	for i, bc := range bcs {
		if _, _, err := func() (string, map[string]string, error) {
			cleaned, err := cards.Validate(bc.Type, bc.Params)
			return bc.Type, cleaned, err
		}(); err != nil {
			return fmt.Errorf("card[%d]: %w", i, err)
		}
	}
	return nil
}

// boardCardViewsOf converts a []boardCard to []boardCardView, resolving
// each card's grid span from the registry. Unknown types are silently kept
// with W=4 so the page stays usable after a registry change.
func boardCardViewsOf(bcs []boardCard) []boardCardView {
	out := make([]boardCardView, 0, len(bcs))
	for i, bc := range bcs {
		w := 4
		if spec, ok := cards.Get(bc.Type); ok {
			w = spec.W
		}
		var q string
		if len(bc.Params) > 0 {
			vals := url.Values{}
			for k, v := range bc.Params {
				vals.Set(k, v)
			}
			q = "?" + vals.Encode()
		}
		out = append(out, boardCardView{
			Type:  bc.Type,
			W:     w,
			Query: q,
			Idx:   i,
		})
	}
	return out
}

// boardRecordOf builds a boardRecord from a PocketBase record.
func boardRecordOf(rec *core.Record) *boardRecord {
	var bcs []boardCard
	_ = json.Unmarshal([]byte(rec.GetString("cards")), &bcs)
	return &boardRecord{
		ID:    rec.Id,
		Name:  rec.GetString("name"),
		Sort:  int(rec.GetFloat("sort")),
		Cards: boardCardViewsOf(bcs),
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
		if err := validateBoardCards(d.cards); err != nil {
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

// boardsIndex handles GET /boards — redirects to the first board by sort.
func (h *handlers) boardsIndex(e *core.RequestEvent) error {
	if err := h.ensureDefaultBoards(); err != nil {
		return e.InternalServerError("seeding default boards", err)
	}
	boards, err := h.loadBoards()
	if err != nil || len(boards) == 0 {
		return e.InternalServerError("loading boards", err)
	}
	return e.Redirect(http.StatusFound, "/boards/"+boards[0].ID)
}

// boardsPage handles GET /boards/{id} — the full boards page.
func (h *handlers) boardsPage(e *core.RequestEvent) error {
	if err := h.ensureDefaultBoards(); err != nil {
		return e.InternalServerError("seeding default boards", err)
	}
	id := e.Request.PathValue("id")
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
	if current == nil {
		return e.NotFoundError("board not found", nil)
	}

	return h.render(e, "boards.html", boardView{
		Boards:  boards,
		Current: current,
		Specs:   cards.All(),
	})
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
	existing, _ := h.loadBoards()
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

	boards, _ := h.loadBoards()
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}
	return h.render(e, "board_header", boardView{
		Boards:  boards,
		Current: current,
		Specs:   cards.All(),
	})
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
	return e.Redirect(http.StatusFound, "/boards")
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

	var bcs []boardCard
	_ = json.Unmarshal([]byte(rec.GetString("cards")), &bcs)

	newCard := boardCard{Type: typ}
	if len(cleaned) > 0 {
		newCard.Params = cleaned
	}
	bcs = append(bcs, newCard)

	raw, _ := json.Marshal(bcs)
	rec.Set("cards", string(raw))
	if err := h.app.Save(rec); err != nil {
		return e.InternalServerError("saving board", err)
	}

	boards, _ := h.loadBoards()
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}
	return h.render(e, "board_grid", boardView{
		Boards:  boards,
		Current: current,
		Specs:   cards.All(),
	})
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

	var bcs []boardCard
	_ = json.Unmarshal([]byte(rec.GetString("cards")), &bcs)

	if idx < 0 || idx >= len(bcs) {
		return e.BadRequestError("index out of bounds", nil)
	}

	bcs = append(bcs[:idx], bcs[idx+1:]...)
	raw, _ := json.Marshal(bcs)
	rec.Set("cards", string(raw))
	if err := h.app.Save(rec); err != nil {
		return e.InternalServerError("saving board", err)
	}

	boards, _ := h.loadBoards()
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}
	return h.render(e, "board_grid", boardView{
		Boards:  boards,
		Current: current,
		Specs:   cards.All(),
	})
}
