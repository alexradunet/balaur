// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'household_account_screen.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<HouseholdAccountScreen, StoryArgs<HouseholdAccountScreen>>;
typedef _Scenario = HouseholdAccountScreenScenario;
typedef _Defaults = HouseholdAccountScreenDefaults;
typedef _Story = HouseholdAccountScreenStory;
typedef _Args = HouseholdAccountScreenArgs;
final HouseholdAccountScreenComponent =
    Component<HouseholdAccountScreen, StoryArgs<HouseholdAccountScreen>>(
      name: component.name ?? 'HouseholdAccountScreen',
      path: component.path ?? 'household/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [$Administrator..$generatedName = 'Administrator'],
    );
typedef HouseholdAccountScreenScenario =
    Scenario<HouseholdAccountScreen, HouseholdAccountScreenArgs>;
typedef HouseholdAccountScreenDefaults =
    Defaults<HouseholdAccountScreen, HouseholdAccountScreenArgs>;

class HouseholdAccountScreenStory
    extends Story<HouseholdAccountScreen, HouseholdAccountScreenArgs> {
  HouseholdAccountScreenStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<HouseholdAccountScreen, HouseholdAccountScreenArgs>?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => HouseholdAccountScreen(
               key: args.key,
               session: args.session,
               onSignOut: args.onSignOut,
             ),
       );
}

class HouseholdAccountScreenArgs extends StoryArgs<HouseholdAccountScreen> {
  HouseholdAccountScreenArgs({
    Arg<Key?>? key,
    required Arg<HouseholdSession> session,
    required Arg<Future<void> Function()> onSignOut,
  }) : this.keyArg = $initArg('key', key, null),
       this.sessionArg = $initArg('session', session, null)!,
       this.onSignOutArg = $initArg('onSignOut', onSignOut, null)!;

  HouseholdAccountScreenArgs.fixed({
    Key? key,
    required HouseholdSession session,
    required Future<void> Function() onSignOut,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.sessionArg = $initArg('session', Arg.fixed(session), null)!,
       this.onSignOutArg = $initArg('onSignOut', Arg.fixed(onSignOut), null)!;

  final Arg<Key?>? keyArg;

  final Arg<HouseholdSession> sessionArg;

  final Arg<Future<void> Function()> onSignOutArg;

  Key? get key => keyArg?.value;

  HouseholdSession get session => sessionArg.value;

  Future<void> Function() get onSignOut => onSignOutArg.value;

  @override
  List<Arg?> get list => [keyArg, sessionArg, onSignOutArg];
}
