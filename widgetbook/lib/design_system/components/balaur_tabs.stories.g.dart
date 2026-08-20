// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_tabs.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurTabs, StoryArgs<BalaurTabs>>;
typedef _Scenario = BalaurTabsScenario;
typedef _Defaults = BalaurTabsDefaults;
typedef _Story = BalaurTabsStory;
typedef _Args = BalaurTabsArgs;
final BalaurTabsComponent = Component<BalaurTabs, StoryArgs<BalaurTabs>>(
  name: component.name ?? 'BalaurTabs',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a row of parchment navigation tabs.''',
  stories: [$TaskViews..$generatedName = 'TaskViews'],
);
typedef BalaurTabsScenario = Scenario<BalaurTabs, BalaurTabsArgs>;
typedef BalaurTabsDefaults = Defaults<BalaurTabs, BalaurTabsArgs>;

class BalaurTabsStory extends Story<BalaurTabs, BalaurTabsArgs> {
  BalaurTabsStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurTabs, BalaurTabsArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurTabs(
               key: args.key,
               items: args.items,
               active: args.active,
               onSelect: args.onSelect,
             ),
       );
}

class BalaurTabsArgs extends StoryArgs<BalaurTabs> {
  BalaurTabsArgs({
    Arg<Key?>? key,
    required Arg<List<String>> items,
    Arg<String>? active,
    required Arg<void Function(String)> onSelect,
  }) : this.keyArg = $initArg('key', key, null),
       this.itemsArg = $initArg('items', items, null)!,
       this.activeArg = $initArg('active', active, StringArg(''))!,
       this.onSelectArg = $initArg('onSelect', onSelect, null)!;

  BalaurTabsArgs.fixed({
    Key? key,
    required List<String> items,
    String active = '',
    required void Function(String) onSelect,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.itemsArg = $initArg('items', Arg.fixed(items), null)!,
       this.activeArg = $initArg('active', Arg.fixed(active), null)!,
       this.onSelectArg = $initArg('onSelect', Arg.fixed(onSelect), null)!;

  final Arg<Key?>? keyArg;

  final Arg<List<String>> itemsArg;

  final Arg<String> activeArg;

  final Arg<void Function(String)> onSelectArg;

  Key? get key => keyArg?.value;

  List<String> get items => itemsArg.value;

  String get active => activeArg.value;

  void Function(String) get onSelect => onSelectArg.value;

  @override
  List<Arg?> get list => [keyArg, itemsArg, activeArg, onSelectArg];
}
