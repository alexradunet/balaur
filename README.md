# Balaur

> **A sovereign local-first personal agent, served from one binary.**

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/go-%E2%89%A51.26-00ADD8.svg)](./go.mod)

Balaur is a personal AI companion that lives on a box you own: a single Go
executable embedding [PocketBase](https://pocketbase.io) for data, auth and
migrations, an HTMX web interface, and local LLM inference through
[kronk](https://github.com/ardanlabs/kronk) (llama.cpp loaded via purego —
no CGO anywhere in the build).

The name comes from the Romanian fairy-tale balaur: a dragon with multiple
heads. Balaur keeps one main head — the master life conversation. Focused
work happens as temporary sub-heads: each is a real auth identity whose data
access is scoped by explicit grants, checked on every access, and written to
an audit log. When the work is done, the head merges back and its access
dies with it.

Your life is not a product. The record of your life should live in a
database you own and can open with any SQLite tool.

## Current shape

- **One binary:** `balaur` — web UI, database, migrations, agent loop.
- **Data:** PocketBase collections — `conversations`, `messages`,
  `memories`, `skills`, `heads`, `grants`, `audit_log` — in plain SQLite
  under `pb_data/`.
- **UI:** server-rendered Go templates + HTMX, styled by the Basm design
  system (see `DESIGN.md`). The PocketBase dashboard at `/_/` stays the
  superuser engine room.
- **Models:** local GGUF via kronk, downloadable from the web UI, or any
  OpenAI-compatible endpoint — chosen explicitly, never auto-routed.
- **Heads:** sub-agents are auth records with short-lived tokens; their
  permissions are rows in `grants`, enforced in one code path
  (`internal/heads`), audited in `audit_log`. Tests prove out-of-scope
  access fails.
- **OS access mode:** the four classic tools — `read`, `write`, `edit`,
  `bash` — exist but ship **disabled**. Set `BALAUR_OS_ACCESS=1` to enable;
  every invocation is audited.

## Quick start

```bash
go run . serve
```

Then open http://127.0.0.1:8090/ for Balaur, or
http://127.0.0.1:8090/_/ to create the superuser and inspect data.

Pick a model from the web UI. Balaur ships a small curated catalog of
tool-capable 2026 GGUF chat models; downloaded files are stored under
`pb_data/models/` and the completed download becomes the active chat model.

Environment variables still work as bootstrap/fallback configuration:

```bash
# Local GGUF through kronk (the web UI removes the need for this):
BALAUR_CHAT_MODEL=/path/to/model.gguf go run . serve

# Or any OpenAI-compatible endpoint (llama-server, Ollama, remote):
BALAUR_REMOTE_URL=http://127.0.0.1:11434/v1 \
BALAUR_REMOTE_MODEL=qwen3:8b \
go run . serve
```

Optional:

```bash
BALAUR_EMBED_MODEL=/path/to/embedding.gguf   # local embeddings model
BALAUR_REMOTE_API_KEY=...                    # key for remote endpoints
BALAUR_OS_ACCESS=1                           # enable read/write/edit/bash tools
```

## Build

```bash
CGO_ENABLED=0 go build -o balaur .
```

Cross-compiles to linux/darwin/windows, amd64/arm64, from any machine —
no C toolchain. The binary is static; models and the llama.cpp runtime are
downloaded data, stored outside the repo and outside the binary.

## Development

```bash
gofmt -l .        # formatting (must be empty)
go vet ./...
go test ./...
```

Project layout:

```txt
main.go            wire-up: PocketBase app, migrations, routes
migrations/        schema as Go code (collections + API rules)
internal/agent/    the conversation loop: model → tools → model
internal/llm/      one model seam: kronk (local) + OpenAI-compatible HTTP
internal/heads/    sub-agent identities, grants, audit — the rule boundary
internal/tools/    OS access tools (read, write, edit, bash), opt-in
internal/web/      HTMX handlers
web/               embedded templates and static assets (Basm CSS)
```

Read `AGENTS.md` for the engineering rules (KISS, YAGNI, suckless, the rule
boundary) and `DESIGN.md` for the Basm design system.

## Roadmap (not shipped — honesty ledger)

- Johnny Decimal Markdown vault mirror: one-way export + git history
- Vault auto-recall (embeddings exist; recall wiring does not)
- Conversation persistence + branch/merge UI for heads
- Encrypted export
- Multi-human accounts (the schema allows it; v1 serves one owner)

## License

AGPL-3.0-or-later. Built in the open in Brașov.
