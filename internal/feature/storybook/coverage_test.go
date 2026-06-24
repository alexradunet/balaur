package storybook

import (
	"testing"

	// Blank-import every feature package so their init() card registrations run
	// — ui.RegisteredCardTypes is then the full set, the same as at app startup.
	_ "github.com/alexradunet/balaur/internal/feature/all"
	"github.com/alexradunet/balaur/internal/ui"
)

// cardStoryID maps each registered card type (ui.RegisterCard) to the storybook
// story id that documents it. Card-type names and story ids differ (e.g.
// "quests" -> "questsfocus"), so the map is explicit. A registered type with NO
// entry fails the test below — mirroring how tours_test.go guards the .tours
// artifact: registering a new card forces adding a story (or a tracked gap).
// An empty id ("") marks a KNOWN, deliberate gap — a card surface that still
// lacks a dedicated story — NOT a story id.
var cardStoryID = map[string]string{
	"calendar": "calendar",
	"graph":    "graphcard",
	"habits":   "habits",
	"heads":    "heads",
	"lifelog":  "lifelogfocus",
	"lines":    "lines",
	"measure":  "measure",
	"network":  "networkcard",
	"note":     "notecard",
	"period":   "periodfocus",
	"quests":   "questsfocus",
	"related":  "relatedcard",
	"review":   "reviewqueue",
	"settings": "settingsfocus",
	"tasks":    "taskcard",
	"timeline": "timeline",
	"today":    "today",
	"day":      "day",
	"memory":   "memory",
	"skills":   "skills",
}

// TestEveryRegisteredCardHasAStory keeps the storybook honest as the declared
// source of truth: every card type a feature registers is documented here, and
// every non-gap entry resolves to a real story.
func TestEveryRegisteredCardHasAStory(t *testing.T) {
	for _, typ := range ui.RegisteredCardTypes() {
		id, mapped := cardStoryID[typ]
		if !mapped {
			t.Errorf("registered card type %q has no cardStoryID entry — add its story id (or \"\" + a comment for a tracked gap)", typ)
			continue
		}
		if id == "" {
			continue // tracked gap, documented above
		}
		if _, ok := Lookup(id); !ok {
			t.Errorf("card type %q maps to story %q, but no such story is registered", typ, id)
		}
	}
}
