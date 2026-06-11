package self

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
)

// Tool is the model's window into itself: embedded self-knowledge plus
// the live capability inventory. Read-only and deterministic — knowing
// yourself must never require a model call or mutate anything.
// toolNames is the assembled registry (including "self"), supplied by
// the caller so the capabilities section reports what is actually live.
func Tool(app core.App, toolNames []string) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("self",
			"Consult your own self-knowledge: who you are, your architecture, "+
				"your live capabilities, where your source code lives, and the "+
				"self-development loop. Use it before answering questions about "+
				"yourself — never guess about your own capabilities.",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"section": map[string]any{
						"type":        "string",
						"enum":        SectionNames(),
						"description": "overview (default) | architecture | capabilities (live inventory) | source (where your code is) | devloop (how to develop yourself)",
					},
				},
			}),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Section string `json:"section"`
			}
			if argsJSON != "" {
				if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
					return "", fmt.Errorf("self: bad arguments: %w", err)
				}
			}
			if args.Section == "" {
				args.Section = "overview"
			}

			switch args.Section {
			case "capabilities":
				return renderCapabilities(app, toolNames), nil
			case "source":
				return renderSource()
			default:
				text, err := Section(args.Section)
				if err != nil {
					return "", err
				}
				if args.Section == "overview" {
					info := BuildInfo()
					text = fmt.Sprintf("Build: version %s, commit %s, built %s, %s.\n\n%s",
						info.Version, info.Commit, info.Built, info.Go, text)
				}
				return text, nil
			}
		},
	}
}

// renderCapabilities folds the static capabilities section together with
// the live inventory, so the model sees both the semantics and the truth.
func renderCapabilities(app core.App, toolNames []string) string {
	var b strings.Builder
	if text, err := Section("capabilities"); err == nil {
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	inv := Inventory(app, toolNames)
	b.WriteString("Live inventory of this process:\n")
	if tools, ok := inv["tools"].([]string); ok {
		fmt.Fprintf(&b, "- tools registered: %s\n", strings.Join(tools, ", "))
	}
	if skills, ok := inv["skills"].([]string); ok {
		if len(skills) == 0 {
			b.WriteString("- approved skills: none yet\n")
		} else {
			fmt.Fprintf(&b, "- approved skills: %s\n", strings.Join(skills, ", "))
		}
	}
	if gates, ok := inv["gates"].(map[string]any); ok {
		fmt.Fprintf(&b, "- gates: os_access=%v recap=%v nudge=%v briefing=%v\n",
			gates["os_access"], gates["recap"], gates["nudge"], gates["briefing"])
	}
	if mc, ok := inv["model_choice"].(map[string]any); ok {
		fmt.Fprintf(&b, "- saved model choice: %v (%v)\n", mc["model"], mc["provider"])
	}
	fmt.Fprintf(&b, "- build: %v commit %v\n", inv["version"], inv["commit"])
	return b.String()
}

func renderSource() (string, error) {
	text, err := Section("source")
	if err != nil {
		return "", err
	}
	dir, srcErr := SourceDir()
	if srcErr != nil {
		return fmt.Sprintf("Source seam: NOT available — %s.\n\n%s", srcErr, text), nil
	}
	return fmt.Sprintf("Source seam: available at %s.\n\n%s", dir, text), nil
}
