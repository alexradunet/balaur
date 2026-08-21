import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:flutter/foundation.dart';

enum HouseholdInvitationsStatus { loading, ready, failed }

final class HouseholdInvitationsViewModel extends ChangeNotifier {
  HouseholdInvitationsViewModel({required this._gateway});

  final HouseholdGateway _gateway;

  HouseholdInvitationsStatus _status = HouseholdInvitationsStatus.loading;
  HouseholdInvitationsStatus get status => _status;

  List<HouseholdInvitation> _invitations = const [];
  List<HouseholdInvitation> get invitations => _invitations;

  CreatedHouseholdInvitation? _createdInvitation;
  CreatedHouseholdInvitation? get createdInvitation => _createdInvitation;

  String? _errorMessage;
  String? get errorMessage => _errorMessage;

  bool _isCreating = false;
  bool get isCreating => _isCreating;

  final Set<String> _cancelingIds = {};
  bool isCanceling(String invitationId) => _cancelingIds.contains(invitationId);

  bool _disposed = false;

  Future<void> initialize() async {
    _status = HouseholdInvitationsStatus.loading;
    _errorMessage = null;
    _notify();
    try {
      _invitations = await _gateway.listInvitations();
      _status = HouseholdInvitationsStatus.ready;
    } on HouseholdGatewayException catch (error) {
      _status = HouseholdInvitationsStatus.failed;
      _errorMessage = _messageFor(error.failure);
    } on Object {
      _status = HouseholdInvitationsStatus.failed;
      _errorMessage = 'Balaur could not load the Household Invitations.';
    }
    _notify();
  }

  Future<void> create(HouseholdMemberRole role) async {
    if (_isCreating) {
      return;
    }
    _isCreating = true;
    _errorMessage = null;
    _notify();
    try {
      final created = await _gateway.createInvitation(role: role);
      _createdInvitation = created;
      _invitations = [
        created.invitation,
        ..._invitations.where(
          (invitation) => invitation.id != created.invitation.id,
        ),
      ];
      _status = HouseholdInvitationsStatus.ready;
    } on HouseholdGatewayException catch (error) {
      _errorMessage = _messageFor(error.failure);
    } on Object {
      _errorMessage = 'Balaur could not create the Household Invitation.';
    } finally {
      _isCreating = false;
      _notify();
    }
  }

  Future<void> cancel(String invitationId) async {
    if (_cancelingIds.contains(invitationId)) {
      return;
    }
    _cancelingIds.add(invitationId);
    _errorMessage = null;
    _notify();
    try {
      await _gateway.cancelInvitation(invitationId);
      _invitations = _invitations
          .map(
            (invitation) => invitation.id == invitationId
                ? invitation.copyWith(
                    status: HouseholdInvitationStatus.canceled,
                  )
                : invitation,
          )
          .toList(growable: false);
    } on HouseholdGatewayException catch (error) {
      _errorMessage = _messageFor(error.failure);
    } on Object {
      _errorMessage = 'Balaur could not cancel the Household Invitation.';
    } finally {
      _cancelingIds.remove(invitationId);
      _notify();
    }
  }

  void hideCreatedInvitation() {
    _createdInvitation = null;
    _notify();
  }

  String _messageFor(HouseholdGatewayFailure failure) {
    return switch (failure) {
      HouseholdGatewayFailure.connection =>
        'Balaur could not reach the Household Server.',
      HouseholdGatewayFailure.authentication =>
        'The Household session is no longer valid.',
      HouseholdGatewayFailure.storage =>
        'Balaur could not use secure storage for the Household session.',
      HouseholdGatewayFailure.invalidSession =>
        'The Household Server returned invalid data.',
      HouseholdGatewayFailure.invalidInput =>
        'The Household Server rejected the Household Invitation details.',
      HouseholdGatewayFailure.forbidden =>
        'Only a Household Administrator can manage Household Invitations.',
      HouseholdGatewayFailure.invalidInvitation =>
        'The Household Invitation does not exist.',
      HouseholdGatewayFailure.expiredInvitation =>
        'The Household Invitation expired.',
      HouseholdGatewayFailure.unavailableInvitation =>
        'The Household Invitation is not active.',
      HouseholdGatewayFailure.calendarAuthorization =>
        'The Household Server Calendar Connection is not configured.',
      HouseholdGatewayFailure.server =>
        'The Household Server could not complete the request.',
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
