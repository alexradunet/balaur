// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_calendar_cell.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<BalaurCalendarCell, StoryArgs<BalaurCalendarCell>>;
typedef _Scenario = BalaurCalendarCellScenario;
typedef _Defaults = BalaurCalendarCellDefaults;
typedef _Story = BalaurCalendarCellStory;
typedef _Args = BalaurCalendarCellArgs;
final BalaurCalendarCellComponent =
    Component<BalaurCalendarCell, StoryArgs<BalaurCalendarCell>>(
      name: component.name ?? 'BalaurCalendarCell',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays one square calendar day.''',
      stories: [
        $Default..$generatedName = 'Default',
        $Today..$generatedName = 'Today',
        $Selected..$generatedName = 'Selected',
        $OtherMonth..$generatedName = 'OtherMonth',
      ],
    );
typedef BalaurCalendarCellScenario =
    Scenario<BalaurCalendarCell, BalaurCalendarCellArgs>;
typedef BalaurCalendarCellDefaults =
    Defaults<BalaurCalendarCell, BalaurCalendarCellArgs>;

class BalaurCalendarCellStory
    extends Story<BalaurCalendarCell, BalaurCalendarCellArgs> {
  BalaurCalendarCellStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurCalendarCellArgs? args,
    StoryWidgetBuilder<BalaurCalendarCell, BalaurCalendarCellArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurCalendarCellArgs(),
         builder:
             builder ??
             (context, args) => BalaurCalendarCell(
               key: args.key,
               day: args.day,
               pips: args.pips,
               today: args.today,
               selected: args.selected,
               dimmed: args.dimmed,
               onPressed: args.onPressed,
             ),
       );
}

class BalaurCalendarCellArgs extends StoryArgs<BalaurCalendarCell> {
  BalaurCalendarCellArgs({
    Arg<Key?>? key,
    Arg<int>? day,
    Arg<int>? pips,
    Arg<bool>? today,
    Arg<bool>? selected,
    Arg<bool>? dimmed,
    Arg<void Function()?>? onPressed,
  }) : this.keyArg = $initArg('key', key, null),
       this.dayArg = $initArg('day', day, IntArg(0))!,
       this.pipsArg = $initArg('pips', pips, IntArg(0))!,
       this.todayArg = $initArg('today', today, BoolArg(false))!,
       this.selectedArg = $initArg('selected', selected, BoolArg(false))!,
       this.dimmedArg = $initArg('dimmed', dimmed, BoolArg(false))!,
       this.onPressedArg = $initArg('onPressed', onPressed, null);

  BalaurCalendarCellArgs.fixed({
    Key? key,
    int day = 0,
    int pips = 0,
    bool today = false,
    bool selected = false,
    bool dimmed = false,
    void Function()? onPressed,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.dayArg = $initArg('day', Arg.fixed(day), null)!,
       this.pipsArg = $initArg('pips', Arg.fixed(pips), null)!,
       this.todayArg = $initArg('today', Arg.fixed(today), null)!,
       this.selectedArg = $initArg('selected', Arg.fixed(selected), null)!,
       this.dimmedArg = $initArg('dimmed', Arg.fixed(dimmed), null)!,
       this.onPressedArg = $initArg(
         'onPressed',
         onPressed == null ? null : Arg.fixed(onPressed),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<int> dayArg;

  final Arg<int> pipsArg;

  final Arg<bool> todayArg;

  final Arg<bool> selectedArg;

  final Arg<bool> dimmedArg;

  final Arg<void Function()?>? onPressedArg;

  Key? get key => keyArg?.value;

  int get day => dayArg.value;

  int get pips => pipsArg.value;

  bool get today => todayArg.value;

  bool get selected => selectedArg.value;

  bool get dimmed => dimmedArg.value;

  void Function()? get onPressed => onPressedArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    dayArg,
    pipsArg,
    todayArg,
    selectedArg,
    dimmedArg,
    onPressedArg,
  ];
}
