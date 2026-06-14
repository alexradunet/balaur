// Package life is the owner-defined tracking infrastructure: free-form
// entry kinds, numeric or textual, logged in conversation and mirrored on
// /life. Balaur does not decide what a life is made of — the owner does;
// this package only provides the verbs. System kinds belong to other
// machinery (completion → task_done, journal → day pages) and are refused
// here.
package life

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/alexradunet/balaur/internal/store"
)

var reserved = map[string]bool{"completion": true, "journal": true}

// NormalizeKind canonicalizes an owner-invented kind: lowercase, trimmed,
// inner whitespace to dashes — "Blood Pressure" and "blood pressure" are
// the same tracker.
func NormalizeKind(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), "-")
}

// LogOpts carries one life-log entry. ValueNum zero means "no numeric
// value" (PB number columns default 0) — a true-zero measurement carries
// its meaning in Text.
type LogOpts struct {
	Kind     string
	ValueNum float64
	Unit     string
	Text     string
	Details  map[string]any
	NotedAt  time.Time
}

// Log stores one entry. The owner's statement is the consent; corrections
// go through Drop.
func Log(app core.App, o LogOpts) (*core.Record, error) {
	kind := NormalizeKind(o.Kind)
	if kind == "" {
		return nil, fmt.Errorf("life: kind is required")
	}
	if reserved[kind] {
		return nil, fmt.Errorf("life: %q is a system kind — completions come from task_done, journal entries from the journal", kind)
	}
	if o.NotedAt.IsZero() {
		o.NotedAt = time.Now()
	}
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return nil, fmt.Errorf("finding entries collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("kind", kind)
	rec.Set("value_num", o.ValueNum)
	rec.Set("unit", strings.ToLower(strings.TrimSpace(o.Unit)))
	rec.Set("text", strings.TrimSpace(o.Text))
	if o.Details != nil {
		rec.Set("value", o.Details)
	}
	rec.Set("noted_at", o.NotedAt.UTC())
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving %s entry: %w", kind, err)
	}
	store.Audit(app, "life", "entry.log", rec.Id, true, map[string]any{"kind": kind})
	return rec, nil
}

// Drop deletes one owner-logged entry (a correction, not a lifecycle).
func Drop(app core.App, id string) (string, error) {
	rec, err := app.FindRecordById("entries", strings.TrimSpace(id))
	if err != nil {
		return "", fmt.Errorf("life: no entry %q — check entry_series for ids", id)
	}
	kind := rec.GetString("kind")
	if reserved[kind] {
		return "", fmt.Errorf("life: %q entries are managed by their own machinery", kind)
	}
	if err := app.Delete(rec); err != nil {
		return "", fmt.Errorf("dropping entry: %w", err)
	}
	store.Audit(app, "life", "entry.drop", id, true, map[string]any{"kind": kind})
	return kind, nil
}

// KindInfo describes one tracker the owner actually uses.
type KindInfo struct {
	Kind     string
	Count    int
	Last     time.Time
	NumCount int // entries carrying a numeric value
	Unit     string
}

// Kinds returns the owner's tracker inventory, most recently used first.
// What exists is what the owner logged — nothing is predefined.
func Kinds(app core.App) ([]KindInfo, error) {
	var rows []struct {
		Kind string `db:"kind"`
		N    int    `db:"n"`
		Last string `db:"last"`
		Num  int    `db:"num"`
		Unit string `db:"unit"`
	}
	err := app.DB().NewQuery(`
		SELECT kind, COUNT(*) AS n, MAX(noted_at) AS last,
		       SUM(CASE WHEN value_num != 0 THEN 1 ELSE 0 END) AS num,
		       COALESCE(MAX(NULLIF(unit, '')), '') AS unit
		FROM entries
		WHERE kind NOT IN ('completion', 'journal')
		GROUP BY kind
		ORDER BY last DESC`).All(&rows)
	if err != nil {
		return nil, fmt.Errorf("listing kinds: %w", err)
	}
	out := make([]KindInfo, 0, len(rows))
	for _, r := range rows {
		info := KindInfo{Kind: r.Kind, Count: r.N, NumCount: r.Num, Unit: r.Unit}
		if t, err := time.Parse(types.DefaultDateLayout, r.Last); err == nil {
			info.Last = t
		}
		out = append(out, info)
	}
	return out, nil
}

// Series returns a kind's entries since a time, oldest first.
func Series(app core.App, kind string, since time.Time) ([]*core.Record, error) {
	return app.FindRecordsByFilter("entries",
		"kind = {:k} && noted_at >= {:since}", "noted_at", 500, 0,
		dbx.Params{"k": NormalizeKind(kind), "since": store.PBTime(since)})
}

// Summary reduces a series' numeric points.
type Summary struct {
	Points                int
	First, Last, Min, Max float64
	LastAt                time.Time
	Unit                  string
}

// Summarize walks a series collecting value_num points (zero = no numeric
// value, see LogOpts).
func Summarize(recs []*core.Record) Summary {
	var s Summary
	for _, r := range recs {
		v := r.GetFloat("value_num")
		if v == 0 {
			continue
		}
		if s.Points == 0 {
			s.First, s.Min, s.Max = v, v, v
		}
		if v < s.Min {
			s.Min = v
		}
		if v > s.Max {
			s.Max = v
		}
		s.Last = v
		s.LastAt = r.GetDateTime("noted_at").Time()
		if u := r.GetString("unit"); u != "" {
			s.Unit = u
		}
		s.Points++
	}
	return s
}
