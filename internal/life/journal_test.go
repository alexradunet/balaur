package life

import (
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestJournalWriteVerbatim(t *testing.T) {
	app := storetest.NewApp(t)

	if _, err := JournalWrite(app, "   ", time.Time{}); err == nil {
		t.Error("empty journal entry: want error")
	}

	text := "The notary went fine.\nAndreea laughed at my tie."
	rec, err := JournalWrite(app, "  "+text+"  ", time.Time{})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	// plan 171: JournalWrite writes to the day node (type=day), not type=journal.
	if rec.GetString("type") != "day" {
		t.Errorf("type = %q, want day", rec.GetString("type"))
	}
	if rec.GetString("status") != nodes.StatusActive {
		t.Errorf("day node must be born active, got %q", rec.GetString("status"))
	}
	if rec.GetString("body") != text {
		t.Errorf("not verbatim (beyond trim): %q", rec.GetString("body"))
	}
}

func TestJournalOnePerDayAppends(t *testing.T) {
	app := storetest.NewApp(t)
	day := time.Now()

	first, err := JournalWrite(app, "first thought", day)
	if err != nil {
		t.Fatalf("write 1: %v", err)
	}
	second, err := JournalWrite(app, "second thought", day)
	if err != nil {
		t.Fatalf("write 2: %v", err)
	}
	// Same node (the day node), appended verbatim.
	if second.Id != first.Id {
		t.Fatalf("second write created a new node (%q != %q); want one per day", second.Id, first.Id)
	}
	body := second.GetString("body")
	if !strings.Contains(body, "first thought") || !strings.Contains(body, "second thought") {
		t.Errorf("appended body lost content: %q", body)
	}

	// Day() returns the day's journal node (now the day node).
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	dd, err := Day(app, "", d)
	if err != nil {
		t.Fatalf("Day: %v", err)
	}
	if len(dd.Journal) != 1 || dd.Journal[0].Id != first.Id {
		t.Errorf("Day().Journal = %d entries, want the day's day node", len(dd.Journal))
	}
}

// TestJournalDropClearsBody: JournalDrop clears the body WITHOUT deleting the
// node (the day node is the on_day hub; deleting it would orphan on_day edges).
func TestJournalDropClearsBody(t *testing.T) {
	app := storetest.NewApp(t)

	j, err := JournalWrite(app, "delete me", time.Time{})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	nodeID := j.Id

	if err := JournalDrop(app, j.Id); err != nil {
		t.Fatalf("drop: %v", err)
	}

	// Node must still exist (not deleted).
	rec, err := app.FindRecordById("nodes", nodeID)
	if err != nil {
		t.Fatal("day node was deleted after drop; must survive as on_day hub")
	}
	// Body must be cleared.
	if rec.GetString("body") != "" {
		t.Errorf("body after drop = %q, want empty", rec.GetString("body"))
	}
}

// TestJournalDropOnlyDay: non-day nodes must be rejected.
func TestJournalDropOnlyDay(t *testing.T) {
	app := storetest.NewApp(t)

	// A non-day node is not the journal's to clear.
	note, err := nodes.Create(app, "note", "a note", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if err := JournalDrop(app, note.Id); err == nil || !strings.Contains(err.Error(), "not a journal entry") {
		t.Errorf("non-day drop: %v", err)
	}
	if err := JournalDrop(app, "missing"); err == nil {
		t.Error("missing id: want error")
	}
}

// TestJournalAndDayNodeSame: unification proof — a journal write and a
// LinkOnDay call for the same date both resolve to the same type=day node.
func TestJournalAndDayNodeSame(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	// Write a journal entry — this creates or updates the day node.
	j, err := JournalWrite(app, "some thoughts", now)
	if err != nil {
		t.Fatalf("JournalWrite: %v", err)
	}

	// Create a measure and link it to the day.
	measure, err := nodes.Create(app, "note", "a note", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if err := nodes.LinkOnDay(app, measure); err != nil {
		t.Fatalf("LinkOnDay: %v", err)
	}

	// The day node resolved by DayNode must be the same as what JournalWrite returned.
	dayNode, err := nodes.DayNode(app, now)
	if err != nil {
		t.Fatalf("DayNode: %v", err)
	}
	if dayNode.Id != j.Id {
		t.Errorf("journal day node %q != DayNode %q; want single resolver", j.Id, dayNode.Id)
	}
}

func TestLogStillRefusesJournalKind(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Log(app, LogOpts{Kind: "journal", Text: "sneaky"}); err == nil {
		t.Error("Log must refuse the journal kind — journal_write owns it")
	}
}
