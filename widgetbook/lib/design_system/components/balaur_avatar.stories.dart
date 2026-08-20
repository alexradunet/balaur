import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_avatar.stories.g.dart';

const component = ComponentMeta(name: 'Avatar', path: 'Design system/Chat');
const meta = Meta(BalaurAvatar.new);

final $Balaur = _Story(
  args: _Args(
    image: Arg.fixed(BalaurAssets.balaurAvatar(1)),
    kind: EnumArg(BalaurAvatarKind.balaur, values: BalaurAvatarKind.values),
    state: EnumArg(BalaurAvatarState.idle, values: BalaurAvatarState.values),
    size: DoubleArg(96),
  ),
);

final $Thinking = _Story(
  args: _Args(
    image: Arg.fixed(BalaurAssets.balaurAvatar(1)),
    state: EnumArg(
      BalaurAvatarState.thinking,
      values: BalaurAvatarState.values,
    ),
    size: DoubleArg(96),
  ),
);

final $Soul = _Story(
  args: _Args(
    image: Arg.fixed(BalaurAssets.soulAvatar(5)),
    kind: EnumArg(BalaurAvatarKind.soul, values: BalaurAvatarKind.values),
    size: DoubleArg(96),
    mirrored: BoolArg(true),
  ),
);
