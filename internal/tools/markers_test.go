package tools

import (
	"testing"

	"github.com/alexradunet/balaur/internal/cards"
)

// markerCases is EXHAUSTIVE: every NUL-prefixed marker constant in this package
// must appear here. Adding a Mark*/Parse* pair without adding a case is exactly
// the silent drift this test guards.
func markerCases() []struct {
	name   string
	sample string
	parse  func(string) bool
} {
	return []struct {
		name   string
		sample string
		parse  func(string) bool
	}{
		{"uicard", MarkUICard("today", map[string]string{}, "showing today"),
			func(s string) bool { _, _, _, ok := ParseUICard(s); return ok }},
		{"choices", MarkChoices("Pick one", []Choice{{Label: "Option A"}, {Label: "Option B"}}, "choose"),
			func(s string) bool { _, _, _, ok := ParseChoices(s); return ok }},
		{"proposal", MarkProposal("memory", "abc123", "proposed a memory"),
			func(s string) bool { _, _, _, ok := ParseProposal(s); return ok }},
		{"refresh", MarkRefresh([]string{"today"}, "refreshed"),
			func(s string) bool { _, _, ok := ParseRefresh(s); return ok }},
		{"artifact", MarkArtifact([]cards.Card{{Type: "today"}}, "Cluster", "showing a cluster"),
			func(s string) bool { _, _, _, ok := ParseArtifact(s); return ok }},
	}
}

func TestMarkersRoundTripAndIsolate(t *testing.T) {
	cases := markerCases()
	// Round-trip: each marker's own Parse accepts its own sample.
	for _, c := range cases {
		if !c.parse(c.sample) {
			t.Errorf("%s: own Parse rejected its own Mark output", c.name)
		}
	}
	// Isolation: every Parse rejects every OTHER marker's output.
	for _, owner := range cases {
		for _, other := range cases {
			if owner.name == other.name {
				continue
			}
			if owner.parse(other.sample) {
				t.Errorf("%s.Parse accepted %s's output (markers must not cross-match)", owner.name, other.name)
			}
		}
	}
	// Plain text: every Parse rejects unmarked text.
	for _, c := range cases {
		if c.parse("just some assistant text, no marker") {
			t.Errorf("%s.Parse accepted plain text", c.name)
		}
	}
}
