// Package knowledge is Balaur's memory and skill layer: what the companion
// knows about the owner and what it knows how to do.
//
// THE CONSENT BOUNDARY: the model never changes knowledge on its own. Model
// tools create records with status=proposed; only the owner's explicit
// action (approve / edit / dismiss in the UI) activates, changes, or removes
// them. Every transition is audited. This mirrors the heads rule boundary:
// enforcement lives in the data layer, not in the prompt.
package knowledge

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/search"
	"github.com/alexradunet/balaur/internal/store"
)

// Kind selects which collection a lifecycle call operates on.
type Kind string

const (
	Memory Kind = "memories"
	Skill  Kind = "skills"
)

// Statuses (mirrors migrations/1749700000_knowledge.go).
const (
	StatusProposed = "proposed"
	StatusActive   = "active"
	StatusArchived = "archived"
	StatusRejected = "rejected"
)

func clampImportance(n int) int {
	if n < 1 {
		return 1
	}
	if n > 5 {
		return 5
	}
	return n
}

// MemoryProposal is what the model may suggest remembering.
type MemoryProposal struct {
	Title      string
	Content    string
	Category   string // fact | preference | person | project | context
	Importance int    // 1..5
	WhenToUse  string
	Source     string // e.g. "chat", a conversation id, "import"
}

// ProposeMemory stores a proposal awaiting the owner's decision.
func ProposeMemory(app core.App, p MemoryProposal) (*core.Record, error) {
	if strings.TrimSpace(p.Title) == "" {
		return nil, fmt.Errorf("knowledge: memory title is required")
	}
	col, err := app.FindCollectionByNameOrId(string(Memory))
	if err != nil {
		return nil, fmt.Errorf("finding memories collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("title", p.Title)
	rec.Set("content", p.Content)
	rec.Set("category", p.Category)
	rec.Set("importance", clampImportance(p.Importance))
	rec.Set("when_to_use", p.WhenToUse)
	rec.Set("source", p.Source)
	rec.Set("status", StatusProposed)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving memory proposal: %w", err)
	}
	store.Audit(app, "model", "knowledge.propose", "memories/"+rec.Id, true,
		map[string]any{"title": p.Title})
	return rec, nil
}

// SkillProposal is what the model may suggest learning.
type SkillProposal struct {
	Name        string
	Description string
	Content     string // the procedure itself, Markdown
	WhenToUse   string
}

// ProposeSkill stores a skill proposal awaiting the owner's decision.
func ProposeSkill(app core.App, p SkillProposal) (*core.Record, error) {
	if strings.TrimSpace(p.Name) == "" {
		return nil, fmt.Errorf("knowledge: skill name is required")
	}
	col, err := app.FindCollectionByNameOrId(string(Skill))
	if err != nil {
		return nil, fmt.Errorf("finding skills collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", p.Name)
	rec.Set("description", p.Description)
	rec.Set("content", p.Content)
	rec.Set("when_to_use", p.WhenToUse)
	rec.Set("status", StatusProposed)
	rec.Set("enabled", false) // enabled flips on approval
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving skill proposal: %w", err)
	}
	store.Audit(app, "model", "knowledge.propose", "skills/"+rec.Id, true,
		map[string]any{"name": p.Name})
	return rec, nil
}

// validTransitions encodes the owner-driven lifecycle. Key: from → allowed to.
var validTransitions = map[string][]string{
	StatusProposed: {StatusActive, StatusRejected},
	StatusActive:   {StatusArchived},
	StatusArchived: {StatusActive},
	StatusRejected: {},
}

// Transition moves a record to a new status on the owner's behalf. It
// validates the lifecycle, flips skill enablement, and audits.
func Transition(app core.App, kind Kind, id, to string) (*core.Record, error) {
	rec, err := app.FindRecordById(string(kind), id)
	if err != nil {
		return nil, fmt.Errorf("finding %s record: %w", kind, err)
	}
	from := rec.GetString("status")

	allowed := slices.Contains(validTransitions[from], to)
	if !allowed {
		store.Audit(app, "owner", "knowledge."+to, string(kind)+"/"+rec.Id, false,
			map[string]any{"from": from})
		return nil, fmt.Errorf("knowledge: cannot move %s from %q to %q", kind, from, to)
	}

	rec.Set("status", to)
	if kind == Skill {
		rec.Set("enabled", to == StatusActive)
	}
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("updating %s status: %w", kind, err)
	}
	store.Audit(app, "owner", "knowledge."+to, string(kind)+"/"+rec.Id, true,
		map[string]any{"from": from})
	return rec, nil
}

