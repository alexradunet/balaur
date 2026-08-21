import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/calendar_connection_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('connect opens authorization in the system browser', (
    tester,
  ) async {
    final state = _state(_connection(CalendarConnectionStatus.disconnected));
    Uri? openedUri;
    await _pumpScreen(
      tester,
      state: state,
      openAuthorization: (uri) async {
        openedUri = uri;
        return true;
      },
    );

    await tester.tap(find.byKey(const Key('connect-google-calendar')));
    await tester.pumpAndSettle();

    expect(openedUri?.scheme, 'https');
    expect(find.text('Authorization in progress'), findsOneWidget);
    expect(
      find.byKey(const Key('check-calendar-connection-status')),
      findsOneWidget,
    );
  });

  testWidgets('selects one calendar and disconnects it', (tester) async {
    final state = _state(
      _connection(CalendarConnectionStatus.awaitingSelection),
    );
    await _pumpScreen(tester, state: state);

    await tester.tap(
      find.byKey(const Key('select-calendar-family@example.com')),
    );
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('selected-calendar-name')), findsOneWidget);
    expect(find.text('Family'), findsOneWidget);

    final disconnect = find.byKey(const Key('disconnect-calendar-connection'));
    await tester.ensureVisible(disconnect);
    await tester.tap(disconnect);
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('confirm-disconnect-calendar-connection')),
    );
    await tester.pumpAndSettle();

    expect(find.text('Disconnected'), findsOneWidget);
    expect(state.calendarConnection.selectedCalendar, isNull);
  });

  testWidgets('shows authorization denial and reconnects', (tester) async {
    final state = _state(
      _connection(CalendarConnectionStatus.authorizationFailed),
    );
    var openCount = 0;
    await _pumpScreen(
      tester,
      state: state,
      openAuthorization: (_) async {
        openCount += 1;
        return true;
      },
    );

    expect(find.text('Authorization failed'), findsOneWidget);
    await tester.tap(find.byKey(const Key('reconnect-google-calendar')));
    await tester.pumpAndSettle();

    expect(openCount, 1);
    expect(find.text('Authorization in progress'), findsOneWidget);
  });

  testWidgets('denies Calendar Connection management to a Household Member', (
    tester,
  ) async {
    final memberSession = HouseholdSession(
      serverAddress: _address,
      member: const HouseholdMember(
        id: 'member-example',
        displayName: 'Mara',
        email: 'mara@example.com',
        role: HouseholdMemberRole.member,
      ),
    );
    final state = InMemoryHouseholdGatewayState(
      storedSession: memberSession,
      calendarConnection: _connection(CalendarConnectionStatus.disconnected),
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: BalaurTheme.light(),
        home: CalendarConnectionScreen(
          gateway: InMemoryHouseholdGateway(state: state),
          session: memberSession,
          onBack: () {},
          openAuthorization: (_) async => true,
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text('HOUSEHOLD ADMINISTRATOR ACCESS IS REQUIRED'),
      findsOneWidget,
    );
    expect(find.byKey(const Key('connect-google-calendar')), findsNothing);
  });
}

final _address = HouseholdServerAddress.parse('https://household.example.com');

final _administratorSession = HouseholdSession(
  serverAddress: _address,
  member: const HouseholdMember(
    id: 'administrator-example',
    displayName: 'Alex',
    email: 'alex@example.com',
    role: HouseholdMemberRole.administrator,
  ),
);

const _calendars = [
  CalendarSourceCalendar(
    id: 'family@example.com',
    name: 'Family',
    colorHex: '#4285f4',
  ),
  CalendarSourceCalendar(
    id: 'school@example.com',
    name: 'School',
    colorHex: '#d50000',
  ),
];

CalendarConnection _connection(CalendarConnectionStatus status) {
  return CalendarConnection(
    provider: 'google',
    status: status,
    householdTimeZone: 'Europe/Bucharest',
    availableCalendars: status == CalendarConnectionStatus.awaitingSelection
        ? _calendars
        : const [],
  );
}

InMemoryHouseholdGatewayState _state(CalendarConnection connection) {
  return InMemoryHouseholdGatewayState(
    storedSession: _administratorSession,
    calendarConnection: connection,
  );
}

Future<void> _pumpScreen(
  WidgetTester tester, {
  required InMemoryHouseholdGatewayState state,
  Future<bool> Function(Uri uri)? openAuthorization,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: BalaurTheme.light(),
      home: CalendarConnectionScreen(
        gateway: InMemoryHouseholdGateway(state: state),
        session: _administratorSession,
        onBack: () {},
        openAuthorization: openAuthorization ?? (_) async => true,
      ),
    ),
  );
  await tester.pumpAndSettle();
}
