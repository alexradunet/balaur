import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_sparkline.stories.g.dart';

const component = ComponentMeta(
  name: 'Sparkline',
  path: 'Design system/Data display',
);
const meta = Meta(BalaurSparkline.new);

final $Trend = _Story(
  args: _Args.fixed(
    values: [62, 64, 61, 67, 70, 66, 72, 75, 73, 78],
    width: 200,
    height: 48,
  ),
);
