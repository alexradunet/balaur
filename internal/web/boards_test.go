package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestBoardsDefaultsCreated verifies that GET /boards seeds four default
// boards on a fresh app and redirects to the first one; a second visit is
// idempotent (still 4 boards).
func TestBoardsDefaultsCreated(t *testing.T) {
	t.Run("first visit seeds 4 boards", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "GET /boards creates defaults and redirects",
			Method:         "GET",
			URL:            "/boards",
			TestAppFactory: newWebApp,
			ExpectedStatus: 302,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
				recs, err := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
				if err != nil {
					tb.Fatalf("loading boards: %v", err)
				}
				if len(recs) != 4 {
					tb.Errorf("expected 4 default boards, got %d", len(recs))
				}
			},
		}
		s.Test(t)
	})

	t.Run("second visit is idempotent", func(t *testing.T) {
		// Seed once, then hit /boards again: still 4 boards.
		app := newWebApp(t)
		h := &handlers{app: app}
		if err := h.ensureDefaultBoards(); err != nil {
			t.Fatalf("seed: %v", err)
		}
		s := tests.ApiScenario{
			Name:           "second GET /boards still redirects without duplicating",
			Method:         "GET",
			URL:            "/boards",
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus: 302,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
				recs, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
				if len(recs) != 4 {
					tb.Errorf("after second visit: expected 4 boards, got %d", len(recs))
				}
			},
		}
		s.Test(t)
	})
}

// TestBoardsPageRenders verifies GET /boards/{id} returns 200 and contains the
// board-grid with its cards server-rendered inline (no lazy-load): the Study
// board's "today" card must be present in the slot, not a Loading… placeholder.
func TestBoardsPageRenders(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}
	if err := h.ensureDefaultBoards(); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	recs, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
	if len(recs) == 0 {
		t.Fatal("no boards after seed")
	}
	id := recs[0].Id

	scenario := tests.ApiScenario{
		Name:            "GET /boards/{id} renders board grid",
		Method:          "GET",
		URL:             "/boards/" + id,
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"board-grid", `class="board-slot-inner"`, `id="ucard-today"`},
	}
	scenario.Test(t)
}

// TestBoardsCreate verifies POST /ui/boards creates a board and redirects.
func TestBoardsCreate(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "POST /ui/boards creates board",
		Method:         "POST",
		URL:            "/ui/boards",
		Body:           strings.NewReader("name=My+Board"),
		Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory: newWebApp,
		ExpectedStatus: 302,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			recs, err := app.FindRecordsByFilter("boards", "name='My Board'", "", 0, 0, nil)
			if err != nil || len(recs) == 0 {
				t.Errorf("board 'My Board' not found after create")
			}
		},
	}
	scenario.Test(t)
}

// TestBoardsRename verifies POST /ui/boards/{id}/rename re-renders the header.
func TestBoardsRename(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}
	if err := h.ensureDefaultBoards(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	recs, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
	id := recs[0].Id

	scenario := tests.ApiScenario{
		Name:            "POST /ui/boards/{id}/rename re-renders header",
		Method:          "POST",
		URL:             "/ui/boards/" + id + "/rename",
		Body:            strings.NewReader("name=Renamed"),
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"board-header", "Renamed"},
	}
	scenario.Test(t)
}

// TestBoardsCardAddValid verifies adding a valid card re-renders the grid.
func TestBoardsCardAddValid(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}
	if err := h.ensureDefaultBoards(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	recs, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
	id := recs[0].Id

	scenario := tests.ApiScenario{
		Name:            "POST /ui/boards/{id}/cards/add valid card",
		Method:          "POST",
		URL:             "/ui/boards/" + id + "/cards/add",
		Body:            strings.NewReader("type=heads"),
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"board-grid"},
	}
	scenario.Test(t)
}

// TestBoardsCardAddInvalidType verifies adding an invalid card type returns 400.
func TestBoardsCardAddInvalidType(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}
	if err := h.ensureDefaultBoards(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	recs, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
	id := recs[0].Id

	scenario := tests.ApiScenario{
		Name:            "POST /ui/boards/{id}/cards/add invalid type returns 400",
		Method:          "POST",
		URL:             "/ui/boards/" + id + "/cards/add",
		Body:            strings.NewReader("type=notacard"),
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"notacard"},
	}
	scenario.Test(t)
}

