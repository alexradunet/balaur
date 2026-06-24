package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/nodes"
)

// GraphTools returns the four graph-traversal and relation tools.
// They are appended to KnowledgeTools in knowledge.go.
func GraphTools(app core.App) []agent.Tool {
	return []agent.Tool{
		nodeLinkTool(app),
		nodeRelatedTool(app),
		nodeQueryTool(app),
		nodeSchemaTool(app),
	}
}

// nodeLinkTool creates a typed, audited edge between two active nodes.
func nodeLinkTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_link",
			"Assert a typed relation between two knowledge nodes (by id). "+
				"Both nodes must be active. Relation types: "+strings.Join(nodes.RelationTypes, ", ")+". "+
				"Default relation is 'relates_to' (agent-asserted); "+
				"'links' is reserved for wikilink-origin edges. "+
				"Idempotent: a duplicate call returns the existing edge. "+
				"Use node_query or node_list to find node ids.",
			obj(map[string]any{
				"source":   str("Source node id."),
				"target":   str("Target node id."),
				"relation": map[string]any{"type": "string", "enum": nodes.RelationTypes, "description": "Relation type (default relates_to)."},
				"context":  str("Optional: why this link exists."),
			}, "source", "target")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Source   string `json:"source"`
				Target   string `json:"target"`
				Relation string `json:"relation"`
				Context  string `json:"context"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_link: bad arguments: %w", err)
			}
			if strings.TrimSpace(args.Source) == "" || strings.TrimSpace(args.Target) == "" {
				return "", fmt.Errorf("node_link: source and target are required")
			}
			rel := args.Relation
			if rel == "" {
				rel = "relates_to"
			}

			// Resolve both nodes — they must exist and be active (consent boundary).
			src, err := nodes.Get(app, strings.TrimSpace(args.Source))
			if err != nil {
				return "", fmt.Errorf("node_link: source node not found: %w", err)
			}
			if src.GetString("status") != nodes.StatusActive {
				return "", fmt.Errorf("node_link: source node %q is not active (status=%s)", args.Source, src.GetString("status"))
			}
			tgt, err := nodes.Get(app, strings.TrimSpace(args.Target))
			if err != nil {
				return "", fmt.Errorf("node_link: target node not found: %w", err)
			}
			if tgt.GetString("status") != nodes.StatusActive {
				return "", fmt.Errorf("node_link: target node %q is not active (status=%s)", args.Target, tgt.GetString("status"))
			}

			if _, err := nodes.AddEdge(app, src.Id, tgt.Id, rel, args.Context); err != nil {
				return "", fmt.Errorf("node_link: %w", err)
			}
			return fmt.Sprintf("Linked %q --%s--> %q.", src.GetString("title"), rel, tgt.GetString("title")), nil
		},
	}
}

// nodeRelatedTool returns the 1-hop neighbours of a node.
func nodeRelatedTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_related",
			"Return the 1-hop neighbours of a knowledge node (active only). "+
				"direction=both (default) returns backlinks ∪ outbound; "+
				"direction=out returns outbound only; direction=in returns backlinks only.",
			obj(map[string]any{
				"id":        str("The node id."),
				"direction": map[string]any{"type": "string", "enum": []string{"both", "out", "in"}, "description": "Traversal direction (default both)."},
			}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				ID        string `json:"id"`
				Direction string `json:"direction"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_related: bad arguments: %w", err)
			}
			id := strings.TrimSpace(args.ID)
			if id == "" {
				return "", fmt.Errorf("node_related: id is required")
			}
			dir := args.Direction
			if dir == "" {
				dir = "both"
			}

			var recs []*core.Record
			var err error
			switch dir {
			case "out":
				recs, err = nodes.Outbound(app, id)
			case "in":
				recs, err = nodes.Backlinks(app, id)
			default:
				recs, err = nodes.Neighborhood(app, id)
			}
			if err != nil {
				return "", fmt.Errorf("node_related: %w", err)
			}
			if len(recs) == 0 {
				return "No related active nodes.", nil
			}
			var b strings.Builder
			for _, r := range recs {
				fmt.Fprintf(&b, "- [%s] %s (id %s)\n", r.GetString("type"), r.GetString("title"), r.Id)
			}
			return b.String(), nil
		},
	}
}

