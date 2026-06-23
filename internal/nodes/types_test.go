package nodes_test

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestTypeExists(t *testing.T) {
	app := storetest.NewApp(t)

	ok, err := nodes.TypeExists(app, "note")
	if err != nil {
		t.Fatalf("TypeExists(note): %v", err)
	}
	if !ok {
		t.Error("TypeExists(note) = false, want true")
	}

	ok, err = nodes.TypeExists(app, "nonsense")
	if err != nil {
		t.Fatalf("TypeExists(nonsense): %v", err)
	}
	if ok {
		t.Error("TypeExists(nonsense) = true, want false")
	}
}

func TestOwnerAuthoredTypes(t *testing.T) {
	app := storetest.NewApp(t)

	typs, err := nodes.OwnerAuthoredTypes(app)
	if err != nil {
		t.Fatalf("OwnerAuthoredTypes: %v", err)
	}

	hasNote, hasPerson, hasMemory, hasSkill := false, false, false, false
	for _, name := range typs {
		switch name {
		case "note":
			hasNote = true
		case "person":
			hasPerson = true
		case "memory":
			hasMemory = true
		case "skill":
			hasSkill = true
		}
	}
	if !hasNote {
		t.Error("OwnerAuthoredTypes: note missing")
	}
	if !hasPerson {
		t.Error("OwnerAuthoredTypes: person missing")
	}
	if hasMemory {
		t.Error("OwnerAuthoredTypes: memory should be excluded (born proposed)")
	}
	if hasSkill {
		t.Error("OwnerAuthoredTypes: skill should be excluded (born proposed)")
	}
}

func TestBornStatus(t *testing.T) {
	app := storetest.NewApp(t)

	s, err := nodes.BornStatus(app, "memory")
	if err != nil {
		t.Fatalf("BornStatus(memory): %v", err)
	}
	if s != "proposed" {
		t.Errorf("BornStatus(memory) = %q, want proposed", s)
	}

	s, err = nodes.BornStatus(app, "note")
	if err != nil {
		t.Fatalf("BornStatus(note): %v", err)
	}
	if s != "active" {
		t.Errorf("BornStatus(note) = %q, want active", s)
	}
}

func TestCreateUnknownTypeErrors(t *testing.T) {
	app := storetest.NewApp(t)

	_, err := nodes.Create(app, "nonsense", "Test", "", nodes.StatusActive, nil)
	if err == nil {
		t.Error("Create with unknown type should have returned an error")
	}
}

func TestCreateKnownTypeSucceeds(t *testing.T) {
	app := storetest.NewApp(t)

	rec, err := nodes.Create(app, "note", "My note", "body text", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create(note): %v", err)
	}
	if rec.GetString("type") != "note" {
		t.Errorf("type = %q, want note", rec.GetString("type"))
	}
}

// TestCreateNewRegistryTypeSucceeds is the headline extensibility test:
// adding a new node_types row makes Create accept that type without any code change.
func TestCreateNewRegistryTypeSucceeds(t *testing.T) {
	app := storetest.NewApp(t)

	// Add a brand-new type row.
	col, err := app.FindCollectionByNameOrId("node_types")
	if err != nil {
		t.Fatalf("finding node_types: %v", err)
	}
	row := core.NewRecord(col)
	row.Set("name", "recipe")
	row.Set("label", "Recipe")
	row.Set("born_status", "active")
	row.Set("system", false)
	if err := app.Save(row); err != nil {
		t.Fatalf("saving recipe type: %v", err)
	}

	// Now Create should accept it.
	rec, err := nodes.Create(app, "recipe", "Banana bread", "Mix and bake.", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create(recipe): %v", err)
	}
	if rec.GetString("type") != "recipe" {
		t.Errorf("type = %q, want recipe", rec.GetString("type"))
	}
}
