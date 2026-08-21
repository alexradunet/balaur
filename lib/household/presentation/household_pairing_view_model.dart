import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:flutter/foundation.dart';

enum HouseholdPairingStatus { connecting, unpaired, failed, paired }

final class HouseholdPairingViewModel extends ChangeNotifier {
  HouseholdPairingViewModel({required this._gateway});

  final HouseholdGateway _gateway;

  HouseholdPairingStatus _status = HouseholdPairingStatus.connecting;
  HouseholdPairingStatus get status => _status;

  HouseholdSession? _session;
  HouseholdSession? get session => _session;

  String? _errorMessage;
  String? get errorMessage => _errorMessage;

  bool _disposed = false;

  Future<void> initialize() async {
    _setConnecting();
    try {
      _session = await _gateway.restoreSession();
      _status = _session == null
          ? HouseholdPairingStatus.unpaired
          : HouseholdPairingStatus.paired;
    } on HouseholdGatewayException catch (error) {
      _status = HouseholdPairingStatus.failed;
      _errorMessage = _messageFor(error.failure, restoring: true);
    } on Object {
      _status = HouseholdPairingStatus.failed;
      _errorMessage = 'Balaur could not restore the Household session.';
    }
    _notify();
  }

  Future<void> pair({
    required HouseholdServerAddress serverAddress,
    required String email,
    required String password,
  }) async {
    _setConnecting();
    try {
      _session = await _gateway.pair(
        serverAddress: serverAddress,
        email: email,
        password: password,
      );
      _status = HouseholdPairingStatus.paired;
    } on HouseholdGatewayException catch (error) {
      _status = HouseholdPairingStatus.failed;
      _errorMessage = _messageFor(error.failure, restoring: false);
    } on Object {
      _status = HouseholdPairingStatus.failed;
      _errorMessage = 'Balaur could not pair this device.';
    }
    _notify();
  }

  Future<void> redeemInvitation({
    required HouseholdInvitationPayload invitation,
    required String displayName,
    required String email,
    required String password,
  }) async {
    _setConnecting();
    try {
      _session = await _gateway.redeemInvitation(
        invitation: invitation,
        displayName: displayName,
        email: email,
        password: password,
      );
      _status = HouseholdPairingStatus.paired;
    } on HouseholdGatewayException catch (error) {
      _status = HouseholdPairingStatus.failed;
      _errorMessage = _messageFor(error.failure, restoring: false);
    } on Object {
      _status = HouseholdPairingStatus.failed;
      _errorMessage = 'Balaur could not use the Household Invitation.';
    }
    _notify();
  }

  Future<void> signOut() async {
    _session = null;
    _setConnecting();
    try {
      await _gateway.signOut();
      _status = HouseholdPairingStatus.unpaired;
    } on HouseholdGatewayException catch (error) {
      _status = HouseholdPairingStatus.failed;
      _errorMessage = _messageFor(error.failure, restoring: false);
    } on Object {
      _status = HouseholdPairingStatus.failed;
      _errorMessage = 'Balaur could not clear the Household session.';
    }
    _notify();
  }

  void _setConnecting() {
    _status = HouseholdPairingStatus.connecting;
    _errorMessage = null;
    _notify();
  }

  String _messageFor(
    HouseholdGatewayFailure failure, {
    required bool restoring,
  }) {
    return switch (failure) {
      HouseholdGatewayFailure.connection =>
        restoring ? 'Balaur could not reach the saved Household Server.' : 'Balaur could not reach the Household Server. Check its address and your connection.',
      HouseholdGatewayFailure.authentication =>
        'The email address or password is not correct.',
      HouseholdGatewayFailure.storage =>
        'Balaur could not use secure storage for the Household session.',
      HouseholdGatewayFailure.invalidSession =>
        'The Household Server returned invalid data.',
      HouseholdGatewayFailure.invalidInput => 'The Household Server rejected the member details. Check the email address and password.',
      HouseholdGatewayFailure.forbidden =>
        'Only a Household Administrator can complete this action.',
      HouseholdGatewayFailure.invalidInvitation =>
        'The Household Invitation is not valid.',
      HouseholdGatewayFailure.expiredInvitation => 'The Household Invitation expired. Ask a Household Administrator for a new one.',
      HouseholdGatewayFailure.unavailableInvitation =>
        'The Household Invitation was used or canceled.',
    };
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
