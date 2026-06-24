# Balaur

> **A sovereign local-first personal agent, served from one binary.**

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/go-%E2%89%A51.26-00ADD8.svg)](./go.mod)

Balaur is a personal AI companion that lives on a box you own: a single Go
executable embedding [PocketBase](https://pocketbase.io) for data, auth and
migrations, a Datastar web interface, and local LLM inference run in-process via
the embedded [Kronk](https://github.com/ardanlabs/kronk) engine (llama.cpp, CGO-free).

The name comes from the Romanian fairy-tale balaur: a dragon with multiple
heads. A head is a switchable persona — a name, purpose, avatar, and an
optional tool-group filter. The active head flavors the one shared life
conversation; switch heads in the dock to give the same master turn a
different voice and a narrower (or wider) set of tools.

Your life is not a product. The record of your life should live in a
database you own and can open with any SQLite tool.

## Current shape

- **One binary:** `balaur` — web UI, database, migrations, agent loop.
- **Data:** PocketBase collections — `conversations`, `messages`, `nodes`
  (the unified knowledge spine: memories, skills, notes, and typed objects),
  `edges`, `heads`, `audit_log` — in plain SQLite under `pb_data/`.
- **UI:** server-rendered typed `gomponents` over Datastar (SSE hypermedia),
  styled by the Hearthwood/Basm design system (see `DESIGN.md`). The PocketBase
  dashboard at `/_/` stays the superuser engine room.
- **Models:** Balaur runs local GGUF models in-process. Install one from the
  settings models section (an absolute `.gguf` path);
  it runs on CPU by default, or set `BALAUR_PROCESSOR=vulkan` to offload to a
  Vulkan GPU. Local is the default and stays first-class; an EU/GDPR-compliant
  cloud model (Mistral today — the curated picker is EU-only for AI sovereignty)
  can be added opt-in from the Models page, consent-gated — never the default,
  and a turn only leaves the box on your explicit, confirmed selection.
- **Heads:** switchable personas. A head is a name + purpose + avatar +
  optional capability-group tool filter. Built-in `balaur`, `scholar`,
  `planner`, and `coach` ship out of the box; create your own from the heads
  card. The active head is chosen in the dock switcher and applied to the one
  shared conversation — full trust, no data scoping. `internal/heads` owns
  the roster.
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
- **The quests card — life organization:** the operational list (cards
  with Done / Snooze / Drop) opens as the quests card in the right panel
  (`/ui/show/quests`); a month calendar and a 14-day timeline are their own
  cards — the forward mirror of the recap telescope. Calendar and timeline
  project recurrence rules forward, read-only; actions live on the task cards.
- **The life log — owner-defined:** Balaur does not decide what a life is
  made of. `log_entry` keeps whatever you track under kinds you invent
  (weight, mood, sleep, pages-read…), numeric or textual, backdatable;
  `entry_series` reads trends, `entry_drop` corrects. The lifelog card
  (`/ui/show/lifelog`) mirrors what exists: sparklines for numeric kinds,
  recent lines for the rest, live habit streaks on top. Nothing is
  predefined; the briefing reflects
  yesterday's log in one line.
- **The day card — where a day lives:** the `day` card
  (`/ui/show/day?date={date}`) assembles your journal (written in chat via
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

`make run` serves your **real** instance — the personal data dir at
`~/.local/share/balaur/pb_data` — on port **8080** (bound `0.0.0.0` so it's
reachable over a NetBird mesh; loopback at http://127.0.0.1:8080/, superuser at
`/_/`). There is no daemon; for an always-on box, run it inside a long-lived
[zellij](https://zellij.dev) session so it survives SSH logout (`make dev`
likewise).

For live-reload development, run `make dev` (uses [air](https://github.com/air-verse/air);
if `air` is not installed it downloads and runs the latest release
automatically). It serves a **separate, throwaway** instance from the repo-local
`pb_data/` directory on port **8090** (http://127.0.0.1:8090/), restarting
whenever Go, HTML, CSS, JS, or static asset files change. The two never share
data, so `make run` (prod, 8080) and `make dev` (dev, 8090) can run side by side.

To run the binary directly with custom flags:

```bash
go run . serve --http 127.0.0.1:8080 --dir ~/.local/share/balaur/pb_data
```

**Local inference**: Balaur runs GGUF models in-process via the embedded Kronk
engine (`internal/kronk`) — yzma `dlopen`s the prebuilt llama.cpp library at
runtime, so the Go build stays CGO-free. Two runtime assets are owner-supplied
(the engine never downloads them on boot):

- the native llama.cpp library — point `BALAUR_LIB_PATH` at its directory
  (CPU by default; `BALAUR_PROCESSOR=vulkan` selects the Vulkan variant)
- a GGUF model file — install it from the settings models section
  (`/ui/show/settings?section=models`).
  That page can also fetch Balaur's official curated model in one click
  (owner-initiated download into `BALAUR_MODELS_DIR`; plan 086)

```bash
# Run on a Vulkan GPU (install the GGUF from the Models page, not an env var):
BALAUR_LIB_PATH=~/.local/share/balaur/kronk/lib \
BALAUR_PROCESSOR=vulkan go run . serve
```

Vulkan needs the host Vulkan loader + GPU driver/ICD (e.g. `mesa-vulkan-drivers`)
— host setup, outside the repo.

**Extension engine**: Balaur uses goja (no tags; pins a master commit) for the JavaScript sandbox. Bumping it is a deliberate act—run `go test ./internal/ext/` after changing.

Secrets (OAuth tokens, vault entries) live in the local PocketBase data
directory and its backups. Treat `pb_data/` as secret.

Optional environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `BALAUR_ALLOWED_HOSTS` | (unset) | Comma-separated `host[:port]` values allowed as the Host header beyond loopback (LAN names, NetBird — see [docs/netbird.md](docs/netbird.md)) |
| `BALAUR_LIB_PATH` | XDG `~/.local/share/balaur/kronk/lib` | Directory holding the prebuilt llama.cpp library (yzma dlopens it) |
| `BALAUR_PROCESSOR` | `cpu` | llama.cpp variant to load — `cpu` or `vulkan` |
| `BALAUR_MODELS_DIR` | `~/.local/share/balaur/models` | Directory where in-app model downloads are saved |
| `BALAUR_HF_TOKEN` | (unset) | Optional Hugging Face token for downloading gated models |
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

There is no daemon or systemd unit — Balaur runs in the foreground. To keep your
real instance always on, run `make run` inside a long-lived
[zellij](https://zellij.dev) session and detach:

```bash
zellij              # or: zellij attach -c balaur
make run            # prod: ~/.local/share/balaur/pb_data on :8080
# Ctrl-o d          # detach; the session (and Balaur) keeps running
```

The session survives SSH logout. To also keep it alive across reboots, enable
logind linger once:

```bash
loginctl enable-linger "$USER"
```

Set model paths or optional features as environment variables in front of
`make run` (or export them in the session). Cross-compiles to linux/darwin/windows, amd64/arm64, from any machine —
no C toolchain. The Go binary is CGO-free; the native llama.cpp library and GGUF
model weights are runtime assets, dlopen'd/read at runtime from outside the binary.

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
| `balaur note add/list/show/drop` | Owner-authored notes as `type=note` nodes. | no |
| `balaur search <terms>` | Cross-type FTS5 recall over approved knowledge nodes. | no |
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

# Register the fake server as a cloud model (Models page → add an
# OpenAI-compatible model with base URL http://127.0.0.1:11435/v1 and
# model ID "fake"), or seed the provider/model/settings rows directly as
# .github/workflows/ci.yml does in its harness job.
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
internal/llm/      one model seam: OpenAI-compatible HTTP client (local)
internal/kronk/    embedded inference engine: in-process GGUF via the Kronk SDK (CPU/Vulkan)
internal/turn/     the channel-agnostic turn pipeline + model resolution
internal/conversation/ master conversation: persistence + context window
internal/recap/    the telescope: period math + hierarchical summaries
internal/tasks/    commitments: recurrence DSL + task verbs on the entries life log
internal/verify/   runtime honesty check: words audited against tool deeds
internal/heads/    switchable personas: built-ins + custom CRUD, active head, tool-group filter
internal/knowledge/ memory & skill lifecycle, context injection — the consent boundary
internal/store/    shared PocketBase helpers (audit)
internal/tools/    agent tools: knowledge (always) + OS access (opt-in)
internal/self/     self-awareness: embedded self-knowledge + live inventory
internal/ext/      balaur-extensions: consent-gated runtime tools (JS/goja)
internal/web/      Datastar gateway: dock chat, cards & panels, recap
internal/web/assets/ embedded static assets (Basm CSS, JS, icons, fonts, avatars)
internal/cli/      machine-facing gateway: balaur subcommands, JSON out
```

Read `AGENTS.md` for the engineering rules (KISS, YAGNI, suckless, the rule
boundary) and `DESIGN.md` for the Basm design system.

## Roadmap (not shipped — honesty ledger)

- Johnny Decimal Markdown vault mirror: one-way export + git history
- Embedding recall (FTS5 lexical recall shipped; `Embed()` seam reserved
  for a second ranking stage behind the same SearchActive call)
- Encrypted export
- Multi-human accounts (the schema allows it; v1 serves one owner)

## License

AGPL-3.0-or-later. Built in the open in Brașov.
