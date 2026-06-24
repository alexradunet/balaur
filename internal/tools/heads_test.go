package tools

import (
	"context"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestHeadSwitchTool(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := headSwitchTool(app).Execute(context.Background(), `{"id":"scholar"}`); err != nil {
		t.Fatalf("head_switch: %v", err)
	}
	if got := heads.Active(app); got.ID != "scholar" {
		t.Errorf("active head = %q, want scholar", got.ID)
	}
	if countModelAudit(t, app, "head.switch") != 1 {
		t.Error("head.switch should audit actor=model")
	}
	if _, err := headSwitchTool(app).Execute(context.Background(), `{"id":"nope"}`); err == nil {
		t.Error("switching to an unknown head should fail")
	}
}

func TestHeadCreateAndDeleteTool(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := headCreateTool(app).Execute(context.Background(),
		`{"name":"Scout","purpose":"recon","groups":["memory"]}`); err != nil {
		t.Fatalf("head_create: %v", err)
	}
	if countModelAudit(t, app, "head.create") != 1 {
		t.Error("head.create should audit actor=model")
	}
	recs, err := app.FindRecordsByFilter("heads", "name = 'Scout'", "", 0, 0, nil)
	if err != nil || len(recs) != 1 {
		t.Fatalf("expected 1 custom head Scout, got %d (err %v)", len(recs), err)
	}
	id := recs[0].Id

	// A built-in head cannot be deleted.
	if _, err := headDeleteTool(app).Execute(context.Background(), `{"id":"balaur"}`); err == nil {
		t.Error("deleting a built-in head should fail")
	}
	// The custom head deletes and audits.
	if _, err := headDeleteTool(app).Execute(context.Background(), `{"id":"`+id+`"}`); err != nil {
		t.Fatalf("head_delete: %v", err)
	}
	if countModelAudit(t, app, "head.delete") != 1 {
		t.Error("head.delete should audit actor=model")
	}
}

func TestHeadCreateRejectsBadGroup(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := headCreateTool(app).Execute(context.Background(), `{"name":"X","groups":["bogus"]}`); err == nil {
		t.Error("an unknown capability group should fail")
	}
}

// countModelAudit counts allowed audit rows for an action attributed to the
// model — shared by the heads and profile tool tests.
func countModelAudit(t *testing.T, app core.App, action string) int {
	t.Helper()
	recs, err := app.FindRecordsByFilter("audit_log",
		"action = {:a} && actor = 'model' && allowed = true", "", 0, 0,
		dbx.Params{"a": action})
	if err != nil {
		t.Fatalf("querying audit_log: %v", err)
	}
	return len(recs)
}
