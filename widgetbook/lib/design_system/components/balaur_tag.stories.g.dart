// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_tag.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurTag, StoryArgs<BalaurTag>>;
typedef _Scenario = BalaurTagScenario;
typedef _Defaults = BalaurTagDefaults;
typedef _Story = BalaurTagStory;
typedef _Args = BalaurTagArgs;
final BalaurTagComponent = Component<BalaurTag, StoryArgs<BalaurTag>>(
  name: component.name ?? 'BalaurTag',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a small wood label with the teal stitch rune.''',
  stories: [$Default..$generatedName = 'Default'],
);
typedef BalaurTagScenario = Scenario<BalaurTag, BalaurTagArgs>;
typedef BalaurTagDefaults = Defaults<BalaurTag, BalaurTagArgs>;

class BalaurTagStory extends Story<BalaurTag, BalaurTagArgs> {
  BalaurTagStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurTagArgs? args,
    StoryWidgetBuilder<BalaurTag, BalaurTagArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurTagArgs(),
         builder:
             builder ??
             (context, args) => BalaurTag(key: args.key, label: args.label),
       );
}

class BalaurTagArgs extends StoryArgs<BalaurTag> {
  BalaurTagArgs({Arg<Key?>? key, Arg<String>? label})
    : this.keyArg = $initArg('key', key, null),
      this.labelArg = $initArg('label', label, StringArg(''))!;

  BalaurTagArgs.fixed({Key? key, String label = ''})
    : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
      this.labelArg = $initArg('label', Arg.fixed(label), null)!;

  final Arg<Key?>? keyArg;

  final Arg<String> labelArg;

  Key? get key => keyArg?.value;

  String get label => labelArg.value;

  @override
  List<Arg?> get list => [keyArg, labelArg];
}
