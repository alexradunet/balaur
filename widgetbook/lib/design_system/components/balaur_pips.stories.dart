import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_pips.stories.g.dart';

const component = ComponentMeta(name: 'Pips', path: 'Design system/Core');
const meta = Meta(BalaurPips.new);

final $Importance = _Story(
  args: _Args(
    level: IntArg(
      3,
      style: const SliderIntArgStyle(min: 0, max: 5, divisions: 5),
    ),
    max: IntArg(5),
  ),
);
