// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_nudge_banner.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurNudgeBanner, StoryArgs<BalaurNudgeBanner>>;
typedef _Scenario = BalaurNudgeBannerScenario;
typedef _Defaults = BalaurNudgeBannerDefaults;
typedef _Story = BalaurNudgeBannerStory;
typedef _Args = BalaurNudgeBannerArgs;
final BalaurNudgeBannerComponent =
    Component<BalaurNudgeBanner, StoryArgs<BalaurNudgeBanner>>(
      name: component.name ?? 'BalaurNudgeBanner',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays a nudge with spoken owner replies.''',
      stories: [$Evening..$generatedName = 'Evening'],
    );
typedef BalaurNudgeBannerScenario =
    Scenario<BalaurNudgeBanner, BalaurNudgeBannerArgs>;
typedef BalaurNudgeBannerDefaults =
    Defaults<BalaurNudgeBanner, BalaurNudgeBannerArgs>;

class BalaurNudgeBannerStory
    extends Story<BalaurNudgeBanner, BalaurNudgeBannerArgs> {
  BalaurNudgeBannerStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurNudgeBanner, BalaurNudgeBannerArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurNudgeBanner(
               key: args.key,
               message: args.message,
               onDone: args.onDone,
               onSnooze: args.onSnooze,
               onTomorrow: args.onTomorrow,
               kicker: args.kicker,
               when: args.when,
             ),
       );
}

class BalaurNudgeBannerArgs extends StoryArgs<BalaurNudgeBanner> {
  BalaurNudgeBannerArgs({
    Arg<Key?>? key,
    Arg<String>? message,
    required Arg<void Function()> onDone,
    required Arg<void Function()> onSnooze,
    required Arg<void Function()> onTomorrow,
    Arg<String>? kicker,
    Arg<String?>? when,
  }) : this.keyArg = $initArg('key', key, null),
       this.messageArg = $initArg('message', message, StringArg(''))!,
       this.onDoneArg = $initArg('onDone', onDone, null)!,
       this.onSnoozeArg = $initArg('onSnooze', onSnooze, null)!,
       this.onTomorrowArg = $initArg('onTomorrow', onTomorrow, null)!,
       this.kickerArg = $initArg('kicker', kicker, StringArg('Nudge'))!,
       this.whenArg = $initArg('when', when, NullableStringArg(null))!;

  BalaurNudgeBannerArgs.fixed({
    Key? key,
    String message = '',
    required void Function() onDone,
    required void Function() onSnooze,
    required void Function() onTomorrow,
    String kicker = 'Nudge',
    String? when = null,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.messageArg = $initArg('message', Arg.fixed(message), null)!,
       this.onDoneArg = $initArg('onDone', Arg.fixed(onDone), null)!,
       this.onSnoozeArg = $initArg('onSnooze', Arg.fixed(onSnooze), null)!,
       this.onTomorrowArg = $initArg(
         'onTomorrow',
         Arg.fixed(onTomorrow),
         null,
       )!,
       this.kickerArg = $initArg('kicker', Arg.fixed(kicker), null)!,
       this.whenArg = $initArg(
         'when',
         when == null ? null : Arg.fixed(when),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<String> messageArg;

  final Arg<void Function()> onDoneArg;

  final Arg<void Function()> onSnoozeArg;

  final Arg<void Function()> onTomorrowArg;

  final Arg<String> kickerArg;

  final Arg<String?>? whenArg;

  Key? get key => keyArg?.value;

  String get message => messageArg.value;

  void Function() get onDone => onDoneArg.value;

  void Function() get onSnooze => onSnoozeArg.value;

  void Function() get onTomorrow => onTomorrowArg.value;

  String get kicker => kickerArg.value;

  String? get when => whenArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    messageArg,
    onDoneArg,
    onSnoozeArg,
    onTomorrowArg,
    kickerArg,
    whenArg,
  ];
}
