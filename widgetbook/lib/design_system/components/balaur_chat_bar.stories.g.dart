// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_chat_bar.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurChatBar, StoryArgs<BalaurChatBar>>;
typedef _Scenario = BalaurChatBarScenario;
typedef _Defaults = BalaurChatBarDefaults;
typedef _Story = BalaurChatBarStory;
typedef _Args = BalaurChatBarArgs;
final BalaurChatBarComponent =
    Component<BalaurChatBar, StoryArgs<BalaurChatBar>>(
      name: component.name ?? 'BalaurChatBar',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays a page-level wood input ledge.''',
      stories: [$Capture..$generatedName = 'Capture'],
    );
typedef BalaurChatBarScenario = Scenario<BalaurChatBar, BalaurChatBarArgs>;
typedef BalaurChatBarDefaults = Defaults<BalaurChatBar, BalaurChatBarArgs>;

class BalaurChatBarStory extends Story<BalaurChatBar, BalaurChatBarArgs> {
  BalaurChatBarStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurChatBar, BalaurChatBarArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurChatBar(
               key: args.key,
               onSend: args.onSend,
               controller: args.controller,
               placeholder: args.placeholder,
               sendLabel: args.sendLabel,
               above: args.above,
             ),
       );
}

class BalaurChatBarArgs extends StoryArgs<BalaurChatBar> {
  BalaurChatBarArgs({
    Arg<Key?>? key,
    required Arg<void Function(String)> onSend,
    Arg<TextEditingController?>? controller,
    Arg<String>? placeholder,
    Arg<String>? sendLabel,
    Arg<Widget?>? above,
  }) : this.keyArg = $initArg('key', key, null),
       this.onSendArg = $initArg('onSend', onSend, null)!,
       this.controllerArg = $initArg('controller', controller, null),
       this.placeholderArg = $initArg(
         'placeholder',
         placeholder,
         StringArg('Speak; I am listening.'),
       )!,
       this.sendLabelArg = $initArg('sendLabel', sendLabel, StringArg('Send'))!,
       this.aboveArg = $initArg('above', above, null);

  BalaurChatBarArgs.fixed({
    Key? key,
    required void Function(String) onSend,
    TextEditingController? controller,
    String placeholder = 'Speak; I am listening.',
    String sendLabel = 'Send',
    Widget? above,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.onSendArg = $initArg('onSend', Arg.fixed(onSend), null)!,
       this.controllerArg = $initArg(
         'controller',
         controller == null ? null : Arg.fixed(controller),
         null,
       ),
       this.placeholderArg = $initArg(
         'placeholder',
         Arg.fixed(placeholder),
         null,
       )!,
       this.sendLabelArg = $initArg('sendLabel', Arg.fixed(sendLabel), null)!,
       this.aboveArg = $initArg(
         'above',
         above == null ? null : Arg.fixed(above),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<void Function(String)> onSendArg;

  final Arg<TextEditingController?>? controllerArg;

  final Arg<String> placeholderArg;

  final Arg<String> sendLabelArg;

  final Arg<Widget?>? aboveArg;

  Key? get key => keyArg?.value;

  void Function(String) get onSend => onSendArg.value;

  TextEditingController? get controller => controllerArg?.value;

  String get placeholder => placeholderArg.value;

  String get sendLabel => sendLabelArg.value;

  Widget? get above => aboveArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    onSendArg,
    controllerArg,
    placeholderArg,
    sendLabelArg,
    aboveArg,
  ];
}
