# Plan 248: Prove the encrypted-backup envelope's two security properties with failure-path tests

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 077318a..HEAD -- internal/export/`
> If any file under `internal/export/` changed since this plan was written,
> compare the "Current state" excerpts against the live code before
> proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

`internal/export/encrypt.go` is Balaur's disaster-recovery format: the
passphrase-encrypted archive the owner carries off-box (`balaur export
--encrypt` / `balaur restore`). It makes two security promises in its own
doc comments: (1) the plaintext header (salt + nonce preamble) is bound as
GCM additional-authenticated-data, so "tampering with any header byte fails
the auth tag"; (2) `untar` rejects any tar entry whose cleaned target
escapes the destination directory ("unsafe tar path"). **Neither promise
has a test.** Existing coverage is round-trip, wrong-passphrase, and a
plaintext-leak canary only. Because `EncryptDir` can never emit a `../`
tar entry, the traversal branch is dead in every current test; and because
no test tampers with the archive, someone could drop the AAD argument from
`gcm.Seal`/`gcm.Open`, or replace the prefix check with a bare
`filepath.Join`, and the whole suite would stay green. This plan adds
tests-only failure-path coverage that turns both properties into
regressions that fail loudly.

## Current state

All excerpts below were read from the live files at commit `077318a`.

### Files

- `internal/export/encrypt.go` — the envelope: `EncryptDir`, `DecryptDir`
  (exported), `buildHeader`, `parseHeader`, `tarDir`, `untar` (unexported).
  **Do not modify this file.**
- `internal/export/encrypt_test.go` — existing tests, in the **external**
  test package `package export_test`: `TestEncryptDecryptRoundTrip`,
  `TestDecryptWrongPassphraseFails`, `TestCiphertextHasNoPlaintextTitle`,
  plus helpers `writeTree` and `countRegularFiles`. **Do not modify this
  file** (tour `.tours/15-sovereign-export.tour` anchors
  `internal/export/encrypt.go:57`; neither file may shift).
- `internal/export/encrypt_failure_test.go` — **new file you will create**,
  in the **internal** test package `package export`, so the unexported
  `untar` and `parseHeader` are callable. (Go allows `package export` and
  `package export_test` test files to coexist in one directory; helpers in
  `encrypt_test.go` are NOT visible from the new file — it is a separate
  package.)

### The AAD binding (the first untested property)

`internal/export/encrypt.go:89-90` (inside `EncryptDir`):

```go
	header := buildHeader(salt, nonce)
	ciphertext := gcm.Seal(nil, nonce, plaintext, header)
```

`internal/export/encrypt.go:135-140` (inside `DecryptDir`):

```go
	plaintext, err := gcm.Open(nil, nonce, ciphertext, header)
	if err != nil {
		// Never wrap the GCM error — a tampered tag and a wrong passphrase are
		// indistinguishable and neither leaks anything useful.
		return ErrBadPassphrase
	}
