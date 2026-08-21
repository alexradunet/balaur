import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/calendar_entry.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/calendar_view_model.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('loads one selected-day agenda through HouseholdGateway', () async {
    final state = _state(entries: _entries);
    final viewModel = _viewModel(state);
    addTearDown(viewModel.dispose);

    await viewModel.initialize();

    expect(viewModel.state.status, CalendarVisibleStatus.populated);
    expect(viewModel.selectedEntries.map((entry) => entry.id), [
      'multi-day',
      'overnight',
    ]);
    expect(viewModel.state.householdTimeZone, 'Europe/Bucharest');
    expect(state.calendarEntryRanges, hasLength(1));
    expect(
      state.calendarEntryRanges.single.end.difference(
        state.calendarEntryRanges.single.start,
      ),
      const Duration(days: 42),
    );
  });

  test('navigates months and requests only the visible grid', () async {
    final state = _state(entries: _entries);
    final viewModel = _viewModel(state);
    addTearDown(viewModel.dispose);
    await viewModel.initialize();

    await viewModel.nextMonth();

    expect(viewModel.state.displayMonth, DateTime.utc(2026, 9));
    expect(viewModel.visibleDays.first, DateTime.utc(2026, 8, 31));
    expect(viewModel.visibleDays.last, DateTime.utc(2026, 10, 11));
    expect(state.calendarEntryRanges, hasLength(2));
  });

  test('covers leap years, selection, and multi-day overlap', () async {
    final state = _state(
      entries: [
        CalendarEntry(
          id: 'leap-entry',
          title: 'Leap day',
          start: DateTime.utc(2028, 2, 29),
          end: DateTime.utc(2028, 3, 2),
          allDay: true,
          colorHex: '#4285f4',
        ),
      ],
    );
    final viewModel = _viewModel(
      state,
      now: () => DateTime.utc(2028, 2, 29, 10),
    );
    addTearDown(viewModel.dispose);
    await viewModel.initialize();

    expect(viewModel.visibleDays, contains(DateTime.utc(2028, 2, 29)));
    expect(viewModel.entryCountFor(DateTime.utc(2028, 3, 1)), 1);
    expect(viewModel.entryCountFor(DateTime.utc(2028, 3, 2)), 0);

    await viewModel.selectDate(DateTime.utc(2028, 3, 1));
    expect(viewModel.state.displayMonth, DateTime.utc(2028, 3));
    expect(viewModel.selectedEntries.single.id, 'leap-entry');
  });

  test('groups timed entries across a daylight-saving transition', () async {
    final state = _state(
      entries: [
        CalendarEntry(
          id: 'dst-entry',
          title: 'Night shift',
          start: DateTime.utc(2026, 3, 28, 23, 30),
          end: DateTime.utc(2026, 3, 29, 1, 30),
          allDay: false,
          colorHex: '#4285f4',
        ),
      ],
    );
    final viewModel = _viewModel(
      state,
      now: () => DateTime.utc(2026, 3, 29, 9),
    );
    addTearDown(viewModel.dispose);

    await viewModel.initialize();

    expect(viewModel.entryCountFor(DateTime.utc(2026, 3, 29)), 1);
    expect(viewModel.entryTimeLabel(viewModel.selectedEntries.single), '01:30');
  });

  test('keeps current entries when manual refresh fails', () async {
    final state = _state(entries: _entries);
    final viewModel = _viewModel(state);
    addTearDown(viewModel.dispose);
    await viewModel.initialize();
    final currentEntries = viewModel.state.entries;
    state.calendarEntryFailure = HouseholdGatewayFailure.server;

    await viewModel.refresh();

    expect(viewModel.state.status, CalendarVisibleStatus.refreshFailure);
    expect(viewModel.state.entries, currentEntries);
    expect(viewModel.state.message, contains('kept'));
  });

  test('distinguishes disconnected and authorization failure states', () async {
    final disconnectedState = _state(
      connectionStatus: CalendarConnectionStatus.disconnected,
    );
    final disconnected = _viewModel(disconnectedState, administrator: false);
    addTearDown(disconnected.dispose);
    await disconnected.initialize();
    expect(disconnected.state.status, CalendarVisibleStatus.disconnected);

    final failedState = _state(
      connectionStatus: CalendarConnectionStatus.authorizationFailed,
    );
    final failed = _viewModel(failedState);
    addTearDown(failed.dispose);
    await failed.initialize();
    expect(failed.state.status, CalendarVisibleStatus.authorizationFailure);
  });

  test('starts reconnection only for a Household Administrator', () async {
    final state = _state(
      connectionStatus: CalendarConnectionStatus.disconnected,
    );
    var opened = false;
    final administrator = CalendarViewModel(
      InMemoryHouseholdGateway(state: state),
      true,
      (_) async {
        opened = true;
        return true;
      },
      now: _now,
    );
    addTearDown(administrator.dispose);
    await administrator.initialize();

    await administrator.reconnect();

    expect(opened, isTrue);
    expect(
      state.calendarConnection.status,
      CalendarConnectionStatus.authorizing,
    );
  });
}

final _address = HouseholdServerAddress.parse('https://household.example.com');

final _session = HouseholdSession(
  serverAddress: _address,
  member: const HouseholdMember(
    id: 'administrator-example',
    displayName: 'Alex',
    email: 'alex@example.com',
    role: HouseholdMemberRole.administrator,
  ),
);

final _entries = [
  CalendarEntry(
    id: 'multi-day',
    title: 'Family visit',
    start: DateTime.utc(2026, 8, 20),
    end: DateTime.utc(2026, 8, 23),
    allDay: true,
    colorHex: '#d50000',
  ),
  CalendarEntry(
    id: 'overnight',
    title: 'Late train',
    start: DateTime.utc(2026, 8, 21, 20, 30),
    end: DateTime.utc(2026, 8, 21, 22, 30),
    allDay: false,
    colorHex: '#4285f4',
  ),
];

InMemoryHouseholdGatewayState _state({
  List<CalendarEntry> entries = const [],
  CalendarConnectionStatus connectionStatus =
      CalendarConnectionStatus.connected,
}) {
  return InMemoryHouseholdGatewayState(
    storedSession: _session,
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
    calendarEntries: entries,
  );
}

CalendarViewModel _viewModel(
  InMemoryHouseholdGatewayState state, {
  DateTime Function()? now,
  bool administrator = true,
}) {
  return CalendarViewModel(
    InMemoryHouseholdGateway(state: state),
    administrator,
    (_) async => true,
    now: now ?? _now,
  );
}

DateTime _now() => DateTime.utc(2026, 8, 21, 9);
