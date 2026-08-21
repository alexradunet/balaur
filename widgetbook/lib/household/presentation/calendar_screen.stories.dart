import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/calendar_entry.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/calendar_screen.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'calendar_screen.stories.g.dart';

const component = ComponentMeta(name: 'Calendar', path: 'Screens/Household');
const meta = Meta(CalendarScreen.new);

final _session = HouseholdSession(
  serverAddress: HouseholdServerAddress.parse('https://household.example.com'),
  member: const HouseholdMember(
    id: 'administrator-example',
    displayName: 'Alex',
    email: 'alex@example.com',
    role: HouseholdMemberRole.administrator,
  ),
);

final _memberSession = HouseholdSession(
  serverAddress: _session.serverAddress,
  member: const HouseholdMember(
    id: 'member-example',
    displayName: 'Mara',
    email: 'mara@example.com',
    role: HouseholdMemberRole.member,
  ),
);

final _connected = CalendarConnection(
  provider: 'google',
  status: CalendarConnectionStatus.connected,
  householdTimeZone: 'Europe/Bucharest',
  selectedCalendar: const CalendarSourceCalendar(
    id: 'family@example.com',
    name: 'Family',
    colorHex: '#4285f4',
  ),
);

final _entries = [
  CalendarEntry(
    id: 'dinner',
    title: 'Family dinner',
    start: DateTime.utc(2026, 8, 21, 15),
    end: DateTime.utc(2026, 8, 21, 17),
    allDay: false,
    location: 'Home',
    colorHex: '#4285f4',
  ),
  CalendarEntry(
    id: 'family-day',
    title: 'Family day',
    start: DateTime.utc(2026, 8, 21),
    end: DateTime.utc(2026, 8, 23),
    allDay: true,
    colorHex: '#d50000',
  ),
];

InMemoryHouseholdGatewayState _state({
  CalendarConnection? connection,
  List<CalendarEntry> entries = const [],
  HouseholdGatewayFailure? failure,
  HouseholdSession? session,
}) {
  return InMemoryHouseholdGatewayState(
    storedSession: session ?? _session,
    calendarConnection: connection ?? _connected,
    calendarEntries: entries,
    calendarEntryFailure: failure,
  );
}

_Story _story(
  InMemoryHouseholdGatewayState state, {
  HouseholdSession? session,
  bool pause = false,
  List<_Scenario>? scenarios,
}) {
  return _Story(
    args: _Args.fixed(
      gateway: InMemoryHouseholdGateway(
        state: state,
        pauseCalendarEntries: pause,
      ),
      session: session ?? _session,
      openAuthorization: _openAuthorization,
      now: _now,
    ),
    scenarios:
        scenarios ??
        [
          _Scenario(
            name: 'Default',
            run: (tester, _) async => tester.pumpAndSettle(),
          ),
        ],
  );
}

final $Loading = _story(_state(entries: _entries), pause: true);
final $Populated = _story(_state(entries: _entries));
final $EmptyDay = _story(
  _state(
    entries: [
      CalendarEntry(
        id: 'tomorrow',
        title: 'Tomorrow',
        start: DateTime.utc(2026, 8, 22, 10),
        end: DateTime.utc(2026, 8, 22, 11),
        allDay: false,
        colorHex: '#4285f4',
      ),
    ],
  ),
);
final $EmptyMonth = _story(_state());
final $DisconnectedAdministrator = _story(
  _state(
    connection: CalendarConnection(
      provider: 'google',
      status: CalendarConnectionStatus.disconnected,
      householdTimeZone: 'Europe/Bucharest',
    ),
  ),
);
final $DisconnectedMember = _story(
  _state(
    connection: CalendarConnection(
      provider: 'google',
      status: CalendarConnectionStatus.disconnected,
      householdTimeZone: 'Europe/Bucharest',
    ),
    session: _memberSession,
  ),
  session: _memberSession,
);
final $AuthorizationFailure = _story(
  _state(
    connection: CalendarConnection(
      provider: 'google',
      status: CalendarConnectionStatus.authorizationFailed,
      householdTimeZone: 'Europe/Bucharest',
    ),
  ),
);
final $ServerFailure = _story(_state(failure: HouseholdGatewayFailure.server));
final $RefreshFailure = _story(
  _state(entries: _entries),
  scenarios: [
    _Scenario(
      name: 'Failed refresh',
      run: (tester, args) async {
        await tester.pumpAndSettle();
        final gateway = args.gateway as InMemoryHouseholdGateway;
        gateway.state.calendarEntryFailure = HouseholdGatewayFailure.server;
        await tester.tap(find.byKey(const Key('refresh-calendar')));
        await tester.pumpAndSettle();
      },
    ),
  ],
);

DateTime _now() => DateTime.utc(2026, 8, 21, 9);
Future<bool> _openAuthorization(Uri _) async => true;
