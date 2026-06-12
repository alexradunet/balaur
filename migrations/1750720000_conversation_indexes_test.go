package migrations_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestConversationIndexesExist(t *testing.T) {
	app := storetest.NewApp(t)
	for _, idx := range []string{
		"idx_conversations_open_branch_head",
		"idx_conversations_open_master",
		"idx_conversations_head",
		"idx_tasks_done_at",
	} {
		var name string
		err := app.DB().
			NewQuery("SELECT name FROM sqlite_master WHERE type='index' AND name={:n}").
			Bind(map[string]any{"n": idx}).Row(&name)
		if err != nil || name != idx {
			t.Errorf("index %s missing (err=%v)", idx, err)
		}
	}
}
