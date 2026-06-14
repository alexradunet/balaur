# Ollama Follow-up Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the `extractOllama` lib-dropping bug and harden three rough edges (duplicate local-model rows, no pre-pull disk guard, `IsPulled` hitting the daemon every render).

**Architecture:** Four independent fixes in `internal/ollama` + one migration. Full-archive extraction replaces single-file extraction; a new migration dedups local-model rows; `Pull` gains a free-space floor; `IsPulled`/`List` gain a 3 s tags cache.

**Tech Stack:** Go 1.26, PocketBase migrations, `archive/tar` + `compress/gzip` + `klauspost/compress/zstd`, `syscall.Statfs`.

**Reference spec:** `docs/superpowers/specs/2026-06-14-ollama-followup-fixes-design.md`

---

## Task 1: Full-archive extraction (fix the `extractOllama` bug)

**Files:**
- Modify: `internal/ollama/binary.go` (`extractOllama` → `extractArchive`; `installBinary` signature; `EnsureInstalled` is in `manager.go`)
- Modify: `internal/ollama/manager.go` (`EnsureInstalled` call site only)
- Test: `internal/ollama/binary_test.go`

### Context
`extractOllama` copies only the single `ollama` tar entry and drops `lib/ollama/` (runner libs + version symlinks). The real release ships `bin/ollama` + `lib/ollama/`. Fix: extract the whole archive into a `destRoot`, preserving layout, handling dirs/files/symlinks, with a zip-slip guard.

- [ ] **Step 1: Rewrite the extract tests (`internal/ollama/binary_test.go`)**

Replace the existing `writeTestTgz`, `writeTestZst`, `TestExtractTgz`, and `TestExtractZst` with the versions below, and add `TestExtractArchiveRejectsZipSlip`. The helpers now take a list of entries (regular files + symlinks):

```go
type tarEntry struct {
	name     string
	data     []byte
	linkname string // if set, write a symlink instead of a regular file
}

func writeTarEntries(t *testing.T, tw *tar.Writer, entries []tarEntry) {
	t.Helper()
	for _, e := range entries {
		if e.linkname != "" {
			if err := tw.WriteHeader(&tar.Header{Name: e.name, Typeflag: tar.TypeSymlink, Linkname: e.linkname, Mode: 0o777}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := tw.WriteHeader(&tar.Header{Name: e.name, Mode: 0o755, Size: int64(len(e.data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
}

func writeTestTgz(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	writeTarEntries(t, tw, entries)
	tw.Close()
	gz.Close()
}

func writeTestZst(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw, err := zstd.NewWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(zw)
	writeTarEntries(t, tw, entries)
	tw.Close()
	zw.Close()
}

func assertExtracted(t *testing.T, root string) {
	t.Helper()
	bin := filepath.Join(root, "bin", "ollama")
	b, err := os.ReadFile(bin)
	if err != nil || string(b) != "ELF-fake" {
		t.Fatalf("bin/ollama = %q err=%v", b, err)
	}
	if info, _ := os.Stat(bin); info.Mode()&0o100 == 0 {
		t.Fatal("bin/ollama not executable")
	}
	lib := filepath.Join(root, "lib", "ollama", "libfoo.so")
	if b, err := os.ReadFile(lib); err != nil || string(b) != "LIB" {
		t.Fatalf("lib/ollama/libfoo.so = %q err=%v", b, err)
	}
	link := filepath.Join(root, "lib", "ollama", "libfoo.so.1")
	if lt, err := os.Readlink(link); err != nil || lt != "libfoo.so" {
		t.Fatalf("symlink = %q err=%v", lt, err)
	}
}

func extractEntries() []tarEntry {
	return []tarEntry{
		{name: "bin/ollama", data: []byte("ELF-fake")},
		{name: "lib/ollama/libfoo.so", data: []byte("LIB")},
		{name: "lib/ollama/libfoo.so.1", linkname: "libfoo.so"},
	}
}

func TestExtractTgz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "o.tgz")
	writeTestTgz(t, archive, extractEntries())
	root := filepath.Join(dir, "out")
	if err := extractArchive(archive, root); err != nil {
		t.Fatal(err)
	}
	assertExtracted(t, root)
}

func TestExtractZst(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "o.tar.zst")
	writeTestZst(t, archive, extractEntries())
	root := filepath.Join(dir, "out")
	if err := extractArchive(archive, root); err != nil {
		t.Fatal(err)
	}
	assertExtracted(t, root)
}

func TestExtractArchiveRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "evil.tgz")
	writeTestTgz(t, archive, []tarEntry{{name: "../evil.txt", data: []byte("pwned")}})
	root := filepath.Join(dir, "out")
	if err := extractArchive(archive, root); err == nil {
		t.Fatal("expected zip-slip rejection")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); err == nil {
		t.Fatal("zip-slip wrote a file outside destRoot")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ollama/ -run 'TestExtract'`
