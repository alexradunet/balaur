package knowledgecards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
)

// renderMemory is a helper that renders MemoryCard and returns HTML.
func renderMemory(t *testing.T, v knowledgecards.MemoryView) string {
	t.Helper()
	var b strings.Builder
	if err := knowledgecards.MemoryCard(v).Render(&b); err != nil {
		t.Fatalf("MemoryCard.Render: %v", err)
	}
	return b.String()
}

// renderMemoryRecord is a helper that renders MemoryRecordCard and returns HTML.
func renderMemoryRecord(t *testing.T, r knowledgecards.MemoryRecord) string {
	t.Helper()
	var b strings.Builder
	if err := knowledgecards.MemoryRecordCard(r).Render(&b); err != nil {
		t.Fatalf("MemoryRecordCard.Render: %v", err)
	}
	return b.String()
}

// renderMemoryManage is a helper that renders MemoryManageCard and returns HTML.
func renderMemoryManage(t *testing.T, v knowledgecards.MemoryManageView) string {
	t.Helper()
	var b strings.Builder
	if err := knowledgecards.MemoryManageCard(v).Render(&b); err != nil {
		t.Fatalf("MemoryManageCard.Render: %v", err)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// MemoryCard (summary view)
// ---------------------------------------------------------------------------

func TestMemoryCardStructure(t *testing.T) {
	v := knowledgecards.MemoryView{
		ParamLine: "limit: 6",
		Rows: []knowledgecards.MemoryRow{
			{Title: "Prefers dark mode", Importance: 3},
		},
	}
	out := renderMemory(t, v)

	for _, want := range []string{
		`class="kcard ucard ucard-memory"`,
		`id="ucard-memory"`,
		`class="kcard-kind"`,
		`src="/static/icons/tome.png"`,
		`alt=""`,
		"Memory",
		`class="kcard-meta"`,
		"limit: 6",
		`class="ucard-list"`,
		`class="ucard-row"`,
		// Row title is a Datastar @get (not a bare href) so the click morphs the
		// panel instead of full-navigating to the SSE-only /ui/show route.
		`<a href="/ui/show/memory" data-on:click__prevent="@get(&#39;/ui/show/memory&#39;)">`,
		"Prefers dark mode",
		`class="kcard-pips"`,
		`title="importance 3/5"`,
		`class="pip pip-on"`,
		`class="kcard-actions"`,
		`href="/ui/show/memory"`,
		"all memories →",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("MemoryCard missing %q in:\n%s", want, out)
		}
	}
}

func TestMemoryCardEmptyState(t *testing.T) {
	out := renderMemory(t, knowledgecards.MemoryView{})
	if !strings.Contains(out, "No active memories yet.") {
		t.Errorf("expected empty state, got:\n%s", out)
	}
	if strings.Contains(out, "ucard-row") {
		t.Errorf("empty view must not render rows:\n%s", out)
	}
}

func TestMemoryCardNoParamLineOmitted(t *testing.T) {
	v := knowledgecards.MemoryView{
		// No ParamLine set — the kcard-head must not contain a kcard-meta span.
		Rows: []knowledgecards.MemoryRow{
			{Title: "Test memory", Importance: 1},
		},
	}
	out := renderMemory(t, v)
	// The kcard-head section comes before the list; it must not contain a
	// kcard-meta span when ParamLine is empty. We verify by checking that the
	// header ends immediately after the kind span (no second child span).
	// Since "kcard-kind" closes before "kcard-meta" would appear, we can check
	// that the header element itself does not contain kcard-meta text by looking
	// at what's between <header and </header>.
	headerStart := strings.Index(out, "<header")
	headerEnd := strings.Index(out, "</header>")
	if headerStart == -1 || headerEnd == -1 {
		t.Fatalf("header tags not found in:\n%s", out)
	}
	headerContent := out[headerStart : headerEnd+len("</header>")]
	if strings.Contains(headerContent, `class="kcard-meta"`) {
		t.Errorf("empty ParamLine should not render kcard-meta in header:\n%s", headerContent)
	}
}

func TestMemoryCardPipCount(t *testing.T) {
	// Importance 2 → 2 pip-on, 3 plain pip (5 total)
	v := knowledgecards.MemoryView{
		Rows: []knowledgecards.MemoryRow{
			{Title: "Low importance", Importance: 2},
		},
	}
	out := renderMemory(t, v)
	// 5 pips total: 2 on + 3 off
	onCount := strings.Count(out, `class="pip pip-on"`)
	offCount := strings.Count(out, `class="pip"`)
	if onCount != 2 {
		t.Errorf("expected 2 pip-on, got %d:\n%s", onCount, out)
	}
	if offCount != 3 {
		t.Errorf("expected 3 plain pip, got %d:\n%s", offCount, out)
	}
}

func TestMemoryCardMultipleRows(t *testing.T) {
	v := knowledgecards.MemoryView{
		ParamLine: "limit: 6 · q: dark",
		Rows: []knowledgecards.MemoryRow{
			{Title: "Dark mode", Importance: 5},
			{Title: "Alex lives in London", Importance: 4},
		},
	}
	out := renderMemory(t, v)
	if !strings.Contains(out, "Dark mode") {
		t.Errorf("first row missing:\n%s", out)
	}
	if !strings.Contains(out, "Alex lives in London") {
		t.Errorf("second row missing:\n%s", out)
	}
	if !strings.Contains(out, "limit: 6 · q: dark") {
		t.Errorf("paramLine missing:\n%s", out)
	}
}

func TestMemoryCardFooter(t *testing.T) {
	out := renderMemory(t, knowledgecards.MemoryView{})
	if !strings.Contains(out, `<footer class="kcard-actions">`) {
		t.Errorf("footer missing:\n%s", out)
	}
	if !strings.Contains(out, `href="/ui/show/memory"`) {
		t.Errorf("footer link missing:\n%s", out)
	}
	if !strings.Contains(out, "all memories →") {
		t.Errorf("footer text missing:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// MemoryRecordCard (individual memory record)
// ---------------------------------------------------------------------------

func TestMemoryRecordCardIDs(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:         "abc123",
		Status:     "active",
		Title:      "Alex is left-handed",
		Importance: 3,
	}
	out := renderMemoryRecord(t, r)

	for _, want := range []string{
		`class="kcard kcard-active"`,
		`id="kcard-abc123"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("MemoryRecordCard missing %q:\n%s", want, out)
		}
	}
}

func TestMemoryRecordCardHeader(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:         "r1",
		Status:     "proposed",
		Title:      "Prefers tea",
		Importance: 4,
	}
	out := renderMemoryRecord(t, r)

	for _, want := range []string{
		`class="kcard-kind"`,
		"▪ memory",
		`class="kcard-pips"`,
		`title="importance 4/5"`,
		`class="kcard-title"`,
		"Prefers tea",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q:\n%s", want, out)
		}
	}
}

func TestMemoryRecordCardNoCategoryShowsMemory(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:     "r2",
		Status: "active",
		Title:  "No category memory",
	}
	out := renderMemoryRecord(t, r)
	if !strings.Contains(out, "▪ memory") {
		t.Errorf("empty category should show '▪ memory':\n%s", out)
	}
}

func TestMemoryRecordCardBody(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:         "r3",
		Status:     "active",
		Title:      "Some fact",
		Content:    "The content here",
		WhenToUse:  "When discussing history",
		Importance: 2,
	}
	out := renderMemoryRecord(t, r)

	for _, want := range []string{
		`class="kcard-body"`,
		"The content here",
		`class="kcard-when"`,
		"recall: When discussing history",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("body missing %q:\n%s", want, out)
		}
	}
}

func TestMemoryRecordCardBodyEmptyContentOmitted(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:     "r4",
		Status: "active",
		Title:  "No content",
	}
	out := renderMemoryRecord(t, r)
	if strings.Contains(out, `class="kcard-body"`) {
		t.Errorf("empty content should not render kcard-body:\n%s", out)
	}
	if strings.Contains(out, `class="kcard-when"`) {
		t.Errorf("empty when_to_use should not render kcard-when:\n%s", out)
	}
}

func TestMemoryRecordCardEditForm(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:         "edit1",
		Status:     "active",
		Title:      "Edit me",
		Content:    "Old content",
		WhenToUse:  "anytime",
		Importance: 3,
	}
	out := renderMemoryRecord(t, r)

	for _, want := range []string{
		`class="kcard-edit"`,
		"Edit",
		`data-on:submit__prevent="@post(&#39;/ui/knowledge/memories/edit1/edit&#39;, {contentType:&#39;form&#39;})"`,
		`name="title"`,
		`value="Edit me"`,
		`name="content"`,
		"Old content",
		`name="importance"`,
		`value="3"`,
		`name="when_to_use"`,
		`value="anytime"`,
		`type="submit"`,
		"Save",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("edit form missing %q:\n%s", want, out)
		}
	}
}

func TestMemoryRecordCardStatusProposed(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:     "p1",
		Status: "proposed",
		Title:  "New proposal",
	}
	out := renderMemoryRecord(t, r)

	for _, want := range []string{
		`class="kcard kcard-proposed"`,
		`id="kcard-p1"`,
		`@post(&#39;/ui/knowledge/memories/p1/transition&#39;`,
		`value="active"`,
		"Approve",
		`value="rejected"`,
		"Dismiss",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("proposed status missing %q:\n%s", want, out)
		}
	}
	// proposed must not show Archive or Restore
	if strings.Contains(out, "Archive") {
		t.Errorf("proposed must not show Archive:\n%s", out)
	}
	if strings.Contains(out, "Restore") {
		t.Errorf("proposed must not show Restore:\n%s", out)
	}
}

func TestMemoryRecordCardStatusActive(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:       "a1",
		Status:   "active",
		Title:    "Active memory",
		UseCount: 7,
	}
	out := renderMemoryRecord(t, r)

	for _, want := range []string{
		`class="kcard kcard-active"`,
		`id="kcard-a1"`,
		`value="archived"`,
		"Archive",
		"used ×7",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("active status missing %q:\n%s", want, out)
		}
	}
	// active must not show Approve/Dismiss/Restore
	if strings.Contains(out, "Approve") {
		t.Errorf("active must not show Approve:\n%s", out)
	}
	if strings.Contains(out, "Dismiss") {
		t.Errorf("active must not show Dismiss:\n%s", out)
	}
	if strings.Contains(out, "Restore") {
		t.Errorf("active must not show Restore:\n%s", out)
	}
}

func TestMemoryRecordCardActiveZeroUseCountOmitted(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:       "a2",
		Status:   "active",
		Title:    "Active unused",
		UseCount: 0,
	}
	out := renderMemoryRecord(t, r)
	if strings.Contains(out, "used ×") {
		t.Errorf("zero use_count should not render 'used ×':\n%s", out)
	}
}

