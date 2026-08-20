// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_alert.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurAlert, StoryArgs<BalaurAlert>>;
typedef _Scenario = BalaurAlertScenario;
typedef _Defaults = BalaurAlertDefaults;
typedef _Story = BalaurAlertStory;
typedef _Args = BalaurAlertArgs;
final BalaurAlertComponent = Component<BalaurAlert, StoryArgs<BalaurAlert>>(
  name: component.name ?? 'BalaurAlert',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment:
      r'''Displays a persistent callout with a labelled status color.''',
  stories: [
    $Information..$generatedName = 'Information',
    $Warning..$generatedName = 'Warning',
    $Danger..$generatedName = 'Danger',
  ],
);
typedef BalaurAlertScenario = Scenario<BalaurAlert, BalaurAlertArgs>;
typedef BalaurAlertDefaults = Defaults<BalaurAlert, BalaurAlertArgs>;

class BalaurAlertStory extends Story<BalaurAlert, BalaurAlertArgs> {
  BalaurAlertStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurAlertArgs? args,
    StoryWidgetBuilder<BalaurAlert, BalaurAlertArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurAlertArgs(),
         builder:
             builder ??
             (context, args) => BalaurAlert(
               key: args.key,
               message: args.message,
               title: args.title,
               tone: args.tone,
               iconName: args.iconName,
             ),
       );
}

class BalaurAlertArgs extends StoryArgs<BalaurAlert> {
  BalaurAlertArgs({
    Arg<Key?>? key,
    Arg<String>? message,
    Arg<String?>? title,
    Arg<BalaurAlertTone>? tone,
    Arg<String?>? iconName,
  }) : this.keyArg = $initArg('key', key, null),
       this.messageArg = $initArg('message', message, StringArg(''))!,
       this.titleArg = $initArg('title', title, NullableStringArg(null))!,
       this.toneArg = $initArg(
         'tone',
         tone,
         EnumArg<BalaurAlertTone>(
           BalaurAlertTone.info,
           values: BalaurAlertTone.values,
         ),
       )!,
       this.iconNameArg = $initArg(
         'iconName',
         iconName,
         NullableStringArg(null),
       )!;

  BalaurAlertArgs.fixed({
    Key? key,
    String message = '',
    String? title = null,
    BalaurAlertTone tone = BalaurAlertTone.info,
    String? iconName = null,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.messageArg = $initArg('message', Arg.fixed(message), null)!,
       this.titleArg = $initArg(
         'title',
         title == null ? null : Arg.fixed(title),
         null,
       ),
       this.toneArg = $initArg('tone', Arg.fixed(tone), null)!,
       this.iconNameArg = $initArg(
         'iconName',
         iconName == null ? null : Arg.fixed(iconName),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<String> messageArg;

  final Arg<String?>? titleArg;

  final Arg<BalaurAlertTone> toneArg;

  final Arg<String?>? iconNameArg;

  Key? get key => keyArg?.value;

  String get message => messageArg.value;

  String? get title => titleArg?.value;

  BalaurAlertTone get tone => toneArg.value;

  String? get iconName => iconNameArg?.value;

  @override
  List<Arg?> get list => [keyArg, messageArg, titleArg, toneArg, iconNameArg];
}
