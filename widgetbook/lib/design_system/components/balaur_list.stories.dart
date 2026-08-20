import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_list.stories.g.dart';

const component = ComponentMeta(
  name: 'List',
  path: 'Design system/Data display',
);
const meta = Meta(BalaurList.new);

final $Today = _Story(
  args: _Args.fixed(
    title: 'Today',
    items: [
      BalaurListItem(
        iconName: 'quill',
        title: 'Water the tomatoes',
        subtitle: 'every 2 days · garden',
        meta: '18:00',
      ),
      BalaurListItem(
        iconName: 'shield',
        title: 'Mend the deer fence',
        subtitle: 'before the weekend',
        meta: 'overdue',
        metaTone: BalaurListMetaTone.warning,
      ),
      BalaurListItem(
        iconName: 'tome',
        title: 'Export this week’s notes',
        subtitle: 'as Markdown',
        meta: 'done',
      ),
    ],
  ),
);

final $NoHeader = _Story(
  args: _Args.fixed(
    items: [
      BalaurListItem(title: 'Water the tomatoes', meta: '18:00'),
      BalaurListItem(title: 'Feed the hens', meta: 'done'),
    ],
  ),
);
