// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_toast.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurToast, StoryArgs<BalaurToast>>;
typedef _Scenario = BalaurToastScenario;
typedef _Defaults = BalaurToastDefaults;
typedef _Story = BalaurToastStory;
typedef _Args = BalaurToastArgs;
final BalaurToastComponent = Component<BalaurToast, StoryArgs<BalaurToast>>(
  name: component.name ?? 'BalaurToast',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a short parchment status message.''',
  stories: [
    $Success..$generatedName = 'Success',
    $Information..$generatedName = 'Information',
    $Warning..$generatedName = 'Warning',
  ],
);
typedef BalaurToastScenario = Scenario<BalaurToast, BalaurToastArgs>;
typedef BalaurToastDefaults = Defaults<BalaurToast, BalaurToastArgs>;

class BalaurToastStory extends Story<BalaurToast, BalaurToastArgs> {
  BalaurToastStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurToastArgs? args,
    StoryWidgetBuilder<BalaurToast, BalaurToastArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurToastArgs(),
         builder:
             builder ??
             (context, args) => BalaurToast(
               key: args.key,
               message: args.message,
               tone: args.tone,
             ),
       );
}

class BalaurToastArgs extends StoryArgs<BalaurToast> {
  BalaurToastArgs({
    Arg<Key?>? key,
    Arg<String>? message,
    Arg<BalaurToastTone>? tone,
  }) : this.keyArg = $initArg('key', key, null),
       this.messageArg = $initArg('message', message, StringArg(''))!,
       this.toneArg = $initArg(
         'tone',
         tone,
         EnumArg<BalaurToastTone>(
           BalaurToastTone.info,
           values: BalaurToastTone.values,
         ),
       )!;

  BalaurToastArgs.fixed({
    Key? key,
    String message = '',
    BalaurToastTone tone = BalaurToastTone.info,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.messageArg = $initArg('message', Arg.fixed(message), null)!,
       this.toneArg = $initArg('tone', Arg.fixed(tone), null)!;

  final Arg<Key?>? keyArg;

  final Arg<String> messageArg;

  final Arg<BalaurToastTone> toneArg;

  Key? get key => keyArg?.value;

  String get message => messageArg.value;

  BalaurToastTone get tone => toneArg.value;

  @override
  List<Arg?> get list => [keyArg, messageArg, toneArg];
}
