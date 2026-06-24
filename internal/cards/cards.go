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
	"slices"
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
	H      int         // default height in row units (row unit = 10px)
	Params []ParamSpec // accepted query parameters
}

// maxParamLen is the maximum byte length for free-string card params.
// Model-composable params are capped here to prevent tool calls from bloating
// stored board JSON without bound.
const maxParamLen = 256

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
			H:     16,
			// no params — open tasks due/overdue today
		},
		{
			Type:  "quests",
			Label: "Quest log",
			Icon:  "scroll",
			W:     8,
			H:     30,
			Params: []ParamSpec{
				{Name: "mode", Enum: []string{"summary", "manage"}, Doc: "summary (read-only) or manage (Done/Snooze/Drop inline)"},
				{Name: "status", Enum: []string{"open", "done", "all"}, Doc: "filter by task status (default: open)"},
				{Name: "limit", Doc: "maximum rows to show (default 10, max 50)"},
			},
		},
		{
			Type:  "calendar",
			Label: "Calendar",
			Icon:  "hourglass",
			W:     4,
			H:     26,
			Params: []ParamSpec{
				{Name: "month", Doc: "YYYY-MM month to display (default: current month)"},
			},
		},
		{
			Type:  "timeline",
			Label: "Timeline",
			Icon:  "hourglass",
			W:     8,
			H:     26,
			Params: []ParamSpec{
				{Name: "days", Doc: "how many days to look ahead (default 14, max 31)"},
			},
		},
		{
			Type:  "day",
			Label: "Day",
			Icon:  "scroll",
			W:     4,
			H:     22,
			Params: []ParamSpec{
				{Name: "date", Doc: "YYYY-MM-DD to show (default: today)"},
			},
		},
		{
			// period is the coarser telescope lens (week/month/quarter/year):
			// a SYNTHESISED view over the summary + range aggregates, never a
			// stored node (only type=day nodes exist — plan 171).
			Type:  "period",
			Label: "Period",
			Icon:  "hourglass",
			W:     4,
			H:     22,
			Params: []ParamSpec{
				{Name: "type", Required: true, Enum: []string{"week", "month", "quarter", "year"}, Doc: "period granularity"},
				{Name: "start", Required: true, Doc: "period start as unix seconds"},
			},
		},
		{
			Type:  "measure",
			Label: "Measure",
			Icon:  "orb",
			W:     4,
			H:     12,
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
			H:     12,
			Params: []ParamSpec{
				{Name: "kind", Required: true, Doc: "a text life-entry kind (owner-defined)"},
				{Name: "limit", Doc: "number of recent entries to show (default 5, max 50)"},
			},
		},
		{
			Type:  "note",
			Label: "Note",
			Icon:  "tome",
			W:     4,
			H:     20,
			Params: []ParamSpec{
				{Name: "id", Required: true, Doc: "node id to show"},
				{Name: "type", Doc: "node type for typed-object render"},
			},
		},
		{
			Type:  "memory",
			Label: "Memory",
			Icon:  "tome",
			W:     4,
			H:     20,
			Params: []ParamSpec{
				{Name: "mode", Enum: []string{"summary", "manage"}, Doc: "summary (read-only) or manage (approve/archive inline)"},
				{Name: "category", Enum: []string{"fact", "preference", "person", "project", "context"}, Doc: "show one memory category (the Knowledge sidebar sub-items)"},
				{Name: "view", Enum: []string{"active", "proposed"}, Doc: "active (default — the category listing) or proposed (the Awaiting approval queue)"},
				{Name: "query", Doc: "optional search terms to filter active memories"},
				{Name: "limit", Doc: "number of memories to show (default 6, max 50)"},
			},
		},
		{
			Type:  "skills",
			Label: "Skills",
			Icon:  "key",
			W:     4,
			H:     14,
			Params: []ParamSpec{
				{Name: "mode", Enum: []string{"summary", "manage"}, Doc: "summary (read-only) or manage (approve/archive inline)"},
				{Name: "limit", Doc: "number of skills to show (default 6, max 50)"},
			},
		},
		{
			Type:  "related",
			Label: "Related",
			Icon:  "tome",
			W:     4,
			H:     20,
			Params: []ParamSpec{
				{Name: "id", Required: true, Doc: "the focus node id whose neighbors to list"},
				{Name: "limit", Doc: "max related nodes to show (default 12, max 50)"},
			},
		},
		{
			Type:  "graph",
			Label: "Graph",
			Icon:  "tome",
			W:     6,
			H:     24,
			Params: []ParamSpec{
				{Name: "id", Required: true, Doc: "the focus node id whose 1-hop neighborhood to draw"},
				{Name: "limit", Doc: "max neighbors to draw (default 12, max 24)"},
			},
		},
		{
			Type:  "network",
			Label: "Graph",
			Icon:  "lens",
			W:     6,
			H:     24,
			// no params — the network card draws the whole active graph
		},
		{
			Type:  "heads",
			Label: "Heads",
			Icon:  "tome",
			W:     4,
			H:     16,
			// no params — the heads card is the persona manager
		},
		{
			Type:  "habits",
			Label: "Habits",
			Icon:  "flame",
			W:     4,
			H:     14,
			// no params — recurring tasks with their current streak
		},
		{
			Type:  "lifelog",
			Label: "Life",
			Icon:  "orb",
			W:     6,
			H:     24,
			// no params — the full tracked overview + habits
		},
		{
			Type:  "tasks",
			Label: "Tasks",
			Icon:  "scroll",
			W:     4,
			H:     20,
			Params: []ParamSpec{
				{Name: "status", Enum: []string{"open", "done", "all"}, Doc: "filter by task status (default: open)"},
				{Name: "bucket", Enum: []string{"overdue", "today", "upcoming", "someday"}, Doc: "narrow OPEN tasks to one due bucket (ignored unless status is open)"},
				{Name: "terms", Doc: "space-separated search terms matched against title and notes (ANDed)"},
				{Name: "limit", Doc: "maximum cards to draw (default 12, max 50)"},
			},
		},
		{
			Type:  "settings",
			Label: "Settings",
			Icon:  "key",
			W:     6,
			H:     24,
			Params: []ParamSpec{
				{Name: "section", Enum: []string{"profile", "models", "heads", "appearance", "capabilities", "nudges"}, Doc: "settings section (default profile)"},
			},
		},
		{
			Type:  "review",
			Label: "Review",
			Icon:  "key",
			W:     6,
			H:     24,
			// no params — the unified queue of everything awaiting the owner's approval
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

// HasManage reports whether typ accepts mode=manage — i.e. it has a richer,
// self-targeting interactive view that the focus surface should prefer over the
// plain summary tile.
func HasManage(typ string) bool {
	s, ok := byType[typ]
	if !ok {
		return false
	}
	for _, p := range s.Params {
		if p.Name == "mode" {
			return enumContains(p.Enum, "manage")
		}
	}
	return false
}

// Card is a typed, parameterized card reference — the composition unit for
// on-the-spot UI. It matches the JSON shape stored in card markers
// collection's cards field and produced by the board_compose tool.
//
// Layout fields (X, Y, W, H) are optional — zero means "use the spec default".
// omitempty keeps the JSON shape backward-compatible with records that have no
// stored layout.
type Card struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
	X      int               `json:"x,omitempty"` // 0-based col, 0..11
	Y      int               `json:"y,omitempty"` // 0-based row unit
	W      int               `json:"w,omitempty"` // col span 1..12; 0 = spec default
	H      int               `json:"h,omitempty"` // row-unit span; 0 = spec default
}

