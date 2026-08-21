import 'dart:async';
import 'dart:convert';

import 'package:balaur/household/data/household_credential_store.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_invitation.dart';
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
  HouseholdServerAddress? _serverAddress;

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
    _serverAddress = serverAddress;
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
    _serverAddress = serverAddress;
    return session;
  }

  @override
  Future<List<HouseholdInvitation>> listInvitations() async {
    final client = _authenticatedClient();
    final Map<String, dynamic> response;
    try {
      response = await client
          .send<Map<String, dynamic>>('/api/balaur/household-invitations')
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    } on ClientException catch (error) {
      throw HouseholdGatewayException(_invitationFailure(error));
    }

    try {
      final items = response['items'];
      if (items is! List<dynamic>) {
        throw const FormatException('Invalid Household Invitation list.');
      }
      return items
          .map((item) => _invitationFromJson(item as Map<String, dynamic>))
          .toList(growable: false);
    } on Object {
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.invalidSession,
      );
    }
  }

  @override
  Future<CreatedHouseholdInvitation> createInvitation({
    required HouseholdMemberRole role,
  }) async {
    final client = _authenticatedClient();
    final address = _serverAddress!;
    final Map<String, dynamic> response;
    try {
      response = await client
          .send<Map<String, dynamic>>(
            '/api/balaur/household-invitations',
            method: 'POST',
            body: {'role': _roleValue(role)},
          )
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    } on ClientException catch (error) {
      throw HouseholdGatewayException(_invitationFailure(error));
    }

    try {
      final invitation = response['invitation'];
      final value = response['value'];
      if (invitation is! Map<String, dynamic> || value is! String) {
        throw const FormatException('Invalid Household Invitation response.');
      }
      return CreatedHouseholdInvitation(
        invitation: _invitationFromJson(invitation),
        payload: HouseholdInvitationPayload(
          serverAddress: address,
          value: value,
        ),
      );
    } on Object {
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.invalidSession,
      );
    }
  }

  @override
  Future<void> cancelInvitation(String invitationId) async {
    final client = _authenticatedClient();
    try {
      await client
          .send<Map<String, dynamic>>(
            '/api/balaur/household-invitations/$invitationId/cancel',
            method: 'POST',
          )
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    } on ClientException catch (error) {
      throw HouseholdGatewayException(_invitationFailure(error));
    }
  }

  @override
  Future<HouseholdSession> redeemInvitation({
    required HouseholdInvitationPayload invitation,
    required String displayName,
    required String email,
    required String password,
  }) async {
    _requireAllowedAddress(invitation.serverAddress);
    final authStore = AuthStore();
    final client = PocketBase(
      invitation.serverAddress.value,
      authStore: authStore,
    );
    final Map<String, dynamic> response;
    try {
      response = await client
          .send<Map<String, dynamic>>(
            '/api/balaur/household-invitations/redeem',
            method: 'POST',
            body: {
              'invitation': invitation.value,
              'displayName': displayName.trim(),
              'email': email.trim(),
              'password': password,
              'passwordConfirm': password,
            },
          )
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    } on ClientException catch (error) {
      throw HouseholdGatewayException(_invitationFailure(error));
    }

    final RecordAuth authentication;
    final HouseholdSession session;
    try {
      authentication = RecordAuth.fromJson(response);
      authStore.save(authentication.token, authentication.record);
      if (!authStore.isValid) {
        throw const FormatException('Invalid Household authentication token.');
      }
      session = _sessionFromRecord(
        invitation.serverAddress,
        authentication.record,
      );
    } on Object {
      authStore.clear();
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.invalidSession,
      );
    }

    await _persist(invitation.serverAddress, authentication);
    _authStore = authStore;
    _serverAddress = invitation.serverAddress;
    return session;
  }

  @override
  Future<void> signOut() async {
    _authStore?.clear();
    _authStore = null;
    _serverAddress = null;
    await _clearStoredSession();
  }

  PocketBase _authenticatedClient() {
    final authStore = _authStore;
    final serverAddress = _serverAddress;
    if (authStore == null || !authStore.isValid || serverAddress == null) {
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.authentication,
      );
    }
    return PocketBase(serverAddress.value, authStore: authStore);
  }

  HouseholdInvitation _invitationFromJson(Map<String, dynamic> json) {
    final id = json['id'];
    final creator = json['creator'];
    final expiresAt = json['expiresAt'];
    if (id is! String ||
        id.isEmpty ||
        creator is! Map<String, dynamic> ||
        expiresAt is! String) {
      throw const FormatException('Invalid Household Invitation record.');
    }
    final creatorId = creator['id'];
    final creatorDisplayName = creator['displayName'];
    final expiration = DateTime.tryParse(expiresAt);
    if (creatorId is! String ||
        creatorId.isEmpty ||
        creatorDisplayName is! String ||
        creatorDisplayName.isEmpty ||
        expiration == null) {
      throw const FormatException('Invalid Household Invitation record.');
    }
    return HouseholdInvitation(
      id: id,
      creatorId: creatorId,
      creatorDisplayName: creatorDisplayName,
      expiresAt: expiration.toUtc(),
      role: _roleFromValue(json['role']),
      status: switch (json['state']) {
        'active' => HouseholdInvitationStatus.active,
        'expired' => HouseholdInvitationStatus.expired,
        'used' => HouseholdInvitationStatus.used,
        'canceled' => HouseholdInvitationStatus.canceled,
        _ => throw const FormatException('Invalid Household Invitation state.'),
      },
    );
  }

  HouseholdMemberRole _roleFromValue(Object? value) {
    return switch (value) {
      'member' => HouseholdMemberRole.member,
      'household_administrator' => HouseholdMemberRole.administrator,
      _ => throw const FormatException('Invalid Household Member role.'),
    };
  }

  String _roleValue(HouseholdMemberRole role) {
    return switch (role) {
      HouseholdMemberRole.member => 'member',
      HouseholdMemberRole.administrator => 'household_administrator',
    };
  }

  HouseholdGatewayFailure _invitationFailure(ClientException error) {
    return switch (error.statusCode) {
      400 => HouseholdGatewayFailure.invalidInput,
      401 => HouseholdGatewayFailure.authentication,
      403 => HouseholdGatewayFailure.forbidden,
      404 => HouseholdGatewayFailure.invalidInvitation,
      409 => HouseholdGatewayFailure.unavailableInvitation,
      410 => HouseholdGatewayFailure.expiredInvitation,
      _ => HouseholdGatewayFailure.connection,
    };
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
    final role = _roleFromValue(record.get<String>('role'));
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
