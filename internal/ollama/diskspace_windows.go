//go:build windows

package ollama

// freeBytes is a no-op on Windows (not a deployment target): it reports plenty
// of free space so the pre-pull disk guard never blocks.
func freeBytes(path string) (uint64, error) {
	return ^uint64(0), nil
}
