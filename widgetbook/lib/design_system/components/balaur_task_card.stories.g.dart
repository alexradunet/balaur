// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_task_card.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurTaskCard, StoryArgs<BalaurTaskCard>>;
typedef _Scenario = BalaurTaskCardScenario;
typedef _Defaults = BalaurTaskCardDefaults;
typedef _Story = BalaurTaskCardStory;
typedef _Args = BalaurTaskCardArgs;
final BalaurTaskCardComponent =
    Component<BalaurTaskCard, StoryArgs<BalaurTaskCard>>(
      name: component.name ?? 'BalaurTaskCard',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays an operational task in chat or a task list.''',
      stories: [
        $OpenRecurring..$generatedName = 'OpenRecurring',
        $Overdue..$generatedName = 'Overdue',
        $Done..$generatedName = 'Done',
      ],
    );
typedef BalaurTaskCardScenario = Scenario<BalaurTaskCard, BalaurTaskCardArgs>;
typedef BalaurTaskCardDefaults = Defaults<BalaurTaskCard, BalaurTaskCardArgs>;

class BalaurTaskCardStory extends Story<BalaurTaskCard, BalaurTaskCardArgs> {
  BalaurTaskCardStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurTaskCardArgs? args,
    StoryWidgetBuilder<BalaurTaskCard, BalaurTaskCardArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurTaskCardArgs(),
         builder:
             builder ??
             (context, args) => BalaurTaskCard(
               key: args.key,
               title: args.title,
               status: args.status,
               dueLine: args.dueLine,
               overdue: args.overdue,
               recurrence: args.recurrence,
               notes: args.notes,
               onDone: args.onDone,
               onSnooze: args.onSnooze,
               onDrop: args.onDrop,
             ),
       );
}

class BalaurTaskCardArgs extends StoryArgs<BalaurTaskCard> {
  BalaurTaskCardArgs({
    Arg<Key?>? key,
    Arg<String>? title,
    Arg<BalaurTaskStatus>? status,
    Arg<String?>? dueLine,
    Arg<bool>? overdue,
    Arg<String?>? recurrence,
    Arg<String?>? notes,
    Arg<void Function()?>? onDone,
    Arg<void Function()?>? onSnooze,
    Arg<void Function()?>? onDrop,
  }) : this.keyArg = $initArg('key', key, null),
       this.titleArg = $initArg('title', title, StringArg(''))!,
       this.statusArg = $initArg(
         'status',
         status,
         EnumArg<BalaurTaskStatus>(
           BalaurTaskStatus.open,
           values: BalaurTaskStatus.values,
         ),
       )!,
       this.dueLineArg = $initArg('dueLine', dueLine, NullableStringArg(null))!,
       this.overdueArg = $initArg('overdue', overdue, BoolArg(false))!,
       this.recurrenceArg = $initArg(
         'recurrence',
         recurrence,
         NullableStringArg(null),
       )!,
       this.notesArg = $initArg('notes', notes, NullableStringArg(null))!,
       this.onDoneArg = $initArg('onDone', onDone, null),
       this.onSnoozeArg = $initArg('onSnooze', onSnooze, null),
       this.onDropArg = $initArg('onDrop', onDrop, null);

  BalaurTaskCardArgs.fixed({
    Key? key,
    String title = '',
    BalaurTaskStatus status = BalaurTaskStatus.open,
    String? dueLine = null,
    bool overdue = false,
    String? recurrence = null,
    String? notes = null,
    void Function()? onDone,
    void Function()? onSnooze,
    void Function()? onDrop,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.titleArg = $initArg('title', Arg.fixed(title), null)!,
       this.statusArg = $initArg('status', Arg.fixed(status), null)!,
       this.dueLineArg = $initArg(
         'dueLine',
         dueLine == null ? null : Arg.fixed(dueLine),
         null,
       ),
       this.overdueArg = $initArg('overdue', Arg.fixed(overdue), null)!,
       this.recurrenceArg = $initArg(
         'recurrence',
         recurrence == null ? null : Arg.fixed(recurrence),
         null,
       ),
       this.notesArg = $initArg(
         'notes',
         notes == null ? null : Arg.fixed(notes),
         null,
       ),
       this.onDoneArg = $initArg(
         'onDone',
         onDone == null ? null : Arg.fixed(onDone),
         null,
       ),
       this.onSnoozeArg = $initArg(
         'onSnooze',
         onSnooze == null ? null : Arg.fixed(onSnooze),
         null,
       ),
       this.onDropArg = $initArg(
         'onDrop',
         onDrop == null ? null : Arg.fixed(onDrop),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<String> titleArg;

  final Arg<BalaurTaskStatus> statusArg;

  final Arg<String?>? dueLineArg;

  final Arg<bool> overdueArg;

  final Arg<String?>? recurrenceArg;

  final Arg<String?>? notesArg;

  final Arg<void Function()?>? onDoneArg;

  final Arg<void Function()?>? onSnoozeArg;

  final Arg<void Function()?>? onDropArg;

  Key? get key => keyArg?.value;

  String get title => titleArg.value;

  BalaurTaskStatus get status => statusArg.value;

  String? get dueLine => dueLineArg?.value;

  bool get overdue => overdueArg.value;

  String? get recurrence => recurrenceArg?.value;

  String? get notes => notesArg?.value;

  void Function()? get onDone => onDoneArg?.value;

  void Function()? get onSnooze => onSnoozeArg?.value;

  void Function()? get onDrop => onDropArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    titleArg,
    statusArg,
    dueLineArg,
    overdueArg,
    recurrenceArg,
    notesArg,
    onDoneArg,
    onSnoozeArg,
    onDropArg,
  ];
}
