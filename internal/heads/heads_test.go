package heads

import (
	"testing"

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
	if err := SetActive(app, "scholar"); err != nil {
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
	id, err := Create(app, "Scribe", "edits prose", "balaur-07", []string{"journal"})
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
	if err := SetActive(app, id); err != nil {
		t.Fatalf("SetActive(custom): %v", err)
	}
	if Active(app).ID != id {
		t.Errorf("active custom head id = %q, want %q", Active(app).ID, id)
	}
}

func TestDeletedActiveCustomFallsBackToMain(t *testing.T) {
	app := storetest.NewApp(t)
	id, _ := Create(app, "Temp", "", "", nil)
	_ = SetActive(app, id)
	if err := Delete(app, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := Active(app).ID; got != MainKey {
		t.Errorf("after deleting active custom, active = %q, want %q", got, MainKey)
	}
}
