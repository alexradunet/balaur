// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_list.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurList, StoryArgs<BalaurList>>;
typedef _Scenario = BalaurListScenario;
typedef _Defaults = BalaurListDefaults;
typedef _Story = BalaurListStory;
typedef _Args = BalaurListArgs;
final BalaurListComponent = Component<BalaurList, StoryArgs<BalaurList>>(
  name: component.name ?? 'BalaurList',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a homogeneous parchment list.''',
  stories: [
    $Today..$generatedName = 'Today',
    $NoHeader..$generatedName = 'NoHeader',
  ],
);
typedef BalaurListScenario = Scenario<BalaurList, BalaurListArgs>;
typedef BalaurListDefaults = Defaults<BalaurList, BalaurListArgs>;

class BalaurListStory extends Story<BalaurList, BalaurListArgs> {
  BalaurListStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurList, BalaurListArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurList(
               key: args.key,
               items: args.items,
               title: args.title,
             ),
       );
}

class BalaurListArgs extends StoryArgs<BalaurList> {
  BalaurListArgs({
    Arg<Key?>? key,
    required Arg<List<BalaurListItem>> items,
    Arg<String?>? title,
  }) : this.keyArg = $initArg('key', key, null),
       this.itemsArg = $initArg('items', items, null)!,
       this.titleArg = $initArg('title', title, NullableStringArg(null))!;

  BalaurListArgs.fixed({
    Key? key,
    required List<BalaurListItem> items,
    String? title = null,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.itemsArg = $initArg('items', Arg.fixed(items), null)!,
       this.titleArg = $initArg(
         'title',
         title == null ? null : Arg.fixed(title),
         null,
       );

  final Arg<Key?>? keyArg;

  final Arg<List<BalaurListItem>> itemsArg;

  final Arg<String?>? titleArg;

  Key? get key => keyArg?.value;

  List<BalaurListItem> get items => itemsArg.value;

  String? get title => titleArg?.value;

  @override
  List<Arg?> get list => [keyArg, itemsArg, titleArg];
}
