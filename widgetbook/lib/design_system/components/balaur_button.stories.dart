import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_button.stories.g.dart';

const component = ComponentMeta(name: 'Button', path: 'Design system/Core');
const meta = Meta(BalaurButton.new);

final $Primary = _Story(
  args: _Args(label: StringArg('Keep it.'), onPressed: Arg.fixed(() {})),
);

final $Ghost = _Story(
  args: _Args(
    label: StringArg('Not this.'),
    onPressed: Arg.fixed(() {}),
    variant: EnumArg(
      BalaurButtonVariant.ghost,
      values: BalaurButtonVariant.values,
    ),
  ),
);

final $Wood = _Story(
  args: _Args(
    label: StringArg('Open the conversation'),
    onPressed: Arg.fixed(() {}),
    variant: EnumArg(
      BalaurButtonVariant.wood,
      values: BalaurButtonVariant.values,
    ),
  ),
);

final $Disabled = _Story(
  args: _Args(label: StringArg('Unavailable'), onPressed: Arg.fixed(null)),
);
