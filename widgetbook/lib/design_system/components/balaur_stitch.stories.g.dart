// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_stitch.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurStitch, StoryArgs<BalaurStitch>>;
typedef _Scenario = BalaurStitchScenario;
typedef _Defaults = BalaurStitchDefaults;
typedef _Story = BalaurStitchStory;
typedef _Args = BalaurStitchArgs;
final BalaurStitchComponent = Component<BalaurStitch, StoryArgs<BalaurStitch>>(
  name: component.name ?? 'BalaurStitch',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Draws a dashed stitch between sections.''',
  stories: [$Default..$generatedName = 'Default'],
);
typedef BalaurStitchScenario = Scenario<BalaurStitch, BalaurStitchArgs>;
typedef BalaurStitchDefaults = Defaults<BalaurStitch, BalaurStitchArgs>;

class BalaurStitchStory extends Story<BalaurStitch, BalaurStitchArgs> {
  BalaurStitchStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurStitchArgs? args,
    StoryWidgetBuilder<BalaurStitch, BalaurStitchArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurStitchArgs(),
         builder:
             builder ??
             (context, args) => BalaurStitch(key: args.key, color: args.color),
       );
}

class BalaurStitchArgs extends StoryArgs<BalaurStitch> {
  BalaurStitchArgs({Arg<Key?>? key, Arg<Color?>? color})
    : this.keyArg = $initArg('key', key, null),
      this.colorArg = $initArg('color', color, NullableColorArg(null))!;

  BalaurStitchArgs.fixed({Key? key, Color? color = null})
    : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
      this.colorArg = $initArg(
        'color',
        color == null ? null : Arg.fixed(color),
        null,
      );

  final Arg<Key?>? keyArg;

  final Arg<Color?>? colorArg;

  Key? get key => keyArg?.value;

  Color? get color => colorArg?.value;

  @override
  List<Arg?> get list => [keyArg, colorArg];
}
