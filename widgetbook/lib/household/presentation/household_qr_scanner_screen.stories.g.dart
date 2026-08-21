// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'household_qr_scanner_screen.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<HouseholdQrScannerScreen, StoryArgs<HouseholdQrScannerScreen>>;
typedef _Scenario = HouseholdQrScannerScreenScenario;
typedef _Defaults = HouseholdQrScannerScreenDefaults;
typedef _Story = HouseholdQrScannerScreenStory;
typedef _Args = HouseholdQrScannerScreenArgs;
final HouseholdQrScannerScreenComponent =
    Component<HouseholdQrScannerScreen, StoryArgs<HouseholdQrScannerScreen>>(
      name: component.name ?? 'HouseholdQrScannerScreen',
      path: component.path ?? 'household/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Active..$generatedName = 'Active',
        $CameraError..$generatedName = 'CameraError',
        $Unavailable..$generatedName = 'Unavailable',
      ],
    );
typedef HouseholdQrScannerScreenScenario =
    Scenario<HouseholdQrScannerScreen, HouseholdQrScannerScreenArgs>;
typedef HouseholdQrScannerScreenDefaults =
    Defaults<HouseholdQrScannerScreen, HouseholdQrScannerScreenArgs>;

class HouseholdQrScannerScreenStory
    extends Story<HouseholdQrScannerScreen, HouseholdQrScannerScreenArgs> {
  HouseholdQrScannerScreenStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    HouseholdQrScannerScreenArgs? args,
    StoryWidgetBuilder<HouseholdQrScannerScreen, HouseholdQrScannerScreenArgs>?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? HouseholdQrScannerScreenArgs(),
         builder:
             builder ??
             (context, args) => HouseholdQrScannerScreen(
               key: args.key,
               state: args.state,
               cameraPreview: args.cameraPreview,
             ),
       );
}

class HouseholdQrScannerScreenArgs extends StoryArgs<HouseholdQrScannerScreen> {
  HouseholdQrScannerScreenArgs({
    Arg<Key?>? key,
    Arg<HouseholdQrScannerState>? state,
    Arg<Widget?>? cameraPreview,
  }) : this.keyArg = $initArg('key', key, null),
       this.stateArg = $initArg(
         'state',
         state,
         EnumArg<HouseholdQrScannerState>(
           HouseholdQrScannerState.automatic,
           values: HouseholdQrScannerState.values,
         ),
       )!,
       this.cameraPreviewArg = $initArg('cameraPreview', cameraPreview, null);

  HouseholdQrScannerScreenArgs.fixed({
    Key? key,
    HouseholdQrScannerState state = HouseholdQrScannerState.automatic,
    Widget? cameraPreview,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.stateArg = $initArg('state', Arg.fixed(state), null)!,
       this.cameraPreviewArg = $initArg(
         'cameraPreview',
         cameraPreview == null ? null : Arg.fixed(cameraPreview),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<HouseholdQrScannerState> stateArg;

  final Arg<Widget?>? cameraPreviewArg;

  Key? get key => keyArg?.value;

  HouseholdQrScannerState get state => stateArg.value;

  Widget? get cameraPreview => cameraPreviewArg?.value;

  @override
  List<Arg?> get list => [keyArg, stateArg, cameraPreviewArg];
}
