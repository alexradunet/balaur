// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_knowledge_card.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<BalaurKnowledgeCard, StoryArgs<BalaurKnowledgeCard>>;
typedef _Scenario = BalaurKnowledgeCardScenario;
typedef _Defaults = BalaurKnowledgeCardDefaults;
typedef _Story = BalaurKnowledgeCardStory;
typedef _Args = BalaurKnowledgeCardArgs;
final BalaurKnowledgeCardComponent =
    Component<BalaurKnowledgeCard, StoryArgs<BalaurKnowledgeCard>>(
      name: component.name ?? 'BalaurKnowledgeCard',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment:
          r'''Displays a knowledge record through its full lifecycle.''',
      stories: [
        $Proposed..$generatedName = 'Proposed',
        $Active..$generatedName = 'Active',
        $Archived..$generatedName = 'Archived',
      ],
    );
typedef BalaurKnowledgeCardScenario =
    Scenario<BalaurKnowledgeCard, BalaurKnowledgeCardArgs>;
typedef BalaurKnowledgeCardDefaults =
    Defaults<BalaurKnowledgeCard, BalaurKnowledgeCardArgs>;

class BalaurKnowledgeCardStory
    extends Story<BalaurKnowledgeCard, BalaurKnowledgeCardArgs> {
  BalaurKnowledgeCardStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurKnowledgeCardArgs? args,
    StoryWidgetBuilder<BalaurKnowledgeCard, BalaurKnowledgeCardArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurKnowledgeCardArgs(),
         builder:
             builder ??
             (context, args) => BalaurKnowledgeCard(
               key: args.key,
               title: args.title,
               kind: args.kind,
               status: args.status,
               body: args.body,
               whenToUse: args.whenToUse,
               importance: args.importance,
               usedCount: args.usedCount,
               onApprove: args.onApprove,
               onDismiss: args.onDismiss,
               onArchive: args.onArchive,
               onRestore: args.onRestore,
             ),
       );
}

class BalaurKnowledgeCardArgs extends StoryArgs<BalaurKnowledgeCard> {
  BalaurKnowledgeCardArgs({
    Arg<Key?>? key,
    Arg<String>? title,
    Arg<BalaurKnowledgeKind>? kind,
    Arg<BalaurKnowledgeStatus>? status,
    Arg<String?>? body,
    Arg<String?>? whenToUse,
    Arg<int?>? importance,
    Arg<int?>? usedCount,
    Arg<void Function()?>? onApprove,
    Arg<void Function()?>? onDismiss,
    Arg<void Function()?>? onArchive,
    Arg<void Function()?>? onRestore,
  }) : this.keyArg = $initArg('key', key, null),
       this.titleArg = $initArg('title', title, StringArg(''))!,
       this.kindArg = $initArg(
         'kind',
         kind,
         EnumArg<BalaurKnowledgeKind>(
           BalaurKnowledgeKind.memory,
           values: BalaurKnowledgeKind.values,
         ),
       )!,
       this.statusArg = $initArg(
         'status',
         status,
         EnumArg<BalaurKnowledgeStatus>(
           BalaurKnowledgeStatus.active,
           values: BalaurKnowledgeStatus.values,
         ),
       )!,
       this.bodyArg = $initArg('body', body, NullableStringArg(null))!,
       this.whenToUseArg = $initArg(
         'whenToUse',
         whenToUse,
         NullableStringArg(null),
       )!,
       this.importanceArg = $initArg(
         'importance',
         importance,
         NullableIntArg(null),
       )!,
       this.usedCountArg = $initArg(
         'usedCount',
         usedCount,
         NullableIntArg(null),
       )!,
       this.onApproveArg = $initArg('onApprove', onApprove, null),
       this.onDismissArg = $initArg('onDismiss', onDismiss, null),
       this.onArchiveArg = $initArg('onArchive', onArchive, null),
       this.onRestoreArg = $initArg('onRestore', onRestore, null);

  BalaurKnowledgeCardArgs.fixed({
    Key? key,
    String title = '',
    BalaurKnowledgeKind kind = BalaurKnowledgeKind.memory,
    BalaurKnowledgeStatus status = BalaurKnowledgeStatus.active,
    String? body = null,
    String? whenToUse = null,
    int? importance = null,
    int? usedCount = null,
    void Function()? onApprove,
    void Function()? onDismiss,
    void Function()? onArchive,
    void Function()? onRestore,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.titleArg = $initArg('title', Arg.fixed(title), null)!,
       this.kindArg = $initArg('kind', Arg.fixed(kind), null)!,
       this.statusArg = $initArg('status', Arg.fixed(status), null)!,
       this.bodyArg = $initArg(
         'body',
         body == null ? null : Arg.fixed(body),
         null,
       ),
       this.whenToUseArg = $initArg(
         'whenToUse',
         whenToUse == null ? null : Arg.fixed(whenToUse),
         null,
       ),
       this.importanceArg = $initArg(
         'importance',
         importance == null ? null : Arg.fixed(importance),
         null,
       ),
       this.usedCountArg = $initArg(
         'usedCount',
         usedCount == null ? null : Arg.fixed(usedCount),
         null,
       ),
       this.onApproveArg = $initArg(
         'onApprove',
         onApprove == null ? null : Arg.fixed(onApprove),
         null,
       ),
       this.onDismissArg = $initArg(
         'onDismiss',
         onDismiss == null ? null : Arg.fixed(onDismiss),
         null,
       ),
       this.onArchiveArg = $initArg(
         'onArchive',
         onArchive == null ? null : Arg.fixed(onArchive),
         null,
       ),
       this.onRestoreArg = $initArg(
         'onRestore',
         onRestore == null ? null : Arg.fixed(onRestore),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<String> titleArg;

  final Arg<BalaurKnowledgeKind> kindArg;

  final Arg<BalaurKnowledgeStatus> statusArg;

  final Arg<String?>? bodyArg;

  final Arg<String?>? whenToUseArg;

  final Arg<int?>? importanceArg;

  final Arg<int?>? usedCountArg;

  final Arg<void Function()?>? onApproveArg;

  final Arg<void Function()?>? onDismissArg;

  final Arg<void Function()?>? onArchiveArg;

  final Arg<void Function()?>? onRestoreArg;

  Key? get key => keyArg?.value;

  String get title => titleArg.value;

  BalaurKnowledgeKind get kind => kindArg.value;

  BalaurKnowledgeStatus get status => statusArg.value;

  String? get body => bodyArg?.value;

  String? get whenToUse => whenToUseArg?.value;

  int? get importance => importanceArg?.value;

  int? get usedCount => usedCountArg?.value;

  void Function()? get onApprove => onApproveArg?.value;

  void Function()? get onDismiss => onDismissArg?.value;

  void Function()? get onArchive => onArchiveArg?.value;

  void Function()? get onRestore => onRestoreArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    titleArg,
    kindArg,
    statusArg,
    bodyArg,
    whenToUseArg,
    importanceArg,
    usedCountArg,
    onApproveArg,
    onDismissArg,
    onArchiveArg,
    onRestoreArg,
  ];
}
