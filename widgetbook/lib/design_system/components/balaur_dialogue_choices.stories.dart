import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_dialogue_choices.stories.g.dart';

const component = ComponentMeta(
  name: 'Dialogue choices',
  path: 'Design system/Chat',
);
const meta = Meta(BalaurDialogueChoices.new);

final $MemoryDecision = _Story(
  args: _Args.fixed(
    choices: [
      BalaurDialogueChoice(label: 'Keep it.', hint: 'approve memory'),
      BalaurDialogueChoice(label: 'Not this.', hint: 'dismiss'),
      BalaurDialogueChoice(label: 'Ask me another time.', hint: 'defer'),
    ],
    onPick: _pick,
    kicker: 'Your word',
    who: 'You',
    avatar: null,
  ),
);

final $TaskDecision = _Story(
  args: _Args.fixed(
    choices: [
      BalaurDialogueChoice(label: 'Hold me to it.', hint: 'keep task'),
      BalaurDialogueChoice(label: 'Just this once.', hint: 'one time'),
      BalaurDialogueChoice(label: 'Let it be.', hint: 'dismiss'),
    ],
    onPick: _pick,
    kicker: 'Will you keep this task?',
    who: 'You',
    avatar: null,
  ),
);

void _pick(int _) {}
