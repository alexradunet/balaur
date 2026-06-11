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
- **Verify, don't trust:** a runtime check audits each reply's words
  against its deeds. If the model claims a reminder or log was saved but
  no capture tool succeeded that turn, it gets one bounded pass to
  actually call the tool — and if it still claims without doing, a plain
  *Balaur · check* note lands in the chat and the record. Built from real
  live-test failures; trust the task card, not the words.
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
- **Day pages — where a day lives:** `/day/{date}` assembles your journal
  (written in chat via `journal_write` — your words, verbatim — or on the
  page itself), the day's recap with its preserved transcript, what got
  done, and what was logged. Prev/next navigation; calendar cells and
  recap day cards link in. Journal entries are removable on the page —
  the owner's right over their own words, never a model verb.
- **OS access mode:** the four classic tools — `read`, `write`, `edit`,
  `bash` — exist but ship **disabled**. Set `BALAUR_OS_ACCESS=1` to enable;
  every invocation is audited.
- **A machine-facing CLI:** the same binary speaks JSON for external
  harnesses — including other LLMs — that drive, seed, inspect, and verify
  a box without scraping HTML. `balaur chat` runs the identical turn
  pipeline the web UI runs (`internal/turn`); `task`, `memory`, `skill`,
  `life`, `journal`, `day`, `recap`, `history`, `audit`, `model` work
  deterministically without one; `balaur verify` replays the words-vs-deeds
  check on the record. See "CLI for agents & test harnesses".
- **Self-awareness:** the binary embeds its own self-knowledge
  (`internal/self`) — what Balaur is, its architecture, the
  self-development loop — plus a build stamp and a live capability
  inventory (registered tools, approved skills, gates, model choice).
  The model consults it through the read-only `self` tool instead of
  guessing about itself; harnesses read it as JSON via `balaur self`.
- **Self-development (opt-in):** with `BALAUR_OS_ACCESS=1` and
  `BALAUR_SOURCE` pointing at the repo checkout, Balaur can analyze and
  modify its own code using the ordinary OS tools, following the embedded
  devloop: branch → edit → gofmt/vet/test → build a candidate binary →
  verify it with its own CLI harness and `scripts/fake-model.py` → report
  for the owner to restart. It never restarts or replaces its own running
  binary, and the honesty check applies: "fixed" and "tested" are claims
  that need deeds in the same turn.