Expected: FAIL — `extractArchive` undefined (and the old `extractOllama` single-file tests are gone).

- [ ] **Step 3: Replace `extractOllama` with `extractArchive` in `internal/ollama/binary.go`**

Delete the entire `extractOllama` function and replace it with:

```go
// extractArchive extracts every entry of a release tarball (.tgz or .tar.zst)
// into destRoot, preserving the archive's bin/ + lib/ layout. The ollama binary
// resolves ../lib/ollama relative to itself, so the runner libs must travel
// with it — extracting only the binary yields a non-functional install.
func extractArchive(archivePath, destRoot string) error {
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

	cleanRoot := filepath.Clean(destRoot)
	tr := tar.NewReader(decompressed)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(cleanRoot, hdr.Name)
		// Zip-slip guard: every target must stay inside destRoot.
		if target != cleanRoot && !strings.HasPrefix(target, cleanRoot+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry %q escapes destination", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Link(filepath.Join(cleanRoot, hdr.Linkname), target); err != nil {
				return err
			}
		}
	}
}
```

Then replace the `installBinary` function with this (signature changes from `dest` to `dataDir`):

```go
// installBinary downloads the pinned release tarball and extracts the full
// archive (bin/ + lib/) into <dataDir>, returning the binary path.
func installBinary(ctx context.Context, dataDir string) (string, error) {
	binPath := filepath.Join(dataDir, "bin", "ollama")
	tmp := filepath.Join(dataDir, "ollama.tar.download")
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
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
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
	if err := extractArchive(tmp, dataDir); err != nil {
		return "", err
	}
	if err := os.Chmod(binPath, 0o755); err != nil {
		return "", err
	}
	return binPath, nil
}
```

- [ ] **Step 4: Update the `EnsureInstalled` call site in `internal/ollama/manager.go`**

Find:
```go
	// BinaryPath returned the install target (<dataDir>/bin/ollama); download it.
	return installBinary(ctx, path)
```
Replace with:
```go
	// Binary absent — install the full release (bin/ + lib/) into <dataDir>.
	return installBinary(ctx, dataDir)
```
(`path` is still used by the `LookPath` check above; leave that line. Only the `installBinary` call changes.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ollama/ -run 'TestExtract' && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/ollama/binary.go internal/ollama/binary_test.go internal/ollama/manager.go
go vet ./internal/ollama/
git add internal/ollama/binary.go internal/ollama/binary_test.go internal/ollama/manager.go
git commit -m "fix(ollama): extract full release archive (bin/ + lib/) on install"
```

---

## Task 2: Dedup duplicate local-model rows (migration)

**Files:**
- Create: `migrations/1750810000_dedup_local_models.go`

### Context
The `1750800000` migration rewrote each legacy path-based local model to `gemma4:e4b` without deduping, leaving N identical rows on boxes that had N legacy local models. This one-off migration collapses them. The shipped `1750800000` is NOT edited. No dedicated unit test (the `storetest` harness applies migrations at init with no dup data to seed, and an internal `package migrations` test cannot import `storetest` without an import cycle — this matches how `1750800000` was handled); correctness is verified by build + the migrations/store suites staying green (the migration is a no-op on a fresh DB) and a live check after merge.

- [ ] **Step 1: Write the migration**

```go
package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The 1750800000 migration rewrote each legacy path-based local model to the
// Ollama default tag without deduping, so a box with N legacy local models ends
// up with N identical rows (the model picker then shows N copies). Collapse
// duplicate chat_model rows per local provider to the oldest survivor; if a
// deleted row was the active model, repoint active_model to the survivor (same
// tag, so no behavior change). Fresh installs never create dupes.
func init() {
	m.Register(dedupLocalModelsUp, dedupLocalModelsDown)
}

func dedupLocalModelsUp(app core.App) error {
	providers, err := app.FindRecordsByFilter("llm_providers", "kind = 'local'", "", 0, 0)
	if err != nil {
		return nil // collection not yet created on this box
	}
	var settings *core.Record
	if s, err := app.FindFirstRecordByData("llm_settings", "key", "default"); err == nil {
		settings = s
	}
	for _, p := range providers {
		models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "created", 0, 0, dbx.Params{"p": p.Id})
		if err != nil {
			return err
		}
		seen := map[string]string{} // chat_model -> survivor id
		for _, mdl := range models {
			chat := mdl.GetString("chat_model")
			survivor, dup := seen[chat]
			if !dup {
				seen[chat] = mdl.Id
				continue
			}
			if settings != nil && settings.GetString("active_model") == mdl.Id {
				settings.Set("active_model", survivor)
				if err := app.Save(settings); err != nil {
					return err
				}
			}
			if err := app.Delete(mdl); err != nil {
				return err
			}
		}
	}
	return nil
}

