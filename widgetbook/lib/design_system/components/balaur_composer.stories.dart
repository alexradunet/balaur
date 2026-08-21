import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_composer.stories.g.dart';

const component = ComponentMeta(name: 'Composer', path: 'Design system/Chat');
const meta = Meta(BalaurComposer.new);
const choicesMeta = Meta(BalaurComposer.choices);

final $Draft = _Story(
  args: _Args.fixed(
    onSend: _send,
    placeholder: 'Speak; I am listening.',
    sendLabel: 'Send',
    tools: _tools,
  ),
);

final $Responding = _Story(
  args: _Args.fixed(
    onSend: _send,
    tools: _tools,
    responding: true,
    onStop: _noop,
  ),
);

final $Disabled = _Story(
  args: _Args.fixed(onSend: _send, tools: _tools, enabled: false),
);

final $DialogueChoices = _ChoicesStory(
  name: 'Dialogue choices',
  args: _ChoicesArgs.fixed(
    choices: const [
      BalaurDialogueChoice(label: 'Keep it.', hint: 'approve memory'),
      BalaurDialogueChoice(label: 'Not this.', hint: 'dismiss'),
      BalaurDialogueChoice(label: 'Ask me another time.', hint: 'defer'),
    ],
    onPick: _pick,
    promptLabel: 'A memory awaits your word',
  ),
);

final _tools = [
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
];

void _send(String _) {}
void _pick(int _) {}
void _noop() {}
