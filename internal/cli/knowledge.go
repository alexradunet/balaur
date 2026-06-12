package cli

import (
	"fmt"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/knowledge"
)

func memoryJSON(r *core.Record) map[string]any {
	return map[string]any{
		"id":          r.Id,
		"title":       r.GetString("title"),
		"content":     r.GetString("content"),
		"category":    r.GetString("category"),
		"importance":  r.GetInt("importance"),
		"when_to_use": r.GetString("when_to_use"),
		"status":      r.GetString("status"),
		"source":      r.GetString("source"),
		"use_count":   r.GetInt("use_count"),
		"last_used":   jsonTime(r.GetDateTime("last_used").Time()),
		"created":     jsonTime(r.GetDateTime("created").Time()),
	}
}

func skillJSON(r *core.Record, withContent bool) map[string]any {
	out := map[string]any{
		"id":          r.Id,
		"name":        r.GetString("name"),
		"description": r.GetString("description"),
		"when_to_use": r.GetString("when_to_use"),
		"status":      r.GetString("status"),
		"enabled":     r.GetBool("enabled"),
		"use_count":   r.GetInt("use_count"),
		"last_used":   jsonTime(r.GetDateTime("last_used").Time()),
		"created":     jsonTime(r.GetDateTime("created").Time()),
	}
	if withContent {
		out["content"] = r.GetString("content")
	}
	return out
}

// transitionCmd builds approve/reject/archive verbs shared by memory and
// skill: thin wrappers over knowledge.Transition, which owns the lifecycle
// rules and the audit trail. cliKind is the <command>.<subcommand> string
// for the v1 envelope (e.g. "memory.approve").
func transitionCmd(app core.App, kind knowledge.Kind, verb, to string, cliKind string, render func(*core.Record) map[string]any) *cobra.Command {
	short := map[string]string{
		"approve": "Move a proposal to active (the owner's consent)",
		"reject":  "Dismiss a proposal",
		"archive": "Archive an active record",
	}
	cmd := &cobra.Command{
		Use:   verb + " <id>",
		Short: short[verb],
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, cliKind, func(cmd *cobra.Command, args []string) (any, error) {
		rec, err := knowledge.Transition(app, kind, args[0], to)
		if err != nil {
			return nil, err
		}
		return render(rec), nil
	})
	return cmd
}

func statusListCmd(app core.App, kind knowledge.Kind, cliKind string, render func(*core.Record) map[string]any) *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List " + string(kind) + " by status",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&status, "status", knowledge.StatusActive, "proposed | active | archived | rejected")
	cmd.RunE = run(app, cliKind, func(cmd *cobra.Command, args []string) (any, error) {
		recs, err := knowledge.ListByStatus(app, kind, status)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, render(r))
		}
		return out, nil
	})
	return cmd
}

func memoryCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Propose, list, recall, and curate memories — deterministic, no model",
	}
	mem := func(r *core.Record) map[string]any { return memoryJSON(r) }
	cmd.AddCommand(
		memoryProposeCmd(app),
		statusListCmd(app, knowledge.Memory, "memory.list", mem),
		memoryRecallCmd(app),
		transitionCmd(app, knowledge.Memory, "approve", knowledge.StatusActive, "memory.approve", mem),
		transitionCmd(app, knowledge.Memory, "reject", knowledge.StatusRejected, "memory.reject", mem),
		transitionCmd(app, knowledge.Memory, "archive", knowledge.StatusArchived, "memory.archive", mem),
		memoryEditCmd(app),
	)
	return cmd
}

