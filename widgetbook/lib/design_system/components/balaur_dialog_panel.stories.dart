import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_dialog_panel.stories.g.dart';

const component = ComponentMeta(
  name: 'Dialog panel',
  path: 'Design system/Feedback',
);
const meta = Meta(BalaurDialogPanel.new);

final $Confirm = _Story(
  args: _Args.fixed(
    title: 'Forget this memory?',
    body: 'Ana — sister, birthday May 3rd. Once forgotten, I cannot bring it back unless you tell me again.',
    kicker: 'A matter of weight',
    actions: [
      BalaurDialogAction(
        label: 'Forget it.',
        onPressed: _noop,
        variant: BalaurButtonVariant.primary,
      ),
      BalaurDialogAction(label: 'Keep it.', onPressed: _noop),
    ],
  ),
);

void _noop() {}
