package modelcards

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/uitest"
)

func render(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}

// TestProcessorControlContract guards the "Run on" control's Datastar @post
// contract and the form-per-button markup (mirrors the avatar-picker tests), so
// a dropped __prevent / wrong contentType / missing hidden input is caught.
func TestProcessorControlContract(t *testing.T) {
	got := render(t, Panel(PanelView{
		ProcessorRunning: "cpu",
		RestartPending:   true,
		Processors: []ProcessorOption{
			{Key: "cpu", Installed: true, Selected: true},
			{Key: "vulkan", Installed: true},
		},
	}))
	for _, want := range []string{
		`id="processor-control"`,
		// form-per-button @post (gomponents escapes ' → &#39;)
		`data-on:submit__prevent="@post(&#39;/ui/model/processor&#39;, {contentType:&#39;form&#39;})"`,
		`type="hidden" name="processor" value="cpu"`,
		`type="hidden" name="processor" value="vulkan"`,
		// the selected pill is current AND disabled so it can't re-POST itself
		`aria-current="true"`,
		`disabled`,
		`Run on`,
		// restart note surfaces the running variant
		`Restart Balaur to apply`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("processor control missing %q in:\n%s", want, got)
		}
	}
}

// TestOfficialCTACards: each curated model renders a download/install card whose
// hidden "model" input carries the catalog key, posting to /ui/model/download. An
// on-disk model offers "Install" (no re-download); others show the size.
func TestOfficialCTACards(t *testing.T) {
	got := render(t, Panel(PanelView{
		OfficialCTAs: []OfficialCTA{
			{Key: "small", Name: "Qwen3.5 4B", Tagline: "Small & fast", Meta: "Q4_K_M · 4B · Apache-2.0", SizeLabel: "2.7 GB"},
			{Key: "medium", Name: "Gemma 4 E4B", Tagline: "Balanced", Meta: "Q4_K_M · E4B · Gemma", SizeLabel: "5.3 GB", OnDisk: true},
		},
	}))
	for _, want := range []string{
		`data-on:submit__prevent="@post(&#39;/ui/model/download&#39;, {contentType:&#39;form&#39;})"`,
		`type="hidden" name="model" value="small"`,
		`type="hidden" name="model" value="medium"`,
		`Qwen3.5 4B`,
		`Small &amp; fast`,
		`Download &amp; install · 2.7 GB`, // small: not on disk → download w/ size
		`Install`,                         // medium: on disk → install, no re-download
		`Already downloaded`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("official CTA cards missing %q in:\n%s", want, got)
		}
	}
}

// TestProcessorPillDisabledStates: an uninstalled variant says "install first";
// an unsupported one says "not available here" — never a form/button, and never
// the wrong copy.
func TestProcessorPillDisabledStates(t *testing.T) {
	t.Run("uninstalled → install first", func(t *testing.T) {
		got := render(t, processorPill(ProcessorOption{Key: "vulkan", Installed: false}))
		for _, want := range []string{`proc-pill proc-pill-disabled`, `aria-disabled="true"`, `install first`} {
			if !strings.Contains(got, want) {
				t.Errorf("uninstalled pill missing %q in: %s", want, got)
			}
		}
		if strings.Contains(got, "<form") || strings.Contains(got, "@post") {
			t.Errorf("uninstalled pill must not be submittable: %s", got)
		}
	})

	t.Run("unsupported → not available here", func(t *testing.T) {
		got := render(t, processorPill(ProcessorOption{Key: "vulkan", Unsupported: true}))
		if !strings.Contains(got, "not available here") {
			t.Errorf("unsupported pill must say 'not available here': %s", got)
		}
		if strings.Contains(got, "install first") {
			t.Errorf("unsupported pill must not say 'install first': %s", got)
		}
		if strings.Contains(got, "<form") {
			t.Errorf("unsupported pill must not be submittable: %s", got)
		}
	})
}
