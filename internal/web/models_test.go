package web

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

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

// TestModelsPanelOfficialCTAGate guards the official-model CTA gate. The CTA must
// follow REGISTRATION, not mere file presence — otherwise a downloaded-but-unknown
// file (a prior download whose DB record was lost) hides the only install path and
// strands the owner with a usable model the UI won't surface.
func TestModelsPanelOfficialCTAGate(t *testing.T) {
	official := kronk.Official()

	t.Run("fresh box: download CTA, not on disk", func(t *testing.T) {
		app := newWebApp(t)
		t.Setenv("BALAUR_MODELS_DIR", t.TempDir())
		v, err := settingscards.BuildModelsPanelView(app, "")
		if err != nil {
			t.Fatal(err)
		}
		if !v.ShowOfficialCTA {
			t.Error("fresh box must show the official CTA")
		}
		if v.OfficialOnDisk {
			t.Error("fresh box: OfficialOnDisk must be false")
		}
	})

	t.Run("downloaded but unregistered: CTA installs from disk", func(t *testing.T) {
		app := newWebApp(t)
		dir := t.TempDir()
		t.Setenv("BALAUR_MODELS_DIR", dir)
		if err := os.WriteFile(filepath.Join(dir, official.FileName), []byte("GGUF"), 0o644); err != nil {
			t.Fatal(err)
		}
		v, err := settingscards.BuildModelsPanelView(app, "")
		if err != nil {
			t.Fatal(err)
		}
		if !v.ShowOfficialCTA {
			t.Error("a downloaded-but-unregistered model must still surface the CTA")
		}
		if !v.OfficialOnDisk {
			t.Error("OfficialOnDisk must be true when the file is on disk")
		}
	})

	t.Run("registered + on disk: no CTA", func(t *testing.T) {
		app := newWebApp(t)
		dir := t.TempDir()
		t.Setenv("BALAUR_MODELS_DIR", dir)
		path := filepath.Join(dir, official.FileName)
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
		if v.ShowOfficialCTA {
			t.Error("a registered official model on disk must NOT show the CTA")
		}
	})
}
