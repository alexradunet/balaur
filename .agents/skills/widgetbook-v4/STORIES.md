# Widgetbook Stories

Use the typed Story interface from Widgetbook `4.0.0-beta.11`.

## File shape

Mirror the production path under `widgetbook/lib/`.
Use one `*.stories.dart` library for one widget.
Each `Meta` in the library must target the same widget.

```dart
import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_message_bubble.stories.g.dart';

const component = ComponentMeta(
  name: 'Message bubble',
  path: 'Design system/Chat',
);

const meta = Meta(BalaurMessageBubble.new);
```

The Flutter import supplies `Key` to the generated part.
Keep it even when the handwritten Story does not use `Key`.

`Meta` uses a constructor tear-off and must be constant.
Declare one `Meta` for each named or factory constructor.
Use one optional `ComponentMeta` for a custom name, path, or documentation.

Start each Story variable with `$`:

```dart
final $Agent = _Story(
  name: 'Agent',
  args: _Args(
    content: StringArg('I added milk to the Shopping List.'),
    role: EnumArg(
      BalaurMessageBubbleRole.agent,
      values: BalaurMessageBubbleRole.values,
    ),
  ),
);
```

The generator removes `$` from the displayed name.
Set `name` when the generated name is not sufficient.

## Args

Use `_Args(...)` for interactive controls.
Wrap each value in its matching `Arg` type.
Use `Arg.fixed(...)` for callbacks and fixed objects.

Use `_Args.fixed(...)` for fixed Scenario values.
Pass raw values to this constructor.
Do not put `Arg` wrappers inside `_Args.fixed(...)`.

Beta.11 requires `values` for enum controls:

```dart
EnumArg(
  BalaurMessageBubbleStatus.complete,
  values: BalaurMessageBubbleStatus.values,
)
```

Use `argsType` only when the widget constructor is a poor control interface.
Typical causes are controllers, providers, and several callbacks.
Keep ordinary widget-derived Args when they are sufficient.

## States

Create one `$Story` for each meaningful state that developers must browse.
Examples include default, loading, empty, error, disabled, and overflow states.
Do not hide a browsable state only inside `scenarios`.
Scenarios run during tests and do not become navigation entries.

A Story with no local Scenarios gets an implicit `Default` Scenario.
Global `ScenarioDefinition` entries can replace that implicit capture.
Read [TESTING.md](TESTING.md) before you add global definitions.

## Story-first exemplar

Use this file as the Balaur convention:

`widgetbook/lib/design_system/components/`
`balaur_message_bubble.stories.dart`

It shows:

- A mirrored production path.
- `ComponentMeta` and `Meta`.
- Interactive strings and enums.
- Separate states for browsing.
- Long content and streaming states.

## Incompatible Widgetbook 3 forms

Do not use these forms:

- `@App` or `@UseCase`.
- `context.knobs.*`.
- `main.directories.g.dart`.
- `Widgetbook.material(directories: ...)`.
- `Meta<Widget>()`.

## Sources

- [Tagged Story overview][overview]
- [Tagged code-generation guide][generation]
- [Tagged custom-card Story][example]
- [Tagged Args guide][args]

[overview]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/stories/overview.mdx
[generation]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/stories/code-generation.mdx
[example]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/examples/full_example/lib/components/custom_card.stories.dart
[args]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/args/overview.mdx
