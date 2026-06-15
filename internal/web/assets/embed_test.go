package assets

import "testing"

// TestEmbedAssetPresence verifies the Hearthwood static assets are carried in
// the embedded FS so the single-binary build serves them.
func TestEmbedAssetPresence(t *testing.T) {
	paths := []string{
		"static/basm.css",
		"static/icons/scroll.png",
		"static/icons/tome.png",
		"static/fonts/piazzolla.ttf",
		"static/fonts/jersey-15.ttf",
	}
	for _, p := range paths {
		f, err := FS.Open(p)
		if err != nil {
			t.Errorf("asset missing from embed FS: %s: %v", p, err)
			continue
		}
		f.Close()
	}
}
