package web

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/store"
)

// fakeRuntimeLib makes RuntimeInstalledFor(processor) deterministic by pointing
// BALAUR_LIB_PATH at a temp root and (optionally) dropping a libllama.so into the
// variant's install dir. Returns the lib root.
func fakeRuntimeLib(t *testing.T, installed ...string) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("BALAUR_LIB_PATH", root)
	for _, proc := range installed {
		dir := kronk.InstallDirFor(root, runtime.GOARCH, runtime.GOOS, proc)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "libllama.so"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestSetProcessor: /ui/model/processor persists the owner's CPU/GPU choice and
// re-renders the panel. It cannot switch the live engine (the native library
// loads once per process) — it only saves the restart-pending preference, and it
// refuses a variant whose runtime isn't installed.
func TestSetProcessor(t *testing.T) {
	postProcessor := func(app *tests.TestApp, value string) tests.ApiScenario {
		return tests.ApiScenario{
			Method:         "POST",
			URL:            "/ui/model/processor",
			Body:           strings.NewReader("processor=" + value),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus: 200,
		}
	}

	t.Run("cpu is saved", func(t *testing.T) {
		app := newWebApp(t)
		s := postProcessor(app, "cpu")
		s.Name = "select cpu"
		s.ExpectedContent = []string{"models-panel"}
		s.AfterTestFunc = func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
			if got := store.GetOwnerSetting(a, "llm_processor", ""); got != "cpu" {
				tb.Errorf("llm_processor = %q, want %q", got, "cpu")
			}
		}
		s.Test(t)
	})

	t.Run("installed vulkan is saved", func(t *testing.T) {
		fakeRuntimeLib(t, "vulkan")
		app := newWebApp(t)
		s := postProcessor(app, "vulkan")
		s.Name = "select installed vulkan"
		s.ExpectedContent = []string{"models-panel"}
		s.AfterTestFunc = func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
			if got := store.GetOwnerSetting(a, "llm_processor", ""); got != "vulkan" {
				tb.Errorf("llm_processor = %q, want %q", got, "vulkan")
			}
		}
		s.Test(t)
	})

	t.Run("uninstalled vulkan is rejected", func(t *testing.T) {
		fakeRuntimeLib(t) // empty root: vulkan runtime absent
		app := newWebApp(t)
		s := postProcessor(app, "vulkan")
		s.Name = "reject uninstalled vulkan"
		s.ExpectedContent = []string{"installed yet — install it above first"}
		s.AfterTestFunc = func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
			if got := store.GetOwnerSetting(a, "llm_processor", "unset"); got != "unset" {
				tb.Errorf("uninstalled variant must not be saved, got %q", got)
			}
		}
		s.Test(t)
	})

	t.Run("invalid processor is rejected", func(t *testing.T) {
		app := newWebApp(t)
		s := postProcessor(app, "tpu")
		s.Name = "select bogus"
		s.ExpectedContent = []string{"processor must be cpu or vulkan"}
		s.AfterTestFunc = func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
			if got := store.GetOwnerSetting(a, "llm_processor", "unset"); got != "unset" {
				tb.Errorf("invalid value must not be saved, got %q", got)
			}
		}
		s.Test(t)
	})
}

// ctaFor returns the CTA for catalog key, or ok=false if none is offered.
func ctaFor(ctas []modelcards.OfficialCTA, key string) (modelcards.OfficialCTA, bool) {
	for _, c := range ctas {
		if c.Key == key {
			return c, true
		}
	}
	return modelcards.OfficialCTA{}, false
}

// TestModelsPanelOfficialCTAGate guards the curated-catalog CTA gate. A CTA must
// follow REGISTRATION, not mere file presence — otherwise a downloaded-but-unknown
// file (a prior download whose DB record was lost) hides the only install path and
// strands the owner with a usable model the UI won't surface.
func TestModelsPanelOfficialCTAGate(t *testing.T) {
	medium, _ := kronk.OfficialByKey("medium")

	t.Run("fresh box: every model offered, none on disk", func(t *testing.T) {
		app := newWebApp(t)
		t.Setenv("BALAUR_MODELS_DIR", t.TempDir())
		v, err := settingscards.BuildModelsPanelView(app, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(v.OfficialCTAs) != len(kronk.OfficialModels()) {
			t.Fatalf("fresh box must offer all %d curated models, got %d", len(kronk.OfficialModels()), len(v.OfficialCTAs))
		}
		for _, c := range v.OfficialCTAs {
			if c.OnDisk {
				t.Errorf("%s: OnDisk must be false on a fresh box", c.Key)
			}
		}
	})

	t.Run("downloaded but unregistered: CTA installs from disk", func(t *testing.T) {
		app := newWebApp(t)
		dir := t.TempDir()
		t.Setenv("BALAUR_MODELS_DIR", dir)
		if err := os.WriteFile(filepath.Join(dir, medium.FileName), []byte("GGUF"), 0o644); err != nil {
			t.Fatal(err)
		}
		v, err := settingscards.BuildModelsPanelView(app, "")
		if err != nil {
			t.Fatal(err)
		}
		c, ok := ctaFor(v.OfficialCTAs, "medium")
		if !ok {
			t.Fatal("a downloaded-but-unregistered model must still surface its CTA")
		}
		if !c.OnDisk {
			t.Error("OnDisk must be true when the file is on disk")
		}
	})

	t.Run("registered + on disk: no CTA for it", func(t *testing.T) {
		app := newWebApp(t)
		dir := t.TempDir()
		t.Setenv("BALAUR_MODELS_DIR", dir)
		path := filepath.Join(dir, medium.FileName)
		if err := os.WriteFile(path, []byte("GGUF"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := store.SaveLocalModel(app, path, ""); err != nil {
			t.Fatal(err)
		}
		v, err := settingscards.BuildModelsPanelView(app, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := ctaFor(v.OfficialCTAs, "medium"); ok {
			t.Error("a registered model on disk must NOT show its CTA")
		}
	})
}
