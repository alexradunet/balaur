# Model Registry

Manual registry of available models, their tiers, and recommended use cases.
Updated by the user based on benchmarks, budget, and quota.

The orchestrator reads this file to recommend models for each task phase.

## Complexity rubric

| Tier | When to use | Examples |
|------|-------------|----------|
| premium | Deep reasoning, architecture, adversarial grilling, complex multi-file plans | Grill on a new system, plan a migration, audit security |
| mid | Solid implementation, standard planning, code review | Implement a well-specified plan, review a PR, explore a bug |
| budget | Fast execution of literal steps, simple fixes, formatting | Execute a plan step-by-step, fix a typo, run a checklist |

## Active models

Provider `pi` is the only enabled provider (2026-07-24; claude, codex, copilot, opencode, omp disabled).
The Model column holds the `paseo run --provider` value. **Tier, Strength, and Status are user-edited**: Paseo cannot know quota or tier. Fill these in.

| Model (paseo --provider value) | Tier | Strength | Status |
|-------|------|----------|--------|
| pi/openai-codex/gpt-5.3-codex-spark | TBD | TBD | TBD |
| pi/openai-codex/gpt-5.4 | TBD | TBD | TBD |
| pi/openai-codex/gpt-5.4-mini | TBD | TBD | TBD |
| pi/openai-codex/gpt-5.5 | TBD | TBD | TBD |
| pi/openai-codex/gpt-5.6-luna | TBD | TBD | TBD |
| pi/openai-codex/gpt-5.6-sol | TBD | TBD | TBD |
| pi/openai-codex/gpt-5.6-terra | TBD | TBD | TBD |
| pi/qwen-token-plan/MiniMax-M2.5 | TBD | TBD | TBD |
| pi/qwen-token-plan/deepseek-v3.2 | TBD | TBD | TBD |
| pi/qwen-token-plan/deepseek-v4-flash | TBD | TBD | TBD |
| pi/qwen-token-plan/deepseek-v4-pro | TBD | TBD | TBD |
| pi/qwen-token-plan/glm-5 | TBD | TBD | TBD |
| pi/qwen-token-plan/glm-5.1 | TBD | TBD | TBD |
| pi/qwen-token-plan/glm-5.2 | TBD | TBD | TBD |
| pi/qwen-token-plan/kimi-k2.5 | TBD | TBD | TBD |
| pi/qwen-token-plan/kimi-k2.6 | TBD | TBD | TBD |
| pi/qwen-token-plan/kimi-k2.7-code | TBD | TBD | TBD |
| pi/qwen-token-plan/qwen3.6-flash | TBD | TBD | TBD |
| pi/qwen-token-plan/qwen3.6-plus | TBD | TBD | TBD |
| pi/qwen-token-plan/qwen3.7-max | TBD | TBD | TBD |
| pi/qwen-token-plan/qwen3.7-plus | TBD | TBD | TBD |
| pi/qwen-token-plan/qwen3.8-max-preview | TBD | TBD | TBD |

## Phase defaults

| Phase | Default tier | Fallback |
|-------|-------------|----------|
| grill | premium | mid |
| spec | premium | mid |
| domain-model (bootstrap) | mid | budget |
| plan (feature) | premium | mid |
| plan (fix) | mid | budget |
| plan (audit) | premium | mid |
| tickets | premium | mid |
| implement | mid | budget |
| review-standards | mid | budget |
| review-feature | premium | mid |
| domain-model (reconcile / close) | mid | budget |
| teach | mid | premium (for deep topics) |
| janitor | budget | mid |

Spawned phases (delegated via `paseo run`; model comes from the role briefing's `provider` field): spec, domain-model, plan, tickets, implement, review-standards, review-feature. In-session phases (run on the orchestrator's own model): grill, teach, janitor. `integrate` is a ticket (or an orchestrator merge), not a spawned phase. `explore` is ad-hoc, outside the pipeline.

## Notes

- Update this file when quota changes, new models become available, or benchmarks shift.
- The orchestrator recommends; the user always confirms before spawn.
- If a model's status is `exhausted` or `removed`, the orchestrator falls back to the next model in the same tier.
- Multi-model review panels should use models from different providers when possible (reduces correlated blind spots).
- A bundled role briefing's `provider` field is the floor for that phase, not the recommendation: `roles/review.md` ships at premium tier even though the phase-defaults table lists mid. The registry governs what the orchestrator recommends; the role briefing governs what runs when nobody overrides.
