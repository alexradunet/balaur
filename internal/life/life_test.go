package life

import (
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestNormalizeKind(t *testing.T) {
	for in, want := range map[string]string{
		"Weight":            "weight",
		"  Blood Pressure ": "blood-pressure",
		"pages   read":      "pages-read",
		"":                  "",
	} {
		if got := NormalizeKind(in); got != want {
			t.Errorf("NormalizeKind(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLogValidationAndRoundtrip(t *testing.T) {
	app := storetest.NewApp(t)

	if _, err := Log(app, LogOpts{Kind: "  "}); err == nil {
		t.Error("empty kind: want error")
	}
	for _, k := range []string{"completion", "Journal"} {
		if _, err := Log(app, LogOpts{Kind: k, Text: "x"}); err == nil {
			t.Errorf("reserved kind %q: want error", k)
		}
	}

	rec, err := Log(app, LogOpts{Kind: "Weight", ValueNum: 82.5, Unit: "KG", Text: "morning"})
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if rec.GetString("kind") != "weight" || rec.GetString("unit") != "kg" {
		t.Errorf("normalization: kind=%q unit=%q", rec.GetString("kind"), rec.GetString("unit"))
	}
	if rec.GetFloat("value_num") != 82.5 {
		t.Errorf("value_num = %v", rec.GetFloat("value_num"))
	}
	if rec.GetDateTime("noted_at").IsZero() {
		t.Error("noted_at not defaulted")
	}
}

func TestLogBackdating(t *testing.T) {
	app := storetest.NewApp(t)
	past := time.Now().AddDate(0, 0, -3)
	rec, err := Log(app, LogOpts{Kind: "mood", ValueNum: 7, NotedAt: past})
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	got := rec.GetDateTime("noted_at").Time()
	if d := got.Sub(past); d > time.Second || d < -time.Second {
		t.Errorf("backdated noted_at = %v, want ~%v", got, past)
	}
}

func TestKindsInventory(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Log(app, LogOpts{Kind: "weight", ValueNum: 82.5, Unit: "kg"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Log(app, LogOpts{Kind: "weight", ValueNum: 82.1, Unit: "kg"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Log(app, LogOpts{Kind: "gratitude", Text: "the morning was quiet"}); err != nil {
		t.Fatal(err)
	}

	kinds, err := Kinds(app)
	if err != nil {
		t.Fatalf("kinds: %v", err)
	}
	if len(kinds) != 2 {
		t.Fatalf("kinds = %d, want 2", len(kinds))
	}
	byName := map[string]KindInfo{}
	for _, k := range kinds {
		byName[k.Kind] = k
	}
	w := byName["weight"]
	if w.Count != 2 || w.NumCount != 2 || w.Unit != "kg" {
		t.Errorf("weight info = %+v", w)
	}
	g := byName["gratitude"]
	if g.Count != 1 || g.NumCount != 0 {
		t.Errorf("gratitude info = %+v", g)
	}
}

func TestSeriesAndSummarize(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()
	for i, v := range []float64{83.0, 82.6, 82.5} {
		if _, err := Log(app, LogOpts{Kind: "weight", ValueNum: v, Unit: "kg", NotedAt: now.AddDate(0, 0, i-3)}); err != nil {
			t.Fatal(err)
		}
	}
	// A text-only entry must not pollute the numeric summary.
	if _, err := Log(app, LogOpts{Kind: "weight", Text: "skipped the scale", NotedAt: now}); err != nil {
		t.Fatal(err)
	}

	recs, err := Series(app, "WEIGHT", now.AddDate(0, 0, -10))
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(recs) != 4 {
		t.Fatalf("series rows = %d, want 4", len(recs))
	}
	s := Summarize(recs)
	if s.Points != 3 || s.First != 83.0 || s.Last != 82.5 || s.Min != 82.5 || s.Max != 83.0 || s.Unit != "kg" {
		t.Errorf("summary = %+v", s)
	}
}

func TestDrop(t *testing.T) {
	app := storetest.NewApp(t)
	rec, err := Log(app, LogOpts{Kind: "mood", ValueNum: 7})
	if err != nil {
		t.Fatal(err)
	}
	kind, err := Drop(app, rec.Id)
	if err != nil || kind != "mood" {
		t.Fatalf("drop: kind=%q err=%v", kind, err)
	}
	if _, err := app.FindRecordById("nodes", rec.Id); err == nil {
		t.Error("measure node still exists after drop")
	}
	if _, err := Drop(app, "nope"); err == nil || !strings.Contains(err.Error(), "no entry") {
		t.Errorf("missing id: %v", err)
	}
}
