package nodes

// RelationTypes is the curated relation-type vocabulary for node_link.
// Free-text edge types are still accepted by AddEdge — this slice is used
// for the node_link tool's enum suggestion and inverse-label lookups.
//
// TWO DEFAULTS, intentionally distinct:
//   - "links"      — wikilink-origin (DefaultEdgeType); written by [[wikilink]] parsing.
//   - "relates_to" — agent-asserted association; default for node_link.
//
// Keeping them separate lets you tell how an edge arose.
var RelationTypes = []string{
	"links",      // generic association — the wikilink default
	"relates_to", // symmetric association — the node_link default
	"part_of",    // hierarchy / membership
	"about",      // this node is about that node (e.g. a note about a person)
}

var inverseLabels = map[string]string{
	"links":      "linked from",
	"relates_to": "relates to",
	"part_of":    "has part",
	"about":      "referenced by",
}

// InverseLabel returns the display inverse for a relation type.
// Falls back to "linked from" for unrecognised types.
func InverseLabel(relType string) string {
	if label, ok := inverseLabels[relType]; ok {
		return label
	}
	return "linked from"
}
