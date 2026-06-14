// Package ext implements balaur-extensions: runtime tools written as
// single JavaScript files under pb_extensions/, run by goja (the same
// pure-Go engine PocketBase's jsvm uses; CGO stays off). The extensions
// collection is the consent ledger — a file's code joins the agent loop
// only while its row is active AND the file still hashes to the approved
// sha256. Discovery proposes, the owner approves, any change re-proposes,
// every invocation is audited. Extensions add verbs, not privileges.
package ext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
)

// maxFileSize bounds one extension file (an extension is a small verb;
// anything bigger belongs in Go).
const maxFileSize = 128 * 1024

const (
	StatusProposed = "proposed"
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// Dir is where extension files live: pb_extensions/ next to the data dir,
// mirroring PocketBase's pb_hooks and pb_migrations conventions.
// BALAUR_EXT_DIR overrides it explicitly (owners who keep extension code
// elsewhere, and tests, which need isolation from the shared default).
func Dir(app core.App) string {
	if dir := os.Getenv("BALAUR_EXT_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(app.DataDir(), "..", "pb_extensions")
}

// Active is one approved, checksum-verified extension ready to serve.
type Active struct {
	Name  string
	Path  string
	Tools []ToolDef
	src   string
}

// Sync reconciles pb_extensions/ with the consent ledger and returns the
// servable set. New files become proposals (never loaded); changed files
// are re-proposed and dropped from service; vanished active files are
// disabled. Per-file failures are audited and skipped — one broken
// extension must never break the turn.
func Sync(app core.App) []Active {
	dir := Dir(app)
	files, _ := filepath.Glob(filepath.Join(dir, "*.js"))
	sort.Strings(files)

	seen := map[string]bool{}
	var out []Active
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".js")
		seen[name] = true

		src, sum, err := readHashed(path)
		if err != nil {
			store.Audit(app, "extensions", "ext.error", name, false, map[string]any{"error": err.Error()})
			continue
		}

		rec, err := find(app, name)
		switch {
		case err != nil: // first sight: propose, never load
			if _, err := upsertProposal(app, nil, name, path, src, sum, "discovered"); err == nil {
				store.Audit(app, "extensions", "ext.discover", name, true, map[string]any{"sha256": sum})
			}
		case rec.GetString("status") == StatusActive && rec.GetString("sha256") == sum:
			defs, err := extract(src, name)
			if err != nil {
				store.Audit(app, "extensions", "ext.error", name, false, map[string]any{"error": err.Error()})
				continue
			}
			out = append(out, Active{Name: name, Path: path, Tools: defs, src: src})
		case rec.GetString("status") == StatusActive: // approved content changed
			if _, err := upsertProposal(app, rec, name, path, src, sum, rec.GetString("description")); err == nil {
				store.Audit(app, "extensions", "ext.changed", name, false, map[string]any{"sha256": sum})
			}
		}
	}

	// Active rows whose file vanished lose their consent target: disable.
	recs, _ := app.FindRecordsByFilter("extensions", "status = {:s}", "", 0, 0,
		dbx.Params{"s": StatusActive})
	for _, rec := range recs {
		if !seen[rec.GetString("name")] {
			rec.Set("status", StatusDisabled)
			if err := app.Save(rec); err == nil {
				store.Audit(app, "extensions", "ext.missing", rec.GetString("name"), false, nil)
			}
		}
	}
	return out
}

// Tools bridges the servable set into agent tools. taken guards the
// namespace: an extension can never shadow a built-in (or another
// extension); collisions are audited and skipped.
func Tools(app core.App, taken map[string]bool) []agent.Tool {
	var out []agent.Tool
	for _, ext := range Sync(app) {
		for _, def := range ext.Tools {
			if taken[def.Name] {
				store.Audit(app, "extensions", "ext.collision", ext.Name+"/"+def.Name, false, nil)
				continue
			}
			taken[def.Name] = true
			out = append(out, extTool(app, ext, def))
		}
	}
	return out
}

func extTool(app core.App, ext Active, def ToolDef) agent.Tool {
	params := def.Parameters
	if params == nil {
		params = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	src, extName, toolName := ext.src, ext.Name, def.Name
	return agent.Tool{
		Spec: agent.ToolSpecOf(toolName, def.Description+" (balaur-extension: "+extName+")", params),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			res, err := invoke(ctx, src, extName, toolName, argsJSON)
			store.Audit(app, "extensions", "ext.invoke", extName+"/"+toolName, err == nil, nil)
			return res, err
		},
	}
}

// Approve is the owner's consent to the file's CURRENT content: it
// re-hashes now, pins that sha256, and activates — also how a disabled or
// re-proposed extension comes back. The content must load cleanly first.
func Approve(app core.App, name string) (*core.Record, error) {
	rec, err := find(app, name)
	if err != nil {
		return nil, fmt.Errorf("no extension %q — drop a .js file into %s or check `ext list`", name, "pb_extensions/")
	}
	src, sum, err := readHashed(rec.GetString("path"))
	if err != nil {
		return nil, fmt.Errorf("approving %s: %w", name, err)
	}
	defs, err := extract(src, name)
	if err != nil {
		return nil, fmt.Errorf("approving %s: refuses to load: %w", name, err)
	}
	if len(defs) == 0 {
		return nil, fmt.Errorf("approving %s: registers no tools", name)
	}
	names := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		names = append(names, map[string]any{"name": d.Name, "description": d.Description})
	}
	rec.Set("sha256", sum)
	rec.Set("status", StatusActive)
	rec.Set("tools", names)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("approving %s: %w", name, err)
	}
	store.Audit(app, "owner", "ext.approve", name, true, map[string]any{"sha256": sum})
	return rec, nil
}

// Disable turns an extension off without touching its file.
func Disable(app core.App, name string) (*core.Record, error) {
	rec, err := find(app, name)
	if err != nil {
		return nil, fmt.Errorf("no extension %q", name)
	}
	rec.Set("status", StatusDisabled)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("disabling %s: %w", name, err)
	}
	store.Audit(app, "owner", "ext.disable", name, true, nil)
	return rec, nil
}

func find(app core.App, name string) (*core.Record, error) {
	return app.FindFirstRecordByData("extensions", "name", strings.TrimSpace(name))
}

func readHashed(path string) (string, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", "", err
	}
	if info.Size() > maxFileSize {
		return "", "", fmt.Errorf("%s is %d bytes; extensions are capped at %d", path, info.Size(), maxFileSize)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(raw)
	return string(raw), hex.EncodeToString(sum[:]), nil
}

// upsertProposal writes the consent ledger row for a new or changed file.
func upsertProposal(app core.App, rec *core.Record, name, path, src, sum, description string) (*core.Record, error) {
	if rec == nil {
		col, err := app.FindCollectionByNameOrId("extensions")
		if err != nil {
			return nil, err
		}
		rec = core.NewRecord(col)
		rec.Set("name", name)
		rec.Set("source", "discovered")
	}
	if description == "" || description == "discovered" {
		description = headerDescription(src)
	}
	rec.Set("description", description)
	rec.Set("path", path)
	rec.Set("sha256", sum)
	rec.Set("status", StatusProposed)
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// headerDescription reads the conventional first-line comment:
// // balaur-extension: <what it does>
func headerDescription(src string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(src), "\n")
	if d, ok := strings.CutPrefix(first, "// balaur-extension:"); ok {
		return strings.TrimSpace(d)
	}
	return "(no description — first line should be: // balaur-extension: …)"
}
