# Balaur

> **A sovereign local-first personal agent, served from one binary.**

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/go-%E2%89%A51.26-00ADD8.svg)](./go.mod)

Balaur is a personal AI companion that lives on a box you own: a single Go
executable embedding [PocketBase](https://pocketbase.io) for data, auth and
migrations, an HTMX web interface, and local LLM inference served by a
[llamafile](https://github.com/Mozilla-Ocho/llamafile) engine that Balaur runs
as a subprocess and reaches over the OpenAI-compatible API — the same seam it
uses for any remote provider.

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
- **Models:** out of the box, a self-contained **Qwen3.5-4B llamafile**
  (engine + weights in one executable) that Balaur downloads on first serve and
  runs as a subprocess. Add any OpenAI-compatible endpoint — your own
  llama-server, Ollama, or a remote API — and select it explicitly. Advanced:
  point `BALAUR_CHAT_MODEL` at your own `.llamafile` or `.gguf`.
- **Heads:** sub-agents are auth records with short-lived tokens; their
  permissions are rows in `grants`, enforced in one code path
  (`internal/heads`), audited in `audit_log`. Tests prove out-of-scope
  access fails. Each active head also has a persistent, focused,
  tool-free chat channel — a branch conversation separate from the
  master, opened in the dock from the heads card focus.
- **Memory & skills with consent:** the model proposes (`remember`,
  `propose_skill`); proposals render as cards — in chat and in the memory
  & skills card focuses — that the owner approves, edits, or dismisses. Nothing
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
- **The quests focus — life organization:** the operational list (cards
  with Done / Snooze / Drop) lives in the quests card's focus
  (`/focus/quests`); a month calendar and a 14-day timeline are their own
  cards — the forward mirror of the recap telescope. Calendar and timeline
  project recurrence rules forward, read-only; actions live on the task cards.
- **The life log — owner-defined:** Balaur does not decide what a life is
  made of. `log_entry` keeps whatever you track under kinds you invent
  (weight, mood, sleep, pages-read…), numeric or textual, backdatable;
  `entry_series` reads trends, `entry_drop` corrects. The lifelog card
  (`/focus/lifelog`) mirrors what exists: sparklines for numeric kinds,
  recent lines for the rest, live habit streaks on top. Nothing is
  predefined; the briefing reflects
  yesterday's log in one line.
- **The day card — where a day lives:** the `day` card's focus
  (`/focus/day?date={date}`) assembles your journal (written in chat via
  `journal_write` — your words, verbatim — or in the focus itself), the day's
  recap with its preserved transcript, what got done, and what was logged.
  Prev/next navigation; calendar cells and recap day cards deep-link in. The
  tile is a read-only summary. Journal entries are removable in the focus —
  the owner's right over their own words, never a model verb.
- **OS access mode:** the four classic tools — `read`, `write`, `edit`,
  `bash` — exist but ship **disabled**. Set `BALAUR_OS_ACCESS=1` to enable;
  every invocation is audited.
- **A machine-facing CLI (API v1):** the same binary speaks JSON for external
  harnesses — including other LLMs — that drive, seed, inspect, and verify
  a box without scraping HTML. Every command emits a versioned envelope
  `{"v":1,"kind":"<cmd>.<sub>","data":{…}}`; additive fields inside `data`
  are free, renames or removals bump `v`. `balaur doctor` preflights the box
  (no model calls). `balaur chat` runs the identical turn pipeline the web UI
  runs (`internal/turn`); `task`, `memory`, `skill`, `life`, `journal`, `day`,
  `recap`, `history`, `audit`, `model` work deterministically without one;
  `balaur verify` replays the words-vs-deeds check on the record. See "CLI for
  agents & test harnesses".
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
make run
```

For live-reload development, run `make dev` (uses [air](https://github.com/air-verse/air)).

If `air` is not installed, `make dev` downloads and runs the latest release automatically.

Then open http://127.0.0.1:8090/ for Balaur, or
http://127.0.0.1:8090/_/ to create the superuser and inspect data.

`make dev` uses the repo-local `pb_data/` directory and restarts Balaur
whenever Go, template, CSS, JS, or static asset files change. If the
always-on user service is running, stop it first or move one process to a
different port:

```bash
make stop-user-service
make dev
```

Just run it:

```bash
go run . serve
```

On first serve, Balaur downloads its default model — the **Qwen3.5-4B
llamafile** (~4 GB, a single self-contained executable) — into
`pb_data/models/` and activates it once the download finishes. Progress shows
in the settings card's models section (`/focus/settings?section=models`).
Disable the auto-download with `BALAUR_AUTO_MODEL=0`.

Overrides:

```bash
# Use your own model file (a fat .llamafile, or a bare .gguf):
BALAUR_CHAT_MODEL=/path/to/model.llamafile go run . serve

# A bare .gguf needs a separate llamafile engine binary; default location is
# pb_data/bin/llamafile, override with BALAUR_LLAMAFILE:
BALAUR_CHAT_MODEL=/path/to/model.gguf BALAUR_LLAMAFILE=/opt/llamafile go run . serve

# Add OpenAI-compatible endpoints (your own llama-server, Ollama, a remote
# API) from the settings card's models section
# (/focus/settings?section=models). Base URL, model id, and optional API key are
# stored in PocketBase; the active model is selected explicitly.
```

**llamafile**: Balaur runs the model as a subprocess with `--server` on a
local port and talks to it over the OpenAI-compatible API, so local and remote
models share one code path (`internal/llm`). A fat `.llamafile` is its own
engine; a bare `.gguf` is served by a separate llamafile engine binary. Either
way the llama.cpp runtime is bundled — no C toolchain, no llama.cpp-head
tracking. The process starts on the first chat and is torn down on shutdown.

**Extension engine**: Balaur uses goja (no tags; pins a master commit) for the JavaScript sandbox. Bumping it is a deliberate act—run `go test ./internal/ext/` after changing.

API keys are never rendered back into the UI or audit log, but they live in
the local PocketBase data directory and backups. Treat `pb_data/` as secret.

Optional environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `BALAUR_ALLOWED_HOSTS` | (unset) | Comma-separated `host[:port]` values allowed as the Host header beyond loopback (LAN names, NetBird — see [docs/netbird.md](docs/netbird.md)) |
| `BALAUR_CHAT_MODEL` | (unset) | Path to a local model file (`.llamafile` or `.gguf`); overrides the default and the settings models-section choice |
| `BALAUR_AUTO_MODEL` | `1` | Set to `0` to skip the serve-start auto-download of the default Qwen3.5-4B llamafile |
| `BALAUR_LLAMAFILE` | (unset) | Path to the llamafile engine binary used to serve a bare `.gguf`; defaults to `pb_data/bin/llamafile`. Not needed for a fat `.llamafile`. |
| `BALAUR_EMBED_MODEL` | (unset) | Path to a local embedding model GGUF (reserved for future embedding recall; lexical recall uses FTS5 today) |
| `BALAUR_OS_ACCESS` | `0` | Set to `1` to enable read/write/edit/bash tools (every invocation is audited) |
| `BALAUR_SOURCE` | (unset) | Path to the Balaur source checkout for self-development (requires `BALAUR_OS_ACCESS=1`) |
| `BALAUR_MAX_STEPS` | (unset) | Raise the tool-round cap per turn; default is 8 (useful for coding sessions) |
| `BALAUR_EXT_DIR` | `pb_data/` | Relocate the `pb_extensions/` directory (default: next to pb_data) |
| `BALAUR_RECAP` | `1` | Set to `0` to disable hourly recap generation |
| `BALAUR_NUDGE` | `1` | Set to `0` to disable minute-cadence task nudging |
| `BALAUR_BRIEFING` | `1` | Set to `0` to disable the morning briefing |
| `BALAUR_BRIEFING_HOUR` | `9` | Hour (0–23) to open the day with a briefing |
| `BALAUR_DEV_SEED` | `0` | Set to `1` to enable the `/ui/dev/seed-recaps` endpoint for testing |

### Pinning recap periods to a timezone

By default, recap period boundaries (daily, weekly, monthly…) follow the
box's local clock. If the box ever moves to a different timezone — laptop
travels, VPS migrated, `TZ` reconfigured — the period boundaries shift, and
old summaries stop matching the new boundaries.

To keep boundaries stable across box moves, add a record to the
`owner_settings` collection (PocketBase dashboard → `owner_settings` →
`+`):

| key        | value (example)    |
|------------|--------------------|
| `timezone` | `Europe/Bucharest` |

The value must be a valid IANA timezone name (e.g. `America/New_York`,
`Asia/Tokyo`). Balaur resolves it per cron tick — a dashboard edit takes
effect on the next hourly run without a restart. An unrecognised name falls
back silently to the box clock.

**Note on existing summaries**: changing the `timezone` key re-anchors
*future* period boundaries only. Summaries generated under the previous zone
keep their original boundaries and remain visible; you may see a one-time
seam at the change date. No migration of historical summaries is attempted.

To reach Balaur from a NetBird network without embedding any VPN code into
the binary, see [docs/netbird.md](docs/netbird.md).

## Build

```bash
CGO_ENABLED=0 go build -o balaur .
```

## Always on

On Linux with systemd, Balaur can run as a user service. This keeps it on
loopback at http://127.0.0.1:8090/ and stores production data under
`~/.local/share/balaur/pb_data`, separate from the repo-local development
data directory.

```bash
make install-user-service
make start-user-service
```

The install target builds a CGO-free binary, copies it to
`~/.local/bin/balaur`, installs `contrib/systemd/balaur.service` to
`~/.config/systemd/user/balaur.service`, creates `~/.config/balaur/env` if it
does not already exist, and reloads the user systemd manager.

Useful service commands:

```bash
make status-user-service
make logs-user-service
make restart-user-service
make stop-user-service
```

Edit `~/.config/balaur/env` for model paths, remote provider settings, or
optional features. The file is not overwritten by later installs.

To let the user service start at boot before you log in, enable linger once:

```bash
loginctl enable-linger "$USER"
```

Cross-compiles to linux/darwin/windows, amd64/arm64, from any machine —
no C toolchain. The binary is static; the GGUF weights and the llamafile
engine are downloaded data, stored outside the repo and outside the binary.

## CLI for agents & test harnesses

Every command prints one JSON envelope on stdout:

```json
{"v": 1, "kind": "task.add", "data": { … }}
```

`v` is the CLI API version (integer, bumped only on breaking changes to the
envelope or any command's data shape; additive fields inside `data` are
free). `kind` is `<command>.<subcommand>` — a discriminator for consumers.
`data` is exactly the value each command returns. Failures print an error
envelope `{"v":1,"kind":"error","data":{"error":"…"}}` on stderr and exit
non-zero. Web and CLI are gateways over the same turn pipeline
(`internal/turn`), so what the CLI observes is evidence about what the web
UI does.

Before driving a box with scripts, run `balaur doctor` — a no-model
preflight that checks the data dir, core collections, model readiness,
gates, and extensions. Exit code 0 means all fatal checks pass; the
top-level `ok` field is the AND of fatal checks only (model not configured
is non-fatal).

| Command | What it does | Model? |
|---|---|---|
| `balaur doctor` | Preflight: data dir writable, core collections present, model readiness, OS-access gate, extensions. Exit 0 if box is operable. | no |
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
internal/llm/      one model seam: OpenAI-compatible HTTP (local + remote)
internal/llama/    llamafile subprocess supervisor serving a local GGUF
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
- Embedding recall (FTS5 lexical recall shipped; `Embed()` seam reserved
  for a second ranking stage behind the same SearchActive call)
- Sub-head merge-back and scoped head tools (branch chat shipped;
  merge and grant-scoped tools are the next slices)
- Encrypted export
- Multi-human accounts (the schema allows it; v1 serves one owner)

## License

AGPL-3.0-or-later. Built in the open in Brașov.
