# Plan 195: Passphrase-encrypted off-box backup of the Markdown mirror

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat e06346d..HEAD -- internal/export/ internal/cli/export.go internal/cli/export_test.go internal/self/knowledge.md go.mod`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: LOW
- **Depends on**: plans/194-*.md (the Phase-2 Markdown mirror)
- **Category**: direction
- **Planned at**: commit `e06346d`, 2026-06-25

## Why this matters

Balaur's **Sovereignty** pillar promises the life lives "on the owner's box,
in inspectable SQLite **and exported Markdown** — never in a vendor's
database." The plaintext Markdown mirror (Phase 2, plan 194) makes the life
inspectable, but a mirror you can copy to a USB stick or a friend's NAS is only
safe to carry off-box if it is encrypted. This plan adds the opt-in,
owner-initiated **encrypted backup**: one passphrase-encrypted archive over the
mirror tree, built entirely from `golang.org/x/crypto` (already an indirect
dependency) + the Go standard library, so the binary stays CGO-free and gains
no new direct module. The plaintext mirror stays the default; encryption is the
carry layer on top. The hard, non-negotiable property: **a lost passphrase
means a lost backup** — no escrow, no cloud, no key-on-disk — and that has to
be said bluntly at the moment of creation.

## Current state

The repo today ships only **Phase 1** (plan 192, the read-only spike). **Plan
194 (the Phase-2 mirror) is a dependency that is not yet present in this tree at
the planned-at commit** — there is no full mirror exporter, only the one-type
`ExportType`. See the **"DEPENDENCY: plan 194" note** below for how this plan is
written to compose with whatever 194 produces; the encryption layer in this plan
is deliberately tree-shaped (it encrypts *a directory*, not the DB), so it does
not care which exporter filled that directory.

### Files in scope and their current role

- `internal/export/export.go` — the Phase-1 exporter. Package doc + the
  redaction boundary. You will ADD a sibling `encrypt.go`; do NOT modify
  `export.go`. Verbatim header:

```go
// internal/export/export.go:1
// Package export is Balaur's sovereign-export spike (plan 192): a one-way,
// read-only renderer of the active knowledge record to Markdown. It is the thin
// slice of the Johnny Decimal vault mirror — ONE node type, no git, no
// encryption. The redaction boundary is hard: it reads ONLY the `nodes`
// collection, ONLY status=active rows, and never touches any secret/token
// collection (api_key, OAuth tokens, vault entries — see the design note at
// docs/superpowers/specs/2026-06-25-sovereign-export-design.md).
package export
```

```go
// internal/export/export.go:29 — the only exported function today
func ExportType(app core.App, typ, destDir string) ([]string, error) {
```

- `internal/export/export_test.go` — the redaction canary tests
  (`TestExportHappyPath`, `TestExportExcludesNonActive`,
  `TestExportNeverLeaksStoredSecret`). It shows the test idioms you must reuse:
  `storetest.NewApp(t)`, `t.TempDir()`, `nodes.Create(...)`,
  `store.SaveCloudModel(...)`, and a `readAll` walker. Verbatim
  signatures the tests call (confirmed in source):

```go
// internal/nodes/nodes.go:89
func Create(app core.App, typ, title, body, status string, props map[string]any) (*core.Record, error)

// internal/store/llm_settings.go:158
func SaveCloudModel(app core.App, name, baseURL, apiKey, label, chatModel, embedModel string) (string, error)
```

  The seeded-secret pattern (reuse it for the no-plaintext-in-ciphertext test):

```go
// internal/export/export_test.go:109-116
func TestExportNeverLeaksStoredSecret(t *testing.T) {
	app := storetest.NewApp(t)
	const secret = "sk-SECRET-TOKEN-DO-NOT-LEAK"
	if _, err := store.SaveCloudModel(app, "TestProvider", "https://example.test",
		secret, "Test", "test-model", ""); err != nil {
		t.Fatalf("seed cloud model: %v", err)
	}
```

- `internal/cli/export.go` — the `export` CLI subcommand. You will ADD an
  `--encrypt` flag + a `--passphrase` flag (or env fallback) to this command.
  Verbatim current body:

```go
// internal/cli/export.go:16-34
func exportCmd(app core.App) *cobra.Command {
	var typ, out string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Spike: read-only one-type Markdown export of active nodes (no git, no encryption)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&typ, "type", "note", "node type to export (spike: note only)")
	cmd.Flags().StringVar(&out, "out", "", "destination directory (required; never the data dir)")
	_ = cmd.MarkFlagRequired("out")
	cmd.RunE = run(app, "export", func(cmd *cobra.Command, args []string) (any, error) {
		paths, err := export.ExportType(app, typ, out)
		if err != nil {
			return nil, fmt.Errorf("export: %w", err)
		}
		return map[string]any{"type": typ, "out": out, "files": paths}, nil
	})
	return cmd
}
```

- `internal/cli/export_test.go` — the CLI test pattern. `executeEnvelope`
  returns the parsed v1 envelope (`{"v":1,"kind":...,"data":...}`) and asserts
  structural validity; on failure it returns the error envelope from stderr.
  Reuse `executeEnvelope(t, exportCmd(app), ...)`.

- `internal/cli/cli.go` — the `run(app, kind, body)` wrapper every command body
  uses. Verbatim contract (errors become a stderr v1 error envelope + non-zero
  exit; the body returns `(any, error)`):

```go
// internal/cli/cli.go:84-101
func run(app core.App, kind string, body func(cmd *cobra.Command, args []string) (any, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		defer func() {
			if r := recover(); r != nil {
				err = failJSON(cmd, fmt.Errorf("panic: %v", r))
			}
		}()
		if mErr := app.RunAllMigrations(); mErr != nil {
			return failJSON(cmd, fmt.Errorf("applying migrations: %w", mErr))
		}
		out, bErr := body(cmd, args)
		if bErr != nil {
			return failJSON(cmd, bErr)
		}
		return emit(cmd.OutOrStdout(), kind, out)
	}
}
```

- `internal/self/knowledge.md` — the running binary's self-description. The
  `balaur export` paragraph currently says "no git, no encryption yet":

```
// internal/self/knowledge.md:321-326
owner's engine room, never your surface. `balaur export` is a sovereign-export
SPIKE stub (plan 192): a read-only, one-type (`note`) Markdown render of active
nodes to a caller-supplied dir — no git, no encryption yet; the redaction
boundary (active `nodes` rows only, never any secret/token collection) and the
phased mirror→encryption plan live in
docs/superpowers/specs/2026-06-25-sovereign-export-design.md.
```

- `go.mod` — `golang.org/x/crypto v0.53.0` is present **as an indirect
  require** (line in the second `require` block, suffixed `// indirect`):

```
// go.mod (indirect require block)
golang.org/x/crypto v0.53.0 // indirect
```

  `go mod why golang.org/x/crypto` prints "(main module does not need package
  golang.org/x/crypto)" — i.e. nothing in our code imports it *yet*. The moment
  `encrypt.go` imports `golang.org/x/crypto/scrypt`, `go mod tidy` will move
  this line out of `// indirect` into the direct `require` block. **That
  promotion is expected and allowed** (the brief: "a justified x/crypto
  promotion only"). It is NOT a new module — `v0.53.0` is already pinned in
  `go.sum`. Do not add `filippo.io/age` or any other new module.

### Design constraints inlined from the design note

From `docs/superpowers/specs/2026-06-25-sovereign-export-design.md`:

- **Open question 4 (key handling), the chosen answer (lines 191-194):** "a
  single **owner-supplied passphrase, KDF-stretched**, with a loud **'if you
  lose this passphrase, the backup is unrecoverable'** warning at the point of
  creation. No key escrow, no cloud, no generated-key-on-disk. This keeps the
  secret in the owner's head, where sovereignty wants it."
- **Open question 5 (composition), the chosen answer (lines 204-206):** "**(B)
  encrypts the (A) Markdown mirror** — one `age`/archive over the mirror tree.
  One artifact; the plaintext Markdown mirror stays the inspectable, sovereign
  default; encryption is the opt-in off-box carry layered on top, not a parallel
  export of secret state." → **Encrypt the mirror DIRECTORY (tar then encrypt).
  Do NOT re-export or encrypt the SQLite DB.**
- **Phase 3 scope (lines 222-225):** "a pure-Go `age` (or stdlib) envelope over
  the Phase-2 mirror tree (per question 5), an owner-passphrase KDF (per
  question 4), and the 'lost passphrase = lost backup' warning UX surfaced at
  creation time. Must stay CGO-free (`CGO_ENABLED=0`) — no CGO crypto binding."
- **Standing constraint (lines 228-234):** the mirror is "strictly one-way,
  additive, owner-initiated, and offline." Encryption does not read any new
  collection — it operates on an already-written directory — so it adds no new
  leak surface. Keep it that way: `encrypt.go` takes a *directory path*, not an
  `app`/`core.App`.

### DEPENDENCY: plan 194 (the mirror) — how this plan composes with it

Plan 194 produces a mirror **directory tree** (the JD folder layout) from a
full-mirror exporter. This plan's encryption is written against **a directory**,
not against 194's API, so the two compose cleanly:

- `encrypt.go` exposes `EncryptDir(srcDir, destFile, passphrase string) error`
  and `DecryptDir(srcFile, destDir, passphrase string) (err error)` — pure,
  app-free, tree-shaped.
- The CLI `--encrypt` path FIRST produces a plaintext mirror into a temp/`--out`
  dir (today via `export.ExportType`; after 194 via 194's full-mirror entry
  point), THEN calls `EncryptDir` over that dir to produce the archive.
- **If plan 194 has landed** when you execute this plan, wire the CLI's plaintext
  step to 194's full-mirror function (look for the exported mirror entry point in
  `internal/export`, e.g. `ExportMirror`/`Mirror`); the `EncryptDir` half is
  unchanged.
