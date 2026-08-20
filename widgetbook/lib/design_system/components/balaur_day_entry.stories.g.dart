// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_day_entry.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurDayEntry, StoryArgs<BalaurDayEntry>>;
typedef _Scenario = BalaurDayEntryScenario;
typedef _Defaults = BalaurDayEntryDefaults;
typedef _Story = BalaurDayEntryStory;
typedef _Args = BalaurDayEntryArgs;
final BalaurDayEntryComponent =
    Component<BalaurDayEntry, StoryArgs<BalaurDayEntry>>(
      name: component.name ?? 'BalaurDayEntry',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays one item on a vertical day timeline.''',
      stories: [$Gold..$generatedName = 'Gold', $Due..$generatedName = 'Due'],
    );
typedef BalaurDayEntryScenario = Scenario<BalaurDayEntry, BalaurDayEntryArgs>;
typedef BalaurDayEntryDefaults = Defaults<BalaurDayEntry, BalaurDayEntryArgs>;

class BalaurDayEntryStory extends Story<BalaurDayEntry, BalaurDayEntryArgs> {
  BalaurDayEntryStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurDayEntryArgs? args,
    StoryWidgetBuilder<BalaurDayEntry, BalaurDayEntryArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurDayEntryArgs(),
         builder:
             builder ??
             (context, args) => BalaurDayEntry(
               key: args.key,
               time: args.time,
               title: args.title,
               detail: args.detail,
               tone: args.tone,
               last: args.last,
             ),
       );
}

class BalaurDayEntryArgs extends StoryArgs<BalaurDayEntry> {
  BalaurDayEntryArgs({
    Arg<Key?>? key,
    Arg<String>? time,
    Arg<String>? title,
    Arg<String?>? detail,
    Arg<BalaurDayEntryTone>? tone,
    Arg<bool>? last,
  }) : this.keyArg = $initArg('key', key, null),
       this.timeArg = $initArg('time', time, StringArg(''))!,
       this.titleArg = $initArg('title', title, StringArg(''))!,
       this.detailArg = $initArg('detail', detail, NullableStringArg(null))!,
       this.toneArg = $initArg(
         'tone',
         tone,
         EnumArg<BalaurDayEntryTone>(
           BalaurDayEntryTone.gold,
           values: BalaurDayEntryTone.values,
         ),
       )!,
       this.lastArg = $initArg('last', last, BoolArg(false))!;

  BalaurDayEntryArgs.fixed({
    Key? key,
    String time = '',
    String title = '',
    String? detail = null,
    BalaurDayEntryTone tone = BalaurDayEntryTone.gold,
    bool last = false,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.timeArg = $initArg('time', Arg.fixed(time), null)!,
       this.titleArg = $initArg('title', Arg.fixed(title), null)!,
       this.detailArg = $initArg(
         'detail',
         detail == null ? null : Arg.fixed(detail),
         null,
       ),
       this.toneArg = $initArg('tone', Arg.fixed(tone), null)!,
       this.lastArg = $initArg('last', Arg.fixed(last), null)!;

  final Arg<Key?>? keyArg;

  final Arg<String> timeArg;

  final Arg<String> titleArg;

  final Arg<String?>? detailArg;

  final Arg<BalaurDayEntryTone> toneArg;

  final Arg<bool> lastArg;

  Key? get key => keyArg?.value;

  String get time => timeArg.value;

  String get title => titleArg.value;

  String? get detail => detailArg?.value;

  BalaurDayEntryTone get tone => toneArg.value;

  bool get last => lastArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    timeArg,
    titleArg,
    detailArg,
    toneArg,
    lastArg,
  ];
}
