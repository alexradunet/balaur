# Development policy

## Purpose

Use this policy before feature, refactor, architecture, or dependency work.

The Pareto principle prioritizes the small set of changes that creates most
value.
Use 80/20 as a prioritization heuristic, not as a measurement.

## Pareto pass

Complete this pass before implementation.

1. State one user or operational outcome.
2. Identify the smallest coherent change that can produce that outcome.
3. Label each proposed item as `Core` or `Deferred`.
4. Select the fastest reliable validation for the Core work.
5. Stop when the Core work meets its completion criteria.

Keep security, data integrity, accessibility, and required reliability in
`Core`.
Use evidence from user needs, failures, tests, and recent changes to expand
`Core`.
Defer speculative layers, generalized interfaces, and unused dependencies.

Completion criterion: The Core plan delivers one outcome, names its
validation, and lists deferred work.

## Architecture pass

Read `CONTEXT.md` and applicable architecture decision records before you
change a seam.
Read `.agents/skills/codebase-design/SKILL.md` and its `DEEPENING.md`
reference before module design work.
Use these terms exactly: module, interface, implementation, depth, seam,
adapter, leverage, and locality.

Read `.agents/skills/improve-codebase-architecture/SKILL.md` before a broad
architecture review.
Use its hotspot scan, report, and user-selection process.

Apply these checks to each architecture change:

- Prefer a deep module with a small interface and substantial hidden behavior.
- Apply the deletion test to each suspected shallow module.
- Test through the module interface.
- Accept dependencies through constructors or parameters.
- Return results when a result can replace a side effect.
- Add a seam when two adapters justify it.
- Keep internal seams out of the external interface.

## Flutter fit

Read `.agents/skills/flutter-apply-architecture-best-practices/SKILL.md`
before Flutter architecture work.
Use the user interface layer, the data layer, and an optional domain layer.

Apply the Flutter layers only when application needs justify them:

- Use local `State` for local, short-lived presentation state.
- Add a ViewModel when state or commands need isolated tests or shared
  ownership.
- Add a Repository when data has an authoritative source or multiple callers.
- Treat each external integration as an adapter at a seam.
- Add an abstract Repository interface when production and test adapters both
  exist.
- Add a Use Case only for complex or reused business logic.
- Remove classes that only forward arguments and results.

The project term `ViewModel` means a Model-View-ViewModel presentation module.

## Runtime inspection

For agent-driven user interface checks, start the app with this command:

```bash
flutter run -d <device-id> --dart-define=ENABLE_FLUTTER_DRIVER=true
```

Use `dart_dtd` to connect before you call runtime tools.
Keep `ENABLE_FLUTTER_DRIVER` unset for normal and production runs.
Web builds do not support Flutter Driver finder commands.

## Validation

At the first Dart Model Context Protocol (MCP) use in each Pi session, add the
project URI with `dart_roots`.
Use `dart_analyze_files` after each coherent Dart change.
Run the smallest relevant test before the full test suite.
Run the full test suite when a change crosses more than one module interface.
Check narrow and wide layouts after a user interface change.

Completion criterion: The Core outcome works, relevant tests pass, analysis
is clear, and deferred work remains separate.

## References

- [Dart and Flutter MCP server][dart-mcp]
- [Flutter architecture recommendations][flutter-architecture]

[dart-mcp]: https://docs.flutter.dev/ai/mcp-server
[flutter-architecture]:
  https://docs.flutter.dev/app-architecture/recommendations
