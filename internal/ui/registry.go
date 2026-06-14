package ui

import g "maragu.dev/gomponents"

// CardSize selects which size a card renders at: a compact board tile or the
// full-canvas focus view.
type CardSize int

const (
	Tile CardSize = iota
	Focus
)

// CardFunc renders one card of a feature-owned type to a gomponents node. A
// feature package registers one per card type it owns.
type CardFunc func(size CardSize, params map[string]string) (g.Node, error)

// cardRegistry maps card type -> renderer. It is written only at startup (each
// feature's Mount) and read during requests, so no locking is needed.
var cardRegistry = map[string]CardFunc{}

// RegisterCard registers a gomponents renderer for a card type. Call at startup
// only.
func RegisterCard(typ string, fn CardFunc) { cardRegistry[typ] = fn }

// UnregisterCard removes a registration. Intended for tests.
func UnregisterCard(typ string) { delete(cardRegistry, typ) }

// LookupCard returns the renderer for typ, if a feature has registered one.
func LookupCard(typ string) (CardFunc, bool) {
	fn, ok := cardRegistry[typ]
	return fn, ok
}
