# Widgetbook setup

This reference applies to Widgetbook `4.0.0-beta.11`.

## Requirements

Widgetbook requires Dart `>=3.10.0 <4.0.0` and Flutter `>=3.38.0`.
The version includes its generator.
It does not use `widgetbook_annotation` or `widgetbook_generator`.

Use this command to install the matching command-line interface (CLI):

```bash
dart pub global activate widgetbook_cli 4.0.0-beta.11
```

Use this command once from the application package:

```bash
$HOME/.pub-cache/bin/widgetbook init --package . --output . --empty
```

The explicit path works when `$HOME/.pub-cache/bin` is not in `PATH`.
The `--empty` flag stops automatic Story creation for existing widgets.
It does not stop generated component discovery.

Pin the workspace dependency after the CLI creates the package:

```yaml
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

The catalog imports `flutter_driver` for agent-driven runtime checks.
It enables the extension only when `ENABLE_FLUTTER_DRIVER` is true.
Keep this definition unset for normal catalog runs and builds.

## Configuration

Use one `Config` for the catalog and `testWidgetbook`.
Import the generated `components` list.
Do not maintain a second component list for tests.

Use the production theme module in each theme addon.
Keep the addon order shown below.
The first addon is the outermost wrapper.

```dart
final config = Config(
  components: components,
  addons: [
    ViewportAddon(Viewports.all),
    TextScaleAddon(),
    MaterialThemeAddon({
      'Light': BalaurTheme.light(),
      'Dark': BalaurTheme.dark(),
    }),
    GridAddon(8),
    AlignmentAddon(),
    SemanticsAddon(),
  ],
);
```

Apply these order rules:

1. Put `ViewportAddon` before all other addons.
2. Put `GridAddon` before `AlignmentAddon`.
3. Put `SemanticsAddon` after all other addons.

Use `runWidgetbook(config)` in `widgetbook/lib/main.dart`.
Use `materialAppBuilder`, which is the default, for Material widgets.

## Generated files

`components.g.dart` contains the generated component registry.
Each Story library gets one `*.stories.g.dart` part.
Do not edit these files by hand.
Run generation after each Story or constructor change.

## Sources

- [Widgetbook 4.0.0-beta.11 package][package]
- [Tagged v4 quick start][quick-start]
- [Tagged addon order][addons]
- [Tagged CLI template][config-template]
- [Balaur research brief][research]

[package]: https://pub.dev/packages/widgetbook/versions/4.0.0-beta.11
[quick-start]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/quick-start.mdx
[addons]: https://github.com/widgetbook/widgetbook/blob/widgetbook-v4.0.0-beta.11/docs/addons/overview.mdx
[config-template]: https://github.com/widgetbook/widgetbook/blob/widgetbook_cli-v4.0.0-beta.11/packages/widgetbook_cli/lib/src/templates/config_template.dart
[research]: ../../../docs/research/widgetbook-4.0.0-beta.11.md