- **If plan 194 has NOT landed** (the planned-at state), wire the CLI's plaintext
  step to the existing `export.ExportType(app, typ, dir)` so `--encrypt` is
  end-to-end testable now. Either way `EncryptDir`/`DecryptDir` are identical and
  the round-trip + no-plaintext tests pass. Note this choice in your handoff so a
  reviewer can re-point it at 194's entry point later.

This means **you can implement and fully test this plan even if 194 is absent** —
the encryption is the load-bearing new code and it is decoupled from the mirror
renderer by design.

### Repo conventions that apply here

- `gofmt` is law; `go vet ./...` and `staticcheck`/`govulncheck` gate CI.
- Tests: standard `testing`, table-driven where it helps, **no assertion
  frameworks**, **no `time.Sleep`**, `t.TempDir()` for I/O, `storetest.NewApp(t)`
  for a PB app. Model after `internal/export/export_test.go`.
- Errors wrap: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code.
- `crypto/rand.Read` for salt + nonce — never `math/rand`.
- New deps must justify against suckless: PREFER stdlib + already-present
  indirect deps. Here that means `crypto/aes`, `crypto/cipher`, `crypto/rand`,
  `archive/tar`, `encoding/binary`, `io`, `os`, `path/filepath` from stdlib +
  `golang.org/x/crypto/scrypt` (already pinned). No new module.
