import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_empty_state.stories.g.dart';

const component = ComponentMeta(
  name: 'Empty state',
  path: 'Design system/Feedback',
);
const meta = Meta(BalaurEmptyState.new);

final $Memory = _Story(
  args: _Args(
    title: StringArg('Nothing kept yet.'),
    message: NullableStringArg(
      'Tell Balaur what matters and it will remember.',
    ),
    actionLabel: NullableStringArg('Open the conversation'),
    onAction: Arg.fixed(() {}),
  ),
);

final $Quiet = _Story(
  args: _Args(
    title: StringArg('quiet'),
    message: NullableStringArg('No date yet — they wait without nagging.'),
    showCrest: BoolArg(false),
  ),
);
