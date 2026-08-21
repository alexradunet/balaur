import 'package:flutter/foundation.dart';

@immutable
final class CalendarEntry {
  CalendarEntry({
    required this.id,
    required this.title,
    required this.start,
    required this.end,
    required this.allDay,
    required this.colorHex,
    this.location,
  }) {
    if (id.isEmpty || title.isEmpty || !end.isAfter(start)) {
      throw const FormatException('Invalid Calendar Entry.');
    }
    if (allDay &&
        (start.isUtc == false ||
            end.isUtc == false ||
            start.hour != 0 ||
            start.minute != 0 ||
            end.hour != 0 ||
            end.minute != 0)) {
      throw const FormatException('Invalid all-day Calendar Entry.');
    }
  }

  final String id;
  final String title;
  final DateTime start;
  final DateTime end;
  final bool allDay;
  final String? location;
  final String colorHex;

  @override
  bool operator ==(Object other) {
    return other is CalendarEntry &&
        other.id == id &&
        other.title == title &&
        other.start == start &&
        other.end == end &&
        other.allDay == allDay &&
        other.location == location &&
        other.colorHex == colorHex;
  }

  @override
  int get hashCode =>
      Object.hash(id, title, start, end, allDay, location, colorHex);
}
