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
	if rec.GetString("type") != "journal" {
		t.Errorf("type = %q", rec.GetString("type"))
	}
	if rec.GetString("status") != nodes.StatusActive {
		t.Errorf("journal node must be born active, got %q", rec.GetString("status"))
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
	// Same node, appended verbatim.
	if second.Id != first.Id {
		t.Fatalf("second write created a new node (%q != %q); want one per day", second.Id, first.Id)
	}
	body := second.GetString("body")
	if !strings.Contains(body, "first thought") || !strings.Contains(body, "second thought") {
		t.Errorf("appended body lost content: %q", body)
	}
	if n, _ := app.CountRecords("nodes"); n != 1 {
		t.Errorf("journal node count = %d, want 1", n)
	}

	// Day() returns the day's journal node.
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	dd, err := Day(app, "", d)
	if err != nil {
		t.Fatalf("Day: %v", err)
	}
	if len(dd.Journal) != 1 || dd.Journal[0].Id != first.Id {
		t.Errorf("Day().Journal = %d entries, want the day's journal node", len(dd.Journal))
	}
}

func TestJournalDropOnlyJournal(t *testing.T) {
	app := storetest.NewApp(t)

	j, err := JournalWrite(app, "delete me", time.Time{})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := JournalDrop(app, j.Id); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if _, err := app.FindRecordById("nodes", j.Id); err == nil {
		t.Error("journal node still exists after drop")
	}

	// A non-journal node is not the journal's to delete.
	note, err := nodes.Create(app, "note", "a note", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if err := JournalDrop(app, note.Id); err == nil || !strings.Contains(err.Error(), "not a journal entry") {
		t.Errorf("non-journal drop: %v", err)
	}
	if err := JournalDrop(app, "missing"); err == nil {
		t.Error("missing id: want error")
	}
}

func TestLogStillRefusesJournalKind(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Log(app, LogOpts{Kind: "journal", Text: "sneaky"}); err == nil {
		t.Error("Log must refuse the journal kind — journal_write owns it")
	}
}
