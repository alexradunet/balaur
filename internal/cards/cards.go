// Package cards is the typed card registry — the single source of truth for
// every card type that Balaur can compose on-the-spot. All card types are
// parameterized server resources at GET /ui/cards/{type}?params, rendered
// server-side from PocketBase data.
//
// This package has zero imports of internal/web so that plan-030 agent tools
// can call Get/Validate without creating an import cycle.
package cards

import (
	"fmt"
	"strconv"
	"strings"
)

// ParamSpec describes one accepted query parameter for a card type.
type ParamSpec struct {
	Name     string   // query/JSON key
	Required bool     // if true, Validate returns error when absent
	Enum     []string // optional closed value set; nil/empty = any string
	Doc      string   // one-line, model- and owner-facing description
}

// Spec is the static description of one card type.
type Spec struct {
	Type   string      // "today", "quests", …
	Label  string      // "Today"
	Icon   string      // icon file stem under /static/icons
	W      int         // default grid span (of 12)
	Params []ParamSpec // accepted query parameters
}

// registry holds the canonical set of card specs, in definition order.
var registry []Spec

// byType is a fast-lookup index built at init time.
var byType map[string]Spec

func init() {
	registry = []Spec{
		{
			Type:  "today",
			Label: "Today",
			Icon:  "scroll",
			W:     4,
			// no params — open tasks due/overdue today
		},
		{
			Type:  "quests",
			Label: "Quest log",
			Icon:  "scroll",
			W:     8,
			Params: []ParamSpec{
				{Name: "status", Enum: []string{"open", "done", "all"}, Doc: "filter by task status (default: open)"},
				{Name: "limit", Doc: "maximum rows to show (default 10, max 50)"},
			},
		},
		{
			Type:  "calendar",
			Label: "Calendar",
			Icon:  "hourglass",
			W:     4,
			Params: []ParamSpec{
				{Name: "month", Doc: "YYYY-MM month to display (default: current month)"},
			},
		},
		{
			Type:  "timeline",
			Label: "Timeline",
			Icon:  "hourglass",
			W:     8,
			Params: []ParamSpec{
				{Name: "days", Doc: "how many days to look ahead (default 14, max 31)"},
			},
		},
		{
			Type:  "journal",
			Label: "Journal",
			Icon:  "quill",
			W:     4,
			Params: []ParamSpec{
				{Name: "limit", Doc: "number of recent journal entries to show (default 5, max 50)"},
			},
		},
		{
			Type:  "measure",
			Label: "Measure",
			Icon:  "orb",
			W:     4,
			Params: []ParamSpec{
				{Name: "kind", Required: true, Doc: "a numeric life-entry kind (owner-defined)"},
				{Name: "days", Doc: "look-back window in days (default 90, max 366)"},
			},
		},
		{
			Type:  "lines",
			Label: "Recent lines",
			Icon:  "orb",
			W:     4,
			Params: []ParamSpec{
				{Name: "kind", Required: true, Doc: "a text life-entry kind (owner-defined)"},
				{Name: "limit", Doc: "number of recent entries to show (default 5, max 50)"},
			},
		},
		{
			Type:  "memory",
			Label: "Memory",
			Icon:  "tome",
			W:     4,
			Params: []ParamSpec{
				{Name: "query", Doc: "optional search terms to filter active memories"},
				{Name: "limit", Doc: "number of memories to show (default 6, max 50)"},
			},
		},
		{
			Type:  "skills",
			Label: "Skills",
			Icon:  "key",
			W:     4,
			Params: []ParamSpec{
				{Name: "limit", Doc: "number of skills to show (default 6, max 50)"},
			},
		},
		{
			Type:  "heads",
			Label: "Heads",
			Icon:  "tome",
			W:     4,
			// no params — all active heads
		},
	}

	byType = make(map[string]Spec, len(registry))
	for _, s := range registry {
		byType[s.Type] = s
	}
}

// All returns every registered card spec in definition order.
func All() []Spec { return registry }

// Get looks up a spec by type name. Returns (spec, true) if found.
func Get(typ string) (Spec, bool) {
	s, ok := byType[typ]
	return s, ok
}

// Card is a typed, parameterized card reference — the composition unit for
// boards and on-the-spot UI. It matches the JSON shape stored in the boards
// collection's cards field and produced by the board_compose tool.
type Card struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}

// ValidateCards validates each entry via Validate and returns the cleaned list.
// The first invalid card stops validation and returns an error.
func ValidateCards(cs []Card) ([]Card, error) {
	out := make([]Card, 0, len(cs))
	for i, c := range cs {
		cleaned, err := Validate(c.Type, c.Params)
		if err != nil {
			return nil, fmt.Errorf("card[%d]: %w", i, err)
		}
		card := Card{Type: c.Type}
		if len(cleaned) > 0 {
			card.Params = cleaned
		}
		out = append(out, card)
	}
	return out, nil
}

// Validate checks params against the spec for typ. Rules:
//   - Unknown type → error.
//   - Unknown param keys → silently dropped (returns cleaned map).
//   - Required params absent → error.
//   - Enum params with unknown value → error.
//   - Numeric params (limit, days) are parsed and clamped to sane bounds;
//     a non-parseable value silently falls back to the default (cards must
//     be forgiving — the agent composes these).
//
// Returns a cleaned param map containing only known keys, with clamped
// numeric values stored back as strings.
func Validate(typ string, params map[string]string) (map[string]string, error) {
	spec, ok := byType[typ]
	if !ok {
		return nil, fmt.Errorf("unknown card type %q", typ)
	}

	// Build a set of known param names for quick lookup.
	known := make(map[string]ParamSpec, len(spec.Params))
	for _, p := range spec.Params {
		known[p.Name] = p
	}

	out := make(map[string]string, len(params))
	for _, ps := range spec.Params {
		v, present := params[ps.Name]

		// Required check.
		if ps.Required && (!present || strings.TrimSpace(v) == "") {
			return nil, fmt.Errorf("card %q requires param %q", typ, ps.Name)
		}

		if !present || v == "" {
			continue
		}

		// Enum check.
		if len(ps.Enum) > 0 {
			if !enumContains(ps.Enum, v) {
				return nil, fmt.Errorf("param %q must be one of [%s], got %q",
					ps.Name, strings.Join(ps.Enum, ", "), v)
			}
		}

		// Numeric clamping for well-known numeric params — fall back silently
		// on parse errors.
		switch ps.Name {
		case "limit":
			out[ps.Name] = clampInt(v, 1, 50)
			continue
		case "days":
			out[ps.Name] = clampInt(v, 1, 366)
			continue
		}

		out[ps.Name] = v
	}

	// Drop unknown keys by only copying known ones (already done above).
	// params not in known are silently ignored — they never enter out.
	_ = known // used for correctness reasoning only
	return out, nil
}

func enumContains(enum []string, v string) bool {
	for _, e := range enum {
		if e == v {
			return true
		}
	}
	return false
}

// clampInt parses s as an int, clamps it to [lo, hi], and returns the
// result as a string. On parse error it returns the empty string so the
// renderer can apply its own default.
func clampInt(s string, lo, hi int) string {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return "" // caller applies default
	}
	if n < lo {
		n = lo
	}
	if n > hi {
		n = hi
	}
	return strconv.Itoa(n)
}
