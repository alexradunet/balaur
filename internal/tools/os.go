// Package tools provides the agent's built-in tools. OS access mode — the
// four Pi-default tools read, write, edit, bash — ships DISABLED; the owner
// turns it on explicitly, and every invocation is audited (AGENTS.md).
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
)

// maxOutput bounds tool output fed back to the model.
const maxOutput = 48 * 1024

// OSAccess builds the OS tool set. Callers gate this behind the owner's
// explicit setting; the package itself stays policy-free except auditing.
func OSAccess(app core.App) []agent.Tool {
	return []agent.Tool{
		readTool(app),
		writeTool(app),
		editTool(app),
		bashTool(app),
	}
}

func obj(props map[string]any, required ...string) map[string]any {
	// A variadic with no args is a nil slice, which marshals to "required":
	// null. JSON Schema wants an array, and llama.cpp's tool-call grammar
	// generation rejects null ("type must be array, but is null"). Emit [].
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

func clip(s string) string {
	if len(s) <= maxOutput {
		return s
	}
	return s[:maxOutput] + "\n…(truncated)"
}

// auditOS records one OS tool invocation through the shared audit helper.
func auditOS(app core.App, tool, target string, allowed bool, detail map[string]any) {
	store.Audit(app, "os", "os."+tool, target, allowed, detail)
}

func readTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("read", "Read a file from disk.", obj(map[string]any{
			"path": str("Absolute or working-directory-relative file path."),
		}, "path")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct{ Path string }
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("read: bad arguments: %w", err)
			}
			data, err := os.ReadFile(args.Path)
			auditOS(app, "read", args.Path, err == nil, nil)
			if err != nil {
				return "", err
			}
			return clip(string(data)), nil
		},
	}
}

func writeTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("write", "Create or overwrite a file with the given content.", obj(map[string]any{
			"path":    str("File path to write."),
			"content": str("Full file content."),
		}, "path", "content")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct{ Path, Content string }
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("write: bad arguments: %w", err)
			}
			err := os.WriteFile(args.Path, []byte(args.Content), 0o644)
			auditOS(app, "write", args.Path, err == nil, map[string]any{"bytes": len(args.Content)})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("wrote %d bytes to %s", len(args.Content), args.Path), nil
		},
	}
}

func editTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("edit", "Replace an exact string in a file. The old string must appear exactly once.", obj(map[string]any{
			"path": str("File path to edit."),
			"old":  str("Exact text to replace (must be unique in the file)."),
			"new":  str("Replacement text."),
		}, "path", "old", "new")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct{ Path, Old, New string }
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("edit: bad arguments: %w", err)
			}
			data, err := os.ReadFile(args.Path)
			if err != nil {
				auditOS(app, "edit", args.Path, false, nil)
				return "", err
			}
			text := string(data)
			switch strings.Count(text, args.Old) {
			case 0:
				auditOS(app, "edit", args.Path, false, map[string]any{"reason": "old string not found"})
				return "", fmt.Errorf("edit: old string not found in %s", args.Path)
			case 1: // ok
			default:
				auditOS(app, "edit", args.Path, false, map[string]any{"reason": "old string not unique"})
				return "", fmt.Errorf("edit: old string appears more than once in %s", args.Path)
			}
			err = os.WriteFile(args.Path, []byte(strings.Replace(text, args.Old, args.New, 1)), 0o644)
			auditOS(app, "edit", args.Path, err == nil, nil)
			if err != nil {
				return "", err
			}
			return "edited " + args.Path, nil
		},
	}
}

func bashTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("bash", "Run a shell command and return its combined output.", obj(map[string]any{
			"command": str("The command to run with `sh -c`."),
		}, "command")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct{ Command string }
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("bash: bad arguments: %w", err)
			}
			runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			out, err := exec.CommandContext(runCtx, "sh", "-c", args.Command).CombinedOutput()
			auditOS(app, "bash", args.Command, err == nil, map[string]any{"bytes": len(out)})
			if err != nil {
				// Output often explains the failure; give the model both.
				return "", fmt.Errorf("bash: %w\n%s", err, clip(string(out)))
			}
			return clip(string(out)), nil
		},
	}
}
