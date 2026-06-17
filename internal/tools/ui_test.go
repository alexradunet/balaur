package tools

import (
	"context"
	"strings"
	"testing"

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
