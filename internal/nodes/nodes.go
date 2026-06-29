// Package nodes is Balaur's unified knowledge spine: every piece of knowledge —
// a note, a memory, a skill, a journal day, a typed object (person, book, …) —
// is one row in the `nodes` collection, distinguished by `type` and linked to
// other nodes through the `edges` collection.
//
// THE CONSENT BOUNDARY lives in `status`: owner-authored kinds (note, journal,
// typed objects) are born active and trusted; agent-proposed kinds (memory,
// skill) are born proposed and become active only on the owner's explicit
// approval. Graph traversal AND search filter to status=active so a proposed or
// rejected node is never surfaced as fact. Every mutation is audited.
package nodes

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// Statuses for nodes (mirrors the nodes.status enum in
// migrations/1749600000_init.go).
const (
	StatusProposed = "proposed"
	StatusActive   = "active"
	StatusArchived = "archived"
	StatusRejected = "rejected"
)

// DefaultEdgeType is the edge type used when AddEdge is called with an empty
// type. PocketBase TextField carries no schema default, so the default is a
// write-side concern enforced here.
const DefaultEdgeType = "links"

// ValidTransitions encodes the owner-driven lifecycle. Key: from → allowed to.
// One source of truth — internal/knowledge references this map.
var ValidTransitions = map[string][]string{
	StatusProposed: {StatusActive, StatusRejected},
	StatusActive:   {StatusArchived},
	StatusArchived: {StatusActive},
	StatusRejected: {},
}

// Props reads a node's props json into a map. Returns an empty (non-nil) map
// when props is absent or malformed, so callers can index it unconditionally.
func Props(rec *core.Record) map[string]any {
	m := map[string]any{}
	if raw, ok := rec.Get("props").(map[string]any); ok {
		return raw
	}
	// props may round-trip as types.JSONRaw; decode defensively.
	if err := rec.UnmarshalJSONField("props", &m); err != nil {
		return map[string]any{}
	}
	return m
}

