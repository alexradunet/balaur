package lifecards

import (
	"testing"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestBuildLifelogListsEveryKind seeds two trackers and asserts buildLifelog
// returns a LifeKindView for each with the correct Count — without depending on
// life.Series (the tile reads only the life.Kinds aggregate).
func TestBuildLifelogListsEveryKind(t *testing.T) {
	app := storetest.NewApp(t)
	// weight: two entries → Count 2. mood: one entry → Count 1.
	for _, o := range []life.LogOpts{
		{Kind: "weight", ValueNum: 82.5, Unit: "kg"},
		{Kind: "weight", ValueNum: 82.1, Unit: "kg"},
		{Kind: "mood", ValueNum: 7},
	} {
		if _, err := life.Log(app, o); err != nil {
			t.Fatalf("seed %s: %v", o.Kind, err)
		}
	}

	v := buildLifelog(app)

	got := map[string]int{}
	for _, k := range v.Kinds {
		got[k.Kind] = k.Count
	}
	if len(v.Kinds) != 2 {
		t.Fatalf("Kinds = %d (%v), want 2", len(v.Kinds), got)
	}
	if got["weight"] != 2 {
		t.Errorf("weight Count = %d, want 2", got["weight"])
	}
	if got["mood"] != 1 {
		t.Errorf("mood Count = %d, want 1", got["mood"])
	}
}