// ValidateCards validates each entry via Validate and returns the cleaned list.
// The first invalid card stops validation and returns an error.
// Layout fields are clamped to their valid ranges:
//   - X clamped to 0..11
//   - W clamped to 0..12; X+W shrunk to keep it ≤ 12
//   - Y clamped to 0..500
//   - H clamped to 0..120
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
		// Clamp layout fields.
		card.X = clampLayout(c.X, 0, 11)
		card.Y = clampLayout(c.Y, 0, 500)
		card.W = clampLayout(c.W, 0, 12)
		card.H = clampLayout(c.H, 0, 120)
		// Ensure X+W ≤ 12 (shrink W if needed; 0 means "use spec default" so skip).
		if card.W > 0 && card.X+card.W > 12 {
			card.W = max(12-card.X, 1)
		}
		out = append(out, card)
	}
	return out, nil
}

// clampLayout clamps n to [lo, hi]. Zero is a valid value meaning "use default",
// so clamping always preserves zero when lo == 0.
func clampLayout(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
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

		// Free-string params are model-composable; cap them so a tool call
		// cannot bloat stored board JSON. Truncation, not rejection — cards
		// must stay forgiving.
		if len(v) > maxParamLen {
			v = v[:maxParamLen]
		}
		out[ps.Name] = v
	}

	// Drop unknown keys by only copying known ones (already done above).
	// params not in known are silently ignored — they never enter out.
	_ = known // used for correctness reasoning only
	return out, nil
}

func enumContains(enum []string, v string) bool {
	return slices.Contains(enum, v)
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
