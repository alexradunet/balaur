package web

// knowledge_artifact_test.go — HTTP tests for the Knowledge panel summons:
// the single Memory slice (GET /ui/show/memory) and Skills (GET /ui/show/skills).
// Memory has no category axis (collapsed) and no standalone Awaiting view
// (proposed memories live in the Review queue).

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

func TestKnowledgeArtifacts(t *testing.T) {
	t.Run("GET /ui/show/memory → 200, k-active-grid", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "memory → 200 nav-free slice",
			Method:         "GET",
			URL:            "/ui/show/memory",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"datastar-patch-elements",
				"k-active-grid",
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/memory?mode=bogus → 400 Invalid card params", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:            "memory mode=bogus → 400 (bad enum still rejected)",
			Method:          "GET",
			URL:             "/ui/show/memory?mode=bogus",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  400,
			ExpectedContent: []string{"Invalid card params"},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/skills → 200, k-active-grid", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "skills → 200 nav-free",
			Method:         "GET",
			URL:            "/ui/show/skills",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"datastar-patch-elements",
				"k-active-grid",
			},
		}
		s.Test(t)
	})
}

// TestKnowledgeArtifactRouting verifies routing + content for the memory and
// skills artifacts. Both render without an in-panel tab strip (plan 110);
// sub-views are reached via the /-command palette. The unit tests
// (TestKnowledgeFocusMemoryContract, TestKnowledgeFocusSkillsNoCategories) cover
// the component-level detail.
func TestKnowledgeArtifactRouting(t *testing.T) {
	t.Run("memory has no k-tabs", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:               "memory routing",
			Method:             "GET",
			URL:                "/ui/show/memory",
			TestAppFactory:     newWebApp,
			ExpectedStatus:     200,
			ExpectedContent:    []string{"k-active-grid"},
			NotExpectedContent: []string{`class="k-tabs"`},
		}
		s.Test(t)
	})

	t.Run("skills no k-tabs", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:               "skills routing",
			Method:             "GET",
			URL:                "/ui/show/skills",
			TestAppFactory:     newWebApp,
			ExpectedStatus:     200,
			ExpectedContent:    []string{"k-active-grid"},
			NotExpectedContent: []string{`class="k-tabs"`},
		}
		s.Test(t)
	})
}
