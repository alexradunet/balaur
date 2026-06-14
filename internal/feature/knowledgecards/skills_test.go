package knowledgecards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
)

// ---------------------------------------------------------------------------
// SkillsCard (summary tile)
// ---------------------------------------------------------------------------

func TestSkillsCard_WithRows(t *testing.T) {
	rows := []knowledgecards.SkillRow{
		{Name: "Weekly review", Description: "End-of-week reflection", Enabled: true},
		{Name: "Deep work", Description: "Distraction-free focus block", Enabled: false},
	}
	var b strings.Builder
	if err := knowledgecards.SkillsCard(rows, "limit: 6").Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		`id="ucard-skills"`,
		`class="kcard ucard ucard-skills"`,
		`/static/icons/key.png`,
		`Skills`,
		`limit: 6`,
		`/focus/skills`,
		`Weekly review`,
		`enabled`, // enabled badge for first row
		`End-of-week reflection`,
		`Deep work`,
		`Distraction-free focus block`,
		`all skills →`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("SkillsCard missing %q\nHTML:\n%s", want, out)
		}
	}
}

func TestSkillsCard_EnabledBadgeOnlyForEnabled(t *testing.T) {
	rows := []knowledgecards.SkillRow{
		{Name: "Enabled skill", Enabled: true},
		{Name: "Disabled skill", Enabled: false},
	}
	var b strings.Builder
	_ = knowledgecards.SkillsCard(rows, "").Render(&b)
	out := b.String()

	// Count occurrences of the enabled badge
	count := strings.Count(out, `class="kcard-on"`)
	if count != 1 {
		t.Errorf("expected exactly 1 enabled badge, got %d\nHTML:\n%s", count, out)
	}
}

func TestSkillsCard_EmptyState(t *testing.T) {
	var b strings.Builder
	_ = knowledgecards.SkillsCard(nil, "").Render(&b)
	out := b.String()

	if !strings.Contains(out, "No active skills yet.") {
		t.Errorf("expected empty state, got:\n%s", out)
	}
	if strings.Contains(out, `class="ucard-list"`) {
		t.Errorf("list element should be absent for empty state")
	}
}

func TestSkillsCard_ParamLineOmittedWhenEmpty(t *testing.T) {
	var b strings.Builder
	_ = knowledgecards.SkillsCard([]knowledgecards.SkillRow{{Name: "x"}}, "").Render(&b)
	out := b.String()

	if strings.Contains(out, `class="kcard-meta"`) {
		t.Errorf("param-line span should be absent when paramLine is empty:\n%s", out)
	}
}

func TestSkillsCard_Footer(t *testing.T) {
	var b strings.Builder
	_ = knowledgecards.SkillsCard(nil, "").Render(&b)
	out := b.String()

	if !strings.Contains(out, `href="/focus/skills"`) {
		t.Errorf("footer link missing:\n%s", out)
	}
	if !strings.Contains(out, "all skills →") {
		t.Errorf("footer text missing:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// SkillRecordCard
// ---------------------------------------------------------------------------

func TestSkillRecordCard_ProposedActions(t *testing.T) {
	r := knowledgecards.SkillRecord{
		ID:          "sk1",
		Status:      "proposed",
		Name:        "Pomodoro",
		Description: "Time-boxing technique",
		WhenToUse:   "deep work sessions",
		Content:     "25 min on, 5 min off",
		Enabled:     false,
		UseCount:    0,
	}
	var b strings.Builder
	if err := knowledgecards.SkillRecordCard(r).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		`class="kcard kcard-proposed"`,
		`id="kcard-sk1"`,
		`⌥ skill`,
		`Pomodoro`,
		`Time-boxing technique`,
		`use: deep work sessions`,
		`25 min on, 5 min off`,
		// Procedure details
		`<details`,
		`<summary>Procedure</summary>`,
		`<pre class="kcard-pre">`,
		// Edit form
		`<summary>Edit</summary>`,
		`/ui/knowledge/skills/sk1/edit`,
		`name="name"`,
		`name="description"`,
		`name="content"`,
		`name="when_to_use"`,
		`rows="6"`,
		// Proposed footer actions
		`value="active"`,
		`Approve`,
		`value="rejected"`,
		`Dismiss`,
		`/ui/knowledge/skills/sk1/transition`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("SkillRecordCard(proposed) missing %q\nHTML:\n%s", want, out)
		}
	}
	// proposed should NOT show Archive or Restore
	if strings.Contains(out, "Archive") {
		t.Errorf("proposed card should not show Archive:\n%s", out)
	}
	if strings.Contains(out, "Restore") {
		t.Errorf("proposed card should not show Restore:\n%s", out)
	}
}

func TestSkillRecordCard_EnabledBadge(t *testing.T) {
	r := knowledgecards.SkillRecord{
		ID: "sk2", Status: "active", Name: "GTD", Enabled: true,
	}
	var b strings.Builder
	_ = knowledgecards.SkillRecordCard(r).Render(&b)
	out := b.String()

	if !strings.Contains(out, `class="kcard-on"`) {
		t.Errorf("enabled badge missing:\n%s", out)
	}
}

func TestSkillRecordCard_NoEnabledBadgeWhenDisabled(t *testing.T) {
	r := knowledgecards.SkillRecord{
		ID: "sk3", Status: "proposed", Name: "GTD", Enabled: false,
	}
	var b strings.Builder
	_ = knowledgecards.SkillRecordCard(r).Render(&b)
	out := b.String()

	if strings.Contains(out, `class="kcard-on"`) {
		t.Errorf("enabled badge should be absent:\n%s", out)
	}
}

func TestSkillRecordCard_ActiveActions(t *testing.T) {
	r := knowledgecards.SkillRecord{
		ID:       "sk4",
		Status:   "active",
		Name:     "Inbox zero",
		UseCount: 7,
		Enabled:  true,
	}
	var b strings.Builder
	if err := knowledgecards.SkillRecordCard(r).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		`class="kcard kcard-active"`,
		`id="kcard-sk4"`,
		`value="archived"`,
		`Archive`,
		`used ×7`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("SkillRecordCard(active) missing %q\nHTML:\n%s", want, out)
		}
	}
	// active should NOT show Approve/Dismiss/Restore
	for _, absent := range []string{"Approve", "Dismiss", "Restore"} {
		if strings.Contains(out, absent) {
			t.Errorf("active card should not show %q:\n%s", absent, out)
		}
	}
}