```

The sentinel, `internal/export/encrypt.go:47-49`:

```go
// ErrBadPassphrase is returned by DecryptDir when authentication fails (wrong
// passphrase or tampered archive). Callers test with errors.Is.
var ErrBadPassphrase = errors.New("export: wrong passphrase or corrupted archive")
```

`parseHeader` also returns `ErrBadPassphrase` for every malformed-preamble
case (`internal/export/encrypt.go:162-197`), so truncated/garbage inputs
surface the same sentinel.

### Header layout (needed to pick tamper offsets)

`buildHeader`, `internal/export/encrypt.go:150-160`:

```go
func buildHeader(salt, nonce []byte) []byte {
	var b bytes.Buffer
	b.WriteString(envMagic)
	b.WriteByte(envVersion)
	b.WriteByte(envWarning)
	b.WriteByte(byte(len(salt)))
	b.Write(salt)
	b.WriteByte(byte(len(nonce)))
	b.Write(nonce)
	return b.Bytes()
}
```

With `envMagic = "BALAUREX"` (8 bytes), `saltLen = 16`, and the standard
12-byte GCM nonce, the header byte offsets are: 0–7 magic, **8 version**,
**9 warning flag**, 10 salt-length, 11–26 salt, 27 nonce-length, 28–39
nonce; ciphertext starts at offset 40. Two consequences your tests exploit:

- **Offset 9 (the warning flag) is the perfect AAD probe.** `parseHeader`
  validates the magic and the version but reads the warning byte without
  checking its value (`internal/export/encrypt.go:175-177`). Flipping
  offset 9 leaves the salt and nonce byte-identical to encryption — the
  derived key and nonce are correct, so the ONLY reason `gcm.Open` can fail
  is the changed AAD. If someone dropped the AAD argument, this test (and
  only this test) would catch it: flipping a salt or nonce byte would still
  fail even without AAD binding.
- Do not hardcode 40: compute the header length in-test by calling
  `parseHeader(blob)` (callable from the internal test package) and taking
  `len(header)` from its first return value.

### The traversal guard (the second untested property)

`untar`, `internal/export/encrypt.go:266-272`:

```go
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		target := filepath.Join(cleanDest, filepath.FromSlash(hdr.Name))
		if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("export: unsafe tar path %q", hdr.Name)
		}
```

Note the `Typeflag != tar.TypeReg → continue` on lines 266-268: a
hand-built tar header MUST set `Typeflag: tar.TypeReg` explicitly, or the
malicious entry is silently skipped and the test proves nothing.

### Existing test shape (the structural pattern to match)

`internal/export/encrypt_test.go:92-109` (`TestDecryptWrongPassphraseFails`)
is the model: encrypt a small fixture tree, attempt `DecryptDir`, assert
`errors.Is(err, export.ErrBadPassphrase)`, and assert zero regular files
were written to the destination. Your new tests follow the same shape but
live in `package export`, so they call `EncryptDir`/`DecryptDir`/
`ErrBadPassphrase` unqualified (no `export.` prefix, no import of
`github.com/alexradunet/balaur/internal/export`).

### Conventions that apply

- Tests use the standard `testing` package, table-driven where it helps
  readability; no assertion frameworks; no `time.Sleep`-based
  synchronization. Use `t.TempDir()` for all filesystem fixtures.
- Errors are checked with `errors.Is` against the sentinel, never string
  comparison — except the traversal error, which is a plain `fmt.Errorf`
  with no sentinel; there, match the substring `unsafe tar path`.
- gofmt is law (`gofmt -l .` must print nothing); `staticcheck` and
  `go vet` must stay clean.
- `internal/self/knowledge.md` update: **NOT needed** — this change adds
  tests only; it does not alter the binary's architecture or capabilities.
- `.tours/` update: **NOT needed** — tour 15 anchors
  `internal/export/encrypt.go:57`, and this plan modifies neither
  `encrypt.go` nor `encrypt_test.go`, so no anchored line shifts.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted package tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1` | exit 0, all pass |
| One test by name | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -run TestDecryptTamperedArchiveFails -count=1 -v` | `--- PASS` lines, exit 0 |
| Vet | `go vet ./...` | exit 0, no output |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` for test runs. Use `-count=1`; cached runs are
not the gate.)

## Scope

**In scope** (the only file you may create or modify):

- `internal/export/encrypt_failure_test.go` (create; `package export`)

**Out of scope** (do NOT touch, even though they look related):

- `internal/export/encrypt.go` — production code. Zero diff here is a done
  criterion. If a test seems to require a production change, that is a STOP
  condition, not a refactor opportunity.
- `internal/export/encrypt_test.go` — the existing external-package tests,
  including the plaintext-leak canary. Do not move, rename, or extend them.
- `internal/export/export.go`, `internal/export/export_test.go`,
  `internal/export/mirror_test.go`, and every other file in the repo.
- `.tours/*` — no anchored line moves.

## Git workflow

