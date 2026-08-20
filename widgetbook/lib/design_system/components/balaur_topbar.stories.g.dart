// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_topbar.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurTopbar, StoryArgs<BalaurTopbar>>;
typedef _Scenario = BalaurTopbarScenario;
typedef _Defaults = BalaurTopbarDefaults;
typedef _Story = BalaurTopbarStory;
typedef _Args = BalaurTopbarArgs;
final BalaurTopbarComponent = Component<BalaurTopbar, StoryArgs<BalaurTopbar>>(
  name: component.name ?? 'BalaurTopbar',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment:
      r'''Displays the wood application nameplate and primary navigation.''',
  stories: [$ChatActive..$generatedName = 'ChatActive'],
);
typedef BalaurTopbarScenario = Scenario<BalaurTopbar, BalaurTopbarArgs>;
typedef BalaurTopbarDefaults = Defaults<BalaurTopbar, BalaurTopbarArgs>;

class BalaurTopbarStory extends Story<BalaurTopbar, BalaurTopbarArgs> {
  BalaurTopbarStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurTopbarArgs? args,
    StoryWidgetBuilder<BalaurTopbar, BalaurTopbarArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurTopbarArgs(),
         builder:
             builder ??
             (context, args) => BalaurTopbar(
               key: args.key,
               brand: args.brand,
               links: args.links,
               active: args.active,
               onNavigate: args.onNavigate,
               onToggleTheme: args.onToggleTheme,
               themeGlyph: args.themeGlyph,
             ),
       );
}

class BalaurTopbarArgs extends StoryArgs<BalaurTopbar> {
  BalaurTopbarArgs({
    Arg<Key?>? key,
    Arg<String>? brand,
    Arg<List<String>>? links,
    Arg<String?>? active,
    Arg<void Function(String)?>? onNavigate,
    Arg<void Function()?>? onToggleTheme,
    Arg<String>? themeGlyph,
  }) : this.keyArg = $initArg('key', key, null),
       this.brandArg = $initArg('brand', brand, StringArg('Balaur'))!,
       this.linksArg = $initArg('links', links, ConstArg(const []))!,
       this.activeArg = $initArg('active', active, NullableStringArg(null))!,
       this.onNavigateArg = $initArg('onNavigate', onNavigate, null),
       this.onToggleThemeArg = $initArg('onToggleTheme', onToggleTheme, null),
       this.themeGlyphArg = $initArg('themeGlyph', themeGlyph, StringArg('◑'))!;

  BalaurTopbarArgs.fixed({
    Key? key,
    String brand = 'Balaur',
    List<String> links = const [],
    String? active = null,
    void Function(String)? onNavigate,
    void Function()? onToggleTheme,
    String themeGlyph = '◑',
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.brandArg = $initArg('brand', Arg.fixed(brand), null)!,
       this.linksArg = $initArg('links', Arg.fixed(links), null)!,
       this.activeArg = $initArg(
         'active',
         active == null ? null : Arg.fixed(active),
         null,
       ),
       this.onNavigateArg = $initArg(
         'onNavigate',
         onNavigate == null ? null : Arg.fixed(onNavigate),
         null,
       ),
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

  final Arg<String> brandArg;

  final Arg<List<String>> linksArg;

  final Arg<String?>? activeArg;

  final Arg<void Function(String)?>? onNavigateArg;

  final Arg<void Function()?>? onToggleThemeArg;

  final Arg<String> themeGlyphArg;

  Key? get key => keyArg?.value;

  String get brand => brandArg.value;

  List<String> get links => linksArg.value;

  String? get active => activeArg?.value;

  void Function(String)? get onNavigate => onNavigateArg?.value;

  void Function()? get onToggleTheme => onToggleThemeArg?.value;

  String get themeGlyph => themeGlyphArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    brandArg,
    linksArg,
    activeArg,
    onNavigateArg,
    onToggleThemeArg,
    themeGlyphArg,
  ];
}
