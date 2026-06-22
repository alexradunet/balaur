package web

import (
	"strings"
	"testing"
)

func TestRecapBandsNodeEmpty(t *testing.T) {
	out := renderNodeHTML(recapBandsNode(nil))
	if out != "" {
		t.Errorf("empty []bandView: want empty string, got %q", out)
	}
	if strings.Contains(out, "recap-band") {
		t.Errorf("empty: must not contain recap-band: %s", out)
	}
	if strings.Contains(out, "stitch") {
		t.Errorf("empty: must not contain stitch: %s", out)
	}
}

func TestRecapBandsNodeStructure(t *testing.T) {
	view := []bandView{
		{Heading: "Earlier this week", Cards: []recapView{
			{Type: "day", Label: "Mon", Content: "busy day", Start: "1000", HasChild: false},
		}},
	}
	out := renderNodeHTML(recapBandsNode(view))
	for _, want := range []string{
		`class="recap-band"`,
		`class="recap-heading"`,
		"◇",
		"Earlier this week",
		`class="stitch"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestRecapBandsNodeReverseOrder(t *testing.T) {
	view := []bandView{
		{Heading: "A"},
		{Heading: "B"},
	}
	out := renderNodeHTML(recapBandsNode(view))
	posA := strings.Index(out, "A")
	posB := strings.Index(out, "B")
	if posA == -1 || posB == -1 {
		t.Fatalf("A or B not found in output:\n%s", out)
	}
	if posB >= posA {
		t.Errorf("reverse order: B must appear before A, got B@%d A@%d", posB, posA)
	}
}

func TestRecapCardsNodeHasChild(t *testing.T) {
	cards := []recapView{
		{Type: "week", Label: "Wk 25", Content: "summary", Start: "1700000000", HasChild: true},
	}
	out := renderNodeHTML(recapCardsNode(cards))
	if !strings.Contains(out, `class="recap-card recap-week"`) {
		t.Errorf("missing article class:\n%s", out)
	}
	// gomponents HTML-escapes single quotes in attribute values to &#39;
	if !strings.Contains(out, "@get(&#39;/ui/recap/expand?type=week&amp;start=1700000000&#39;)") {
		t.Errorf("missing expand @get for HasChild card:\n%s", out)
	}
	if !strings.Contains(out, "recap-open") {
		t.Errorf("missing recap-open class toggle:\n%s", out)
	}
	if !strings.Contains(out, ">open<") {
		t.Errorf("missing 'open' button text:\n%s", out)
	}
	if !strings.Contains(out, `id="recap-children-week-1700000000"`) {
		t.Errorf("missing recap-children id:\n%s", out)
	}
	// Day-style transcript button must not appear for HasChild cards.
	if strings.Contains(out, ">transcript<") {
		t.Errorf("HasChild card must not have transcript button:\n%s", out)
	}
}

func TestRecapCardsNodeDayWithDate(t *testing.T) {
	cards := []recapView{
		{Type: "day", Label: "Mon 23 Jun", Content: "quiet day", Start: "1750000000", Date: "2025-06-23", HasChild: false},
	}
	out := renderNodeHTML(recapCardsNode(cards))
	if !strings.Contains(out, `class="recap-daylink"`) {
		t.Errorf("missing recap-daylink:\n%s", out)
	}
	if !strings.Contains(out, `href="/ui/show/day?date=2025-06-23"`) {
		t.Errorf("missing daylink href:\n%s", out)
	}
	if !strings.Contains(out, ">visit<") {
		t.Errorf("missing 'visit' link text:\n%s", out)
	}
	// gomponents HTML-escapes single quotes in attribute values to &#39;
	if !strings.Contains(out, "@get(&#39;/ui/recap/expand?type=day&amp;start=1750000000&#39;)") {
		t.Errorf("missing transcript @get:\n%s", out)
	}
	if !strings.Contains(out, ">transcript<") {
		t.Errorf("missing 'transcript' button text:\n%s", out)
	}
	if !strings.Contains(out, `id="recap-children-day-1750000000"`) {
		t.Errorf("missing recap-children id:\n%s", out)
	}
}
