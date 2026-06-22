package verify

import (
	"testing"

	"github.com/alexradunet/balaur/internal/llm"
)

// The positive cases are lifted from a real live-test transcript in which
// the model claimed reminders without calling any tool.
func TestClaimsCapture(t *testing.T) {
	claims := []string{
		"Got it — stretch reminder in 2 minutes: 11:34. Setting it now.",
		"Same upcoming nudge (11:38) that I already set with your earlier note.",
		"Soothe your muscles after cleaning up. Reminder due 11:34, with the 15-min dishes task.",
		"I'll remind you at six.",
		"Done — I've added it to your list.",
		"The task is now set for tomorrow morning.",
		"Just logged that for you.",
	}
	for _, s := range claims {
		if !ClaimsCapture(s) {
			t.Errorf("should read as a claim: %q", s)
		}
	}

	notClaims := []string{
		"Want me to set a reminder?",
		"Shall I add that to your tasks?",
		"You could set a reminder on your phone for this.",
		"Would you like me to log that weight?",
		"A reminder might help here — say the word.",
		"What time should the reminder be set for?",
		"The notary is waiting on you — a good moment to call.",
		"",
	}
	for _, s := range notClaims {
		if ClaimsCapture(s) {
			t.Errorf("should NOT read as a claim: %q", s)
		}
	}
}

func TestCaptureSucceeded(t *testing.T) {
	mk := func(tool, result string) []llm.Message {
		return []llm.Message{
			{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: "c1", Name: tool}}},
			{Role: "tool", ToolCallID: "c1", Content: result},
			{Role: "assistant", Content: "done"},
		}
	}

	if !CaptureSucceeded(mk("task_add", "Task saved: ...")) {
		t.Error("successful task_add must count")
	}
	if CaptureSucceeded(mk("task_add", "error: tasks: title is required")) {
		t.Error("a failed capture tool must not count")
	}
	if CaptureSucceeded(mk("recall", "some memories")) {
		t.Error("non-capture tools must not count")
	}
	if CaptureSucceeded([]llm.Message{{Role: "assistant", Content: "Setting it now."}}) {
		t.Error("no tools at all must not count")
	}
	if !CaptureSucceeded(mk("journal_write", "Kept in the journal under Thursday.")) {
		t.Error("journal_write must count")
	}
}

func TestSplitSentences(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{
			in:   "I set it. Done! Really? Yes.",
			want: []string{"I set it", " Done", " Really", " Yes"},
		},
		{
			in:   "Just one sentence",
			want: []string{"Just one sentence"},
		},
		{
			in:   "",
			want: nil,
		},
		{
			in:   "Line one\nLine two",
			want: []string{"Line one", "Line two"},
		},
	}
	for _, tc := range cases {
		got := splitSentences(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("splitSentences(%q) = %q, want %q", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitSentences(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestLastAssistantText(t *testing.T) {
	turn := []llm.Message{
		{Role: "assistant", Content: "first"},
		{Role: "tool", Content: "result"},
		{Role: "assistant", Content: ""},
		{Role: "assistant", Content: "final words"},
	}
	if got := LastAssistantText(turn); got != "final words" {
		t.Errorf("last text = %q", got)
	}
	if got := LastAssistantText(nil); got != "" {
		t.Errorf("empty turn = %q", got)
	}
}