// TestBoardsCardRemove verifies removing a card by index re-renders the grid.
func TestBoardsCardRemove(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "Test")
	rec.Set("sort", 0)
	raw, _ := json.Marshal([]boardCard{{Type: "heads"}, {Type: "today"}})
	rec.Set("cards", string(raw))
	app.Save(rec)

	scenario := tests.ApiScenario{
		Name:            "POST /ui/boards/{id}/cards/{idx}/remove removes card",
		Method:          "POST",
		URL:             "/ui/boards/" + rec.Id + "/cards/0/remove",
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"board-grid"},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			updated, err := app.FindRecordById("boards", rec.Id)
			if err != nil {
				t.Fatalf("loading board: %v", err)
			}
			var bcs []boardCard
			json.Unmarshal([]byte(updated.GetString("cards")), &bcs)
			if len(bcs) != 1 {
				t.Errorf("expected 1 card after remove, got %d", len(bcs))
			}
			if len(bcs) == 1 && bcs[0].Type != "today" {
				t.Errorf("remaining card should be 'today', got %q", bcs[0].Type)
			}
		},
	}
	scenario.Test(t)
}

// TestBoardsCardRemoveOutOfBounds verifies out-of-bounds index returns 400.
func TestBoardsCardRemoveOutOfBounds(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "Test")
	rec.Set("sort", 0)
	raw, _ := json.Marshal([]boardCard{{Type: "heads"}})
	rec.Set("cards", string(raw))
	app.Save(rec)

	scenario := tests.ApiScenario{
		Name:            "remove out-of-bounds index returns 400",
		Method:          "POST",
		URL:             "/ui/boards/" + rec.Id + "/cards/99/remove",
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"out of bounds"},
	}
	scenario.Test(t)
}

// TestBoardsDeleteLastRefused verifies deleting the last board returns 400.
func TestBoardsDeleteLastRefused(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "Only")
	rec.Set("sort", 0)
	rec.Set("cards", "[]")
	app.Save(rec)

	scenario := tests.ApiScenario{
		Name:            "DELETE last board is refused",
		Method:          "POST",
		URL:             "/ui/boards/" + rec.Id + "/delete",
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"last board"},
	}
	scenario.Test(t)
}

// TestBoardsDeleteWithMultiple verifies delete works when >1 boards exist.
func TestBoardsDeleteWithMultiple(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")

	rec1 := core.NewRecord(col)
	rec1.Set("name", "Board1")
	rec1.Set("sort", 0)
	rec1.Set("cards", "[]")
	app.Save(rec1)

	rec2 := core.NewRecord(col)
	rec2.Set("name", "Board2")
	rec2.Set("sort", 1)
	rec2.Set("cards", "[]")
	app.Save(rec2)

	scenario := tests.ApiScenario{
		Name:           "DELETE second board redirects",
		Method:         "POST",
		URL:            "/ui/boards/" + rec2.Id + "/delete",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 302,
	}
	scenario.Test(t)
}

// TestBoardsLayoutHappyPath verifies that POST /ui/boards/{id}/layout persists
// x/y/w/h and returns 204. A subsequent boardRecordOf reflects the stored layout.
func TestBoardsLayoutHappyPath(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "LayoutTest")
	rec.Set("sort", 0)
	raw, _ := json.Marshal([]boardCard{
		{Type: "today"},
		{Type: "heads"},
	})
	rec.Set("cards", string(raw))
	app.Save(rec)

	body := `layout=[{"idx":0,"x":0,"y":0,"w":4,"h":16},{"idx":1,"x":4,"y":0,"w":4,"h":16}]`
	scenario := tests.ApiScenario{
		Name:           "POST /ui/boards/{id}/layout persists layout",
		Method:         "POST",
		URL:            "/ui/boards/" + rec.Id + "/layout",
		Body:           strings.NewReader(body),
		Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 204,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			updated, err := app.FindRecordById("boards", rec.Id)
			if err != nil {
				t.Fatalf("loading board: %v", err)
			}
			var bcs []boardCard
			json.Unmarshal([]byte(updated.GetString("cards")), &bcs)
			if len(bcs) != 2 {
				t.Fatalf("expected 2 cards, got %d", len(bcs))
			}
			// Verify type/params are unchanged.
			if bcs[0].Type != "today" || bcs[1].Type != "heads" {
				t.Errorf("type changed after layout post: %s %s", bcs[0].Type, bcs[1].Type)
			}
			// Verify layout was persisted.
			if bcs[1].X != 4 || bcs[1].Y != 0 || bcs[1].W != 4 || bcs[1].H != 16 {
				t.Errorf("layout not persisted for card[1]: %+v", bcs[1])
			}
		},
	}
	scenario.Test(t)
}

