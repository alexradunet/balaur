package kronk

import (
	"os"
	"path/filepath"
)

// RuntimeInstalled reports whether the native llama.cpp library is present at
// the configured path. It never loads or dlopen's anything — just a stat check.
// Returns false when BALAUR_LIB_PATH is unset and the default dir is empty.
func RuntimeInstalled() bool {
	dir, err := resolveLibDir(LibPath(), Processor())
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "libllama.so")) // linux; non-linux is plan 087
	return err == nil
}
