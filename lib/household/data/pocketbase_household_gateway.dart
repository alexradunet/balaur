import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:balaur/household/data/household_credential_store.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/calendar_entry.dart';
import 'package:balaur/household/domain/household_archive.dart';
import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:http/http.dart' as http;
import 'package:pocketbase/pocketbase.dart';

final class PocketBaseHouseholdGateway implements HouseholdGateway {
  PocketBaseHouseholdGateway({
    required this._credentialStore,
    this.requestTimeout = const Duration(seconds: 10),
    this.allowInsecureLoopback = false,
  });

  final HouseholdCredentialStore _credentialStore;
  final Duration requestTimeout;
  final bool allowInsecureLoopback;
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
  Future<CalendarConnection> loadCalendarConnection() async {
    final response = await _sendCalendarRequest(
      '/api/balaur/calendar-connection',
    );
    return _calendarConnectionResponse(response);
  }

  @override
  Future<Uri> beginCalendarConnection({required bool replace}) async {
    final response = await _sendCalendarRequest(
      '/api/balaur/calendar-connection/authorize',
      method: 'POST',
      body: {'replace': replace},
    );
    try {
      final value = response['authorizationUrl'];
      if (value is! String) {
        throw const FormatException('Invalid Calendar authorization response.');
      }
      final uri = Uri.parse(value);
      final isSecure = uri.scheme == 'https' && uri.host.isNotEmpty;
      final isTestLoopback =
          allowInsecureLoopback &&
          uri.scheme == 'http' &&
          (uri.host == 'localhost' ||
              uri.host == '127.0.0.1' ||
              uri.host == '::1');
      if (!isSecure && !isTestLoopback) {
        throw const FormatException('Invalid Calendar authorization URL.');
      }
      return uri;
    } on Object {
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.invalidSession,
      );
    }
  }

  @override
  Future<CalendarConnection> selectCalendar(String calendarId) async {
    final response = await _sendCalendarRequest(
      '/api/balaur/calendar-connection/select',
      method: 'POST',
      body: {'calendarId': calendarId},
    );
    return _calendarConnectionResponse(response);
  }

  @override
  Future<CalendarConnection> disconnectCalendarConnection() async {
    final response = await _sendCalendarRequest(
      '/api/balaur/calendar-connection/disconnect',
      method: 'POST',
    );
    return _calendarConnectionResponse(response);
  }

  @override
  Future<List<CalendarEntry>> loadCalendarEntries({
    required DateTime rangeStart,
    required DateTime rangeEnd,
  }) async {
    final path = Uri(
      path: '/api/balaur/calendar-entries',
      queryParameters: {
        'timeMin': rangeStart.toUtc().toIso8601String(),
        'timeMax': rangeEnd.toUtc().toIso8601String(),
      },
    ).toString();
    final response = await _sendCalendarRequest(path);
    try {
      final items = response['items'];
      if (items is! List<dynamic>) {
        throw const FormatException('Invalid Calendar Entry list.');
      }
      return items
          .map((value) => _calendarEntryFromJson(value as Map<String, dynamic>))
          .toList(growable: false);
    } on Object {
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.invalidSession,
      );
    }
  }

  @override
  Future<HouseholdArchive> exportHouseholdArchive({
    required String password,
    required bool includeCalendarSnapshot,
  }) async {
    _authenticatedClient();
    final authStore = _authStore!;
    final serverAddress = _serverAddress!;
    final http.Response response;
    try {
      response = await http
          .post(
            Uri.parse('${serverAddress.value}/api/balaur/household-archive'),
            headers: <String, String>{
              'Authorization': authStore.token,
              'Content-Type': 'application/json',
            },
            body: jsonEncode(<String, Object>{
              'password': password,
              'includeCalendarSnapshot': includeCalendarSnapshot,
            }),
          )
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    } on Object {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    }
    if (response.statusCode >= 400) {
      throw HouseholdGatewayException(switch (response.statusCode) {
        401 => HouseholdGatewayFailure.authentication,
        403 => HouseholdGatewayFailure.forbidden,
        409 => HouseholdGatewayFailure.invalidInput,
        _ => HouseholdGatewayFailure.server,
      });
    }
    final disposition = response.headers['content-disposition'] ?? '';
    final match = RegExp(r'filename=\"([A-Za-z0-9._-]+)\"')
        .firstMatch(disposition);
    return HouseholdArchive(
      fileName: match?.group(1) ?? 'balaur-household.zip',
      bytes: Uint8List.fromList(response.bodyBytes),
    );
  }

  @override
  Future<void> signOut() async {
    _authStore?.clear();
    _authStore = null;
    _serverAddress = null;
    await _clearStoredSession();
  }

  Future<Map<String, dynamic>> _sendCalendarRequest(
    String path, {
    String method = 'GET',
    Map<String, dynamic>? body,
  }) async {
    final client = _authenticatedClient();
    try {
      return await client
          .send<Map<String, dynamic>>(
            path,
            method: method,
            body: body ?? const <String, dynamic>{},
          )
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const HouseholdGatewayException(HouseholdGatewayFailure.connection);
    } on ClientException catch (error) {
      throw HouseholdGatewayException(_calendarFailure(error));
    }
  }

  CalendarConnection _calendarConnectionResponse(
    Map<String, dynamic> response,
  ) {
    try {
      final value = response['connection'];
      if (value is! Map<String, dynamic>) {
        throw const FormatException('Invalid Calendar Connection response.');
      }
      return _calendarConnectionFromJson(value);
    } on HouseholdGatewayException {
      rethrow;
    } on Object {
      throw const HouseholdGatewayException(
        HouseholdGatewayFailure.invalidSession,
      );
    }
  }

  CalendarConnection _calendarConnectionFromJson(Map<String, dynamic> json) {
    final provider = json['provider'];
    final householdTimeZone = json['householdTimeZone'];
    final available = json['availableCalendars'];
    if (provider is! String ||
        provider != 'google' ||
        householdTimeZone is! String ||
        householdTimeZone.isEmpty ||
        available is! List<dynamic>) {
      throw const FormatException('Invalid Calendar Connection record.');
    }
    final selectedValue = json['selectedCalendar'];
    final CalendarSourceCalendar? selected = switch (selectedValue) {
      null => null,
      Map<String, dynamic>() => _calendarFromJson(selectedValue),
      _ => throw const FormatException(
        'Invalid selected Calendar Source calendar.',
      ),
    };
    return CalendarConnection(
      provider: provider,
      status: switch (json['status']) {
        'disconnected' => CalendarConnectionStatus.disconnected,
        'authorizing' => CalendarConnectionStatus.authorizing,
        'awaiting_selection' => CalendarConnectionStatus.awaitingSelection,
        'connected' => CalendarConnectionStatus.connected,
        'authorization_failed' => CalendarConnectionStatus.authorizationFailed,
        _ => throw const FormatException('Invalid Calendar Connection status.'),
      },
      householdTimeZone: householdTimeZone,
      selectedCalendar: selected,
      availableCalendars: available
          .map((value) => _calendarFromJson(value as Map<String, dynamic>))
          .toList(growable: false),
    );
  }

  CalendarSourceCalendar _calendarFromJson(Map<String, dynamic> json) {
    final id = json['id'];
    final name = json['name'];
    final color = json['colorHex'];
    if (id is! String ||
        id.isEmpty ||
        name is! String ||
        name.isEmpty ||
        color != null && color is! String) {
      throw const FormatException('Invalid Calendar Source calendar.');
    }
    return CalendarSourceCalendar(
      id: id,
      name: name,
      colorHex: color as String?,
    );
  }

  CalendarEntry _calendarEntryFromJson(Map<String, dynamic> json) {
    final id = json['id'];
    final title = json['title'];
    final startValue = json['start'];
    final endValue = json['end'];
    final allDay = json['allDay'];
    final location = json['location'];
    final colorHex = json['colorHex'];
    if (id is! String ||
        title is! String ||
        startValue is! String ||
        endValue is! String ||
        allDay is! bool ||
        location != null && location is! String ||
        colorHex is! String) {
      throw const FormatException('Invalid Calendar Entry record.');
    }
    final start = allDay
        ? _parseDateOnly(startValue)
        : DateTime.parse(startValue).toUtc();
    final end = allDay
        ? _parseDateOnly(endValue)
        : DateTime.parse(endValue).toUtc();
    return CalendarEntry(
      id: id,
      title: title,
      start: start,
      end: end,
      allDay: allDay,
      location: location as String?,
      colorHex: colorHex,
    );
  }

  DateTime _parseDateOnly(String value) {
    final match = RegExp(r'^(\\d{4})-(\\d{2})-(\\d{2})$').firstMatch(value);
    if (match == null) {
      throw const FormatException('Invalid Calendar Entry date.');
    }
    return DateTime.utc(
      int.parse(match.group(1)!),
      int.parse(match.group(2)!),
      int.parse(match.group(3)!),
    );
  }

  HouseholdGatewayFailure _calendarFailure(ClientException error) {
    return switch (error.statusCode) {
      400 || 409 => HouseholdGatewayFailure.invalidInput,
      401 => HouseholdGatewayFailure.authentication,
      403 => HouseholdGatewayFailure.forbidden,
      503 => HouseholdGatewayFailure.calendarAuthorization,
      424 => HouseholdGatewayFailure.calendarAuthorization,
      502 => HouseholdGatewayFailure.server,
      _ => HouseholdGatewayFailure.connection,
    };
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
      if (allowInsecureLoopback) {
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
    if (allowInsecureLoopback && address.uri.scheme == 'http' && isLoopback) {
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
