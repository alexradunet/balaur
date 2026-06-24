package nodes

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// DefaultTypeIcon is drawn for a node whose type has no icon registered (a
// custom type the owner added without one). A neutral dot, never blank.
const DefaultTypeIcon = "•"

// TypeIcons returns a name→icon map from the node_types registry — the single
// source of truth for the per-type glyph drawn in the graph. Types with an empty
// icon are omitted; callers fall back to DefaultTypeIcon.
func TypeIcons(app core.App) (map[string]string, error) {
	recs, err := app.FindRecordsByFilter("node_types", "", "name", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("node_types: listing icons: %w", err)
	}
	icons := make(map[string]string, len(recs))
	for _, r := range recs {
		if icon := r.GetString("icon"); icon != "" {
			icons[r.GetString("name")] = icon
		}
	}
	return icons, nil
}

// TypeNames returns all registered type names from the node_types registry.
func TypeNames(app core.App) ([]string, error) {
	recs, err := app.FindRecordsByFilter("node_types", "", "name", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("node_types: listing types: %w", err)
	}
	names := make([]string, 0, len(recs))
	for _, r := range recs {
		names = append(names, r.GetString("name"))
	}
	return names, nil
}

// TypeExists reports whether name is registered in node_types.
func TypeExists(app core.App, name string) (bool, error) {
	_, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": name})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("node_types: checking %q: %w", name, err)
	}
	return true, nil
}

// BornStatus returns the consent default for the given type ("active" or
// "proposed"). Returns StatusActive when the type is not found.
func BornStatus(app core.App, typ string) (string, error) {
	rec, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": typ})
	if err != nil {
		if isNotFound(err) {
			return StatusActive, nil
		}
		return "", fmt.Errorf("node_types: born_status for %q: %w", typ, err)
	}
	return rec.GetString("born_status"), nil
}

// OwnerAuthoredTypes returns type names whose born_status is "active" — the
// set the AI may create directly via node_write (consent-gated types like
// memory and skill are excluded; those go through remember/propose_skill).
func OwnerAuthoredTypes(app core.App) ([]string, error) {
	recs, err := app.FindRecordsByFilter("node_types", "born_status = 'active'", "name", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("node_types: listing owner-authored types: %w", err)
	}
	names := make([]string, 0, len(recs))
	for _, r := range recs {
		names = append(names, r.GetString("name"))
	}
	return names, nil
}

// isNotFound reports whether an error from a PocketBase Find* call means
// "record not found" (as opposed to a real database error).
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// PocketBase returns sql.ErrNoRows-wrapped errors for not-found lookups.
	return err.Error() == "sql: no rows in result set"
}
