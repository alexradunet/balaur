package migrations_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestHotIndexesExist(t *testing.T) {
	app := storetest.NewApp(t)
	for _, idx := range []string{
		"idx_messages_origin_created",
		"idx_messages_conv_created",
		"idx_tasks_nudge",
		"idx_audit_actor",
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
