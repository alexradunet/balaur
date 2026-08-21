import 'package:balaur/navigation/presentation/balaur_navigation_shell.dart';
import 'package:flutter/material.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_navigation_shell.stories.g.dart';

const component = ComponentMeta(
  name: 'Application navigation',
  path: 'Navigation',
);
const meta = Meta(BalaurNavigationShell.new);

final $Chat = _Story(
  args: _Args.fixed(
    active: BalaurDestination.chat,
    onSelect: _select,
    onToggleTheme: _toggleTheme,
    themeGlyph: '◑',
    child: const Center(child: Text('Conversation')),
  ),
);

final $Household = _Story(
  args: _Args.fixed(
    active: BalaurDestination.household,
    onSelect: _select,
    onToggleTheme: _toggleTheme,
    themeGlyph: '◑',
    child: const Center(child: Text('Household account')),
  ),
);

final $Settings = _Story(
  args: _Args.fixed(
    active: BalaurDestination.settings,
    onSelect: _select,
    onToggleTheme: _toggleTheme,
    themeGlyph: '◑',
    child: const Center(child: Text('Model provider settings')),
  ),
);

void _select(BalaurDestination _) {}
void _toggleTheme() {}
