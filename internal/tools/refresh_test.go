package tools_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/tools"
)

func TestRefreshMarkerRoundTrip(t *testing.T) {
	marked := tools.MarkRefresh([]string{"today"}, `Done: "Buy milk".`)
	types, rest, ok := tools.ParseRefresh(marked)
	if !ok {
		t.Fatal("expected ok=true for well-formed marked text")
	}
	if len(types) != 1 || types[0] != "today" {
		t.Fatalf("types = %v, want [today]", types)
	}
	if rest != `Done: "Buy milk".` {
		t.Fatalf("rest = %q, want the plain text", rest)
	}
}

func TestParseRefreshPlainText(t *testing.T) {
	types, rest, ok := tools.ParseRefresh("just a normal tool reply")
	if ok || types != nil || rest != "just a normal tool reply" {
		t.Fatalf("plain text must not parse; got types=%v rest=%q ok=%v", types, rest, ok)
	}
}

func TestParseRefreshDropsUnknownTypes(t *testing.T) {
	// "today" is a registered card type; "nope" is not.
	types, _, ok := tools.ParseRefresh(tools.MarkRefresh([]string{"nope", "today"}, "x"))
	if !ok || len(types) != 1 || types[0] != "today" {
		t.Fatalf("expected only [today], got %v ok=%v", types, ok)
	}
	if _, _, ok := tools.ParseRefresh(tools.MarkRefresh([]string{"nope"}, "x")); ok {
		t.Fatal("all-unknown types must yield ok=false")
	}
}
