package tools

import (
	"strings"

	"github.com/alexradunet/balaur/internal/cards"
)

// RefreshMarker prefixes a mutating tool's result so the web layer re-renders
// the affected on-screen cards live (in the open chat SSE stream) after the
// mutation commits. Format: marker + comma-joined card types + "\n" +
// model-facing text. Mirrors UICardMarker/ProposalMarker: one NUL-prefixed
// marker per result; the marker line is inert noise to the model, the text
// follows.
const RefreshMarker = "\x00balaur-refresh:"

// MarkRefresh wraps modelText with a refresh directive for the given card
// types. Unknown types are tolerated here and dropped at parse time.
func MarkRefresh(types []string, modelText string) string {
	return RefreshMarker + strings.Join(types, ",") + "\n" + modelText
}

// ParseRefresh splits a refresh-marked result into the registered card types to
// refresh and the model-facing rest. ok is false for ordinary text. Unknown or
// unregistered types are dropped; ok is false when none remain (so a stale/
// hallucinated type can never name a non-existent card).
func ParseRefresh(s string) (types []string, rest string, ok bool) {
	if !strings.HasPrefix(s, RefreshMarker) {
		return nil, s, false
	}
	s = strings.TrimPrefix(s, RefreshMarker)
	head, rest, _ := strings.Cut(s, "\n")
	for t := range strings.SplitSeq(head, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, found := cards.Get(t); found {
			types = append(types, t)
		}
	}
	if len(types) == 0 {
		return nil, rest, false
	}
	return types, rest, true
}
