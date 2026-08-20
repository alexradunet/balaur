import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_card.stories.g.dart';

const component = ComponentMeta(name: 'Card', path: 'Design system/Core');
const meta = Meta(BalaurCard.new);

final $Parchment = _Story(
  args: _Args(
    child: Arg.fixed(const Text('A warm parchment panel with a gold notch.')),
  ),
);

final $Ornate = _Story(
  args: _Args(
    child: Arg.fixed(
      const Text('Reserved for a decision that carries weight.'),
    ),
    ornate: BoolArg(true),
  ),
);

final $Archived = _Story(
  args: _Args(
    child: Arg.fixed(const Text('Dashed material marks an archived record.')),
    dashed: BoolArg(true),
  ),
);
