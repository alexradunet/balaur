import 'package:balaur/design_system/design_system.dart';
import 'package:widgetbook/widgetbook.dart';

import 'components.g.dart';

final _lightTheme = BalaurTheme.light();
final _darkTheme = BalaurTheme.dark();

final config = Config(
  components: components,
  addons: [
    ViewportAddon(Viewports.all),
    TextScaleAddon(),
    MaterialThemeAddon({'Light': _lightTheme, 'Dark': _darkTheme}),
    GridAddon(8),
    AlignmentAddon(),
    SemanticsAddon(),
  ],
  scenarioConfig: ScenarioConfig(
    definitions: [
      ScenarioDefinition(
        name: 'Light',
        modes: [MaterialThemeMode('Light', _lightTheme)],
      ),
      ScenarioDefinition(
        name: 'Dark',
        modes: [MaterialThemeMode('Dark', _darkTheme)],
      ),
      ScenarioDefinition(name: 'Large text', modes: [TextScaleMode(2)]),
    ],
  ),
);
