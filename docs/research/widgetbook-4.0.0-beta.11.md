# Widgetbook 4.0.0-beta.11 research

Date: 2026-08-20

## Summary

Widgetbook 4 beta uses generated Stories, typed Args, and Scenarios.
It does not use the Widgetbook 3 annotation packages.

Use a separate `widgetbook/` Flutter package for Balaur.
Pin Widgetbook and its command-line interface (CLI) to `4.0.0-beta.11`.
This pin keeps the installed interface consistent with the project skills.

Confidence is high for tagged source and versioned pub.dev facts.
Live v4 documents can change after this research date.

## Compatibility

The package requires these minimum versions:

- Dart 3.10.
- Flutter 3.38.

Balaur uses Dart 3.13.1 and Flutter 3.47.1.
These versions satisfy the package requirements.

Use this workspace dependency set:

```yaml
environment:
  sdk: ^3.13.1

dependencies:
  balaur:
    path: ..
  flutter:
    sdk: flutter
  flutter_driver:
    sdk: flutter
  widgetbook: 4.0.0-beta.11

dev_dependencies:
  build_runner: ^2.15.1
  flutter_test:
    sdk: flutter
```

Widgetbook beta.11 includes its generator.
Do not add `widgetbook_annotation` or `widgetbook_generator`.
Those packages support the Widgetbook 3 annotation model.

## Installation

Install the matching CLI:

```bash
dart pub global activate widgetbook_cli 4.0.0-beta.11
```

Create the separate workspace from the application root:

```bash
$HOME/.pub-cache/bin/widgetbook init --package . --output . --empty
```

The explicit path works when `$HOME/.pub-cache/bin` is not in `PATH`.
The `--empty` flag prevents automatic Story creation.
Generated component discovery remains active.

The CLI creates Linux and web targets on this machine.
It also adds Balaur as a local path dependency.
Balaur adds `flutter_driver` for conditional agent-driven checks.
Normal catalog runs keep the driver extension disabled.
Pin the generated Widgetbook dependency after the command completes.

## Workspace structure

```text
balaur/
├── lib/                         # Production widgets and themes
├── pubspec.yaml
└── widgetbook/
    ├── lib/
    │   ├── main.dart
    │   ├── widgetbook.config.dart
    │   ├── components.g.dart
    │   └── design_system/
    │       └── components/
    │           ├── widget.stories.dart
    │           └── widget.stories.g.dart
    ├── test/widgetbook_test.dart
    └── pubspec.yaml
```

Keep production dependencies out of the catalog package.
Mirror each production widget path under `widgetbook/lib/`.
Commit generated Story files with their source files in Balaur.

## Application configuration

Beta.11 uses `runWidgetbook(config)`.
It does not require a custom `WidgetbookApp` class.

```dart
void main() {
  runWidgetbook(config);
}
```

Use one `Config` for the catalog and Scenario tests.
Use the generated `components` list as the component registry.

Recommended Balaur addons are:

1. `ViewportAddon` for phone, tablet, and desktop checks.
2. `TextScaleAddon` for large-text checks.
3. `MaterialThemeAddon` for the production light and dark themes.
4. `GridAddon` for spacing checks.
5. `AlignmentAddon` for isolated placement.
6. `SemanticsAddon` for semantics inspection.

Addon order changes the wrapper order.
Put `ViewportAddon` first and `SemanticsAddon` last.
Put `GridAddon` before `AlignmentAddon`.

The beta.11 CLI template uses `AlignmentAddon`.
A tagged example also contains the older `AlignAddon` alias.
Prefer the CLI template name.

## Stories and generated code

Widgetbook 4 replaces v3 use cases with Stories.
A Story library uses a constructor tear-off:

```dart
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'action_button.stories.g.dart';

const meta = Meta(ActionButton.new);
```

Keep the Flutter import in the Story library.
The generated part needs the `Key` type.

A Story variable starts with `$`:

