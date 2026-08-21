// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_composer.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurComposer, StoryArgs<BalaurComposer>>;
typedef _Scenario = BalaurComposerScenario;
typedef _Defaults = BalaurComposerDefaults;
typedef _Story = BalaurComposerStory;
typedef _Args = BalaurComposerArgs;
typedef _ChoicesScenario = BalaurComposerChoicesScenario;
typedef _ChoicesDefaults = BalaurComposerChoicesDefaults;
typedef _ChoicesStory = BalaurComposerChoicesStory;
typedef _ChoicesArgs = BalaurComposerChoicesArgs;
final BalaurComposerComponent =
    Component<BalaurComposer, StoryArgs<BalaurComposer>>(
      name: component.name ?? 'BalaurComposer',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays the owner's single seat of action.''',
      stories: [
        $Draft..$generatedName = 'Draft',
        $Responding..$generatedName = 'Responding',
        $Disabled..$generatedName = 'Disabled',
        $DialogueChoices..$generatedName = 'DialogueChoices',
      ],
    );
typedef BalaurComposerScenario = Scenario<BalaurComposer, BalaurComposerArgs>;
typedef BalaurComposerDefaults = Defaults<BalaurComposer, BalaurComposerArgs>;

class BalaurComposerStory extends Story<BalaurComposer, BalaurComposerArgs> {
  BalaurComposerStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurComposer, BalaurComposerArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurComposer(
               key: args.key,
               onSend: args.onSend,
               controller: args.controller,
               avatar: args.avatar,
               placeholder: args.placeholder,
               sendLabel: args.sendLabel,
               tools: args.tools,
               promptLabel: args.promptLabel,
               enabled: args.enabled,
               responding: args.responding,
               onStop: args.onStop,
             ),
       );
}

class BalaurComposerArgs extends StoryArgs<BalaurComposer> {
  BalaurComposerArgs({
    Arg<Key?>? key,
    required Arg<void Function(String)> onSend,
    Arg<TextEditingController?>? controller,
    Arg<ImageProvider<Object>?>? avatar,
    Arg<String>? placeholder,
    Arg<String>? sendLabel,
    Arg<List<BalaurComposerTool>>? tools,
    Arg<String?>? promptLabel,
    Arg<bool>? enabled,
    Arg<bool>? responding,
    Arg<void Function()?>? onStop,
  }) : this.keyArg = $initArg('key', key, null),
       this.onSendArg = $initArg('onSend', onSend, null)!,
       this.controllerArg = $initArg('controller', controller, null),
       this.avatarArg = $initArg('avatar', avatar, null),
       this.placeholderArg = $initArg(
         'placeholder',
         placeholder,
         StringArg('Speak; I am listening.'),
       )!,
       this.sendLabelArg = $initArg('sendLabel', sendLabel, StringArg('Send'))!,
       this.toolsArg = $initArg('tools', tools, ConstArg(const []))!,
       this.promptLabelArg = $initArg(
         'promptLabel',
         promptLabel,
         NullableStringArg(null),
       )!,
       this.enabledArg = $initArg('enabled', enabled, BoolArg(true))!,
       this.respondingArg = $initArg('responding', responding, BoolArg(false))!,
       this.onStopArg = $initArg('onStop', onStop, null);

