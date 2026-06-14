// Package heads is Balaur's persona roster: the switchable heads of the one
// dragon. A head is a name + purpose (a system-prompt flavor) + Balaur avatar +
// an optional capability-group filter. Built-in heads live in code; the owner
// can add custom heads as rows in the `heads` collection. The active head is an
// owner setting; switching it changes the voice, avatar, and offered tools for
// the single master conversation. It is a capability filter, NOT a security
// boundary — see docs/superpowers/specs/2026-06-14-heads-as-personas-design.md.
package heads

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// MainKey is the built-in main head: the default active head and the fallback.
const MainKey = "balaur"

// activeHeadSetting is the owner_settings key holding the current head id/key.
const activeHeadSetting = "active_head"

// Head is one persona. ID is a built-in key (e.g. "scholar") or a custom record
// id. Groups is the capability-group filter; empty means "all tools". Avatar is
// a balaur-NN key; "" means "use the owner's default Balaur avatar".
type Head struct {
	ID      string
	Name    string
	Purpose string
	Avatar  string
	Groups  []string
	BuiltIn bool
}

// Groups is the set of selectable capability-group keys, in display order.
// They map onto the tool constructors in internal/turn/tools.go.
var Groups = []string{"memory", "tasks", "life", "journal", "os", "extensions"}

// builtins is the fixed roster: the main head plus three specialists. The order
// is the display order in the switcher and the manage card. Purpose is a
// descriptor (framed by turn.go), not a full prompt.
var builtins = []Head{
	{ID: "balaur", Name: "Balaur", Purpose: "", Avatar: "", Groups: nil, BuiltIn: true},
	{ID: "scholar", Name: "Scholar", Purpose: "explains, researches, and weighs trade-offs; precise and cites its reasoning", Avatar: "balaur-04", Groups: []string{"memory"}, BuiltIn: true},
	{ID: "planner", Name: "Planner", Purpose: "turns goals into concrete tasks and next steps; outcome-oriented", Avatar: "balaur-16", Groups: []string{"tasks", "memory"}, BuiltIn: true},
	{ID: "coach", Name: "Coach", Purpose: "holds you accountable and prompts reflection and journaling; warm and direct", Avatar: "balaur-11", Groups: []string{"journal", "life", "memory"}, BuiltIn: true},
}

// Builtins returns the fixed roster.
func Builtins() []Head { return builtins }

// List returns the full roster: built-ins first, then custom heads (oldest
// first). DB errors degrade to just the built-ins.
func List(app core.App) []Head {
	out := make([]Head, 0, len(builtins)+4)
	out = append(out, builtins...)
	if recs, err := app.FindRecordsByFilter("heads", "", "created", 0, 0); err == nil {
		for _, r := range recs {
			out = append(out, headFromRecord(r))
		}
	}
	return out
}

// Find returns the head with the given id/key. ok is false when missing.
func Find(app core.App, id string) (Head, bool) {
	for _, b := range builtins {
		if b.ID == id {
			return b, true
		}
	}
	if id != "" {
		if r, err := app.FindRecordById("heads", id); err == nil {
			return headFromRecord(r), true
		}
	}
	return Head{}, false
}

// Active returns the owner's current head, falling back to the main head when
// the setting is unset or points at a head that no longer exists.
func Active(app core.App) Head {
	id := store.GetOwnerSetting(app, activeHeadSetting, MainKey)
	if h, ok := Find(app, id); ok {
		return h
	}
	main, _ := Find(app, MainKey)
	return main
}

// SetActive persists the active head id/key.
func SetActive(app core.App, id string) error {
	return store.SetOwnerSetting(app, activeHeadSetting, id)
}

// Create adds a custom head and returns its record id.
func Create(app core.App, name, purpose, avatar string, groups []string) (string, error) {
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		return "", err
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("purpose", purpose)
	rec.Set("balaur_avatar", avatar)
	rec.Set("tools", marshalGroups(groups))
	if err := app.Save(rec); err != nil {
		return "", err
	}
	return rec.Id, nil
}

// Delete removes a custom head record. Built-ins (keys, not record ids) never
// reach here — callers gate on BuiltIn first.
func Delete(app core.App, id string) error {
	rec, err := app.FindRecordById("heads", id)
	if err != nil {
		return err
	}
	return app.Delete(rec)
}

func headFromRecord(r *core.Record) Head {
	var groups []string
	if raw := r.GetString("tools"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &groups)
	}
	return Head{
		ID:      r.Id,
		Name:    r.GetString("name"),
		Purpose: r.GetString("purpose"),
		Avatar:  r.GetString("balaur_avatar"),
		Groups:  groups,
		BuiltIn: false,
	}
}

func marshalGroups(groups []string) string {
	if len(groups) == 0 {
		return ""
	}
	b, _ := json.Marshal(groups)
	return string(b)
}
