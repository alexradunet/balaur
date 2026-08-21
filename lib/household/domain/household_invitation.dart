import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:flutter/foundation.dart';

const _invitationValueLength = 48;

@immutable
final class HouseholdInvitationPayload {
  HouseholdInvitationPayload({
    required this.serverAddress,
    required this.value,
  }) {
    if (!_isValidValue(value)) {
      throw const FormatException('Invalid Household Invitation value.');
    }
  }

  final HouseholdServerAddress serverAddress;
  final String value;

  String get qrValue => Uri(
    scheme: 'balaur',
    host: 'household',
    path: '/join',
    queryParameters: {'server': serverAddress.value, 'invitation': value},
  ).toString();

  static HouseholdInvitationPayload parseQrValue(
    String input, {
    bool allowInsecureLoopback = false,
  }) {
    final uri = Uri.tryParse(input.trim());
    if (uri == null ||
        uri.scheme != 'balaur' ||
        uri.host != 'household' ||
        uri.path != '/join') {
      throw const FormatException('Invalid Household Invitation QR code.');
    }
    final server = uri.queryParameters['server'];
    final invitation = uri.queryParameters['invitation'];
    if (server == null || invitation == null) {
      throw const FormatException('Incomplete Household Invitation QR code.');
    }
    return HouseholdInvitationPayload(
      serverAddress: HouseholdServerAddress.parse(
        server,
        allowInsecureLoopback: allowInsecureLoopback,
      ),
      value: invitation,
    );
  }

  static bool isValidValue(String value) => _isValidValue(value);

  static bool _isValidValue(String value) {
    return value.length == _invitationValueLength &&
        RegExp(r'^[A-Za-z0-9]+$').hasMatch(value);
  }

  @override
  bool operator ==(Object other) {
    return other is HouseholdInvitationPayload &&
        other.serverAddress == serverAddress &&
        other.value == value;
  }

  @override
  int get hashCode => Object.hash(serverAddress, value);
}

enum HouseholdInvitationStatus { active, expired, used, canceled }

@immutable
final class HouseholdInvitation {
  const HouseholdInvitation({
    required this.id,
    required this.creatorId,
    required this.creatorDisplayName,
    required this.expiresAt,
    required this.role,
    required this.status,
  });

  final String id;
  final String creatorId;
  final String creatorDisplayName;
  final DateTime expiresAt;
  final HouseholdMemberRole role;
  final HouseholdInvitationStatus status;

  HouseholdInvitation copyWith({HouseholdInvitationStatus? status}) {
    return HouseholdInvitation(
      id: id,
      creatorId: creatorId,
      creatorDisplayName: creatorDisplayName,
      expiresAt: expiresAt,
      role: role,
      status: status ?? this.status,
    );
  }

  @override
  bool operator ==(Object other) {
    return other is HouseholdInvitation &&
        other.id == id &&
        other.creatorId == creatorId &&
        other.creatorDisplayName == creatorDisplayName &&
        other.expiresAt == expiresAt &&
        other.role == role &&
        other.status == status;
  }

  @override
  int get hashCode =>
      Object.hash(id, creatorId, creatorDisplayName, expiresAt, role, status);
}

@immutable
final class CreatedHouseholdInvitation {
  const CreatedHouseholdInvitation({
    required this.invitation,
    required this.payload,
  });

  final HouseholdInvitation invitation;
  final HouseholdInvitationPayload payload;
}