- Update `internal/self/knowledge.md` because capability changes.

## Commands you will need

| Purpose            | Command                                                                 | Expected on success            |
|--------------------|-------------------------------------------------------------------------|--------------------------------|
| Set tmpdir (first) | `export TMPDIR=/home/alex/.cache/go-tmp`                                 | (no output)                    |
| Drift check        | `git diff --stat e06346d..HEAD -- internal/export/ internal/cli/export.go internal/cli/export_test.go internal/self/knowledge.md go.mod` | empty (no drift) |
| Confirm dep status | `go mod why golang.org/x/crypto`                                        | prints a "needs" path after Step 1, see notes |
| Build (CGO-free)   | `CGO_ENABLED=0 go build ./...`                                           | exit 0                         |
| Test export pkg    | `go test ./internal/export/`                                            | ok, all pass                   |
| Test cli pkg       | `go test ./internal/cli/`                                               | ok, all pass                   |
| Full suite         | `go test ./...`                                                         | ok, all pass                   |
| Vet                | `go vet ./...`                                                          | exit 0, no output              |
| Format check       | `gofmt -l .`                                                            | empty (no files listed)        |
| Diff whitespace    | `git diff --check`                                                      | empty                          |
| Tidy (after Step 1)| `go mod tidy`                                                           | exit 0; only x/crypto moves    |

