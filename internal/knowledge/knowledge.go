// Package knowledge is Balaur's memory and skill layer: what the companion
// knows about the owner and what it knows how to do. It is a typed view over
// the unified `nodes` spine (internal/nodes) — a memory is a type=memory node,
// a skill is a type=skill node — preserving the original consent lifecycle.
//
// THE CONSENT BOUNDARY: the model never changes knowledge on its own. Model
// tools create nodes with status=proposed, or — for an existing active node —
// PARK a proposed edit in props (ProposeEdit) without touching the approved
// content; only the owner's explicit action (approve / edit / dismiss in the UI)
// activates, changes, applies, or removes them. Every transition is audited.
// This mirrors the heads rule boundary: enforcement lives in the data layer, not
// in the prompt.
//
// Read paths return *core.Record node rows with the legacy memory/skill field
// names (content, name, description, importance, when_to_use) hydrated as
// read-only aliases over title/body/props, so the card and CLI layers read
// records the same way they always have.
package knowledge

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
)

// Kind is the node TYPE a lifecycle call operates on.
type Kind string

const (
	Memory Kind = "memory"
	Skill  Kind = "skill"
)

// Statuses for nodes (mirrors the nodes.status enum in
// migrations/1749600000_init.go).
const (
	StatusProposed = nodes.StatusProposed
	StatusActive   = nodes.StatusActive
	StatusArchived = nodes.StatusArchived
	StatusRejected = nodes.StatusRejected
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

// Hydrate exposes the legacy-field aliasing for callers that load a node row
// directly (e.g. rendering a proposal card from its node id). It is a no-op for
// kinds other than Memory/Skill.
func Hydrate(kind Kind, rec *core.Record) *core.Record { return hydrate(kind, rec) }

// hydrate aliases the node's title/body/props onto the legacy memory/skill
// field names as read-only raw values, so existing readers (cards, CLI) keep
// calling r.GetString("content") / GetInt("importance") etc. unchanged. These
// custom keys are never persisted — app.Save writes only schema fields.
func hydrate(kind Kind, rec *core.Record) *core.Record {
	if rec == nil {
		return nil
	}
	props := nodes.Props(rec)
	getStr := func(k string) string {
		if s, ok := props[k].(string); ok {
			return s
		}
		return ""
	}
	switch kind {
	case Memory:
		rec.Set("content", rec.GetString("body"))
		rec.Set("when_to_use", getStr("when_to_use"))
		rec.Set("source", getStr("source"))
		rec.Set("importance", nodes.PropInt(rec, "importance"))
		rec.Set("use_count", nodes.PropInt(rec, "use_count"))
	case Skill:
		rec.Set("name", rec.GetString("title"))
		rec.Set("content", rec.GetString("body"))
		rec.Set("description", getStr("description"))
		rec.Set("when_to_use", getStr("when_to_use"))
		rec.Set("use_count", nodes.PropInt(rec, "use_count"))
	}
	return rec
}

func hydrateAll(kind Kind, recs []*core.Record) []*core.Record {
	for _, r := range recs {
		hydrate(kind, r)
	}
	return recs
}

// MemoryProposal is what the model may suggest remembering.
type MemoryProposal struct {
	Title      string
	Content    string
	Importance int // 1..5
	WhenToUse  string
	Source     string // e.g. "chat", a conversation id, "import"
}

// ProposeMemory stores a proposal awaiting the owner's decision.
func ProposeMemory(app core.App, p MemoryProposal) (*core.Record, error) {
	if strings.TrimSpace(p.Title) == "" {
		return nil, fmt.Errorf("knowledge: memory title is required")
	}
	props := map[string]any{
		"importance":  clampImportance(p.Importance),
		"when_to_use": p.WhenToUse,
		"source":      p.Source,
	}
	rec, err := nodes.Create(app, string(Memory), p.Title, p.Content, StatusProposed, props)
	if err != nil {
		return nil, fmt.Errorf("saving memory proposal: %w", err)
	}
	store.Audit(app, "model", "knowledge.propose", "nodes/"+rec.Id, true,
		map[string]any{"title": p.Title})
	return hydrate(Memory, rec), nil
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
	props := map[string]any{
		"description": p.Description,
		"when_to_use": p.WhenToUse,
	}
	rec, err := nodes.Create(app, string(Skill), p.Name, p.Content, StatusProposed, props)
	if err != nil {
		return nil, fmt.Errorf("saving skill proposal: %w", err)
	}
	store.Audit(app, "model", "knowledge.propose", "nodes/"+rec.Id, true,
		map[string]any{"name": p.Name})
	return hydrate(Skill, rec), nil
}

// Transition moves a node to a new status on the owner's behalf. It validates
// the lifecycle and audits.
func Transition(app core.App, kind Kind, id, to string) (*core.Record, error) {
	rec, err := nodes.Transition(app, id, to, "knowledge")
	if err != nil {
		return nil, err
	}
	if kind == Memory || kind == Skill {
		invalidateContextCache(app)
	}
	return hydrate(kind, rec), nil
}

// UpdateFields applies owner edits to a node (from the edit form or an
// edit-then-approve flow). Only whitelisted fields are writable.
func UpdateFields(app core.App, kind Kind, id string, fields map[string]string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding %s record: %w", kind, err)
	}
	props := nodes.Props(rec)
	writable := map[Kind][]string{
		Memory: {"title", "content", "importance", "when_to_use"},
		Skill:  {"name", "description", "content", "when_to_use"},
	}
	for _, f := range writable[kind] {
		v, ok := fields[f]
		if !ok {
			continue
		}
		switch f {
		case "title", "name": // both map to the node title
			rec.Set("title", v)
		case "content": // maps to the node body
			rec.Set("body", v)
		case "importance":
			n, err := strconv.Atoi(v)
			if err != nil {
				continue // ignore a malformed importance rather than coercing to 0
			}
			props["importance"] = clampImportance(n)
		default: // description, when_to_use → props
			props[f] = v
		}
	}
	rec.Set("props", props)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("updating %s: %w", kind, err)
	}
	store.Audit(app, "owner", "knowledge.edit", "nodes/"+rec.Id, true, nil)
	// An edit to title/body/importance changes the rendered upfront line or the
	// importance>=4 membership; drop the cache so the next turn recomputes.
	if kind == Memory || kind == Skill {
		invalidateContextCache(app)
	}
	return hydrate(kind, rec), nil
}

