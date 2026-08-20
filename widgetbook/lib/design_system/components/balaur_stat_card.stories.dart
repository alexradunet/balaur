import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_stat_card.stories.g.dart';

const component = ComponentMeta(
  name: 'Stat card',
  path: 'Design system/Data display',
);
const meta = Meta(BalaurStatCard.new);

final $Weight = _Story(
  args: _Args.fixed(
    label: 'Weight',
    value: '81.2',
    unit: 'kg',
    delta: '0.6 this week',
    deltaTone: BalaurStatDeltaTone.down,
    values: [83, 82.6, 82.1, 82.4, 81.9, 81.6, 81.2],
    iconName: 'gem',
  ),
);

final $Steps = _Story(
  args: _Args.fixed(
    label: 'Steps',
    value: '8,210',
    delta: '12% vs avg',
    deltaTone: BalaurStatDeltaTone.up,
    values: [6100, 7200, 5400, 8900, 7600, 9100, 8210],
    iconName: 'flame',
  ),
);
