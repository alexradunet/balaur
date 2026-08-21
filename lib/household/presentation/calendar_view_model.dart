import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/calendar_entry.dart';
import 'package:flutter/foundation.dart';
import 'package:timezone/data/latest.dart' as time_zone_data;
import 'package:timezone/timezone.dart' as time_zone;

enum CalendarVisibleStatus {
  loading,
  populated,
  emptyDay,
  emptyMonth,
  disconnected,
  authorizationFailure,
  serverFailure,
  refreshFailure,
}

@immutable
final class CalendarViewState {
  CalendarViewState({
    required this.status,
    required this.displayMonth,
    required this.selectedDate,
    required this.householdTimeZone,
    List<CalendarEntry> entries = const [],
    this.isRefreshing = false,
    this.message,
  }) : entries = List.unmodifiable(entries);

  final CalendarVisibleStatus status;
  final DateTime displayMonth;
  final DateTime selectedDate;
  final String householdTimeZone;
  final List<CalendarEntry> entries;
  final bool isRefreshing;
  final String? message;

  CalendarViewState copyWith({
    CalendarVisibleStatus? status,
    DateTime? displayMonth,
    DateTime? selectedDate,
    String? householdTimeZone,
    List<CalendarEntry>? entries,
    bool? isRefreshing,
    String? message,
    bool clearMessage = false,
  }) {
    return CalendarViewState(
      status: status ?? this.status,
      displayMonth: displayMonth ?? this.displayMonth,
      selectedDate: selectedDate ?? this.selectedDate,
      householdTimeZone: householdTimeZone ?? this.householdTimeZone,
      entries: entries ?? this.entries,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      message: clearMessage ? null : message ?? this.message,
    );
  }
}

final class CalendarViewModel extends ChangeNotifier {
  CalendarViewModel(
    this._gateway,
    this._isAdministrator,
    this._openAuthorization, {
    DateTime Function()? now,
  }) : _now = now ?? DateTime.now {
    time_zone_data.initializeTimeZones();
    final current = _dateOnly(_now().toUtc());
    _state = CalendarViewState(
      status: CalendarVisibleStatus.loading,
      displayMonth: DateTime.utc(current.year, current.month),
      selectedDate: current,
      householdTimeZone: 'UTC',
    );
  }

  final HouseholdGateway _gateway;
  final bool _isAdministrator;
  final Future<bool> Function(Uri uri) _openAuthorization;
  final DateTime Function() _now;

  late CalendarViewState _state;
  time_zone.Location _location = time_zone.UTC;
  bool _disposed = false;

  CalendarViewState get state => _state;
  bool get isAdministrator => _isAdministrator;

  DateTime get today {
    final now = time_zone.TZDateTime.from(_now(), _location);
    return DateTime.utc(now.year, now.month, now.day);
  }

  String entryTimeLabel(CalendarEntry entry) {
    if (entry.allDay) {
      return 'All day';
    }
    final start = time_zone.TZDateTime.from(entry.start, _location);
    return '${start.hour.toString().padLeft(2, '0')}:'
        '${start.minute.toString().padLeft(2, '0')}';
  }

  List<DateTime> get visibleDays {
    final first = DateTime.utc(
      _state.displayMonth.year,
      _state.displayMonth.month,
    );
    final gridStart = first.subtract(Duration(days: first.weekday - 1));
    return List.generate(42, (index) => gridStart.add(Duration(days: index)));
  }

  List<CalendarEntry> get selectedEntries {
    return _state.entries
        .where((entry) => _entryOccursOn(entry, _state.selectedDate))
        .toList(growable: false);
  }

  int entryCountFor(DateTime date) {
    return _state.entries.where((entry) => _entryOccursOn(entry, date)).length;
  }

  Future<void> initialize() async {
    await _loadConnectionAndRange(showLoading: true);
  }

  Future<void> previousMonth() async {
    final month = DateTime.utc(
      _state.displayMonth.year,
      _state.displayMonth.month - 1,
    );
    _state = _state.copyWith(
      displayMonth: month,
      selectedDate: month,
      status: CalendarVisibleStatus.loading,
      clearMessage: true,
    );
    _notify();
    await _loadConnectionAndRange(showLoading: false);
  }

  Future<void> nextMonth() async {
    final month = DateTime.utc(
      _state.displayMonth.year,
      _state.displayMonth.month + 1,
    );
    _state = _state.copyWith(
      displayMonth: month,
      selectedDate: month,
      status: CalendarVisibleStatus.loading,
      clearMessage: true,
    );
    _notify();
    await _loadConnectionAndRange(showLoading: false);
  }

  Future<void> goToToday() async {
    final now = time_zone.TZDateTime.from(_now(), _location);
    final today = DateTime.utc(now.year, now.month, now.day);
    final changedMonth =
        today.year != _state.displayMonth.year ||
        today.month != _state.displayMonth.month;
    _state = _state.copyWith(
      displayMonth: DateTime.utc(today.year, today.month),
      selectedDate: today,
      clearMessage: true,
    );
    if (changedMonth) {
      _state = _state.copyWith(status: CalendarVisibleStatus.loading);
      _notify();
      await _loadConnectionAndRange(showLoading: false);
    } else {
      _applyContentStatus();
      _notify();
    }
  }

