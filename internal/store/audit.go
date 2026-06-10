// Package store is Balaur's one seam to PocketBase internals shared by
// multiple packages. Keep it small: helpers move here only when a third
// caller appears (suckless: one source of truth per concern).
package store

import "github.com/pocketbase/pocketbase/core"

// Audit appends one row to audit_log. headID may be empty for actions not
// tied to a sub-agent. Auditing must never take the runtime down, so all
// failures are swallowed — this is the only intentionally silent path.
func Audit(app core.App, headID, actor, action, target string, allowed bool, detail map[string]any) {
	col, err := app.FindCollectionByNameOrId("audit_log")
	if err != nil {
		return
	}
	rec := core.NewRecord(col)
	if headID != "" {
		rec.Set("head", headID)
	}
	rec.Set("actor", actor)
	rec.Set("action", action)
	rec.Set("target", target)
	rec.Set("allowed", allowed)
	if detail != nil {
		rec.Set("detail", detail)
	}
	_ = app.Save(rec)
}
