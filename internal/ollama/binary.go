package ollama

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// ollamaPinnedTag pins the Ollama release Balaur auto-installs. Never "latest"
// so an upstream release cannot silently break first-run. Update deliberately.
// Tag confirmed via: gh release view --repo ollama/ollama --json tagName,assets
const ollamaPinnedTag = "v0.30.8"

// assetName is the release asset for this host. The linux-amd64 asset ships as
// .tar.zst (confirmed from the v0.30.8 release assets).
// Only linux/amd64 release naming is supported (the deployment target); macOS/Windows assets use different naming.
func assetName() string {
	return fmt.Sprintf("ollama-%s-%s.tar.zst", runtime.GOOS, runtime.GOARCH)
}

func downloadURL() string {
	return fmt.Sprintf("https://github.com/ollama/ollama/releases/download/%s/%s", ollamaPinnedTag, assetName())
}

// BinaryPath resolves the ollama binary: BALAUR_OLLAMA, else <dataDir>/bin/ollama
// when present, else a PATH lookup. Returns the data-dir path (the install
// target) when none exists yet.
func BinaryPath(dataDir string) string {
	if p := os.Getenv("BALAUR_OLLAMA"); p != "" {
		return p
	}
	dataBin := filepath.Join(dataDir, "bin", "ollama")
	if _, err := os.Stat(dataBin); err == nil {
		return dataBin
	}
	if p, err := exec.LookPath("ollama"); err == nil {
		return p
	}
	return dataBin
}

// extractOllama extracts the `ollama` binary out of a release tarball (.tgz or
// .tar.zst) to dest, marking it executable. It scans for a tar entry whose base
// name is "ollama".
func extractOllama(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var decompressed io.Reader
	if strings.HasSuffix(archivePath, ".zst") {
		zr, err := zstd.NewReader(f)
		if err != nil {
			return err
		}
		defer zr.Close()
		decompressed = zr
	} else {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		decompressed = gz
	}

	tr := tar.NewReader(decompressed)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("ollama binary not found in %s", archivePath)
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != "ollama" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
}

// installBinary downloads the pinned release tarball and extracts ollama to
// dest. Returns dest on success.
func installBinary(ctx context.Context, dest string) (string, error) {
	tmp := dest + ".tar.download"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL(), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading ollama: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading ollama: status %d from %s", resp.StatusCode, downloadURL())
	}
	if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
		return "", err
	}
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	defer os.Remove(tmp)
	if err := extractOllama(tmp, dest); err != nil {
		return "", err
	}
	return dest, nil
}
