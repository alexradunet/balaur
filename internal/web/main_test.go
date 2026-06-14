package web

import (
	"os"
	"testing"
)

// TestMain makes the web test package hermetic against any live Ollama daemon
// on the host. Model-pull handlers (e.g. POST /ui/model/gguf/download) launch a
// real background pull goroutine whose onDone callback writes to the DB; on a
// box actually running Ollama that callback can complete and fire after a
// test's app has been torn down, panicking in a PocketBase query. Pointing the
// engine at an unreachable host makes every pull fail fast (connection refused)
// so onDone never fires. Mirrors the BALAUR_OLLAMA_HOST=127.0.0.1:1 pattern used
// by TestModelChoicesBareBox in internal/turn.
func TestMain(m *testing.M) {
	os.Setenv("BALAUR_OLLAMA_HOST", "127.0.0.1:1")
	os.Exit(m.Run())
}
