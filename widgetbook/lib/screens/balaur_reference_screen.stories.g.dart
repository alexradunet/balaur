// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_reference_screen.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<BalaurReferenceScreen, StoryArgs<BalaurReferenceScreen>>;
typedef _Scenario = BalaurReferenceScreenScenario;
typedef _Defaults = BalaurReferenceScreenDefaults;
typedef _Story = BalaurReferenceScreenStory;
typedef _Args = BalaurReferenceScreenArgs;
final BalaurReferenceScreenComponent =
    Component<BalaurReferenceScreen, StoryArgs<BalaurReferenceScreen>>(
      name: component.name ?? 'BalaurReferenceScreen',
      path: component.path ?? 'screens',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Chat..$generatedName = 'Chat',
        $Tasks..$generatedName = 'Tasks',
        $Memory..$generatedName = 'Memory',
        $Life..$generatedName = 'Life',
        $Profile..$generatedName = 'Profile',
      ],
    );
typedef BalaurReferenceScreenScenario =
    Scenario<BalaurReferenceScreen, BalaurReferenceScreenArgs>;
typedef BalaurReferenceScreenDefaults =
    Defaults<BalaurReferenceScreen, BalaurReferenceScreenArgs>;

class BalaurReferenceScreenStory
    extends Story<BalaurReferenceScreen, BalaurReferenceScreenArgs> {
  BalaurReferenceScreenStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurReferenceScreenArgs? args,
    StoryWidgetBuilder<BalaurReferenceScreen, BalaurReferenceScreenArgs>?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurReferenceScreenArgs(),
         builder:
             builder ??
             (context, args) =>
                 BalaurReferenceScreen(key: args.key, screen: args.screen),
       );
}

class BalaurReferenceScreenArgs extends StoryArgs<BalaurReferenceScreen> {
  BalaurReferenceScreenArgs({
    Arg<Key?>? key,
    Arg<BalaurReferenceScreenKind>? screen,
  }) : this.keyArg = $initArg('key', key, null),
       this.screenArg = $initArg(
         'screen',
         screen,
         EnumArg<BalaurReferenceScreenKind>(
           BalaurReferenceScreenKind.chat,
           values: BalaurReferenceScreenKind.values,
         ),
       )!;

  BalaurReferenceScreenArgs.fixed({
    Key? key,
    BalaurReferenceScreenKind screen = BalaurReferenceScreenKind.chat,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.screenArg = $initArg('screen', Arg.fixed(screen), null)!;

  final Arg<Key?>? keyArg;

  final Arg<BalaurReferenceScreenKind> screenArg;

  Key? get key => keyArg?.value;

  BalaurReferenceScreenKind get screen => screenArg.value;

  @override
  List<Arg?> get list => [keyArg, screenArg];
}
