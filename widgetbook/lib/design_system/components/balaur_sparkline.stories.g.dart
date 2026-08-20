// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_sparkline.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurSparkline, StoryArgs<BalaurSparkline>>;
typedef _Scenario = BalaurSparklineScenario;
typedef _Defaults = BalaurSparklineDefaults;
typedef _Story = BalaurSparklineStory;
typedef _Args = BalaurSparklineArgs;
final BalaurSparklineComponent =
    Component<BalaurSparkline, StoryArgs<BalaurSparkline>>(
      name: component.name ?? 'BalaurSparkline',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays a compact trend without axes.''',
      stories: [$Trend..$generatedName = 'Trend'],
    );
typedef BalaurSparklineScenario =
    Scenario<BalaurSparkline, BalaurSparklineArgs>;
typedef BalaurSparklineDefaults =
    Defaults<BalaurSparkline, BalaurSparklineArgs>;

class BalaurSparklineStory extends Story<BalaurSparkline, BalaurSparklineArgs> {
  BalaurSparklineStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurSparkline, BalaurSparklineArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurSparkline(
               key: args.key,
               values: args.values,
               color: args.color,
               width: args.width,
               height: args.height,
             ),
       );
}

class BalaurSparklineArgs extends StoryArgs<BalaurSparkline> {
  BalaurSparklineArgs({
    Arg<Key?>? key,
    required Arg<List<double>> values,
    Arg<Color?>? color,
    Arg<double>? width,
    Arg<double>? height,
  }) : this.keyArg = $initArg('key', key, null),
       this.valuesArg = $initArg('values', values, null)!,
       this.colorArg = $initArg('color', color, NullableColorArg(null))!,
       this.widthArg = $initArg('width', width, DoubleArg(150))!,
       this.heightArg = $initArg('height', height, DoubleArg(34))!;

  BalaurSparklineArgs.fixed({
    Key? key,
    required List<double> values,
    Color? color = null,
    double width = 150,
    double height = 34,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.valuesArg = $initArg('values', Arg.fixed(values), null)!,
       this.colorArg = $initArg(
         'color',
         color == null ? null : Arg.fixed(color),
         null,
       ),
       this.widthArg = $initArg('width', Arg.fixed(width), null)!,
       this.heightArg = $initArg('height', Arg.fixed(height), null)!;

  final Arg<Key?>? keyArg;

  final Arg<List<double>> valuesArg;

  final Arg<Color?>? colorArg;

  final Arg<double> widthArg;

  final Arg<double> heightArg;

  Key? get key => keyArg?.value;

  List<double> get values => valuesArg.value;

  Color? get color => colorArg?.value;

  double get width => widthArg.value;

  double get height => heightArg.value;

  @override
  List<Arg?> get list => [keyArg, valuesArg, colorArg, widthArg, heightArg];
}
