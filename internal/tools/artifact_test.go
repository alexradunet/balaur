package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/storetest"
)

// -- MarkArtifact / ParseArtifact round-trip tests --

func TestMarkParseArtifactRoundTrip(t *testing.T) {
	cs := []cards.Card{
		{Type: "quests", Params: map[string]string{"status": "open"}},
		{Type: "today"},
	}
	marked := MarkArtifact(cs, "Your week", "showing the owner a cluster of 2 cards")

	title, got, rest, ok := ParseArtifact(marked)
	if !ok {
		t.Fatal("ParseArtifact: ok=false, want true")
	}
	if title != "Your week" {
		t.Errorf("title = %q, want %q", title, "Your week")
	}
	if len(got) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(got))
	}
	if got[0].Type != "quests" {
		t.Errorf("cards[0].Type = %q, want quests", got[0].Type)
	}
	if got[1].Type != "today" {
		t.Errorf("cards[1].Type = %q, want today", got[1].Type)
	}
	if rest != "showing the owner a cluster of 2 cards" {
		t.Errorf("rest = %q, want model text", rest)
	}
}

func TestMarkParseArtifactNoTitle(t *testing.T) {
	cs := []cards.Card{{Type: "today"}}
	marked := MarkArtifact(cs, "", "model text")
	title, got, _, ok := ParseArtifact(marked)
	if !ok {
		t.Fatal("ParseArtifact: ok=false, want true")
	}
	if title != "" {
		t.Errorf("title = %q, want empty", title)
	}
	if len(got) != 1 || got[0].Type != "today" {
		t.Errorf("unexpected cards: %v", got)
	}
}

func TestParseArtifactOnPlainText(t *testing.T) {
	_, _, _, ok := ParseArtifact("just a plain message")
	if ok {
		t.Error("ParseArtifact on plain text: ok=true, want false")
	}
}

func TestParseArtifactOnUICardMarked(t *testing.T) {
	marked := MarkUICard("today", map[string]string{}, "showing the Today card")
	_, _, _, ok := ParseArtifact(marked)
	if ok {
		t.Error("ParseArtifact on UICardMarker text: ok=true, want false")
	}
}

func TestParseArtifactOnChoicesMarked(t *testing.T) {
	marked := MarkChoices("Pick one", []Choice{{Label: "A"}, {Label: "B"}}, "offered")
	_, _, _, ok := ParseArtifact(marked)
	if ok {
		t.Error("ParseArtifact on ChoicesMarker text: ok=true, want false")
	}
}

func TestParseArtifactUnknownCardTypeReturnsFalse(t *testing.T) {
	// Craft the raw marker manually (bypassing MarkArtifact which would also
	// call ValidateCards) to prove ParseArtifact's reload-path guard works.
	raw := ArtifactMarker + `{"cards":[{"type":"nonexistent_card_type_xyz"}]}` + "\nmodel text"
	_, _, _, ok := ParseArtifact(raw)
	if ok {
		t.Error("ParseArtifact with unknown card type: ok=true, want false (ValidateCards should reject)")
	}
}

// -- show_cards tool tests --

func TestShowCardsHappyPath(t *testing.T) {
	app := storetest.NewApp(t)
	tool := showCardsTool(app)

	out, err := tool.Execute(context.Background(),
		`{"title":"My cluster","cards":[{"type":"today"},{"type":"quests","params":{"status":"open"}}]}`)
	if err != nil {
		t.Fatalf("show_cards: unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, ArtifactMarker) {
		t.Errorf("show_cards: result does not start with ArtifactMarker: %q", out[:min(len(out), 80)])
	}
	title, cs, _, ok := ParseArtifact(out)
	if !ok {
		t.Fatal("show_cards: ParseArtifact ok=false on happy-path result")
	}
	if title != "My cluster" {
		t.Errorf("title = %q, want My cluster", title)
	}
	if len(cs) != 2 {
		t.Errorf("cards len = %d, want 2", len(cs))
	}
}

func TestShowCardsZeroCardsRejected(t *testing.T) {
	app := storetest.NewApp(t)
	tool := showCardsTool(app)

	out, err := tool.Execute(context.Background(), `{"cards":[]}`)
	if err != nil {
		t.Fatalf("show_cards 0 cards: unexpected Go error: %v", err)
	}
	if strings.HasPrefix(out, ArtifactMarker) {
		t.Error("show_cards 0 cards: result starts with ArtifactMarker, must be plain error")
	}
	if !strings.Contains(out, "at least 1") {
		t.Errorf("show_cards 0 cards: expected rejection, got: %q", out)
	}
}

func TestShowCardsTooManyRejected(t *testing.T) {
	app := storetest.NewApp(t)
	tool := showCardsTool(app)

	// Build 9 cards JSON (exceeds showCardsMax=8).
	nineCards := strings.Repeat(`{"type":"today"},`, 8) + `{"type":"today"}`
	out, err := tool.Execute(context.Background(), `{"cards":[`+nineCards+`]}`)
	if err != nil {
		t.Fatalf("show_cards 9 cards: unexpected Go error: %v", err)
	}
	if strings.HasPrefix(out, ArtifactMarker) {
		t.Error("show_cards 9 cards: result starts with ArtifactMarker, must be plain error")
	}
	if !strings.Contains(out, "8") {
		t.Errorf("show_cards 9 cards: expected rejection with limit 8, got: %q", out)
	}
}

func TestShowCardsInvalidCardTypeRejected(t *testing.T) {
	app := storetest.NewApp(t)
	tool := showCardsTool(app)

	out, err := tool.Execute(context.Background(), `{"cards":[{"type":"nonexistent_xyz"}]}`)
	if err != nil {
		t.Fatalf("show_cards invalid type: unexpected Go error: %v", err)
	}
	if strings.HasPrefix(out, ArtifactMarker) {
		t.Error("show_cards invalid type: result starts with ArtifactMarker, must be plain error")
	}
}

func TestShowCardsInUITools(t *testing.T) {
	app := storetest.NewApp(t)
	ts := UITools(app)
	found := false
	for _, tool := range ts {
		if tool.Spec.Name == "show_cards" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("show_cards not found in UITools")
	}
}
