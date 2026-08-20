// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_tool_row.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurToolRow, StoryArgs<BalaurToolRow>>;
typedef _Scenario = BalaurToolRowScenario;
typedef _Defaults = BalaurToolRowDefaults;
typedef _Story = BalaurToolRowStory;
typedef _Args = BalaurToolRowArgs;
final BalaurToolRowComponent =
    Component<BalaurToolRow, StoryArgs<BalaurToolRow>>(
      name: component.name ?? 'BalaurToolRow',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment:
          r'''Displays one visible tool or operating-system access record.''',
      stories: [
        $Search..$generatedName = 'Search',
        $FallbackGlyph..$generatedName = 'FallbackGlyph',
      ],
    );
typedef BalaurToolRowScenario = Scenario<BalaurToolRow, BalaurToolRowArgs>;
typedef BalaurToolRowDefaults = Defaults<BalaurToolRow, BalaurToolRowArgs>;

class BalaurToolRowStory extends Story<BalaurToolRow, BalaurToolRowArgs> {
  BalaurToolRowStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurToolRowArgs? args,
    StoryWidgetBuilder<BalaurToolRow, BalaurToolRowArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurToolRowArgs(),
         builder:
             builder ??
             (context, args) => BalaurToolRow(
               key: args.key,
               tool: args.tool,
               message: args.message,
               iconName: args.iconName,
               glyph: args.glyph,
             ),
       );
}

class BalaurToolRowArgs extends StoryArgs<BalaurToolRow> {
  BalaurToolRowArgs({
    Arg<Key?>? key,
    Arg<String>? tool,
    Arg<String>? message,
    Arg<String?>? iconName,
    Arg<String>? glyph,
  }) : this.keyArg = $initArg('key', key, null),
       this.toolArg = $initArg('tool', tool, StringArg(''))!,
       this.messageArg = $initArg('message', message, StringArg(''))!,
       this.iconNameArg = $initArg(
         'iconName',
         iconName,
         NullableStringArg(null),
       )!,
       this.glyphArg = $initArg('glyph', glyph, StringArg('⌥'))!;

  BalaurToolRowArgs.fixed({
    Key? key,
    String tool = '',
    String message = '',
    String? iconName = null,
    String glyph = '⌥',
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.toolArg = $initArg('tool', Arg.fixed(tool), null)!,
       this.messageArg = $initArg('message', Arg.fixed(message), null)!,
       this.iconNameArg = $initArg(
         'iconName',
         iconName == null ? null : Arg.fixed(iconName),
         null,
       ),
       this.glyphArg = $initArg('glyph', Arg.fixed(glyph), null)!;

  final Arg<Key?>? keyArg;

  final Arg<String> toolArg;

  final Arg<String> messageArg;

  final Arg<String?>? iconNameArg;

  final Arg<String> glyphArg;

  Key? get key => keyArg?.value;

  String get tool => toolArg.value;

  String get message => messageArg.value;

  String? get iconName => iconNameArg?.value;

  String get glyph => glyphArg.value;

  @override
  List<Arg?> get list => [keyArg, toolArg, messageArg, iconNameArg, glyphArg];
}
