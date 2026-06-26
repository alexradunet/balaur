package taskcards

import "testing"

// TestQuestGroupName mirrors the deleted TestQuestGroup in internal/web/tasks_test.go,
// keeping the rhythm-bucketing logic covered after web.questGroup was removed.
func TestQuestGroupName(t *testing.T) {
	cases := []struct {
		name   string
		recur  string
		hasDue bool
		want   string
	}{
		{name: "daily rule", recur: "daily", hasDue: false, want: "Dailies"},
		{name: "daily rule with due", recur: "daily", hasDue: true, want: "Dailies"},
		{name: "every:1d rule", recur: "every:1d", hasDue: false, want: "Dailies"},
		{name: "every:3d rule", recur: "every:3d", hasDue: false, want: "Rituals"},
		{name: "weekly:mon rule", recur: "weekly:mon", hasDue: false, want: "Rituals"},
		{name: "monthly:1 rule", recur: "monthly:1", hasDue: false, want: "Rituals"},
		{name: "bad rule with due", recur: "bogus-rule", hasDue: true, want: "Quests"},
		{name: "bad rule no due", recur: "bogus-rule", hasDue: false, want: "Side quests"},
		{name: "empty with due", recur: "", hasDue: true, want: "Quests"},
		{name: "empty no due", recur: "", hasDue: false, want: "Side quests"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := questGroupName(tc.recur, tc.hasDue)
			if got != tc.want {
				t.Errorf("questGroupName(%q, %v) = %q, want %q", tc.recur, tc.hasDue, got, tc.want)
			}
		})
	}
}
