// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'household_archive_screen.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<HouseholdArchiveScreen, StoryArgs<HouseholdArchiveScreen>>;
typedef _Scenario = HouseholdArchiveScreenScenario;
typedef _Defaults = HouseholdArchiveScreenDefaults;
typedef _Story = HouseholdArchiveScreenStory;
typedef _Args = HouseholdArchiveScreenArgs;
final HouseholdArchiveScreenComponent =
    Component<HouseholdArchiveScreen, StoryArgs<HouseholdArchiveScreen>>(
      name: component.name ?? 'HouseholdArchiveScreen',
      path: component.path ?? 'household/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Idle..$generatedName = 'Idle',
        $Exporting..$generatedName = 'Exporting',
        $Success..$generatedName = 'Success',
        $Failed..$generatedName = 'Failed',
      ],
    );
typedef HouseholdArchiveScreenScenario =
    Scenario<HouseholdArchiveScreen, HouseholdArchiveScreenArgs>;
typedef HouseholdArchiveScreenDefaults =
    Defaults<HouseholdArchiveScreen, HouseholdArchiveScreenArgs>;

class HouseholdArchiveScreenStory
    extends Story<HouseholdArchiveScreen, HouseholdArchiveScreenArgs> {
  HouseholdArchiveScreenStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<HouseholdArchiveScreen, HouseholdArchiveScreenArgs>?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => HouseholdArchiveScreen(
               key: args.key,
               gateway: args.gateway,
               session: args.session,
               onBack: args.onBack,
               saveArchive: args.saveArchive,
             ),
       );
}

class HouseholdArchiveScreenArgs extends StoryArgs<HouseholdArchiveScreen> {
  HouseholdArchiveScreenArgs({
    Arg<Key?>? key,
    required Arg<HouseholdGateway> gateway,
    required Arg<HouseholdSession> session,
    required Arg<void Function()> onBack,
    required Arg<Future<bool> Function(HouseholdArchive)> saveArchive,
  }) : this.keyArg = $initArg('key', key, null),
       this.gatewayArg = $initArg('gateway', gateway, null)!,
       this.sessionArg = $initArg('session', session, null)!,
       this.onBackArg = $initArg('onBack', onBack, null)!,
       this.saveArchiveArg = $initArg('saveArchive', saveArchive, null)!;

  HouseholdArchiveScreenArgs.fixed({
    Key? key,
    required HouseholdGateway gateway,
    required HouseholdSession session,
    required void Function() onBack,
    required Future<bool> Function(HouseholdArchive) saveArchive,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.gatewayArg = $initArg('gateway', Arg.fixed(gateway), null)!,
       this.sessionArg = $initArg('session', Arg.fixed(session), null)!,
       this.onBackArg = $initArg('onBack', Arg.fixed(onBack), null)!,
       this.saveArchiveArg = $initArg(
         'saveArchive',
         Arg.fixed(saveArchive),
         null,
       )!;

  final Arg<Key?>? keyArg;

  final Arg<HouseholdGateway> gatewayArg;

  final Arg<HouseholdSession> sessionArg;

  final Arg<void Function()> onBackArg;

  final Arg<Future<bool> Function(HouseholdArchive)> saveArchiveArg;

  Key? get key => keyArg?.value;

  HouseholdGateway get gateway => gatewayArg.value;

  HouseholdSession get session => sessionArg.value;

  void Function() get onBack => onBackArg.value;

  Future<bool> Function(HouseholdArchive) get saveArchive =>
      saveArchiveArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    gatewayArg,
    sessionArg,
    onBackArg,
    saveArchiveArg,
  ];
}
