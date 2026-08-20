import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_toggle.stories.g.dart';

const component = ComponentMeta(name: 'Toggle', path: 'Design system/Inputs');
const meta = Meta(BalaurToggle.new);

final $Off = _Story(
  args: _Args(
    checked: BoolArg(false),
    onChanged: Arg.fixed((bool _) {}),
    label: NullableStringArg('Notifications'),
  ),
);

final $On = _Story(
  args: _Args(
    checked: BoolArg(true),
    onChanged: Arg.fixed((bool _) {}),
    label: NullableStringArg('Notifications'),
  ),
);

final $Disabled = _Story(
  args: _Args(
    checked: BoolArg(false),
    onChanged: Arg.fixed(null),
    label: NullableStringArg('Notifications'),
    enabled: BoolArg(false),
  ),
);
