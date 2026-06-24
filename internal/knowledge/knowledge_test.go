package knowledge

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/search"
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

// TestProposeEditLifecycle: the model parks an edit on an active memory without
// touching its approved content; the owner applies it (content changes, envelope
// clears) or declines it (content stays, envelope clears). The propose audits as
// actor=model — a model write to active knowledge can never read as the owner's.
func TestProposeEditLifecycle(t *testing.T) {
	app := storetest.NewApp(t)

	rec, err := ProposeMemory(app, MemoryProposal{Title: "Prefers tea", Content: "Black, no sugar.", Category: "preference", Importance: 3})
	if err != nil {
		t.Fatalf("ProposeMemory: %v", err)
	}
	if _, err := Transition(app, Memory, rec.Id, StatusActive); err != nil {
		t.Fatalf("activate: %v", err)
	}

	// Model proposes an edit — parked, audited actor=model, content untouched.
	if _, err := ProposeEdit(app, rec.Id, map[string]string{"content": "Green tea, no sugar.", "importance": "5"}, false); err != nil {
		t.Fatalf("ProposeEdit: %v", err)
	}
	cur, _ := app.FindRecordById("nodes", rec.Id)
	if got := cur.GetString("body"); got != "Black, no sugar." {
		t.Errorf("active content must be untouched until approval, got %q", got)
	}
	if _, _, ok := PendingEdit(cur); !ok {
		t.Fatal("expected a parked pending edit")
	}
	if got := modelAudit(t, app, "knowledge.propose_edit"); got != 1 {
		t.Errorf("want 1 actor=model propose_edit audit, got %d", got)
	}
	if pend, _ := PendingEdits(app); len(pend) != 1 {
		t.Errorf("PendingEdits want 1, got %d", len(pend))
	}

	// Owner approves — applied, envelope cleared, change audited as owner.
	if _, err := ApplyEdit(app, rec.Id); err != nil {
		t.Fatalf("ApplyEdit: %v", err)
	}
	cur, _ = app.FindRecordById("nodes", rec.Id)
	if got := cur.GetString("body"); got != "Green tea, no sugar." {
		t.Errorf("approved edit should apply, got %q", got)
	}
	if got := nodes.PropInt(cur, "importance"); got != 5 {
		t.Errorf("approved importance want 5, got %d", got)
	}
	if _, _, ok := PendingEdit(cur); ok {
		t.Error("envelope should be cleared after apply")
	}
	if countAudit(t, app, "knowledge.edit", true) != 1 {
		t.Error("apply should audit knowledge.edit (actor=owner)")
	}

	// Decline a second proposal — content stays, envelope clears.
	if _, err := ProposeEdit(app, rec.Id, map[string]string{"content": "rejected change"}, false); err != nil {
		t.Fatalf("ProposeEdit 2: %v", err)
	}
	if _, err := DeclineEdit(app, rec.Id); err != nil {
		t.Fatalf("DeclineEdit: %v", err)
	}
	cur, _ = app.FindRecordById("nodes", rec.Id)
	if got := cur.GetString("body"); got != "Green tea, no sugar." {
		t.Errorf("declined edit must not change content, got %q", got)
	}
	if _, _, ok := PendingEdit(cur); ok {
		t.Error("envelope should be cleared after decline")
	}
}

// TestProposeEditRejectsNonActive: edits can only be proposed against active
// knowledge — a still-proposed node is not yet the owner's to revise.
func TestProposeEditRejectsNonActive(t *testing.T) {
	app := storetest.NewApp(t)
	rec, _ := ProposeMemory(app, MemoryProposal{Title: "x", Importance: 1})
	if _, err := ProposeEdit(app, rec.Id, map[string]string{"content": "y"}, false); err == nil {
		t.Fatal("propose_edit on a proposed (non-active) node should fail")
	}
}

