import 'dart:async';

import 'package:balaur/household/data/household_gateway.dart';
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
  InMemoryHouseholdGatewayState({this.storedSession});

  HouseholdSession? storedSession;
}

final class InMemoryHouseholdGateway implements HouseholdGateway {
  InMemoryHouseholdGateway({
    List<InMemoryHouseholdAccount> accounts = const [],
    InMemoryHouseholdGatewayState? state,
    this.restoreFailure,
    this.pairFailure,
    this.signOutFailure,
    this.pauseRestore = false,
    this.pausePairing = false,
  }) : _accounts = List.unmodifiable(accounts),
       state = state ?? InMemoryHouseholdGatewayState();

  final List<InMemoryHouseholdAccount> _accounts;
  final InMemoryHouseholdGatewayState state;
  final HouseholdGatewayFailure? restoreFailure;
  final HouseholdGatewayFailure? pairFailure;
  final HouseholdGatewayFailure? signOutFailure;
  final bool pauseRestore;
  final bool pausePairing;

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
    for (final candidate in _accounts) {
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
  Future<void> signOut() {
    if (signOutFailure case final failure?) {
      return Future.error(HouseholdGatewayException(failure));
    }
    state.storedSession = null;
    return Future.value();
  }
}
