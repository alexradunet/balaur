package web

import (
	"testing"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/store"
)

// TestLifeEntryDrop drives POST /ui/life/entry/{id}/drop through the real router
// and confirms it deletes one owner-logged measure and audits the drop — the
// per-row drop affordance, end to end (route mounted → handler → life.Drop).
func TestLifeEntryDrop(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()

	rec, err := life.Log(app, life.LogOpts{Kind: "gratitude", Text: "the morning was quiet"})
	if err != nil {
		t.Fatalf("log entry: %v", err)
	}

	w := serveReviewRoute(t, app, "/ui/life/entry/"+rec.Id+"/drop")
	if w.Code != 200 {
		t.Fatalf("drop route status = %d, want 200", w.Code)
	}

	if _, err := app.FindRecordById("nodes", rec.Id); err == nil {
		t.Error("entry should have been deleted but still exists")
	}

	rows, err := store.ListAudit(app, "life.drop", "", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(rows) == 0 {
		t.Error("expected a life.drop audit row after the drop")
	}
}