// PropString reads one string field out of props (empty when absent).
func PropString(rec *core.Record, key string) string {
	if v, ok := Props(rec)[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// PropInt reads one int field out of props (0 when absent). JSON numbers decode
// as float64, so a float is truncated to int.
func PropInt(rec *core.Record, key string) int {
	switch v := Props(rec)[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// Create writes a node of the given type/status with the supplied props and
// audits node.create after the write. props may be nil.
//
// The template for the type is applied first (filling missing prop keys and an
// empty body), then props are validated against the type's property schema.
// Types with an empty schema accept any props.
func Create(app core.App, typ, title, body, status string, props map[string]any) (*core.Record, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("nodes: title is required")
	}
	ok, err := TypeExists(app, typ)
	if err != nil {
		return nil, fmt.Errorf("nodes: checking type: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("nodes: unknown type %q (not in node_types registry)", typ)
	}

	// Apply template defaults before validation so required-with-default fields pass.
	tmpl, err := TypeTemplate(app, typ)
	if err != nil {
		return nil, fmt.Errorf("nodes: loading template for %q: %w", typ, err)
	}
	if props == nil {
		props = map[string]any{}
	}
	body, props = ApplyTemplate(tmpl, body, props)

	// Validate props against the type's schema (empty schema = any props ok).
	defs, err := TypeSchema(app, typ)
	if err != nil {
		return nil, fmt.Errorf("nodes: loading schema for %q: %w", typ, err)
	}
	if err := ValidateProps(defs, props); err != nil {
		return nil, fmt.Errorf("nodes: invalid props for type %q: %w", typ, err)
	}

	col, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return nil, fmt.Errorf("finding nodes collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("type", typ)
	rec.Set("title", title)
	rec.Set("body", body)
	rec.Set("status", status)
	if len(props) > 0 {
		rec.Set("props", props)
	}
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving node: %w", err)
	}
	store.Audit(app, "owner", "node.create", "nodes/"+rec.Id, true,
		map[string]any{"type": typ, "title": title})
	return rec, nil
}

// Update edits an existing ACTIVE node's title, body, and/or props in place and
// audits node.update after the write. Only non-nil arguments change a field:
// title/body are pointers (nil = leave unchanged, "" = clear body); props, when
// non-nil, REPLACES the node's props (after template-apply + schema validation).
//
// The CONSENT BOUNDARY is enforced here: Update refuses any node whose type is
// not owner-authored (born active) — specifically memory and skill, which must
// change only through the consent-gated propose_edit path. A non-active node is
// likewise refused: owner-authored typed nodes are born active, so a proposed or
// rejected node is not an editable owner object.
func Update(app core.App, id string, title, body *string, props map[string]any) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("nodes: no node %q", id)
	}
	typ := rec.GetString("type")

	// Owner-authored types only: memory/skill are consent-gated and must go
	// through propose_edit, never an in-place edit. Refuse any type whose
	// born_status is not active.
	ownerTypes, err := OwnerAuthoredTypes(app)
	if err != nil {
		return nil, fmt.Errorf("nodes: loading owner-authored types: %w", err)
	}
	if !slices.Contains(ownerTypes, typ) {
		return nil, fmt.Errorf("nodes: type %q is not owner-authored — use remember/propose_edit for memory and skill", typ)
	}
	if rec.GetString("status") != StatusActive {
		return nil, fmt.Errorf("nodes: node %q is not active (status=%s)", id, rec.GetString("status"))
	}

	if title != nil {
		if strings.TrimSpace(*title) == "" {
			return nil, fmt.Errorf("nodes: title cannot be cleared")
		}
		rec.Set("title", *title)
	}
	if body != nil {
		rec.Set("body", *body)
	}
	if props != nil {
		// Validate the replacement props against the type's schema, mirroring
		// Create (template-apply first so required-with-default fields pass).
		tmpl, err := TypeTemplate(app, typ)
		if err != nil {
			return nil, fmt.Errorf("nodes: loading template for %q: %w", typ, err)
		}
		_, merged := ApplyTemplate(tmpl, rec.GetString("body"), props)
		defs, err := TypeSchema(app, typ)
		if err != nil {
			return nil, fmt.Errorf("nodes: loading schema for %q: %w", typ, err)
		}
		if err := ValidateProps(defs, merged); err != nil {
			return nil, fmt.Errorf("nodes: invalid props for type %q: %w", typ, err)
		}
		rec.Set("props", merged)
	}

	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving node: %w", err)
	}
	store.Audit(app, "owner", "node.update", "nodes/"+rec.Id, true,
		map[string]any{"type": typ})
	return rec, nil
}

// Get fetches one node by id.
func Get(app core.App, id string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding node %q: %w", id, err)
	}
	return rec, nil
}

// Drop deletes one node by id (cascading its edges) and audits node.drop.
func Drop(app core.App, id string) error {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("nodes: no node %q", id)
	}
	if err := app.Delete(rec); err != nil {
		return fmt.Errorf("dropping node: %w", err)
	}
	store.Audit(app, "owner", "node.drop", "nodes/"+id, true, nil)
	return nil
}

// ListByTypeStatus returns nodes of one type in one status, newest first.
func ListByTypeStatus(app core.App, typ, status string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("nodes",
		"type = {:t} && status = {:s}", "-created", 0, 0,
		dbx.Params{"t": typ, "s": status})
}

// Transition moves a node to a new status on the owner's behalf, validating the
// lifecycle and auditing the outcome. auditPrefix is prepended to the action
// (e.g. "node" → "node.archived"; "knowledge" → "knowledge.archived").
func Transition(app core.App, id, to, auditPrefix string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding node %q: %w", id, err)
	}
	from := rec.GetString("status")
	if !slices.Contains(ValidTransitions[from], to) {
		store.Audit(app, "owner", auditPrefix+"."+to, "nodes/"+rec.Id, false,
			map[string]any{"from": from})
		return nil, fmt.Errorf("nodes: cannot move from %q to %q", from, to)
	}
	rec.Set("status", to)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("updating node status: %w", err)
	}
	store.Audit(app, "owner", auditPrefix+"."+to, "nodes/"+rec.Id, true,
		map[string]any{"from": from})
	return rec, nil
}

