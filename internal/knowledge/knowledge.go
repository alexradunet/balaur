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
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/search"
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

// validTransitions encodes the owner-driven lifecycle. Key: from → allowed to.
// Sourced from internal/nodes (one source of truth).
var validTransitions = nodes.ValidTransitions

// Transition moves a node to a new status on the owner's behalf. It validates
// the lifecycle and audits.
func Transition(app core.App, kind Kind, id, to string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding %s record: %w", kind, err)
	}
	from := rec.GetString("status")

	allowed := slices.Contains(validTransitions[from], to)
	if !allowed {
		store.Audit(app, "owner", "knowledge."+to, "nodes/"+rec.Id, false,
			map[string]any{"from": from})
		return nil, fmt.Errorf("knowledge: cannot move %s from %q to %q", kind, from, to)
	}

	rec.Set("status", to)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("updating %s status: %w", kind, err)
	}
	store.Audit(app, "owner", "knowledge."+to, "nodes/"+rec.Id, true,
		map[string]any{"from": from})
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
	return hydrate(kind, rec), nil
}

// pendingEditKey is the props key under which a model-proposed edit to an ACTIVE
// node is parked. The active node's approved title/body/metadata stay exactly as
// the owner left them until the owner approves the edit — the model proposes a
// change, it never applies one (the consent boundary in the package doc holds).
const pendingEditKey = "pending_edit"

// ProposeEdit parks a model-proposed change on an active memory/skill without
// applying it: the proposed fields (and/or an archive flag) live in the node's
// props until the owner approves or declines in the review queue. It overwrites
// any prior pending edit (latest proposal wins) and audits actor=model. The
// node's approved content is untouched.
func ProposeEdit(app core.App, id string, fields map[string]string, archive bool) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding node: %w", err)
	}
	if rec.GetString("status") != StatusActive {
		return nil, fmt.Errorf("knowledge: can only propose edits to active knowledge")
	}
	if len(fields) == 0 && !archive {
		return nil, fmt.Errorf("knowledge: nothing to propose")
	}
	props := nodes.Props(rec)
	env := map[string]any{
		"by":      "model",
		"at":      time.Now().UTC().Format(time.RFC3339),
		"archive": archive,
	}
	if len(fields) > 0 {
		fm := make(map[string]any, len(fields))
		for k, v := range fields {
			fm[k] = v
		}
		env["fields"] = fm
	}
	props[pendingEditKey] = env
	rec.Set("props", props)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("parking edit proposal: %w", err)
	}
	store.Audit(app, "model", "knowledge.propose_edit", "nodes/"+rec.Id, true,
		map[string]any{"archive": archive})
	return rec, nil
}

// PendingEdit returns the parked edit proposal on a node, if any. fields are the
// proposed field→value pairs (UpdateFields whitelist keys); archive is true when
// the proposal is to archive rather than edit.
func PendingEdit(rec *core.Record) (fields map[string]string, archive bool, ok bool) {
	raw, has := nodes.Props(rec)[pendingEditKey]
	if !has {
		return nil, false, false
	}
	m, isMap := raw.(map[string]any)
	if !isMap {
		return nil, false, false
	}
	archive, _ = m["archive"].(bool)
	fields = map[string]string{}
	if fm, isFM := m["fields"].(map[string]any); isFM {
		for k, v := range fm {
			if s, isStr := v.(string); isStr {
				fields[k] = s
			}
		}
	}
	return fields, archive, true
}

// PendingEdits returns active memory and skill nodes carrying a model-proposed
// edit awaiting the owner's decision, hydrated with the legacy field aliases.
func PendingEdits(app core.App) ([]*core.Record, error) {
	var out []*core.Record
	for _, k := range []Kind{Memory, Skill} {
		recs, err := nodes.ListByTypeStatus(app, string(k), StatusActive)
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if _, _, ok := PendingEdit(r); ok {
				out = append(out, hydrate(k, r))
			}
		}
	}
	return out, nil
}

// ApplyEdit approves a parked edit on the owner's behalf: it applies the proposed
// fields (via UpdateFields) or archives the node (via Transition), then clears
// the envelope. Those calls audit as actor=owner — the owner is the one who
// activated the change. Returns the hydrated, updated record.
func ApplyEdit(app core.App, id string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding node: %w", err)
	}
	fields, archive, ok := PendingEdit(rec)
	if !ok {
		return nil, fmt.Errorf("knowledge: no pending edit on %s", id)
	}
	kind := Kind(rec.GetString("type"))
	switch {
	case archive:
		if _, err := Transition(app, kind, id, StatusArchived); err != nil {
			return nil, err
		}
	case len(fields) > 0:
		if _, err := UpdateFields(app, kind, id, fields); err != nil {
			return nil, err
		}
	}
	return clearPendingEdit(app, kind, id)
}

