import 'package:balaur/household/domain/household_server_address.dart';
import 'package:flutter/foundation.dart';

enum HouseholdMemberRole { member, administrator }

@immutable
final class HouseholdMember {
  const HouseholdMember({
    required this.id,
    required this.displayName,
    required this.email,
    required this.role,
  });

  final String id;
  final String displayName;
  final String email;
  final HouseholdMemberRole role;

  @override
  bool operator ==(Object other) {
    return other is HouseholdMember &&
        other.id == id &&
        other.displayName == displayName &&
        other.email == email &&
        other.role == role;
  }

  @override
  int get hashCode => Object.hash(id, displayName, email, role);
}

@immutable
final class HouseholdSession {
  const HouseholdSession({required this.serverAddress, required this.member});

  final HouseholdServerAddress serverAddress;
  final HouseholdMember member;

  @override
  bool operator ==(Object other) {
    return other is HouseholdSession &&
        other.serverAddress == serverAddress &&
        other.member == member;
  }

  @override
  int get hashCode => Object.hash(serverAddress, member);
}
