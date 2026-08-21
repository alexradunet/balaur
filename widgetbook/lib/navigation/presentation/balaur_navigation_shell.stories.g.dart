// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_navigation_shell.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<BalaurNavigationShell, StoryArgs<BalaurNavigationShell>>;
typedef _Scenario = BalaurNavigationShellScenario;
typedef _Defaults = BalaurNavigationShellDefaults;
typedef _Story = BalaurNavigationShellStory;
typedef _Args = BalaurNavigationShellArgs;
final BalaurNavigationShellComponent =
    Component<BalaurNavigationShell, StoryArgs<BalaurNavigationShell>>(
      name: component.name ?? 'BalaurNavigationShell',
      path: component.path ?? 'navigation/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Chat..$generatedName = 'Chat',
        $Calendar..$generatedName = 'Calendar',
        $Household..$generatedName = 'Household',
        $Settings..$generatedName = 'Settings',
      ],
    );
typedef BalaurNavigationShellScenario =
    Scenario<BalaurNavigationShell, BalaurNavigationShellArgs>;
typedef BalaurNavigationShellDefaults =
    Defaults<BalaurNavigationShell, BalaurNavigationShellArgs>;

class BalaurNavigationShellStory
    extends Story<BalaurNavigationShell, BalaurNavigationShellArgs> {
  BalaurNavigationShellStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurNavigationShell, BalaurNavigationShellArgs>?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurNavigationShell(
               key: args.key,
               active: args.active,
               onSelect: args.onSelect,
               child: args.child,
               onToggleTheme: args.onToggleTheme,
               themeGlyph: args.themeGlyph,
             ),
       );
}

class BalaurNavigationShellArgs extends StoryArgs<BalaurNavigationShell> {
  BalaurNavigationShellArgs({
    Arg<Key?>? key,
    Arg<BalaurDestination>? active,
    required Arg<void Function(BalaurDestination)> onSelect,
    required Arg<Widget> child,
    Arg<void Function()?>? onToggleTheme,
    Arg<String>? themeGlyph,
  }) : this.keyArg = $initArg('key', key, null),
       this.activeArg = $initArg(
         'active',
         active,
         EnumArg<BalaurDestination>(
           BalaurDestination.chat,
           values: BalaurDestination.values,
         ),
       )!,
       this.onSelectArg = $initArg('onSelect', onSelect, null)!,
       this.childArg = $initArg('child', child, null)!,
       this.onToggleThemeArg = $initArg('onToggleTheme', onToggleTheme, null),
       this.themeGlyphArg = $initArg('themeGlyph', themeGlyph, StringArg('◑'))!;

  BalaurNavigationShellArgs.fixed({
    Key? key,
    BalaurDestination active = BalaurDestination.chat,
    required void Function(BalaurDestination) onSelect,
    required Widget child,
    void Function()? onToggleTheme,
    String themeGlyph = '◑',
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.activeArg = $initArg('active', Arg.fixed(active), null)!,
       this.onSelectArg = $initArg('onSelect', Arg.fixed(onSelect), null)!,
       this.childArg = $initArg('child', Arg.fixed(child), null)!,
       this.onToggleThemeArg = $initArg(
         'onToggleTheme',
         onToggleTheme == null ? null : Arg.fixed(onToggleTheme),
         null,
       ),
       this.themeGlyphArg = $initArg(
         'themeGlyph',
         Arg.fixed(themeGlyph),
         null,
       )!;

  final Arg<Key?>? keyArg;

  final Arg<BalaurDestination> activeArg;

  final Arg<void Function(BalaurDestination)> onSelectArg;

  final Arg<Widget> childArg;

  final Arg<void Function()?>? onToggleThemeArg;

  final Arg<String> themeGlyphArg;

  Key? get key => keyArg?.value;

  BalaurDestination get active => activeArg.value;

  void Function(BalaurDestination) get onSelect => onSelectArg.value;

  Widget get child => childArg.value;

  void Function()? get onToggleTheme => onToggleThemeArg?.value;

  String get themeGlyph => themeGlyphArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    activeArg,
    onSelectArg,
    childArg,
    onToggleThemeArg,
    themeGlyphArg,
  ];
}
