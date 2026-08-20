// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_dialogue_choices.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<BalaurDialogueChoices, StoryArgs<BalaurDialogueChoices>>;
typedef _Scenario = BalaurDialogueChoicesScenario;
typedef _Defaults = BalaurDialogueChoicesDefaults;
typedef _Story = BalaurDialogueChoicesStory;
typedef _Args = BalaurDialogueChoicesArgs;
final BalaurDialogueChoicesComponent =
    Component<BalaurDialogueChoices, StoryArgs<BalaurDialogueChoices>>(
      name: component.name ?? 'BalaurDialogueChoices',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment:
          r'''Displays a numbered dialogue decision in the owner's voice.''',
      stories: [
        $MemoryDecision..$generatedName = 'MemoryDecision',
        $TaskDecision..$generatedName = 'TaskDecision',
      ],
    );
typedef BalaurDialogueChoicesScenario =
    Scenario<BalaurDialogueChoices, BalaurDialogueChoicesArgs>;
typedef BalaurDialogueChoicesDefaults =
    Defaults<BalaurDialogueChoices, BalaurDialogueChoicesArgs>;

class BalaurDialogueChoicesStory
    extends Story<BalaurDialogueChoices, BalaurDialogueChoicesArgs> {
  BalaurDialogueChoicesStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurDialogueChoices, BalaurDialogueChoicesArgs>?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurDialogueChoices(
               key: args.key,
               choices: args.choices,
               onPick: args.onPick,
               kicker: args.kicker,
               who: args.who,
               avatar: args.avatar,
             ),
       );
}

class BalaurDialogueChoicesArgs extends StoryArgs<BalaurDialogueChoices> {
  BalaurDialogueChoicesArgs({
    Arg<Key?>? key,
    required Arg<List<BalaurDialogueChoice>> choices,
    required Arg<void Function(int)> onPick,
    Arg<String>? kicker,
    Arg<String>? who,
    Arg<ImageProvider<Object>?>? avatar,
  }) : this.keyArg = $initArg('key', key, null),
       this.choicesArg = $initArg('choices', choices, null)!,
       this.onPickArg = $initArg('onPick', onPick, null)!,
       this.kickerArg = $initArg('kicker', kicker, StringArg('Your word'))!,
       this.whoArg = $initArg('who', who, StringArg('You'))!,
       this.avatarArg = $initArg('avatar', avatar, null);

  BalaurDialogueChoicesArgs.fixed({
    Key? key,
    required List<BalaurDialogueChoice> choices,
    required void Function(int) onPick,
    String kicker = 'Your word',
    String who = 'You',
    ImageProvider<Object>? avatar,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.choicesArg = $initArg('choices', Arg.fixed(choices), null)!,
       this.onPickArg = $initArg('onPick', Arg.fixed(onPick), null)!,
       this.kickerArg = $initArg('kicker', Arg.fixed(kicker), null)!,
       this.whoArg = $initArg('who', Arg.fixed(who), null)!,
       this.avatarArg = $initArg(
         'avatar',
         avatar == null ? null : Arg.fixed(avatar),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<List<BalaurDialogueChoice>> choicesArg;

  final Arg<void Function(int)> onPickArg;

  final Arg<String> kickerArg;

  final Arg<String> whoArg;

  final Arg<ImageProvider<Object>?>? avatarArg;

  Key? get key => keyArg?.value;

  List<BalaurDialogueChoice> get choices => choicesArg.value;

  void Function(int) get onPick => onPickArg.value;

  String get kicker => kickerArg.value;

  String get who => whoArg.value;

  ImageProvider<Object>? get avatar => avatarArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    choicesArg,
    onPickArg,
    kickerArg,
    whoArg,
    avatarArg,
  ];
}
