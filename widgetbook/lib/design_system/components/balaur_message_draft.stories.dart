import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_message_draft.stories.g.dart';

const component = ComponentMeta(
  name: 'Message draft',
  path: 'Design system/Chat',
);
const meta = Meta(BalaurMessageDraft.new);

final $Unsent = _Story(
  args: _Args(
    onSend: Arg.fixed((String _) {}),
    who: StringArg('You'),
    placeholder: StringArg('Speak; I am listening.'),
    sendLabel: StringArg('Send'),
  ),
);
