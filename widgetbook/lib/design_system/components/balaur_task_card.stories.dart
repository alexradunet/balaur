import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_task_card.stories.g.dart';

const component = ComponentMeta(
  name: 'Task card',
  path: 'Design system/Knowledge',
);
const meta = Meta(BalaurTaskCard.new);

final $OpenRecurring = _Story(
  args: _Args(
    title: StringArg('Water the tomatoes'),
    status: EnumArg(BalaurTaskStatus.open, values: BalaurTaskStatus.values),
    dueLine: NullableStringArg('due today 18:00'),
    recurrence: NullableStringArg('every 2 days'),
    onDone: Arg.fixed(() {}),
    onSnooze: Arg.fixed(() {}),
    onDrop: Arg.fixed(() {}),
  ),
);

final $Overdue = _Story(
  args: _Args(
    title: StringArg('Mend the deer fence'),
    dueLine: NullableStringArg('overdue · yesterday'),
    overdue: BoolArg(true),
    onDone: Arg.fixed(() {}),
    onSnooze: Arg.fixed(() {}),
    onDrop: Arg.fixed(() {}),
  ),
);

final $Done = _Story(
  args: _Args(
    title: StringArg('Feed the hens'),
    status: EnumArg(BalaurTaskStatus.done, values: BalaurTaskStatus.values),
    recurrence: NullableStringArg('daily 06:30'),
  ),
);
