// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'calendar_connection_screen.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component =
    Component<CalendarConnectionScreen, StoryArgs<CalendarConnectionScreen>>;
typedef _Scenario = CalendarConnectionScreenScenario;
typedef _Defaults = CalendarConnectionScreenDefaults;
typedef _Story = CalendarConnectionScreenStory;
typedef _Args = CalendarConnectionScreenArgs;
final CalendarConnectionScreenComponent =
    Component<CalendarConnectionScreen, StoryArgs<CalendarConnectionScreen>>(
      name: component.name ?? 'CalendarConnectionScreen',
      path: component.path ?? 'household/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Disconnected..$generatedName = 'Disconnected',
        $Authorizing..$generatedName = 'Authorizing',
        $SelectCalendar..$generatedName = 'SelectCalendar',
        $Connected..$generatedName = 'Connected',
        $AuthorizationFailed..$generatedName = 'AuthorizationFailed',
      ],
    );
typedef CalendarConnectionScreenScenario =
    Scenario<CalendarConnectionScreen, CalendarConnectionScreenArgs>;
typedef CalendarConnectionScreenDefaults =
    Defaults<CalendarConnectionScreen, CalendarConnectionScreenArgs>;

class CalendarConnectionScreenStory
    extends Story<CalendarConnectionScreen, CalendarConnectionScreenArgs> {
  CalendarConnectionScreenStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<CalendarConnectionScreen, CalendarConnectionScreenArgs>?
    builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => CalendarConnectionScreen(
               key: args.key,
               gateway: args.gateway,
               session: args.session,
               onBack: args.onBack,
               openAuthorization: args.openAuthorization,
             ),
       );
}

class CalendarConnectionScreenArgs extends StoryArgs<CalendarConnectionScreen> {
  CalendarConnectionScreenArgs({
    Arg<Key?>? key,
    required Arg<HouseholdGateway> gateway,
    required Arg<HouseholdSession> session,
    required Arg<void Function()> onBack,
    required Arg<Future<bool> Function(Uri)> openAuthorization,
  }) : this.keyArg = $initArg('key', key, null),
       this.gatewayArg = $initArg('gateway', gateway, null)!,
       this.sessionArg = $initArg('session', session, null)!,
       this.onBackArg = $initArg('onBack', onBack, null)!,
       this.openAuthorizationArg = $initArg(
         'openAuthorization',
         openAuthorization,
         null,
       )!;

  CalendarConnectionScreenArgs.fixed({
    Key? key,
    required HouseholdGateway gateway,
    required HouseholdSession session,
    required void Function() onBack,
    required Future<bool> Function(Uri) openAuthorization,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.gatewayArg = $initArg('gateway', Arg.fixed(gateway), null)!,
       this.sessionArg = $initArg('session', Arg.fixed(session), null)!,
       this.onBackArg = $initArg('onBack', Arg.fixed(onBack), null)!,
       this.openAuthorizationArg = $initArg(
         'openAuthorization',
         Arg.fixed(openAuthorization),
         null,
       )!;

  final Arg<Key?>? keyArg;

  final Arg<HouseholdGateway> gatewayArg;

  final Arg<HouseholdSession> sessionArg;

  final Arg<void Function()> onBackArg;

  final Arg<Future<bool> Function(Uri)> openAuthorizationArg;

  Key? get key => keyArg?.value;

  HouseholdGateway get gateway => gatewayArg.value;

  HouseholdSession get session => sessionArg.value;

  void Function() get onBack => onBackArg.value;

  Future<bool> Function(Uri) get openAuthorization =>
      openAuthorizationArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    gatewayArg,
    sessionArg,
    onBackArg,
    openAuthorizationArg,
  ];
}
