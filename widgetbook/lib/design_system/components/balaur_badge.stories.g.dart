// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_badge.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurBadge, StoryArgs<BalaurBadge>>;
typedef _Scenario = BalaurBadgeScenario;
typedef _Defaults = BalaurBadgeDefaults;
typedef _Story = BalaurBadgeStory;
typedef _Args = BalaurBadgeArgs;
final BalaurBadgeComponent = Component<BalaurBadge, StoryArgs<BalaurBadge>>(
  name: component.name ?? 'BalaurBadge',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a small count, status, or bare marker.''',
  stories: [
    $Count..$generatedName = 'Count',
    $Urgent..$generatedName = 'Urgent',
    $Dot..$generatedName = 'Dot',
  ],
);
typedef BalaurBadgeScenario = Scenario<BalaurBadge, BalaurBadgeArgs>;
typedef BalaurBadgeDefaults = Defaults<BalaurBadge, BalaurBadgeArgs>;

class BalaurBadgeStory extends Story<BalaurBadge, BalaurBadgeArgs> {
  BalaurBadgeStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurBadgeArgs? args,
    StoryWidgetBuilder<BalaurBadge, BalaurBadgeArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurBadgeArgs(),
         builder:
             builder ??
             (context, args) => BalaurBadge(
               key: args.key,
               label: args.label,
               tone: args.tone,
               dot: args.dot,
             ),
       );
}

class BalaurBadgeArgs extends StoryArgs<BalaurBadge> {
  BalaurBadgeArgs({
    Arg<Key?>? key,
    Arg<String?>? label,
    Arg<BalaurBadgeTone>? tone,
    Arg<bool>? dot,
  }) : this.keyArg = $initArg('key', key, null),
       this.labelArg = $initArg('label', label, NullableStringArg(null))!,
       this.toneArg = $initArg(
         'tone',
         tone,
         EnumArg<BalaurBadgeTone>(
           BalaurBadgeTone.gold,
           values: BalaurBadgeTone.values,
         ),
       )!,
       this.dotArg = $initArg('dot', dot, BoolArg(false))!;

  BalaurBadgeArgs.fixed({
    Key? key,
    String? label = null,
    BalaurBadgeTone tone = BalaurBadgeTone.gold,
    bool dot = false,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.labelArg = $initArg(
         'label',
         label == null ? null : Arg.fixed(label),
         null,
       ),
       this.toneArg = $initArg('tone', Arg.fixed(tone), null)!,
       this.dotArg = $initArg('dot', Arg.fixed(dot), null)!;

  final Arg<Key?>? keyArg;

  final Arg<String?>? labelArg;

  final Arg<BalaurBadgeTone> toneArg;

  final Arg<bool> dotArg;

  Key? get key => keyArg?.value;

  String? get label => labelArg?.value;

  BalaurBadgeTone get tone => toneArg.value;

  bool get dot => dotArg.value;

  @override
  List<Arg?> get list => [keyArg, labelArg, toneArg, dotArg];
}