  BalaurComposerArgs.fixed({
    Key? key,
    required void Function(String) onSend,
    TextEditingController? controller,
    ImageProvider<Object>? avatar,
    String placeholder = 'Speak; I am listening.',
    String sendLabel = 'Send',
    List<BalaurComposerTool> tools = const [],
    String? promptLabel = null,
    bool enabled = true,
    bool responding = false,
    void Function()? onStop,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.onSendArg = $initArg('onSend', Arg.fixed(onSend), null)!,
       this.controllerArg = $initArg(
         'controller',
         controller == null ? null : Arg.fixed(controller),
         null,
       ),
       this.avatarArg = $initArg(
         'avatar',
         avatar == null ? null : Arg.fixed(avatar),
         null,
       ),
       this.placeholderArg = $initArg(
         'placeholder',
         Arg.fixed(placeholder),
         null,
       )!,
       this.sendLabelArg = $initArg('sendLabel', Arg.fixed(sendLabel), null)!,
       this.toolsArg = $initArg('tools', Arg.fixed(tools), null)!,
       this.promptLabelArg = $initArg(
         'promptLabel',
         promptLabel == null ? null : Arg.fixed(promptLabel),
         null,
       ),
       this.enabledArg = $initArg('enabled', Arg.fixed(enabled), null)!,
       this.respondingArg = $initArg(
         'responding',
         Arg.fixed(responding),
         null,
       )!,
       this.onStopArg = $initArg(
         'onStop',
         onStop == null ? null : Arg.fixed(onStop),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<void Function(String)> onSendArg;

  final Arg<TextEditingController?>? controllerArg;

  final Arg<ImageProvider<Object>?>? avatarArg;

  final Arg<String> placeholderArg;

  final Arg<String> sendLabelArg;

  final Arg<List<BalaurComposerTool>> toolsArg;

  final Arg<String?>? promptLabelArg;

  final Arg<bool> enabledArg;

  final Arg<bool> respondingArg;

  final Arg<void Function()?>? onStopArg;

  Key? get key => keyArg?.value;

  void Function(String) get onSend => onSendArg.value;

  TextEditingController? get controller => controllerArg?.value;

  ImageProvider<Object>? get avatar => avatarArg?.value;

  String get placeholder => placeholderArg.value;

  String get sendLabel => sendLabelArg.value;

  List<BalaurComposerTool> get tools => toolsArg.value;

  String? get promptLabel => promptLabelArg?.value;

  bool get enabled => enabledArg.value;

  bool get responding => respondingArg.value;

  void Function()? get onStop => onStopArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    onSendArg,
    controllerArg,
    avatarArg,
    placeholderArg,
    sendLabelArg,
    toolsArg,
    promptLabelArg,
    enabledArg,
    respondingArg,
    onStopArg,
  ];
}

typedef BalaurComposerChoicesScenario =
    Scenario<BalaurComposer, BalaurComposerChoicesArgs>;
typedef BalaurComposerChoicesDefaults =
    Defaults<BalaurComposer, BalaurComposerChoicesArgs>;

class BalaurComposerChoicesStory
    extends Story<BalaurComposer, BalaurComposerChoicesArgs> {
  BalaurComposerChoicesStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurComposer, BalaurComposerChoicesArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurComposer.choices(
               key: args.key,
               choices: args.choices,
               onPick: args.onPick,
               avatar: args.avatar,
               promptLabel: args.promptLabel,
             ),
       );
}

class BalaurComposerChoicesArgs extends StoryArgs<BalaurComposer> {
  BalaurComposerChoicesArgs({
    Arg<Key?>? key,
    required Arg<List<BalaurDialogueChoice>> choices,
    required Arg<void Function(int)> onPick,
    Arg<ImageProvider<Object>?>? avatar,
    Arg<String?>? promptLabel,
  }) : this.keyArg = $initArg('key', key, null),
       this.choicesArg = $initArg('choices', choices, null)!,
       this.onPickArg = $initArg('onPick', onPick, null)!,
       this.avatarArg = $initArg('avatar', avatar, null),
       this.promptLabelArg = $initArg(
         'promptLabel',
         promptLabel,
         NullableStringArg('Your word'),
       )!;

  BalaurComposerChoicesArgs.fixed({
    Key? key,
    required List<BalaurDialogueChoice> choices,
    required void Function(int) onPick,
    ImageProvider<Object>? avatar,
    String? promptLabel = 'Your word',
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.choicesArg = $initArg('choices', Arg.fixed(choices), null)!,
       this.onPickArg = $initArg('onPick', Arg.fixed(onPick), null)!,
       this.avatarArg = $initArg(
         'avatar',
         avatar == null ? null : Arg.fixed(avatar),
         null,
       ),
       this.promptLabelArg = $initArg(
         'promptLabel',
         promptLabel == null ? null : Arg.fixed(promptLabel),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<List<BalaurDialogueChoice>> choicesArg;

  final Arg<void Function(int)> onPickArg;

  final Arg<ImageProvider<Object>?>? avatarArg;

  final Arg<String?>? promptLabelArg;

  Key? get key => keyArg?.value;

  List<BalaurDialogueChoice> get choices => choicesArg.value;

  void Function(int) get onPick => onPickArg.value;

  ImageProvider<Object>? get avatar => avatarArg?.value;

  String? get promptLabel => promptLabelArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    choicesArg,
    onPickArg,
    avatarArg,
    promptLabelArg,
  ];
}
