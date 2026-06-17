// Package modelget provides a streaming GGUF downloader: HTTP Range resume,
// sha256 verification, and atomic rename. CGO-free, stdlib only.
package modelget

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Progress reports download state to the caller.
type Progress struct {
	Current     int64
	Total       int64
	BytesPerSec float64
	Done        bool
}

// freeBytesFunc is the injectable seam for the free-space check.
// Defaults to the build-tagged freeBytes; tests can override it.
var freeBytesFunc = func(dir string) (uint64, error) { return freeBytes(dir) }

// Fetch downloads url into destDir as fileName, resuming from a <fileName>.part
// if present (HTTP Range), streaming a sha256 it checks against wantSHA256, then
// atomically renames to the final path. Calls onProgress at most ~every 250ms.
// Honors ctx cancellation (a cancel leaves the .part for a later resume). On a
// checksum mismatch it deletes the .part and returns an error (never renames).
// wantSize is used only for the free-space pre-flight; pass 0 to skip.
// token is an optional Bearer token (never logged or embedded in the URL).
func Fetch(ctx context.Context, url, destDir, fileName, wantSHA256 string,
	wantSize int64, token string, onProgress func(Progress)) (finalPath string, err error) {

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("creating models dir: %w", err)
	}

	finalPath = filepath.Join(destDir, fileName)
	partPath := finalPath + ".part"

	// Dedupe: if the final file exists and is the expected size, return immediately.
	if wantSize > 0 {
		if fi, err := os.Stat(finalPath); err == nil && fi.Size() == wantSize {
			if onProgress != nil {
				onProgress(Progress{Current: wantSize, Total: wantSize, Done: true})
			}
			return finalPath, nil
		}
	} else {
		// No size known — skip dedupe
		if _, err := os.Stat(finalPath); err == nil {
			if onProgress != nil {
				onProgress(Progress{Done: true})
			}
			return finalPath, nil
		}
	}

	// Free-space pre-flight: need wantSize + ~10% headroom.
	if wantSize > 0 {
		avail, err := freeBytesFunc(destDir)
		if err == nil {
			need := uint64(wantSize) + uint64(wantSize/10)
			if avail < need {
				return "", fmt.Errorf("insufficient disk space: need %d bytes, have %d", need, avail)
			}
		}
	}

	// Resume: stat any existing .part file.
	var resumeAt int64
	if fi, err := os.Stat(partPath); err == nil {
		resumeAt = fi.Size()
	}

	// Seed the sha256 from already-downloaded bytes (resume path).
	h := sha256.New()
	if resumeAt > 0 {
		f, err := os.Open(partPath)
		if err != nil {
			resumeAt = 0 // can't read existing part — restart
		} else {
			if _, err := io.Copy(h, f); err != nil {
				_ = f.Close()
				resumeAt = 0
				h = sha256.New()
			} else {
				_ = f.Close()
			}
		}
	}

	// Open the .part file before issuing the HTTP request so a mid-flight cancel
	// always leaves a resumable .part on disk (even if 0 bytes were transferred).
	partFlag := os.O_CREATE | os.O_WRONLY
	if resumeAt > 0 {
		partFlag |= os.O_APPEND
	} else {
		partFlag |= os.O_TRUNC
	}
	part, err := os.OpenFile(partPath, partFlag, 0o644)
	if err != nil {
		return "", fmt.Errorf("opening .part file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		_ = part.Close()
		return "", fmt.Errorf("building request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if resumeAt > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeAt))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = part.Close()
		// Leave the .part for resume; return context error unwrapped so callers
		// can check errors.Is(err, context.Canceled).
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK && resumeAt > 0 {
		// Server ignored the Range header — restart download and hash.
		resumeAt = 0
		h = sha256.New()
		// Re-open the .part file with truncation since we're starting over.
		_ = part.Close()
		part, err = os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return "", fmt.Errorf("opening .part file for restart: %w", err)
		}
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		_ = part.Close()
		return "", fmt.Errorf("server returned %d", resp.StatusCode)
	}

	total := wantSize
	if resp.ContentLength > 0 {
		total = resumeAt + resp.ContentLength
	}

	var current int64 = resumeAt
	throttle := time.Now()
	startTime := time.Now()

	tr := io.TeeReader(resp.Body, h)
	buf := make([]byte, 32*1024)
	var loopErr error
	for {
		n, readErr := tr.Read(buf)
		if n > 0 {
			if _, writeErr := part.Write(buf[:n]); writeErr != nil {
				_ = part.Close()
				return "", fmt.Errorf("writing .part: %w", writeErr)
			}
			current += int64(n)
			if onProgress != nil && time.Since(throttle) >= 250*time.Millisecond {
				elapsed := time.Since(startTime).Seconds()
				var bps float64
				if elapsed > 0 {
					bps = float64(current-resumeAt) / elapsed
				}
				onProgress(Progress{Current: current, Total: total, BytesPerSec: bps})
				throttle = time.Now()
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			// Check cancellation — leave .part for resume.
			if ctx.Err() != nil {
				loopErr = ctx.Err()
			} else {
				loopErr = fmt.Errorf("reading response: %w", readErr)
			}
			break
		}
		// Check cancellation after each chunk — leave .part for resume.
		if ctx.Err() != nil {
			loopErr = ctx.Err()
			break
		}
	}

	// Always close the .part file; sync only when not cancelling.
	if loopErr == nil {
		if err := part.Sync(); err != nil {
			_ = part.Close()
			return "", fmt.Errorf("syncing .part: %w", err)
		}
	}
	_ = part.Close()

	// On cancellation or read error, leave .part for resume and return the error.
	if loopErr != nil {
		return "", loopErr
	}

	// Verify checksum. An empty wantSHA256 means the caller has no pinned hash
	// to check against (never the case for the official model, whose pin always
	// carries a real sha256) — anything non-empty is enforced fail-closed.
	if wantSHA256 != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if got != wantSHA256 {
			_ = os.Remove(partPath)
			return "", fmt.Errorf("sha256 mismatch: want %s got %s", wantSHA256, got)
		}
	}

	// Atomic rename: the final .gguf exists ONLY when complete + verified.
	if err := os.Rename(partPath, finalPath); err != nil {
		return "", fmt.Errorf("renaming .part to final: %w", err)
	}

	if onProgress != nil {
		onProgress(Progress{Current: current, Total: total, Done: true})
	}
	return finalPath, nil
}
