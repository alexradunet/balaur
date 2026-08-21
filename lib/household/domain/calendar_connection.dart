import 'package:flutter/foundation.dart';

enum CalendarConnectionStatus {
  disconnected,
  authorizing,
  awaitingSelection,
  connected,
  authorizationFailed,
}

@immutable
final class CalendarSourceCalendar {
  const CalendarSourceCalendar({
    required this.id,
    required this.name,
    this.colorHex,
  });

  final String id;
  final String name;
  final String? colorHex;

  @override
  bool operator ==(Object other) {
    return other is CalendarSourceCalendar &&
        other.id == id &&
        other.name == name &&
        other.colorHex == colorHex;
  }

  @override
  int get hashCode => Object.hash(id, name, colorHex);
}

@immutable
final class CalendarConnection {
  CalendarConnection({
    required this.provider,
    required this.status,
    required this.householdTimeZone,
    this.selectedCalendar,
    List<CalendarSourceCalendar> availableCalendars = const [],
  }) : availableCalendars = List.unmodifiable(availableCalendars);

  final String provider;
  final CalendarConnectionStatus status;
  final String householdTimeZone;
  final CalendarSourceCalendar? selectedCalendar;
  final List<CalendarSourceCalendar> availableCalendars;

  CalendarConnection copyWith({
    CalendarConnectionStatus? status,
    String? householdTimeZone,
    CalendarSourceCalendar? selectedCalendar,
    bool clearSelectedCalendar = false,
    List<CalendarSourceCalendar>? availableCalendars,
  }) {
    return CalendarConnection(
      provider: provider,
      status: status ?? this.status,
      householdTimeZone: householdTimeZone ?? this.householdTimeZone,
      selectedCalendar: clearSelectedCalendar
          ? null
          : selectedCalendar ?? this.selectedCalendar,
      availableCalendars: availableCalendars ?? this.availableCalendars,
    );
  }

  @override
  bool operator ==(Object other) {
    return other is CalendarConnection &&
        other.provider == provider &&
        other.status == status &&
        other.householdTimeZone == householdTimeZone &&
        other.selectedCalendar == selectedCalendar &&
        listEquals(other.availableCalendars, availableCalendars);
  }

  @override
  int get hashCode => Object.hash(
    provider,
    status,
    householdTimeZone,
    selectedCalendar,
    Object.hashAll(availableCalendars),
  );
}
