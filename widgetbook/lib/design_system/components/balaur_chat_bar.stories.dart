import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_chat_bar.stories.g.dart';

const component = ComponentMeta(name: 'Chat bar', path: 'Design system/Chat');
const meta = Meta(BalaurChatBar.new);

final $Capture = _Story(
  args: _Args(
    onSend: Arg.fixed((String _) {}),
    placeholder: StringArg('Add a task to the book…'),
    sendLabel: StringArg('Keep'),
  ),
);
