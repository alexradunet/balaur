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

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
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

// fmtTime formats a time as the PocketBase date string "2006-01-02 15:04:05.000Z".
// Stored in props.noted_at so hydrated rec.GetDateTime("noted_at") works.
func fmtTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.000Z")
}

// Log stores one entry as a type=measure node. The owner's statement is the
// consent; corrections go through Drop.
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

	props := map[string]any{
		"kind":     kind,
		"noted_at": fmtTime(o.NotedAt.UTC()),
	}
	if o.ValueNum != 0 {
		props["value_num"] = o.ValueNum
	}
	if u := strings.ToLower(strings.TrimSpace(o.Unit)); u != "" {
		props["unit"] = u
	}
	// Merge extra details (e.g. {"seed":true}).
	for k, v := range o.Details {
		if _, exists := props[k]; !exists {
			props[k] = v
		}
	}

	// Title: "<kind> <date>" e.g. "weight 2026-06-24"
	title := kind + " " + o.NotedAt.UTC().Format("2006-01-02")
	body := strings.TrimSpace(o.Text)

	rec, err := nodes.Create(app, "measure", title, body, nodes.StatusActive, props)
	if err != nil {
		return nil, fmt.Errorf("saving %s measure: %w", kind, err)
	}
	hydrate(rec)
	store.Audit(app, "life", "life.log", rec.Id, true, map[string]any{"kind": kind})
	return rec, nil
}

// Drop deletes one owner-logged measure (a correction, not a lifecycle).
func Drop(app core.App, id string) (string, error) {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return "", fmt.Errorf("life: no entry %q — check entry_series for ids", id)
	}
	if rec.GetString("type") != "measure" {
		return "", fmt.Errorf("life: %q is not a measure node", id)
	}
	kind := nodes.PropString(rec, "kind")
	if reserved[kind] {
		return "", fmt.Errorf("life: %q entries are managed by their own machinery", kind)
	}
	if err := app.Delete(rec); err != nil {
		return "", fmt.Errorf("dropping measure: %w", err)
	}
	store.Audit(app, "life", "life.drop", id, true, map[string]any{"kind": kind})
	return kind, nil
}

// hydrate sets legacy field aliases on a measure node so callers can use
// rec.GetString("kind"), rec.GetFloat("value_num"), rec.GetDateTime("noted_at"), etc.
// without knowing the node storage shape. Uses SetRaw to bypass schema validation —
// these are ephemeral read-only aliases.
func hydrate(rec *core.Record) {
	props := nodes.Props(rec)
	getString := func(key string) string {
		if v, ok := props[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	getFloat := func(key string) float64 {
		if v, ok := props[key]; ok {
			if f, ok := v.(float64); ok {
				return f
			}
		}
		return 0
	}

	rec.SetRaw("kind", getString("kind"))
	rec.SetRaw("value_num", getFloat("value_num"))
	rec.SetRaw("unit", getString("unit"))
	// noted_at stored as PB datetime string; SetRaw so GetDateTime("noted_at") works.
	rec.SetRaw("noted_at", getString("noted_at"))
	rec.SetRaw("text", rec.GetString("body"))
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
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "-created", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("listing kinds: %w", err)
	}

	type kindAgg struct {
		count    int
		numCount int
		unit     string
		last     time.Time
	}
	agg := map[string]*kindAgg{}
	for _, r := range recs {
		hydrate(r)
		kind := r.GetString("kind")
		if kind == "" {
			continue
		}
		a, ok := agg[kind]
		if !ok {
			a = &kindAgg{}
			agg[kind] = a
		}
		a.count++
		if r.GetFloat("value_num") != 0 {
			a.numCount++
			if u := r.GetString("unit"); u != "" {
				a.unit = u
			}
		}
		if t := r.GetDateTime("noted_at").Time(); !t.IsZero() && t.After(a.last) {
			a.last = t
		}
	}

	out := make([]KindInfo, 0, len(agg))
	for kind, a := range agg {
		out = append(out, KindInfo{
			Kind:     kind,
			Count:    a.count,
			NumCount: a.numCount,
			Unit:     a.unit,
			Last:     a.last,
		})
	}
	// Sort most recently used first.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Last.After(out[i].Last) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// Series returns a kind's entries since a time, oldest first.
func Series(app core.App, kind string, since time.Time) ([]*core.Record, error) {
	k := NormalizeKind(kind)
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "created", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("loading measure series: %w", err)
	}
	out := make([]*core.Record, 0)
	for _, r := range recs {
		if nodes.PropString(r, "kind") != k {
			continue
		}
		notedAtStr := nodes.PropString(r, "noted_at")
		if notedAtStr == "" {
			continue
		}
		notedAt, err := time.Parse("2006-01-02 15:04:05.000Z", notedAtStr)
		if err != nil {
			continue
		}
		if notedAt.Before(since) {
			continue
		}
		hydrate(r)
		out = append(out, r)
	}
	return out, nil
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

// listMeasuresInRange loads active type=measure nodes whose noted_at falls in
// [start, end), hydrated. Used by Day.
func listMeasuresInRange(app core.App, start, end time.Time) ([]*core.Record, error) {
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "created", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("loading measures for range: %w", err)
	}
	// Filter by noted_at in Go since noted_at is stored in props (JSON), not a
	// top-level DateField — PocketBase filter cannot reach inside JSON easily.
	out := make([]*core.Record, 0)
	for _, r := range recs {
		notedAtStr := nodes.PropString(r, "noted_at")
		if notedAtStr == "" {
			continue
		}
		notedAt, err := time.Parse("2006-01-02 15:04:05.000Z", notedAtStr)
		if err != nil {
			continue
		}
		if notedAt.Before(start) || !notedAt.Before(end) {
			continue
		}
		hydrate(r)
		out = append(out, r)
	}
	return out, nil
}