  Future<void> selectDate(DateTime date) async {
    final selected = _dateOnly(date);
    final changedMonth =
        selected.year != _state.displayMonth.year ||
        selected.month != _state.displayMonth.month;
    _state = _state.copyWith(
      selectedDate: selected,
      displayMonth: changedMonth
          ? DateTime.utc(selected.year, selected.month)
          : _state.displayMonth,
      clearMessage: true,
    );
    if (changedMonth) {
      _state = _state.copyWith(status: CalendarVisibleStatus.loading);
      _notify();
      await _loadConnectionAndRange(showLoading: false);
    } else {
      _applyContentStatus();
      _notify();
    }
  }

  Future<void> refresh() async {
    if (_state.isRefreshing) {
      return;
    }
    _state = _state.copyWith(isRefreshing: true, clearMessage: true);
    _notify();
    await _loadConnectionAndRange(showLoading: false, refreshing: true);
  }

  Future<void> reconnect() async {
    if (!_isAdministrator) {
      return;
    }
    try {
      final uri = await _gateway.beginCalendarConnection(replace: false);
      final opened = await _openAuthorization(uri);
      _state = _state.copyWith(
        status: opened
            ? CalendarVisibleStatus.disconnected
            : CalendarVisibleStatus.authorizationFailure,
        message: opened
            ? 'Complete authorization in the system browser. Then refresh the calendar.'
            : 'Balaur could not open the system browser.',
      );
    } on Object {
      _state = _state.copyWith(
        status: CalendarVisibleStatus.authorizationFailure,
        message: 'Balaur could not start Calendar Connection authorization.',
      );
    }
    _notify();
  }

  Future<void> _loadConnectionAndRange({
    required bool showLoading,
    bool refreshing = false,
  }) async {
    if (showLoading) {
      _state = _state.copyWith(
        status: CalendarVisibleStatus.loading,
        clearMessage: true,
      );
      _notify();
    }
    try {
      final connection = await _gateway.loadCalendarConnection();
      _setTimeZone(connection.householdTimeZone);
      if (connection.status == CalendarConnectionStatus.authorizationFailed) {
        _state = _state.copyWith(
          status: CalendarVisibleStatus.authorizationFailure,
          isRefreshing: false,
          message: 'The Calendar Connection authorization failed.',
        );
        _notify();
        return;
      }
      if (connection.status != CalendarConnectionStatus.connected) {
        _state = _state.copyWith(
          status: CalendarVisibleStatus.disconnected,
          isRefreshing: false,
          message: 'The Household has no connected Calendar Source.',
        );
        _notify();
        return;
      }

      final days = visibleDays;
      final rangeStart = _startOfDay(days.first);
      final rangeEnd = _startOfDay(days.last.add(const Duration(days: 1)));
      final entries = await _gateway.loadCalendarEntries(
        rangeStart: rangeStart,
        rangeEnd: rangeEnd,
      );
      _state = _state.copyWith(
        entries: entries,
        isRefreshing: false,
        clearMessage: true,
      );
      _applyContentStatus();
    } on HouseholdGatewayException catch (error) {
      _applyFailure(error.failure, refreshing: refreshing);
    } on Object {
      _applyFailure(HouseholdGatewayFailure.server, refreshing: refreshing);
    }
    _notify();
  }

  void _setTimeZone(String name) {
    _location = name == 'UTC' ? time_zone.UTC : time_zone.getLocation(name);
    _state = _state.copyWith(householdTimeZone: name);
    final now = time_zone.TZDateTime.from(_now(), _location);
    if (_state.status == CalendarVisibleStatus.loading &&
        _state.entries.isEmpty) {
      final today = DateTime.utc(now.year, now.month, now.day);
      _state = _state.copyWith(
        displayMonth: DateTime.utc(today.year, today.month),
        selectedDate: today,
      );
    }
  }

  void _applyFailure(
    HouseholdGatewayFailure failure, {
    required bool refreshing,
  }) {
    if (failure == HouseholdGatewayFailure.calendarAuthorization) {
      _state = _state.copyWith(
        status: CalendarVisibleStatus.disconnected,
        isRefreshing: false,
        message: 'The Calendar Connection is disconnected.',
      );
      return;
    }
    if (refreshing && _state.entries.isNotEmpty) {
      _state = _state.copyWith(
        status: CalendarVisibleStatus.refreshFailure,
        isRefreshing: false,
        message: 'Refresh failed. Balaur kept the current Calendar Entries.',
      );
      return;
    }
    _state = _state.copyWith(
      status: CalendarVisibleStatus.serverFailure,
      isRefreshing: false,
      message:
          'Balaur could not load Calendar Entries from the Household Server.',
    );
  }

  void _applyContentStatus() {
    final status = _state.entries.isEmpty
        ? CalendarVisibleStatus.emptyMonth
        : selectedEntries.isEmpty
        ? CalendarVisibleStatus.emptyDay
        : CalendarVisibleStatus.populated;
    _state = _state.copyWith(status: status, isRefreshing: false);
  }

  bool _entryOccursOn(CalendarEntry entry, DateTime date) {
    if (entry.allDay) {
      final day = _dateOnly(date);
      return !day.isBefore(_dateOnly(entry.start)) &&
          day.isBefore(_dateOnly(entry.end));
    }
    final dayStart = _startOfDay(date);
    final dayEnd = _startOfDay(date.add(const Duration(days: 1)));
    return entry.start.isBefore(dayEnd) && entry.end.isAfter(dayStart);
  }

  DateTime _startOfDay(DateTime date) {
    return time_zone.TZDateTime(
      _location,
      date.year,
      date.month,
      date.day,
    ).toUtc();
  }

  static DateTime _dateOnly(DateTime value) {
    return DateTime.utc(value.year, value.month, value.day);
  }

  void _notify() {
    if (!_disposed) {
      notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