// AddEdge links source → target with edgeType (defaulting to DefaultEdgeType).
// It is idempotent against the unique (source, target, type) index: a duplicate
// returns the existing edge rather than erroring. context is optional free text.
func AddEdge(app core.App, sourceID, targetID, edgeType, context string) (*core.Record, error) {
	if strings.TrimSpace(edgeType) == "" {
		edgeType = DefaultEdgeType
	}
	col, err := app.FindCollectionByNameOrId("edges")
	if err != nil {
		return nil, fmt.Errorf("finding edges collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("source", sourceID)
	rec.Set("target", targetID)
	rec.Set("type", edgeType)
	rec.Set("context", context)
	if err := app.Save(rec); err != nil {
		// Unique-constraint violation → the edge already exists; return it.
		if existing, ferr := app.FindFirstRecordByFilter("edges",
			"source = {:s} && target = {:t} && type = {:y}",
			dbx.Params{"s": sourceID, "t": targetID, "y": edgeType}); ferr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("saving edge: %w", err)
	}
	store.Audit(app, "owner", "edge.create", "edges/"+rec.Id, true,
		map[string]any{"source": sourceID, "target": targetID, "type": edgeType})
	return rec, nil
}

// activeByIDs loads nodes by id and keeps only the active ones, preserving the
// caller's id order. The status=active filter is the consent spine: traversal
// must never surface a proposed or rejected node.
func activeByIDs(app core.App, ids []string) ([]*core.Record, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	recs, err := app.FindRecordsByIds("nodes", ids)
	if err != nil {
		return nil, fmt.Errorf("loading nodes: %w", err)
	}
	byID := make(map[string]*core.Record, len(recs))
	for _, r := range recs {
		byID[r.Id] = r
	}
	out := make([]*core.Record, 0, len(ids))
	for _, id := range ids {
		if r, ok := byID[id]; ok && r.GetString("status") == StatusActive {
			out = append(out, r)
		}
	}
	return out, nil
}

// Backlinks returns the active nodes that link TO id (inbound edges).
func Backlinks(app core.App, id string) ([]*core.Record, error) {
	edges, err := app.FindRecordsByFilter("edges", "target = {:id}", "", 0, 0, dbx.Params{"id": id})
	if err != nil {
		return nil, fmt.Errorf("loading inbound edges: %w", err)
	}
	ids := make([]string, 0, len(edges))
	for _, e := range edges {
		ids = append(ids, e.GetString("source"))
	}
	return activeByIDs(app, ids)
}

// Outbound returns the active nodes that id links TO (outbound edges).
func Outbound(app core.App, id string) ([]*core.Record, error) {
	edges, err := app.FindRecordsByFilter("edges", "source = {:id}", "", 0, 0, dbx.Params{"id": id})
	if err != nil {
		return nil, fmt.Errorf("loading outbound edges: %w", err)
	}
	ids := make([]string, 0, len(edges))
	for _, e := range edges {
		ids = append(ids, e.GetString("target"))
	}
	return activeByIDs(app, ids)
}

// Neighborhood returns the 1-hop set (backlinks ∪ outbound), active only, with
// duplicates removed.
func Neighborhood(app core.App, id string) ([]*core.Record, error) {
	back, err := Backlinks(app, id)
	if err != nil {
		return nil, err
	}
	out, err := Outbound(app, id)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	merged := make([]*core.Record, 0, len(back)+len(out))
	for _, r := range append(back, out...) {
		if !seen[r.Id] {
			seen[r.Id] = true
			merged = append(merged, r)
		}
	}
	return merged, nil
}
