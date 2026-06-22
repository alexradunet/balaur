# Plan 141: Redact secrets from bash commands before they hit `audit_log`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report. When done, update the status row for
> this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat 0c06da8..HEAD -- internal/tools/os.go`

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `0c06da8`, 2026-06-22

## Why this matters

The bash OS-tool records the full command string to `audit_log`:
`auditOS(app, "bash", args.Command, …)`. The OS tools ship disabled and are
owner-initiated, but when enabled a command can carry an inline secret —
`curl -H "Authorization: Bearer sk-…"`, `export TOKEN=…`, `API_KEY=… ./run` — and
it lands in the audit ledger unredacted, where it may later leave the box in
exports/backups. AGENTS.md requires redacting secrets before persisting records
that may leave the box, while preserving auditability (blanking the whole command
would break the transparency pillar). The fix masks the *values* of common secret
shapes while keeping the rest of the command legible. The output the model
receives is unchanged — only the audited string is redacted.

## Current state

`internal/tools/os.go`:
- `auditOS` (54-57): `store.Audit(app, "os", "os."+tool, target, allowed, detail)`.
- `bashTool` (137-158) — the audit call (150):
  ```go
  out, err := exec.CommandContext(runCtx, "sh", "-c", args.Command).CombinedOutput()
  auditOS(app, "bash", args.Command, err == nil, map[string]any{"bytes": len(out)})
  ```
  `args.Command` is the raw command, logged verbatim as the audit `target`.
- The read/write/edit tools audit a file `Path`, not a command — they are not in
  scope (paths aren't secrets in the same way; only the bash command carries
  arbitrary inline secrets).

## Commands you will need

| Purpose | Command                          | Expected |
|---------|----------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`   | exit 0   |
| Tests   | `go test ./internal/tools/`      | all pass |
| Lint    | `make lint`                      | exit 0   |

## Steps

### Step 1: Add a conservative `redactSecrets` helper in `os.go`

Add a package-level helper (and the `regexp` import). Mask the VALUE, keep the
key/scheme so the audit stays meaningful. Compile the patterns once as package
vars (`regexp.MustCompile`):
```go
var secretPatterns = []*regexp.Regexp{
	// Authorization: Bearer <token>  and bare  Bearer <token>
	regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]{8,}`),
	// key=value / key: value for common secret-bearing names (quoted or not)
	regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?key|secret[_-]?key|client[_-]?secret|token|secret|password|passwd|pwd)(\s*[=:]\s*)("?)[^\s"']{4,}`),
	// AWS access key id
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
}

// redactSecrets masks the values of common secret shapes in a command string so
// it can be audited without leaking credentials, while keeping the rest legible.
func redactSecrets(s string) string {
	out := s
	out = secretPatterns[0].ReplaceAllString(out, `${1}***`)
	out = secretPatterns[1].ReplaceAllString(out, `${1}${2}${3}***`)
	out = secretPatterns[2].ReplaceAllString(out, `AKIA****************`)
	return out
}
```
(Keep it conservative — masking by named key + Bearer + AWS avoids the
over-redaction risk of a generic "long high-entropy token" rule, which would
mangle legitimate hashes/paths. Note that limitation in a comment.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Apply redaction to the bash audit target

In `bashTool`, redact the command before auditing (the executed command and the
returned output are UNCHANGED — only the audit string is redacted):
```go
auditOS(app, "bash", redactSecrets(args.Command), err == nil, map[string]any{"bytes": len(out)})
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`grep -n "redactSecrets(args.Command)" internal/tools/os.go` → one match.

### Step 3: Full gate

**Verify**: `go test ./internal/tools/` → all pass; `make lint` → exit 0.

## Test plan

- Add `TestRedactSecrets` to `internal/tools/os_test.go` (create if absent, else
  append). `redactSecrets` is a pure function — table-test it:
  - `curl -H "Authorization: Bearer sk-abc123def456" https://x` → contains
    `Bearer ***`, does NOT contain `sk-abc123def456`.
  - `export TOKEN=supersecretvalue` → contains `TOKEN=***` (or `token=***` per
    your group capture), does NOT contain `supersecretvalue`.
  - `API_KEY=AKIAEXAMPLE1234567890 ./run` and a literal `AKIA0123456789ABCDEF` →
    AKIA id masked.
  - `password: hunter2longenough` → value masked.
  - A benign command `ls -la /tmp && go test ./...` → returned UNCHANGED (no
    over-redaction).
  - Assert in each secret case that the original secret substring is absent from
    the output. NEVER hardcode a real credential — use obvious fake values.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./internal/tools/` passes, including `TestRedactSecrets`
- [ ] `make lint` exits 0
- [ ] `grep -n "redactSecrets(args.Command)" internal/tools/os.go` returns a match
- [ ] The benign-command test proves no over-redaction
- [ ] Only `internal/tools/os.go` (+ `os_test.go`) and `plans/readme.md` modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report if:
- The redaction regexes mangle a benign command in the test (over-redaction) —
  tighten the pattern rather than shipping it; report if you can't make benign
  commands pass through untouched.
- You're tempted to add a generic "any long token" rule — don't; it over-redacts.
  If the named patterns feel insufficient, report and ask before broadening.

## Scope

**In scope**: `internal/tools/os.go`, `internal/tools/os_test.go`,
`plans/readme.md` (status row).
**Out of scope**: the read/write/edit tools (they audit file paths, not
commands); the command actually executed (unchanged — redaction is audit-only);
the model-facing tool output (unchanged); `store.Audit`/`store/audit.go`.

## Git workflow

- Branch off `origin/main`: `improve/141-bash-audit-secret-redaction`.
- One commit; subject e.g. `fix(tools): redact secrets from bash commands in audit_log`.
- Do NOT push or open a PR.

## Maintenance notes

- This is best-effort masking of *common* secret shapes, not a guarantee — note
  that in the helper comment. If new secret formats show up in practice, add a
  named pattern; keep avoiding generic high-entropy rules that over-redact.
- The same redactor could later be applied to other audited free-text that may
  leave the box (exports, error details) — but keep this plan scoped to the bash
  command.
