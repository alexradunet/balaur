import 'dart:async';
import 'dart:convert';

import 'package:balaur/household/data/household_credential_store.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:pocketbase/pocketbase.dart';

final class PocketBaseHouseholdGateway implements HouseholdGateway {
  PocketBaseHouseholdGateway({
    required this._credentialStore,
    this.requestTimeout = const Duration(seconds: 10),
    this.allowInsecureLoopbackForTesting = false,
  });

  final HouseholdCredentialStore _credentialStore;
  final Duration requestTimeout;
  final bool allowInsecureLoopbackForTesting;
  AuthStore? _authStore;

  @override
  Future<HouseholdSession?> restoreSession() async {
    final StoredHouseholdCredentials? credentials;
    try {
      credentials = await _credentialStore.load();
    } on Object {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.storage);
    }
    if (credentials == null) {
      return null;
    }

    final HouseholdServerAddress serverAddress;
    final AuthStore authStore;
    try {
      serverAddress = _restoreAddress(credentials.serverAddress);
      authStore = _restoreAuthStore(credentials.authentication);
      if (!authStore.isValid) {
        throw const FormatException('Invalid Household authentication token.');
      }
    } on Object {
      await _clearStoredSession();
      return null;
    }

    final client = PocketBase(serverAddress.value, authStore: authStore);
    final RecordAuth authentication;
    try {
      authentication = await client
          .collection('members')
          .authRefresh()
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    } on ClientException catch (error) {
      if (_isAuthenticationFailure(error)) {
        await _clearStoredSession();
        return null;
      }
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    }

    final HouseholdSession session;
    try {
      if (!authStore.isValid) {
        throw const FormatException('Invalid Household authentication token.');
      }
      session = _sessionFromRecord(serverAddress, authentication.record);
    } on Object {
      await _clearStoredSession();
      return null;
    }
    await _persist(serverAddress, authentication);
    _authStore = authStore;
    return session;
  }

  @override
  Future<HouseholdSession> pair({
    required HouseholdServerAddress serverAddress,
    required String email,
    required String password,
  }) async {
    _requireAllowedAddress(serverAddress);
    final authStore = AuthStore();
    final client = PocketBase(serverAddress.value, authStore: authStore);
    final RecordAuth authentication;
    try {
      authentication = await client
          .collection('members')
          .authWithPassword(email.trim(), password)
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    } on ClientException catch (error) {
      throw HouseholdGatewayException(
        _isAuthenticationFailure(error)
            ? HouseholdGatewayFailure.authentication
            : HouseholdGatewayFailure.connection,
      );
    }

    final HouseholdSession session;
    try {
      if (!authStore.isValid) {
        throw const FormatException('Invalid Household authentication token.');
      }
      session = _sessionFromRecord(serverAddress, authentication.record);
    } on Object {
      authStore.clear();
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.invalidSession,
      );
    }

    await _persist(serverAddress, authentication);
    _authStore = authStore;
    return session;
  }

  @override
  Future<void> signOut() async {
    _authStore?.clear();
    await _clearStoredSession();
  }

  Future<void> _persist(
    HouseholdServerAddress serverAddress,
    RecordAuth authentication,
  ) async {
    final encodedAuthentication = jsonEncode({
      'token': authentication.token,
      'record': authentication.record.toJson(),
    });
    try {
      await _credentialStore.save(
        StoredHouseholdCredentials(
          serverAddress: serverAddress.value,
          authentication: encodedAuthentication,
        ),
      );
    } on Object {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.storage);
    }
  }

  Future<void> _clearStoredSession() async {
    try {
      await _credentialStore.clear();
    } on Object {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.storage);
    }
  }

  HouseholdServerAddress _restoreAddress(String value) {
    try {
      return HouseholdServerAddress.parse(value);
    } on FormatException {
      if (allowInsecureLoopbackForTesting) {
        return HouseholdServerAddress.loopbackForTesting(value);
      }
      rethrow;
    }
  }

  void _requireAllowedAddress(HouseholdServerAddress address) {
    if (address.uri.scheme == 'https') {
      return;
    }
    final host = address.uri.host.toLowerCase();
    final isLoopback =
        host == 'localhost' || host == '127.0.0.1' || host == '::1';
    if (allowInsecureLoopbackForTesting &&
        address.uri.scheme == 'http' &&
        isLoopback) {
      return;
    }
    throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
  }

  AuthStore _restoreAuthStore(String authentication) {
    final decoded = jsonDecode(authentication);
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Invalid Household authentication data.');
    }
    final token = decoded['token'];
    final record = decoded['record'];
    if (token is! String || record is! Map<String, dynamic>) {
      throw const FormatException('Invalid Household authentication data.');
    }
    return AuthStore()..save(token, RecordModel.fromJson(record));
  }

  HouseholdSession _sessionFromRecord(
    HouseholdServerAddress serverAddress,
    RecordModel record,
  ) {
    if (record.collectionName != 'members' || record.id.isEmpty) {
      throw const FormatException('Invalid Household Member record.');
    }
    final role = switch (record.get<String>('role')) {
      'member' => HouseholdMemberRole.member,
      'household_administrator' => HouseholdMemberRole.administrator,
      _ => throw const FormatException('Invalid Household Member role.'),
    };
    final email = record.get<String>('email');
    final displayName = record.get<String>('displayName');
    if (email.isEmpty || displayName.isEmpty) {
      throw const FormatException('Incomplete Household Member record.');
    }
    return HouseholdSession(
      serverAddress: serverAddress,
      member: HouseholdMember(
        id: record.id,
        displayName: displayName,
        email: email,
        role: role,
      ),
    );
  }

  bool _isAuthenticationFailure(ClientException error) {
    return error.statusCode == 400 ||
        error.statusCode == 401 ||
        error.statusCode == 403;
  }
}
