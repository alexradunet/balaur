import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_badge.stories.g.dart';

const component = ComponentMeta(name: 'Badge', path: 'Design system/Core');
const meta = Meta(BalaurBadge.new);

final $Count = _Story(
  args: _Args(
    label: NullableStringArg('3'),
    tone: EnumArg(BalaurBadgeTone.gold, values: BalaurBadgeTone.values),
  ),
);

final $Urgent = _Story(
  args: _Args(
    label: NullableStringArg('!'),
    tone: EnumArg(BalaurBadgeTone.ember, values: BalaurBadgeTone.values),
  ),
);

final $Dot = _Story(args: _Args(dot: BoolArg(true)));
