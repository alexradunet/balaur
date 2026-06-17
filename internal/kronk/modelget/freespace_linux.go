//go:build linux

package modelget

import "syscall"

// freeBytes returns the available bytes on the filesystem holding dir.
// Returns (0, err) on failure. Uses syscall.Statfs (stdlib, no CGO).
func freeBytes(dir string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Bsize), nil
}
