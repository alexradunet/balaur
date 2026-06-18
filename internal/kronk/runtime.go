package kronk

import (
	"os"
	"path/filepath"
)

// RuntimeInstalled reports whether the native llama.cpp library is present for
// the active processor. It never loads or dlopen's anything — just a stat check.
// Returns false when BALAUR_LIB_PATH is unset and the default dir is empty.
func RuntimeInstalled() bool {
	return RuntimeInstalledFor(Processor())
}

// RuntimeInstalledFor reports whether the native library for a specific
// processor variant (cpu|vulkan) is present — a stat check only, never a
// dlopen. resolveProcessor uses it to keep a saved/env choice from stranding
// inference when that variant's runtime isn't actually installed.
func RuntimeInstalledFor(processor string) bool {
	dir, err := resolveLibDir(LibRoot(), processor)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "libllama.so")) // linux; non-linux is plan 087
	return err == nil
}
