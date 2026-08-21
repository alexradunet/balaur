import 'dart:async';

import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';

final class InMemoryHouseholdAccount {
  const InMemoryHouseholdAccount({
    required this.email,
    required this.password,
    required this.member,
  });

  final String email;
  final String password;
  final HouseholdMember member;
}

final class InMemoryHouseholdGatewayState {
  InMemoryHouseholdGatewayState({
    this.storedSession,
    List<HouseholdInvitation> invitations = const [],
  }) : _invitations = List.of(invitations);

  HouseholdSession? storedSession;
  final List<HouseholdInvitation> _invitations;
  final Map<String, HouseholdInvitationPayload> _invitationPayloads = {};
  final List<InMemoryHouseholdAccount> _accounts = [];
  int _nextInvitation = 1;
}

final class InMemoryHouseholdGateway implements HouseholdGateway {
  InMemoryHouseholdGateway({
    List<InMemoryHouseholdAccount> accounts = const [],
    InMemoryHouseholdGatewayState? state,
    this.restoreFailure,
    this.pairFailure,
    this.signOutFailure,
    this.invitationFailure,
    this.redemptionFailure,
    this.pauseRestore = false,
    this.pausePairing = false,
    this.pauseInvitationList = false,
    this.pauseInvitationCancellation = false,
    DateTime Function()? now,
  }) : state = state ?? InMemoryHouseholdGatewayState(),
       _now = now ?? DateTime.now {
    for (final account in accounts) {
      final exists = this.state._accounts.any(
        (candidate) =>
            candidate.email.toLowerCase() == account.email.toLowerCase(),
      );
      if (!exists) {
        this.state._accounts.add(account);
      }
    }
  }

  final InMemoryHouseholdGatewayState state;
  final HouseholdGatewayFailure? restoreFailure;
  final HouseholdGatewayFailure? pairFailure;
  final HouseholdGatewayFailure? signOutFailure;
  final HouseholdGatewayFailure? invitationFailure;
  final HouseholdGatewayFailure? redemptionFailure;
  final bool pauseRestore;
  final bool pausePairing;
  final bool pauseInvitationList;
  final bool pauseInvitationCancellation;
  final DateTime Function() _now;

  @override
  Future<HouseholdSession?> restoreSession() {
    if (pauseRestore) {
      return Completer<HouseholdSession?>().future;
    }
    if (restoreFailure case final failure?) {
      return Future.error(HouseholdGatewayException(failure));
    }
    return Future.value(state.storedSession);
  }

  @override
  Future<HouseholdSession> pair({
    required HouseholdServerAddress serverAddress,
    required String email,
    required String password,
  }) {
    if (pausePairing) {
      return Completer<HouseholdSession>().future;
    }
    if (pairFailure case final failure?) {
      return Future.error(HouseholdGatewayException(failure));
    }

    InMemoryHouseholdAccount? account;
    for (final candidate in state._accounts) {
      if (candidate.email.toLowerCase() == email.trim().toLowerCase() &&
          candidate.password == password) {
        account = candidate;
        break;
      }
    }
    if (account == null) {
      return Future.error(
        const HouseholdGatewayException(HouseholdGatewayFailure.authentication),
      );
    }

    final session = HouseholdSession(
      serverAddress: serverAddress,
      member: account.member,
    );
    state.storedSession = session;
    return Future.value(session);
  }

  @override
  Future<List<HouseholdInvitation>> listInvitations() async {
    if (pauseInvitationList) {
      return Completer<List<HouseholdInvitation>>().future;
    }
    _requireAdministrator();
    _throwInvitationFailure();
    final invitations =
        state._invitations.map(_withCurrentStatus).toList(growable: false)
          ..sort((left, right) => right.expiresAt.compareTo(left.expiresAt));
    return invitations;
  }

