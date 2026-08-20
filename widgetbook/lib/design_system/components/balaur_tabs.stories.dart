import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_tabs.stories.g.dart';

const component = ComponentMeta(name: 'Tabs', path: 'Design system/Navigation');
const meta = Meta(BalaurTabs.new);

final $TaskViews = _Story(
  args: _Args.fixed(
    items: ['list', 'calendar', 'timeline'],
    active: 'list',
    onSelect: _select,
  ),
);

void _select(String _) {}
