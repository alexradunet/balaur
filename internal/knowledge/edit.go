package knowledge

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
)

// edit.go is the parked-edit envelope: a model proposes a change to an ACTIVE
// memory/skill by parking it in the node's props (ProposeEdit) without touching
// the approved content; only the owner approves (ApplyEdit), declines
// (DeclineEdit), or the review queue lists pending edits (PendingEdits). Split
// out of knowledge.go (plan 204).

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