  @override
  Future<CreatedHouseholdInvitation> createInvitation({
    required HouseholdMemberRole role,
  }) async {
    final session = _requireAdministrator();
    _throwInvitationFailure();
    final sequence = state._nextInvitation;
    state._nextInvitation += 1;
    final id = 'invitation-$sequence';
    final value = 'Invitation${sequence.toString().padLeft(6, '0')}'.padRight(
      48,
      'A',
    );
    final invitation = HouseholdInvitation(
      id: id,
      creatorId: session.member.id,
      creatorDisplayName: session.member.displayName,
      expiresAt: _now().toUtc().add(const Duration(hours: 24)),
      role: role,
      status: HouseholdInvitationStatus.active,
    );
    final payload = HouseholdInvitationPayload(
      serverAddress: session.serverAddress,
      value: value,
    );
    state._invitations.insert(0, invitation);
    state._invitationPayloads[id] = payload;
    return CreatedHouseholdInvitation(invitation: invitation, payload: payload);
  }

  @override
  Future<void> cancelInvitation(String invitationId) async {
    _requireAdministrator();
    _throwInvitationFailure();
    final index = state._invitations.indexWhere(
      (invitation) => invitation.id == invitationId,
    );
    if (index < 0 ||
        _withCurrentStatus(state._invitations[index]).status !=
            HouseholdInvitationStatus.active) {
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.unavailableInvitation,
      );
    }
    if (pauseInvitationCancellation) {
      await Completer<void>().future;
    }
    state._invitations[index] = state._invitations[index].copyWith(
      status: HouseholdInvitationStatus.canceled,
    );
  }

  @override
  Future<HouseholdSession> redeemInvitation({
    required HouseholdInvitationPayload invitation,
    required String displayName,
    required String email,
    required String password,
  }) {
    if (redemptionFailure case final failure?) {
      return Future.error(HouseholdGatewayException(failure));
    }
    final index = state._invitations.indexWhere((candidate) {
      final stored = state._invitationPayloads[candidate.id];
      return stored?.value == invitation.value &&
          stored?.serverAddress == invitation.serverAddress;
    });
    if (index < 0) {
      return Future.error(
        const HouseholdGatewayException(
          HouseholdGatewayFailure.invalidInvitation,
        ),
      );
    }
    final current = _withCurrentStatus(state._invitations[index]);
    if (current.status == HouseholdInvitationStatus.expired) {
      return Future.error(
        const HouseholdGatewayException(
          HouseholdGatewayFailure.expiredInvitation,
        ),
      );
    }
    if (current.status != HouseholdInvitationStatus.active) {
      return Future.error(
        const HouseholdGatewayException(
          HouseholdGatewayFailure.unavailableInvitation,
        ),
      );
    }
    if (state._accounts.any(
      (account) => account.email.toLowerCase() == email.trim().toLowerCase(),
    )) {
      return Future.error(
        const HouseholdGatewayException(HouseholdGatewayFailure.authentication),
      );
    }

    final member = HouseholdMember(
      id: 'member-${state._accounts.length + 1}',
      displayName: displayName.trim(),
      email: email.trim().toLowerCase(),
      role: current.role,
    );
    state._accounts.add(
      InMemoryHouseholdAccount(
        email: member.email,
        password: password,
        member: member,
      ),
    );
    state._invitations[index] = current.copyWith(
      status: HouseholdInvitationStatus.used,
    );
    final session = HouseholdSession(
      serverAddress: invitation.serverAddress,
      member: member,
    );
    state.storedSession = session;
    return Future.value(session);
  }

  @override
  Future<void> signOut() {
    if (signOutFailure case final failure?) {
      return Future.error(HouseholdGatewayException(failure));
    }
    state.storedSession = null;
    return Future.value();
  }

  HouseholdSession _requireAdministrator() {
    final session = state.storedSession;
    if (session == null ||
        session.member.role != HouseholdMemberRole.administrator) {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.forbidden);
    }
    return session;
  }

  void _throwInvitationFailure() {
    if (invitationFailure case final failure?) {
      throw HouseholdGatewayException(failure);
    }
  }

  HouseholdInvitation _withCurrentStatus(HouseholdInvitation invitation) {
    if (invitation.status == HouseholdInvitationStatus.active &&
        !invitation.expiresAt.isAfter(_now().toUtc())) {
      return invitation.copyWith(status: HouseholdInvitationStatus.expired);
    }
    return invitation;
  }
}
