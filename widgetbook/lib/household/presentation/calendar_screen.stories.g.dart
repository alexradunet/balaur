// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'calendar_screen.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<CalendarScreen, StoryArgs<CalendarScreen>>;
typedef _Scenario = CalendarScreenScenario;
typedef _Defaults = CalendarScreenDefaults;
typedef _Story = CalendarScreenStory;
typedef _Args = CalendarScreenArgs;
final CalendarScreenComponent =
    Component<CalendarScreen, StoryArgs<CalendarScreen>>(
      name: component.name ?? 'CalendarScreen',
      path: component.path ?? 'household/presentation',
      docsBuilder: component.docsBuilder,
      docComment: null,
      stories: [
        $Loading..$generatedName = 'Loading',
        $Populated..$generatedName = 'Populated',
        $EmptyDay..$generatedName = 'EmptyDay',
        $EmptyMonth..$generatedName = 'EmptyMonth',
        $DisconnectedAdministrator
          ..$generatedName = 'DisconnectedAdministrator',
        $DisconnectedMember..$generatedName = 'DisconnectedMember',
        $AuthorizationFailure..$generatedName = 'AuthorizationFailure',
        $ServerFailure..$generatedName = 'ServerFailure',
        $RefreshFailure..$generatedName = 'RefreshFailure',
      ],
    );
typedef CalendarScreenScenario = Scenario<CalendarScreen, CalendarScreenArgs>;
typedef CalendarScreenDefaults = Defaults<CalendarScreen, CalendarScreenArgs>;

class CalendarScreenStory extends Story<CalendarScreen, CalendarScreenArgs> {
  CalendarScreenStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<CalendarScreen, CalendarScreenArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => CalendarScreen(
               key: args.key,
               gateway: args.gateway,
               session: args.session,
               openAuthorization: args.openAuthorization,
               onManageConnection: args.onManageConnection,
               now: args.now,
             ),
       );
}

class CalendarScreenArgs extends StoryArgs<CalendarScreen> {
  CalendarScreenArgs({
    Arg<Key?>? key,
    required Arg<HouseholdGateway> gateway,
    required Arg<HouseholdSession> session,
    required Arg<Future<bool> Function(Uri)> openAuthorization,
    Arg<void Function()?>? onManageConnection,
    Arg<DateTime Function()?>? now,
  }) : this.keyArg = $initArg('key', key, null),
       this.gatewayArg = $initArg('gateway', gateway, null)!,
       this.sessionArg = $initArg('session', session, null)!,
       this.openAuthorizationArg = $initArg(
         'openAuthorization',
         openAuthorization,
         null,
       )!,
       this.onManageConnectionArg = $initArg(
         'onManageConnection',
         onManageConnection,
         null,
       ),
       this.nowArg = $initArg('now', now, null);

  CalendarScreenArgs.fixed({
    Key? key,
    required HouseholdGateway gateway,
    required HouseholdSession session,
    required Future<bool> Function(Uri) openAuthorization,
    void Function()? onManageConnection,
    DateTime Function()? now,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.gatewayArg = $initArg('gateway', Arg.fixed(gateway), null)!,
       this.sessionArg = $initArg('session', Arg.fixed(session), null)!,
       this.openAuthorizationArg = $initArg(
         'openAuthorization',
         Arg.fixed(openAuthorization),
         null,
       )!,
       this.onManageConnectionArg = $initArg(
         'onManageConnection',
         onManageConnection == null ? null : Arg.fixed(onManageConnection),
         null,
       ),
       this.nowArg = $initArg('now', now == null ? null : Arg.fixed(now), null);

  final Arg<Key?>? keyArg;

  final Arg<HouseholdGateway> gatewayArg;

  final Arg<HouseholdSession> sessionArg;

  final Arg<Future<bool> Function(Uri)> openAuthorizationArg;

  final Arg<void Function()?>? onManageConnectionArg;

  final Arg<DateTime Function()?>? nowArg;

  Key? get key => keyArg?.value;

  HouseholdGateway get gateway => gatewayArg.value;

  HouseholdSession get session => sessionArg.value;

  Future<bool> Function(Uri) get openAuthorization =>
      openAuthorizationArg.value;

  void Function()? get onManageConnection => onManageConnectionArg?.value;

  DateTime Function()? get now => nowArg?.value;

  @override
  List<Arg?> get list => [
    keyArg,
    gatewayArg,
    sessionArg,
    openAuthorizationArg,
    onManageConnectionArg,
    nowArg,
  ];
}