- Run in an isolated git worktree branched from `origin/main`; branch name
  `advisor/248-encrypt-failure-path-tests`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`); this plan is one logical unit — a single commit like
  `test(export): failure-path tests for envelope AAD binding and untar traversal guard`.
- Stage with explicit pathspecs only
  (`git add internal/export/encrypt_failure_test.go`) — the main checkout
  is shared by parallel agents; never `git add -A` or `git add .`.
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Create the test file scaffold and the shared fixture helper

Create `internal/export/encrypt_failure_test.go` with `package export`
(internal test package — this is what makes `untar` and `parseHeader`
callable). Imports you will need: `archive/tar`, `bytes`, `errors`, `os`,
`path/filepath`, `strings`, `testing`.

Add one file-local helper that produces a small encrypted archive and
returns its raw bytes plus the parsed header length:

```go
// encryptFixture encrypts a one-file tree with pass and returns the raw
// archive bytes and the header length (computed via parseHeader so the
// tests never hardcode the envelope layout).
func encryptFixture(t *testing.T, pass string) (blob []byte, headerLen int) {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.md"), []byte("# body\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	archive := filepath.Join(t.TempDir(), "backup.bin")
	if err := EncryptDir(src, archive, pass); err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	blob, err := os.ReadFile(archive)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	header, _, _, _, err := parseHeader(blob)
	if err != nil {
		t.Fatalf("parse header of fresh archive: %v", err)
	}
	return blob, len(header)
}
```

Also add a file-local `regularFileCount(t *testing.T, dir string) int`
helper (walk `dir`, count `d.Type().IsRegular()` entries — same logic as
`countRegularFiles` in `encrypt_test.go:30-46`, which is invisible from
this package and must be duplicated; keep the different name so the two
files never collide if one is ever folded into the other).

**Verify**: deferred to Step 4 (single compile unit). `go build` ignores
`_test.go` files entirely, so no build command can check this file in
isolation — write the whole file across Steps 1–4, then run the Step 4
`go test` verification, which compiles and runs everything (unused imports
fail that compile). For an earlier type-check signal while drafting, use
`go vet ./internal/export/`, which does analyze test files.

### Step 2: Tamper tests — header byte (AAD) and ciphertext byte (tag)

Add `TestDecryptTamperedArchiveFails`, table-driven over the byte offset to
flip. Encrypt with `encryptFixture(t, "correct horse")`, flip exactly one
byte (`mut[tc.offset] ^= 0xFF` on a copy of `blob`), write the mutated
archive to a temp file, `DecryptDir` it **with the correct passphrase**,
and assert failure. Table rows (compute offsets from `headerLen`):

| case name | offset | what it proves |
|-----------|--------|----------------|
| `header warning byte (AAD only)` | `9` | key and nonce are unchanged (offset 9 is the warning flag, not salt/nonce, and `parseHeader` does not validate its value) — the failure is attributable purely to the header being bound as AAD. This is the row that catches a dropped AAD argument. |
| `first ciphertext byte` | `headerLen` | ciphertext body is tag-covered |
| `last byte (auth tag)` | `len(blob) - 1` | the tag itself is checked |

For each row assert, in order:

1. `errors.Is(err, ErrBadPassphrase)` is true (`t.Fatalf` with the got
   error otherwise) — per `encrypt.go:135-140` the GCM failure maps to the
   sentinel, unwrapped.
2. `regularFileCount(t, dst) == 0` where `dst` is a fresh `t.TempDir()` —
   nothing may be written on auth failure (`DecryptDir` doc,
   `encrypt.go:101-105`: "NOTHING written to destDir").

Guard the fixture itself: before the table loop, `DecryptDir` the
**unmutated** blob once with the correct passphrase into a scratch dir and
require `err == nil` — this proves the offsets you flip are the only cause
of failure.

**Verify**: deferred to Step 4 (single compile unit).

### Step 3: Traversal test — hand-built tar through `untar`

Add `TestUntarRejectsPathTraversal`, table-driven over malicious entry
names `"../evil.md"` and `"nested/../../evil.md"`. For each:

1. Build a tar in memory with `archive/tar`:

```go
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	data := []byte("evil")
	hdr := &tar.Header{
		Name:     tc.name,
		Mode:     0o644,
		Size:     int64(len(data)),
		Typeflag: tar.TypeReg, // REQUIRED: untar skips non-TypeReg entries (encrypt.go:266-268)
	}
	if err := tw.WriteHeader(hdr); err != nil { t.Fatalf(...) }
	if _, err := tw.Write(data); err != nil { t.Fatalf(...) }
	if err := tw.Close(); err != nil { t.Fatalf(...) }
