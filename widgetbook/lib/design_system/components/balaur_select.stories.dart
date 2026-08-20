import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_select.stories.g.dart';

const component = ComponentMeta(name: 'Select', path: 'Design system/Inputs');
const meta = Meta(BalaurSelect.new);

final $Model = _Story(
  args: _Args.fixed(
    options: ['llama-3.1-70b', 'mistral-large', 'qwen-2.5', 'phi-4'],
    value: 'mistral-large',
    onChanged: null,
    label: 'Model',
  ),
);

final $Disabled = _Story(
  args: _Args.fixed(
    options: ['llama-3.1-70b', 'mistral-large', 'qwen-2.5', 'phi-4'],
    value: 'phi-4',
    onChanged: null,
    label: 'Model',
    enabled: false,
  ),
);
