package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCardRegistry(t *testing.T) {
	stub := func(ui.CardSize, map[string]string) (g.Node, error) {
		return g.Text("x"), nil
	}

	if _, ok := ui.LookupCard("probe"); ok {
		t.Fatal("probe should be absent before registration")
	}

	ui.RegisterCard("probe", stub)
	if _, ok := ui.LookupCard("probe"); !ok {
		t.Fatal("probe should be registered")
	}

	ui.UnregisterCard("probe")
	if _, ok := ui.LookupCard("probe"); ok {
		t.Fatal("probe should be removed after unregister")
	}
}
