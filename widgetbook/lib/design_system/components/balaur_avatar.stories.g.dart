// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_avatar.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurAvatar, StoryArgs<BalaurAvatar>>;
typedef _Scenario = BalaurAvatarScenario;
typedef _Defaults = BalaurAvatarDefaults;
typedef _Story = BalaurAvatarStory;
typedef _Args = BalaurAvatarArgs;
final BalaurAvatarComponent = Component<BalaurAvatar, StoryArgs<BalaurAvatar>>(
  name: component.name ?? 'BalaurAvatar',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a borderless local pixel portrait.''',
  stories: [
    $Balaur..$generatedName = 'Balaur',
    $Thinking..$generatedName = 'Thinking',
    $Soul..$generatedName = 'Soul',
  ],
);
typedef BalaurAvatarScenario = Scenario<BalaurAvatar, BalaurAvatarArgs>;
typedef BalaurAvatarDefaults = Defaults<BalaurAvatar, BalaurAvatarArgs>;

class BalaurAvatarStory extends Story<BalaurAvatar, BalaurAvatarArgs> {
  BalaurAvatarStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurAvatar, BalaurAvatarArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurAvatar(
               key: args.key,
               image: args.image,
               kind: args.kind,
               state: args.state,
               size: args.size,
               semanticLabel: args.semanticLabel,
               mirrored: args.mirrored,
             ),
       );
}

class BalaurAvatarArgs extends StoryArgs<BalaurAvatar> {
  BalaurAvatarArgs({
    Arg<Key?>? key,
    required Arg<ImageProvider<Object>> image,
    Arg<BalaurAvatarKind>? kind,
    Arg<BalaurAvatarState>? state,
    Arg<double>? size,
    Arg<String?>? semanticLabel,
    Arg<bool>? mirrored,
  }) : this.keyArg = $initArg('key', key, null),
       this.imageArg = $initArg('image', image, null)!,
       this.kindArg = $initArg(
         'kind',
         kind,
         EnumArg<BalaurAvatarKind>(
           BalaurAvatarKind.balaur,
           values: BalaurAvatarKind.values,
         ),
       )!,
       this.stateArg = $initArg(
         'state',
         state,
         EnumArg<BalaurAvatarState>(
           BalaurAvatarState.idle,
           values: BalaurAvatarState.values,
         ),
       )!,
       this.sizeArg = $initArg('size', size, DoubleArg(54))!,
       this.semanticLabelArg = $initArg(
         'semanticLabel',
         semanticLabel,
         NullableStringArg(null),
       )!,
       this.mirroredArg = $initArg('mirrored', mirrored, BoolArg(false))!;

  BalaurAvatarArgs.fixed({
    Key? key,
    required ImageProvider<Object> image,
    BalaurAvatarKind kind = BalaurAvatarKind.balaur,
    BalaurAvatarState state = BalaurAvatarState.idle,
    double size = 54,
    String? semanticLabel = null,
    bool mirrored = false,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.imageArg = $initArg('image', Arg.fixed(image), null)!,
       this.kindArg = $initArg('kind', Arg.fixed(kind), null)!,
       this.stateArg = $initArg('state', Arg.fixed(state), null)!,
       this.sizeArg = $initArg('size', Arg.fixed(size), null)!,
       this.semanticLabelArg = $initArg(
         'semanticLabel',
         semanticLabel == null ? null : Arg.fixed(semanticLabel),
         null,
       ),
       this.mirroredArg = $initArg('mirrored', Arg.fixed(mirrored), null)!;

  final Arg<Key?>? keyArg;

  final Arg<ImageProvider<Object>> imageArg;

  final Arg<BalaurAvatarKind> kindArg;

  final Arg<BalaurAvatarState> stateArg;

  final Arg<double> sizeArg;

  final Arg<String?>? semanticLabelArg;

  final Arg<bool> mirroredArg;

  Key? get key => keyArg?.value;

  ImageProvider<Object> get image => imageArg.value;

  BalaurAvatarKind get kind => kindArg.value;

  BalaurAvatarState get state => stateArg.value;

  double get size => sizeArg.value;

  String? get semanticLabel => semanticLabelArg?.value;

  bool get mirrored => mirroredArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    imageArg,
    kindArg,
    stateArg,
    sizeArg,
    semanticLabelArg,
    mirroredArg,
  ];
}
