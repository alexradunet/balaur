// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_message_draft.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<BalaurMessageDraft, StoryArgs<BalaurMessageDraft>>;
typedef _Scenario = BalaurMessageDraftScenario;
typedef _Defaults = BalaurMessageDraftDefaults;
typedef _Story = BalaurMessageDraftStory;
typedef _Args = BalaurMessageDraftArgs;
final BalaurMessageDraftComponent =
    Component<BalaurMessageDraft, StoryArgs<BalaurMessageDraft>>(
      name: component.name ?? 'BalaurMessageDraft',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment:
          r'''Displays the owner's unsent message in the dialogue column.''',
      stories: [$Unsent..$generatedName = 'Unsent'],
    );
typedef BalaurMessageDraftScenario =
    Scenario<BalaurMessageDraft, BalaurMessageDraftArgs>;
typedef BalaurMessageDraftDefaults =
    Defaults<BalaurMessageDraft, BalaurMessageDraftArgs>;

class BalaurMessageDraftStory
    extends Story<BalaurMessageDraft, BalaurMessageDraftArgs> {
  BalaurMessageDraftStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurMessageDraft, BalaurMessageDraftArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurMessageDraft(
               key: args.key,
               onSend: args.onSend,
               controller: args.controller,
               who: args.who,
               avatar: args.avatar,
               placeholder: args.placeholder,
               sendLabel: args.sendLabel,
             ),
       );
}

class BalaurMessageDraftArgs extends StoryArgs<BalaurMessageDraft> {
  BalaurMessageDraftArgs({
    Arg<Key?>? key,
    required Arg<void Function(String)> onSend,
    Arg<TextEditingController?>? controller,
    Arg<String>? who,
    Arg<ImageProvider<Object>?>? avatar,
    Arg<String>? placeholder,
    Arg<String>? sendLabel,
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
       this.sendLabelArg = $initArg('sendLabel', sendLabel, StringArg('Send'))!;

  BalaurMessageDraftArgs.fixed({
    Key? key,
    required void Function(String) onSend,
    TextEditingController? controller,
    String who = 'You',
    ImageProvider<Object>? avatar,
    String placeholder = 'Speak; I am listening.',
    String sendLabel = 'Send',
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
       this.sendLabelArg = $initArg('sendLabel', Arg.fixed(sendLabel), null)!;

  final Arg<Key?>? keyArg;

  final Arg<void Function(String)> onSendArg;

  final Arg<TextEditingController?>? controllerArg;

  final Arg<String> whoArg;

  final Arg<ImageProvider<Object>?>? avatarArg;

  final Arg<String> placeholderArg;

  final Arg<String> sendLabelArg;

  Key? get key => keyArg?.value;

  void Function(String) get onSend => onSendArg.value;

  TextEditingController? get controller => controllerArg?.value;

  String get who => whoArg.value;

  ImageProvider<Object>? get avatar => avatarArg?.value;

  String get placeholder => placeholderArg.value;

  String get sendLabel => sendLabelArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    onSendArg,
    controllerArg,
    whoArg,
    avatarArg,
    placeholderArg,
    sendLabelArg,
  ];
}
