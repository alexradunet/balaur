import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_nudge_banner.stories.g.dart';

const component = ComponentMeta(
  name: 'Nudge banner',
  path: 'Design system/Domain',
);
const meta = Meta(BalaurNudgeBanner.new);

final $Evening = _Story(
  args: _Args(
    message: StringArg(
      'The evening comes, and the tomatoes thirst. Will you tend them now?',
    ),
    when: NullableStringArg('18:00'),
    onDone: Arg.fixed(() {}),
    onSnooze: Arg.fixed(() {}),
    onTomorrow: Arg.fixed(() {}),
  ),
);