func TestMemoryRecordCardStatusArchived(t *testing.T) {
	r := knowledgecards.MemoryRecord{
		ID:     "ar1",
		Status: "archived",
		Title:  "Old memory",
	}
	out := renderMemoryRecord(t, r)

	for _, want := range []string{
		`class="kcard kcard-archived"`,
		`id="kcard-ar1"`,
		`value="active"`,
		"Restore",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("archived status missing %q:\n%s", want, out)
		}
	}
	// archived must not show Approve/Dismiss/Archive
	if strings.Contains(out, "Approve") {
		t.Errorf("archived must not show Approve:\n%s", out)
	}
	if strings.Contains(out, "Archive") {
		t.Errorf("archived must not show Archive:\n%s", out)
	}
}

func TestMemoryRecordCardPipCounts(t *testing.T) {
	// importance 5 → all 5 on
	r := knowledgecards.MemoryRecord{ID: "pip1", Status: "active", Title: "Max", Importance: 5}
	out := renderMemoryRecord(t, r)
	if strings.Count(out, `class="pip pip-on"`) != 5 {
		t.Errorf("importance 5 should have 5 pip-on:\n%s", out)
	}
	if strings.Count(out, `class="pip"`) != 0 {
		t.Errorf("importance 5 should have 0 plain pip:\n%s", out)
	}
}

