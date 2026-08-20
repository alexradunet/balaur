// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_card.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurCard, StoryArgs<BalaurCard>>;
typedef _Scenario = BalaurCardScenario;
typedef _Defaults = BalaurCardDefaults;
typedef _Story = BalaurCardStory;
typedef _Args = BalaurCardArgs;
final BalaurCardComponent = Component<BalaurCard, StoryArgs<BalaurCard>>(
  name: component.name ?? 'BalaurCard',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment:
      r'''Displays content on a parchment card with a gold pixel notch.''',
  stories: [
    $Parchment..$generatedName = 'Parchment',
    $Ornate..$generatedName = 'Ornate',
    $Archived..$generatedName = 'Archived',
  ],
);
typedef BalaurCardScenario = Scenario<BalaurCard, BalaurCardArgs>;
typedef BalaurCardDefaults = Defaults<BalaurCard, BalaurCardArgs>;

class BalaurCardStory extends Story<BalaurCard, BalaurCardArgs> {
  BalaurCardStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurCard, BalaurCardArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurCard(
               key: args.key,
               child: args.child,
               padding: args.padding,
               ornate: args.ornate,
               dashed: args.dashed,
             ),
       );
}

class BalaurCardArgs extends StoryArgs<BalaurCard> {
  BalaurCardArgs({
    Arg<Key?>? key,
    required Arg<Widget> child,
    Arg<EdgeInsetsGeometry>? padding,
    Arg<bool>? ornate,
    Arg<bool>? dashed,
  }) : this.keyArg = $initArg('key', key, null),
       this.childArg = $initArg('child', child, null)!,
       this.paddingArg = $initArg(
         'padding',
         padding,
         ConstArg(const EdgeInsets.all(18)),
       )!,
       this.ornateArg = $initArg('ornate', ornate, BoolArg(false))!,
       this.dashedArg = $initArg('dashed', dashed, BoolArg(false))!;

  BalaurCardArgs.fixed({
    Key? key,
    required Widget child,
    EdgeInsetsGeometry padding = const EdgeInsets.all(18),
    bool ornate = false,
    bool dashed = false,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.childArg = $initArg('child', Arg.fixed(child), null)!,
       this.paddingArg = $initArg('padding', Arg.fixed(padding), null)!,
       this.ornateArg = $initArg('ornate', Arg.fixed(ornate), null)!,
       this.dashedArg = $initArg('dashed', Arg.fixed(dashed), null)!;

  final Arg<Key?>? keyArg;

  final Arg<Widget> childArg;

  final Arg<EdgeInsetsGeometry> paddingArg;

  final Arg<bool> ornateArg;

  final Arg<bool> dashedArg;

  Key? get key => keyArg?.value;

  Widget get child => childArg.value;

  EdgeInsetsGeometry get padding => paddingArg.value;

  bool get ornate => ornateArg.value;

  bool get dashed => dashedArg.value;

  @override
  List<Arg?> get list => [keyArg, childArg, paddingArg, ornateArg, dashedArg];
}
