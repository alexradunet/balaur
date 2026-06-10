package knowledge

import (
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

func countAudit(t *testing.T, app core.App, action string, allowed bool) int {
	t.Helper()
	recs, err := app.FindRecordsByFilter("audit_log",
		"action = {:action} && allowed = {:allowed}", "", 0, 0,
		dbx.Params{"action": action, "allowed": allowed})
	if err != nil {
		t.Fatalf("querying audit_log: %v", err)
	}
	return len(recs)
}

func TestProposeAndApproveMemory(t *testing.T) {
	app := storetest.NewApp(t)

	rec, err := ProposeMemory(app, MemoryProposal{
		Title:      "Owner lives in Brașov",
		Content:    "Home base is Brașov, Romania.",
		Category:   "fact",
		Importance: 9, // clamped to 5
		Source:     "chat",
	})
	if err != nil {
		t.Fatalf("ProposeMemory: %v", err)
	}
	if got := rec.GetString("status"); got != StatusProposed {
		t.Fatalf("status = %q, want proposed", got)
	}
	if got := rec.GetInt("importance"); got != 5 {
		t.Fatalf("importance = %d, want clamped 5", got)
	}

	// A proposed memory must NOT appear in any injection path.
	if ms, _ := UpfrontMemories(app, 10); len(ms) != 0 {
		t.Fatalf("proposed memory leaked into upfront context: %d", len(ms))
	}

	if _, err := Transition(app, Memory, rec.Id, StatusActive); err != nil {
		t.Fatalf("approve: %v", err)
	}
	ms, _ := UpfrontMemories(app, 10)
	if len(ms) != 1 {
		t.Fatalf("approved high-importance memory missing from upfront: %d", len(ms))
	}
	if countAudit(t, app, "knowledge.active", true) != 1 {
		t.Fatal("approval not audited")
	}
}

func TestLifecycleRejectsInvalidTransitions(t *testing.T) {
	app := storetest.NewApp(t)
	rec, err := ProposeMemory(app, MemoryProposal{Title: "x", Importance: 1})
	if err != nil {
		t.Fatalf("ProposeMemory: %v", err)
	}

	// proposed → archived is not a thing: dismiss or approve first.
	if _, err := Transition(app, Memory, rec.Id, StatusArchived); err == nil {
		t.Fatal("proposed → archived should fail")
	}
	if countAudit(t, app, "knowledge.archived", false) != 1 {
		t.Fatal("denied transition not audited")
	}

	// proposed → rejected works; rejected is terminal.
	if _, err := Transition(app, Memory, rec.Id, StatusRejected); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if _, err := Transition(app, Memory, rec.Id, StatusActive); err == nil {
		t.Fatal("rejected → active should fail (terminal)")
	}
}

func TestSearchActiveOnlyFindsActive(t *testing.T) {
	app := storetest.NewApp(t)

	approved, _ := ProposeMemory(app, MemoryProposal{Title: "Prefers espresso", Category: "preference", Importance: 2})
	if _, err := Transition(app, Memory, approved.Id, StatusActive); err != nil {
		t.Fatalf("approve: %v", err)
	}
	// This one stays proposed — must never surface.
	if _, err := ProposeMemory(app, MemoryProposal{Title: "Espresso machine budget", Importance: 2}); err != nil {
		t.Fatalf("propose: %v", err)
	}

	got, err := SearchActive(app, []string{"espresso"}, 10)
	if err != nil {
		t.Fatalf("SearchActive: %v", err)
	}
	if len(got) != 1 || got[0].GetString("title") != "Prefers espresso" {
		t.Fatalf("unexpected search result: %d records", len(got))
	}
}

func TestSkillLifecycleAndLoad(t *testing.T) {
	app := storetest.NewApp(t)

	rec, err := ProposeSkill(app, SkillProposal{
		Name:        "weekly-review",
		Description: "Run the owner's weekly review ritual",
		Content:     "1. Ask about the week\n2. Summarise\n3. Save highlights",
		WhenToUse:   "When the owner asks for a weekly review.",
	})
	if err != nil {
		t.Fatalf("ProposeSkill: %v", err)
	}
	if rec.GetBool("enabled") {
		t.Fatal("proposed skill must not be enabled")
	}

	// Not loadable while proposed.
	if _, err := LoadSkill(app, "weekly-review"); err == nil {
		t.Fatal("proposed skill should not load")
	}

	if _, err := Transition(app, Skill, rec.Id, StatusActive); err != nil {
		t.Fatalf("approve skill: %v", err)
	}
	loaded, err := LoadSkill(app, "weekly-review")
	if err != nil {
		t.Fatalf("LoadSkill after approve: %v", err)
	}
	if loaded.GetInt("use_count") != 1 {
		t.Fatalf("use_count = %d, want 1", loaded.GetInt("use_count"))
	}

	// Archive disables it again.
	if _, err := Transition(app, Skill, rec.Id, StatusArchived); err != nil {
		t.Fatalf("archive skill: %v", err)
	}
	if _, err := LoadSkill(app, "weekly-review"); err == nil {
		t.Fatal("archived skill should not load")
	}
}

func TestBuildContext(t *testing.T) {
	app := storetest.NewApp(t)

	important, _ := ProposeMemory(app, MemoryProposal{
		Title: "Allergic to peanuts", Category: "fact", Importance: 5})
	Transition(app, Memory, important.Id, StatusActive)

	niche, _ := ProposeMemory(app, MemoryProposal{
		Title: "Favourite hiking trail is Piatra Mare", Category: "preference", Importance: 2})
	Transition(app, Memory, niche.Id, StatusActive)

	skill, _ := ProposeSkill(app, SkillProposal{
		Name: "trip-plan", WhenToUse: "When planning a trip."})
	Transition(app, Skill, skill.Id, StatusActive)

	ctx, used := BuildContext(app, "let's go hiking this weekend")

	if !strings.Contains(ctx, "Allergic to peanuts") {
		t.Fatal("tier-1 memory missing from context")
	}
	if !strings.Contains(ctx, "Piatra Mare") {
		t.Fatal("tier-2 recall missing: 'hiking' should match the trail memory")
	}
	if !strings.Contains(ctx, "trip-plan") {
		t.Fatal("skills index missing from context")
	}
	if strings.Contains(ctx, "## What you remember") == false {
		t.Fatal("context headers missing")
	}
	if len(used) != 2 {
		t.Fatalf("used = %d records, want 2", len(used))
	}

	// Low-importance memory must NOT be upfront when nothing matches it.
	ctx2, _ := BuildContext(app, "what is the weather")
	if strings.Contains(ctx2, "Piatra Mare") {
		t.Fatal("niche memory leaked into context without a match")
	}
	if !strings.Contains(ctx2, "Allergic to peanuts") {
		t.Fatal("tier-1 memory should always be present")
	}
}

func TestUpdateFieldsWhitelist(t *testing.T) {
	app := storetest.NewApp(t)
	rec, _ := ProposeMemory(app, MemoryProposal{Title: "draft", Importance: 1})

	updated, err := UpdateFields(app, Memory, rec.Id, map[string]string{
		"title":      "edited title",
		"importance": "4",
		"status":     "active", // NOT writable through this path
	})
	if err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	if updated.GetString("title") != "edited title" {
		t.Fatal("title edit not applied")
	}
	if updated.GetInt("importance") != 4 {
		t.Fatal("importance edit not applied")
	}
	if updated.GetString("status") != StatusProposed {
		t.Fatal("status must not be writable via UpdateFields — consent boundary")
	}
}
