// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_folk_band.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurFolkBand, StoryArgs<BalaurFolkBand>>;
typedef _Scenario = BalaurFolkBandScenario;
typedef _Defaults = BalaurFolkBandDefaults;
typedef _Story = BalaurFolkBandStory;
typedef _Args = BalaurFolkBandArgs;
final BalaurFolkBandComponent =
    Component<BalaurFolkBand, StoryArgs<BalaurFolkBand>>(
      name: component.name ?? 'BalaurFolkBand',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Draws the Romanian folk color band.''',
      stories: [$Default..$generatedName = 'Default'],
    );
typedef BalaurFolkBandScenario = Scenario<BalaurFolkBand, BalaurFolkBandArgs>;
typedef BalaurFolkBandDefaults = Defaults<BalaurFolkBand, BalaurFolkBandArgs>;

class BalaurFolkBandStory extends Story<BalaurFolkBand, BalaurFolkBandArgs> {
  BalaurFolkBandStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurFolkBandArgs? args,
    StoryWidgetBuilder<BalaurFolkBand, BalaurFolkBandArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurFolkBandArgs(),
         builder: builder ?? (context, args) => BalaurFolkBand(key: args.key),
       );
}

class BalaurFolkBandArgs extends StoryArgs<BalaurFolkBand> {
  BalaurFolkBandArgs({Arg<Key?>? key})
    : this.keyArg = $initArg('key', key, null);

  BalaurFolkBandArgs.fixed({Key? key})
    : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null);

  final Arg<Key?>? keyArg;

  Key? get key => keyArg?.value;

  @override
  List<Arg?> get list => [keyArg];
}