```

2. Choose the destination as a **subdirectory** of a temp dir, so the
   escape target is observable inside the sandbox:
   `parent := t.TempDir(); dest := filepath.Join(parent, "restore")`.
3. Call `err := untar(buf.Bytes(), dest)` directly (unexported — this is
   why the file is `package export`).
4. Assert `err != nil` and
   `strings.Contains(err.Error(), "unsafe tar path")` (the exact text from
   `encrypt.go:271`: `fmt.Errorf("export: unsafe tar path %q", hdr.Name)`).
5. Assert the escape did not happen:
   `os.Stat(filepath.Join(parent, "evil.md"))` returns an error satisfying
   `os.IsNotExist` — nothing was written outside `dest`. Also assert
   `regularFileCount(t, dest) == 0` (guard fires before any write of the
   entry; the only tar entry is the malicious one).

**Verify**: deferred to Step 4.

### Step 4: Truncation test — malformed inputs error, never panic

Add `TestDecryptTruncatedArchive`, table-driven with `t.Run` subtests.
Build one good archive via `encryptFixture(t, "correct horse")`, then for
each case write the mangled bytes to a temp file and `DecryptDir` with the
correct passphrase:

| case name | bytes | expected |
|-----------|-------|----------|
| `ten byte garbage file` | `[]byte("0123456789")` (must not start with `BALAUREX`) | `errors.Is(err, ErrBadPassphrase)` — magic mismatch in `parseHeader` (`encrypt.go:167-170`) |
| `cut mid-header` | `blob[:20]` (inside the 16-byte salt region, offsets 11–26) | `errors.Is(err, ErrBadPassphrase)` — `io.ReadFull` on the salt fails (`encrypt.go:181-184`) |
| `cut mid-ciphertext` | `blob[:len(blob)-8]` (header intact, tag destroyed) | `errors.Is(err, ErrBadPassphrase)` — GCM `Open` fails (`encrypt.go:135-140`) |

Each subtest also asserts `regularFileCount(t, dst) == 0` for its fresh
dest dir. A returned error (as opposed to a crash) is itself the no-panic
proof — no `recover` scaffolding needed.

**Verify** (the whole file, Steps 1–4):
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1 -v` →
exit 0; output contains `--- PASS: TestDecryptTamperedArchiveFails`,
`--- PASS: TestUntarRejectsPathTraversal`,
`--- PASS: TestDecryptTruncatedArchive`, and the three pre-existing tests
(`TestEncryptDecryptRoundTrip`, `TestDecryptWrongPassphraseFails`,
`TestCiphertextHasNoPlaintextTitle`) all still `PASS`.

### Step 5: Full gate and hygiene

Run, in order:

1. `gofmt -l .` → empty output.
2. `go vet ./...` → exit 0.
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output,
   exit 0.
4. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0.
5. `git status --porcelain` → the ONLY entry is
   `?? internal/export/encrypt_failure_test.go` (or `A ` once staged).
6. `git diff --stat -- internal/export/encrypt.go internal/export/encrypt_test.go`
   → empty (zero production/existing-test diff).

**Verify**: all six commands produce exactly the stated results.

## Test plan

All new tests live in `internal/export/encrypt_failure_test.go`
(`package export`), modeled structurally on
`TestDecryptWrongPassphraseFails` in `internal/export/encrypt_test.go:92-109`:

- `TestDecryptTamperedArchiveFails` — table over three flipped offsets:
  header warning byte at offset 9 (pure AAD-binding proof), first
  ciphertext byte, last byte (auth tag). Each expects `ErrBadPassphrase`
  via `errors.Is` and an empty destination. Includes an unmutated
  control decrypt that must succeed.
