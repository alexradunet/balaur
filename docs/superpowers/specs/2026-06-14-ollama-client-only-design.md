# Ollama: client-only of an existing server

**Date:** 2026-06-14
**Status:** Approved, ready for implementation plan

## Goal

Stop Balaur from supervising Ollama. Today Balaur downloads a pinned Ollama
binary, spawns `ollama serve`, health-probes it, pulls default models on first
run, and kills the server on shutdown. After this change Balaur is a pure
**client** of an Ollama server the owner already runs. It connects, lists and
manages models, and runs inference — it never installs, spawns, or stops the
server.

This makes the deployment model explicit and simple: "run Ollama, then run
Balaur." It removes ~600 lines of lifecycle code (binary download, archive
extraction, process management, disk preflighting) and the
`github.com/klauspost/compress` dependency, in exchange for one new dependency,
the official Ollama Go client.

## Decisions (locked)

1. **Control plane uses the official `github.com/ollama/ollama/api`.** It is
   CGO-free (stdlib + four small internal ollama packages: `auth`, `envconfig`,
   `format`, `version`), mirrors the REST API the Ollama CLI itself uses, and
   covers exactly what Balaur needs: `Heartbeat` (readiness), `List`, `Pull`
   (streaming progress), `Delete`. The hand-rolled `internal/ollama/api.go` is
   deleted.

2. **Inference stays on the OpenAI-compatible `/v1` client.**
   `internal/ollama/client.go` (`llm.OpenAIClient` against
   `http://<host>/v1`) is unchanged. Ollama's `/v1` is its stable
   OpenAI-compatible surface, and every remote provider already shares that one
   `llm.OpenAIClient`. We do **not** route local chat through `ollama/api` —
   that would fork the inference path for no user benefit. One inference path;
   one typed control client.

3. **No LLM router, no agent framework.** Routers (Bifrost, LiteLLM, etc.) are
   separate proxy processes that break "one binary, local-first" and duplicate
   the minimal routing `internal/llm` already provides. Agent frameworks
   (LangChainGo ~170 deps, Genkit ~129, Eino ~37) would replace the small,
   auditable core loop with an opaque, heavy dependency tree — the opposite of
   Balaur's transparency and minimal-trust-surface goals. Rejected.

4. **No startup auto-pull.** `main.ensureLocalDefault` (install → spawn → pull
   defaults → activate) is deleted entirely. A fresh box has no active LLM
   model until the owner pulls and activates one via the `/models` UI. The
   default Gemma model is still **pre-listed** there (via
   `store.EnsureDefaultLLMConfig`), so the owner clicks pull→activate; that code
   is unchanged.

5. **No local disk-space preflight.** Balaur may talk to a remote server, so a
   local `~/.ollama` free-space check is wrong. The server reports its own pull
   errors over the stream, which Balaur surfaces. `diskspace_unix.go`,
   `diskspace_windows.go`, and the disk helpers are deleted.

6. **Host config is unchanged.** `BALAUR_OLLAMA_HOST` (default
   `127.0.0.1:11434`) already supports local and remote servers. The new
   `*api.Client` is built from `ollama.Host()`.

## Scope of change

### Files deleted

- `internal/ollama/binary.go` — download + archive extraction + `BinaryPath`,
  `EnsureInstalled`, `installBinary`, `extractArchive`.
- `internal/ollama/binary_test.go`.
- `internal/ollama/api.go` — hand-rolled HTTP client (replaced by `ollama/api`).
- `internal/ollama/api_test.go` — replaced by tests against the typed client.
- `internal/ollama/diskspace_unix.go`, `internal/ollama/diskspace_windows.go`.
- `internal/ollama/process_unix.go`, `internal/ollama/process_windows.go` —
  process-group helpers, used only by `spawn`/`Stop`.

### `internal/ollama/manager.go` — trimmed to a control client

**Remove:** `EnsureRunning`, `EnsureInstalled`, `spawn`, `Stop`, `maxLoad`, the
disk helpers (`minFreeGB`, `modelStorePath`, `checkDiskSpace`,
`defaultMinFreeGB`), and the lifecycle struct fields (`dataDir`, `cmd`,
`spawned`, `tail`, and the `ringBuffer` type if it has no other user).

**Keep (behavior unchanged from the caller's view):** `Pull` (minus its
disk-check preamble), `runPull`, `Cancel`, `Snapshot`, `cachedTags` /
`invalidateTags`, `List`, `Delete`, `IsPulled`.

