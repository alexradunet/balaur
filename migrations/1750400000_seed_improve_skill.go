package migrations

import (
	"embed"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

//go:embed skills/improve/SKILL.md skills/improve/references/*.md
var improveSkillFS embed.FS

func init() {
	m.Register(seedImproveSkillUp, seedImproveSkillDown)
}

func seedImproveSkillUp(app core.App) error {
	if recs, err := app.FindRecordsByFilter("skills", "name = {:name}", "", 1, 0, dbx.Params{"name": "improve"}); err != nil {
		return err
	} else if len(recs) > 0 {
		return nil
	}

	col, err := app.FindCollectionByNameOrId("skills")
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	content, err := improveSkillContent()
	if err != nil {
		return err
	}
	rec.Set("name", "improve")
	rec.Set("description", "Survey the codebase and write prioritized, self-contained implementation plans for another agent to execute.")
	rec.Set("content", content)
	rec.Set("when_to_use", "When asked to audit the codebase, find high-leverage improvements, suggest roadmap direction, or write implementation plans.")
	rec.Set("status", "active")
	rec.Set("enabled", true)
	if err := app.Save(rec); err != nil {
		return err
	}
	return nil
}

func seedImproveSkillDown(app core.App) error {
	recs, err := app.FindRecordsByFilter("skills", "name = {:name}", "", 1, 0, dbx.Params{"name": "improve"})
	if err != nil {
		return err
	}
	for _, rec := range recs {
		if err := app.Delete(rec); err != nil {
			return err
		}
	}
	return nil
}

func improveSkillContent() (string, error) {
	main, err := improveSkillRead("skills/improve/SKILL.md")
	if err != nil {
		return "", err
	}
	refs := []struct {
		title string
		path  string
	}{
		{"audit-playbook.md", "skills/improve/references/audit-playbook.md"},
		{"plan-template.md", "skills/improve/references/plan-template.md"},
		{"closing-the-loop.md", "skills/improve/references/closing-the-loop.md"},
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(main))
	b.WriteString("\n\n---\n\n")
	b.WriteString("## Balaur packaging note\n\n")
	b.WriteString("The upstream skill references local files. Balaur stores skills as one Markdown body, so those references are inlined below. When the instructions say to read a referenced file, use the matching inlined section.\n")
	for _, ref := range refs {
		b.WriteString("\n---\n\n")
		fmt.Fprintf(&b, "## Inlined reference: %s\n\n", ref.title)
		refContent, err := improveSkillRead(ref.path)
		if err != nil {
			return "", err
		}
		b.WriteString(strings.TrimSpace(refContent))
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func improveSkillRead(path string) (string, error) {
	b, err := improveSkillFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading embedded improve skill file %s: %w", path, err)
	}
	return string(b), nil
}
