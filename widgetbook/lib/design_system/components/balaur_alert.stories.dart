import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_alert.stories.g.dart';

const component = ComponentMeta(name: 'Alert', path: 'Design system/Feedback');
const meta = Meta(BalaurAlert.new);

final $Information = _Story(
  args: _Args(
    message: StringArg('A local head. Nothing you say here leaves the box.'),
    title: NullableStringArg('Running on llama-3.1'),
    tone: EnumArg(BalaurAlertTone.info, values: BalaurAlertTone.values),
  ),
);

final $Warning = _Story(
  args: _Args(
    message: StringArg('Your nightly mirror has nowhere to go.'),
    title: NullableStringArg('Vault disk not found'),
    tone: EnumArg(BalaurAlertTone.warning, values: BalaurAlertTone.values),
  ),
);

final $Danger = _Story(
  args: _Args(
    message: StringArg('Under 1 GB remains on the box.'),
    title: NullableStringArg('Low disk'),
    tone: EnumArg(BalaurAlertTone.danger, values: BalaurAlertTone.values),
  ),
);
