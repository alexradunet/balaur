import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_skeleton.stories.g.dart';

const component = ComponentMeta(name: 'Skeleton', path: 'Design system/Core');
const meta = Meta(BalaurSkeleton.new);

final $Line = _Story(
  args: _Args(
    variant: EnumArg(
      BalaurSkeletonVariant.line,
      values: BalaurSkeletonVariant.values,
    ),
  ),
);

final $Block = _Story(
  args: _Args(
    variant: EnumArg(
      BalaurSkeletonVariant.block,
      values: BalaurSkeletonVariant.values,
    ),
    height: NullableDoubleArg(72),
  ),
);

final $Avatar = _Story(
  args: _Args(
    variant: EnumArg(
      BalaurSkeletonVariant.avatar,
      values: BalaurSkeletonVariant.values,
    ),
    width: NullableDoubleArg(56),
  ),
);
