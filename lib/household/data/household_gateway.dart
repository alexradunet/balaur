import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';

abstract interface class HouseholdGateway {
  Future<HouseholdSession?> restoreSession();

  Future<HouseholdSession> pair({
    required HouseholdServerAddress serverAddress,
    required String email,
    required String password,
  });

  Future<List<HouseholdInvitation>> listInvitations();

  Future<CreatedHouseholdInvitation> createInvitation({
    required HouseholdMemberRole role,
  });

  Future<void> cancelInvitation(String invitationId);

  Future<HouseholdSession> redeemInvitation({
    required HouseholdInvitationPayload invitation,
    required String displayName,
    required String email,
    required String password,
  });

  Future<void> signOut();
}

enum HouseholdGatewayFailure {
  connection,
  authentication,
  storage,
  invalidSession,
  invalidInput,
  forbidden,
  invalidInvitation,
  expiredInvitation,
  unavailableInvitation,
}

final class HouseholdGatewayException implements Exception {
  const HouseholdGatewayException(this.failure);

  final HouseholdGatewayFailure failure;

  @override
  String toString() => 'Household Gateway failed: ${failure.name}';
}
