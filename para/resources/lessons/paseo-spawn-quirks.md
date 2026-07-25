# Lesson: Paseo spawn output quirks

From orchestrating project `001-directory-vault` (Balaur, 2026-07). Three
trips, all mechanical:

1. **`paseo run --json` (and `--background`) prints an info line first.**
   Output begins with `Using workspace wks_…` before the JSON object, and the
   JSON is pretty-printed across multiple lines. Piping straight into
   `json.load(sys.stdin)` or `grep '^{.*}$'` fails. Reliable form:
   `paseo agent ls | head -3` right after the spawn to read the new agent ID.

2. **Foreground `paseo run --wait-timeout` can return before the structured
   result lands** — the call comes back with only the `Using workspace …` line
   while the worker keeps running. Recover with `paseo agent ls` (find it,
   status `running`) and `paseo wait <id> --timeout 25m`, which ends with the
   worker's summary; the artifact is on disk regardless.

3. **`paseo send` to a `closed` agent resumes it.** A finished implement worker
   can take a review fix list without spawning a replacement — the session
   context survives. Use this for post-review nits.