- `TestUntarRejectsPathTraversal` — hand-built tar (entry `Typeflag:
  tar.TypeReg`) with names `../evil.md` and `nested/../../evil.md` fed
  straight to unexported `untar`; expects an error containing
  `unsafe tar path`, no file at the escape target, empty destination.
- `TestDecryptTruncatedArchive` — table: 10-byte garbage file, archive cut
  at byte 20 (mid-header), archive cut 8 bytes short (mid-ciphertext/tag).
  Each expects `ErrBadPassphrase` and an empty destination; the returned
  error is the no-panic proof.

Verification:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1 -v` → all
pass, including the 3 new test functions and the 3 pre-existing ones.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/export/ -count=1 -v`
      exits 0 and its output contains
      `--- PASS: TestDecryptTamperedArchiveFails`,
      `--- PASS: TestUntarRejectsPathTraversal`, and
      `--- PASS: TestDecryptTruncatedArchive`
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` exits 0
- [ ] `gofmt -l .` prints nothing; `go vet ./...` exits 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` exits 0 with
      no output
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `grep -c "unsafe tar path" internal/export/encrypt_failure_test.go`
      prints at least `1` (the traversal assertion exists)
- [ ] `head -1 internal/export/encrypt_failure_test.go` prints
      `package export` (internal test package — proves `untar` is exercised
      directly, not via a production seam)
- [ ] `git diff --stat 077318a..HEAD -- internal/export/encrypt.go internal/export/encrypt_test.go`
      shows no changes from this branch (zero diff to production code and
      the existing test file)
- [ ] `git status --porcelain` shows no modified/created files outside
      `internal/export/encrypt_failure_test.go` (plus the
      `plans/README.md` status row if you maintain it)
- [ ] `plans/README.md` status row for 248 updated (unless a reviewer told
      you they maintain the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/export/encrypt.go` or
  `internal/export/encrypt_test.go` changed since `077318a` and the
  "Current state" excerpts (AAD lines 89-90/135-140, header layout at
  150-160, guard at 266-272, sentinel at 47-49) no longer match the live
  code.
- Any test appears to require modifying `internal/export/encrypt.go` — for
  example, if `untar` or `parseHeader` turn out not to be callable from an
  in-package test file. Do NOT add a production seam, export a symbol, or
  refactor; report instead.
- The unmutated control decrypt in Step 2 fails: that means the fixture or
  offsets are wrong, not the production code — but if a second look at the
  offsets doesn't fix it, stop rather than loosening assertions.
- The offset-9 (warning flag) tamper test passes decryption: that would
  mean the AAD binding is already broken in production — a real security
  bug, out of scope to fix here; report it immediately.
- The traversal test finds `evil.md` written outside the destination: same
  situation — a live security bug in `untar`; report, do not patch.
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

- **What a reviewer should scrutinize**: that the tamper table includes the
  offset-9 (warning flag) row — the other two rows would still fail even
  without AAD binding, so that row alone carries the AAD guarantee; and
  that the tar header sets `Typeflag: tar.TypeReg` (without it the
  traversal entry is silently skipped and the test is vacuous).
- **Future interactions**: if the envelope format ever changes (a version
  bump of `envVersion`, a different KDF, a streaming decrypt), the
  offset-9 assumption ("parseHeader does not validate the warning byte,
  and it precedes salt/nonce") must be revisited; the `parseHeader`-derived
  `headerLen` keeps the other offsets format-agnostic. If `untar` ever
  gains symlink or directory-entry support, extend
  `TestUntarRejectsPathTraversal` with symlink-escape cases.
- **Explicitly deferred**: no fuzz target for `parseHeader`/`DecryptDir`
  (worth considering later via `go test -fuzz`, but the deterministic
  table above covers the advertised guarantees); no test for absolute-path
  tar entries (`/abs/evil.md`) because `filepath.Join(cleanDest, ...)`
  neutralizes them to a path inside the destination — they are contained,
  not rejected, and asserting an error there would be wrong.
