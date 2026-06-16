package kronk

import "os"

// LibPath returns the directory holding the prebuilt llama.cpp shared library
// that yzma dlopens at runtime. Empty hands resolution to Kronk's own default
// (it honors a directory containing version.json read-only). Slice 1 never
// downloads it — a missing library surfaces as a plain error at first inference.
func LibPath() string { return os.Getenv("BALAUR_LIB_PATH") }
