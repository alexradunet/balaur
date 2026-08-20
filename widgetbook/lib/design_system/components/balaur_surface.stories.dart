import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_surface.stories.g.dart';

const component = ComponentMeta(
  name: 'Materials',
  path: 'Design system/Foundations',
);
const meta = Meta(BalaurSurface.new);

final $Parchment = _Story(
  args: _Args(
    child: Arg.fixed(const SizedBox(width: 280, height: 100)),
    material: EnumArg(
      BalaurSurfaceMaterial.parchment,
      values: BalaurSurfaceMaterial.values,
    ),
  ),
);

final $Wood = _Story(
  args: _Args(
    child: Arg.fixed(const SizedBox(width: 280, height: 100)),
    material: EnumArg(
      BalaurSurfaceMaterial.wood,
      values: BalaurSurfaceMaterial.values,
    ),
  ),
);

final $Inset = _Story(
  args: _Args(
    child: Arg.fixed(const SizedBox(width: 280, height: 100)),
    material: EnumArg(
      BalaurSurfaceMaterial.inset,
      values: BalaurSurfaceMaterial.values,
    ),
  ),
);
