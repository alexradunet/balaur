# Ollama follow-up fixes — design

- **Date:** 2026-06-14
- **Status:** Approved design, ready for implementation planning
- **Builds on:** the Ollama migration (`docs/superpowers/specs/2026-06-14-ollama-local-inference-design.md`). Closes the known `extractOllama` bug + three deferred follow-ups.

## Goal

Make the Ollama auto-install actually work on a clean box, and harden three rough edges: duplicate local-model rows, no disk-space guard before a multi-GB pull, and `IsPulled` hitting the daemon on every page render.

## Fix 1 — `extractOllama` drops the runner libs (the real bug)

**Problem (confirmed empirically):** `internal/ollama/binary.go:extractOllama` extracts only the single tar entry whose base name is `ollama`, writing it to a file path. But the real Ollama release (`ollama-linux-amd64.tar.zst`, ~1.4 GB) ships `bin/ollama` **plus** `lib/ollama/` — ggml runner libs (`libggml-base.so`, `cuda_v12/`, `cuda_v13/`, …) including version symlinks (`libggml-base.so → .so.0 → .so.0.13.1`). Dropping `lib/` yields a binary that can't load its runners, so the app's auto-install produces a non-functional Ollama on a clean deploy box. (The dev box works only because Ollama was installed manually with a full-archive extract.)

**Fix:** replace `extractOllama(archivePath, dest)` with `extractArchive(archivePath, destRoot string) error` that extracts **every** entry into `destRoot`, preserving the archive's relative layout:
- `tar.TypeDir` → `os.MkdirAll(target, 0o755)`.
- `tar.TypeReg` → create file, `io.Copy`, `chmod(hdr.Mode)` (preserves the archive's exec bits, so `bin/ollama` stays `0755`).
- `tar.TypeSymlink` / `tar.TypeLink` → `os.Symlink(hdr.Linkname, target)` (remove any pre-existing target first). Required: the lib dir uses version symlinks.
- **Zip-slip guard:** reject any entry whose cleaned `filepath.Join(destRoot, hdr.Name)` escapes `destRoot` (return an error).

`installBinary` changes signature from `(ctx, dest)` to `(ctx context.Context, dataDir string) (string, error)`:
1. download the release to `<dataDir>/bin/ollama.tar.download` (MkdirAll the bin dir first),
2. `extractArchive(tmp, dataDir)` → yields `<dataDir>/bin/ollama` + `<dataDir>/lib/ollama/`,
3. `os.Chmod(<dataDir>/bin/ollama, 0o755)` defensively,
4. `os.Remove(tmp)`, return `<dataDir>/bin/ollama`.

`EnsureInstalled(ctx, dataDir)` calls `installBinary(ctx, dataDir)` (instead of passing the binary path). `BinaryPath` is unchanged.

**Tests (`binary_test.go`):** extend `writeTestTgz`/`writeTestZst` to accept a list of entries (regular files + a symlink). Update `TestExtractTgz`/`TestExtractZst` to build `bin/ollama` + `lib/ollama/libfoo.so` + a symlink `lib/ollama/libfoo.so.1 -> libfoo.so`, extract into a temp `destRoot`, and assert: `destRoot/bin/ollama` exists + executable, `destRoot/lib/ollama/libfoo.so` exists, and the symlink resolves. Add `TestExtractArchiveRejectsZipSlip` (an entry named `../evil` returns an error and writes nothing outside `destRoot`).

## Fix 2 — dedup duplicate local-model rows (new migration)

**Problem:** the `1750800000` migration rewrote each legacy path-based local model to `chat_model="gemma4:e4b"` without deduping, so a box with N legacy local models now has N identical rows (the dev box has 3). The model picker shows N identical entries.

**Fix:** a **new** migration `migrations/1750810000_dedup_local_models.go` (do NOT edit the shipped `1750800000`). For each `kind='local'` provider, scan its `llm_models` ordered by `created`; keep the first row per distinct `chat_model`, delete later duplicates. If a deleted row's id equals `llm_settings.active_model`, repoint `active_model` to the surviving row id (identical tag → no behavior change). Down migration is a no-op (can't restore deleted dupes; harmless). Fresh installs never create dupes, so this is a one-off cleanup for existing boxes.

## Fix 3 — pre-pull disk-space guard

