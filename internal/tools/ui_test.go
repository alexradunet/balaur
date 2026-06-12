package tools

import (
	"context"
	"strings"
	"testing"

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
