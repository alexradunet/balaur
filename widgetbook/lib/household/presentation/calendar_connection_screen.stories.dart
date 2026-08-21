import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/calendar_connection_screen.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'calendar_connection_screen.stories.g.dart';

const component = ComponentMeta(
  name: 'Calendar Connection',
  path: 'Screens/Household',
);
const meta = Meta(CalendarConnectionScreen.new);

final _session = HouseholdSession(
  serverAddress: HouseholdServerAddress.parse('https://household.example.com'),
  member: const HouseholdMember(
    id: 'administrator-example',
    displayName: 'Alex',
    email: 'alex@example.com',
    role: HouseholdMemberRole.administrator,
  ),
);

const _familyCalendar = CalendarSourceCalendar(
  id: 'family@example.com',
  name: 'Family',
  colorHex: '#4285f4',
);

InMemoryHouseholdGateway _gateway(CalendarConnection connection) {
  return InMemoryHouseholdGateway(
    state: InMemoryHouseholdGatewayState(
      storedSession: _session,
      calendarConnection: connection,
    ),
  );
}

CalendarConnection _connection(CalendarConnectionStatus status) {
  return CalendarConnection(
    provider: 'google',
    status: status,
    householdTimeZone: 'Europe/Bucharest',
    selectedCalendar: status == CalendarConnectionStatus.connected
        ? _familyCalendar
        : null,
    availableCalendars: status == CalendarConnectionStatus.awaitingSelection
        ? const [
            _familyCalendar,
            CalendarSourceCalendar(
              id: 'school@example.com',
              name: 'School',
              colorHex: '#d50000',
            ),
          ]
        : const [],
  );
}

final $Disconnected = _story(CalendarConnectionStatus.disconnected);
final $Authorizing = _story(CalendarConnectionStatus.authorizing);
final $SelectCalendar = _story(CalendarConnectionStatus.awaitingSelection);
final $Connected = _story(CalendarConnectionStatus.connected);
final $AuthorizationFailed = _story(
  CalendarConnectionStatus.authorizationFailed,
);

_Story _story(CalendarConnectionStatus status) {
  return _Story(
    args: _Args.fixed(
      gateway: _gateway(_connection(status)),
      session: _session,
      onBack: _noOp,
      openAuthorization: _openAuthorization,
    ),
    scenarios: [
      _Scenario(
        name: 'Default',
        run: (tester, _) async => tester.pumpAndSettle(),
      ),
    ],
  );
}

void _noOp() {}

Future<bool> _openAuthorization(Uri _) async => true;
