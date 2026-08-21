import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_archive.dart';
import 'package:flutter/foundation.dart';

enum HouseholdArchiveStatus { idle, exporting, saving, success, failed }

final class HouseholdArchiveViewModel extends ChangeNotifier {
  HouseholdArchiveViewModel(this._gateway, this._saveArchive);

  final HouseholdGateway _gateway;
  final Future<bool> Function(HouseholdArchive archive) _saveArchive;

  HouseholdArchiveStatus _status = HouseholdArchiveStatus.idle;
  String? _message;
  String? _fileName;

  HouseholdArchiveStatus get status => _status;
  String? get message => _message;
  String? get fileName => _fileName;
  bool get isBusy =>
      _status == HouseholdArchiveStatus.exporting ||
      _status == HouseholdArchiveStatus.saving;

  Future<void> export({
    required String password,
    required bool includeCalendarSnapshot,
  }) async {
    if (isBusy) {
      return;
    }
    if (password.isEmpty) {
      _status = HouseholdArchiveStatus.failed;
      _message = 'Enter the Household Administrator password.';
      notifyListeners();
      return;
    }
    _status = HouseholdArchiveStatus.exporting;
    _message = 'The Household Server is creating the Household Archive.';
    _fileName = null;
    notifyListeners();
    try {
      final archive = await _gateway.exportHouseholdArchive(
        password: password,
        includeCalendarSnapshot: includeCalendarSnapshot,
      );
      _status = HouseholdArchiveStatus.saving;
      _message = 'Select where to save the Household Archive.';
      notifyListeners();
      final saved = await _saveArchive(archive);
      if (!saved) {
        _status = HouseholdArchiveStatus.idle;
        _message = 'Household Archive export was canceled.';
        notifyListeners();
        return;
      }
      _status = HouseholdArchiveStatus.success;
      _fileName = archive.fileName;
      _message = 'The Household Archive was saved.';
    } on HouseholdGatewayException catch (error) {
      _status = HouseholdArchiveStatus.failed;
      _message = switch (error.failure) {
        HouseholdGatewayFailure.authentication =>
          'The Household Administrator password is not correct.',
        HouseholdGatewayFailure.forbidden =>
          'Only a Household Administrator can export a Household Archive.',
        HouseholdGatewayFailure.invalidInput =>
          'Connect the Calendar Source or omit the calendar snapshot.',
        _ => 'The Household Server could not create the Household Archive.',
      };
    } on Object {
      _status = HouseholdArchiveStatus.failed;
      _message = 'Balaur could not save the Household Archive.';
    }
    notifyListeners();
  }
}
