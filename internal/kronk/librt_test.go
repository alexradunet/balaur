package kronk

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ardanlabs/kronk/sdk/tools/libs"
)

// TestResolverAgreement is the headline guard: the directory that InstallRuntime
// writes into must be the same directory that resolveLibDir (and therefore the
// engine) loads from. Tested with BALAUR_LIB_PATH set and unset (default
// LibRoot driven by HOME).
func TestResolverAgreement(t *testing.T) {
	cases := []struct {
		name       string
		setLibPath bool
	}{
		{"BALAUR_LIB_PATH set", true},
		{"BALAUR_LIB_PATH unset (default)", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setLibPath {
				t.Setenv("BALAUR_LIB_PATH", t.TempDir())
			} else {
				t.Setenv("BALAUR_LIB_PATH", "")
				t.Setenv("HOME", t.TempDir())
			}
			for _, proc := range []string{"cpu", "vulkan"} {
				installDir := filepath.Clean(InstallDirFor(LibRoot(), "amd64", "linux", proc))
				loadDir, err := resolveLibDir(LibRoot(), proc)
				if err != nil {
					t.Fatalf("resolveLibDir(%q): %v", proc, err)
				}
				loadDir = filepath.Clean(loadDir)
				if installDir != loadDir {
					t.Errorf("proc=%s: install dir %q != load dir %q", proc, installDir, loadDir)
				}
			}
		})
	}
}

// TestVerifyInstall_Match verifies a correctly installed fixture passes.
func TestVerifyInstall_Match(t *testing.T) {
	dir := t.TempDir()
	content := []byte("fake libllama content")
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	if err := os.WriteFile(filepath.Join(dir, "libllama.so"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	orig := runtimeSums
	defer func() { runtimeSums = orig }()
	runtimeSums = map[string]map[string]map[string]string{
		"testver": {"linux/amd64/cpu": {"libllama.so": hash}},
	}

	if err := verifyInstall(dir, "testver", "linux", "amd64", "cpu"); err != nil {
		t.Errorf("expected no error for matching hash, got: %v", err)
	}
}

// TestVerifyInstall_Mismatch verifies mismatch causes dir deletion + error.
func TestVerifyInstall_Mismatch(t *testing.T) {
	dir := t.TempDir()
	soPath := filepath.Join(dir, "libllama.so")
	if err := os.WriteFile(soPath, []byte("wrong content"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := runtimeSums
	defer func() { runtimeSums = orig }()
	runtimeSums = map[string]map[string]map[string]string{
		"testver": {"linux/amd64/cpu": {"libllama.so": "deadbeef0000000000000000000000000000000000000000000000000000dead"}},
	}

	if err := verifyInstall(dir, "testver", "linux", "amd64", "cpu"); err == nil {
		t.Error("expected error for sha256 mismatch")
	}
	if _, statErr := os.Stat(soPath); statErr == nil {
		t.Error("expected dir to be deleted after mismatch, but file still exists")
	}
}

// TestVerifyInstall_MissingVersion verifies an unknown version errors fail-closed.
func TestVerifyInstall_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	orig := runtimeSums
	defer func() { runtimeSums = orig }()
	runtimeSums = map[string]map[string]map[string]string{}

	if err := verifyInstall(dir, "nosuchver", "linux", "amd64", "cpu"); err == nil {
		t.Error("expected error for missing version entry")
	}
}

// TestVerifyInstall_EmptyHash verifies placeholder (empty) hashes are skipped.
func TestVerifyInstall_EmptyHash(t *testing.T) {
	dir := t.TempDir()
	orig := runtimeSums
	defer func() { runtimeSums = orig }()
	runtimeSums = map[string]map[string]map[string]string{
		"testver": {"linux/amd64/cpu": {"libllama.so": ""}},
	}

	if err := verifyInstall(dir, "testver", "linux", "amd64", "cpu"); err != nil {
		t.Errorf("expected empty hash to be skipped, got: %v", err)
	}
}

// TestInstallRuntime_Unsupported verifies an unsupported processor returns an error
// without hitting the network.
func TestInstallRuntime_Unsupported(t *testing.T) {
	if err := InstallRuntime(context.Background(), "bogus-proc", nil); err == nil {
		t.Error("expected error for unsupported processor 'bogus-proc'")
	}
}

// TestInstallRuntime_SeamWorks verifies the installFn seam is invoked by InstallRuntime.
func TestInstallRuntime_SeamWorks(t *testing.T) {
	goarch := runtime.GOARCH
	goos := runtime.GOOS
	if !libs.IsSupported(goarch, goos, "cpu") {
		t.Skipf("cpu not supported on %s/%s — seam test skipped", goos, goarch)
	}

	called := false
	origFn := installFn
	defer func() { installFn = origFn }()
	installFn = func(ctx context.Context, root, arch, goosArg, processor, version string, log libs.Logger) error {
		called = true
		return nil
	}

	t.Setenv("BALAUR_LIB_PATH", t.TempDir())

	orig := runtimeSums
	defer func() { runtimeSums = orig }()
	key := goos + "/" + goarch + "/cpu"
	runtimeSums = map[string]map[string]map[string]string{
		runtimeVersion: {key: {"libllama.so": ""}},
	}

	if err := InstallRuntime(context.Background(), "cpu", nil); err != nil {
		t.Errorf("InstallRuntime with seam: unexpected error: %v", err)
	}
	if !called {
		t.Error("installFn seam was not called")
	}
}
