package knowledge

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// Context injection policy — deliberately Pareto-simple, in two tiers:
//
//	tier 1 (upfront): active memories with importance >= 4, capped. The
//	  things the companion should never forget: who you are, hard
//	  constraints, standing preferences.
//	tier 2 (recall): active memories matching the current user message,
//	  capped. Cheap LIKE relevance today; the seam stays put when FTS5 or
//	  embeddings land.
//
// Skills are indexed, not injected: the model sees name + when-to-use lines
// and loads a skill's full content through the `skill` tool only when
// needed. Token budgets stay sane and the prompt stays inspectable.
const (
	upfrontLimit = 12
	recallLimit  = 6
)

// BuildContext assembles the knowledge block appended to the system prompt
// for one turn, and returns the records it injected (so the caller can
// Touch the ones that informed the reply).
func BuildContext(app core.App, userMessage string) (string, []*core.Record) {
	var b strings.Builder
	var used []*core.Record
	seen := map[string]bool{}

	upfront, err := UpfrontMemories(app, upfrontLimit)
	if err == nil && len(upfront) > 0 {
		b.WriteString("\n\n## What you remember (always relevant)\n")
		for _, m := range upfront {
			writeMemoryLine(&b, m)
			seen[m.Id] = true
			used = append(used, m)
		}
	}

	recalled, err := SearchActive(app, recallTerms(userMessage), recallLimit)
	if err == nil {
		var fresh []*core.Record
		for _, m := range recalled {
			if !seen[m.Id] {
				fresh = append(fresh, m)
			}
		}
		if len(fresh) > 0 {
			b.WriteString("\n## What you remember (recalled for this message)\n")
			for _, m := range fresh {
				writeMemoryLine(&b, m)
				used = append(used, m)
			}
		}
	}

	skills, err := ActiveSkills(app)
	if err == nil && len(skills) > 0 {
		b.WriteString("\n## Skills you know (load with the `skill` tool before using)\n")
		for _, s := range skills {
			fmt.Fprintf(&b, "- %s — %s\n", s.GetString("name"), firstNonEmpty(
				s.GetString("when_to_use"), s.GetString("description")))
		}
	}

	return b.String(), used
}

func writeMemoryLine(b *strings.Builder, m *core.Record) {
	line := m.GetString("title")
	if c := strings.TrimSpace(m.GetString("content")); c != "" && c != line {
		line += ": " + c
	}
	fmt.Fprintf(b, "- %s\n", compress(line, 400))
}

// recallTerms reduces a chat message to its few longest distinct words — a
// cheap proxy for salience that pairs well with LIKE matching. Replaced
// wholesale when real ranking lands; callers won't notice.
func recallTerms(msg string) []string {
	words := strings.FieldsFunc(strings.ToLower(msg), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r >= 0x80)
	})
	uniq := map[string]bool{}
	var candidates []string
	for _, w := range words {
		if len(w) >= 4 && !uniq[w] {
			uniq[w] = true
			candidates = append(candidates, w)
		}
	}
	// Longest first, keep the top 3.
	slices.SortStableFunc(candidates, func(a, b string) int {
		return cmp.Compare(len(b), len(a)) // longest first; stable for equal lengths
	})
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	return candidates
}

func compress(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
