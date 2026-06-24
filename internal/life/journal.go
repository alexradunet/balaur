package life

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
)

// The journal: the owner's own words, kept as the body of the type=day node
// (plan 171). One day node per calendar date is both the journal page and the
// on_day hub. Writes for the same day append to that node's body
// (blank-line separated). type=journal is retired.

// JournalWrite keeps one journal entry, verbatim. Writes for the same day
// append to the day node's body (blank-line separated); the first write of a
// day creates it. The day node is resolved via nodes.DayNode, keyed on
// props.date = YYYY-MM-DD in the owner's location.
func JournalWrite(app core.App, text string, notedAt time.Time) (*core.Record, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("journal: nothing to keep — the entry is empty")
	}
	if notedAt.IsZero() {
		notedAt = time.Now()
	}

	rec, err := nodes.DayNode(app, notedAt)
	if err != nil {
		return nil, fmt.Errorf("journal: resolving day node: %w", err)
	}

	body := strings.TrimRight(rec.GetString("body"), "\n")
	if body == "" {
		body = text
	} else {
		body = body + "\n\n" + text
	}
	rec.Set("body", body)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("appending journal entry: %w", err)
	}
	store.Audit(app, "journal", "journal.write", "nodes/"+rec.Id, true, nil)
	return rec, nil
}

// JournalDrop clears the journal body of a day node. It does NOT delete the
// node — the day node is the on_day hub and deleting it would orphan all
// on_day edges for that date. Only type=day nodes qualify.
func JournalDrop(app core.App, id string) error {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("journal: no entry %q", id)
	}
	if rec.GetString("type") != "day" {
		return fmt.Errorf("journal: %q is not a journal entry", id)
	}
	rec.Set("body", "")
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("clearing journal entry: %w", err)
	}
	store.Audit(app, "journal", "journal.drop", "nodes/"+id, true, nil)
	return nil
}
