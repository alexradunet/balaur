package heads

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestActiveDefaultsToMain(t *testing.T) {
	app := storetest.NewApp(t)
	if got := Active(app).ID; got != MainKey {
		t.Errorf("default active head = %q, want %q", got, MainKey)
	}
}

func TestSetAndResolveBuiltin(t *testing.T) {
	app := storetest.NewApp(t)
	if err := SetActive(app, "owner", "scholar"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	h := Active(app)
	if h.ID != "scholar" || h.Name != "Scholar" {
		t.Fatalf("active = %+v, want scholar", h)
	}
	if len(h.Groups) == 0 {
		t.Error("scholar should carry a non-empty tool-group filter")
	}
}

func TestCustomHeadRoundTripAndActive(t *testing.T) {
	app := storetest.NewApp(t)
	id, err := Create(app, "owner", "Scribe", "edits prose", "balaur-07", []string{"journal"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Appears in the roster after the built-ins.
	roster := List(app)
	if len(roster) != len(Builtins())+1 {
		t.Fatalf("roster len = %d, want %d", len(roster), len(Builtins())+1)
	}
	last := roster[len(roster)-1]
	if last.ID != id || last.Name != "Scribe" || last.BuiltIn {
		t.Fatalf("custom head = %+v", last)
	}
	if len(last.Groups) != 1 || last.Groups[0] != "journal" {
		t.Fatalf("custom groups = %v, want [journal]", last.Groups)
	}
	// Becomes active, then resolves.
	if err := SetActive(app, "owner", id); err != nil {
		t.Fatalf("SetActive(custom): %v", err)
	}
	if Active(app).ID != id {
		t.Errorf("active custom head id = %q, want %q", Active(app).ID, id)
	}
}

func TestDeletedActiveCustomFallsBackToMain(t *testing.T) {
	app := storetest.NewApp(t)
	id, _ := Create(app, "owner", "Temp", "", "", nil)
	_ = SetActive(app, "owner", id)
	if err := Delete(app, "owner", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := Active(app).ID; got != MainKey {
		t.Errorf("after deleting active custom, active = %q, want %q", got, MainKey)
	}
}

func TestSetActiveAudits(t *testing.T) {
	app := storetest.NewApp(t)
	if err := SetActive(app, "owner", "scholar"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if n := countAudit(t, app, "head.switch", "owner"); n != 1 {
		t.Errorf("head.switch audit rows = %d, want 1", n)
	}
}

func TestCreateAudits(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Create(app, "owner", "Gardener", "tends the garden", "", nil); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := countAudit(t, app, "head.create", "owner"); n != 1 {
		t.Errorf("head.create audit rows = %d, want 1", n)
	}
}

func TestDeleteAudits(t *testing.T) {
	app := storetest.NewApp(t)
	id, err := Create(app, "owner", "Temp", "", "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := Delete(app, "owner", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n := countAudit(t, app, "head.delete", "owner"); n != 1 {
		t.Errorf("head.delete audit rows = %d, want 1", n)
	}
}

func countAudit(t *testing.T, app core.App, action, actor string) int {
	t.Helper()
	recs, err := app.FindRecordsByFilter("audit_log",
		"action = {:a} && actor = {:actor} && allowed = true", "", 0, 0,
		dbx.Params{"a": action, "actor": actor})
	if err != nil {
		t.Fatalf("querying audit_log: %v", err)
	}
	return len(recs)
}
