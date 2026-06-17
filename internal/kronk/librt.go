package kronk

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ardanlabs/kronk/sdk/tools/libs"
)

// runtimeVersion is the pinned llama.cpp release installed by InstallRuntime.
// Changing this is a reviewed code change — it aligns with what DownloadFor pins.
const runtimeVersion = "b9664"

// RuntimeVersion returns the pinned llama.cpp version used by InstallRuntime.
func RuntimeVersion() string { return runtimeVersion }

// installFn is an injectable seam so tests can replace the real SDK install
// without hitting the network. nil means use the real SDK path.
var installFn func(ctx context.Context, root, arch, goos, processor, version string, log libs.Logger) error

// InstallRuntime downloads and installs the llama.cpp runtime bundle for the
// given processor ("cpu" or "vulkan") into LibRoot(). It uses the host arch
// and OS, validates the triple via libs.IsSupported, and verifies the install
// via verifyInstall after download. Owner-initiated only — never called on boot.
func InstallRuntime(ctx context.Context, processor string, log libs.Logger) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if !libs.IsSupported(goarch, goos, processor) {
		return fmt.Errorf("runtime %s/%s/%s is not a supported build", goarch, goos, processor)
	}
	if installFn != nil {
		if err := installFn(ctx, LibRoot(), goarch, goos, processor, runtimeVersion, log); err != nil {
			return err
		}
		return verifyInstall(InstallDirFor(LibRoot(), goarch, goos, processor), runtimeVersion, goos, goarch, processor)
	}
	lib, err := libs.New(libs.WithLibPath(LibRoot()))
	if err != nil {
		return fmt.Errorf("resolving lib root: %w", err)
	}
	if _, err := lib.DownloadFor(ctx, log, goarch, goos, processor, runtimeVersion); err != nil {
		return fmt.Errorf("installing %s runtime: %w", processor, err)
	}
	return verifyInstall(InstallDirFor(LibRoot(), goarch, goos, processor), runtimeVersion, goos, goarch, processor)
}

// InstallDirFor returns the canonical install directory for a triple under root.
// Mirrors the SDK's private installPathFor: <root>/<os>/<arch>/<processor>/.
// Exported so settingscards can check per-variant install state.
func InstallDirFor(root, arch, goos, processor string) string {
	return filepath.Join(root, goos, arch, processor)
}

//go:embed runtime_sums.json
var runtimeSumsJSON []byte

// runtimeSums is the embedded checksum manifest. Keyed version → "os/arch/proc" → filename → sha256.
var runtimeSums map[string]map[string]map[string]string

func init() {
	if err := json.Unmarshal(runtimeSumsJSON, &runtimeSums); err != nil {
		panic("kronk: malformed runtime_sums.json: " + err.Error())
	}
}

// verifyInstall checks that the installed files in dir match the manifest.
// Files with an empty (placeholder) sha256 in the manifest are skipped —
// the reviewer fills real hashes at merge (Reviewer Note 3). A missing manifest
// entry for the version is an error (fail-closed). Files with a non-empty pinned
// hash that do not match cause dir to be deleted and an error to be returned.
func verifyInstall(dir, version, goos, arch, processor string) error {
	versionEntry, ok := runtimeSums[version]
	if !ok {
		return fmt.Errorf("verifyInstall: no manifest entry for version %q", version)
	}
	key := goos + "/" + arch + "/" + processor
	fileHashes, ok := versionEntry[key]
	if !ok {
		return fmt.Errorf("verifyInstall: no manifest entry for %q @ version %q", key, version)
	}
	for filename, want := range fileHashes {
		if want == "" {
			// Placeholder — filled at merge (Reviewer Note 3).
			continue
		}
		path := filepath.Join(dir, filename)
		got, err := sha256File(path)
		if err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("verifyInstall: reading %s: %w", filename, err)
		}
		if got != want {
			os.RemoveAll(dir)
			return fmt.Errorf("verifyInstall: sha256 mismatch for %s: want %s got %s", filename, want, got)
		}
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
