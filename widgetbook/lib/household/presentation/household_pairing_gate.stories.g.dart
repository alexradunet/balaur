// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'household_pairing_gate.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<HouseholdPairingGate, StoryArgs<HouseholdPairingGate>>;
typedef _Scenario = HouseholdPairingGateScenario;
typedef _Defaults = HouseholdPairingGateDefaults;
typedef _Story = HouseholdPairingGateStory;
typedef _Args = HouseholdPairingGateArgs;
final HouseholdPairingGateComponent =
    Component<HouseholdPairingGate, StoryArgs<HouseholdPairingGate>>(
      name: component.name ?? 'HouseholdPairingGate',
      path: component.path ?? 'household/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Unpaired..$generatedName = 'Unpaired',
        $Connecting..$generatedName = 'Connecting',
        $Failed..$generatedName = 'Failed',
        $Paired..$generatedName = 'Paired',
      ],
    );
typedef HouseholdPairingGateScenario =
    Scenario<HouseholdPairingGate, HouseholdPairingGateArgs>;
typedef HouseholdPairingGateDefaults =
    Defaults<HouseholdPairingGate, HouseholdPairingGateArgs>;

class HouseholdPairingGateStory
    extends Story<HouseholdPairingGate, HouseholdPairingGateArgs> {
  HouseholdPairingGateStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<HouseholdPairingGate, HouseholdPairingGateArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => HouseholdPairingGate(
               key: args.key,
               gateway: args.gateway,
               pairedChild: args.pairedChild,
               onScanInvitation: args.onScanInvitation,
             ),
       );
}

class HouseholdPairingGateArgs extends StoryArgs<HouseholdPairingGate> {
  HouseholdPairingGateArgs({
    Arg<Key?>? key,
    required Arg<HouseholdGateway> gateway,
    required Arg<Widget> pairedChild,
    Arg<Future<String?> Function(BuildContext)?>? onScanInvitation,
  }) : this.keyArg = $initArg('key', key, null),
       this.gatewayArg = $initArg('gateway', gateway, null)!,
       this.pairedChildArg = $initArg('pairedChild', pairedChild, null)!,
       this.onScanInvitationArg = $initArg(
         'onScanInvitation',
         onScanInvitation,
         null,
       );

  HouseholdPairingGateArgs.fixed({
    Key? key,
    required HouseholdGateway gateway,
    required Widget pairedChild,
    Future<String?> Function(BuildContext)? onScanInvitation,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.gatewayArg = $initArg('gateway', Arg.fixed(gateway), null)!,
       this.pairedChildArg = $initArg(
         'pairedChild',
         Arg.fixed(pairedChild),
         null,
       )!,
       this.onScanInvitationArg = $initArg(
         'onScanInvitation',
         onScanInvitation == null ? null : Arg.fixed(onScanInvitation),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<HouseholdGateway> gatewayArg;

  final Arg<Widget> pairedChildArg;

  final Arg<Future<String?> Function(BuildContext)?>? onScanInvitationArg;

  Key? get key => keyArg?.value;

  HouseholdGateway get gateway => gatewayArg.value;

  Widget get pairedChild => pairedChildArg.value;

  Future<String?> Function(BuildContext)? get onScanInvitation =>
      onScanInvitationArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    gatewayArg,
    pairedChildArg,
    onScanInvitationArg,
  ];
}
