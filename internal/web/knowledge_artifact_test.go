package web

// knowledge_artifact_test.go — HTTP tests for the Knowledge sidebar summons
// introduced in plan 095: memory category slices accessed via
// GET /ui/show/memory?category=... and GET /ui/show/skills. (Proposed memories
// live in the Review queue, not a standalone Awaiting view.)

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

func TestKnowledgeArtifacts(t *testing.T) {
	t.Run("GET /ui/show/memory?category=person → 200, k-active-grid, fixed category in @get", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "memory category=person → 200 nav-free slice",
			Method:         "GET",
			URL:            "/ui/show/memory?category=person",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"datastar-patch-elements",
				"k-active-grid",
				// fixed category baked into @get (& → &amp; in HTML)
				"&amp;category=person",
			},
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				// Confirm k-tabs is absent in the SSE body.
				// The response body is already consumed by ExpectedContent checks,
				// but the unit test TestKnowledgeFocusMemoryContract covers this
				// at the component level. HTTP test confirms routing + params.
				_ = res
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/memory?category=bogus → 400 Invalid card params", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:            "memory category=bogus → 400",
			Method:          "GET",
			URL:             "/ui/show/memory?category=bogus",
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

// TestKnowledgeArtifactRouting verifies routing + content for category and skills
// artifacts. Memory and skills artifacts render without an in-panel tab strip
// (plan 110); sub-views are reached via the /-command palette. The unit tests
// (TestKnowledgeFocusMemoryContract, TestKnowledgeFocusSkillsNoCategories) cover
// the component-level detail.
func TestKnowledgeArtifactRouting(t *testing.T) {
	t.Run("memory category=fact has no k-tabs", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:               "memory category=fact routing",
			Method:             "GET",
			URL:                "/ui/show/memory?category=fact",
			TestAppFactory:     newWebApp,
			ExpectedStatus:     200,
			ExpectedContent:    []string{"k-active-grid"},
			NotExpectedContent: []string{`class="k-tabs"`},
		}
		s.Test(t)
	})

	t.Run("memory category=person has no k-tabs", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:               "memory category=person routing",
			Method:             "GET",
			URL:                "/ui/show/memory?category=person",
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
