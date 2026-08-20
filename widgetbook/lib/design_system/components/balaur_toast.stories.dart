import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_toast.stories.g.dart';

const component = ComponentMeta(name: 'Toast', path: 'Design system/Feedback');
const meta = Meta(BalaurToast.new);

final $Success = _Story(
  args: _Args(
    message: StringArg('Logged to weight — 81.2 kg. It joins the life log.'),
    tone: EnumArg(BalaurToastTone.success, values: BalaurToastTone.values),
  ),
);

final $Information = _Story(
  args: _Args(
    message: StringArg('I shall weigh it. It waits for your word.'),
    tone: EnumArg(BalaurToastTone.info, values: BalaurToastTone.values),
  ),
);

final $Warning = _Story(
  args: _Args(
    message: StringArg('The vault disk is not available.'),
    tone: EnumArg(BalaurToastTone.warning, values: BalaurToastTone.values),
  ),
);
