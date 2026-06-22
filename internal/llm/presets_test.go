package llm

import (
	"strings"
	"testing"
)

// store.SaveCloudModel field caps (internal/store/llm_settings.go:166-176),
// inlined here so the test does not import internal/store.
const (
	maxName       = 80
	maxLabel      = 80
	maxChatModel  = 200
	maxEmbedModel = 200
	maxBaseURL    = 2048
)

func TestCloudPresetsCatalog(t *testing.T) {
	presets := CloudPresets()
	if len(presets) == 0 {
		t.Fatal("CloudPresets() is empty")
	}

	keys := map[string]CloudPreset{}
	for _, p := range presets {
		keys[p.Key] = p
	}
	for _, want := range []string{"mistral", "openai"} {
		if _, ok := keys[want]; !ok {
			t.Errorf("catalog missing key %q", want)
		}
	}

	t.Run("exactly one default and it is mistral", func(t *testing.T) {
		var defaults []string
		for _, p := range presets {
			if p.Default {
				defaults = append(defaults, p.Key)
			}
		}
		if len(defaults) != 1 {
			t.Fatalf("want exactly one Default==true, got %v", defaults)
		}
		if defaults[0] != "mistral" {
			t.Errorf("default preset = %q, want %q", defaults[0], "mistral")
		}
	})

	t.Run("base URLs are https with no trailing slash", func(t *testing.T) {
		for _, p := range presets {
			if !strings.HasPrefix(p.BaseURL, "https://") {
				t.Errorf("%s: BaseURL %q must start with https://", p.Key, p.BaseURL)
			}
			if strings.HasSuffix(p.BaseURL, "/") {
				t.Errorf("%s: BaseURL %q must not end with /", p.Key, p.BaseURL)
			}
		}
	})

	t.Run("field lengths satisfy SaveCloudModel caps", func(t *testing.T) {
		for _, p := range presets {
			if len(p.Name) > maxName {
				t.Errorf("%s: Name len %d > %d", p.Key, len(p.Name), maxName)
			}
			if len(p.Label) > maxLabel {
				t.Errorf("%s: Label len %d > %d", p.Key, len(p.Label), maxLabel)
			}
			if len(p.ChatModel) > maxChatModel {
				t.Errorf("%s: ChatModel len %d > %d", p.Key, len(p.ChatModel), maxChatModel)
			}
			if len(p.EmbedModel) > maxEmbedModel {
				t.Errorf("%s: EmbedModel len %d > %d", p.Key, len(p.EmbedModel), maxEmbedModel)
			}
			if len(p.BaseURL) > maxBaseURL {
				t.Errorf("%s: BaseURL len %d > %d", p.Key, len(p.BaseURL), maxBaseURL)
			}
		}
	})
}

func TestCloudPresetByKey(t *testing.T) {
	t.Run("known key", func(t *testing.T) {
		p, ok := CloudPresetByKey("mistral")
		if !ok {
			t.Fatal("CloudPresetByKey(\"mistral\") ok=false, want true")
		}
		if p.Name != "Mistral" {
			t.Errorf("Name = %q, want %q", p.Name, "Mistral")
		}
	})

	t.Run("unknown key returns zero value", func(t *testing.T) {
		p, ok := CloudPresetByKey("nope")
		if ok {
			t.Error("CloudPresetByKey(\"nope\") ok=true, want false")
		}
		if p != (CloudPreset{}) {
			t.Errorf("unknown key returned non-zero preset %+v", p)
		}
	})
}
