package modelcards

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
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
