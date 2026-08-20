// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_surface.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurSurface, StoryArgs<BalaurSurface>>;
typedef _Scenario = BalaurSurfaceScenario;
typedef _Defaults = BalaurSurfaceDefaults;
typedef _Story = BalaurSurfaceStory;
typedef _Args = BalaurSurfaceArgs;
final BalaurSurfaceComponent = Component<BalaurSurface, StoryArgs<BalaurSurface>>(
  name: component.name ?? 'BalaurSurface',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment:
      r'''Draws a reusable Hearthwood material with pixel-hard construction.''',
  stories: [
    $Parchment..$generatedName = 'Parchment',
    $Wood..$generatedName = 'Wood',
    $Inset..$generatedName = 'Inset',
  ],
);
typedef BalaurSurfaceScenario = Scenario<BalaurSurface, BalaurSurfaceArgs>;
typedef BalaurSurfaceDefaults = Defaults<BalaurSurface, BalaurSurfaceArgs>;

class BalaurSurfaceStory extends Story<BalaurSurface, BalaurSurfaceArgs> {
  BalaurSurfaceStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurSurface, BalaurSurfaceArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurSurface(
               key: args.key,
               child: args.child,
               material: args.material,
               padding: args.padding,
               ornate: args.ornate,
               dashed: args.dashed,
               hardShadow: args.hardShadow,
               borderColor: args.borderColor,
             ),
       );
}

class BalaurSurfaceArgs extends StoryArgs<BalaurSurface> {
  BalaurSurfaceArgs({
    Arg<Key?>? key,
    required Arg<Widget> child,
    Arg<BalaurSurfaceMaterial>? material,
    Arg<EdgeInsetsGeometry>? padding,
    Arg<bool>? ornate,
    Arg<bool>? dashed,
    Arg<bool>? hardShadow,
    Arg<Color?>? borderColor,
  }) : this.keyArg = $initArg('key', key, null),
       this.childArg = $initArg('child', child, null)!,
       this.materialArg = $initArg(
         'material',
         material,
         EnumArg<BalaurSurfaceMaterial>(
           BalaurSurfaceMaterial.parchment,
           values: BalaurSurfaceMaterial.values,
         ),
       )!,
       this.paddingArg = $initArg(
         'padding',
         padding,
         ConstArg(EdgeInsets.zero),
       )!,
       this.ornateArg = $initArg('ornate', ornate, BoolArg(false))!,
       this.dashedArg = $initArg('dashed', dashed, BoolArg(false))!,
       this.hardShadowArg = $initArg('hardShadow', hardShadow, BoolArg(true))!,
       this.borderColorArg = $initArg(
         'borderColor',
         borderColor,
         NullableColorArg(null),
       )!;

  BalaurSurfaceArgs.fixed({
    Key? key,
    required Widget child,
    BalaurSurfaceMaterial material = BalaurSurfaceMaterial.parchment,
    EdgeInsetsGeometry padding = EdgeInsets.zero,
    bool ornate = false,
    bool dashed = false,
    bool hardShadow = true,
    Color? borderColor = null,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.childArg = $initArg('child', Arg.fixed(child), null)!,
       this.materialArg = $initArg('material', Arg.fixed(material), null)!,
       this.paddingArg = $initArg('padding', Arg.fixed(padding), null)!,
       this.ornateArg = $initArg('ornate', Arg.fixed(ornate), null)!,
       this.dashedArg = $initArg('dashed', Arg.fixed(dashed), null)!,
       this.hardShadowArg = $initArg(
         'hardShadow',
         Arg.fixed(hardShadow),
         null,
       )!,
       this.borderColorArg = $initArg(
         'borderColor',
         borderColor == null ? null : Arg.fixed(borderColor),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<Widget> childArg;

  final Arg<BalaurSurfaceMaterial> materialArg;

  final Arg<EdgeInsetsGeometry> paddingArg;

  final Arg<bool> ornateArg;

  final Arg<bool> dashedArg;

  final Arg<bool> hardShadowArg;

  final Arg<Color?>? borderColorArg;

  Key? get key => keyArg?.value;

  Widget get child => childArg.value;

  BalaurSurfaceMaterial get material => materialArg.value;

  EdgeInsetsGeometry get padding => paddingArg.value;

  bool get ornate => ornateArg.value;

  bool get dashed => dashedArg.value;

  bool get hardShadow => hardShadowArg.value;

  Color? get borderColor => borderColorArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    childArg,
    materialArg,
    paddingArg,
    ornateArg,
    dashedArg,
    hardShadowArg,
    borderColorArg,
  ];
}