// nodeQueryTool searches active nodes by type and/or property substrings.
func nodeQueryTool(app core.App) agent.Tool {
	allTypes, err := nodes.TypeNames(app)
	if err != nil || len(allTypes) == 0 {
		allTypes = []string{"note", "memory", "skill", "day", "person", "book", "idea", "place"}
	}
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_query",
			"Search active knowledge nodes by type and/or property substrings (AND across keys). "+
				"Known types: "+strings.Join(allTypes, ", ")+". "+
				"Returns up to limit nodes (default 50). Proposed/rejected nodes are never returned.",
			obj(map[string]any{
				"type":  str("Node type to filter by (optional)."),
				"match": map[string]any{"type": "object", "description": "Property key → substring pairs to filter by (optional, AND semantics).", "additionalProperties": map[string]any{"type": "string"}},
				"limit": map[string]any{"type": "integer", "description": "Maximum results (default 50)."},
			})),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Type  string            `json:"type"`
				Match map[string]string `json:"match"`
				Limit int               `json:"limit"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_query: bad arguments: %w", err)
			}
			recs, err := nodes.Query(app, nodes.QueryOpts{
				Type:      strings.TrimSpace(args.Type),
				PropMatch: args.Match,
				Limit:     args.Limit,
			})
			if err != nil {
				return "", fmt.Errorf("node_query: %w", err)
			}
			if len(recs) == 0 {
				return "No matching nodes.", nil
			}
			var b strings.Builder
			for _, r := range recs {
				fmt.Fprintf(&b, "- [%s] %s (id %s)\n", r.GetString("type"), r.GetString("title"), r.Id)
			}
			return b.String(), nil
		},
	}
}

// nodeSchemaTool introspects the node_types registry.
func nodeSchemaTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_schema",
			"Discover registered node types and their property schemas. "+
				"Omit 'type' to list all types. "+
				"Provide 'type' to see that type's props (key, value-type, required). "+
				"Read this before writing a typed node so you supply the right props.",
			obj(map[string]any{
				"type": str("Node type name (optional — omit for all types)."),
			})),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_schema: bad arguments: %w", err)
			}
			typ := strings.TrimSpace(args.Type)

			if typ != "" {
				return renderTypeSchema(app, typ)
			}

			// All types.
			names, err := nodes.TypeNames(app)
			if err != nil {
				return "", fmt.Errorf("node_schema: %w", err)
			}
			if len(names) == 0 {
				return "No node types registered.", nil
			}
			var b strings.Builder
			for _, name := range names {
				s, err := renderTypeSchema(app, name)
				if err != nil {
					return "", err
				}
				b.WriteString(s)
				b.WriteByte('\n')
			}
			return strings.TrimRight(b.String(), "\n"), nil
		},
	}
}

// renderTypeSchema returns a human-readable schema line for one type.
func renderTypeSchema(app core.App, typ string) (string, error) {
	defs, err := nodes.TypeSchema(app, typ)
	if err != nil {
		return "", fmt.Errorf("node_schema: loading schema for %q: %w", typ, err)
	}
	if len(defs) == 0 {
		return fmt.Sprintf("- %s: no required props (open schema)", typ), nil
	}
	parts := make([]string, 0, len(defs))
	for _, d := range defs {
		s := string(d.Key) + ":" + string(d.Type)
		if d.Required {
			s += "[required]"
		}
		if len(d.Options) > 0 {
			s += "(" + strings.Join(d.Options, "|") + ")"
		}
		parts = append(parts, s)
	}
	return fmt.Sprintf("- %s: props=[%s]", typ, strings.Join(parts, ", ")), nil
}
