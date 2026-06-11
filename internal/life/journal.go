package life

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// The journal: the owner's own words, kind=journal in the entries log.
// Deliberately separate from Log — journal is reserved there, the rules
// differ (verbatim text, no numbers, no units), and deleting an entry is
// the owner's right over their own writing, exercised on the day page,
// never a model verb.

// JournalWrite keeps one journal entry, verbatim.
func JournalWrite(app core.App, text string, notedAt time.Time) (*core.Record, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("journal: nothing to keep — the entry is empty")
	}
	if notedAt.IsZero() {
		notedAt = time.Now()
	}
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return nil, fmt.Errorf("finding entries collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("kind", "journal")
	rec.Set("text", text)
	rec.Set("noted_at", notedAt.UTC())
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving journal entry: %w", err)
	}
	store.Audit(app, "", "journal", "journal.write", rec.Id, true, nil)
	return rec, nil
}

// JournalDrop deletes one journal entry. Only journal entries qualify —
// everything else has its own machinery.
func JournalDrop(app core.App, id string) error {
	rec, err := app.FindRecordById("entries", strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("journal: no entry %q", id)
	}
	if rec.GetString("kind") != "journal" {
		return fmt.Errorf("journal: %q is not a journal entry", id)
	}
	if err := app.Delete(rec); err != nil {
		return fmt.Errorf("dropping journal entry: %w", err)
	}
	store.Audit(app, "", "journal", "journal.drop", id, true, nil)
	return nil
}
