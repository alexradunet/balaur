// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'household_pairing_view.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<HouseholdPairingView, StoryArgs<HouseholdPairingView>>;
typedef _Scenario = HouseholdPairingViewScenario;
typedef _Defaults = HouseholdPairingViewDefaults;
typedef _Story = HouseholdPairingViewStory;
typedef _Args = HouseholdPairingViewArgs;
final HouseholdPairingViewComponent =
    Component<HouseholdPairingView, StoryArgs<HouseholdPairingView>>(
      name: component.name ?? 'HouseholdPairingView',
      path: component.path ?? 'household/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Manual..$generatedName = 'Manual',
        $Scanned..$generatedName = 'Scanned',
      ],
    );
typedef HouseholdPairingViewScenario =
    Scenario<HouseholdPairingView, HouseholdPairingViewArgs>;
typedef HouseholdPairingViewDefaults =
    Defaults<HouseholdPairingView, HouseholdPairingViewArgs>;

class HouseholdPairingViewStory
    extends Story<HouseholdPairingView, HouseholdPairingViewArgs> {
  HouseholdPairingViewStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<HouseholdPairingView, HouseholdPairingViewArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => HouseholdPairingView(
               key: args.key,
               status: args.status,
               onPair: args.onPair,
               onRedeemInvitation: args.onRedeemInvitation,
               onSignOut: args.onSignOut,
               session: args.session,
               errorMessage: args.errorMessage,
               onEnter: args.onEnter,
               onScanInvitation: args.onScanInvitation,
               initialMode: args.initialMode,
             ),
       );
}

class HouseholdPairingViewArgs extends StoryArgs<HouseholdPairingView> {
  HouseholdPairingViewArgs({
    Arg<Key?>? key,
    Arg<HouseholdPairingStatus>? status,
    required Arg<
      Future<void> Function({
        required String email,
        required String password,
        required HouseholdServerAddress serverAddress,
      })
    >
    onPair,
    required Arg<
      Future<void> Function({
        required String displayName,
        required String email,
        required HouseholdInvitationPayload invitation,
        required String password,
      })
    >
    onRedeemInvitation,
    required Arg<Future<void> Function()> onSignOut,
    Arg<HouseholdSession?>? session,
    Arg<String?>? errorMessage,
    Arg<void Function()?>? onEnter,
    Arg<Future<String?> Function(BuildContext)?>? onScanInvitation,
    Arg<HouseholdPairingMode>? initialMode,
  }) : this.keyArg = $initArg('key', key, null),
       this.statusArg = $initArg(
         'status',
         status,
         EnumArg<HouseholdPairingStatus>(
           HouseholdPairingStatus.connecting,
           values: HouseholdPairingStatus.values,
         ),
       )!,
       this.onPairArg = $initArg('onPair', onPair, null)!,
       this.onRedeemInvitationArg = $initArg(
         'onRedeemInvitation',
         onRedeemInvitation,
         null,
       )!,
       this.onSignOutArg = $initArg('onSignOut', onSignOut, null)!,
       this.sessionArg = $initArg('session', session, null),
       this.errorMessageArg = $initArg(
         'errorMessage',
         errorMessage,
         NullableStringArg(null),
       )!,
       this.onEnterArg = $initArg('onEnter', onEnter, null),
       this.onScanInvitationArg = $initArg(
         'onScanInvitation',
         onScanInvitation,
         null,
       ),
       this.initialModeArg = $initArg(
         'initialMode',
         initialMode,
         EnumArg<HouseholdPairingMode>(
           HouseholdPairingMode.credentials,
           values: HouseholdPairingMode.values,
         ),
       )!;

  HouseholdPairingViewArgs.fixed({
    Key? key,
    HouseholdPairingStatus status = HouseholdPairingStatus.connecting,
    required Future<void> Function({
      required String email,
      required String password,
      required HouseholdServerAddress serverAddress,
    })
    onPair,
    required Future<void> Function({
      required String displayName,
      required String email,
      required HouseholdInvitationPayload invitation,
      required String password,
    })
    onRedeemInvitation,
    required Future<void> Function() onSignOut,
    HouseholdSession? session,
    String? errorMessage = null,
    void Function()? onEnter,
    Future<String?> Function(BuildContext)? onScanInvitation,
    HouseholdPairingMode initialMode = HouseholdPairingMode.credentials,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.statusArg = $initArg('status', Arg.fixed(status), null)!,
       this.onPairArg = $initArg('onPair', Arg.fixed(onPair), null)!,
       this.onRedeemInvitationArg = $initArg(
         'onRedeemInvitation',
         Arg.fixed(onRedeemInvitation),
         null,
       )!,
       this.onSignOutArg = $initArg('onSignOut', Arg.fixed(onSignOut), null)!,
       this.sessionArg = $initArg(
         'session',
         session == null ? null : Arg.fixed(session),
         null,
       ),
       this.errorMessageArg = $initArg(
         'errorMessage',
         errorMessage == null ? null : Arg.fixed(errorMessage),
         null,
       ),
       this.onEnterArg = $initArg(
         'onEnter',
         onEnter == null ? null : Arg.fixed(onEnter),
         null,
       ),
       this.onScanInvitationArg = $initArg(
         'onScanInvitation',
         onScanInvitation == null ? null : Arg.fixed(onScanInvitation),
         null,
       ),
       this.initialModeArg = $initArg(
         'initialMode',
         Arg.fixed(initialMode),
         null,
       )!;

  final Arg<Key?>? keyArg;

  final Arg<HouseholdPairingStatus> statusArg;

  final Arg<
    Future<void> Function({
      required String email,
      required String password,
      required HouseholdServerAddress serverAddress,
    })
  >
  onPairArg;

  final Arg<
    Future<void> Function({
      required String displayName,
      required String email,
      required HouseholdInvitationPayload invitation,
      required String password,
    })
  >
  onRedeemInvitationArg;

  final Arg<Future<void> Function()> onSignOutArg;

  final Arg<HouseholdSession?>? sessionArg;

  final Arg<String?>? errorMessageArg;

  final Arg<void Function()?>? onEnterArg;

  final Arg<Future<String?> Function(BuildContext)?>? onScanInvitationArg;

  final Arg<HouseholdPairingMode> initialModeArg;

  Key? get key => keyArg?.value;

  HouseholdPairingStatus get status => statusArg.value;

  Future<void> Function({
    required String email,
    required String password,
    required HouseholdServerAddress serverAddress,
  })
  get onPair => onPairArg.value;

  Future<void> Function({
    required String displayName,
    required String email,
    required HouseholdInvitationPayload invitation,
    required String password,
  })
  get onRedeemInvitation => onRedeemInvitationArg.value;

  Future<void> Function() get onSignOut => onSignOutArg.value;

  HouseholdSession? get session => sessionArg?.value;

  String? get errorMessage => errorMessageArg?.value;

  void Function()? get onEnter => onEnterArg?.value;

  Future<String?> Function(BuildContext)? get onScanInvitation =>
      onScanInvitationArg?.value;

  HouseholdPairingMode get initialMode => initialModeArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    statusArg,
    onPairArg,
    onRedeemInvitationArg,
    onSignOutArg,
    sessionArg,
    errorMessageArg,
    onEnterArg,
    onScanInvitationArg,
    initialModeArg,
  ];
}
