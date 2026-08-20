// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_pips.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurPips, StoryArgs<BalaurPips>>;
typedef _Scenario = BalaurPipsScenario;
typedef _Defaults = BalaurPipsDefaults;
typedef _Story = BalaurPipsStory;
typedef _Args = BalaurPipsArgs;
final BalaurPipsComponent = Component<BalaurPips, StoryArgs<BalaurPips>>(
  name: component.name ?? 'BalaurPips',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a square importance dial.''',
  stories: [$Importance..$generatedName = 'Importance'],
);
typedef BalaurPipsScenario = Scenario<BalaurPips, BalaurPipsArgs>;
typedef BalaurPipsDefaults = Defaults<BalaurPips, BalaurPipsArgs>;

class BalaurPipsStory extends Story<BalaurPips, BalaurPipsArgs> {
  BalaurPipsStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurPipsArgs? args,
    StoryWidgetBuilder<BalaurPips, BalaurPipsArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurPipsArgs(),
         builder:
             builder ??
             (context, args) =>
                 BalaurPips(key: args.key, level: args.level, max: args.max),
       );
}

class BalaurPipsArgs extends StoryArgs<BalaurPips> {
  BalaurPipsArgs({Arg<Key?>? key, Arg<int>? level, Arg<int>? max})
    : this.keyArg = $initArg('key', key, null),
      this.levelArg = $initArg('level', level, IntArg(0))!,
      this.maxArg = $initArg('max', max, IntArg(5))!;

  BalaurPipsArgs.fixed({Key? key, int level = 0, int max = 5})
    : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
      this.levelArg = $initArg('level', Arg.fixed(level), null)!,
      this.maxArg = $initArg('max', Arg.fixed(max), null)!;

  final Arg<Key?>? keyArg;

  final Arg<int> levelArg;

  final Arg<int> maxArg;

  Key? get key => keyArg?.value;

  int get level => levelArg.value;

  int get max => maxArg.value;

  @override
  List<Arg?> get list => [keyArg, levelArg, maxArg];
}
