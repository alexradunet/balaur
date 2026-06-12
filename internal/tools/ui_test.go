package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/storetest"
)

// -- MarkUICard / ParseUICard round-trip tests --

func TestMarkParseUICardRoundTrip(t *testing.T) {
	params := map[string]string{"status": "open", "limit": "5"}
	marked := MarkUICard("quests", params, "showing the owner the Quest log card")

	typ, query, rest, ok := ParseUICard(marked)
	if !ok {
		t.Fatal("ParseUICard: ok=false, want true")
	}
	if typ != "quests" {
		t.Errorf("typ = %q, want %q", typ, "quests")
	}
	if !strings.Contains(query, "status=open") {
		t.Errorf("query %q missing status=open", query)
	}
	if !strings.Contains(query, "limit=5") {
		t.Errorf("query %q missing limit=5", query)
	}
	if rest != "showing the owner the Quest log card" {
		t.Errorf("rest = %q, want model text", rest)
	}
}

func TestMarkUICardNoParams(t *testing.T) {
	marked := MarkUICard("today", map[string]string{}, "showing the owner the Today card")
	if !strings.HasPrefix(marked, UICardMarker) {
		t.Fatal("MarkUICard: result does not start with UICardMarker")
	}
	typ, query, rest, ok := ParseUICard(marked)
	if !ok {
		t.Fatal("ParseUICard: ok=false")
	}
	if typ != "today" {
		t.Errorf("typ = %q, want today", typ)
	}
	if query != "" {
		t.Errorf("query = %q, want empty for no params", query)
	}
	if rest != "showing the owner the Today card" {
		t.Errorf("rest = %q, want model text", rest)
	}
}

func TestMarkUICardParamsNeedingURLEncoding(t *testing.T) {
	params := map[string]string{"kind": "body weight kg"}
	marked := MarkUICard("measure", params, "showing measure card")
	_, query, _, ok := ParseUICard(marked)
	if !ok {
		t.Fatal("ParseUICard: ok=false")
	}
	// The space should be URL-encoded.
	if strings.Contains(query, " ") {
		t.Errorf("query %q contains unencoded space — URL encoding failed", query)
	}
	if !strings.Contains(query, "body+weight+kg") && !strings.Contains(query, "body%20weight%20kg") {
		t.Errorf("query %q does not contain encoded kind value", query)
	}
}

func TestParseUICardOnPlainText(t *testing.T) {
	_, _, _, ok := ParseUICard("plain text with no marker")
	if ok {
		t.Error("ParseUICard on plain text: ok=true, want false")
	}
}

func TestParseUICardOnChoicesMarked(t *testing.T) {
	marked := MarkChoices("Pick one",
		[]Choice{{Label: "A"}, {Label: "B"}},
		"offered choices")
	_, _, _, ok := ParseUICard(marked)
	if ok {
		t.Error("ParseUICard on choices-marked text: ok=true, want false")
	}
}

func TestParseUICardOnProposalMarked(t *testing.T) {
	marked := MarkProposal("memories", "abc123", "memory proposal")
	_, _, _, ok := ParseUICard(marked)
	if ok {
		t.Error("ParseUICard on proposal-marked text: ok=true, want false")
	}
}

func TestParseUICardRegistryValidation(t *testing.T) {
	cases := []struct {
		name   string
		typ    string
		wantOK bool
	}{
		// Unknown type must be rejected.
		{name: "unknown type", typ: "nope", wantOK: false},
		// Path traversal attempts must be rejected.
		{name: "traversal dotdot", typ: "../model", wantOK: false},
		{name: "traversal encoded", typ: "..%2Fmodel", wantOK: false},
		// Known registered types must still pass.
		{name: "valid today", typ: "today", wantOK: true},
		{name: "valid quests", typ: "quests", wantOK: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			marked := UICardMarker + tc.typ + "?x=1\nmodel text"
			_, _, _, ok := ParseUICard(marked)
			if ok != tc.wantOK {
				t.Errorf("ParseUICard with typ=%q: ok=%v, want %v", tc.typ, ok, tc.wantOK)
			}
		})
	}
}

// -- card_show tool tests --

