import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_recap_card.stories.g.dart';

const component = ComponentMeta(
  name: 'Recap card',
  path: 'Design system/Domain',
);
const meta = Meta(BalaurRecapCard.new);

final $EarlierToday = _Story(
  args: _Args.fixed(
    summary: 'You asked me to keep the garden and your note format in view.',
    points: [
      'Garden — tomatoes and peppers, watered at dusk',
      'Notes exported as Markdown',
      'Mend the deer fence before the weekend',
    ],
    kicker: 'Recap',
    when: 'earlier today',
  ),
);
