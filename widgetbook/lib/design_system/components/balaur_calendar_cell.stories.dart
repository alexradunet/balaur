import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_calendar_cell.stories.g.dart';

const component = ComponentMeta(
  name: 'Calendar cell',
  path: 'Design system/Data display',
);
const meta = Meta(BalaurCalendarCell.new);

final $Default = _Story(
  args: _Args(day: IntArg(8), pips: IntArg(1)),
);
final $Today = _Story(
  args: _Args(day: IntArg(14), pips: IntArg(2), today: BoolArg(true)),
);
final $Selected = _Story(
  args: _Args(day: IntArg(14), pips: IntArg(2), selected: BoolArg(true)),
);
final $OtherMonth = _Story(
  args: _Args(day: IntArg(30), dimmed: BoolArg(true)),
);