func TestSkillRecordCard_ActiveZeroUseCount(t *testing.T) {
	r := knowledgecards.SkillRecord{
		ID: "sk5", Status: "active", Name: "Zero", UseCount: 0,
	}
	var b strings.Builder
	_ = knowledgecards.SkillRecordCard(r).Render(&b)
	out := b.String()

	// use_count=0 should not render the "used ×N" span (mirrors template {{with .GetInt "use_count"}})
	if strings.Contains(out, "used ×") {
		t.Errorf("zero use_count should not render 'used ×':\n%s", out)
	}
}

func TestSkillRecordCard_ArchivedActions(t *testing.T) {
	r := knowledgecards.SkillRecord{
		ID:     "sk6",
		Status: "archived",
		Name:   "Old technique",
	}
	var b strings.Builder
	if err := knowledgecards.SkillRecordCard(r).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		`class="kcard kcard-archived"`,
		`id="kcard-sk6"`,
		`value="active"`,
		`Restore`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("SkillRecordCard(archived) missing %q\nHTML:\n%s", want, out)
		}
	}
	for _, absent := range []string{"Approve", "Dismiss", "Archive"} {
		if strings.Contains(out, absent) {
			t.Errorf("archived card should not show %q:\n%s", absent, out)
		}
	}
}

func TestSkillRecordCard_OptionalFieldsOmitted(t *testing.T) {
	r := knowledgecards.SkillRecord{
		ID:     "sk7",
		Status: "proposed",
		Name:   "Bare skill",
		// Description, WhenToUse, Content all empty
	}
	var b strings.Builder
	_ = knowledgecards.SkillRecordCard(r).Render(&b)
	out := b.String()

	if strings.Contains(out, `class="kcard-body"`) {
		t.Errorf("kcard-body should be absent when description is empty:\n%s", out)
	}
	if strings.Contains(out, `class="kcard-when"`) {
		t.Errorf("kcard-when should be absent when when_to_use is empty:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// SkillsManageCard
// ---------------------------------------------------------------------------

func TestSkillsManageCard_WithProposedAndActive(t *testing.T) {
	proposed := []knowledgecards.SkillRecord{
		{ID: "p1", Status: "proposed", Name: "New skill"},
	}
	active := []knowledgecards.SkillRecord{
		{ID: "a1", Status: "active", Name: "Running skill", Enabled: true},
	}
	var b strings.Builder
	if err := knowledgecards.SkillsManageCard(proposed, active).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		`id="ucard-skills-manage"`,
		`class="kcard ucard ucard-manage ucard-skills-manage"`,
		`/static/icons/key.png`,
		`Skills`,
		`href="/focus/skills"`,
		`manage all →`,
		`Awaiting your word`,
		`kcard-proposed`,
		`id="kcard-p1"`,
		`New skill`,
		`Active`,
		`kcard-active`,
		`id="kcard-a1"`,
		`Running skill`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("SkillsManageCard missing %q\nHTML:\n%s", want, out)
		}
	}
}

func TestSkillsManageCard_EmptyState(t *testing.T) {
	var b strings.Builder
	_ = knowledgecards.SkillsManageCard(nil, nil).Render(&b)
	out := b.String()

	if !strings.Contains(out, "Nothing yet") {
		t.Errorf("expected empty state:\n%s", out)
	}
	if !strings.Contains(out, "Skills appears as Balaur proposes") {
		t.Errorf("expected empty state description:\n%s", out)
	}
}

func TestSkillsManageCard_OnlyProposed(t *testing.T) {
	proposed := []knowledgecards.SkillRecord{
		{ID: "p2", Status: "proposed", Name: "Only proposed"},
	}
	var b strings.Builder
	_ = knowledgecards.SkillsManageCard(proposed, nil).Render(&b)
	out := b.String()

	if !strings.Contains(out, "Awaiting your word") {
		t.Errorf("proposed section missing:\n%s", out)
	}
	// "Active" heading should be absent when no active records
	if strings.Contains(out, ">Active<") {
		t.Errorf("Active section should be absent:\n%s", out)
	}
}

func TestSkillsManageCard_OnlyActive(t *testing.T) {
	active := []knowledgecards.SkillRecord{
		{ID: "a2", Status: "active", Name: "Only active"},
	}
	var b strings.Builder
	_ = knowledgecards.SkillsManageCard(nil, active).Render(&b)
	out := b.String()

	if !strings.Contains(out, "Active") {
		t.Errorf("active section missing:\n%s", out)
	}
	// "Awaiting your word" should be absent
	if strings.Contains(out, "Awaiting your word") {
		t.Errorf("proposed section should be absent:\n%s", out)
	}
}
