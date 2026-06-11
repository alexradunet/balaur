package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Owner-defined tracking: the entries kind enum becomes free text — Balaur
// provides the infrastructure, the owner decides what a life is made of
// (weight, mood, sleep, pages-read…). Reserved system kinds stay enforced
// in code (internal/life), not schema: completion (habit machinery) and
// journal (day pages). value_num + unit let any numeric metric chart
// without ceremony; value JSON keeps structured extras.
//
// Swapping a field type drops its column, so existing kind values are
// held in memory across the swap and written back.
func init() {
	m.Register(trackersUp, trackersDown)
}

func trackersUp(app core.App) error {
	return swapEntriesKind(app, func(entries *core.Collection) {
		entries.Fields.Add(
			&core.TextField{Name: "kind", Required: true, Max: 60},
			&core.NumberField{Name: "value_num"},
			&core.TextField{Name: "unit", Max: 20},
		)
	})
}

func trackersDown(app core.App) error {
	return swapEntriesKind(app, func(entries *core.Collection) {
		entries.Fields.RemoveByName("value_num")
		entries.Fields.RemoveByName("unit")
		// Down is best-effort: owner-invented kinds collapse to "note".
		entries.Fields.Add(&core.SelectField{Name: "kind", Required: true, Values: entryKinds})
	})
}

// swapEntriesKind preserves kind values across a field-type swap. mutate
// removes/adds fields; the kind column is re-filled afterwards (unknown
// values fall back to "note" so the down migration stays valid).
func swapEntriesKind(app core.App, mutate func(*core.Collection)) error {
	entries, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return err
	}
	recs, err := app.FindAllRecords("entries")
	if err != nil {
		return err
	}
	saved := make(map[string]string, len(recs))
	for _, r := range recs {
		saved[r.Id] = r.GetString("kind")
	}

	entries.Fields.RemoveByName("kind")
	mutate(entries)
	if err := app.Save(entries); err != nil {
		return err
	}

	known := map[string]bool{}
	for _, k := range entryKinds {
		known[k] = true
	}
	isSelect := entries.Fields.GetByName("kind").Type() == core.FieldTypeSelect
	for id, kind := range saved {
		rec, err := app.FindRecordById("entries", id)
		if err != nil {
			continue
		}
		if isSelect && !known[kind] {
			kind = "note"
		}
		rec.Set("kind", kind)
		if err := app.Save(rec); err != nil {
			return err
		}
	}
	return nil
}
