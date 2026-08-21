import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:flutter/foundation.dart';

final class CalendarConnectionViewModel extends ChangeNotifier {
  CalendarConnectionViewModel(this._gateway, this._openAuthorization);

  final HouseholdGateway _gateway;
  final Future<bool> Function(Uri uri) _openAuthorization;

  CalendarConnection? _connection;
  bool _isLoading = true;
  bool _isBusy = false;
  String? _errorMessage;

  CalendarConnection? get connection => _connection;
  bool get isLoading => _isLoading;
  bool get isBusy => _isBusy;
  String? get errorMessage => _errorMessage;

  Future<void> initialize() => refresh();

  Future<void> refresh() async {
    _isLoading = _connection == null;
    _errorMessage = null;
    notifyListeners();
    try {
      _connection = await _gateway.loadCalendarConnection();
    } on HouseholdGatewayException catch (error) {
      _errorMessage = _messageFor(error.failure);
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> connect({required bool replace}) async {
    if (_isBusy) {
      return;
    }
    _setBusy(true);
    try {
      final uri = await _gateway.beginCalendarConnection(replace: replace);
      _connection = _connection?.copyWith(
        status: CalendarConnectionStatus.authorizing,
        clearSelectedCalendar: true,
        availableCalendars: const [],
      );
      notifyListeners();
      final opened = await _openAuthorization(uri);
      if (!opened) {
        _errorMessage = 'Balaur could not open the system browser.';
      }
    } on HouseholdGatewayException catch (error) {
      _errorMessage = _messageFor(error.failure);
    } on Object {
      _errorMessage = 'Balaur could not open the system browser.';
    } finally {
      _setBusy(false);
    }
  }

  Future<void> selectCalendar(String calendarId) async {
    if (_isBusy) {
      return;
    }
    _setBusy(true);
    try {
      _connection = await _gateway.selectCalendar(calendarId);
    } on HouseholdGatewayException catch (error) {
      _errorMessage = _messageFor(error.failure);
    } finally {
      _setBusy(false);
    }
  }

  Future<void> disconnect() async {
    if (_isBusy) {
      return;
    }
    _setBusy(true);
    try {
      _connection = await _gateway.disconnectCalendarConnection();
    } on HouseholdGatewayException catch (error) {
      _errorMessage = _messageFor(error.failure);
    } finally {
      _setBusy(false);
    }
  }

  void clearError() {
    if (_errorMessage == null) {
      return;
    }
    _errorMessage = null;
    notifyListeners();
  }

  void _setBusy(bool value) {
    _isBusy = value;
    if (value) {
      _errorMessage = null;
    }
    notifyListeners();
  }

  String _messageFor(HouseholdGatewayFailure failure) {
    return switch (failure) {
      HouseholdGatewayFailure.forbidden =>
        'Only a Household Administrator can manage the Calendar Connection.',
      HouseholdGatewayFailure.authentication ||
      HouseholdGatewayFailure.invalidSession =>
        'The Household session is not valid. Pair this device again.',
      HouseholdGatewayFailure.calendarAuthorization => 'Google Calendar authorization is not configured on the Household Server.',
      HouseholdGatewayFailure.invalidInput =>
        'The Calendar Connection request is not valid.',
      _ => 'Balaur could not reach the Household Server.',
    };
  }
}