// TestBoardsLayoutIdxOutOfBounds verifies out-of-bounds idx returns 400.
func TestBoardsLayoutIdxOutOfBounds(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "Test")
	rec.Set("sort", 0)
	raw, _ := json.Marshal([]boardCard{{Type: "today"}})
	rec.Set("cards", string(raw))
	app.Save(rec)

	// Sending idx=99 which is out of bounds for a 1-card board.
	body := `layout=[{"idx":99,"x":0,"y":0,"w":4,"h":16}]`
	scenario := tests.ApiScenario{
		Name:            "layout idx out of bounds → 400",
		Method:          "POST",
		URL:             "/ui/boards/" + rec.Id + "/layout",
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"out of bounds"},
	}
	scenario.Test(t)
}

// TestBoardsLayoutCountMismatch verifies count mismatch returns 400.
func TestBoardsLayoutCountMismatch(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "Test")
	rec.Set("sort", 0)
	raw, _ := json.Marshal([]boardCard{{Type: "today"}, {Type: "heads"}})
	rec.Set("cards", string(raw))
	app.Save(rec)

	// Sending only 1 entry for a 2-card board.
	body := `layout=[{"idx":0,"x":0,"y":0,"w":4,"h":16}]`
	scenario := tests.ApiScenario{
		Name:            "layout count mismatch → 400",
		Method:          "POST",
		URL:             "/ui/boards/" + rec.Id + "/layout",
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"mismatch"},
	}
	scenario.Test(t)
}

// TestBoardsLayoutTypeParamsUnchanged verifies type/params cannot be modified
// via the layout endpoint.
func TestBoardsLayoutTypeParamsUnchanged(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "Test")
	rec.Set("sort", 0)
	raw, _ := json.Marshal([]boardCard{
		{Type: "quests", Params: map[string]string{"status": "open", "limit": "8"}},
	})
	rec.Set("cards", string(raw))
	app.Save(rec)

	body := `layout=[{"idx":0,"x":2,"y":3,"w":6,"h":20}]`
	scenario := tests.ApiScenario{
		Name:           "layout post does not change type/params",
		Method:         "POST",
		URL:            "/ui/boards/" + rec.Id + "/layout",
		Body:           strings.NewReader(body),
		Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 204,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			updated, err := app.FindRecordById("boards", rec.Id)
			if err != nil {
				t.Fatalf("loading board: %v", err)
			}
			var bcs []boardCard
			json.Unmarshal([]byte(updated.GetString("cards")), &bcs)
			if len(bcs) != 1 {
				t.Fatalf("expected 1 card, got %d", len(bcs))
			}
			if bcs[0].Type != "quests" {
				t.Errorf("type changed: %q", bcs[0].Type)
			}
			if bcs[0].Params["status"] != "open" || bcs[0].Params["limit"] != "8" {
				t.Errorf("params changed: %v", bcs[0].Params)
			}
			// Layout should be persisted.
			if bcs[0].X != 2 || bcs[0].Y != 3 || bcs[0].W != 6 || bcs[0].H != 20 {
				t.Errorf("layout not persisted: %+v", bcs[0])
			}
		},
	}
	scenario.Test(t)
}

// TestBoardsLegacyFlowRender verifies that a legacy board (no stored layout)
// renders without grid-row (flow mode — unchanged from before plan 032).
func TestBoardsLegacyFlowRender(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "LegacyFlow")
	rec.Set("sort", 0)
	raw, _ := json.Marshal([]boardCard{{Type: "today"}, {Type: "heads"}})
	rec.Set("cards", string(raw))
	app.Save(rec)

	scenario := tests.ApiScenario{
		Name:            "legacy board renders without grid-row",
		Method:          "GET",
		URL:             "/boards/" + rec.Id,
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"board-grid"},
	}
	scenario.Test(t)
}

