# Plan 224 (DIR-03): Close the backup loop — add a `balaur restore` CLI verb over the existing `DecryptDir`

> **Direction bet, but the most execute-ready of the six.** The crypto and the
> reverse function already exist and are tested; this plan wires the missing
> owner-facing half. It is still a *new capability*, so it needs an explicit
> owner go-ahead before an executor runs — but unlike the other DIR plans, the
> design space is small and the slice is thin.

## Status

- **Priority**: P2 (direction, but concrete and high-value)
- **Effort**: M
- **Risk**: MEDIUM (writes owner data back onto the box; must be fail-safe)
- **Depends on**: none (builds on shipped plans 192/194/195 export)
- **Category**: direction / sovereignty completeness
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

Balaur ships **half** a backup story. The export side is complete and
owner-facing:

```go
// internal/cli/export.go:36-40 — the --encrypt path is wired
cmd.Flags().BoolVar(&encrypt, "encrypt", false, "encrypt the export into a single passphrase-protected archive")
cmd.Flags().StringVar(&archive, "archive", "", "output archive path (required with --encrypt)")
...
if encrypt { return runEncrypt(cmd, app, archive) }
```

`runEncrypt` renders the mirror into a temp dir and calls
`export.EncryptDir(tmp, archive, passphrase)` (`encrypt.go:57`). AES-256-GCM over
a deterministic tar, scrypt-derived key, tamper-bound header. Solid.

But the **reverse** function is explicitly marked as test-only:

```go
// internal/export/encrypt.go:104-106
// ErrBadPassphrase with NOTHING written to destDir (decrypt fully in memory,
// untar only after Open succeeds). Used by the round-trip test; not a CLI verb.
func DecryptDir(srcFile, destDir, passphrase string) error {
```

So an owner can produce an off-box encrypted backup but has **no supported way to
open it** with the binary that made it. On the Sovereignty pillar ("your data,
your box, exportable, recoverable"), a backup you cannot restore is not a backup
— it's a checksum of your loss. This closes that gap with the function that
already exists.

## The bet, scoped

Add a `balaur restore` CLI verb that decrypts an archive produced by
`export --encrypt` back into a plaintext directory tree the owner can read.

**Deliberately NOT in scope for the thin slice (Pareto first):**
restore-into-the-live-database (re-importing nodes into PocketBase). That is a
much harder, riskier merge/conflict problem (dedup, id collisions, status
reconciliation) and belongs in its own later plan. The v1 win is: *get my
readable data back out of the encrypted blob.* The export is already a Markdown
mirror, so a decrypted tree IS the owner's recovered, human-readable data.

State this boundary in the command's `Short`/`Long` so it never over-promises.

## Current state (verified at `ef9f2df`)

- `export.DecryptDir(srcFile, destDir, passphrase)` — `internal/export/encrypt.go:106`.
  Fails closed: returns `ErrBadPassphrase` with nothing written on bad
  passphrase/tamper (decrypts fully in memory, untars only after auth succeeds —
  `encrypt.go:104-105`).
- `export.EncryptDir` — `encrypt.go:57`; the format the restore must read.
- Passphrase convention: env var `BALAUR_EXPORT_PASSPHRASE`, never a flag — see
  `cli/export.go:14-17,64-67`. Restore MUST reuse the same env var, same
  no-plaintext-in-shell-history reasoning.
- CLI verb registration pattern: `exportCmd(app)` returns a `*cobra.Command`
  using the shared `run(app, "verb", fn)` envelope (`cli/export.go:25,38`).
  `restore` follows the identical shape.

## Proposed shape

A new `internal/cli/restore.go` (sibling of `export.go`), `restoreCmd(app)`:

```go
// restore decrypts an archive produced by `export --encrypt` into a plaintext
// directory tree. It does NOT re-import into the live database (that is a
// separate, later capability) — it recovers the owner's readable Markdown mirror.
// The passphrase comes from BALAUR_EXPORT_PASSPHRASE (never a flag), matching export.
func restoreCmd(app core.App) *cobra.Command {
    var archive, out string
    cmd := &cobra.Command{
        Use:   "restore",
        Short: "decrypt an --encrypt export archive into a readable Markdown tree",
        Args:  cobra.NoArgs,
    }
    cmd.Flags().StringVar(&archive, "archive", "", "encrypted archive path (required)")
    cmd.Flags().StringVar(&out, "out", "", "destination directory (must be empty or non-existent)")
    cmd.RunE = run(app, "restore", func(cmd *cobra.Command, args []string) (any, error) {
        // validate archive != "" and out != "";
        // refuse a non-empty existing out dir (never clobber owner data);
        // read passphrase from BALAUR_EXPORT_PASSPHRASE, error if empty;
        // export.DecryptDir(archive, out, passphrase);
        // map ErrBadPassphrase to a clean owner-facing message;
        // return {"dest": out} for the JSON envelope.
    })
    return cmd
}
```

Then register `restoreCmd(app)` wherever `exportCmd(app)` is added to the root
command (find the registration site; it is in the CLI wiring next to the other
verbs).

### Safety rules (non-negotiable — restore writes to the box)

1. **Never clobber.** Refuse to write into a non-empty existing `--out` dir.
   Recovery must not be able to destroy live data.
2. **Fail closed, already does.** `DecryptDir` writes nothing on a bad
   passphrase; surface `ErrBadPassphrase` as a plain message ("wrong passphrase
   or corrupt archive — nothing was written"), not a stack trace, and not the
   passphrase itself.
3. **No secrets in output.** The JSON envelope and any log line carry the dest
   path only — never the passphrase, never archive contents.

## Test plan

- `internal/cli/restore_test.go` (or extend the export test): full round-trip
  through the CLI surface — `export --encrypt` to a temp archive, then `restore`
  to a temp out dir, assert the recovered tree is byte-identical to a plain
  `ExportMirror` of the same data (the export is deterministic — `export.go:94-95`).
- Bad-passphrase case: assert `restore` errors AND the out dir is empty/untouched.
- Non-empty-out case: assert it refuses without writing.
- `DecryptDir`'s own round-trip + tamper tests already exist
  (`internal/export/encrypt_test.go`) — do not duplicate the crypto-level tests;
  test only the CLI verb behavior.

## Done criteria

- [ ] `balaur restore --archive X --out Y` decrypts a real `export --encrypt`
      archive into a readable tree.
- [ ] Wrong passphrase → clean error, nothing written.
- [ ] Non-empty `--out` → refused, nothing written.
- [ ] `grep -n "not a CLI verb" internal/export/encrypt.go` — update the stale
      `DecryptDir` comment ("not a CLI verb") since it now is one.
- [ ] CLI table in any docs/`internal/self/knowledge.md` listing verbs includes
      `restore` (this overlaps plan 214's doc-truth sweep — coordinate so the
      verb table is updated once).
- [ ] `go test ./internal/cli/... ./internal/export/...` PASS; `go test ./...` green.

## STOP conditions

- If the registration site for CLI verbs is not the obvious sibling of
  `exportCmd` (e.g. verbs are auto-discovered), follow the real pattern and note
  it — do not invent a parallel registration mechanism.
- If `DecryptDir`'s signature/format differs from the excerpt above (a later
  change to the archive format), reconcile against the live `encrypt.go` before
  writing the verb.
- Do NOT attempt DB re-import in this plan — if that scope creeps in, stop and
  split it into a new plan.

## Notes

- The `runEncrypt` warning copy ("if you lose this passphrase, this backup is
  UNRECOVERABLE", `cli/export.go:68-69`) is now *literally enforced* by the
  matching restore — good. Keep the symmetry.
