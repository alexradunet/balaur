package migrations

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(upNodeTypeIcons, downNodeTypeIcons)
}

// typeIconSeed maps each built-in node type to its graph glyph. The node_types
// registry has carried an (empty) icon field since plan 164; this backfills it so
// the graph can draw a per-type icon. Emoji, not PNG stems: drawn directly on the
// force-graph canvas / as SVG <text>, no new assets, instantly distinct.
var typeIconSeed = map[string]string{
	"note":    "📝",
	"memory":  "🧠",
	"skill":   "🔑",
	"person":  "👤",
	"book":    "📖",
	"idea":    "💡",
	"place":   "📍",
	"task":    "✅",
	"day":     "📅",
	"measure": "📊",
}

// upNodeTypeIcons backfills node_types.icon with a per-type emoji. Idempotent:
// it updates whichever rows exist and skips any type not present, so it is safe
// across installs at different schema points (e.g. journal already retired).
func upNodeTypeIcons(app core.App) error {
	for name, icon := range typeIconSeed {
		row, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": name})
		if err != nil {
			continue // type not registered on this install — nothing to backfill
		}
		row.Set("icon", icon)
		if err := app.Save(row); err != nil {
			return fmt.Errorf("node_type_icons: setting icon for %q: %w", name, err)
		}
	}
	return nil
}

// downNodeTypeIcons clears the icons it set (best-effort; leaves rows intact).
func downNodeTypeIcons(app core.App) error {
	for name := range typeIconSeed {
		row, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": name})
		if err != nil {
			continue
		}
		row.Set("icon", "")
		if err := app.Save(row); err != nil {
			return fmt.Errorf("node_type_icons down: clearing icon for %q: %w", name, err)
		}
	}
	return nil
}