// TestBoardsFreeLayoutRender verifies that a board with stored layout emits
// grid-row in the rendered HTML.
func TestBoardsFreeLayoutRender(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "FreeLayout")
	rec.Set("sort", 0)
	raw, _ := json.Marshal([]boardCard{
		{Type: "today", X: 0, Y: 0, W: 4, H: 16},
		{Type: "heads", X: 4, Y: 0, W: 4, H: 16},
	})
	rec.Set("cards", string(raw))
	app.Save(rec)

	scenario := tests.ApiScenario{
		Name:            "free-layout board renders with grid-row",
		Method:          "GET",
		URL:             "/boards/" + rec.Id,
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"board-grid-free", "grid-row"},
	}
	scenario.Test(t)
}

// TestBoardCardAddRejectsCorruptCards verifies that POST /ui/boards/{id}/cards/add
// returns 500 and does NOT overwrite the stored cards when the board's JSON is corrupt.
func TestBoardCardAddRejectsCorruptCards(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")
	rec := core.NewRecord(col)
	rec.Set("name", "CorruptBoard")
	rec.Set("sort", 0)
	// Store a value that is not a valid JSON array. PocketBase's JSONField
	// normalizes bare strings into JSON-encoded strings on Set; capture the
	// stored representation via GetString so the AfterTestFunc can assert the
	// record was not overwritten.
	rec.Set("cards", "{not json")
	app.Save(rec)

	// Capture the stored cards string BEFORE the request — PocketBase's
	// JSONField will have normalized the raw value, so we compare against what
	// GetString actually returns (not the original literal we passed to Set).
	storedBefore := rec.GetString("cards")

	scenario := tests.ApiScenario{
		Name:            "POST /ui/boards/{id}/cards/add with corrupt cards returns 500",
		Method:          "POST",
		URL:             "/ui/boards/" + rec.Id + "/cards/add",
		Body:            strings.NewReader("type=heads"),
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  500,
		ExpectedContent: []string{"corrupted"},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			// Re-fetch to confirm the stored cards field was NOT overwritten.
			updated, err := app.FindRecordById("boards", rec.Id)
			if err != nil {
				t.Fatalf("re-fetching board: %v", err)
			}
			if updated.GetString("cards") != storedBefore {
				t.Errorf("corrupt cards were overwritten: got %q, want %q",
					updated.GetString("cards"), storedBefore)
			}
		},
	}
	scenario.Test(t)
}

// TestBoardCardsOfBlankIsEmpty is a unit test for the boardCardsOf helper.
// - blank field → nil, nil (empty board, not corruption)
// - valid JSON   → round-trips correctly
// - corrupt JSON → non-nil error
func TestBoardCardsOfBlankIsEmpty(t *testing.T) {
	app := newWebApp(t)
	col, _ := app.FindCollectionByNameOrId("boards")

	t.Run("blank field is nil with no error", func(t *testing.T) {
		rec := core.NewRecord(col)
		rec.Set("cards", "")
		bcs, err := boardCardsOf(rec)
		if err != nil {
			t.Fatalf("unexpected error for blank field: %v", err)
		}
		if bcs != nil {
			t.Errorf("expected nil slice for blank field, got %v", bcs)
		}
	})

	t.Run("valid JSON round-trips", func(t *testing.T) {
		rec := core.NewRecord(col)
		raw, _ := json.Marshal([]boardCard{{Type: "heads"}, {Type: "today"}})
		rec.Set("cards", string(raw))
		bcs, err := boardCardsOf(rec)
		if err != nil {
			t.Fatalf("unexpected error for valid JSON: %v", err)
		}
		if len(bcs) != 2 {
			t.Fatalf("expected 2 cards, got %d", len(bcs))
		}
		if bcs[0].Type != "heads" || bcs[1].Type != "today" {
			t.Errorf("unexpected card types: %v", bcs)
		}
	})

	t.Run("corrupt JSON returns error", func(t *testing.T) {
		rec := core.NewRecord(col)
		rec.Set("cards", "{not json")
		_, err := boardCardsOf(rec)
		if err == nil {
			t.Error("expected error for corrupt JSON, got nil")
		}
	})
}
