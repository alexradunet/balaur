package headscards_test

import (
	"bytes"
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/store"
)

// render is a test helper that renders a Node to a string.
func render(t *testing.T, n g.Node) string {
	t.Helper()
	var buf bytes.Buffer
	if err := n.Render(&buf); err != nil {
		t.Fatalf("render error: %v", err)
	}
	return buf.String()
}

// syntheticView returns a predictable HeadsView for pure component tests.
func syntheticView() headscards.HeadsView {
	return headscards.HeadsView{
		Heads: []headscards.HeadRow{
			{
				ID:        "balaur",
				Name:      "Balaur",
				Purpose:   "",
				AvatarURL: "/static/avatars/balaur-01.png",
				BuiltIn:   true,
				Active:    true,
				Groups: []headscards.GroupChoice{
					{Key: "memory", On: false},
					{Key: "tasks", On: false},
				},
			},
			{
				ID:        "scholar",
				Name:      "Scholar",
				Purpose:   "explains and researches",
				AvatarURL: "/static/avatars/balaur-04.png",
				BuiltIn:   true,
				Active:    false,
				Groups: []headscards.GroupChoice{
					{Key: "memory", On: true},
					{Key: "tasks", On: false},
				},
			},
			{
				ID:        "custom-01",
				Name:      "My Head",
				Purpose:   "does stuff",
				AvatarURL: "/static/avatars/balaur-03.png",
				BuiltIn:   false,
				Active:    false,
				Groups: []headscards.GroupChoice{
					{Key: "memory", On: false},
					{Key: "tasks", On: true},
				},
			},
		},
		Avatars: []store.AvatarEntry{
			{Key: "balaur-01", Label: "One", URL: "/static/avatars/balaur-01.png"},
			{Key: "balaur-02", Label: "Two", URL: "/static/avatars/balaur-02.png"},
		},
		Groups: []string{"memory", "tasks"},
	}
}

func TestHeadsCard_RootElement(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	if !strings.Contains(html, `id="ucard-heads"`) {
		t.Errorf("missing id=ucard-heads:\n%s", html)
	}
	if !strings.Contains(html, `class="kcard ucard ucard-heads ucard-manage"`) {
		t.Errorf("missing root classes:\n%s", html)
	}
}

func TestHeadsCard_HeadRows(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// Each head has a row with its ID.
	for _, id := range []string{"balaur", "scholar", "custom-01"} {
		if !strings.Contains(html, `id="head-`+id+`"`) {
			t.Errorf("missing row id head-%s:\n%s", id, html)
		}
	}
}

func TestHeadsCard_ActiveRow(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// Active head (balaur) gets head-row-active class.
	if !strings.Contains(html, `class="head-row head-row-active"`) {
		t.Errorf("active row missing head-row-active class:\n%s", html)
	}
	// Non-active head should not have head-row-active.
	if strings.Contains(html, `id="head-scholar" class="head-row head-row-active"`) {
		t.Errorf("scholar incorrectly has head-row-active:\n%s", html)
	}
}

func TestHeadsCard_Avatar(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	if !strings.Contains(html, `class="px head-row-avatar"`) {
		t.Errorf("missing avatar img class:\n%s", html)
	}
	if !strings.Contains(html, `src="/static/avatars/balaur-01.png"`) {
		t.Errorf("missing balaur avatar URL:\n%s", html)
	}
}

func TestHeadsCard_NameTags(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// built-in tag appears for built-in heads.
	if !strings.Contains(html, `<span class="tag">built-in</span>`) {
		t.Errorf("missing built-in tag:\n%s", html)
	}
	// active tag appears for the active head.
	if !strings.Contains(html, `<span class="tag">active</span>`) {
		t.Errorf("missing active tag:\n%s", html)
	}
}

func TestHeadsCard_Purpose(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// Scholar has a purpose; it should render in kcard-meta.
	if !strings.Contains(html, `explains and researches`) {
		t.Errorf("missing purpose text:\n%s", html)
	}
}

func TestHeadsCard_GroupPip(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// Scholar has memory On=true → a pip for "memory".
	if !strings.Contains(html, `<span class="head-group-pip">memory</span>`) {
		t.Errorf("missing group pip for memory:\n%s", html)
	}
}

