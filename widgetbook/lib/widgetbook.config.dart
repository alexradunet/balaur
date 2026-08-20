import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/material.dart';
import 'package:widgetbook/widgetbook.dart';

import 'components.g.dart';

final _lightTheme = BalaurTheme.light();
final _darkTheme = BalaurTheme.dark();

Widget _appBuilder(BuildContext context, Widget child) {
  return MaterialApp(
    debugShowCheckedModeBanner: false,
    home: Material(
      child: Center(child: SizedBox(width: 1080, height: 800, child: child)),
    ),
  );
}

final config = Config(
  appBuilder: _appBuilder,
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
      ScenarioDefinition(
        name: 'Compact',
        modes: [ViewportMode(AndroidViewports.onePlus8Pro)],
      ),
      ScenarioDefinition(
        name: 'Wide',
        modes: [ViewportMode(LinuxViewports.desktop)],
      ),
    ],
  ),
);