// UpdateFields applies owner edits to a record (from the edit form or an
// edit-then-approve flow). Only whitelisted fields are writable.
func UpdateFields(app core.App, kind Kind, id string, fields map[string]string) (*core.Record, error) {
	rec, err := app.FindRecordById(string(kind), id)
	if err != nil {
		return nil, fmt.Errorf("finding %s record: %w", kind, err)
	}
	writable := map[Kind][]string{
		Memory: {"title", "content", "category", "importance", "when_to_use"},
		Skill:  {"name", "description", "content", "when_to_use"},
	}
	for _, f := range writable[kind] {
		v, ok := fields[f]
		if !ok {
			continue
		}
		if f == "importance" {
			n, err := strconv.Atoi(v)
			if err != nil {
				continue // ignore a malformed importance rather than coercing to 0
			}
			rec.Set(f, clampImportance(n))
			continue
		}
		rec.Set(f, v)
	}
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("updating %s: %w", kind, err)
	}
	store.Audit(app, "owner", "knowledge.edit", string(kind)+"/"+rec.Id, true, nil)
	return rec, nil
}

// Touch records that a piece of knowledge was actually used: bumps
// use_count and last_used. Usage statistics inform the owner's curation
// (and future relevance ranking) — they are not consent-gated because they
// change metadata, not content.
func Touch(app core.App, kind Kind, rec *core.Record) {
	rec.Set("use_count", rec.GetInt("use_count")+1)
	rec.Set("last_used", time.Now().UTC())
	if err := app.Save(rec); err != nil {
		app.Logger().Warn("knowledge touch failed", "kind", string(kind), "id", rec.Id, "err", err)
	}
}

// ListByStatus returns records of one kind in one status, newest first.
func ListByStatus(app core.App, kind Kind, status string) ([]*core.Record, error) {
	return app.FindRecordsByFilter(string(kind),
		"status = {:status}", "-created", 0, 0, dbx.Params{"status": status})
}

// FilterActive narrows active records for the management pages: optional
// substring query across the text fields, optional category (memories only).
// Empty query and category degrade to a plain active listing.
func FilterActive(app core.App, kind Kind, query, category string) ([]*core.Record, error) {
	filter := "status = 'active'"
	params := dbx.Params{}

	if q := strings.TrimSpace(query); q != "" {
		params["q"] = q
		if kind == Memory {
			filter += " && (title ~ {:q} || content ~ {:q} || when_to_use ~ {:q})"
		} else {
			filter += " && (name ~ {:q} || description ~ {:q} || content ~ {:q} || when_to_use ~ {:q})"
		}
	}
	if c := strings.TrimSpace(category); c != "" && kind == Memory {
		filter += " && category = {:cat}"
		params["cat"] = c
	}
	return app.FindRecordsByFilter(string(kind), filter, "-importance,-created", 0, 0, params)
}

// SearchActive finds active memories matching any of the given terms.
// When a FTS5 sidecar index is available in app.Store() (key
// search.StoreKey), results are bm25-ranked by the index and the LIKE path
// is skipped. On any error, a missing index, or zero FTS results, it falls
// through to the plain LIKE body unchanged — deterministic, offline-safe.
func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := ix.Query(terms, limit)
			if err == nil && len(ids) > 0 {
				recs, err := app.FindRecordsByIds(string(Memory), ids)
				if err == nil {
					// Filter defensively: only active (the index may lag briefly).
					var active []*core.Record
					for _, r := range recs {
						if r.GetString("status") == StatusActive {
							active = append(active, r)
						}
					}
					if len(active) > 0 {
						// Preserve FTS rank order (FindRecordsByIds returns unordered).
						order := make(map[string]int, len(ids))
						for i, id := range ids {
							order[id] = i
						}
						sort.Slice(active, func(i, j int) bool {
							return order[active[i].Id] < order[active[j].Id]
						})
						if len(active) > limit {
							active = active[:limit]
						}
						return active, nil
					}
				}
			}
		}
	}

	// --- LIKE fallback (unchanged) ---
	params := dbx.Params{}
	var clauses []string
	for i, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := fmt.Sprintf("q%d", i)
		params[key] = t
		clauses = append(clauses,
			fmt.Sprintf("title ~ {:%[1]s} || content ~ {:%[1]s} || when_to_use ~ {:%[1]s} || category ~ {:%[1]s}", key))
	}
	if len(clauses) == 0 {
		return nil, nil
	}
	return app.FindRecordsByFilter(
		string(Memory),
		"status = 'active' && ("+strings.Join(clauses, " || ")+")",
		"-importance,-created", limit, 0,
		params,
	)
}

// UpfrontMemories returns the highest-importance active memories that are
// always injected into context (tier 1 of the injection policy).
func UpfrontMemories(app core.App, limit int) ([]*core.Record, error) {
	return app.FindRecordsByFilter(string(Memory),
		"status = 'active' && importance >= 4",
		"-importance,-created", limit, 0, nil)
}

// ActiveSkills returns enabled, active skills for the context index.
func ActiveSkills(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter(string(Skill),
		"status = 'active' && enabled = true", "name", 0, 0, nil)
}

// LoadSkill fetches one active skill by exact name and records usage.
func LoadSkill(app core.App, name string) (*core.Record, error) {
	rec, err := app.FindFirstRecordByFilter(string(Skill),
		"status = 'active' && enabled = true && name = {:name}",
		dbx.Params{"name": name})
	if err != nil {
		return nil, fmt.Errorf("skill %q not found or not active", name)
	}
	Touch(app, Skill, rec)
	return rec, nil
}
