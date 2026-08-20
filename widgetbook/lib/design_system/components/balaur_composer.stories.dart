import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_composer.stories.g.dart';

const component = ComponentMeta(name: 'Composer', path: 'Design system/Chat');
const meta = Meta(BalaurComposer.new);

final $Draft = _Story(
  args: _Args.fixed(
    onSend: _send,
    who: 'You',
    avatar: null,
    placeholder: 'Speak; I am listening.',
    sendLabel: 'Send',
    tools: [
      BalaurComposerTool(
        iconName: 'scroll',
        tooltip: 'Attach a scroll',
        onPressed: _noop,
      ),
      BalaurComposerTool(
        iconName: 'tome',
        tooltip: 'Add from memory',
        onPressed: _noop,
      ),
      BalaurComposerTool(
        iconName: 'lens',
        tooltip: 'Recall a thread',
        onPressed: _noop,
      ),
    ],
  ),
);

final $Decision = _Story(
  args: _Args.fixed(
    onSend: _send,
    promptLabel: 'A memory awaits your word',
    decision: BalaurDialogueChoices(
      choices: [
        BalaurDialogueChoice(label: 'Keep it.', hint: 'approve memory'),
        BalaurDialogueChoice(label: 'Not this.', hint: 'dismiss'),
      ],
      onPick: _pick,
    ),
  ),
);

void _send(String _) {}
void _pick(int _) {}
void _noop() {}