func TestMemoryRecordCardPipCountZero(t *testing.T) {
	// importance 0 → all 5 off
	r := knowledgecards.MemoryRecord{ID: "pip2", Status: "active", Title: "Zero", Importance: 0}
	out := renderMemoryRecord(t, r)
	if strings.Count(out, `class="pip pip-on"`) != 0 {
		t.Errorf("importance 0 should have 0 pip-on:\n%s", out)
	}
	if strings.Count(out, `class="pip"`) != 5 {
		t.Errorf("importance 0 should have 5 plain pip:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// MemoryManageCard (manage view with proposed + active sections)
// ---------------------------------------------------------------------------

func TestMemoryManageCardStructure(t *testing.T) {
	v := knowledgecards.MemoryManageView{
		Proposed: []knowledgecards.MemoryRecord{
			{ID: "prop1", Status: "proposed", Title: "Proposed thing"},
		},
		Active: []knowledgecards.MemoryRecord{
			{ID: "act1", Status: "active", Title: "Active thing"},
		},
	}
	out := renderMemoryManage(t, v)

	for _, want := range []string{
		`class="kcard ucard ucard-manage ucard-memories-manage"`,
		`id="ucard-memories-manage"`,
		`class="kcard-kind"`,
		`src="/static/icons/tome.png"`,
		"Memory",
		`class="kcard-meta"`,
		`href="/ui/show/memory"`,
		"manage all →",
		`class="k-heading k-heading-proposed"`,
		"Awaiting your word",
		`class="k-heading k-heading-muted"`,
		"Active",
		`class="ucard-manage-list"`,
		`id="kcard-prop1"`,
		"Proposed thing",
		`id="kcard-act1"`,
		"Active thing",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("MemoryManageCard missing %q:\n%s", want, out)
		}
	}
}

func TestMemoryManageCardOnlyProposed(t *testing.T) {
	v := knowledgecards.MemoryManageView{
		Proposed: []knowledgecards.MemoryRecord{
			{ID: "p1", Status: "proposed", Title: "Only proposed"},
		},
	}
	out := renderMemoryManage(t, v)
	if !strings.Contains(out, "Awaiting your word") {
		t.Errorf("proposed section missing:\n%s", out)
	}
	// No active → no Active heading
	if strings.Contains(out, `class="k-heading k-heading-muted"`) {
		t.Errorf("should not show Active heading when active is empty:\n%s", out)
	}
}

func TestMemoryManageCardOnlyActive(t *testing.T) {
	v := knowledgecards.MemoryManageView{
		Active: []knowledgecards.MemoryRecord{
			{ID: "a1", Status: "active", Title: "Only active"},
		},
	}
	out := renderMemoryManage(t, v)
	if !strings.Contains(out, "Active") {
		t.Errorf("active section missing:\n%s", out)
	}
	// No proposed → no proposed heading
	if strings.Contains(out, "Awaiting your word") {
		t.Errorf("should not show proposed heading when proposed is empty:\n%s", out)
	}
}

func TestMemoryManageCardEmptyState(t *testing.T) {
	v := knowledgecards.MemoryManageView{}
	out := renderMemoryManage(t, v)
	if !strings.Contains(out, "Nothing yet — Memory appears as Balaur proposes.") {
		t.Errorf("expected empty state:\n%s", out)
	}
	if strings.Contains(out, "Awaiting your word") {
		t.Errorf("empty state should not show proposed heading:\n%s", out)
	}
	if strings.Contains(out, `class="k-heading k-heading-muted"`) {
		t.Errorf("empty state should not show active heading:\n%s", out)
	}
}