**Problem:** `Pull` starts a multi-GB download with no free-space check; on a near-full disk it fails with a cryptic mid-pull error.

**Fix:** before launching the pull goroutine, `Manager.Pull` checks free space on the model-store volume and refuses with a clear error when it is below a conservative threshold. Honest limitation: we cannot cheaply know the target model's size pre-pull, so this is a "fail fast on a near-full disk" guard, not an exact check.
- `const defaultMinFreeGB = 12` (covers `gemma4:e4b` ~9.6 GB + headroom; the 26B GPU model is opt-in and a GPU box would be sized for it). Overridable via `BALAUR_OLLAMA_MIN_FREE_GB` (parse int; invalid/empty → default).
- New helper `freeBytes(path string) (uint64, error)`: `internal/ollama/diskspace_unix.go` (build tag `unix`, `syscall.Statfs` → `Bavail * Bsize`) and `internal/ollama/diskspace_windows.go` (build tag `windows`, returns `^uint64(0)` so the check always passes — Windows is not a deploy target).
- The store path checked: the Ollama model store. Default is `~/.ollama`; honor `OLLAMA_MODELS` if set, else use the user home `.ollama`. If the path doesn't exist yet, check its nearest existing parent (or the home dir).
- On insufficient space, `Pull` returns an error like `"insufficient disk space: %d GB free, need ≥ %d GB (set BALAUR_OLLAMA_MIN_FREE_GB to override)"` **before** starting the goroutine, so the web handler surfaces it in the models panel.

**Tests (`manager_test.go`):** unit-test the threshold parsing + the guard decision via a small seam — `Pull` calls an internal `checkDiskSpace(minGB int, free uint64) error` (pure function) so tests assert: free below threshold → error; free above → nil; `BALAUR_OLLAMA_MIN_FREE_GB` override parsed. (The OS `freeBytes` call itself is not unit-tested — it's a thin syscall wrapper.)

## Fix 4 — short-TTL cache for `IsPulled`/`List`

**Problem:** `IsPulled` is on the board-render hot path (`turn.availableChoices` per page load) and hits `/api/tags` every time; a slow daemon adds latency per render.

**Fix:** add a small tags cache to `Manager`:
- Fields (under `mu`): `tagsCache []Model`, `tagsCacheAt time.Time`. `const tagsTTL = 3 * time.Second`.
- `cachedTags() ([]Model, error)`: read `tagsCache`/`tagsCacheAt` under `mu`; if `time.Since(tagsCacheAt) < tagsTTL` return a copy; otherwise fetch via `api.tags` **with `mu` released** (do not hold the lock across network I/O — same lesson as the spawn lock-hold fix), then store under `mu` and return. A concurrent double-fetch is acceptable (last write wins).
- `IsPulled` and `List` call `cachedTags()` instead of `api.tags` directly.
- **Invalidation:** zero `tagsCacheAt` (under `mu`) on a successful `Pull` (in `runPull` after success, before/with the `onDone` call) and at the end of `Delete`, so a freshly pulled/removed model reflects immediately rather than waiting out the TTL.

**Tests (`manager_test.go`):** with an httptest `/api/tags` that counts requests, assert two `IsPulled` calls within the TTL hit the server once; after invalidation (simulate via a successful pull against the fake server, or a `Delete`) the next call refetches. Keep timing-robust (don't sleep for the full TTL; test the cache-hit and the invalidate paths, not TTL expiry wall-clock).

## Files

- `internal/ollama/binary.go`, `internal/ollama/binary_test.go`
- `internal/ollama/manager.go`, `internal/ollama/manager_test.go`
- `internal/ollama/diskspace_unix.go`, `internal/ollama/diskspace_windows.go`
- `migrations/1750810000_dedup_local_models.go`

## Verification beyond unit tests

After implementation, re-run the **real auto-install path** against the actual v0.30.8 release into a throwaway temp dir (not the live `pb_data`): `installBinary(ctx, tmp)` (or the equivalent extract), then confirm `tmp/bin/ollama` + `tmp/lib/ollama/` landed and `tmp/bin/ollama --version` runs. This proves Fix 1 against the genuine archive, not just synthetic test tars.

## Non-goals

- No manifest-based exact pre-pull size lookup (Fix 3 stays a free-space floor).
- No change to the already-shipped `1750800000` migration.
- No GPU auto-detection.