**Rework internals to call the typed client:**
- The manager holds an `*api.Client` built once from `ollama.Host()` (e.g. a
  lazily-initialised field or constructed in `apiClient()`).
- `List` → `client.List(ctx)`, mapping `api.ListResponse` entries to the
  existing `Model{Name, Size}` (keep `Path` empty as today, templates bind
  unchanged).
- `Delete` → `client.Delete(ctx, &api.DeleteRequest{...})`.
- `Pull` → `client.Pull(ctx, &api.PullRequest{...}, progressFn)`, mapping
  `api.ProgressResponse` (`Status`, `Completed`, `Total`) onto the existing
  `PullSnapshot` fields so `/models` and the chatbar loading bar bind unchanged.
- `IsPulled` keeps its current contract (lists tags, checks membership).

**Add:** `Reachable(ctx) bool` → `client.Heartbeat(ctx) == nil`. Used only by
the new startup log line. This keeps one honest readiness seam on the manager.

### `main.go`

- Delete `ensureLocalDefault`, `waitPull`, and `activateLocal`. (`waitPull` and
  `activateLocal` are only used by `ensureLocalDefault`; the `/models` pull
  handler already performs `SaveLocalModel` + `SetActiveLLMModel` itself, so
  deleting them orphans nothing.)
- Replace the `ensureLocalDefault(se.App)` call in the serve hook with a
  one-shot, non-fatal reachability log: on `ollama.Default.Reachable(ctx)` log
  info `"ollama: ready"` with the host; otherwise log warn
  `"ollama: not reachable — start Ollama or set BALAUR_OLLAMA_HOST"` with the
  host. Never blocks startup, never spawns.
- In `OnTerminate`, remove `ollama.Default.Stop()` and its comment. Keep the
  search-index close in that block.
- Prune now-unused imports (`context`, `time` if no longer referenced).

### `go.mod`

- Add `github.com/ollama/ollama` (for `.../api`).
- `github.com/klauspost/compress` leaves the direct require set after
  `binary.go` is deleted — `go mod tidy` drops it or demotes it to `// indirect`
  if a transitive user remains. Run `go mod tidy` and commit the result.

### Docs / self-knowledge

- `internal/ollama/presets.go` package doc: drop "process lifecycle, binary
  install"; describe the package as model control (list/pull/delete + readiness)
  over the official client, never inference or lifecycle.
- `internal/self/knowledge.md`: update any text describing Balaur as
  auto-installing / supervising Ollama. Balaur is now a client of an existing
  Ollama server. Same commit as the code change (a stale self-description makes
  Balaur lie about itself).
- If any user-facing copy or `docs/` references the auto-install/auto-spawn
  behavior, update it.

## Behavior after the change

- **Server running, models present:** identical to today — chat, embeddings,
  `/models` list/pull/delete all work.
- **Server running, no models:** owner sees the pre-listed default in `/models`,
  clicks to pull and activate (existing flow).
- **Server not running:** startup logs a clear warning naming the host;
  Balaur still boots. `/models` shows its existing "not reachable" / pull-hint
  state (`List` returns an error, already handled). Inference attempts fail
  through the normal `llm` error path with a clear message. No crash, no spawn.
- **Remote server (`BALAUR_OLLAMA_HOST=host:port`):** works; no local disk
  check to give a false negative.

## Testing & validation

- Rework `manager_test.go`: drop spawn/adopt/`Stop`/install/disk tests. Keep
  pull/list/delete/snapshot/`IsPulled` tests; they run against an `httptest`
  server, and the official `api.Client` accepts an injected `*url.URL` +
  `*http.Client`, so point it at the test server's URL.
- Replace `api_test.go` coverage with tests that exercise the manager's mapping
  of `api.ProgressResponse` → `PullSnapshot` and `api.ListResponse` → `Model`.
- Remove `binary_test.go` and any disk-space tests with their deleted code.
- Review `e2e_test.go` for references to install/spawn; update or remove.
- Gates before "done": `go vet ./...`, `go test ./...`,
  `CGO_ENABLED=0 go build ./...`, `git diff --check`.

## Out of scope / deferred

- Replacing inference with `ollama/api`'s `Chat`/`Generate` (we keep `/v1`).
- Adopting any structured-output or agent helper library — revisit only if one
  narrow piece of the loop becomes genuinely painful (YAGNI).
- Surfacing server reachability in the UI beyond the existing `/models` state
  and the new startup log (could add a status pill later; not required now).
