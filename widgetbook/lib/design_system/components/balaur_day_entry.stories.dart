import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_day_entry.stories.g.dart';

const component = ComponentMeta(
  name: 'Day entry',
  path: 'Design system/Data display',
);
const meta = Meta(BalaurDayEntry.new);

final $Gold = _Story(
  args: _Args(
    time: StringArg('07:30'),
    title: StringArg('Fed the hens'),
    detail: NullableStringArg('daily · streak 12'),
    tone: EnumArg(BalaurDayEntryTone.gold, values: BalaurDayEntryTone.values),
  ),
);

final $Due = _Story(
  args: _Args(
    time: StringArg('18:00'),
    title: StringArg('Water the tomatoes'),
    detail: NullableStringArg('every 2 days · due'),
    tone: EnumArg(BalaurDayEntryTone.ember, values: BalaurDayEntryTone.values),
    last: BoolArg(true),
  ),
);