func TestCardShowHappyPath(t *testing.T) {
	app := storetest.NewApp(t)
	tool := cardShowTool(app)

	out, err := tool.Execute(context.Background(), `{"type":"today"}`)
	if err != nil {
		t.Fatalf("card_show today: unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, UICardMarker) {
		t.Errorf("card_show today: result does not start with UICardMarker: %q", out[:min(len(out), 60)])
	}
	typ, _, _, ok := ParseUICard(out)
	if !ok {
		t.Fatal("card_show today: ParseUICard ok=false")
	}
	if typ != "today" {
		t.Errorf("card_show today: typ = %q, want today", typ)
	}
}

func TestCardShowWithParams(t *testing.T) {
	app := storetest.NewApp(t)
	tool := cardShowTool(app)

	out, err := tool.Execute(context.Background(), `{"type":"quests","params":{"status":"open","limit":"5"}}`)
	if err != nil {
		t.Fatalf("card_show quests: unexpected error: %v", err)
	}
	typ, query, _, ok := ParseUICard(out)
	if !ok {
		t.Fatalf("card_show quests: ParseUICard ok=false, out=%q", out)
	}
	if typ != "quests" {
		t.Errorf("card_show quests: typ = %q, want quests", typ)
	}
	if !strings.Contains(query, "status=open") {
		t.Errorf("card_show quests: query %q missing status=open", query)
	}
}

func TestCardShowUnknownTypeReturnsPlainText(t *testing.T) {
	app := storetest.NewApp(t)
	tool := cardShowTool(app)

	out, err := tool.Execute(context.Background(), `{"type":"nonexistent_type"}`)
	if err != nil {
		t.Fatalf("card_show unknown type: unexpected Go error (must be plain-text result): %v", err)
	}
	// Must NOT use the marker.
	if strings.HasPrefix(out, UICardMarker) {
		t.Error("card_show unknown type: result starts with UICardMarker, must be plain text error")
	}
	// Must contain useful error text.
	if !strings.Contains(out, "unknown card type") && !strings.Contains(out, "invalid card") {
		t.Errorf("card_show unknown type: error text unhelpful: %q", out)
	}
}

func TestCardShowBadParamsReturnsPlainText(t *testing.T) {
	app := storetest.NewApp(t)
	tool := cardShowTool(app)

	// quests: status enum value "bogus" should fail validation.
	out, err := tool.Execute(context.Background(), `{"type":"quests","params":{"status":"bogus"}}`)
	if err != nil {
		t.Fatalf("card_show bad params: unexpected Go error: %v", err)
	}
	if strings.HasPrefix(out, UICardMarker) {
		t.Error("card_show bad params: result starts with UICardMarker, must be plain text error")
	}
}

// -- board_compose tool tests --

func TestBoardComposeCreatesRecord(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	tool := findTool(t, ts, "board_compose")

	out, err := tool.Execute(context.Background(),
		`{"name":"Trip planning","cards":[{"type":"today"},{"type":"quests","params":{"status":"open"}}]}`)
	if err != nil {
		t.Fatalf("board_compose: unexpected error: %v", err)
	}
	if !strings.Contains(out, "board raised") {
		t.Errorf("board_compose: result does not contain 'board raised': %q", out)
	}
	if !strings.Contains(out, "Trip planning") {
		t.Errorf("board_compose: result does not contain board name: %q", out)
	}
	if !strings.Contains(out, "/boards/") {
		t.Errorf("board_compose: result does not contain /boards/ URL: %q", out)
	}
	if !strings.Contains(out, "2 cards") {
		t.Errorf("board_compose: result does not report card count: %q", out)
	}

	// Verify record was actually created.
	recs, err := app.FindRecordsByFilter("boards", "name = 'Trip planning'", "", 0, 0, nil)
	if err != nil || len(recs) == 0 {
		t.Fatal("board_compose: board record not found in database")
	}
}

func TestBoardComposeWritesAuditLog(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	tool := findTool(t, ts, "board_compose")

	_, err := tool.Execute(context.Background(),
		`{"name":"Audit test","cards":[{"type":"heads"}]}`)
	if err != nil {
		t.Fatalf("board_compose audit: unexpected error: %v", err)
	}

	// Verify an audit_log entry was created with action "board_compose".
	recs, listErr := app.FindRecordsByFilter("audit_log", "action = 'board_compose'", "", 0, 0, nil)
	if listErr != nil {
		t.Fatalf("audit_log query: %v", listErr)
	}
	if len(recs) == 0 {
		t.Fatal("board_compose: no audit_log record written")
	}
}

func TestBoardComposeZeroCardsRejected(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	tool := findTool(t, ts, "board_compose")

	out, err := tool.Execute(context.Background(), `{"name":"Empty","cards":[]}`)
	if err != nil {
		t.Fatalf("board_compose 0 cards: unexpected Go error: %v", err)
	}
	if !strings.Contains(out, "at least 1") && !strings.Contains(out, "1 card") {
		t.Errorf("board_compose 0 cards: expected rejection message, got: %q", out)
	}
}

func TestBoardComposeNineCardsRejected(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	tool := findTool(t, ts, "board_compose")

	// Build 9-card JSON.
	nineCards := `[{"type":"today"},{"type":"today"},{"type":"today"},{"type":"today"},{"type":"today"},` +
		`{"type":"today"},{"type":"today"},{"type":"today"},{"type":"today"}]`
	out, err := tool.Execute(context.Background(), `{"name":"Too many","cards":`+nineCards+`}`)
	if err != nil {
		t.Fatalf("board_compose 9 cards: unexpected Go error: %v", err)
	}
	if !strings.Contains(out, "8") {
		t.Errorf("board_compose 9 cards: expected rejection with limit 8, got: %q", out)
	}
}

func TestBoardComposeNameTooLongRejected(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	tool := findTool(t, ts, "board_compose")

	longName := strings.Repeat("x", 81)
	out, err := tool.Execute(context.Background(),
		`{"name":"`+longName+`","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose long name: unexpected Go error: %v", err)
	}
	if !strings.Contains(out, "80") {
		t.Errorf("board_compose long name: expected rejection with limit 80, got: %q", out)
	}
}

func TestBoardComposeInvalidCardTypeRejected(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	tool := findTool(t, ts, "board_compose")

	out, err := tool.Execute(context.Background(),
		`{"name":"Bad cards","cards":[{"type":"nonexistent"}]}`)
	if err != nil {
		t.Fatalf("board_compose invalid card type: unexpected Go error: %v", err)
	}
	if strings.Contains(out, "board raised") {
		t.Error("board_compose invalid card type: board was created despite invalid type")
	}
}

func TestValidateCards(t *testing.T) {
	// Happy path.
	cs := []cards.Card{
		{Type: "today"},
		{Type: "quests", Params: map[string]string{"status": "open"}},
	}
	out, err := cards.ValidateCards(cs)
	if err != nil {
		t.Fatalf("ValidateCards happy: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("ValidateCards happy: len = %d, want 2", len(out))
	}

	// Invalid type errors.
	bad := []cards.Card{{Type: "nope"}}
	_, err = cards.ValidateCards(bad)
	if err == nil {
		t.Error("ValidateCards invalid type: want error, got nil")
	}
	if !strings.Contains(err.Error(), "card[0]") {
		t.Errorf("ValidateCards invalid type: error missing card index: %v", err)
	}
}

// -- board_add_card tool tests --

func TestBoardAddCardHappyPath(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	composeTool := findTool(t, ts, "board_compose")
	addTool := findTool(t, ts, "board_add_card")

	// Create a board first.
	_, err := composeTool.Execute(context.Background(),
		`{"name":"Trip","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose: %v", err)
	}

	// Add a card to it.
	out, err := addTool.Execute(context.Background(),
		`{"board":"Trip","type":"quests","params":{"status":"open"}}`)
	if err != nil {
		t.Fatalf("board_add_card: unexpected Go error: %v", err)
	}
	if !strings.Contains(out, "Quest log") {
		t.Errorf("board_add_card: result missing card label: %q", out)
	}
	if !strings.Contains(out, "Trip") {
		t.Errorf("board_add_card: result missing board name: %q", out)
	}
	if !strings.Contains(out, "/boards/") {
		t.Errorf("board_add_card: result missing /boards/ URL: %q", out)
	}

	// Verify the board now has 2 cards.
	recs, findErr := app.FindRecordsByFilter("boards", "name = 'Trip'", "", 0, 0, nil)
	if findErr != nil || len(recs) == 0 {
		t.Fatal("board_add_card: board record not found")
	}
	var cardList []cards.Card
	raw := recs[0].GetString("cards")
	if jsonErr := json.Unmarshal([]byte(raw), &cardList); jsonErr != nil {
		t.Fatalf("board_add_card: decoding cards JSON: %v", jsonErr)
	}
	if len(cardList) != 2 {
		t.Errorf("board_add_card: expected 2 cards after append, got %d", len(cardList))
	}
	if cardList[0].Type != "today" {
		t.Errorf("board_add_card: first card changed, want today, got %q", cardList[0].Type)
	}
	if cardList[1].Type != "quests" {
		t.Errorf("board_add_card: second card wrong, want quests, got %q", cardList[1].Type)
	}
}

func TestBoardAddCardWritesAuditLog(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	composeTool := findTool(t, ts, "board_compose")
	addTool := findTool(t, ts, "board_add_card")

	_, err := composeTool.Execute(context.Background(),
		`{"name":"Audit board","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose: %v", err)
	}

	_, err = addTool.Execute(context.Background(),
		`{"board":"Audit board","type":"heads"}`)
	if err != nil {
		t.Fatalf("board_add_card: unexpected Go error: %v", err)
	}

	recs, listErr := app.FindRecordsByFilter("audit_log", "action = 'board_add_card'", "", 0, 0, nil)
	if listErr != nil {
		t.Fatalf("audit_log query: %v", listErr)
	}
	if len(recs) == 0 {
		t.Fatal("board_add_card: no audit_log record written")
	}
}

func TestBoardAddCardResolveCaseInsensitive(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	composeTool := findTool(t, ts, "board_compose")
	addTool := findTool(t, ts, "board_add_card")

	_, err := composeTool.Execute(context.Background(),
		`{"name":"My Board","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose: %v", err)
	}

	// Use lowercased name — should still resolve.
	out, err := addTool.Execute(context.Background(),
		`{"board":"my board","type":"today"}`)
	if err != nil {
		t.Fatalf("board_add_card case-insensitive: unexpected Go error: %v", err)
	}
	if !strings.Contains(out, "My Board") {
		t.Errorf("board_add_card case-insensitive: board not found, got: %q", out)
	}
}

func TestBoardAddCardResolveBySubstring(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	composeTool := findTool(t, ts, "board_compose")
	addTool := findTool(t, ts, "board_add_card")

	_, err := composeTool.Execute(context.Background(),
		`{"name":"Holiday Travel 2026","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose: %v", err)
	}

	// Match by substring "travel".
	out, err := addTool.Execute(context.Background(),
		`{"board":"travel","type":"today"}`)
	if err != nil {
		t.Fatalf("board_add_card substring: unexpected Go error: %v", err)
	}
	if !strings.Contains(out, "Holiday Travel 2026") {
		t.Errorf("board_add_card substring: board not found, got: %q", out)
	}
}

func TestBoardAddCardAmbiguousNameReturnsText(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	composeTool := findTool(t, ts, "board_compose")
	addTool := findTool(t, ts, "board_add_card")

	// Create two boards whose names both contain "plan".
	_, err := composeTool.Execute(context.Background(),
		`{"name":"Trip plan","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose trip plan: %v", err)
	}
	_, err = composeTool.Execute(context.Background(),
		`{"name":"Gym plan","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose gym plan: %v", err)
	}

	out, err := addTool.Execute(context.Background(),
		`{"board":"plan","type":"today"}`)
	if err != nil {
		t.Fatalf("board_add_card ambiguous: unexpected Go error: %v", err)
	}
	// Must return a text error, not succeed.
	if !strings.Contains(out, "multiple") && !strings.Contains(out, "ambiguous") && !strings.Contains(out, "specific") {
		t.Errorf("board_add_card ambiguous: expected disambiguation message, got: %q", out)
	}
	if strings.Contains(out, "/boards/") && !strings.Contains(out, "multiple") {
		t.Errorf("board_add_card ambiguous: unexpectedly succeeded, got: %q", out)
	}
}

func TestBoardAddCardUnknownBoardReturnsText(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	composeTool := findTool(t, ts, "board_compose")
	addTool := findTool(t, ts, "board_add_card")

	// Create one board so we get a useful listing.
	_, err := composeTool.Execute(context.Background(),
		`{"name":"Existing board","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose: %v", err)
	}

	out, err := addTool.Execute(context.Background(),
		`{"board":"nonexistent xyz","type":"today"}`)
	if err != nil {
		t.Fatalf("board_add_card unknown: unexpected Go error: %v", err)
	}
	if !strings.Contains(out, "no board matches") {
		t.Errorf("board_add_card unknown: expected 'no board matches', got: %q", out)
	}
	// Must list existing boards.
	if !strings.Contains(out, "Existing board") {
		t.Errorf("board_add_card unknown: expected board listing, got: %q", out)
	}
}

func TestBoardAddCardInvalidCardTypeReturnsText(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	composeTool := findTool(t, ts, "board_compose")
	addTool := findTool(t, ts, "board_add_card")

	_, err := composeTool.Execute(context.Background(),
		`{"name":"My board","cards":[{"type":"today"}]}`)
	if err != nil {
		t.Fatalf("board_compose: %v", err)
	}

	out, err := addTool.Execute(context.Background(),
		`{"board":"My board","type":"nonexistent_type"}`)
	if err != nil {
		t.Fatalf("board_add_card invalid card: unexpected Go error: %v", err)
	}
	if strings.Contains(out, "/boards/") {
		t.Errorf("board_add_card invalid card: unexpectedly succeeded, got: %q", out)
	}
	if !strings.Contains(out, "invalid card") && !strings.Contains(out, "unknown card type") {
		t.Errorf("board_add_card invalid card: expected error text, got: %q", out)
	}
}

func TestBoardAddCardLayoutFieldsPreserved(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	addTool := findTool(t, ts, "board_add_card")

	// Seed a board directly with a card that has layout fields x and w.
	col, err := app.FindCollectionByNameOrId("boards")
	if err != nil {
		t.Fatalf("find boards collection: %v", err)
	}
	// Seed JSON includes a card with x=2,w=4 — plan 032 fields.
	seedJSON := `[{"type":"today","x":2,"w":4}]`
	rec := core.NewRecord(col)
	rec.Set("name", "Layout board")
	rec.Set("sort", 0)
	rec.Set("cards", seedJSON)
	if saveErr := app.Save(rec); saveErr != nil {
		t.Fatalf("save board: %v", saveErr)
	}

	// Add a new card to the board.
	out, err := addTool.Execute(context.Background(),
		`{"board":"Layout board","type":"heads"}`)
	if err != nil {
		t.Fatalf("board_add_card layout: unexpected Go error: %v", err)
	}
	if !strings.Contains(out, "/boards/") {
		t.Fatalf("board_add_card layout: unexpected failure: %q", out)
	}

	// Read back and verify x and w survive on the original card.
	recs, findErr := app.FindRecordsByFilter("boards", "name = 'Layout board'", "", 0, 0, nil)
	if findErr != nil || len(recs) == 0 {
		t.Fatal("board_add_card layout: board record not found")
	}
	var cardList []cards.Card
	raw := recs[0].GetString("cards")
	if jsonErr := json.Unmarshal([]byte(raw), &cardList); jsonErr != nil {
		t.Fatalf("board_add_card layout: decoding cards JSON: %v", jsonErr)
	}
	if len(cardList) != 2 {
		t.Fatalf("board_add_card layout: expected 2 cards, got %d", len(cardList))
	}
	// Original card must retain its layout fields.
	if cardList[0].X != 2 {
		t.Errorf("board_add_card layout: x field lost, want 2, got %d", cardList[0].X)
	}
	if cardList[0].W != 4 {
		t.Errorf("board_add_card layout: w field lost, want 4, got %d", cardList[0].W)
	}
	// New card is appended correctly.
	if cardList[1].Type != "heads" {
		t.Errorf("board_add_card layout: new card wrong type, want heads, got %q", cardList[1].Type)
	}
}