func memoryProposeCmd(app core.App) *cobra.Command {
	var title, content, category, whenToUse string
	var importance int
	cmd := &cobra.Command{
		Use:   "propose",
		Short: "Create a memory proposal (consent boundary: needs approval to activate)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&title, "title", "", "one-line summary (required)")
	cmd.Flags().StringVar(&content, "content", "", "the full detail worth remembering (required)")
	cmd.Flags().StringVar(&category, "category", "fact", "fact | preference | person | project | context")
	cmd.Flags().IntVar(&importance, "importance", 3, "1 (nice to know) … 5 (core identity)")
	cmd.Flags().StringVar(&whenToUse, "when-to-use", "", "when should this memory be recalled?")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("content")
	cmd.RunE = run(app, "memory.propose", func(cmd *cobra.Command, args []string) (any, error) {
		rec, err := knowledge.ProposeMemory(app, knowledge.MemoryProposal{
			Title:      title,
			Content:    content,
			Category:   category,
			Importance: importance,
			WhenToUse:  whenToUse,
			Source:     "cli",
		})
		if err != nil {
			return nil, err
		}
		return memoryJSON(rec), nil
	})
	return cmd
}

func memoryRecallCmd(app core.App) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "recall <term>...",
		Short: "Search active memories the way the model's recall tool does",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.Flags().IntVar(&limit, "limit", 8, "max results")
	cmd.RunE = run(app, "memory.recall", func(cmd *cobra.Command, args []string) (any, error) {
		// Inspection only: unlike the model's recall tool this does not
		// Touch the records — a harness probing memory must not skew the
		// owner's usage statistics.
		recs, err := knowledge.SearchActive(app, args, limit)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, memoryJSON(r))
		}
		return out, nil
	})
	return cmd
}

func memoryEditCmd(app core.App) *cobra.Command {
	var title, content, category, whenToUse string
	var importance int
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a memory's owner-writable fields",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&content, "content", "", "new content")
	cmd.Flags().StringVar(&category, "category", "", "new category")
	cmd.Flags().IntVar(&importance, "importance", 0, "new importance (1-5)")
	cmd.Flags().StringVar(&whenToUse, "when-to-use", "", "new recall hint")
	cmd.RunE = run(app, "memory.edit", func(cmd *cobra.Command, args []string) (any, error) {
		fields := map[string]string{}
		set := func(flag, field, value string) {
			if cmd.Flags().Changed(flag) {
				fields[field] = value
			}
		}
		set("title", "title", title)
		set("content", "content", content)
		set("category", "category", category)
		set("when-to-use", "when_to_use", whenToUse)
		if cmd.Flags().Changed("importance") {
			fields["importance"] = strconv.Itoa(importance)
		}
		if len(fields) == 0 {
			return nil, fmt.Errorf("nothing to edit: pass at least one field flag")
		}
		rec, err := knowledge.UpdateFields(app, knowledge.Memory, args[0], fields)
		if err != nil {
			return nil, err
		}
		return memoryJSON(rec), nil
	})
	return cmd
}

func skillCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Propose, list, show, and curate skills — deterministic, no model",
	}
	sk := func(r *core.Record) map[string]any { return skillJSON(r, false) }
	cmd.AddCommand(
		skillProposeCmd(app),
		statusListCmd(app, knowledge.Skill, "skill.list", sk),
		skillShowCmd(app),
		transitionCmd(app, knowledge.Skill, "approve", knowledge.StatusActive, "skill.approve", sk),
		transitionCmd(app, knowledge.Skill, "reject", knowledge.StatusRejected, "skill.reject", sk),
		transitionCmd(app, knowledge.Skill, "archive", knowledge.StatusArchived, "skill.archive", sk),
	)
	return cmd
}

func skillProposeCmd(app core.App) *cobra.Command {
	var name, description, content, whenToUse string
	cmd := &cobra.Command{
		Use:   "propose",
		Short: "Create a skill proposal (consent boundary: needs approval to activate)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&name, "name", "", "short kebab-case name (required)")
	cmd.Flags().StringVar(&description, "description", "", "one line: what this skill does (required)")
	cmd.Flags().StringVar(&content, "content", "", "the full procedure in Markdown (required)")
	cmd.Flags().StringVar(&whenToUse, "when-to-use", "", "when should this skill be applied?")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("description")
	_ = cmd.MarkFlagRequired("content")
	cmd.RunE = run(app, "skill.propose", func(cmd *cobra.Command, args []string) (any, error) {
		rec, err := knowledge.ProposeSkill(app, knowledge.SkillProposal{
			Name:        name,
			Description: description,
			Content:     content,
			WhenToUse:   whenToUse,
		})
		if err != nil {
			return nil, err
		}
		return skillJSON(rec, true), nil
	})
	return cmd
}

func skillShowCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Load one active skill by name, content included (the model's skill tool)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, "skill.show", func(cmd *cobra.Command, args []string) (any, error) {
		rec, err := knowledge.LoadSkill(app, args[0])
		if err != nil {
			return nil, err
		}
		return skillJSON(rec, true), nil
	})
	return cmd
}
