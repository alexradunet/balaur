// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_stat_card.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurStatCard, StoryArgs<BalaurStatCard>>;
typedef _Scenario = BalaurStatCardScenario;
typedef _Defaults = BalaurStatCardDefaults;
typedef _Story = BalaurStatCardStory;
typedef _Args = BalaurStatCardArgs;
final BalaurStatCardComponent =
    Component<BalaurStatCard, StoryArgs<BalaurStatCard>>(
      name: component.name ?? 'BalaurStatCard',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays one life metric with a trend.''',
      stories: [
        $Weight..$generatedName = 'Weight',
        $Steps..$generatedName = 'Steps',
      ],
    );
typedef BalaurStatCardScenario = Scenario<BalaurStatCard, BalaurStatCardArgs>;
typedef BalaurStatCardDefaults = Defaults<BalaurStatCard, BalaurStatCardArgs>;

class BalaurStatCardStory extends Story<BalaurStatCard, BalaurStatCardArgs> {
  BalaurStatCardStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurStatCardArgs? args,
    StoryWidgetBuilder<BalaurStatCard, BalaurStatCardArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurStatCardArgs(),
         builder:
             builder ??
             (context, args) => BalaurStatCard(
               key: args.key,
               label: args.label,
               value: args.value,
               unit: args.unit,
               delta: args.delta,
               deltaTone: args.deltaTone,
               values: args.values,
               iconName: args.iconName,
             ),
       );
}

class BalaurStatCardArgs extends StoryArgs<BalaurStatCard> {
  BalaurStatCardArgs({
    Arg<Key?>? key,
    Arg<String>? label,
    Arg<String>? value,
    Arg<String?>? unit,
    Arg<String?>? delta,
    Arg<BalaurStatDeltaTone>? deltaTone,
    Arg<List<double>?>? values,
    Arg<String?>? iconName,
  }) : this.keyArg = $initArg('key', key, null),
       this.labelArg = $initArg('label', label, StringArg(''))!,
       this.valueArg = $initArg('value', value, StringArg(''))!,
       this.unitArg = $initArg('unit', unit, NullableStringArg(null))!,
       this.deltaArg = $initArg('delta', delta, NullableStringArg(null))!,
       this.deltaToneArg = $initArg(
         'deltaTone',
         deltaTone,
         EnumArg<BalaurStatDeltaTone>(
           BalaurStatDeltaTone.flat,
           values: BalaurStatDeltaTone.values,
         ),
       )!,
       this.valuesArg = $initArg('values', values, null),
       this.iconNameArg = $initArg(
         'iconName',
         iconName,
         NullableStringArg(null),
       )!;

  BalaurStatCardArgs.fixed({
    Key? key,
    String label = '',
    String value = '',
    String? unit = null,
    String? delta = null,
    BalaurStatDeltaTone deltaTone = BalaurStatDeltaTone.flat,
    List<double>? values,
    String? iconName = null,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.labelArg = $initArg('label', Arg.fixed(label), null)!,
       this.valueArg = $initArg('value', Arg.fixed(value), null)!,
       this.unitArg = $initArg(
         'unit',
         unit == null ? null : Arg.fixed(unit),
         null,
       ),
       this.deltaArg = $initArg(
         'delta',
         delta == null ? null : Arg.fixed(delta),
         null,
       ),
       this.deltaToneArg = $initArg('deltaTone', Arg.fixed(deltaTone), null)!,
       this.valuesArg = $initArg(
         'values',
         values == null ? null : Arg.fixed(values),
         null,
       ),
       this.iconNameArg = $initArg(
         'iconName',
         iconName == null ? null : Arg.fixed(iconName),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<String> labelArg;

  final Arg<String> valueArg;

  final Arg<String?>? unitArg;

  final Arg<String?>? deltaArg;

  final Arg<BalaurStatDeltaTone> deltaToneArg;

  final Arg<List<double>?>? valuesArg;

  final Arg<String?>? iconNameArg;

  Key? get key => keyArg?.value;

  String get label => labelArg.value;

  String get value => valueArg.value;

  String? get unit => unitArg?.value;

  String? get delta => deltaArg?.value;

  BalaurStatDeltaTone get deltaTone => deltaToneArg.value;

  List<double>? get values => valuesArg?.value;

  String? get iconName => iconNameArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    labelArg,
    valueArg,
    unitArg,
    deltaArg,
    deltaToneArg,
    valuesArg,
    iconNameArg,
  ];
}