func dedupLocalModelsDown(app core.App) error {
	return nil // one-way cleanup; deleted duplicates cannot be restored
}
```

- [ ] **Step 2: Confirm timestamp ordering + build**

Run: `ls migrations/ | sort | tail -3`
Expected: `1750810000_dedup_local_models.go` sorts after `1750800000...` (the current max). If some other migration with a higher prefix exists, bump `1750810000` above it.

Run: `go build ./migrations/ && go vet ./migrations/`
Expected: clean.

- [ ] **Step 3: Run migration + store suites (must stay green; no-op on fresh DB)**

Run: `go test ./migrations/ ./internal/store/`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
gofmt -w migrations/1750810000_dedup_local_models.go
git add migrations/1750810000_dedup_local_models.go
git commit -m "feat(migrations): dedup duplicate local-model rows"
```

---

## Task 3: Pre-pull disk-space guard

**Files:**
- Create: `internal/ollama/diskspace_unix.go`, `internal/ollama/diskspace_windows.go`
- Modify: `internal/ollama/manager.go` (`Pull` + helpers)
- Test: `internal/ollama/manager_test.go`

### Context
`Pull` starts a multi-GB download with no free-space check. Add a free-space floor before launching the pull goroutine. The exact model size isn't cheaply knowable, so this is a "fail fast on a near-full disk" guard with a conservative default.

- [ ] **Step 1: Write the failing tests (`internal/ollama/manager_test.go`)**

Add these tests (and add `"sync/atomic"` is NOT needed here; no new imports required for these two):

```go
func TestCheckDiskSpace(t *testing.T) {
	gb := uint64(1024 * 1024 * 1024)
	if err := checkDiskSpace(12, 20*gb); err != nil {
		t.Fatalf("20GB free, 12 min: want nil, got %v", err)
	}
	if err := checkDiskSpace(12, 5*gb); err == nil {
		t.Fatal("5GB free, 12 min: want error")
	}
}

func TestMinFreeGBOverride(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA_MIN_FREE_GB", "")
	if minFreeGB() != defaultMinFreeGB {
		t.Fatalf("default = %d, want %d", minFreeGB(), defaultMinFreeGB)
	}
	t.Setenv("BALAUR_OLLAMA_MIN_FREE_GB", "50")
	if minFreeGB() != 50 {
		t.Fatalf("override = %d, want 50", minFreeGB())
	}
	t.Setenv("BALAUR_OLLAMA_MIN_FREE_GB", "garbage")
	if minFreeGB() != defaultMinFreeGB {
		t.Fatalf("garbage override = %d, want default", minFreeGB())
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/ollama/ -run 'TestCheckDiskSpace|TestMinFreeGBOverride'`
Expected: FAIL — `checkDiskSpace`, `minFreeGB`, `defaultMinFreeGB` undefined.

- [ ] **Step 3: Create `internal/ollama/diskspace_unix.go`**

```go
//go:build unix

package ollama

import "syscall"

// freeBytes returns the bytes available to a non-root user on the filesystem
// containing path.
func freeBytes(path string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}
```

- [ ] **Step 4: Create `internal/ollama/diskspace_windows.go`**

```go
//go:build windows

package ollama

// freeBytes is a no-op on Windows (not a deployment target): it reports plenty
// of free space so the pre-pull disk guard never blocks.
func freeBytes(path string) (uint64, error) {
	return ^uint64(0), nil
}
```

- [ ] **Step 5: Add the guard helpers + wire into `Pull` (`internal/ollama/manager.go`)**

Add `"os"`, `"path/filepath"`, and `"strconv"` to the import block. Then add these helpers (e.g. just above `Pull`):

