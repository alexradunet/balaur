// Package export is Balaur's sovereign-export spike (plan 192): a one-way,
// read-only renderer of the active knowledge record to Markdown. It is the thin
// slice of the Johnny Decimal vault mirror — ONE node type, no git, no
// encryption. The redaction boundary is hard: it reads ONLY the `nodes`
// collection, ONLY status=active rows, and never touches any secret/token
// collection (api_key, OAuth tokens, vault entries — see the design note at
// docs/superpowers/specs/2026-06-25-sovereign-export-design.md).
package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
)

// ExportType renders every ACTIVE node of one type to a Markdown file under
// destDir (one file per node: YAML frontmatter from props + an H1 title + the
// node body, which already carries [[wikilinks]]). It returns the relative file
// paths written. It is read-only on PocketBase: it calls only the active-filtered
// nodes.ListByTypeStatus reader and writes only under destDir. It never reads
// llm_providers, extensions, owner_settings, or any token/secret collection.
func ExportType(app core.App, typ, destDir string) ([]string, error) {
	// The consent filter: status=active is non-negotiable. Proposed, rejected,
	// and archived nodes are never read, so they can never be exported.
	recs, err := nodes.ListByTypeStatus(app, typ, nodes.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("export: listing %q nodes: %w", typ, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("export: making %s: %w", destDir, err)
	}

	used := map[string]bool{}
	paths := make([]string, 0, len(recs))
	for _, rec := range recs {
		name := uniqueName(slug(rec.GetString("title"), rec.Id), rec.Id, used)
		path := filepath.Join(destDir, name)
		if err := os.WriteFile(path, render(rec), 0o644); err != nil {
			return nil, fmt.Errorf("export: writing %s: %w", path, err)
		}
		paths = append(paths, name)
	}
	return paths, nil
}

// render builds the Markdown for one node: YAML frontmatter (type/status/
// created/updated + each props scalar), an H1 title, then the body verbatim
// (it already carries [[wikilinks]]).
func render(rec *core.Record) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(yamlLine("type", rec.GetString("type")))
	b.WriteString(yamlLine("status", rec.GetString("status")))
	b.WriteString(yamlLine("created", rec.GetDateTime("created").String()))
	b.WriteString(yamlLine("updated", rec.GetDateTime("updated").String()))
	// Props in a stable order so the export is deterministic (full re-export
	// must produce byte-identical output for unchanged data).
	props := nodes.Props(rec)
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		b.WriteString(yamlLine(k, scalar(props[k])))
	}
	b.WriteString("---\n\n")
	b.WriteString("# " + rec.GetString("title") + "\n\n")
	b.WriteString(rec.GetString("body"))
	b.WriteString("\n")
	return []byte(b.String())
}

// yamlLine writes one `key: "value"` frontmatter line. The value is always
// %q-quoted so a `:`, quote, or newline in titles/values cannot break the YAML.
func yamlLine(key, value string) string {
	return fmt.Sprintf("%s: %q\n", key, value)
}

// scalar renders a props value as a frontmatter scalar. Strings/numbers/bools
// stringify directly; a nested map or slice is JSON-encoded inline (nested-prop
// frontmatter is deferred to Phase 2 — see the design note).
func scalar(v any) string {
	switch v.(type) {
	case map[string]any, []any:
		if raw, err := json.Marshal(v); err == nil {
			return string(raw)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// slug lowercases the title and collapses runs of non-alphanumerics to a single
// '-'. An empty result falls back to the node id so every file has a name.
func slug(title, id string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return id
	}
	return s
}

// uniqueName resolves a slug collision within the destination by appending the
// node id, which is unique per record.
func uniqueName(base, id string, used map[string]bool) string {
	name := base + ".md"
	if used[name] {
		name = base + "-" + id + ".md"
	}
	used[name] = true
	return name
}
