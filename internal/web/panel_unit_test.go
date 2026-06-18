package web

import (
	"testing"
)

func TestParseShowURL(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantTyp   string
		wantQuery string
		wantOK    bool
	}{
		{"simple type", "/ui/show/quests", "quests", "", true},
		{"with query", "/ui/show/memory?category=fact", "memory", "category=fact", true},
		{"empty type", "/ui/show/", "", "", false},
		{"bad prefix", "/ui/cards/quests", "", "", false},
		{"empty string", "", "", "", false},
		{"show_cards prefix", "/ui/show_cards/foo", "", "", false},
		{"type with subpath", "/ui/show/settings?section=models", "settings", "section=models", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, query, ok := parseShowURL(tt.raw)
			if ok != tt.wantOK {
				t.Errorf("parseShowURL(%q) ok=%v, want %v", tt.raw, ok, tt.wantOK)
			}
			if typ != tt.wantTyp {
				t.Errorf("parseShowURL(%q) typ=%q, want %q", tt.raw, typ, tt.wantTyp)
			}
			if query != tt.wantQuery {
				t.Errorf("parseShowURL(%q) query=%q, want %q", tt.raw, query, tt.wantQuery)
			}
		})
	}
}

func TestShowURL(t *testing.T) {
	tests := []struct {
		typ   string
		query string
		want  string
	}{
		{"quests", "", "/ui/show/quests"},
		{"memory", "category=fact", "/ui/show/memory?category=fact"},
		{"settings", "section=models", "/ui/show/settings?section=models"},
	}
	for _, tt := range tests {
		got := showURL(tt.typ, tt.query)
		if got != tt.want {
			t.Errorf("showURL(%q, %q) = %q, want %q", tt.typ, tt.query, got, tt.want)
		}
	}
}