```go
const defaultMinFreeGB = 12

// minFreeGB is the free-space floor (GB) required before a pull, overridable
// via BALAUR_OLLAMA_MIN_FREE_GB.
func minFreeGB() int {
	if v := os.Getenv("BALAUR_OLLAMA_MIN_FREE_GB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return defaultMinFreeGB
}

// modelStorePath is the directory Ollama stores models in (OLLAMA_MODELS or
// ~/.ollama), resolved to its nearest existing ancestor for the free-space
// check (the store may not exist before the first pull).
func modelStorePath() string {
	dir := os.Getenv("OLLAMA_MODELS")
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, ".ollama")
		} else {
			dir = "."
		}
	}
	for {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// checkDiskSpace returns an error when free is below minGB gigabytes. Pure, for
// testability; the OS probe is freeBytes.
func checkDiskSpace(minGB int, free uint64) error {
	need := uint64(minGB) * 1024 * 1024 * 1024
	if free < need {
		return fmt.Errorf("insufficient disk space: %d GB free, need ≥ %d GB (set BALAUR_OLLAMA_MIN_FREE_GB to override)", free/(1024*1024*1024), minGB)
	}
	return nil
}
```

Then add the guard at the very start of `Pull`, before `m.mu.Lock()`:

```go
func (m *Manager) Pull(tag string, onDone func(tag string)) error {
	// Fail fast on a near-full disk rather than mid-download. A probe error
	// (statfs failure) is non-fatal — fall through and let the pull proceed.
	if free, err := freeBytes(modelStorePath()); err == nil {
		if err := checkDiskSpace(minFreeGB(), free); err != nil {
			return err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress.Active {
		return fmt.Errorf("a model pull is already in progress")
	}
	// ... rest unchanged ...
```

- [ ] **Step 6: Run tests to verify they pass + build (incl. windows)**

Run: `go test ./internal/ollama/ -run 'TestCheckDiskSpace|TestMinFreeGBOverride' && go build ./... && GOOS=windows go build ./internal/ollama/`
Expected: PASS, both builds clean.

- [ ] **Step 7: Commit**

```bash
gofmt -w internal/ollama/manager.go internal/ollama/manager_test.go internal/ollama/diskspace_unix.go internal/ollama/diskspace_windows.go
go vet ./internal/ollama/
git add internal/ollama/manager.go internal/ollama/manager_test.go internal/ollama/diskspace_unix.go internal/ollama/diskspace_windows.go
git commit -m "feat(ollama): pre-pull free-space guard (BALAUR_OLLAMA_MIN_FREE_GB)"
```

---

## Task 4: Short-TTL tags cache for `IsPulled`/`List`

**Files:**
- Modify: `internal/ollama/manager.go`
- Test: `internal/ollama/manager_test.go`

### Context
`IsPulled` is on the board-render hot path (called per page load via `turn.availableChoices`) and hits `/api/tags` every time. Add a 3 s cache, fetched without holding the mutex, invalidated on a successful pull and on delete.

- [ ] **Step 1: Write the failing test (`internal/ollama/manager_test.go`)**

Add `"sync/atomic"` to the import block, then add:

```go
func TestCachedTagsHitsServerOnceWithinTTL(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte(`{"models":[{"name":"gemma4:e4b","size":1}]}`))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	for i := 0; i < 3; i++ {
		ok, err := m.IsPulled("gemma4:e4b")
		if err != nil || !ok {
			t.Fatalf("IsPulled = %v %v", ok, err)
		}
	}
	if c := atomic.LoadInt32(&calls); c != 1 {
		t.Fatalf("server hit %d times within TTL, want 1 (cache)", c)
	}
	m.invalidateTags()
	if _, err := m.IsPulled("gemma4:e4b"); err != nil {
		t.Fatal(err)
	}
	if c := atomic.LoadInt32(&calls); c != 2 {
		t.Fatalf("server hit %d times after invalidate, want 2", c)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/ollama/ -run 'TestCachedTags'`
Expected: FAIL — `invalidateTags` undefined (and `IsPulled` not yet cached, so calls != 1).

- [ ] **Step 3: Add cache fields to the `Manager` struct (`internal/ollama/manager.go`)**

In the `Manager` struct, after the `// single-slot pull` block, add a tags-cache block:

