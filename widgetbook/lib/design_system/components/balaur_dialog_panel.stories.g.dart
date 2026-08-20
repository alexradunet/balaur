// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_dialog_panel.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurDialogPanel, StoryArgs<BalaurDialogPanel>>;
typedef _Scenario = BalaurDialogPanelScenario;
typedef _Defaults = BalaurDialogPanelDefaults;
typedef _Story = BalaurDialogPanelStory;
typedef _Args = BalaurDialogPanelArgs;
final BalaurDialogPanelComponent =
    Component<BalaurDialogPanel, StoryArgs<BalaurDialogPanel>>(
      name: component.name ?? 'BalaurDialogPanel',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays an ornate decision panel.''',
      stories: [$Confirm..$generatedName = 'Confirm'],
    );
typedef BalaurDialogPanelScenario =
    Scenario<BalaurDialogPanel, BalaurDialogPanelArgs>;
typedef BalaurDialogPanelDefaults =
    Defaults<BalaurDialogPanel, BalaurDialogPanelArgs>;

class BalaurDialogPanelStory
    extends Story<BalaurDialogPanel, BalaurDialogPanelArgs> {
  BalaurDialogPanelStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurDialogPanelArgs? args,
    StoryWidgetBuilder<BalaurDialogPanel, BalaurDialogPanelArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurDialogPanelArgs(),
         builder:
             builder ??
             (context, args) => BalaurDialogPanel(
               key: args.key,
               title: args.title,
               body: args.body,
               kicker: args.kicker,
               actions: args.actions,
             ),
       );
}

class BalaurDialogPanelArgs extends StoryArgs<BalaurDialogPanel> {
  BalaurDialogPanelArgs({
    Arg<Key?>? key,
    Arg<String>? title,
    Arg<String>? body,
    Arg<String?>? kicker,
    Arg<List<BalaurDialogAction>>? actions,
  }) : this.keyArg = $initArg('key', key, null),
       this.titleArg = $initArg('title', title, StringArg(''))!,
       this.bodyArg = $initArg('body', body, StringArg(''))!,
       this.kickerArg = $initArg('kicker', kicker, NullableStringArg(null))!,
       this.actionsArg = $initArg('actions', actions, ConstArg(const []))!;

  BalaurDialogPanelArgs.fixed({
    Key? key,
    String title = '',
    String body = '',
    String? kicker = null,
    List<BalaurDialogAction> actions = const [],
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.titleArg = $initArg('title', Arg.fixed(title), null)!,
       this.bodyArg = $initArg('body', Arg.fixed(body), null)!,
       this.kickerArg = $initArg(
         'kicker',
         kicker == null ? null : Arg.fixed(kicker),
         null,
       ),
       this.actionsArg = $initArg('actions', Arg.fixed(actions), null)!;

  final Arg<Key?>? keyArg;

  final Arg<String> titleArg;

  final Arg<String> bodyArg;

  final Arg<String?>? kickerArg;

  final Arg<List<BalaurDialogAction>> actionsArg;

  Key? get key => keyArg?.value;

  String get title => titleArg.value;

  String get body => bodyArg.value;

  String? get kicker => kickerArg?.value;

  List<BalaurDialogAction> get actions => actionsArg.value;

  @override
  List<Arg?> get list => [keyArg, titleArg, bodyArg, kickerArg, actionsArg];
}
