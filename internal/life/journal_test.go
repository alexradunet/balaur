package life

import (
	"strings"
	"testing"
	"time"

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
	if rec.GetString("kind") != "journal" {
		t.Errorf("kind = %q", rec.GetString("kind"))
	}
	if rec.GetString("text") != text {
		t.Errorf("not verbatim (beyond trim): %q", rec.GetString("text"))
	}
	if rec.GetDateTime("noted_at").IsZero() {
		t.Error("noted_at not defaulted")
	}
}

func TestJournalBackdating(t *testing.T) {
	app := storetest.NewApp(t)
	past := time.Now().AddDate(0, 0, -2)
	rec, err := JournalWrite(app, "a thought for the record", past)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	got := rec.GetDateTime("noted_at").Time()
	if d := got.Sub(past); d > time.Second || d < -time.Second {
		t.Errorf("backdated noted_at = %v, want ~%v", got, past)
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
	if _, err := app.FindRecordById("entries", j.Id); err == nil {
		t.Error("journal entry still exists after drop")
	}

	// A tracker entry is not the journal's to delete.
	w, err := Log(app, LogOpts{Kind: "weight", ValueNum: 82.5})
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if err := JournalDrop(app, w.Id); err == nil || !strings.Contains(err.Error(), "not a journal entry") {
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
