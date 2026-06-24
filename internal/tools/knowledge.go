package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/nodes"
)

// KnowledgeTools gives the model its memory and skill verbs. None of them
// mutate active knowledge: remember and propose_skill create PROPOSALS that
// the owner approves in the UI (the consent boundary lives in
// internal/knowledge, not in tool wording).
func KnowledgeTools(app core.App) []agent.Tool {
	ts := []agent.Tool{
		rememberTool(app),
		recallTool(app),
		searchTool(app),
		skillTool(app),
		proposeSkillTool(app),
		nodeWriteTool(app),
		nodeListTool(app),
		nodeGetTool(app),
		nodeDropTool(app),
	}
	return append(ts, GraphTools(app)...)
}

// ProposalMarker prefixes tool results that carry a proposal id, so the web
// layer can render an approval card instead of a plain tool row. Format:
// marker + kind + "/" + record id, then a newline and the model-facing text.
const ProposalMarker = "\x00balaur-proposal:"

// MarkProposal builds a marked tool result. Exposed for the web layer and
// tests; the model never sees the marker (it is stripped before rendering
// and harmless if echoed — it carries no instructions).
func MarkProposal(kind, id, modelText string) string {
	return ProposalMarker + kind + "/" + id + "\n" + modelText
}

// ParseProposal splits a marked tool result. ok is false for ordinary text.
func ParseProposal(s string) (kind, id, rest string, ok bool) {
	if !strings.HasPrefix(s, ProposalMarker) {
		return "", "", s, false
	}
	s = strings.TrimPrefix(s, ProposalMarker)
	head, rest, _ := strings.Cut(s, "\n")
	kind, id, found := strings.Cut(head, "/")
	if !found {
		return "", "", rest, false
	}
	return kind, id, rest, true
}

func rememberTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("remember",
			"Propose saving a durable memory about the owner (fact, preference, person, project, context). "+
				"The owner must approve it before it becomes part of your memory — never assume it is saved.",
			obj(map[string]any{
				"title":       str("Short one-line summary of the memory."),
				"content":     str("The full detail worth remembering."),
				"category":    map[string]any{"type": "string", "enum": []string{"fact", "preference", "person", "project", "context"}, "description": "Kind of memory."},
				"importance":  map[string]any{"type": "integer", "minimum": 1, "maximum": 5, "description": "5 = core identity/constraints, 1 = nice to know."},
				"when_to_use": str("Optional: when should this memory be recalled?"),
			}, "title", "content", "category", "importance")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Title      string `json:"title"`
				Content    string `json:"content"`
				Category   string `json:"category"`
				Importance int    `json:"importance"`
				WhenToUse  string `json:"when_to_use"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				var fallback string
				if err := json.Unmarshal([]byte(argsJSON), &fallback); err != nil {
					return "", fmt.Errorf("remember: bad arguments: %w", err)
				}
				fallback = strings.TrimSpace(fallback)
				if fallback == "" {
					return "", fmt.Errorf("remember: memory text is required")
				}
				args.Title = fallback
				args.Content = fallback
				args.Category = "fact"
				args.Importance = 3
			}
			if strings.TrimSpace(args.Title) == "" {
				return "", fmt.Errorf("remember: memory title is required")
			}
			rec, err := knowledge.ProposeMemory(app, knowledge.MemoryProposal{
				Title:      args.Title,
				Content:    args.Content,
				Category:   args.Category,
				Importance: args.Importance,
				WhenToUse:  args.WhenToUse,
				Source:     "chat",
			})
			if err != nil {
				return "", err
			}
			return MarkProposal("nodes", rec.Id,
				fmt.Sprintf("Memory proposal %q sent to the owner for approval. It is NOT yet part of your memory.", args.Title)), nil
		},
	}
}

func recallTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("recall",
			"Search your approved memories for terms the automatic context may have missed.",
			obj(map[string]any{
				"terms": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "1-3 search terms."},
			}, "terms")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Terms []string `json:"terms"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("recall: bad arguments: %w", err)
			}
			recs, err := knowledge.SearchActive(app, args.Terms, 8)
			if err != nil {
				return "", err
			}
			if len(recs) == 0 {
				return "No approved memories match.", nil
			}
			var b strings.Builder
			for _, m := range recs {
				fmt.Fprintf(&b, "- [%s] %s: %s\n",
					m.GetString("type"), m.GetString("title"), m.GetString("body"))
			}
			return b.String(), nil
		},
	}
}

func searchTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("search",
			"Full-text search across ALL your approved knowledge — notes, memories, "+
				"skills, journal entries, and typed objects. Returns mixed-type hits "+
				"ranked by relevance. Proposed/unapproved knowledge is never returned.",
			obj(map[string]any{
				"terms": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "1-4 search terms (OR semantics)."},
			}, "terms")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Terms []string `json:"terms"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("search: bad arguments: %w", err)
			}
			recs, err := knowledge.SearchAllActive(app, args.Terms, 10)
			if err != nil {
				return "", err
			}
			if len(recs) == 0 {
				return "No approved knowledge matches.", nil
			}
			var b strings.Builder
			for _, r := range recs {
				fmt.Fprintf(&b, "- [%s] %s: %s\n",
					r.GetString("type"), r.GetString("title"), snippet(r.GetString("body")))
			}
			return b.String(), nil
		},
	}
}

// snippet returns a short single-line preview of node body text for search hits.
func snippet(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len([]rune(s)) > 160 {
		return string([]rune(s)[:160]) + "…"
	}
	return s
}

func skillTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("skill",
			"Load the full content of an approved skill by name before applying it.",
			obj(map[string]any{
				"name": str("Exact skill name from the skills index."),
			}, "name")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("skill: bad arguments: %w", err)
			}
			rec, err := knowledge.LoadSkill(app, args.Name)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("# Skill: %s\n%s\n\n%s",
				rec.GetString("name"), rec.GetString("description"), rec.GetString("content")), nil
		},
	}
}

func proposeSkillTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("propose_skill",
			"Propose a new reusable skill (a procedure you could repeat later). "+
				"The owner must approve it before you can use it.",
			obj(map[string]any{
				"name":        str("Short kebab-case name, e.g. weekly-review."),
				"description": str("One line: what this skill does."),
				"content":     str("The full procedure in Markdown steps."),
				"when_to_use": str("When should this skill be applied?"),
			}, "name", "description", "content")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Content     string `json:"content"`
				WhenToUse   string `json:"when_to_use"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("propose_skill: bad arguments: %w", err)
			}
			rec, err := knowledge.ProposeSkill(app, knowledge.SkillProposal{
				Name:        args.Name,
				Description: args.Description,
				Content:     args.Content,
				WhenToUse:   args.WhenToUse,
			})
			if err != nil {
				return "", err
			}
			return MarkProposal("nodes", rec.Id,
				fmt.Sprintf("Skill proposal %q sent to the owner for approval. You cannot use it until approved.", args.Name)), nil
		},
	}
}