All `go` commands require `export TMPDIR=/home/alex/.cache/go-tmp` first (the
3.9G tmpfs `/tmp` fills during linking and the build fails "No space left on
device").

## Suggested executor toolkit

- Invoke the `go-standards` skill before writing Go — it carries the repo's
  error-wrapping, structured-logging, testing, and suckless idioms.
- Reference: `docs/superpowers/specs/2026-06-25-sovereign-export-design.md`
  (open questions 4 and 5, Phase 3). Already excerpted above; read it if you
  need fuller context.

## Scope

**In scope** (the only files you may modify or create):

- `internal/export/encrypt.go` (create) — `EncryptDir` + `DecryptDir` + the
  envelope header + KDF.
- `internal/export/encrypt_test.go` (create) — round-trip, wrong-passphrase,
  no-plaintext-in-ciphertext tests.
- `internal/cli/export.go` (modify) — add the `--encrypt` / `--passphrase` path
  and the loud unrecoverable-passphrase warning.
- `internal/cli/export_test.go` (modify) — add a CLI-level `--encrypt` test.
- `internal/self/knowledge.md` (modify) — update the `balaur export` paragraph.
- `go.mod` / `go.sum` (modify) — ONLY via `go mod tidy` after Step 1, which
  promotes the already-pinned `golang.org/x/crypto` from indirect to direct. Do
  not hand-edit; do not add any other module.

**Out of scope** (do NOT touch, even though they look related):

- `internal/export/export.go` — the Phase-1 exporter and its redaction
  boundary. Encryption is a sibling file; do not refactor the exporter.
- The full mirror render / JD folder layout — that is plan 194. This plan
  encrypts whatever directory the mirror step produced; it does not produce it.
- A standalone `decrypt` CLI command. `DecryptDir` exists ONLY so the round-trip
  test can prove correctness. Do not add a `balaur export --decrypt` subcommand
  or any user-facing decrypt verb beyond what the test needs.
- Any non-passphrase key mode (generated key, escrow, cloud KMS).
- Any network / cloud / upload path.
- The web UI (`internal/web`, storybook) — CLI-only for this plan.
- `migrations/` — no schema change; this writes files, not records.

## Git workflow

- Land-on-main repo; executors run in a worktree off `origin/main`.
- Conventional-commit subject, e.g.
  `feat(export): passphrase-encrypted archive over the Markdown mirror (195 P3)`.
- Commit per logical unit is fine; one commit for the whole plan is also fine.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Write `internal/export/encrypt.go` (the crypto core)

Create `internal/export/encrypt.go` in `package export`. Implement a small,
self-contained AES-256-GCM-over-tar envelope with a scrypt-stretched passphrase.
Use ONLY: stdlib (`archive/tar`, `bytes`, `crypto/aes`, `crypto/cipher`,
`crypto/rand`, `encoding/binary`, `errors`, `fmt`, `io`, `os`, `path/filepath`,
`sort`) and `golang.org/x/crypto/scrypt`.

Implement exactly these exported symbols (signatures are load-bearing — the
tests and CLI call them):

```go
// EncryptDir tars srcDir (deterministic file order) and writes a single
// passphrase-encrypted archive to destFile. AES-256-GCM over the tar; the
// per-archive scrypt salt and GCM nonce live in a small plaintext header that
// is also bound as GCM additional-authenticated-data, so tampering with the
// header fails decryption. It never reads any PocketBase collection — it
// operates purely on the already-written directory.
func EncryptDir(srcDir, destFile, passphrase string) error

// DecryptDir reverses EncryptDir: it reads the header, re-derives the key from
// the passphrase, GCM-opens the ciphertext, and untars into destDir. A wrong
// passphrase (or any tampered byte) fails the GCM auth tag and returns
// ErrBadPassphrase with NOTHING written to destDir (decrypt fully in memory,
// untar only after Open succeeds). Used by the round-trip test; not a CLI verb.
func DecryptDir(srcFile, destDir, passphrase string) error

// ErrBadPassphrase is returned by DecryptDir when authentication fails (wrong
// passphrase or tampered archive). Callers test with errors.Is.
var ErrBadPassphrase = errors.New("export: wrong passphrase or corrupted archive")
```

Construction details (KISS — write the simplest correct version):

1. **Header / envelope format.** Write a fixed plaintext header at the start of
   `destFile`, then the GCM ciphertext. Header layout (use `encoding/binary`,
   big-endian):
   - magic: 8 bytes ASCII `BALAUREX` (envelope identifier).
   - version: 1 byte = `1`.
   - a 1-byte "WARNING flag" set to `1` meaning "no recovery without the
     passphrase" — this is the envelope's record of the unrecoverable property
     (the brief asks for "a flag in the envelope"). It carries no key material.
   - salt length (`uint8`) + salt bytes (use 16 bytes from `crypto/rand`).
   - nonce length (`uint8`) + nonce bytes (`gcm.NonceSize()`, from
     `crypto/rand`).
   Then the ciphertext (`gcm.Seal` output) to EOF.
   Serialize the header bytes into a `[]byte` first; pass that exact header slice
   as the GCM `additionalData` so any header tamper breaks the auth tag.
2. **KDF.** `scrypt.Key([]byte(passphrase), salt, 1<<15, 8, 1, 32)` → a 32-byte
   key (AES-256). `1<<15` (N=32768), r=8, p=1 are the standard interactive
   params; 32-byte output = AES-256. Reject an empty passphrase with a clear
   error before deriving (`if passphrase == "" { return fmt.Errorf("export:
   empty passphrase") }`).
3. **Cipher.** `aes.NewCipher(key)` → `cipher.NewGCM(block)`. Encrypt with
   `gcm.Seal(nil, nonce, tarBytes, header)`. Decrypt with `gcm.Open(nil, nonce,
   ciphertext, header)`; on error return `ErrBadPassphrase` (wrap nothing
   secret).
4. **Tar.** Walk `srcDir` with `filepath.WalkDir`, collect relative paths, SORT
   them so the tar (and thus, for a fixed salt+nonce, the plaintext) is
   deterministic. For each regular file write a `tar.Header` with the relative
   name (use forward slashes) and the file bytes. Skip directories (or write
   them as tar dir entries — either is fine; regular files alone reconstruct the
   tree on untar via `MkdirAll`). Tar into a `bytes.Buffer`; this is the GCM
   plaintext. (Mirror trees are small — buffering in memory is the simplest
   correct option and matches the spike's full-rewrite model.)
5. **Untar (DecryptDir).** Only after `gcm.Open` succeeds, read the tar from a
   `bytes.Reader`, and for each entry `filepath.Join(destDir, name)`,
   `os.MkdirAll(filepath.Dir(...))`, `os.WriteFile(...)`. Guard against path
   traversal: reject any entry whose cleaned path escapes `destDir` (`if
   !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator))
   { return fmt.Errorf("export: unsafe tar path %q", name) }`).
6. **No secrets in logs/errors.** Never include the passphrase or derived key in
   any error string.

After writing the file, run `go mod tidy` to promote `golang.org/x/crypto` to a
direct require.

**Verify**:
- `export TMPDIR=/home/alex/.cache/go-tmp`
- `CGO_ENABLED=0 go build ./internal/export/` → exit 0
- `go mod why golang.org/x/crypto` → now prints a real package path through
  `internal/export` (no longer "main module does not need"), confirming the
  promotion is justified by a real import.
- `git diff go.mod` → the ONLY change is `golang.org/x/crypto v0.53.0` moving
  from the `// indirect` block into the direct `require` block (no new module
  lines). If `go mod tidy` adds or removes any OTHER module, STOP.

### Step 2: Write `internal/export/encrypt_test.go` (round-trip + canaries)

Create `internal/export/encrypt_test.go` in `package export_test`. Model the
fixtures after `export_test.go`. Three tests:

1. **`TestEncryptDecryptRoundTrip`** — make a temp source dir with at least two
   files in nested subdirs (e.g. `a.md` and `sub/b.md`) with known bytes.
   `EncryptDir(src, archive, "correct horse")`, then `DecryptDir(archive, dst,
   "correct horse")`. Walk `dst` and assert each file's bytes are byte-identical
   to the source (use a small walk helper; do not import a diff library). Assert
   the archive file exists and is non-empty.

2. **`TestDecryptWrongPassphraseFails`** — encrypt with `"correct horse"`,
   decrypt the same archive with `"wrong horse"` into a fresh temp dir. Assert
   `errors.Is(err, export.ErrBadPassphrase)` (the package is `export_test`, so
   reference `export.ErrBadPassphrase`). Assert NO file was written to the
   destination dir (walk it; expect zero regular files) — proving no partial
   plaintext on failure. Assert the function did NOT panic (a normal returned
   error means no panic).

3. **`TestCiphertextHasNoPlaintextTitle`** — the leak canary. Create a source
   dir containing a Markdown file whose body holds a loud, unique marker string
   (e.g. `const marker = "SECRET-NODE-TITLE-DO-NOT-LEAK"`). `EncryptDir` it,
   read the whole archive file back as bytes, and assert
   `!bytes.Contains(archiveBytes, []byte(marker))` — the plaintext title/secret
   must NOT survive into the ciphertext. (This is the encryption analogue of the
   export package's redaction canary.)

**Verify**:
- `export TMPDIR=/home/alex/.cache/go-tmp`
- `go test ./internal/export/` → ok, all pass (3 existing + 3 new = 6 tests).

### Step 3: Add the `--encrypt` path + loud warning to `internal/cli/export.go`

Modify `exportCmd` in `internal/cli/export.go`. Add two flags and an encryption
branch. Keep the existing plaintext path unchanged when `--encrypt` is absent.

- Add `var encrypt bool` and register
  `cmd.Flags().BoolVar(&encrypt, "encrypt", false, "encrypt the export into a single passphrase-protected archive")`.
- Add `var archive string` and
  `cmd.Flags().StringVar(&archive, "archive", "", "output archive path (required with --encrypt)")`.
- The passphrase MUST NOT be a plaintext flag captured in shell history. Read it
  from the environment variable `BALAUR_EXPORT_PASSPHRASE`. In the command body,
  when `encrypt` is true:
  1. Require `archive != ""` and a non-empty `os.Getenv("BALAUR_EXPORT_PASSPHRASE")`;
     return a clear error from the body otherwise (this becomes a v1 error
     envelope via `run`'s `failJSON`).
  2. Print the LOUD warning to **stderr** (`cmd.ErrOrStderr()`), e.g.:
     `WARNING: if you lose this passphrase, this backup is UNRECOVERABLE — there is no recovery, no escrow, no cloud.`
     Use `fmt.Fprintln(cmd.ErrOrStderr(), ...)`. (stderr keeps stdout's JSON
     envelope clean — the CLI contract in `cli.go:1-16` requires stdout to be
     only the envelope.)
  3. Produce the plaintext mirror into a temp dir
     (`tmp, _ := os.MkdirTemp("", "balaur-export-*")`, `defer os.RemoveAll(tmp)`)
     via the mirror step. See the **"DEPENDENCY: plan 194" note** in Current
     state: if 194's full-mirror function exists, call it; otherwise call
     `export.ExportType(app, typ, tmp)`. The encrypted archive must NOT contain a
     leftover plaintext copy, so encrypt from the temp dir and let the `defer`
     wipe it — do NOT write plaintext to `--out` on the encrypt path.
  4. `if err := export.EncryptDir(tmp, archive, passphrase); err != nil { return nil, fmt.Errorf("export encrypt: %w", err) }`.
  5. Return `map[string]any{"type": typ, "archive": archive, "encrypted": true}`
     (do NOT echo the passphrase or file list of plaintext names — keep the
     envelope free of plaintext leakage).
- Update the command `Short` to drop "no encryption" now that it ships, e.g.
  `"read-only Markdown export of active nodes; --encrypt for a passphrase-protected archive"`.
- When `--encrypt` is false, the existing behavior is unchanged (still requires
  `--out`, still calls `ExportType`, still returns the files list).

Note on flag interplay: `--out` is currently marked required globally via
`MarkFlagRequired("out")`. On the `--encrypt` path the output is `--archive`,
not `--out`. To avoid forcing both, remove the unconditional
`MarkFlagRequired("out")` and instead validate inside the body: require `--out`
when NOT encrypting and `--archive` when encrypting. Keep the existing
`TestExportRequiresOut` passing by returning an error when `--out` is empty on
the non-encrypt path (it asserts `cmd.Execute()` returns a non-nil error;
`run`'s `failJSON` returns the error, so a body-level
`return nil, fmt.Errorf("export: --out is required")` satisfies it).

**Verify**:
- `export TMPDIR=/home/alex/.cache/go-tmp`
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go test ./internal/cli/` → existing export tests still pass.

### Step 4: Add a CLI-level `--encrypt` test to `internal/cli/export_test.go`

Add `TestExportEncryptWritesArchiveAndWarns`. Model after
`TestExportEmitsEnvelopeAndWritesFiles`:

- `app := storetest.NewApp(t)`; seed one active note via
  `nodes.Create(app, "note", "Exported Note", "Body with [[Link]].", nodes.StatusActive, nil)`.
- Set the passphrase env for the test: `t.Setenv("BALAUR_EXPORT_PASSPHRASE", "correct horse")`.
- `archive := filepath.Join(t.TempDir(), "backup.bin")`.
- Run `executeEnvelope(t, exportCmd(app), "--type", "note", "--encrypt", "--archive", archive)`.
- Assert `env["kind"] == "export"` and `data["encrypted"] == true`.
- Assert the archive file exists and is non-empty (`os.Stat`).
- Assert the archive bytes do NOT contain the plaintext H1 title:
  read `archive`, assert `!bytes.Contains(b, []byte("Exported Note"))`.
- (Optional but cheap) assert the warning reached stderr: `executeEnvelope`
  swallows stderr on success, so to check the warning, drive the command
  directly like `TestExportRequiresOut` does (set `cmd.SetErr(&errBuf)`, run,
  assert `strings.Contains(errBuf.String(), "UNRECOVERABLE")`). Pick ONE of the
  two run styles for this test; do not assert on both stdout-envelope and the
  raw stderr in the same `executeEnvelope` call.

Also add `TestExportEncryptRequiresPassphrase`: with `--encrypt --archive X` but
NO `BALAUR_EXPORT_PASSPHRASE` set, assert the command fails (error envelope /
non-zero) and the archive file does NOT exist.

**Verify**:
- `export TMPDIR=/home/alex/.cache/go-tmp`
- `go test ./internal/cli/` → ok, all pass (existing + 2 new).

### Step 5: Update `internal/self/knowledge.md`

Edit the `balaur export` paragraph (knowledge.md:321-326) so the self-description
stops claiming "no encryption yet." Replace the spike framing with the shipped
capability, keeping it lean and honest. Target wording (adapt to fit the
surrounding sentence flow):

> `balaur export` renders the active `nodes` knowledge record to a one-way
> Markdown mirror (the redaction boundary: active `nodes` rows only, never any
> secret/token collection). `--encrypt` wraps the whole mirror tree in a single
> passphrase-encrypted archive (scrypt-stretched passphrase, AES-256-GCM,
> CGO-free) for safe off-box backup — owner-supplied passphrase, no escrow, no
> cloud: lose the passphrase and the backup is unrecoverable. The phased design
> lives in docs/superpowers/specs/2026-06-25-sovereign-export-design.md.

Keep the line about the redaction boundary intact. Do not invent capabilities
beyond what Steps 1-4 actually shipped (e.g. if 194's full mirror is NOT wired,
do not claim "all owner-authored types" — describe what the code does).

**Verify**:
- `gofmt -l .` → empty (knowledge.md is not Go, so unaffected; this confirms no
  stray Go formatting issues).
- `git diff internal/self/knowledge.md` → only the export paragraph changed.

### Step 6: Full verification sweep

Run the complete gate before declaring done:

**Verify** (all must hold):
- `export TMPDIR=/home/alex/.cache/go-tmp`
- `CGO_ENABLED=0 go build ./...` → exit 0 (still CGO-free)
- `go test ./internal/export/ ./internal/cli/` → ok, all pass
- `go test ./...` → ok, all pass (no regression elsewhere)
- `go vet ./...` → exit 0, no output
- `gofmt -l .` → empty
- `git diff --check` → empty
- `git status` → only in-scope files changed (plus go.mod/go.sum from tidy)

## Test plan

New tests, by file:

- `internal/export/encrypt_test.go` (create):
  - `TestEncryptDecryptRoundTrip` — happy path: encrypt then decrypt with the
    correct passphrase round-trips byte-identically across nested dirs.
  - `TestDecryptWrongPassphraseFails` — wrong passphrase returns
    `export.ErrBadPassphrase`, writes nothing, does not panic (the
    no-partial-plaintext guarantee).
  - `TestCiphertextHasNoPlaintextTitle` — the leak canary: a seeded marker
    string is absent from the raw archive bytes.
- `internal/cli/export_test.go` (modify):
  - `TestExportEncryptWritesArchiveAndWarns` — `--encrypt` emits the
    `{"kind":"export","data":{"encrypted":true,...}}` envelope, writes a
    non-empty archive, leaks no plaintext title, and prints the UNRECOVERABLE
    warning to stderr.
  - `TestExportEncryptRequiresPassphrase` — `--encrypt` with no
    `BALAUR_EXPORT_PASSPHRASE` fails cleanly and writes no archive.

Structural pattern to copy: `internal/export/export_test.go` for the export
tests; `internal/cli/export_test.go`'s `TestExportEmitsEnvelopeAndWritesFiles`
and `TestExportRequiresOut` for the CLI tests.

Verification: `go test ./internal/export/ ./internal/cli/` → all pass, including
the 5 new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0 (binary stays CGO-free).
- [ ] `go test ./internal/export/ ./internal/cli/` exits 0; the 5 new tests
      exist and pass.
- [ ] `go test ./...` exits 0 (no cross-package regression).
- [ ] `go vet ./...` exits 0; `gofmt -l .` empty; `git diff --check` empty.
- [ ] `go mod why golang.org/x/crypto` prints a real package path through
      `internal/export` (the import that justifies the promotion).
- [ ] `git diff go.mod` shows ONLY `golang.org/x/crypto v0.53.0` moving from
      indirect to direct — NO other new module line. `filippo.io/age` appears
      nowhere.
- [ ] No files outside the in-scope list are modified (`git status`), aside from
      `go.mod`/`go.sum` from `go mod tidy`.
- [ ] `internal/self/knowledge.md` no longer says the export has "no encryption
      yet."
- [ ] `plans/README.md` status row for plan 195 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- The code at the locations in "Current state" does not match the excerpts (the
  codebase drifted since planned-at `e06346d`).
- A CGO-free passphrase-encryption path CANNOT be built from `golang.org/x/crypto`
  + stdlib without adding a new direct module. Do NOT add `filippo.io/age` or any
  other crypto module to satisfy the plan — STOP and report instead. (This should
  not happen: `crypto/aes`, `crypto/cipher`, `crypto/rand`, `archive/tar` are
  stdlib and `golang.org/x/crypto/scrypt` is already pinned at v0.53.0; Step 1's
  verify already proved these compile CGO-free.)
- `go mod tidy` wants to add or remove any module OTHER than promoting
  `golang.org/x/crypto` from indirect to direct.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (especially
  `internal/export/export.go`, the web UI, or `migrations/`).
- You discover the encryption would need to read a PocketBase collection — it
  must NOT; `EncryptDir`/`DecryptDir` take a directory path and never an
  `app`/`core.App`. If the design pulls you toward an `app` parameter, STOP.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **Composition with plan 194.** This plan's CLI encrypt path calls the mirror
  step to produce a plaintext tree, then `EncryptDir` over it. At planned-at the
  mirror step is `export.ExportType` (one type). When plan 194's full-mirror
  function lands, re-point the CLI's plaintext step at it; `EncryptDir`/
  `DecryptDir` are unchanged. A reviewer should confirm the encrypt path encrypts
  the SAME tree 194 renders, and that no plaintext copy is left in `--out` on the
  encrypt path (it goes to a temp dir wiped by `defer`).
- **What a reviewer should scrutinize:** (1) the GCM additional-authenticated-
  data binding of the header — tampering with salt/nonce/version must fail
  decryption; (2) `crypto/rand` (not `math/rand`) for salt and nonce; (3) the
  tar path-traversal guard in `DecryptDir`; (4) no passphrase or derived key in
  any error string or log; (5) `go.mod` shows only the x/crypto promotion.
- **Deferred out of this plan (and why):**
  - A standalone `balaur export --decrypt` / restore CLI — only `DecryptDir` (a
    test helper) ships now. A user-facing restore verb is a follow-up.
  - A web/UI surface for encrypted export — CLI-only for the Pareto slice.
  - Argon2id instead of scrypt — scrypt is sufficient and equally CGO-free; if a
    future audit prefers Argon2id, swap the KDF behind the same envelope (bump
    the envelope `version` byte and branch on it in `DecryptDir`).
  - Streaming/chunked encryption for very large mirrors — the in-memory tar is
    fine for the small mirror sizes the spike targets; revisit if a vault grows
    to gigabytes.
  - The README honesty ledger (`README.md:428` "Encrypted export" under
    "Roadmap — not shipped") — moving it out of the ledger is intentionally left
    for whoever lands the full mirror UX (194 + this), so the ledger flips once
    the owner-facing feature is complete, not on the CLI-only slice.
