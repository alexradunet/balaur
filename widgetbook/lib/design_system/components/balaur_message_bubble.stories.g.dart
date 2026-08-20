// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_message_bubble.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<BalaurMessageBubble, StoryArgs<BalaurMessageBubble>>;
typedef _Scenario = BalaurMessageBubbleScenario;
typedef _Defaults = BalaurMessageBubbleDefaults;
typedef _Story = BalaurMessageBubbleStory;
typedef _Args = BalaurMessageBubbleArgs;
final BalaurMessageBubbleComponent =
    Component<BalaurMessageBubble, StoryArgs<BalaurMessageBubble>>(
      name: component.name ?? 'BalaurMessageBubble',
      path: component.path ?? 'design_system/components',
      docsBuilder: component.docsBuilder,
      docComment: r'''Displays one Conversation message from a Household Member or an agent.''',
      stories: [
        $HouseholdMember..$generatedName = 'HouseholdMember',
        $Agent..$generatedName = 'Agent',
        $Streaming..$generatedName = 'Streaming',
        $EmptyStreaming..$generatedName = 'EmptyStreaming',
        $Stopped..$generatedName = 'Stopped',
        $Failed..$generatedName = 'Failed',
        $LongContent..$generatedName = 'LongContent',
      ],
    );
typedef BalaurMessageBubbleScenario =
    Scenario<BalaurMessageBubble, BalaurMessageBubbleArgs>;
typedef BalaurMessageBubbleDefaults =
    Defaults<BalaurMessageBubble, BalaurMessageBubbleArgs>;

class BalaurMessageBubbleStory
    extends Story<BalaurMessageBubble, BalaurMessageBubbleArgs> {
  BalaurMessageBubbleStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurMessageBubbleArgs? args,
    StoryWidgetBuilder<BalaurMessageBubble, BalaurMessageBubbleArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurMessageBubbleArgs(),
         builder:
             builder ??
             (context, args) => BalaurMessageBubble(
               key: args.key,
               content: args.content,
               role: args.role,
               status: args.status,
             ),
       );
}

class BalaurMessageBubbleArgs extends StoryArgs<BalaurMessageBubble> {
  BalaurMessageBubbleArgs({
    Arg<Key?>? key,
    Arg<String>? content,
    Arg<BalaurMessageBubbleRole>? role,
    Arg<BalaurMessageBubbleStatus>? status,
  }) : this.keyArg = $initArg('key', key, null),
       this.contentArg = $initArg('content', content, StringArg(''))!,
       this.roleArg = $initArg(
         'role',
         role,
         EnumArg<BalaurMessageBubbleRole>(
           BalaurMessageBubbleRole.householdMember,
           values: BalaurMessageBubbleRole.values,
         ),
       )!,
       this.statusArg = $initArg(
         'status',
         status,
         EnumArg<BalaurMessageBubbleStatus>(
           BalaurMessageBubbleStatus.complete,
           values: BalaurMessageBubbleStatus.values,
         ),
       )!;

  BalaurMessageBubbleArgs.fixed({
    Key? key,
    String content = '',
    BalaurMessageBubbleRole role = BalaurMessageBubbleRole.householdMember,
    BalaurMessageBubbleStatus status = BalaurMessageBubbleStatus.complete,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.contentArg = $initArg('content', Arg.fixed(content), null)!,
       this.roleArg = $initArg('role', Arg.fixed(role), null)!,
       this.statusArg = $initArg('status', Arg.fixed(status), null)!;

  final Arg<Key?>? keyArg;

  final Arg<String> contentArg;

  final Arg<BalaurMessageBubbleRole> roleArg;

  final Arg<BalaurMessageBubbleStatus> statusArg;

  Key? get key => keyArg?.value;

  String get content => contentArg.value;

  BalaurMessageBubbleRole get role => roleArg.value;

  BalaurMessageBubbleStatus get status => statusArg.value;

  @override
  List<Arg?> get list => [keyArg, contentArg, roleArg, statusArg];
}
