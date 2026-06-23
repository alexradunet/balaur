package life

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
)

// The journal: the owner's own words, kept as a type=journal node — one node
// per day, born active, verbatim. Deliberately separate from Log — journal is a
// distinct node type, the rules differ (verbatim text, no numbers, no units),
// and deleting a day is the owner's right over their own writing, exercised on
// the day page, never a model verb.

const journalDayKey = "2006-01-02"

// JournalWrite keeps one journal entry, verbatim. Writes for the same day append
// to that day's single journal node (blank-line separated); the first write of a
// day creates it. props.date = YYYY-MM-DD in the owner's location is the day key.
func JournalWrite(app core.App, text string, notedAt time.Time) (*core.Record, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("journal: nothing to keep — the entry is empty")
	}
	if notedAt.IsZero() {
		notedAt = time.Now()
	}
	loc := store.OwnerLocation(app)
	dayKey := notedAt.In(loc).Format(journalDayKey)
	label := notedAt.In(loc).Format("Monday, January 2 2006")

	// Append to the day's node if it already exists.
	if rec, err := app.FindFirstRecordByFilter("nodes",
		"type = 'journal' && status = 'active' && props.date = {:d}",
		dbx.Params{"d": dayKey}); err == nil {
		body := strings.TrimRight(rec.GetString("body"), "\n")
		rec.Set("body", body+"\n\n"+text)
		if err := app.Save(rec); err != nil {
			return nil, fmt.Errorf("appending journal entry: %w", err)
		}
		store.Audit(app, "journal", "journal.write", "nodes/"+rec.Id, true, nil)
		return rec, nil
	}

	rec, err := nodes.Create(app, "journal", label, text, nodes.StatusActive,
		map[string]any{"date": dayKey})
	if err != nil {
		return nil, fmt.Errorf("saving journal entry: %w", err)
	}
	store.Audit(app, "journal", "journal.write", "nodes/"+rec.Id, true, nil)
	return rec, nil
}

// JournalDrop deletes one journal node. Only journal nodes qualify —
// everything else has its own machinery.
func JournalDrop(app core.App, id string) error {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("journal: no entry %q", id)
	}
	if rec.GetString("type") != "journal" {
		return fmt.Errorf("journal: %q is not a journal entry", id)
	}
	if err := app.Delete(rec); err != nil {
		return fmt.Errorf("dropping journal entry: %w", err)
	}
	store.Audit(app, "journal", "journal.drop", "nodes/"+id, true, nil)
	return nil
}
