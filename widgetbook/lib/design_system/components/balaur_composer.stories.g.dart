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
final BalaurComposerComponent =
    Component<BalaurComposer, StoryArgs<BalaurComposer>>(
      name: component.name ?? 'BalaurComposer',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays the owner's single seat of action.''',
      stories: [
        $Draft..$generatedName = 'Draft',
        $Decision..$generatedName = 'Decision',
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
               who: args.who,
               avatar: args.avatar,
               placeholder: args.placeholder,
               sendLabel: args.sendLabel,
               tools: args.tools,
               promptLabel: args.promptLabel,
               decision: args.decision,
             ),
       );
}

class BalaurComposerArgs extends StoryArgs<BalaurComposer> {
  BalaurComposerArgs({
    Arg<Key?>? key,
    required Arg<void Function(String)> onSend,
    Arg<TextEditingController?>? controller,
    Arg<String>? who,
    Arg<ImageProvider<Object>?>? avatar,
    Arg<String>? placeholder,
    Arg<String>? sendLabel,
    Arg<List<BalaurComposerTool>>? tools,
    Arg<String?>? promptLabel,
    Arg<Widget?>? decision,
  }) : this.keyArg = $initArg('key', key, null),
       this.onSendArg = $initArg('onSend', onSend, null)!,
       this.controllerArg = $initArg('controller', controller, null),
       this.whoArg = $initArg('who', who, StringArg('You'))!,
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
       this.decisionArg = $initArg('decision', decision, null);

  BalaurComposerArgs.fixed({
    Key? key,
    required void Function(String) onSend,
    TextEditingController? controller,
    String who = 'You',
    ImageProvider<Object>? avatar,
    String placeholder = 'Speak; I am listening.',
    String sendLabel = 'Send',
    List<BalaurComposerTool> tools = const [],
    String? promptLabel = null,
    Widget? decision,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.onSendArg = $initArg('onSend', Arg.fixed(onSend), null)!,
       this.controllerArg = $initArg(
         'controller',
         controller == null ? null : Arg.fixed(controller),
         null,
       ),
       this.whoArg = $initArg('who', Arg.fixed(who), null)!,
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
       this.decisionArg = $initArg(
         'decision',
         decision == null ? null : Arg.fixed(decision),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<void Function(String)> onSendArg;

  final Arg<TextEditingController?>? controllerArg;

  final Arg<String> whoArg;

  final Arg<ImageProvider<Object>?>? avatarArg;

  final Arg<String> placeholderArg;

  final Arg<String> sendLabelArg;

  final Arg<List<BalaurComposerTool>> toolsArg;

  final Arg<String?>? promptLabelArg;

  final Arg<Widget?>? decisionArg;

  Key? get key => keyArg?.value;

  void Function(String) get onSend => onSendArg.value;

  TextEditingController? get controller => controllerArg?.value;

  String get who => whoArg.value;

  ImageProvider<Object>? get avatar => avatarArg?.value;

  String get placeholder => placeholderArg.value;

  String get sendLabel => sendLabelArg.value;

  List<BalaurComposerTool> get tools => toolsArg.value;

  String? get promptLabel => promptLabelArg?.value;

  Widget? get decision => decisionArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    onSendArg,
    controllerArg,
    whoArg,
    avatarArg,
    placeholderArg,
    sendLabelArg,
    toolsArg,
    promptLabelArg,
    decisionArg,
  ];
}
