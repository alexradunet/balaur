package web

import (
	"testing"
)

// TestEmbedAssetPresence verifies that new Hearthwood assets are included in
// the embedded FS so the single-binary build carries them.
func TestEmbedAssetPresence(t *testing.T) {
	assets := []string{
		"static/icons/scroll.png",
		"static/icons/tome.png",
		"static/icons/orb.png",
		"static/icons/key.png",
		"static/icons/quill.png",
		"static/icons/lens.png",
		"static/icons/shield.png",
		"static/fonts/piazzolla.ttf",
		"static/fonts/jersey-15.ttf",
		"static/board.js",
	}
	for _, path := range assets {
		f, err := FS.Open(path)
		if err != nil {
			t.Errorf("asset missing from embed FS: %s: %v", path, err)
			continue
		}
		f.Close()
	}
}
