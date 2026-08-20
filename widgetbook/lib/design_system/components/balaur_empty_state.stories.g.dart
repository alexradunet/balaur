// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_empty_state.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurEmptyState, StoryArgs<BalaurEmptyState>>;
typedef _Scenario = BalaurEmptyStateScenario;
typedef _Defaults = BalaurEmptyStateDefaults;
typedef _Story = BalaurEmptyStateStory;
typedef _Args = BalaurEmptyStateArgs;
final BalaurEmptyStateComponent =
    Component<BalaurEmptyState, StoryArgs<BalaurEmptyState>>(
      name: component.name ?? 'BalaurEmptyState',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays an empty state with an optional action.''',
      stories: [
        $Memory..$generatedName = 'Memory',
        $Quiet..$generatedName = 'Quiet',
      ],
    );
typedef BalaurEmptyStateScenario =
    Scenario<BalaurEmptyState, BalaurEmptyStateArgs>;
typedef BalaurEmptyStateDefaults =
    Defaults<BalaurEmptyState, BalaurEmptyStateArgs>;

class BalaurEmptyStateStory
    extends Story<BalaurEmptyState, BalaurEmptyStateArgs> {
  BalaurEmptyStateStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurEmptyStateArgs? args,
    StoryWidgetBuilder<BalaurEmptyState, BalaurEmptyStateArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurEmptyStateArgs(),
         builder:
             builder ??
             (context, args) => BalaurEmptyState(
               key: args.key,
               title: args.title,
               message: args.message,
               actionLabel: args.actionLabel,
               onAction: args.onAction,
               showCrest: args.showCrest,
             ),
       );
}

class BalaurEmptyStateArgs extends StoryArgs<BalaurEmptyState> {
  BalaurEmptyStateArgs({
    Arg<Key?>? key,
    Arg<String>? title,
    Arg<String?>? message,
    Arg<String?>? actionLabel,
    Arg<void Function()?>? onAction,
    Arg<bool>? showCrest,
  }) : this.keyArg = $initArg('key', key, null),
       this.titleArg = $initArg(
         'title',
         title,
         StringArg('Nothing on the book.'),
       )!,
       this.messageArg = $initArg('message', message, NullableStringArg(null))!,
       this.actionLabelArg = $initArg(
         'actionLabel',
         actionLabel,
         NullableStringArg(null),
       )!,
       this.onActionArg = $initArg('onAction', onAction, null),
       this.showCrestArg = $initArg('showCrest', showCrest, BoolArg(true))!;

  BalaurEmptyStateArgs.fixed({
    Key? key,
    String title = 'Nothing on the book.',
    String? message = null,
    String? actionLabel = null,
    void Function()? onAction,
    bool showCrest = true,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.titleArg = $initArg('title', Arg.fixed(title), null)!,
       this.messageArg = $initArg(
         'message',
         message == null ? null : Arg.fixed(message),
         null,
       ),
       this.actionLabelArg = $initArg(
         'actionLabel',
         actionLabel == null ? null : Arg.fixed(actionLabel),
         null,
       ),
       this.onActionArg = $initArg(
         'onAction',
         onAction == null ? null : Arg.fixed(onAction),
         null,
       ),
       this.showCrestArg = $initArg('showCrest', Arg.fixed(showCrest), null)!;

  final Arg<Key?>? keyArg;

  final Arg<String> titleArg;

  final Arg<String?>? messageArg;

  final Arg<String?>? actionLabelArg;

  final Arg<void Function()?>? onActionArg;

  final Arg<bool> showCrestArg;

  Key? get key => keyArg?.value;

  String get title => titleArg.value;

  String? get message => messageArg?.value;

  String? get actionLabel => actionLabelArg?.value;

  void Function()? get onAction => onActionArg?.value;

  bool get showCrest => showCrestArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    titleArg,
    messageArg,
    actionLabelArg,
    onActionArg,
    showCrestArg,
  ];
}
