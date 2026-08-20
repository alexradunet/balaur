// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_guardian_card.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<BalaurGuardianCard, StoryArgs<BalaurGuardianCard>>;
typedef _Scenario = BalaurGuardianCardScenario;
typedef _Defaults = BalaurGuardianCardDefaults;
typedef _Story = BalaurGuardianCardStory;
typedef _Args = BalaurGuardianCardArgs;
final BalaurGuardianCardComponent =
    Component<BalaurGuardianCard, StoryArgs<BalaurGuardianCard>>(
      name: component.name ?? 'BalaurGuardianCard',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays an operating-system access decision.''',
      stories: [
        $ReadFiles..$generatedName = 'ReadFiles',
        $RunCommand..$generatedName = 'RunCommand',
      ],
    );
typedef BalaurGuardianCardScenario =
    Scenario<BalaurGuardianCard, BalaurGuardianCardArgs>;
typedef BalaurGuardianCardDefaults =
    Defaults<BalaurGuardianCard, BalaurGuardianCardArgs>;

class BalaurGuardianCardStory
    extends Story<BalaurGuardianCard, BalaurGuardianCardArgs> {
  BalaurGuardianCardStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurGuardianCard, BalaurGuardianCardArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurGuardianCard(
               key: args.key,
               title: args.title,
               onAllowOnce: args.onAllowOnce,
               onAllowAlways: args.onAllowAlways,
               onDeny: args.onDeny,
               kicker: args.kicker,
               detail: args.detail,
               scope: args.scope,
             ),
       );
}

class BalaurGuardianCardArgs extends StoryArgs<BalaurGuardianCard> {
  BalaurGuardianCardArgs({
    Arg<Key?>? key,
    Arg<String>? title,
    required Arg<void Function()> onAllowOnce,
    required Arg<void Function()> onAllowAlways,
    required Arg<void Function()> onDeny,
    Arg<String>? kicker,
    Arg<String?>? detail,
    Arg<String?>? scope,
  }) : this.keyArg = $initArg('key', key, null),
       this.titleArg = $initArg('title', title, StringArg(''))!,
       this.onAllowOnceArg = $initArg('onAllowOnce', onAllowOnce, null)!,
       this.onAllowAlwaysArg = $initArg('onAllowAlways', onAllowAlways, null)!,
       this.onDenyArg = $initArg('onDeny', onDeny, null)!,
       this.kickerArg = $initArg('kicker', kicker, StringArg('OS access'))!,
       this.detailArg = $initArg('detail', detail, NullableStringArg(null))!,
       this.scopeArg = $initArg('scope', scope, NullableStringArg(null))!;

  BalaurGuardianCardArgs.fixed({
    Key? key,
    String title = '',
    required void Function() onAllowOnce,
    required void Function() onAllowAlways,
    required void Function() onDeny,
    String kicker = 'OS access',
    String? detail = null,
    String? scope = null,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.titleArg = $initArg('title', Arg.fixed(title), null)!,
       this.onAllowOnceArg = $initArg(
         'onAllowOnce',
         Arg.fixed(onAllowOnce),
         null,
       )!,
       this.onAllowAlwaysArg = $initArg(
         'onAllowAlways',
         Arg.fixed(onAllowAlways),
         null,
       )!,
       this.onDenyArg = $initArg('onDeny', Arg.fixed(onDeny), null)!,
       this.kickerArg = $initArg('kicker', Arg.fixed(kicker), null)!,
       this.detailArg = $initArg(
         'detail',
         detail == null ? null : Arg.fixed(detail),
         null,
       ),
       this.scopeArg = $initArg(
         'scope',
         scope == null ? null : Arg.fixed(scope),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<String> titleArg;

  final Arg<void Function()> onAllowOnceArg;

  final Arg<void Function()> onAllowAlwaysArg;

  final Arg<void Function()> onDenyArg;

  final Arg<String> kickerArg;

  final Arg<String?>? detailArg;

  final Arg<String?>? scopeArg;

  Key? get key => keyArg?.value;

  String get title => titleArg.value;

  void Function() get onAllowOnce => onAllowOnceArg.value;

  void Function() get onAllowAlways => onAllowAlwaysArg.value;

  void Function() get onDeny => onDenyArg.value;

  String get kicker => kickerArg.value;

  String? get detail => detailArg?.value;

  String? get scope => scopeArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    titleArg,
    onAllowOnceArg,
    onAllowAlwaysArg,
    onDenyArg,
    kickerArg,
    detailArg,
    scopeArg,
  ];
}
