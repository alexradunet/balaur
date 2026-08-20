// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_breadcrumb.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurBreadcrumb, StoryArgs<BalaurBreadcrumb>>;
typedef _Scenario = BalaurBreadcrumbScenario;
typedef _Defaults = BalaurBreadcrumbDefaults;
typedef _Story = BalaurBreadcrumbStory;
typedef _Args = BalaurBreadcrumbArgs;
final BalaurBreadcrumbComponent =
    Component<BalaurBreadcrumb, StoryArgs<BalaurBreadcrumb>>(
      name: component.name ?? 'BalaurBreadcrumb',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays a wood path trail.''',
      stories: [$Path..$generatedName = 'Path'],
    );
typedef BalaurBreadcrumbScenario =
    Scenario<BalaurBreadcrumb, BalaurBreadcrumbArgs>;
typedef BalaurBreadcrumbDefaults =
    Defaults<BalaurBreadcrumb, BalaurBreadcrumbArgs>;

class BalaurBreadcrumbStory
    extends Story<BalaurBreadcrumb, BalaurBreadcrumbArgs> {
  BalaurBreadcrumbStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurBreadcrumb, BalaurBreadcrumbArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) =>
                 BalaurBreadcrumb(key: args.key, items: args.items),
       );
}

class BalaurBreadcrumbArgs extends StoryArgs<BalaurBreadcrumb> {
  BalaurBreadcrumbArgs({
    Arg<Key?>? key,
    required Arg<List<BalaurBreadcrumbItem>> items,
  }) : this.keyArg = $initArg('key', key, null),
       this.itemsArg = $initArg('items', items, null)!;

  BalaurBreadcrumbArgs.fixed({
    Key? key,
    required List<BalaurBreadcrumbItem> items,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.itemsArg = $initArg('items', Arg.fixed(items), null)!;

  final Arg<Key?>? keyArg;

  final Arg<List<BalaurBreadcrumbItem>> itemsArg;

  Key? get key => keyArg?.value;

  List<BalaurBreadcrumbItem> get items => itemsArg.value;

  @override
  List<Arg?> get list => [keyArg, itemsArg];
}
