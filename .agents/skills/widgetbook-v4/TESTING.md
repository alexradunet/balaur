# Widgetbook Scenario testing

Use `testWidgetbook(config)` to render each Story and Scenario in isolation.
The test uses the same generated component list as the running catalog.

## Test entry point

Keep this entry point in `widgetbook/test/widgetbook_test.dart`:

```dart
import 'package:widgetbook/test.dart';
import 'package:widgetbook_workspace/widgetbook.config.dart';

Future<void> main() async {
  await testWidgetbook(config);
}
```

Run these commands from `widgetbook/`:

```bash
dart run build_runner build
dart format lib
flutter test
```

The test writes Portable Network Graphics (PNG) and JSON artifacts to
`widgetbook/build/.widgetbook/`.

## Scenarios

Use `_Scenario` for a fixed state, an interaction, or environment Modes.
Use `_Args.fixed(...)` for repeatable values.

```dart
_Scenario(
  name: 'Long content',
  args: _Args.fixed(
    content: 'A long message',
    role: BalaurMessageBubbleRole.agent,
  ),
)
```

A Story without local Scenarios gets an implicit `Default` Scenario.
Explicit local Scenarios replace that implicit Scenario.
Add an explicit default when you need both forms.

Use `run` for interaction assertions.
The callback receives a `WidgetTester` and the resolved Args.

## Global environment coverage

Use `ScenarioConfig.definitions` for cross-cutting states.
Use it for themes and text scale instead of repeated local Scenarios.

```dart
scenarioConfig: ScenarioConfig(
  definitions: [
    ScenarioDefinition(
      name: 'Dark',
      modes: [MaterialThemeMode('Dark', darkTheme)],
    ),
    ScenarioDefinition(
      name: 'Large text',
      modes: [TextScaleMode(2)],
    ),
  ],
),
```

The default strategy is `ScenarioStrategy.perScenario`.
Each definition crosses each local Scenario.
Control the matrix size before you add more global definitions.

## Accessibility

Beta.11 uses `WidgetbookGuidelines.recommended` by default.
The set checks tap targets, tappable labels, and text contrast.
The runner records violations in each JSON artifact.
A recorded violation does not automatically fail `flutter test`.
Inspect the artifacts when accessibility is part of the change.
Add explicit test assertions when continuous integration must reject a defect.

Use `SemanticsAddon` only for interactive semantics inspection.
Keep it last in the addon list.

## Completion criterion

Finish only when generation is current and all Scenario tests pass.
Inspect runtime errors, overflows, and applicable accessibility violations.

## Sources

- [Tagged testing guide][testing]
- [Tagged default-Scenario guide][default]
- [Tagged test runner][runner]
- [Tagged accessibility guidelines][guidelines]

[testing]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/testing/run-scenarios.mdx
[default]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/testing/default-scenario.mdx
[runner]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/packages/widgetbook/lib/src/test/test.dart
[guidelines]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/packages/widgetbook/lib/src/core/accessibility/guidelines.dart
