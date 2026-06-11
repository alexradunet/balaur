// Package self is Balaur's self-awareness: the embedded self-knowledge
// document, the build stamp, the live capability inventory, and the seam
// to its own source tree. Everything here is deterministic and read-only —
// self-knowledge is consulted, never guessed; self-modification happens
// through the ordinary OS tools under their own gate.
package self

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/store"
)

//go:embed knowledge.md
var knowledgeMD string

// sourceModule is the module path that identifies a checkout as Balaur's
// own source — the BALAUR_SOURCE seam refuses anything else.
const sourceModule = "github.com/alexradunet/balaur"

// SectionNames lists the self-knowledge sections in document order.
func SectionNames() []string {
	return []string{"overview", "architecture", "capabilities", "source", "devloop"}
}

// Section returns one self-knowledge section by its lowercase name.
func Section(name string) (string, error) {
	want := strings.ToLower(strings.TrimSpace(name))
	lines := strings.Split(knowledgeMD, "\n")
	var b strings.Builder
	in := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if in {
				break
			}
			in = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "## ")), want)
			continue
		}
		if in {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	if !in && b.Len() == 0 {
		return "", fmt.Errorf("self: no section %q (have: %s)", name, strings.Join(SectionNames(), ", "))
	}
	return strings.TrimSpace(b.String()), nil
}

// Info is the build stamp of the running binary, read from the build
// metadata the Go toolchain embeds (VCS stamping needs a git checkout at
// build time; fields degrade to "unknown" rather than failing).
type Info struct {
	Version string
	Commit  string
	Built   string
	Go      string
}

// BuildInfo reports the running binary's build stamp.
func BuildInfo() Info {
	info := Info{Version: "unknown", Commit: "unknown", Built: "unknown", Go: "unknown"}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	info.Go = bi.GoVersion
	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = bi.Main.Version
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			info.Commit = s.Value
		case "vcs.time":
			info.Built = s.Value
		}
	}
	return info
}

// SourceDir resolves and validates the BALAUR_SOURCE seam: the path must
// exist and contain Balaur's own go.mod. The error doubles as setup
// guidance for the owner.
func SourceDir() (string, error) {
	dir := os.Getenv("BALAUR_SOURCE")
	if dir == "" {
		return "", fmt.Errorf("BALAUR_SOURCE is not set: point it at Balaur's source checkout to enable self-development")
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("BALAUR_SOURCE=%s does not look like a Go module: %w", dir, err)
	}
	first, _, _ := strings.Cut(string(data), "\n")
	if strings.TrimSpace(first) != "module "+sourceModule {
		return "", fmt.Errorf("BALAUR_SOURCE=%s is not the Balaur source (module line is %q, want %q)", dir, strings.TrimSpace(first), "module "+sourceModule)
	}
	return dir, nil
}

// Inventory is the live capability report: what is actually registered
// and enabled in this process right now. toolNames is supplied by the
// caller that assembled the tool set (the registry includes the self tool
// itself), so the report never drifts from reality.
func Inventory(app core.App, toolNames []string) map[string]any {
	names := append([]string(nil), toolNames...)
	sort.Strings(names)

	info := BuildInfo()
	inv := map[string]any{
		"version": info.Version,
		"commit":  info.Commit,
		"built":   info.Built,
		"go":      info.Go,
		"tools":   names,
		"gates": map[string]any{
			"os_access": os.Getenv("BALAUR_OS_ACCESS") == "1",
			"recap":     os.Getenv("BALAUR_RECAP") != "0",
			"nudge":     os.Getenv("BALAUR_NUDGE") != "0",
			"briefing":  os.Getenv("BALAUR_BRIEFING") != "0",
		},
	}

	skills := []string{}
	if recs, err := knowledge.ActiveSkills(app); err == nil {
		for _, r := range recs {
			skills = append(skills, r.GetString("name"))
		}
	}
	inv["skills"] = skills

	if choice, ok, err := store.ActiveLLMChoice(app); err == nil && ok {
		inv["model_choice"] = map[string]any{"provider": choice.Provider, "model": choice.Model}
	}

	src := map[string]any{"configured": os.Getenv("BALAUR_SOURCE") != ""}
	if dir, err := SourceDir(); err == nil {
		src["path"] = dir
		src["ok"] = true
	} else {
		src["ok"] = false
		src["error"] = err.Error()
	}
	inv["source"] = src

	return inv
}
