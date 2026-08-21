// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'household_invitations_screen.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<
      HouseholdInvitationsScreen,
      StoryArgs<HouseholdInvitationsScreen>
    >;
typedef _Scenario = HouseholdInvitationsScreenScenario;
typedef _Defaults = HouseholdInvitationsScreenDefaults;
typedef _Story = HouseholdInvitationsScreenStory;
typedef _Args = HouseholdInvitationsScreenArgs;
final HouseholdInvitationsScreenComponent =
    Component<
      HouseholdInvitationsScreen,
      StoryArgs<HouseholdInvitationsScreen>
    >(
      name: component.name ?? 'HouseholdInvitationsScreen',
      path: component.path ?? 'household/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Loading..$generatedName = 'Loading',
        $Empty..$generatedName = 'Empty',
        $Existing..$generatedName = 'Existing',
        $Unavailable..$generatedName = 'Unavailable',
        $Canceling..$generatedName = 'Canceling',
        $Created..$generatedName = 'Created',
        $Failed..$generatedName = 'Failed',
        $Forbidden..$generatedName = 'Forbidden',
      ],
    );
typedef HouseholdInvitationsScreenScenario =
    Scenario<HouseholdInvitationsScreen, HouseholdInvitationsScreenArgs>;
typedef HouseholdInvitationsScreenDefaults =
    Defaults<HouseholdInvitationsScreen, HouseholdInvitationsScreenArgs>;

class HouseholdInvitationsScreenStory
    extends Story<HouseholdInvitationsScreen, HouseholdInvitationsScreenArgs> {
  HouseholdInvitationsScreenStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<
      HouseholdInvitationsScreen,
      HouseholdInvitationsScreenArgs
    >?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => HouseholdInvitationsScreen(
               key: args.key,
               gateway: args.gateway,
               session: args.session,
               onBack: args.onBack,
             ),
       );
}

class HouseholdInvitationsScreenArgs
    extends StoryArgs<HouseholdInvitationsScreen> {
  HouseholdInvitationsScreenArgs({
    Arg<Key?>? key,
    required Arg<HouseholdGateway> gateway,
    required Arg<HouseholdSession> session,
    required Arg<void Function()> onBack,
  }) : this.keyArg = $initArg('key', key, null),
       this.gatewayArg = $initArg('gateway', gateway, null)!,
       this.sessionArg = $initArg('session', session, null)!,
       this.onBackArg = $initArg('onBack', onBack, null)!;

  HouseholdInvitationsScreenArgs.fixed({
    Key? key,
    required HouseholdGateway gateway,
    required HouseholdSession session,
    required void Function() onBack,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.gatewayArg = $initArg('gateway', Arg.fixed(gateway), null)!,
       this.sessionArg = $initArg('session', Arg.fixed(session), null)!,
       this.onBackArg = $initArg('onBack', Arg.fixed(onBack), null)!;

  final Arg<Key?>? keyArg;

  final Arg<HouseholdGateway> gatewayArg;

  final Arg<HouseholdSession> sessionArg;

  final Arg<void Function()> onBackArg;

  Key? get key => keyArg?.value;

  HouseholdGateway get gateway => gatewayArg.value;

  HouseholdSession get session => sessionArg.value;

  void Function() get onBack => onBackArg.value;

  @override
  List<Arg?> get list => [keyArg, gatewayArg, sessionArg, onBackArg];
}
