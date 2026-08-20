import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_tool_row.stories.g.dart';

const component = ComponentMeta(name: 'Tool row', path: 'Design system/Chat');
const meta = Meta(BalaurToolRow.new);

final $Search = _Story(
  args: _Args(
    tool: StringArg('search'),
    message: StringArg('found 3 notes about the garden'),
    iconName: NullableStringArg('lens'),
  ),
);

final $FallbackGlyph = _Story(
  args: _Args(
    tool: StringArg('shell'),
    message: StringArg('read the current directory'),
    glyph: StringArg('⌥'),
  ),
);
