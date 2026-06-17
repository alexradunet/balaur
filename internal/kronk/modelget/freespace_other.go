//go:build !linux

package modelget

// freeBytes is a stub on non-Linux platforms: free-space pre-flight is skipped.
// Plan 087 will add platform coverage when the runtime download lands.
func freeBytes(_ string) (uint64, error) {
	return 0, nil
}