func TestHeadsCard_MakeActiveForm(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// Scholar is not active → should have Make active form.
	// Gomponents HTML-escapes single quotes in attribute values to &#39;.
	if !strings.Contains(html, `@post(&#39;/ui/heads/active&#39;, {contentType:&#39;form&#39;})`) {
		t.Errorf("missing make-active post action:\n%s", html)
	}
	if !strings.Contains(html, `>Make active</button>`) {
		t.Errorf("missing Make active button:\n%s", html)
	}

	// Active head (balaur) must NOT have a Make active form.
	// We check by asserting "Make active" only appears twice (for scholar and custom-01).
	count := strings.Count(html, `>Make active</button>`)
	if count != 2 { // scholar + custom-01
		t.Errorf("expected 2 Make active buttons (non-active heads), got %d:\n%s", count, html)
	}
}

func TestHeadsCard_DeleteForm(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// custom-01 is not built-in → should have delete form.
	// Gomponents HTML-escapes single quotes in attribute values to &#39;.
	if !strings.Contains(html, `@post(&#39;/ui/heads/custom-01/delete&#39;, {contentType:&#39;form&#39;})`) {
		t.Errorf("missing delete form for custom-01:\n%s", html)
	}
	if !strings.Contains(html, `>Delete</button>`) {
		t.Errorf("missing Delete button:\n%s", html)
	}

	// Built-in heads (balaur, scholar) must NOT have delete forms.
	deleteCount := strings.Count(html, `>Delete</button>`)
	if deleteCount != 1 {
		t.Errorf("expected exactly 1 Delete button (custom-01 only), got %d:\n%s", deleteCount, html)
	}
}

func TestHeadsCard_NewHeadDetails(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	if !strings.Contains(html, `class="head-new"`) {
		t.Errorf("missing head-new details:\n%s", html)
	}
	if !strings.Contains(html, `+ New head`) {
		t.Errorf("missing '+ New head' summary:\n%s", html)
	}
}

func TestHeadsCard_NewHeadForm(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	if !strings.Contains(html, `class="head-new-form"`) {
		t.Errorf("missing head-new-form class:\n%s", html)
	}
	// Gomponents HTML-escapes single quotes in attribute values to &#39;.
	if !strings.Contains(html, `@post(&#39;/ui/heads/new&#39;, {contentType:&#39;form&#39;})`) {
		t.Errorf("missing new-head post action:\n%s", html)
	}
	// Name input
	if !strings.Contains(html, `name="name"`) {
		t.Errorf("missing name input:\n%s", html)
	}
	// Purpose input
	if !strings.Contains(html, `name="purpose"`) {
		t.Errorf("missing purpose input:\n%s", html)
	}
}

func TestHeadsCard_ToolCheckboxes(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// Groups fieldset
	if !strings.Contains(html, `class="head-new-groups"`) {
		t.Errorf("missing head-new-groups fieldset:\n%s", html)
	}
	if !strings.Contains(html, `Tools (none = all)`) {
		t.Errorf("missing Tools legend:\n%s", html)
	}
	// One checkbox per group
	for _, grp := range []string{"memory", "tasks"} {
		if !strings.Contains(html, `name="tools" value="`+grp+`"`) {
			t.Errorf("missing checkbox for group %q:\n%s", grp, html)
		}
	}
}

func TestHeadsCard_AvatarRadios(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	// Avatars fieldset
	if !strings.Contains(html, `class="head-new-avatars avatar-choice-list"`) {
		t.Errorf("missing avatar-choice-list fieldset:\n%s", html)
	}
	if !strings.Contains(html, `Avatar`) {
		t.Errorf("missing Avatar legend:\n%s", html)
	}
	// Radio per avatar entry
	if !strings.Contains(html, `name="balaur_avatar" value="balaur-01"`) {
		t.Errorf("missing balaur-01 radio:\n%s", html)
	}
	if !strings.Contains(html, `class="avatar-choice"`) {
		t.Errorf("missing avatar-choice label class:\n%s", html)
	}
}

func TestHeadsCard_CreateButton(t *testing.T) {
	html := render(t, headscards.HeadsCard(syntheticView()))

	if !strings.Contains(html, `>Create head</button>`) {
		t.Errorf("missing Create head button:\n%s", html)
	}
	if !strings.Contains(html, `class="btn btn-primary btn-sm"`) {
		t.Errorf("missing btn-primary class on create button:\n%s", html)
	}
}