```go
	// single-slot pull
	cancel   context.CancelFunc
	progress PullSnapshot
	onDone   func(tag string)

	// tags cache (board-render hot path)
	tagsCache   []Model
	tagsCacheAt time.Time
```

- [ ] **Step 4: Add `cachedTags` + `invalidateTags`, and route `IsPulled`/`List` through them**

Add (e.g. just above `List`):

```go
const tagsTTL = 3 * time.Second

// cachedTags returns the model list from a short-TTL cache so the board-render
// path (IsPulled) does not hit the daemon on every request. The network fetch
// runs WITHOUT the mutex held; a concurrent double-fetch is acceptable.
func (m *Manager) cachedTags() ([]Model, error) {
	m.mu.Lock()
	if !m.tagsCacheAt.IsZero() && time.Since(m.tagsCacheAt) < tagsTTL {
		out := append([]Model(nil), m.tagsCache...)
		m.mu.Unlock()
		return out, nil
	}
	m.mu.Unlock()

	models, err := m.apiClient().tags(context.Background())
	if err != nil {
		return nil, err // do not cache errors
	}
	m.mu.Lock()
	m.tagsCache = models
	m.tagsCacheAt = time.Now()
	m.mu.Unlock()
	return models, nil
}

// invalidateTags forces the next cachedTags call to refetch.
func (m *Manager) invalidateTags() {
	m.mu.Lock()
	m.tagsCacheAt = time.Time{}
	m.mu.Unlock()
}
```

Replace `List` and `IsPulled` bodies to use `cachedTags`:

```go
// List returns the models present in Ollama's local store (short-TTL cached).
func (m *Manager) List() ([]Model, error) {
	return m.cachedTags()
}

// IsPulled reports whether tag is present locally (short-TTL cached). A
// reachability failure is treated as "not pulled" so callers degrade to a
// "pull needed" prompt.
func (m *Manager) IsPulled(tag string) (bool, error) {
	models, err := m.cachedTags()
	if err != nil {
		return false, err
	}
	for _, mdl := range models {
		if mdl.Name == tag {
			return true, nil
		}
	}
	return false, nil
}
```

- [ ] **Step 5: Invalidate on successful pull + on delete**

In `runPull`, in the success branch (the `else` after the error check), add the cache reset INSIDE the already-held lock (do NOT call `invalidateTags()` there — it would deadlock on `m.mu`):

```go
	} else {
		m.progress.Done = true
		m.progress.Err = ""
		m.tagsCacheAt = time.Time{} // a new model exists; force a refetch
		cb = m.onDone
	}
```

Replace `Delete` to invalidate after a successful delete:

```go
// Delete removes a model tag from Ollama's store and invalidates the tags cache.
func (m *Manager) Delete(tag string) error {
	if err := m.apiClient().delete(context.Background(), tag); err != nil {
		return err
	}
	m.invalidateTags()
	return nil
}
```

- [ ] **Step 6: Run tests to verify they pass + full package + race**

Run: `go test ./internal/ollama/ -run 'TestCachedTags' && go test -race ./internal/ollama/`
Expected: PASS, race-clean.

- [ ] **Step 7: Commit**

```bash
gofmt -w internal/ollama/manager.go internal/ollama/manager_test.go
go vet ./internal/ollama/
git add internal/ollama/manager.go internal/ollama/manager_test.go
git commit -m "perf(ollama): short-TTL tags cache for IsPulled/List"
```

---

## Final verification

- [ ] `go build ./... && go test ./... && go vet ./...` — all green.
- [ ] `GOOS=windows go build ./...` — cross-compile clean (diskspace_windows + process_windows).
- [ ] `go test -race ./internal/ollama/` — race-clean.
- [ ] **Real-release verification of Fix 1** (the point of the whole change): in a throwaway temp dir, exercise the actual install against the pinned release and confirm the full layout + a working binary:
  ```bash
  TMP=$(mktemp -d); curl -fSL -o "$TMP/o.tar.zst" https://github.com/ollama/ollama/releases/download/v0.30.8/ollama-linux-amd64.tar.zst
  # extract via the same path the code uses (full archive), then:
  tar --zstd -xf "$TMP/o.tar.zst" -C "$TMP" && ls "$TMP/bin/ollama" "$TMP/lib/ollama" && "$TMP/bin/ollama" --version
  rm -rf "$TMP"
  ```
  Expected: `bin/ollama` + `lib/ollama/` present, `--version` prints `0.30.8`. (This mirrors what `extractArchive`/`installBinary` now do.)