// DeclineEdit drops a parked edit without applying it, auditing actor=owner.
func DeclineEdit(app core.App, id string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding node: %w", err)
	}
	kind := Kind(rec.GetString("type"))
	out, err := clearPendingEdit(app, kind, id)
	if err != nil {
		return nil, err
	}
	store.Audit(app, "owner", "knowledge.decline_edit", "nodes/"+id, true, nil)
	return out, nil
}

// clearPendingEdit removes the parked-edit envelope from a node's props.
func clearPendingEdit(app core.App, kind Kind, id string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding node: %w", err)
	}
	props := nodes.Props(rec)
	if _, has := props[pendingEditKey]; !has {
		return hydrate(kind, rec), nil
	}
	delete(props, pendingEditKey)
	rec.Set("props", props)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("clearing pending edit: %w", err)
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

// SearchActive finds active memories matching any of the given terms. When a
// FTS5 sidecar index is available it is bm25-ranked; otherwise it falls back to
// a deterministic substring scan over the active memory nodes.
func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := ix.QueryKind(terms, string(Memory), limit)
			if err == nil && len(ids) > 0 {
				recs, err := app.FindRecordsByIds("nodes", ids)
				if err == nil {
					var active []*core.Record
					for _, r := range recs {
						if r.GetString("type") == string(Memory) && r.GetString("status") == StatusActive {
							active = append(active, hydrate(Memory, r))
						}
					}
					if len(active) > 0 {
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

	// --- substring fallback over active memory nodes ---
	recs, err := nodes.ListByTypeStatus(app, string(Memory), StatusActive)
	if err != nil {
		return nil, err
	}
	hydrateAll(Memory, recs)
	var matched []*core.Record
	for _, r := range recs {
		for _, t := range terms {
			t = strings.ToLower(strings.TrimSpace(t))
			if t == "" {
				continue
			}
			if matchesQuery(Memory, r, t) {
				matched = append(matched, r)
				break
			}
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].GetInt("importance") > matched[j].GetInt("importance")
	})
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, nil
}

// SearchAllActive is the cross-type search surface: it returns active nodes of
// ANY type matching the terms, ranked by bm25 when the FTS5 sidecar is
// available. Unlike SearchActive (which stays memory-scoped and hydrates memory
// aliases for context/recall callers), this returns RAW node records — the
// caller renders each hit by its node `type`. A node that is not active is never
// returned (the consent filter). When the index is unavailable it falls back to
// a deterministic substring scan over active nodes' title/body.
func SearchAllActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := ix.Query(terms, limit)
			if err == nil && len(ids) > 0 {
				recs, err := app.FindRecordsByIds("nodes", ids)
				if err == nil {
					var active []*core.Record
					for _, r := range recs {
						if r.GetString("status") == StatusActive {
							active = append(active, r)
						}
					}
					if len(active) > 0 {
						order := make(map[string]int, len(ids))
						for i, id := range ids {
							order[id] = i
						}
						sort.Slice(active, func(i, j int) bool {
							return order[active[i].Id] < order[active[j].Id]
						})
						if limit > 0 && len(active) > limit {
							active = active[:limit]
						}
						return active, nil
					}
				}
			}
		}
	}

	// --- substring fallback over all active nodes ---
	recs, err := app.FindRecordsByFilter(
		"nodes", "status = 'active'", "-updated,-created", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	var matched []*core.Record
	for _, r := range recs {
		for _, t := range terms {
			t = strings.ToLower(strings.TrimSpace(t))
			if t == "" {
				continue
			}
			if strings.Contains(strings.ToLower(r.GetString("title")), t) ||
				strings.Contains(strings.ToLower(r.GetString("body")), t) {
				matched = append(matched, r)
				break
			}
		}
	}
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, nil
}

// UpfrontMemories returns the highest-importance active memories that are always
// injected into context (tier 1 of the injection policy).
func UpfrontMemories(app core.App, limit int) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, string(Memory), StatusActive)
	if err != nil {
		return nil, err
	}
	hydrateAll(Memory, recs)
	var out []*core.Record
	for _, r := range recs {
		if r.GetInt("importance") >= 4 {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].GetInt("importance") > out[j].GetInt("importance")
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ActiveSkills returns active skills for the context index, ordered by name.
func ActiveSkills(app core.App) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, string(Skill), StatusActive)
	if err != nil {
		return nil, err
	}
	hydrateAll(Skill, recs)
	sort.SliceStable(recs, func(i, j int) bool {
		return recs[i].GetString("title") < recs[j].GetString("title")
	})
	return recs, nil
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