// nodeWriteTool creates an owner-authored node (note or typed object). Unlike
// remember/propose_skill these are born active — owner-voiced, trusted writes.
func nodeWriteTool(app core.App) agent.Tool {
	allowedTypes, err := nodes.OwnerAuthoredTypes(app)
	if err != nil || len(allowedTypes) == 0 {
		app.Logger().Warn("node_write: could not load owner-authored types from registry; falling back to [note]", "error", err)
		allowedTypes = []string{"note"}
	}
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_write",
			"Write an owner-authored knowledge node — a note or a typed object (person, book, idea, place). "+
				"Born active (the owner's own, trusted). For things you want the owner to APPROVE as a memory, use remember instead.",
			obj(map[string]any{
				"type":  map[string]any{"type": "string", "enum": allowedTypes, "description": "Node type (default note)."},
				"title": str("Short title for the node."),
				"body":  str("The node's markdown body."),
			}, "title")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Type  string `json:"type"`
				Title string `json:"title"`
				Body  string `json:"body"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_write: bad arguments: %w", err)
			}
			if strings.TrimSpace(args.Title) == "" {
				return "", fmt.Errorf("node_write: title is required")
			}
			typ := args.Type
			if typ == "" {
				typ = "note"
			}
			if !slices.Contains(allowedTypes, typ) {
				return "", fmt.Errorf("node_write: type %q is not an owner-authored type", typ)
			}
			rec, err := nodes.Create(app, typ, args.Title, args.Body, nodes.StatusActive, nil)
			if err != nil {
				return "", fmt.Errorf("node_write: %w", err)
			}
			return fmt.Sprintf("Saved %s %q (id %s).", typ, args.Title, rec.Id), nil
		},
	}
}

func nodeListTool(app core.App) agent.Tool {
	allTypes, err := nodes.TypeNames(app)
	if err != nil || len(allTypes) == 0 {
		app.Logger().Warn("node_list: could not load types from registry; falling back to [note]", "error", err)
		allTypes = []string{"note"}
	}
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_list",
			"List active knowledge nodes of a given type (newest first).",
			obj(map[string]any{
				"type": map[string]any{"type": "string", "enum": allTypes, "description": "Node type to list (default note)."},
			}, "type")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_list: bad arguments: %w", err)
			}
			typ := args.Type
			if typ == "" {
				typ = "note"
			}
			recs, err := nodes.ListByTypeStatus(app, typ, nodes.StatusActive)
			if err != nil {
				return "", fmt.Errorf("node_list: %w", err)
			}
			if len(recs) == 0 {
				return fmt.Sprintf("No active %s nodes.", typ), nil
			}
			var b strings.Builder
			for _, r := range recs {
				fmt.Fprintf(&b, "- [%s] %s\n", r.Id, r.GetString("title"))
			}
			return b.String(), nil
		},
	}
}

func nodeGetTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_get",
			"Read one knowledge node's full body, props, and link summary by id.",
			obj(map[string]any{"id": str("The node id.")}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_get: bad arguments: %w", err)
			}
			rec, err := nodes.Get(app, strings.TrimSpace(args.ID))
			if err != nil {
				return "", fmt.Errorf("node_get: %w", err)
			}
			var b strings.Builder
			fmt.Fprintf(&b, "# %s (%s)\n", rec.GetString("title"), rec.GetString("type"))
			// Props — skip empty values.
			for k, v := range nodes.Props(rec) {
				s := fmt.Sprintf("%v", v)
				if s != "" && s != "<nil>" {
					fmt.Fprintf(&b, "%s: %s\n", k, s)
				}
			}
			body := rec.GetString("body")
			if body != "" {
				fmt.Fprintf(&b, "\n%s\n", body)
			}
			// Link summary.
			out, _ := nodes.Outbound(app, rec.Id)
			back, _ := nodes.Backlinks(app, rec.Id)
			fmt.Fprintf(&b, "\nLinks: %d outbound, %d backlinks", len(out), len(back))

			// For day nodes, append the day's recap summary if one exists.
			if rec.GetString("type") == "day" {
				dateKey := nodes.PropString(rec, "date")
				if dateKey != "" {
					if conv, err := conversation.Master(app); err == nil {
						sum, err := app.FindFirstRecordByFilter("summaries",
							"conversation = {:conv} && period_type = 'day' && period_start ~ {:d}",
							dbx.Params{"conv": conv.Id, "d": dateKey + "%"})
						if err == nil {
							fmt.Fprintf(&b, "\n\n## Day recap\n%s", sum.GetString("content"))
						} else {
							fmt.Fprintf(&b, "\n\nNo recap yet for %s.", dateKey)
						}
					}
				}
			}
			return b.String(), nil
		},
	}
}

func nodeDropTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_drop",
			"Delete one owner-authored knowledge node by id.",
			obj(map[string]any{"id": str("The node id to delete.")}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_drop: bad arguments: %w", err)
			}
			if err := nodes.Drop(app, strings.TrimSpace(args.ID)); err != nil {
				return "", fmt.Errorf("node_drop: %w", err)
			}
			return fmt.Sprintf("Deleted node %s.", args.ID), nil
		},
	}
}
