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
- **Models:** local GGUF via kronk, or any OpenAI-compatible endpoint. If no
  model is configured, Balaur looks for the small default Qwen2.5 3B GGUF under
  `pb_data/models/`.
- **Heads:** sub-agents are auth records with short-lived tokens; their
  permissions are rows in `grants`, enforced in one code path
  (`internal/heads`), audited in `audit_log`. Tests prove out-of-scope
  access fails.
- **Memory & skills with consent:** the model proposes (`remember`,
  `propose_skill`); proposals render as cards — in chat and on `/memory`
  and `/skills` — that the owner approves, edits, or dismisses. Nothing
  enters context without approval. Injection is two-tier: high-importance
  memories always, message-matched recall per turn, plus a compact skills
  index loaded on demand via the `skill` tool. Every lifecycle step is
  audited.
- **One master conversation, persisted:** every turn is stored; the model
  sees only the recent window plus memory (persistence is not context).
  History survives restarts and renders on page load.
- **The recap telescope:** scrolling up past today reveals the past as
  summaries — days for the current week, then weeks, months, quarters,
  years — each expandable down to the preserved day transcript. Summaries
  generate hierarchically (days → weeks/months → quarters → years) via an
  idempotent hourly catch-up job (`BALAUR_RECAP=0` disables), audited.
- **Commitments captured in chat:** `task_add`, `task_list`, `task_done`,
  `task_snooze`, `task_drop` — one-offs and recurring habits/chores with a
  tiny recurrence DSL (`daily`, `every:3d`, `weekly:mon,thu`, `monthly:15`,
  fixed-schedule or from-completion). Tasks live in the `tasks` collection;
  completions land in `entries`, the life-log substrate. New tasks render
  as live cards in chat. Every turn is grounded in the present moment —
  date, time, timezone — so "tomorrow at 10" resolves against the box's
  clock, never the model's guess.
- **Balaur reminds on its own:** a minute cron fires due reminders into the
  master conversation — composed in Balaur's voice when a model is
  configured, a plain deterministic line otherwise, batched into one
  message per tick. Firing is idempotent across restarts; the first tick
  after downtime is the catch-up. The open chat polls nudges in live;
  `BALAUR_NUDGE=0` disables.
- **The morning briefing:** once per local day, after the briefing hour
  (default 9, `BALAUR_BRIEFING_HOUR` overrides), Balaur opens the day —
  overdue items, today's commitments, habit streaks from the `entries`
  log — composed in its voice with a deterministic fallback. Idempotency
  derives from the message record itself; a box asleep at the hour briefs
  at wake; quiet days stay quiet. `BALAUR_BRIEFING=0` disables. The model
  also sees a Today block of open commitments in every chat turn, so the
  companion knows your day unprompted.
- **/tasks — life organization:** the operational list (cards with
  Done / Snooze / Drop), a month calendar, and a 14-day timeline — the
  forward mirror of the recap telescope. Calendar and timeline project
  recurrence rules forward, read-only; actions live on the list cards.
  Day pages are roadmap.
- **The life log — owner-defined:** Balaur does not decide what a life is
  made of. `log_entry` keeps whatever you track under kinds you invent
  (weight, mood, sleep, pages-read…), numeric or textual, backdatable;
  `entry_series` reads trends, `entry_drop` corrects. `/life` mirrors what
  exists: sparklines for numeric kinds, recent lines for the rest, live
  habit streaks on top. Nothing is predefined; the briefing reflects
  yesterday's log in one line.
- **OS access mode:** the four classic tools — `read`, `write`, `edit`,
  `bash` — exist but ship **disabled**. Set `BALAUR_OS_ACCESS=1` to enable;
  every invocation is audited.

## Quick start

```bash
go run . serve
```

Then open http://127.0.0.1:8090/ for Balaur, or
http://127.0.0.1:8090/_/ to create the superuser and inspect data.

Balaur defaults to a small local tool-capable model:

```bash
llmfit download bartowski/Qwen2.5-3B-Instruct-GGUF \
  --quant Q4_K_M \
  --output-dir pb_data/models
```

Then run:

```bash
go run . serve
```

You can still override the default explicitly:

```bash
# Local GGUF through kronk (downloads llama.cpp runtime on first use):
BALAUR_CHAT_MODEL=/path/to/model.gguf go run . serve

# Or Synthetic's OpenAI-compatible API aliases in the chatbar picker:
SYNTHETIC_API_KEY=... go run . serve

# Or any OpenAI-compatible endpoint (llama-server, Ollama, remote):
BALAUR_REMOTE_URL=http://127.0.0.1:11434/v1 \
BALAUR_REMOTE_MODEL=qwen3:8b \
go run . serve
```

Optional:

```bash
BALAUR_EMBED_MODEL=/path/to/embedding.gguf   # local embeddings model
BALAUR_REMOTE_API_KEY=...                    # key for remote endpoints
SYNTHETIC_API_KEY=...                        # enables Synthetic API choices
# BALAUR_SYNTHETIC_API_KEY also works if you prefer a Balaur-scoped env var.
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
main.go            wire-up: PocketBase app, migrations, routes, recap cron
migrations/        schema as Go code (collections + API rules)
internal/agent/    the conversation loop: model → tools → model
internal/llm/      one model seam: kronk (local) + OpenAI-compatible HTTP
internal/conversation/ master conversation: persistence + context window
internal/recap/    the telescope: period math + hierarchical summaries
internal/tasks/    commitments: recurrence DSL + task verbs on the entries life log
internal/heads/    sub-agent identities, grants, audit — the rule boundary
internal/knowledge/ memory & skill lifecycle, context injection — the consent boundary
internal/store/    shared PocketBase helpers (audit)
internal/tools/    agent tools: knowledge (always) + OS access (opt-in)
internal/web/      HTMX handlers: chat, memory & skills pages, cards, recap
web/               embedded templates and static assets (Basm CSS)
```

Read `AGENTS.md` for the engineering rules (KISS, YAGNI, suckless, the rule
boundary) and `DESIGN.md` for the Basm design system.

## Roadmap (not shipped — honesty ledger)

- Day pages with journaling (`/day/{date}`: your thoughts + the day's recap,
  completions, and logs)
- Johnny Decimal Markdown vault mirror: one-way export + git history
- FTS5/embedding recall (today: importance-gated upfront + LIKE-matched
  recall; the `internal/search` spike holds the FTS5 driver decision)
- Branch sub-conversations with merge-back (the schema is ready; the
  master conversation ships first)
- Encrypted export
- Multi-human accounts (the schema allows it; v1 serves one owner)

## License

AGPL-3.0-or-later. Built in the open in Brașov.
