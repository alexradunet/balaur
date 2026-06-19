# Balaur — product brief

This is Balaur's product north star: who it's for, why it exists, what it
promises, what it refuses to be, and where it's headed. It is written for
**contributors and agents** — the document you judge a new feature against
before building it. It deliberately does *not* re-explain mechanics: README.md
owns the current shape and how to run it, AGENTS.md owns the engineering rules,
DESIGN.md (Basm) owns identity and voice, `internal/self/knowledge.md` is the
running binary's self-description. When this brief and the code disagree about
*what is*, the code wins; when they disagree about *what should be*, this brief
is the appeal.

> **The promise:** *Your personal wise companion that grows with you — on a box
> you own.*

## What Balaur is

Balaur is a sovereign, local-first personal AI companion shipped as one Go
binary. It runs on a box the owner controls, keeps the record of a life in plain
SQLite the owner can open with any tool, and runs a local model in-process so
the conversation never has to leave the box. One master conversation, persisted
forever; switchable heads give it different voices and tool sets; memory and
skills enter its context only by the owner's consent; and every reply is audited
against what it actually did.

## Who it's for

The north-star owner is a **privacy-conscious person who is not a developer** —
someone who wants a capable AI companion for their life and work but is unwilling
to rent their private record to a SaaS, and who should not have to understand
SQLite, systemd, or environment variables to own one.

**This is a stated direction, not the current reality.** Today Balaur asks its
owner to install a service, edit an env file, and supply a model path — work that
fits a technical self-hoster, not the north-star owner. That gap is the product's
central unfinished business: every step that moves setup, model acquisition, and
daily operation toward zero-friction for a non-technical owner is on-mission;
every feature that deepens power-user surface without closing that gap should
justify itself against this brief. The technical self-hoster is the *beachhead*
who can run Balaur today — not the destination.

## The bet

Balaur wagers that there is a person who will run a box for a private companion
they would never trust to the cloud — and that the deciding factor is not raw
capability but **sovereignty plus trust**: the record is theirs, nothing acts
without consent, and the companion cannot lie to them about what it did. The
local model is the price of that wager, not a feature. If hosted assistants win
on capability alone, Balaur still wins the person for whom *whose box it runs on*
is the whole question.

## Product pillars

The non-negotiable promises. These are brand contracts, not implementation
details — a change that erodes one of these is off-mission even if it ships
clean.

- **Sovereignty.** The life lives on the owner's box, in inspectable SQLite and
  exported Markdown — never in a vendor's database. *Your life is not a product.*
- **Consent.** The model proposes; the owner disposes. Memories, skills, and
  extensions enter context only after an explicit approval. Even Balaur editing
  its own code ends at an owner-restarted binary.
- **Honesty.** Words are audited against deeds. A capture claim with no
  successful capture tool gets one repair pass, then an honest note on the
  record. Trust the task card, not the words.
- **One companion that grows.** Balaur does not install plugins or spawn a swarm;
  it grows new heads and reviewable skills, all serving the same companion. Never
  a marketplace, never a multi-agent product.
- **Transparency.** Durable state is in collections and Markdown the owner can
  read; nothing important hides in session memory. *The repo is the system.*

## What success looks like

V1 succeeds when a single owner can live with Balaur for **thirty days** and:

- trust it — they rely on the task card and the briefing, having seen the honesty
  check catch the model when it overstated;
- own it — their conversation, tasks, journal, and life-log are all theirs in
  SQLite, and they have opened the file at least once to prove it;
- grow it — they have approved at least one memory and one skill, and the
  companion is measurably more *theirs* than on day one;
- and never have left the box — no turn required a remote service.

The closing test of the north star, still ahead of us: a non-technical owner gets
from nothing to a working companion **without a terminal**.

## Non-goals

What Balaur refuses to become. Point at this section to kill scope creep.

- **Not a hosted SaaS.** No multi-tenant cloud, no "Balaur account," no servers
  holding owners' data. Self-hosted is the product, not a deployment option.
- **Not a surveillance or total-recall product.** Balaur records what the owner
  chooses to keep, by consent — it does not silently capture screens, audio, or
  ambient activity.
- **Not a multi-agent swarm or sub-agent framework.** One companion, switchable
  heads. No orchestration of autonomous agents acting unsupervised.
- **Not a plugin marketplace.** Extensions add verbs through a consent ledger,
  not a store with discovery, ratings, or third-party trust.
- **Not a cloud-model router.** V1 is local-only on purpose; remote providers are
  not a missing feature to add back, they are a line the product chose not to
  cross in v1.
- **Not an internet-exposed service.** Loopback-first; reaching it remotely is
  the owner's explicit, deliberate act, never a default.

## Decided product tradeoffs

The *why* behind sharp choices. (Engineering rationale lives in the specs under
`docs/superpowers/specs/`; this records the product reason.)

- **Local-only inference in v1** — sovereignty over raw capability. A companion
  that can leak is not the product, even if it is smarter (plan 074).
- **One shared conversation, full trust** — heads are personas, not sandboxed
  agents; there is no per-head data scoping. The simplicity *is* the trust model.
- **Consent over autonomy** — friction at the consent boundary is the feature,
  not a cost to optimize away. A companion that acts without asking is the thing
  Balaur exists to not be.
- **Single human owner in v1** — the schema leaves room for multiple humans
  later, but no code path serves them. One owner keeps the trust model legible.
- **The PocketBase dashboard is the engine room, never the face** — power and
  inspection live at `/_/`; the product surface stays the companion.

## Direction — bets, not commitments

The arc beyond v1, framed as bets with rationale. README.md keeps the honest
shipped/unshipped ledger; this records *why each bet is worth making*. None of
these is a promise.

- **Zero-terminal onboarding.** The biggest bet, straight from the north-star
  persona: model acquisition, runtime install, and first run reachable without a
  shell. Until this lands, the non-technical owner is aspiration, not audience.
- **The Johnny Decimal vault mirror.** One-way export of the life record to
  Markdown + git, so sovereignty is provable and portable, not just claimed.
- **Encrypted export.** Sovereignty includes safe backup; an owner should be able
  to carry their box's record without carrying its risk.
- **Sharper recall.** Embedding recall behind the existing FTS5 seam — the
  companion remembers what matters more precisely as the record grows.
- **More gateways, same pipeline.** A messenger or CLI surface that adapts the
  one shared turn pipeline (`internal/turn`) rather than re-implementing it —
  meeting the owner where they already are without forking behavior.
- **Multi-human, later.** A box a household could share, once the single-owner
  trust model is proven and the schema decisions hold. Explicitly post-v1.

---

*Woven, not rendered. The repo is the system. Your life is not a product.*
