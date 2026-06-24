package knowledgecards_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
)

func renderKnowledgeFocus(t *testing.T, v knowledgecards.KnowledgeFocusView) string {
	t.Helper()
	var b strings.Builder
	if err := knowledgecards.KnowledgeFocus(v).Render(&b); err != nil {
		t.Fatalf("KnowledgeFocus.Render: %v", err)
	}
	return b.String()
}

func renderKnowledgeGrid(t *testing.T, active []g.Node, kind, query string) string {
	t.Helper()
	var b strings.Builder
	if err := knowledgecards.KnowledgeGrid(active, kind, query).Render(&b); err != nil {
		t.Fatalf("KnowledgeGrid.Render: %v", err)
	}
	return b.String()
}

// TestKnowledgeFocusMemoryContract guards the class/id/Datastar contract the
// served CSS and SSE handlers depend on for a category card (mode=active).
// Asserts the escaped form for Datastar attributes (gomponents HTML-escapes
// ' → &#39; and & → &amp;).
func TestKnowledgeFocusMemoryContract(t *testing.T) {
	active := []g.Node{
		knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{
			ID: "a1", Status: "active", Category: "fact", Title: "Active memory", Importance: 3,
		}),
	}
	archived := []g.Node{
		knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{
			ID: "ar1", Status: "archived", Title: "Archived memory",
		}),
	}
	got := renderKnowledgeFocus(t, knowledgecards.KnowledgeFocusView{
		Kind:     "memories",
		Title:    "Facts",
		Category: "fact",
		Active:   active,
		Archived: archived,
	})

	for _, want := range []string{
		// Active section
		`class="k-heading"`,
		`id="k-active-grid"`,
		// Controls — q signal only (no category signal)
		`data-signals:q="&#39;&#39;"`,
		// Search input with fixed category baked into @get — & → &amp; and ' → &#39;
		`data-on:input__debounce.250ms="@get(&#39;/ui/knowledge/memories/grid?q=&#39;+encodeURIComponent($q)+&#39;&amp;category=fact&#39;)"`,
		// Archived section
		`class="k-heading k-heading-muted"`,
		`class="k-grid k-grid-muted"`,
		`id="kcard-ar1"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("KnowledgeFocus (memory category) missing %q in:\n%s", want, got)
		}
	}

	// No in-panel tab strip (plan 110): sub-views are reached via /-command
	// palette entries, not a tab strip. The category signal stays absent too.
	if strings.Contains(got, `class="k-tabs"`) {
		t.Errorf("KnowledgeFocus (memory category) must not contain tab strip:\n%s", got)
	}
	if strings.Contains(got, `data-signals:category`) {
		t.Errorf("KnowledgeFocus (memory category) must not contain data-signals:category:\n%s", got)
	}

	// No proposed section (Proposed is empty).
	if strings.Contains(got, `k-heading-proposed`) {
		t.Errorf("KnowledgeFocus: empty Proposed must not render proposed section:\n%s", got)
	}
}

// TestKnowledgeFocusSkillsNoCategories: skills focus has no category tabs.
func TestKnowledgeFocusSkillsNoCategories(t *testing.T) {
	active := []g.Node{
		knowledgecards.SkillRecordCard(knowledgecards.SkillRecord{
			ID: "s1", Status: "active", Name: "Summarise",
		}),
	}
	got := renderKnowledgeFocus(t, knowledgecards.KnowledgeFocusView{
		Kind:   "skills",
		Title:  "Skills",
		Active: active,
	})

	// Must have the active grid
	if !strings.Contains(got, `id="k-active-grid"`) {
		t.Errorf("skills focus missing #k-active-grid:\n%s", got)
	}
	// Skills grid @get uses skills endpoint
	if !strings.Contains(got, `/ui/knowledge/skills/grid`) {
		t.Errorf("skills focus missing skills/grid endpoint:\n%s", got)
	}
	// Must NOT have category tabs (Kind=="skills" gate — memories get the strip, skills do not)
	if strings.Contains(got, `class="k-tabs"`) {
		t.Errorf("skills focus must not have category tab strip:\n%s", got)
	}
	// No proposed or archived sections
	if strings.Contains(got, `k-heading-proposed`) {
		t.Errorf("skills focus with no proposed must not show proposed section:\n%s", got)
	}
	if strings.Contains(got, `k-grid-muted`) {
		t.Errorf("skills focus with no archived must not show archived section:\n%s", got)
	}
}

// TestKnowledgeFocusNoProposedNoSection: proposed section is omitted when empty.
func TestKnowledgeFocusNoProposedNoSection(t *testing.T) {
	got := renderKnowledgeFocus(t, knowledgecards.KnowledgeFocusView{
		Kind:     "memories",
		Title:    "Facts",
		Category: "fact",
	})
	if strings.Contains(got, "k-heading-proposed") {
		t.Errorf("empty proposed must not render proposed section:\n%s", got)
	}
	if strings.Contains(got, "Awaiting your word") {
		t.Errorf("empty proposed must not render 'Awaiting your word':\n%s", got)
	}
}

// TestKnowledgeGridWithActive: active cards render inside k-grid.
func TestKnowledgeGridWithActive(t *testing.T) {
	nodes := []g.Node{
		knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{
			ID: "x1", Status: "active", Title: "Test active",
		}),
	}
	got := renderKnowledgeGrid(t, nodes, "memories", "")
	if !strings.Contains(got, `class="k-grid"`) {
		t.Errorf("active grid missing k-grid:\n%s", got)
	}
	if !strings.Contains(got, `id="kcard-x1"`) {
		t.Errorf("active grid missing kcard-x1:\n%s", got)
	}
}

// TestKnowledgeGridEmptyNoQuery: empty grid without a query shows invitation copy.
func TestKnowledgeGridEmptyNoQuery(t *testing.T) {
	got := renderKnowledgeGrid(t, nil, "memories", "")
	if !strings.Contains(got, `class="k-empty"`) {
		t.Errorf("empty grid missing k-empty:\n%s", got)
	}
	if !strings.Contains(got, "Nothing here yet.") {
		t.Errorf("empty grid missing invitation copy:\n%s", got)
	}
}

// TestKnowledgeGridEmptyWithQuery: empty grid with a query shows "nothing matches".
func TestKnowledgeGridEmptyWithQuery(t *testing.T) {
	got := renderKnowledgeGrid(t, nil, "memories", "dark mode")
	if !strings.Contains(got, `class="k-empty"`) {
		t.Errorf("query-empty grid missing k-empty:\n%s", got)
	}
	if !strings.Contains(got, "Nothing matches") {
		t.Errorf("query-empty grid missing 'Nothing matches':\n%s", got)
	}
}
