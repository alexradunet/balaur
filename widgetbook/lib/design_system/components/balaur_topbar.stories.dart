import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_topbar.stories.g.dart';

const component = ComponentMeta(
  name: 'Topbar',
  path: 'Design system/Navigation',
);
const meta = Meta(BalaurTopbar.new);

final $ChatActive = _Story(
  args: _Args.fixed(
    brand: 'Balaur',
    links: ['chat', 'tasks', 'memory', 'life', 'profile'],
    active: 'chat',
    onNavigate: _navigate,
    onToggleTheme: _toggle,
    themeGlyph: '◑',
  ),
);

void _navigate(String _) {}
void _toggle() {}