```dart
final $Default = _Story(
  args: _Args(
    label: StringArg('Continue'),
    onPressed: Arg.fixed(() {}),
  ),
);
```

The generator creates these declarations:

- `_Story` for the generated Story type.
- `_Args` for typed interactive Args.
- `_Scenario` for fixed test states.
- `_Defaults` for shared setup and builders.
- `components.g.dart` for the component registry.

Use one `Meta` for each constructor variant.
Use one optional `ComponentMeta` for catalog metadata.

Do not use these Widgetbook 3 forms:

- `@App` or `@UseCase`.
- `context.knobs.*`.
- `main.directories.g.dart`.
- `Widgetbook.material(directories: ...)`.
- `Meta<Widget>()`.

## States and Scenarios

Create one `$Story` for each state that developers must browse.
Typical states are default, loading, empty, failed, disabled, and overflow.

Use `_Args(...)` for interactive controls.
Use `Arg.fixed(...)` for callbacks and fixed objects.
Use `_Args.fixed(...)` for repeatable Scenario values.

A Scenario can add fixed Args, Modes, interactions, and assertions.
Scenarios run under `flutter test`.
They do not become catalog navigation entries.

A Story without local Scenarios gets an implicit `Default` Scenario.
Global `ScenarioDefinition` entries can add theme and text-scale coverage.
The default strategy crosses each definition with each local Scenario.
Control the matrix size before you add more global definitions.

## Testing and accessibility

Use this test entry point:

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
flutter analyze
flutter test
flutter build linux --debug
flutter build web
```

Build Runner 2.15.1 removed the `-d` option.
Use the commands without that option.

Tests write Portable Network Graphics (PNG) and JSON artifacts to
`widgetbook/build/.widgetbook/`.
The JSON files include semantics and accessibility violations.

The default guideline set checks these items:

- Android tap targets.
- iOS tap targets.
- Labels on tappable semantics nodes.
- Text contrast.

A recorded violation does not automatically fail `flutter test`.
Inspect the JSON artifacts after applicable changes.
Add explicit assertions when continuous integration must reject violations.

## Balaur integration implications

The smallest useful design-system seam has these parts:

- One production theme module.
- Public reusable widgets with small constructor interfaces.
- Mirrored Widgetbook Story files.
- One generated component registry.
- Scenario coverage for light, dark, and large text.

Develop each reusable widget in Widgetbook before application integration.
Use fake data and adapters for screens.
Never put a live provider key in a Story.

## Risks and deferred work

The beta interface can change in later releases.
Widgetbook 3 search results can produce incompatible code.
Global Scenario definitions can create a large capture matrix.

Widgetbook Cloud and its continuous integration secret remain deferred.
A future upgrade must update both dependencies and project skills.

## Primary sources

- [Widgetbook 4.0.0-beta.11 package][package]
- [Tagged package pubspec][pubspec]
- [Widgetbook CLI 4.0.0-beta.11][cli]
- [Tagged quick start][quick-start]
- [Tagged code-generation guide][generation]
- [Tagged addon guide][addons]
- [Tagged Scenario test guide][testing]
- [Tagged test runner][runner]
- [Tagged accessibility guidelines][guidelines]

[package]: https://pub.dev/packages/widgetbook/versions/4.0.0-beta.11
[pubspec]: https://raw.githubusercontent.com/widgetbook/widgetbook/widgetbook-v4.0.0-beta.11/packages/widgetbook/pubspec.yaml
[cli]: https://pub.dev/packages/widgetbook_cli/versions/4.0.0-beta.11
[quick-start]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/quick-start.mdx
[generation]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/stories/code-generation.mdx
[addons]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/addons/overview.mdx
[testing]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/testing/run-scenarios.mdx
[runner]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/packages/widgetbook/lib/src/test/test.dart
[guidelines]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/packages/widgetbook/lib/src/core/accessibility/guidelines.dart