- **balaur-extensions — runtime tools, consent-gated:** one JavaScript
  file in `pb_extensions/` registering new tools via
  `balaur.registerTool`; run by goja (the engine PocketBase's jsvm uses —
  still no CGO). The `extensions` collection is the consent ledger:
  discovery proposes, the owner approves (pinning the file's sha256),
  any change re-proposes, every invocation is audited. Balaur can write
  and propose its own extensions in chat (`propose_extension`) — new
  capability without rebuild or restart, but never without the owner.
  See "balaur-extensions".

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
BALAUR_SOURCE=/path/to/balaur                # your source checkout (self-development)
BALAUR_MAX_STEPS=24                          # raise the tool-round cap for coding sessions
BALAUR_EXT_DIR=/path/to/pb_extensions        # relocate balaur-extensions (default: next to pb_data)
```

## Build

```bash
CGO_ENABLED=0 go build -o balaur .
```

Cross-compiles to linux/darwin/windows, amd64/arm64, from any machine —
no C toolchain. The binary is static; models and the llama.cpp runtime are
downloaded data, stored outside the repo and outside the binary.

## CLI for agents & test harnesses

Every command prints one JSON value on stdout; failures print
`{"error": ...}` on stderr and exit non-zero. Web and CLI are gateways
over the same turn pipeline (`internal/turn`), so what the CLI observes
is evidence about what the web UI does.

| Command | What it does | Model? |
|---|---|---|
| `balaur chat "<msg>"` | One real companion turn: context, agent loop, honesty check, persistence. Reports the reply, every tool call with args + result, proposal references, and the words-vs-deeds verdict. | yes |
| `balaur task add/list/done/snooze/drop` | Commitments, directly. | no |
| `balaur memory propose/list/recall/approve/reject/archive/edit` | Memory lifecycle across the consent boundary. | no |
| `balaur skill propose/list/show/approve/reject/archive` | Skill lifecycle. | no |
| `balaur life log/series/kinds/drop` | The owner-defined life log. | no |
| `balaur journal write`, `balaur day <date>` | Keep a journal line verbatim; read one day (journal, log, done, recap). | no |
| `balaur recap show/ensure` | Read stored summaries; run the idempotent catch-up. | ensure |
| `balaur history [--date]` | The persisted master conversation, tool rounds included. | no |
| `balaur audit [--action] [--actor]` | The audit log — the deeds claims are checked against. | no |
| `balaur verify` | Words vs deeds for the last persisted turn. | no |
| `balaur model` | Available and active model choices — a harness precondition check. | no |
| `balaur self [--section]` | Build stamp, live capability inventory, source seam; optionally one self-knowledge section (overview, architecture, capabilities, source, devloop). | no |
| `balaur ext list/approve/disable/show` | balaur-extensions lifecycle: review proposals, consent (pins sha256), turn off, inspect code. | no |

Every command works on a fresh data dir: pending migrations apply on
first touch, so harness runs isolate cheaply with `--dir`:

```bash
balaur --dir "$(mktemp -d)" task add --title "Smoke test"
```

`chat` turns become deterministic with the scriptable fake model server —
script what the "model" says and which tools it calls:

```bash
cat > script.json <<'EOF'
[
  {"tool": "task_add", "args": {"title": "Water the plants", "due": "2027-03-01"}},
  {"text": "I've added watering the plants for March 1."}
]
EOF
python3 scripts/fake-model.py script.json &

export BALAUR_REMOTE_URL=http://127.0.0.1:11435/v1 BALAUR_REMOTE_MODEL=fake
balaur --dir /tmp/box chat "remind me to water the plants on march 1"
balaur --dir /tmp/box verify            # words vs deeds, from the record
balaur --dir /tmp/box audit --action task.
```

The OS-access tools are deliberately not mirrored as commands: a shell
already has the shell. `BALAUR_OS_ACCESS` gates what the *model* may
reach, and that gate applies identically under `balaur chat`.

## balaur-extensions

An extension is one JavaScript file in `pb_extensions/` (next to
`pb_data/`, mirroring the `pb_hooks` convention; `BALAUR_EXT_DIR`
overrides). It registers tools; handlers may fetch over HTTP. That is the
whole API — extensions add verbs, not privileges: no filesystem, no
shell, no npm, no DB.

```js
// balaur-extension: Current weather for the home town.
balaur.registerTool({
  name: "weather_home",
  description: "Current weather at home.",
  parameters: {type: "object", properties: {}},
  handler: function (args) {
    var res = balaur.http({url: "https://wttr.in/Brasov?format=3"});
    return res.body;
  }
})
```

The consent flow, enforced by the `extensions` collection and audited at
every step:

1. A file appears (the owner drops it in, or Balaur writes one in chat
   via `propose_extension`) → it is **proposed**, never executed.
2. The owner reviews (`balaur ext show <name>`) and approves
   (`balaur ext approve <name>`) — approval pins the file's **sha256**.
3. From the next turn, its tools are live in every gateway (web, CLI) —
   no rebuild, no restart.
4. Any change to the file drops it from service and re-proposes it;
   approval is always consent to exact content. Load-time side effects
   are forbidden (`balaur.http` throws outside handlers), invocations
   run in a fresh VM with a 30s cap, and every call lands in
   `audit_log`.

Self-evolution has two speeds: extensions grow new verbs at runtime;
the devloop (above) evolves the Go core through an owner-restarted
binary. Both end at the same gate — nothing becomes part of Balaur
without the owner's explicit yes.

## Development

```bash
gofmt -l .        # formatting (must be empty)
go vet ./...
go test ./...
```

Project layout:

```txt
main.go            wire-up: PocketBase app, migrations, CLI, routes, crons
migrations/        schema as Go code (collections + API rules)
internal/agent/    the conversation loop: model → tools → model
internal/llm/      one model seam: kronk (local) + OpenAI-compatible HTTP
internal/turn/     the channel-agnostic turn pipeline + model resolution
internal/conversation/ master conversation: persistence + context window
internal/recap/    the telescope: period math + hierarchical summaries
internal/tasks/    commitments: recurrence DSL + task verbs on the entries life log
internal/verify/   runtime honesty check: words audited against tool deeds
internal/heads/    sub-agent identities, grants, audit — the rule boundary
internal/knowledge/ memory & skill lifecycle, context injection — the consent boundary
internal/store/    shared PocketBase helpers (audit)
internal/tools/    agent tools: knowledge (always) + OS access (opt-in)
internal/self/     self-awareness: embedded self-knowledge + live inventory
internal/ext/      balaur-extensions: consent-gated runtime tools (JS/goja)
internal/web/      HTMX gateway: chat, memory & skills pages, cards, recap
internal/cli/      machine-facing gateway: balaur subcommands, JSON out
web/               embedded templates and static assets (Basm CSS)
```

Read `AGENTS.md` for the engineering rules (KISS, YAGNI, suckless, the rule
boundary) and `DESIGN.md` for the Basm design system.

## Roadmap (not shipped — honesty ledger)

- Johnny Decimal Markdown vault mirror: one-way export + git history
- FTS5/embedding recall (today: importance-gated upfront + LIKE-matched
  recall; the `internal/search` spike holds the FTS5 driver decision)
- Branch sub-conversations with merge-back (the schema is ready; the
  master conversation ships first)
- Encrypted export
- Multi-human accounts (the schema allows it; v1 serves one owner)

## License

AGPL-3.0-or-later. Built in the open in Brașov.
