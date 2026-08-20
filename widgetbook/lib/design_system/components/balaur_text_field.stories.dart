import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_text_field.stories.g.dart';

const component = ComponentMeta(
  name: 'Text field',
  path: 'Design system/Inputs',
);
const meta = Meta(BalaurTextField.new);

final $WithHint = _Story(
  args: _Args(
    label: NullableStringArg('Your name'),
    placeholder: NullableStringArg('Tell Balaur who you are…'),
    hint: NullableStringArg('Kept on the box, nowhere else.'),
  ),
);

final $Error = _Story(
  args: _Args(
    label: NullableStringArg('Box hostname'),
    placeholder: NullableStringArg('hearth.local'),
    error: NullableStringArg('That name is already taken.'),
  ),
);

final $Disabled = _Story(
  args: _Args(
    label: NullableStringArg('Read only'),
    placeholder: NullableStringArg('Unavailable'),
    enabled: BoolArg(false),
  ),
);
