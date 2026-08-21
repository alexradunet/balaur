import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/calendar_entry.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/calendar_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows a month and updates the selected-day agenda', (
    tester,
  ) async {
    await _pumpCalendar(tester);

    expect(find.text('August 2026'), findsOneWidget);
    expect(find.text('Family dinner'), findsOneWidget);

    await tester.tap(find.byKey(const Key('calendar-day-2026-08-22')));
    await tester.pumpAndSettle();

    expect(find.text('Saturday, August 22, 2026'), findsOneWidget);
    expect(find.text('Family dinner'), findsNothing);
    expect(find.text('Family day'), findsOneWidget);
  });

  testWidgets('uses month navigation and Today controls', (tester) async {
    await _pumpCalendar(tester);

    await tester.tap(find.byKey(const Key('next-calendar-month')));
    await tester.pumpAndSettle();
    expect(find.text('September 2026'), findsOneWidget);

    await tester.tap(find.byKey(const Key('previous-calendar-month')));
    await tester.pumpAndSettle();
    expect(find.text('August 2026'), findsOneWidget);

    await tester.tap(find.byKey(const Key('calendar-today')));
    await tester.pumpAndSettle();
    expect(find.text('Friday, August 21, 2026'), findsOneWidget);
  });

  testWidgets('provides full date and Calendar Entry semantics', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await _pumpCalendar(tester);

    final node = tester.getSemantics(
      find.byKey(const Key('calendar-day-2026-08-21')),
    );
    expect(node.label, contains('Friday, August 21, 2026'));
    expect(node.label, contains('2 Calendar Entries'));
    expect(node.label, contains('today'));
    expect(node.label, contains('selected'));
    semantics.dispose();
  });

  testWidgets('supports keyboard month navigation', (tester) async {
    await _pumpCalendar(tester);

    for (var index = 0; index < 4; index += 1) {
      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.pump();
    }
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();

    expect(find.text('September 2026'), findsOneWidget);
  });

  testWidgets('stacks the month and agenda on a compact layout', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(360, 1100);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await _pumpCalendar(tester);

    final month = tester.getRect(find.byKey(const Key('calendar-month-panel')));
    final agenda = tester.getRect(
      find.byKey(const Key('calendar-agenda-panel')),
    );
    expect(agenda.top, greaterThan(month.bottom));
    final daySize = tester.getSize(
      find.byKey(const Key('calendar-day-2026-08-21')),
    );
    expect(daySize.width, greaterThanOrEqualTo(48));
    expect(daySize.height, greaterThanOrEqualTo(48));
  });

  testWidgets('places the month and agenda beside each other when wide', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1200, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await _pumpCalendar(tester);

    final month = tester.getRect(find.byKey(const Key('calendar-month-panel')));
    final agenda = tester.getRect(
      find.byKey(const Key('calendar-agenda-panel')),
    );
    expect((agenda.top - month.top).abs(), lessThan(1));
    expect(agenda.left, greaterThan(month.right));
    for (final key in const [
      'previous-calendar-month',
      'calendar-today',
      'next-calendar-month',
    ]) {
      final size = tester.getSize(find.byKey(Key(key)));
      expect(size.width, greaterThanOrEqualTo(48));
      expect(size.height, greaterThanOrEqualTo(48));
    }
  });

  testWidgets('shows reconnect actions according to Household role', (
    tester,
  ) async {
    await _pumpCalendar(
      tester,
      connectionStatus: CalendarConnectionStatus.disconnected,
    );
    expect(find.byKey(const Key('reconnect-calendar')), findsOneWidget);

    await _pumpCalendar(
      tester,
      connectionStatus: CalendarConnectionStatus.disconnected,
      session: _memberSession,
    );
    expect(find.byKey(const Key('reconnect-calendar')), findsNothing);
    expect(
      find.textContaining('Ask a Household Administrator'),
      findsOneWidget,
    );
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

final _memberSession = HouseholdSession(
  serverAddress: _address,
  member: const HouseholdMember(
    id: 'member-example',
    displayName: 'Mara',
    email: 'mara@example.com',
    role: HouseholdMemberRole.member,
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

Future<void> _pumpCalendar(
  WidgetTester tester, {
  CalendarConnectionStatus connectionStatus =
      CalendarConnectionStatus.connected,
  HouseholdSession? session,
}) async {
  final activeSession = session ?? _administratorSession;
  final state = InMemoryHouseholdGatewayState(
    storedSession: activeSession,
    calendarConnection: CalendarConnection(
      provider: 'google',
      status: connectionStatus,
      householdTimeZone: 'Europe/Bucharest',
      selectedCalendar: connectionStatus == CalendarConnectionStatus.connected
          ? const CalendarSourceCalendar(
              id: 'family@example.com',
              name: 'Family',
              colorHex: '#4285f4',
            )
          : null,
    ),
    calendarEntries: _entries,
  );
  await tester.pumpWidget(
    MaterialApp(
      theme: BalaurTheme.light(),
      home: CalendarScreen(
        gateway: InMemoryHouseholdGateway(state: state),
        session: activeSession,
        openAuthorization: (_) async => true,
        now: () => DateTime.utc(2026, 8, 21, 9),
      ),
    ),
  );
  await tester.pumpAndSettle();
}
