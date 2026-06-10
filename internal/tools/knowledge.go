package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/knowledge"
)

// KnowledgeTools gives the model its memory and skill verbs. None of them
// mutate active knowledge: remember and propose_skill create PROPOSALS that
// the owner approves in the UI (the consent boundary lives in
// internal/knowledge, not in tool wording).
func KnowledgeTools(app core.App) []agent.Tool {
	return []agent.Tool{
		rememberTool(app),
		recallTool(app),
		skillTool(app),
		proposeSkillTool(app),
	}
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
				return "", fmt.Errorf("remember: bad arguments: %w", err)
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
			return MarkProposal("memories", rec.Id,
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
					m.GetString("category"), m.GetString("title"), m.GetString("content"))
				knowledge.Touch(app, knowledge.Memory, m)
			}
			return b.String(), nil
		},
	}
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
			return MarkProposal("skills", rec.Id,
				fmt.Sprintf("Skill proposal %q sent to the owner for approval. You cannot use it until approved.", args.Name)), nil
		},
	}
}