// Touch records that a piece of knowledge was actually used: bumps use_count
// (in props) and last_used. Not consent-gated — it changes metadata, not
// content. rec must be a node record.
func Touch(app core.App, kind Kind, rec *core.Record) {
	props := nodes.Props(rec)
	props["use_count"] = nodes.PropInt(rec, "use_count") + 1
	props["last_used"] = time.Now().UTC().Format(time.RFC3339)
	rec.Set("props", props)
	if err := app.Save(rec); err != nil {
		app.Logger().Warn("knowledge touch failed", "kind", string(kind), "id", rec.Id, "err", err)
		return
	}
	hydrate(kind, rec)
}

// ListByStatus returns records of one kind in one status, newest first.
func ListByStatus(app core.App, kind Kind, status string) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, string(kind), status)
	if err != nil {
		return nil, err
	}
	return hydrateAll(kind, recs), nil
}

// FilterActive narrows active records for the management pages by an optional
// substring query.
func FilterActive(app core.App, kind Kind, query string) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, string(kind), StatusActive)
	if err != nil {
		return nil, err
	}
	hydrateAll(kind, recs)

	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]*core.Record, 0, len(recs))
	for _, r := range recs {
		if q != "" && !matchesQuery(kind, r, q) {
			continue
		}
		out = append(out, r)
	}
	// Memories order by importance desc, then newest; skills keep newest-first.
	if kind == Memory {
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].GetInt("importance") > out[j].GetInt("importance")
		})
	}
	return out, nil
}

func matchesQuery(kind Kind, r *core.Record, q string) bool {
	fields := []string{r.GetString("title"), r.GetString("content"), r.GetString("when_to_use")}
	if kind == Skill {
		fields = append(fields, r.GetString("description"))
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

// UpfrontMemories returns the highest-importance active memories that are always
// injected into context (tier 1 of the injection policy). Served from the
// process-memory context cache (see cache.go) so a warm turn does not re-scan the
// nodes collection. The returned records are FRESH copies (copyForRead): callers
// may Touch them after a turn without mutating the shared cached snapshot.
func UpfrontMemories(app core.App, limit int) ([]*core.Record, error) {
	c, err := loadContextCache(app)
	if err != nil {
		return nil, err
	}
	out := c.upfront
	if limit > 0 && len(out) > limit {
		out = out[:limit] // cached slice is already importance-sorted
	}
	return copyForRead(out), nil
}

// ActiveSkills returns active skills for the context index, ordered by name.
// Served from the context cache (see cache.go); returns fresh copies so a
// caller's mutation never reaches the cached snapshot.
func ActiveSkills(app core.App) ([]*core.Record, error) {
	c, err := loadContextCache(app)
	if err != nil {
		return nil, err
	}
	return copyForRead(c.skills), nil
}

// LoadSkill fetches the first active skill whose title matches name (exact) and
// records usage. Skill name uniqueness is enforced in Go (nodes have no
// per-type unique index).
func LoadSkill(app core.App, name string) (*core.Record, error) {
	rec, err := app.FindFirstRecordByFilter("nodes",
		"type = {:t} && status = {:s} && title = {:name}",
		dbx.Params{"t": string(Skill), "s": StatusActive, "name": name})
	if err != nil {
		return nil, fmt.Errorf("skill %q not found or not active", name)
	}
	hydrate(Skill, rec)
	Touch(app, Skill, rec)
	return rec, nil
}
