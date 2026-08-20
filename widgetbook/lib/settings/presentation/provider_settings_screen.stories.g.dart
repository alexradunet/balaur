// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'provider_settings_screen.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<ProviderSettingsScreen, StoryArgs<ProviderSettingsScreen>>;
typedef _Scenario = ProviderSettingsScreenScenario;
typedef _Defaults = ProviderSettingsScreenDefaults;
typedef _Story = ProviderSettingsScreenStory;
typedef _Args = ProviderSettingsScreenArgs;
final ProviderSettingsScreenComponent =
    Component<ProviderSettingsScreen, StoryArgs<ProviderSettingsScreen>>(
      name: component.name ?? 'ProviderSettingsScreen',
      path: component.path ?? 'settings/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Empty..$generatedName = 'Empty',
        $Configured..$generatedName = 'Configured',
      ],
    );
typedef ProviderSettingsScreenScenario =
    Scenario<ProviderSettingsScreen, ProviderSettingsScreenArgs>;
typedef ProviderSettingsScreenDefaults =
    Defaults<ProviderSettingsScreen, ProviderSettingsScreenArgs>;

class ProviderSettingsScreenStory
    extends Story<ProviderSettingsScreen, ProviderSettingsScreenArgs> {
  ProviderSettingsScreenStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<ProviderSettingsScreen, ProviderSettingsScreenArgs>?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => ProviderSettingsScreen(
               key: args.key,
               settingsStore: args.settingsStore,
             ),
       );
}

class ProviderSettingsScreenArgs extends StoryArgs<ProviderSettingsScreen> {
  ProviderSettingsScreenArgs({
    Arg<Key?>? key,
    required Arg<ProviderSettingsStore> settingsStore,
  }) : this.keyArg = $initArg('key', key, null),
       this.settingsStoreArg = $initArg('settingsStore', settingsStore, null)!;

  ProviderSettingsScreenArgs.fixed({
    Key? key,
    required ProviderSettingsStore settingsStore,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.settingsStoreArg = $initArg(
         'settingsStore',
         Arg.fixed(settingsStore),
         null,
       )!;

  final Arg<Key?>? keyArg;

  final Arg<ProviderSettingsStore> settingsStoreArg;

  Key? get key => keyArg?.value;

  ProviderSettingsStore get settingsStore => settingsStoreArg.value;

  @override
  List<Arg?> get list => [keyArg, settingsStoreArg];
}
