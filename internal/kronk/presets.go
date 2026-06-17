package kronk

import (
	"os"
	"path/filepath"
)

// LibPath returns the llama.cpp library root (BALAUR_LIB_PATH). Empty hands
// resolution to Kronk's own default root. A directory that already contains a
// version.json is honored as-is; otherwise the per-processor variant lives at
// <root>/<os>/<arch>/<processor>/. Slice 1 never downloads it — a missing library
// surfaces as a plain error at first inference.
func LibPath() string { return os.Getenv("BALAUR_LIB_PATH") }

// Processor returns the llama.cpp build variant to load: "cpu" (default —
// deterministic, present on every supported box) or "vulkan" (offload to a
// Vulkan GPU; requires the host Vulkan loader + driver/ICD). Set via
// BALAUR_PROCESSOR. The choice selects which prebuilt variant dir is loaded; the
// Go binary is identical and CGO stays off either way.
func Processor() string {
	if p := os.Getenv("BALAUR_PROCESSOR"); p != "" {
		return p
	}
	return "cpu"
}

// ModelsDir returns the directory downloaded GGUF model files live in
// (BALAUR_MODELS_DIR). Empty falls back to the XDG data dir
// ~/.local/share/balaur/models. Lazy getter — no module-level global (AGENTS.md).
func ModelsDir() string {
	if d := os.Getenv("BALAUR_MODELS_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "models"
	}
	return filepath.Join(home, ".local", "share", "balaur", "models")
}
