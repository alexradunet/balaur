package storybook_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/storybook"
)

func TestStoriesUniqueAndLookup(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range storybook.Stories() {
		if s.ID == "" || s.Group == "" || s.Title == "" || s.Canvas == nil {
			t.Fatalf("incomplete story: %+v", s)
		}
		if seen[s.ID] {
			t.Fatalf("duplicate story id %q", s.ID)
		}
		seen[s.ID] = true
	}
	if len(seen) < 15 {
		t.Errorf("expected >=15 stories, got %d", len(seen))
	}
	if _, ok := storybook.Lookup("button"); !ok {
		t.Error(`Lookup("button") not found`)
	}
	if _, ok := storybook.Lookup("nope"); ok {
		t.Error(`Lookup("nope") should be false`)
	}
}

func TestButtonCanvasRenders(t *testing.T) {
	s, _ := storybook.Lookup("button")
	var b strings.Builder
	if err := s.Canvas().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, `class="btn btn-primary"`) {
		t.Errorf("button canvas missing button: %s", got)
	}
}

func TestColorsCanvas(t *testing.T) {
	s, ok := storybook.Lookup("colors")
	if !ok {
		t.Fatal("colors story not registered")
	}
	var b strings.Builder
	if err := s.Canvas().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<span class="section-label-text">Accents</span>`,
		`<div class="swatch-chip" style="--sw:var(--gold)"></div>`,
		`<div class="swatch-name">--bg</div>`,
		`<div class="swatch-label">gold</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("colors canvas missing %q", want)
		}
	}
}

func TestTypographyCanvas(t *testing.T) {
	s, ok := storybook.Lookup("typography")
	if !ok {
		t.Fatal("typography story not registered")
	}
	var b strings.Builder
	if err := s.Canvas().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<div class="type-role">Display</div>`,
		`<div class="type-sample-display">A new head wakes</div>`,
		`<div class="type-sample-pixel">BALAUR</div>`,
		`The hearth is lit`,
		`<span class="type-scale-tag">36</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("typography canvas missing %q", want)
		}
	}
}

func TestMaterialsCanvas(t *testing.T) {
	s, ok := storybook.Lookup("materials")
	if !ok {
		t.Fatal("materials story not registered")
	}
	var b strings.Builder
	if err := s.Canvas().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<div class="mat-title">Parchment</div>`,
		`<div class="mat-swatch mat-wood"></div>`,
		`<div class="mat-title">Wood chrome</div>`,
		`class="folk-band"`,
		`class="stitch"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("materials canvas missing %q", want)
		}
	}
}

func TestOverviewRenders(t *testing.T) {
	var b strings.Builder
	if err := storybook.Overview().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{"Woven, not rendered.", `class="sb-stats"`, `href="/storybook/button"`} {
		if !strings.Contains(got, want) {
			t.Errorf("overview missing %q in: %s", want, got)
		}
	}
}
