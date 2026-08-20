// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_pagination.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurPagination, StoryArgs<BalaurPagination>>;
typedef _Scenario = BalaurPaginationScenario;
typedef _Defaults = BalaurPaginationDefaults;
typedef _Story = BalaurPaginationStory;
typedef _Args = BalaurPaginationArgs;
final BalaurPaginationComponent =
    Component<BalaurPagination, StoryArgs<BalaurPagination>>(
      name: component.name ?? 'BalaurPagination',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays previous, current, and next page controls.''',
      stories: [
        $FirstPage..$generatedName = 'FirstPage',
        $MiddlePage..$generatedName = 'MiddlePage',
      ],
    );
typedef BalaurPaginationScenario =
    Scenario<BalaurPagination, BalaurPaginationArgs>;
typedef BalaurPaginationDefaults =
    Defaults<BalaurPagination, BalaurPaginationArgs>;

class BalaurPaginationStory
    extends Story<BalaurPagination, BalaurPaginationArgs> {
  BalaurPaginationStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurPagination, BalaurPaginationArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurPagination(
               key: args.key,
               page: args.page,
               total: args.total,
               onChange: args.onChange,
             ),
       );
}

class BalaurPaginationArgs extends StoryArgs<BalaurPagination> {
  BalaurPaginationArgs({
    Arg<Key?>? key,
    Arg<int>? page,
    Arg<int>? total,
    required Arg<void Function(int)> onChange,
  }) : this.keyArg = $initArg('key', key, null),
       this.pageArg = $initArg('page', page, IntArg(0))!,
       this.totalArg = $initArg('total', total, IntArg(0))!,
       this.onChangeArg = $initArg('onChange', onChange, null)!;

  BalaurPaginationArgs.fixed({
    Key? key,
    int page = 0,
    int total = 0,
    required void Function(int) onChange,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.pageArg = $initArg('page', Arg.fixed(page), null)!,
       this.totalArg = $initArg('total', Arg.fixed(total), null)!,
       this.onChangeArg = $initArg('onChange', Arg.fixed(onChange), null)!;

  final Arg<Key?>? keyArg;

  final Arg<int> pageArg;

  final Arg<int> totalArg;

  final Arg<void Function(int)> onChangeArg;

  Key? get key => keyArg?.value;

  int get page => pageArg.value;

  int get total => totalArg.value;

  void Function(int) get onChange => onChangeArg.value;

  @override
  List<Arg?> get list => [keyArg, pageArg, totalArg, onChangeArg];
}