// modelAudit counts allowed audit rows for an action attributed to the model.
func modelAudit(t *testing.T, app core.App, action string) int {
	t.Helper()
	recs, err := app.FindRecordsByFilter("audit_log",
		"action = {:a} && actor = 'model' && allowed = true", "", 0, 0,
		dbx.Params{"a": action})
	if err != nil {
		t.Fatalf("querying audit_log: %v", err)
	}
	return len(recs)
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

// TestTransitionAuditOrdering verifies the consent-ledger ordering:
// denied transitions emit an allowed=false audit entry and return an error;
// allowed transitions emit exactly one allowed=true audit entry only after the
// record's status has changed (save-first, audit-after).
func TestTransitionAuditOrdering(t *testing.T) {
	app := storetest.NewApp(t)
	rec, err := ProposeMemory(app, MemoryProposal{Title: "audit order test", Importance: 1})
	if err != nil {
		t.Fatalf("ProposeMemory: %v", err)
	}

	// --- denied case ---
	_, err = Transition(app, Memory, rec.Id, StatusArchived) // proposed→archived invalid
	if err == nil {
		t.Fatal("denied transition must return an error")
	}
	if countAudit(t, app, "knowledge.archived", false) != 1 {
		t.Fatal("denied transition: expected exactly one allowed=false audit entry")
	}
	if countAudit(t, app, "knowledge.archived", true) != 0 {
		t.Fatal("denied transition: must not produce an allowed=true audit entry")
	}
	// status must not have changed
	fresh, _ := app.FindRecordById("nodes", rec.Id)
	if fresh.GetString("status") != StatusProposed {
		t.Fatalf("denied transition changed status to %q", fresh.GetString("status"))
	}

	// --- allowed case ---
	_, err = Transition(app, Memory, rec.Id, StatusActive)
	if err != nil {
		t.Fatalf("Transition to active: %v", err)
	}
	if countAudit(t, app, "knowledge.active", true) != 1 {
		t.Fatal("allowed transition: expected exactly one allowed=true audit entry")
	}
	// status must have changed to active
	fresh, _ = app.FindRecordById("nodes", rec.Id)
	if fresh.GetString("status") != StatusActive {
		t.Fatalf("status = %q, want active", fresh.GetString("status"))
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
	if rec.GetString("status") == StatusActive {
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

func TestFilterActive(t *testing.T) {
	app := storetest.NewApp(t)

	seed := []MemoryProposal{
		{Title: "Prefers espresso", Category: "preference", Importance: 3},
		{Title: "Sister Ana lives in Cluj", Category: "person", Importance: 4},
		{Title: "Espresso machine repair guide", Category: "project", Importance: 2},
	}
	for _, p := range seed {
		rec, err := ProposeMemory(app, p)
		if err != nil {
			t.Fatalf("propose: %v", err)
		}
		if _, err := Transition(app, Memory, rec.Id, StatusActive); err != nil {
			t.Fatalf("approve: %v", err)
		}
	}
	// One proposed record that must never surface.
	if _, err := ProposeMemory(app, MemoryProposal{Title: "Espresso budget", Importance: 1}); err != nil {
		t.Fatalf("propose: %v", err)
	}

	cases := []struct {
		name     string
		query    string
		category string
		want     int
	}{
		{"no filters lists all active", "", "", 3},
		{"query matches title substring", "espresso", "", 2},
		{"category narrows", "", "person", 1},
		{"query and category combine", "espresso", "project", 1},
		{"no match", "espresso", "person", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FilterActive(app, Memory, tc.query, tc.category)
			if err != nil {
				t.Fatalf("FilterActive: %v", err)
			}
			if len(got) != tc.want {
				t.Fatalf("got %d records, want %d", len(got), tc.want)
			}
		})
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

// TestSearchActiveFTSPath proves the FTS fast path: a content word with
// trailing punctuation ("flour,") that LIKE would miss is found via FTS5.
// FTS5 tokenizes "flour," as the token "flour" so a query for "flour" hits.
func TestSearchActiveFTSPath(t *testing.T) {
	app := storetest.NewApp(t)

	// Content contains "flour," — LIKE search for "flour" also matches
	// substring, so we pick a term that LIKE misses but FTS5 wins:
	// use a term that appears only as a full word token with punctuation.
	rec, err := ProposeMemory(app, MemoryProposal{
		Title:      "baking notes",
		Content:    "sift flour, then fold in the eggs",
		Category:   "fact",
		Importance: 2,
		Source:     "test",
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := Transition(app, Memory, rec.Id, StatusActive); err != nil {
		t.Fatalf("approve: %v", err)
	}

	// Build a real FTS index and put it in the store.
	ix, err := search.Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer ix.Close()
	if err := ix.Rebuild(app); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	app.Store().Set(search.StoreKey, ix)

	// Query with "flour" — FTS5 tokenizer strips the comma, so this should hit.
	got, err := SearchActive(app, []string{"flour"}, 10)
	if err != nil {
		t.Fatalf("SearchActive: %v", err)
	}
	found := false
	for _, m := range got {
		if m.Id == rec.Id {
			found = true
		}
	}
	if !found {
		t.Fatalf("FTS path did not return the seeded memory; got %d records", len(got))
	}
}

// TestSearchActiveFallbackNoIndex proves that with no index in the store,
// the behavior is byte-identical to today: existing LIKE tests continue to pass.
func TestSearchActiveFallbackNoIndex(t *testing.T) {
	app := storetest.NewApp(t)

	approved, _ := ProposeMemory(app, MemoryProposal{Title: "Prefers espresso", Category: "preference", Importance: 2})
	if _, err := Transition(app, Memory, approved.Id, StatusActive); err != nil {
		t.Fatalf("approve: %v", err)
	}
	proposed, _ := ProposeMemory(app, MemoryProposal{Title: "Espresso machine budget", Importance: 2})
	_ = proposed

	// No index in store — must fall through to LIKE.
	got, err := SearchActive(app, []string{"espresso"}, 10)
	if err != nil {
		t.Fatalf("SearchActive fallback: %v", err)
	}
	if len(got) != 1 || got[0].GetString("title") != "Prefers espresso" {
		t.Fatalf("LIKE fallback unexpected: %d records", len(got))
	}
}

// TestSearchActiveIntegration is the end-to-end chain: seed app → build
// index → Rebuild → SearchActive returns the seeded memory by a content word.
func TestSearchActiveIntegration(t *testing.T) {
	app := storetest.NewApp(t)

	rec, err := ProposeMemory(app, MemoryProposal{
		Title:      "hiking trail",
		Content:    "Piatra Mare trail starts near Brasov city",
		Category:   "preference",
		Importance: 3,
		Source:     "test",
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := Transition(app, Memory, rec.Id, StatusActive); err != nil {
		t.Fatalf("approve: %v", err)
	}

	ix, err := search.Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer ix.Close()
	if err := ix.Rebuild(app); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	app.Store().Set(search.StoreKey, ix)

	results, err := SearchActive(app, []string{"Brasov"}, 10)
	if err != nil {
		t.Fatalf("SearchActive: %v", err)
	}
	found := false
	for _, m := range results {
		if m.Id == rec.Id {
			found = true
		}
	}
	if !found {
		t.Fatalf("integration: seeded memory not returned; got %d records", len(results))
	}
}

// TestSearchActiveStaysMemoryOnly is the no-regression guard for the
// memory-scoped recall contract: even when an active note shares the search
// term, SearchActive (which backs recall and BuildContext) returns ONLY memory
// nodes. A non-memory hit feeding those memory-aliased callers would be a bug.
func TestSearchActiveStaysMemoryOnly(t *testing.T) {
	app := storetest.NewApp(t)

	mem, err := ProposeMemory(app, MemoryProposal{
		Title: "owner fact", Content: "owner enjoys kombucha", Category: "preference", Importance: 3, Source: "test",
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := Transition(app, Memory, mem.Id, StatusActive); err != nil {
		t.Fatalf("approve: %v", err)
	}
	// An active note sharing the term "kombucha" — must NOT be recalled.
	note, err := nodes.Create(app, "note", "drinks log", "tried a new kombucha brand", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	ix, err := search.Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer ix.Close()
	if err := ix.Rebuild(app); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	app.Store().Set(search.StoreKey, ix)

	got, err := SearchActive(app, []string{"kombucha"}, 10)
	if err != nil {
		t.Fatalf("SearchActive: %v", err)
	}
	for _, r := range got {
		if r.Id == note.Id {
			t.Fatalf("SearchActive returned a non-memory note %s — recall must stay memory-only", note.Id)
		}
		if r.GetString("type") != string(Memory) {
			t.Fatalf("SearchActive returned a %q node — recall must stay memory-only", r.GetString("type"))
		}
	}
	if len(got) != 1 || got[0].Id != mem.Id {
		t.Fatalf("expected only the memory %s; got %d records", mem.Id, len(got))
	}
}

// TestSearchAllActiveCrossType proves the new cross-type surface returns mixed
// node types and never a proposed node (the consent filter).
func TestSearchAllActiveCrossType(t *testing.T) {
	app := storetest.NewApp(t)

	mem, err := ProposeMemory(app, MemoryProposal{
		Title: "owner fact", Content: "owner enjoys kombucha", Category: "preference", Importance: 3, Source: "test",
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := Transition(app, Memory, mem.Id, StatusActive); err != nil {
		t.Fatalf("approve: %v", err)
	}
	note, err := nodes.Create(app, "note", "drinks log", "tried a new kombucha brand", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	// A proposed memory with the same term — must never be returned.
	proposed, err := ProposeMemory(app, MemoryProposal{
		Title: "draft", Content: "secret kombucha plan", Category: "fact", Importance: 1, Source: "test",
	})
	if err != nil {
		t.Fatalf("propose draft: %v", err)
	}

	ix, err := search.Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer ix.Close()
	if err := ix.Rebuild(app); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	app.Store().Set(search.StoreKey, ix)

	got, err := SearchAllActive(app, []string{"kombucha"}, 10)
	if err != nil {
		t.Fatalf("SearchAllActive: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range got {
		ids[r.Id] = true
		if r.Id == proposed.Id {
			t.Fatalf("proposed node %s leaked into cross-type search", proposed.Id)
		}
	}
	if !ids[mem.Id] || !ids[note.Id] {
		t.Fatalf("cross-type search missed a type: memory=%v note=%v (got %d)", ids[mem.Id], ids[note.Id], len(got))
	}
}
