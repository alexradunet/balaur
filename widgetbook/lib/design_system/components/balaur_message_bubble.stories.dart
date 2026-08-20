import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_message_bubble.stories.g.dart';

const component = ComponentMeta(
  name: 'Message bubble',
  path: 'Design system/Chat',
);

const meta = Meta(BalaurMessageBubble.new);

final $HouseholdMember = _Story(
  name: 'Household member',
  args: _Args(
    content: StringArg('Can you add milk to the Shopping List?'),
    role: EnumArg(
      BalaurMessageBubbleRole.householdMember,
      values: BalaurMessageBubbleRole.values,
    ),
  ),
);

final $HouseholdMemberLong = _Story(
  name: 'Household member long',
  args: _Args(
    content: StringArg(
      'Please review the Shopping List before tonight and group each item '
      'by the section where I can find it.',
    ),
    role: EnumArg(
      BalaurMessageBubbleRole.householdMember,
      values: BalaurMessageBubbleRole.values,
    ),
  ),
);

final $Agent = _Story(
  name: 'Agent',
  args: _Args(
    content: StringArg('I added milk to the Shopping List.'),
    role: EnumArg(
      BalaurMessageBubbleRole.agent,
      values: BalaurMessageBubbleRole.values,
    ),
  ),
);

final $Streaming = _Story(
  name: 'Streaming',
  args: _Args(
    content: StringArg('I am checking the Shopping List'),
    role: EnumArg(
      BalaurMessageBubbleRole.agent,
      values: BalaurMessageBubbleRole.values,
    ),
    status: EnumArg(
      BalaurMessageBubbleStatus.streaming,
      values: BalaurMessageBubbleStatus.values,
    ),
  ),
);

final $EmptyStreaming = _Story(
  name: 'Empty streaming',
  args: _Args(
    content: StringArg(''),
    role: EnumArg(
      BalaurMessageBubbleRole.agent,
      values: BalaurMessageBubbleRole.values,
    ),
    status: EnumArg(
      BalaurMessageBubbleStatus.streaming,
      values: BalaurMessageBubbleStatus.values,
    ),
  ),
);

final $Stopped = _Story(
  name: 'Stopped',
  args: _Args(
    content: StringArg('I added milk and'),
    role: EnumArg(
      BalaurMessageBubbleRole.agent,
      values: BalaurMessageBubbleRole.values,
    ),
    status: EnumArg(
      BalaurMessageBubbleStatus.stopped,
      values: BalaurMessageBubbleStatus.values,
    ),
  ),
);

final $Failed = _Story(
  name: 'Failed',
  args: _Args(
    content: StringArg('I could not update the Shopping List.'),
    role: EnumArg(
      BalaurMessageBubbleRole.agent,
      values: BalaurMessageBubbleRole.values,
    ),
    status: EnumArg(
      BalaurMessageBubbleStatus.failed,
      values: BalaurMessageBubbleStatus.values,
    ),
  ),
);

final $LongContent = _Story(
  name: 'Long content',
  args: _Args(
    content: StringArg(
      'I reviewed the Shopping List and grouped the items by the section '
      'where you usually find them. You can change the order before you shop.',
    ),
    role: EnumArg(
      BalaurMessageBubbleRole.agent,
      values: BalaurMessageBubbleRole.values,
    ),
  ),
);
