package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestMarkParseChoicesRoundTrip(t *testing.T) {
	choices := []Choice{
		{Label: "Yes, add it now", Hint: "add recurring task"},
		{Label: "Remind me later"},
		{Label: "Skip it"},
	}
	marked := MarkChoices("Your word", choices, "offered choices: 1) Yes, add it now 2) Remind me later 3) Skip it")

	prompt, got, modelText, ok := ParseChoices(marked)
	if !ok {
		t.Fatal("ParseChoices: ok=false, want true")
	}
	if prompt != "Your word" {
		t.Errorf("prompt = %q, want %q", prompt, "Your word")
	}
	if len(got) != len(choices) {
		t.Fatalf("choices len = %d, want %d", len(got), len(choices))
	}
	for i, c := range choices {
		if got[i].Label != c.Label {
			t.Errorf("choice[%d].Label = %q, want %q", i, got[i].Label, c.Label)
		}
		if got[i].Hint != c.Hint {
			t.Errorf("choice[%d].Hint = %q, want %q", i, got[i].Hint, c.Hint)
		}
	}
	if !strings.Contains(modelText, "offered choices:") {
		t.Errorf("modelText = %q, want to contain 'offered choices:'", modelText)
	}
}

func TestParseChoicesOnPlainText(t *testing.T) {
	_, _, _, ok := ParseChoices("plain text with no marker")
	if ok {
		t.Error("ParseChoices on plain text: ok=true, want false")
	}
}

func TestParseChoicesOnProposalMarked(t *testing.T) {
	proposal := MarkProposal("memories", "abc123", "memory proposal")
	_, _, _, ok := ParseChoices(proposal)
	if ok {
		t.Error("ParseChoices on proposal-marked text: ok=true, want false")
	}
}

func TestOfferChoicesExecuteRejectsBounds(t *testing.T) {
	app := storetest.NewApp(t)
	tool := offerChoicesTool(app)

	// Too few choices (1).
	out, err := tool.Execute(context.Background(), `{"choices":[{"label":"only one"}]}`)
	if err != nil {
		t.Fatalf("Execute 1 choice: unexpected error: %v", err)
	}
	if !strings.Contains(out, "at least 2") {
		t.Errorf("Execute 1 choice: got %q, want 'at least 2'", out)
	}

	// Too many choices (6).
	sixChoices := `{"choices":[{"label":"a"},{"label":"b"},{"label":"c"},{"label":"d"},{"label":"e"},{"label":"f"}]}`
	out, err = tool.Execute(context.Background(), sixChoices)
	if err != nil {
		t.Fatalf("Execute 6 choices: unexpected error: %v", err)
	}
	if !strings.Contains(out, "at most 5") {
		t.Errorf("Execute 6 choices: got %q, want 'at most 5'", out)
	}
}

func TestOfferChoicesExecuteOutput(t *testing.T) {
	app := storetest.NewApp(t)
	tool := offerChoicesTool(app)

	args := `{"prompt":"Pick a path","choices":[{"label":"Go north"},{"label":"Go south","hint":"leads to the tavern"}]}`
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.HasPrefix(out, ChoicesMarker) {
		t.Errorf("Execute: result does not start with ChoicesMarker: %q", out[:min(len(out), 40)])
	}

	prompt, choices, modelText, ok := ParseChoices(out)
	if !ok {
		t.Fatal("Execute result: ParseChoices ok=false, want true")
	}
	if prompt != "Pick a path" {
		t.Errorf("prompt = %q, want %q", prompt, "Pick a path")
	}
	if len(choices) != 2 {
		t.Fatalf("choices len = %d, want 2", len(choices))
	}
	if choices[0].Label != "Go north" {
		t.Errorf("choices[0].Label = %q, want %q", choices[0].Label, "Go north")
	}
	if choices[1].Label != "Go south" {
		t.Errorf("choices[1].Label = %q, want %q", choices[1].Label, "Go south")
	}
	if choices[1].Hint != "leads to the tavern" {
		t.Errorf("choices[1].Hint = %q, want %q", choices[1].Hint, "leads to the tavern")
	}
	if !strings.Contains(modelText, "offered choices:") {
		t.Errorf("modelText %q missing 'offered choices:'", modelText)
	}
	if !strings.Contains(modelText, "Go north") || !strings.Contains(modelText, "Go south") {
		t.Errorf("modelText %q does not enumerate both labels", modelText)
	}
}

func TestOfferChoicesDefaultPrompt(t *testing.T) {
	app := storetest.NewApp(t)
	tool := offerChoicesTool(app)

	// No prompt field — should default to "Your word".
	args := `{"choices":[{"label":"Option A"},{"label":"Option B"}]}`
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute default prompt: %v", err)
	}
	prompt, _, _, ok := ParseChoices(out)
	if !ok {
		t.Fatal("ParseChoices: ok=false, want true")
	}
	if prompt != "Your word" {
		t.Errorf("default prompt = %q, want %q", prompt, "Your word")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
