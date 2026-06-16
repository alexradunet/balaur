package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
)

func TestInstallModel(t *testing.T) {
	dir := t.TempDir()
	gguf := filepath.Join(dir, "tiny.gguf")
	if err := os.WriteFile(gguf, []byte("GGUF"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("missing file is rejected", func(t *testing.T) {
		app := newWebApp(t)
		s := tests.ApiScenario{
			Name:            "install missing gguf",
			Method:          "POST",
			URL:             "/ui/model/install",
			Body:            strings.NewReader("path=/no/such/model.gguf"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"file not found"},
		}
		s.Test(t)
	})

	t.Run("non-absolute path is rejected", func(t *testing.T) {
		app := newWebApp(t)
		s := tests.ApiScenario{
			Name:            "install relative path",
			Method:          "POST",
			URL:             "/ui/model/install",
			Body:            strings.NewReader("path=models/x.gguf"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"absolute .gguf"},
		}
		s.Test(t)
	})

	t.Run("valid gguf is installed and activated", func(t *testing.T) {
		app := newWebApp(t)
		s := tests.ApiScenario{
			Name:            "install gguf",
			Method:          "POST",
			URL:             "/ui/model/install",
			Body:            strings.NewReader("path=" + gguf),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"models-panel", "tiny.gguf"},
			AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
				cfg, ok, _ := store.ActiveLLMConfig(a)
				if !ok || cfg.ChatModel != gguf {
					tb.Errorf("active model chat_model = %q (ok=%v), want %q", cfg.ChatModel, ok, gguf)
				}
			},
		}
		s.Test(t)
	})
}
