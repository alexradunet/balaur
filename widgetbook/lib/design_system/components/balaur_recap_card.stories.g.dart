// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_recap_card.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurRecapCard, StoryArgs<BalaurRecapCard>>;
typedef _Scenario = BalaurRecapCardScenario;
typedef _Defaults = BalaurRecapCardDefaults;
typedef _Story = BalaurRecapCardStory;
typedef _Args = BalaurRecapCardArgs;
final BalaurRecapCardComponent =
    Component<BalaurRecapCard, StoryArgs<BalaurRecapCard>>(
      name: component.name ?? 'BalaurRecapCard',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays the older Conversation context that Balaur carries forward.''',
      stories: [$EarlierToday..$generatedName = 'EarlierToday'],
    );
typedef BalaurRecapCardScenario =
    Scenario<BalaurRecapCard, BalaurRecapCardArgs>;
typedef BalaurRecapCardDefaults =
    Defaults<BalaurRecapCard, BalaurRecapCardArgs>;

class BalaurRecapCardStory extends Story<BalaurRecapCard, BalaurRecapCardArgs> {
  BalaurRecapCardStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurRecapCardArgs? args,
    StoryWidgetBuilder<BalaurRecapCard, BalaurRecapCardArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurRecapCardArgs(),
         builder:
             builder ??
             (context, args) => BalaurRecapCard(
               key: args.key,
               summary: args.summary,
               points: args.points,
               kicker: args.kicker,
               when: args.when,
             ),
       );
}

class BalaurRecapCardArgs extends StoryArgs<BalaurRecapCard> {
  BalaurRecapCardArgs({
    Arg<Key?>? key,
    Arg<String>? summary,
    Arg<List<String>>? points,
    Arg<String>? kicker,
    Arg<String?>? when,
  }) : this.keyArg = $initArg('key', key, null),
       this.summaryArg = $initArg('summary', summary, StringArg(''))!,
       this.pointsArg = $initArg('points', points, ConstArg(const []))!,
       this.kickerArg = $initArg('kicker', kicker, StringArg('Recap'))!,
       this.whenArg = $initArg('when', when, NullableStringArg(null))!;

  BalaurRecapCardArgs.fixed({
    Key? key,
    String summary = '',
    List<String> points = const [],
    String kicker = 'Recap',
    String? when = null,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.summaryArg = $initArg('summary', Arg.fixed(summary), null)!,
       this.pointsArg = $initArg('points', Arg.fixed(points), null)!,
       this.kickerArg = $initArg('kicker', Arg.fixed(kicker), null)!,
       this.whenArg = $initArg(
         'when',
         when == null ? null : Arg.fixed(when),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<String> summaryArg;

  final Arg<List<String>> pointsArg;

  final Arg<String> kickerArg;

  final Arg<String?>? whenArg;

  Key? get key => keyArg?.value;

  String get summary => summaryArg.value;

  List<String> get points => pointsArg.value;

  String get kicker => kickerArg.value;

  String? get when => whenArg?.value;

  @override
  List<Arg?> get list => [keyArg, summaryArg, pointsArg, kickerArg, whenArg];
}
