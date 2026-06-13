package ext

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dop251/goja"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
)

var extNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,60}$`)

// ProposeTool is the model's verb for growing a new capability: write a
// balaur-extension and submit it for consent. Proposing is always allowed
// (like remember) because nothing runs until the owner approves; an
// ACTIVE extension can never be silently replaced through this path.
func ProposeTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("propose_extension",
			"Propose a new balaur-extension: one JavaScript file that registers new tools for you. "+
				"It runs ONLY after the owner approves it — never claim the capability exists before then. "+
				"The file calls balaur.registerTool({name, description, parameters, handler}); handlers may "+
				"use balaur.http({url, method, headers, body}) and must return a string or JSON-serializable value. "+
				"No npm, no require, no filesystem.",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":        map[string]any{"type": "string", "description": "kebab-case extension name, e.g. weather-bucharest"},
					"description": map[string]any{"type": "string", "description": "one line: what this extension does"},
					"code":        map[string]any{"type": "string", "description": "the complete JavaScript file content"},
				},
				"required": []string{"name", "description", "code"},
			}),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Code        string `json:"code"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("propose_extension: bad arguments: %w", err)
			}
			args.Name = strings.TrimSpace(args.Name)
			if !extNameRe.MatchString(args.Name) {
				return "", fmt.Errorf("propose_extension: name must be kebab-case ([a-z0-9-], 2-61 chars), got %q", args.Name)
			}
			if len(args.Code) > maxFileSize {
				return "", fmt.Errorf("propose_extension: code is %d bytes; extensions are capped at %d", len(args.Code), maxFileSize)
			}
			// Syntax-check without executing: a proposal must at least parse.
			if _, err := goja.Compile(args.Name+".js", args.Code, true); err != nil {
				return "", fmt.Errorf("propose_extension: the code does not parse: %w", err)
			}
			if !strings.Contains(args.Code, "balaur.registerTool") {
				return "", fmt.Errorf("propose_extension: the code never calls balaur.registerTool — it would register nothing")
			}

			// An approved extension is the owner's; replacing its content
			// is a change the owner must see via the re-propose flow, not
			// a silent overwrite by the model.
			if rec, err := find(app, args.Name); err == nil && rec.GetString("status") == StatusActive {
				return "", fmt.Errorf("propose_extension: %q is active; propose under a new name or ask the owner to disable it first", args.Name)
			}

			dir := Dir(app)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", fmt.Errorf("propose_extension: %w", err)
			}
			path := filepath.Join(dir, args.Name+".js")
			if err := os.WriteFile(path, []byte(args.Code), 0o644); err != nil {
				return "", fmt.Errorf("propose_extension: %w", err)
			}

			_, sum, err := readHashed(path)
			if err != nil {
				return "", fmt.Errorf("propose_extension: %w", err)
			}
			rec, _ := find(app, args.Name)
			rec, err = upsertProposal(app, rec, args.Name, path, args.Code, sum, strings.TrimSpace(args.Description))
			if err != nil {
				return "", fmt.Errorf("propose_extension: %w", err)
			}
			if rec.GetString("source") == "" || rec.GetString("source") == "discovered" {
				rec.Set("source", "chat")
				if err := app.Save(rec); err != nil {
					app.Logger().Warn("ext proposal source update failed", "name", args.Name, "err", err)
				}
			}
			store.Audit(app, "", "model", "ext.propose", args.Name, true, map[string]any{"sha256": sum})

			return fmt.Sprintf("Extension %q proposed for the owner's approval (file: pb_extensions/%s.js). "+
				"It is NOT active: its tools do not exist until the owner runs `balaur ext approve %s` "+
				"or approves it in the dashboard. Do not claim this capability yet.",
				args.Name, args.Name, args.Name), nil
		},
	}
}
