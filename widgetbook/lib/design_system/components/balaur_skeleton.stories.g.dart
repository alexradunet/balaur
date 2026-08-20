// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_skeleton.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurSkeleton, StoryArgs<BalaurSkeleton>>;
typedef _Scenario = BalaurSkeletonScenario;
typedef _Defaults = BalaurSkeletonDefaults;
typedef _Story = BalaurSkeletonStory;
typedef _Args = BalaurSkeletonArgs;
final BalaurSkeletonComponent =
    Component<BalaurSkeleton, StoryArgs<BalaurSkeleton>>(
      name: component.name ?? 'BalaurSkeleton',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays a carved loading placeholder.''',
      stories: [
        $Line..$generatedName = 'Line',
        $Block..$generatedName = 'Block',
        $Avatar..$generatedName = 'Avatar',
      ],
    );
typedef BalaurSkeletonScenario = Scenario<BalaurSkeleton, BalaurSkeletonArgs>;
typedef BalaurSkeletonDefaults = Defaults<BalaurSkeleton, BalaurSkeletonArgs>;

class BalaurSkeletonStory extends Story<BalaurSkeleton, BalaurSkeletonArgs> {
  BalaurSkeletonStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurSkeletonArgs? args,
    StoryWidgetBuilder<BalaurSkeleton, BalaurSkeletonArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurSkeletonArgs(),
         builder:
             builder ??
             (context, args) => BalaurSkeleton(
               key: args.key,
               variant: args.variant,
               width: args.width,
               height: args.height,
             ),
       );
}

class BalaurSkeletonArgs extends StoryArgs<BalaurSkeleton> {
  BalaurSkeletonArgs({
    Arg<Key?>? key,
    Arg<BalaurSkeletonVariant>? variant,
    Arg<double?>? width,
    Arg<double?>? height,
  }) : this.keyArg = $initArg('key', key, null),
       this.variantArg = $initArg(
         'variant',
         variant,
         EnumArg<BalaurSkeletonVariant>(
           BalaurSkeletonVariant.line,
           values: BalaurSkeletonVariant.values,
         ),
       )!,
       this.widthArg = $initArg('width', width, NullableDoubleArg(null))!,
       this.heightArg = $initArg('height', height, NullableDoubleArg(null))!;

  BalaurSkeletonArgs.fixed({
    Key? key,
    BalaurSkeletonVariant variant = BalaurSkeletonVariant.line,
    double? width = null,
    double? height = null,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.variantArg = $initArg('variant', Arg.fixed(variant), null)!,
       this.widthArg = $initArg(
         'width',
         width == null ? null : Arg.fixed(width),
         null,
       ),
       this.heightArg = $initArg(
         'height',
         height == null ? null : Arg.fixed(height),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<BalaurSkeletonVariant> variantArg;

  final Arg<double?>? widthArg;

  final Arg<double?>? heightArg;

  Key? get key => keyArg?.value;

  BalaurSkeletonVariant get variant => variantArg.value;

  double? get width => widthArg?.value;

  double? get height => heightArg?.value;

  @override
  List<Arg?> get list => [keyArg, variantArg, widthArg, heightArg];
}
