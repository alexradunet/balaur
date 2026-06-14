package ui

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ErrorStrip is the inline card-error fragment, the gomponents equivalent of
// the legacy cardErrorStrip. g.Text auto-escapes msg, so a model- or
// user-derived string can never inject markup — the no-raw-HTML firewall.
// Never replace g.Text here with g.Raw.
func ErrorStrip(msg string) g.Node {
	return Div(Class("card-note card-note-error"), g.Text(msg))
}
