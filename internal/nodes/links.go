// links.go (plan 161): turns [[wikilinks]] in node bodies into "links" edges
// between nodes. The parser, the resolve-or-create-stub step, and the idempotent
// edge sync run on node save. It REUSES this package's AddEdge/Backlinks/Outbound
// (plan 160) — it does NOT redeclare them. All operations filter strictly to
// status=active nodes — proposed/rejected nodes never enter the graph (the
// consent boundary).
package nodes

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// wikilinkRe matches [[Target]] and [[Target|alias]]. The target is group 1
// (everything up to an optional pipe); the alias (group 2) is display-only and
// does not affect resolution. Non-greedy inner match so adjacent links
// [[a]][[b]] split correctly; [^\[\]|] forbids brackets/pipes inside the target
// so nested [[ ]] cannot swallow a closing pair.
var wikilinkRe = regexp.MustCompile(`\[\[([^\[\]|]+?)(?:\|([^\[\]]*))?\]\]`)

// ParseLinks returns the distinct link targets in a body, in first-seen order,
// trimmed and de-duplicated case-insensitively-by-trimmed-title. Empty targets
// (e.g. "[[]]" or "[[  ]]") are skipped.
//
// Case-folding policy (deliberate, document it): the case-insensitive dedup here
// only collapses duplicate links WITHIN one body so we never write two edges for
// "[[Alpha]] [[alpha]]" — it picks the first-seen casing as the resolution title.
// Resolution itself (resolveOrCreateStub, below) matches by EXACT title
// (`title = {:title}`), so the stored casing is what resolves. This is fine for
// the Pareto slice; if case-collision across distinct nodes ("Alpha" vs "alpha")
// ever matters, make both ends consistent then — not now.
func ParseLinks(body string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(body, -1)
	var out []string
	seen := map[string]bool{}
	for _, m := range matches {
		title := strings.TrimSpace(m[1])
		if title == "" {
			continue
		}
		key := strings.ToLower(title)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, title)
	}
	return out
}

// resolveOrCreateStub returns the id of the active node titled `title`, creating
// a stub note node if none exists. Stubs are born active (owner-authored content
// links are trusted; the stub is a placeholder the owner can flesh out later).
func resolveOrCreateStub(app core.App, title string) (string, error) {
	rec, err := app.FindFirstRecordByFilter("nodes",
		"status = 'active' && title = {:title}", dbx.Params{"title": title})
	if err == nil {
		return rec.Id, nil
	}
	col, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return "", fmt.Errorf("nodes: find nodes collection: %w", err)
	}
	stub := core.NewRecord(col)
	stub.Set("type", "note")
	stub.Set("title", title)
	stub.Set("body", "")
	stub.Set("status", "active")
	if err := app.Save(stub); err != nil {
		return "", fmt.Errorf("nodes: create stub node %q: %w", title, err)
	}
	store.Audit(app, "owner", "graph.stub", "nodes/"+stub.Id, true,
		map[string]any{"title": title})
	return stub.Id, nil
}

// SyncLinks parses the source node's body, resolves each [[link]] to an active
// node id (creating stubs), and rewrites this source's "links" edges to exactly
// that set. Idempotent: calling twice with the same body is a no-op after the
// first (the delete clears the old set; AddEdge is idempotent against the
// (source,target,type) unique index). A node never links to itself (self-links
// are dropped).
func SyncLinks(app core.App, source *core.Record) error {
	titles := ParseLinks(source.GetString("body"))

	// Resolve to target ids (dedup again post-resolution: two titles may map to
	// the same node).
	wantTargets := map[string]bool{}
	for _, t := range titles {
		id, err := resolveOrCreateStub(app, t)
		if err != nil {
			return err
		}
		if id == source.Id {
			continue // no self-edge
		}
		wantTargets[id] = true
	}

	// Full-replace: delete this source's existing "links" edges, then re-insert
	// the wanted set through AddEdge (160), which is idempotent against the
	// (source,target,type) unique index. (Simplest correct; edge count per node
	// is small — see Maintenance notes for the diff-based optimization.)
	old, err := app.FindRecordsByFilter("edges",
		"source = {:src} && type = {:t}", "", 0, 0,
		dbx.Params{"src": source.Id, "t": DefaultEdgeType})
	if err != nil {
		return fmt.Errorf("nodes: load existing edges: %w", err)
	}
	for _, e := range old {
		if err := app.Delete(e); err != nil {
			return fmt.Errorf("nodes: delete stale edge %s: %w", e.Id, err)
		}
	}
	for tgt := range wantTargets {
		// AddEdge defaults the type to "links" on empty and is idempotent against
		// the unique index (160). In-package call — do NOT redeclare it here.
		if _, err := AddEdge(app, source.Id, tgt, DefaultEdgeType, ""); err != nil {
			return fmt.Errorf("nodes: create edge %s→%s: %w", source.Id, tgt, err)
		}
	}
	return nil
}
