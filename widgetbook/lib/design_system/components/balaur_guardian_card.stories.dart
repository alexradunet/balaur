import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_guardian_card.stories.g.dart';

const component = ComponentMeta(
  name: 'Guardian card',
  path: 'Design system/Domain',
);
const meta = Meta(BalaurGuardianCard.new);

final $ReadFiles = _Story(
  args: _Args(
    title: StringArg('Read your Documents folder?'),
    detail: NullableStringArg(
      'To find the budget spreadsheet you mentioned. Read-only, only this once.',
    ),
    scope: NullableStringArg('read · ~/Documents · this session'),
    onAllowOnce: Arg.fixed(() {}),
    onAllowAlways: Arg.fixed(() {}),
    onDeny: Arg.fixed(() {}),
  ),
);

final $RunCommand = _Story(
  args: _Args(
    title: StringArg('Run a shell command?'),
    detail: NullableStringArg('To back up the SQLite file to your vault disk.'),
    scope: NullableStringArg('exec · cp hearth.db /vault · once'),
    onAllowOnce: Arg.fixed(() {}),
    onAllowAlways: Arg.fixed(() {}),
    onDeny: Arg.fixed(() {}),
  ),
);
