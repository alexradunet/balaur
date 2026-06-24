package web

import (
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	iknowledge "github.com/alexradunet/balaur/internal/knowledge"
)

// TestReviewEditApproveApplies drives the /ui/review/edit/{id}/approve route
// through the real router and confirms it applies a parked model-proposed edit
// to active knowledge and clears the envelope — the queue's approve action,
// end to end (route mounted → handler → domain → re-render).
func TestReviewEditApproveApplies(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()

	rec, err := iknowledge.ProposeMemory(app, iknowledge.MemoryProposal{
		Title: "Prefers tea", Content: "Black, no sugar.", Importance: 3,
	})
	if err != nil {
		t.Fatalf("propose memory: %v", err)
	}
	if _, err := iknowledge.Transition(app, iknowledge.Memory, rec.Id, iknowledge.StatusActive); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := iknowledge.ProposeEdit(app, rec.Id, map[string]string{"content": "Green tea, no sugar."}, false); err != nil {
		t.Fatalf("propose edit: %v", err)
	}

	w := serveReviewRoute(t, app, "/ui/review/edit/"+rec.Id+"/approve")
	if w.Code != 200 {
		t.Fatalf("approve route status = %d, want 200", w.Code)
	}

	cur, err := app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload node: %v", err)
	}
	if got := cur.GetString("body"); got != "Green tea, no sugar." {
		t.Errorf("approved edit not applied: body = %q", got)
	}
	if _, _, ok := iknowledge.PendingEdit(cur); ok {
		t.Error("pending-edit envelope should be cleared after approval")
	}
}

// serveReviewRoute POSTs to a route through the fully-mounted router and returns
// the recorder. The default httptest host ("example.com") is allow-listed by
// newWebApp, so the DNS-rebinding guard passes.
func serveReviewRoute(t *testing.T, app core.App, url string) *httptest.ResponseRecorder {
	t.Helper()
	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	se := &core.ServeEvent{App: app, Router: baseRouter}
	if err := app.OnServe().Trigger(se, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatalf("OnServe trigger: %v", err)
	}
	mux, err := se.Router.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	req := httptest.NewRequest("POST", url, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}
