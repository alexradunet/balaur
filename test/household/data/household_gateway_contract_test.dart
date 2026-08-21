import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:balaur/household/data/household_credential_store.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/data/pocketbase_household_gateway.dart';
import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  _householdGatewayContract('In-memory Household Gateway', () async {
    final state = InMemoryHouseholdGatewayState();
    HouseholdGateway createGateway() =>
        InMemoryHouseholdGateway(state: state, accounts: const [_account]);

    return _GatewayHarness(
      gateway: createGateway(),
      restart: () async => createGateway(),
      serverAddress: HouseholdServerAddress.parse(
        'https://household.example.com',
      ),
      close: () async {},
    );
  });

  _householdGatewayContract('PocketBase Household Gateway', () async {
    final server = await _FakePocketBaseServer.start();
    final credentialStore = InMemoryHouseholdCredentialStore();
    HouseholdGateway createGateway() => PocketBaseHouseholdGateway(
      credentialStore: credentialStore,
      requestTimeout: const Duration(seconds: 2),
      allowInsecureLoopbackForTesting: true,
    );

    return _GatewayHarness(
      gateway: createGateway(),
      restart: () async => createGateway(),
      serverAddress: HouseholdServerAddress.loopbackForTesting(server.baseUrl),
      close: server.close,
    );
  });

  test('does not restore a session over insecure HTTP in production', () async {
    final server = await _FakePocketBaseServer.start();
    addTearDown(server.close);
    final credentialStore = InMemoryHouseholdCredentialStore();
    final address = HouseholdServerAddress.loopbackForTesting(server.baseUrl);
    final testGateway = PocketBaseHouseholdGateway(
      credentialStore: credentialStore,
      allowInsecureLoopbackForTesting: true,
    );
    await testGateway.pair(
      serverAddress: address,
      email: _account.email,
      password: _account.password,
    );
    final requestCount = server.requestCount;
    final productionGateway = PocketBaseHouseholdGateway(
      credentialStore: credentialStore,
    );

    expect(await productionGateway.restoreSession(), isNull);
    expect(server.requestCount, requestCount);
    expect(await credentialStore.load(), isNull);
  });

  test('clears malformed stored authentication data', () async {
    final credentialStore = InMemoryHouseholdCredentialStore();
    await credentialStore.save(
      StoredHouseholdCredentials(
        serverAddress: 'https://household.example.com',
        authentication: jsonEncode({
          'token': 'not-a-token',
          'record': _memberRecord(),
        }),
      ),
    );
    final gateway = PocketBaseHouseholdGateway(
      credentialStore: credentialStore,
    );

    expect(await gateway.restoreSession(), isNull);
    expect(await credentialStore.load(), isNull);
  });

  test('rejects an invalid token returned during pairing', () async {
    final server = await _FakePocketBaseServer.start(invalidToken: true);
    addTearDown(server.close);
    final credentialStore = InMemoryHouseholdCredentialStore();
    final gateway = PocketBaseHouseholdGateway(
      credentialStore: credentialStore,
      allowInsecureLoopbackForTesting: true,
    );

    await expectLater(
      gateway.pair(
        serverAddress: HouseholdServerAddress.loopbackForTesting(
          server.baseUrl,
        ),
        email: _account.email,
        password: _account.password,
      ),
      throwsA(_hasFailure(HouseholdGatewayFailure.invalidSession)),
    );
    expect(await credentialStore.load(), isNull);
  });

  test('maps PocketBase invitation validation failures', () async {
    final server = await _FakePocketBaseServer.start();
    addTearDown(server.close);
    final gateway = PocketBaseHouseholdGateway(
      credentialStore: InMemoryHouseholdCredentialStore(),
      allowInsecureLoopbackForTesting: true,
    );
    await gateway.pair(
      serverAddress: HouseholdServerAddress.loopbackForTesting(server.baseUrl),
      email: _account.email,
      password: _account.password,
    );
    final created = await gateway.createInvitation(
      role: HouseholdMemberRole.member,
    );
    await gateway.signOut();

    await expectLater(
      gateway.redeemInvitation(
        invitation: created.payload,
        displayName: 'Existing',
        email: 'existing@example.com',
        password: 'new-correct-horse',
      ),
      throwsA(_hasFailure(HouseholdGatewayFailure.invalidInput)),
    );
  });

  test('maps a PocketBase connection failure', () async {
    final server = await _FakePocketBaseServer.start();
    final address = HouseholdServerAddress.loopbackForTesting(server.baseUrl);
    await server.close();
    final gateway = PocketBaseHouseholdGateway(
      credentialStore: InMemoryHouseholdCredentialStore(),
      requestTimeout: const Duration(milliseconds: 250),
      allowInsecureLoopbackForTesting: true,
    );

    await expectLater(
      gateway.pair(
        serverAddress: address,
        email: _account.email,
        password: _account.password,
      ),
      throwsA(_hasFailure(HouseholdGatewayFailure.connection)),
    );
  });

  test('binds an in-memory invitation to its Household Server', () async {
    final state = InMemoryHouseholdGatewayState();
    final gateway = InMemoryHouseholdGateway(
      state: state,
      accounts: const [_account],
    );
    await gateway.pair(
      serverAddress: HouseholdServerAddress.parse(
        'https://household.example.com',
      ),
      email: _account.email,
      password: _account.password,
    );
    final created = await gateway.createInvitation(
      role: HouseholdMemberRole.member,
    );
    await gateway.signOut();
    final wrongServerPayload = HouseholdInvitationPayload(
      serverAddress: HouseholdServerAddress.parse(
        'https://other-household.example.com',
      ),
      value: created.payload.value,
    );

    await expectLater(
      gateway.redeemInvitation(
        invitation: wrongServerPayload,
        displayName: 'Mara',
        email: 'mara@example.com',
        password: 'new-correct-horse',
      ),
      throwsA(_hasFailure(HouseholdGatewayFailure.invalidInvitation)),
    );
  });
}

Matcher _hasFailure(HouseholdGatewayFailure failure) {
  return isA<HouseholdGatewayException>().having(
    (error) => error.failure,
    'failure',
    failure,
  );
}

void _householdGatewayContract(
  String name,
  Future<_GatewayHarness> Function() createHarness,
) {
  group(name, () {
    late _GatewayHarness harness;

    setUp(() async {
      harness = await createHarness();
    });

    tearDown(() async {
      await harness.close();
    });

    test('pairs an authenticated Household Member', () async {
      final session = await harness.gateway.pair(
        serverAddress: harness.serverAddress,
        email: _account.email,
        password: _account.password,
      );

      expect(session.serverAddress, harness.serverAddress);
      expect(session.member, _account.member);
    });

    test('restores a valid session after restart', () async {
      await harness.gateway.pair(
        serverAddress: harness.serverAddress,
        email: _account.email,
        password: _account.password,
      );
      final restartedGateway = await harness.restart();

      final restored = await restartedGateway.restoreSession();

      expect(restored?.serverAddress, harness.serverAddress);
      expect(restored?.member, _account.member);
    });

    test('clears the session on sign-out', () async {
      await harness.gateway.pair(
        serverAddress: harness.serverAddress,
        email: _account.email,
        password: _account.password,
      );

      await harness.gateway.signOut();
      final restartedGateway = await harness.restart();

      expect(await restartedGateway.restoreSession(), isNull);
    });

    test('reports invalid credentials as authentication failure', () async {
      await expectLater(
        harness.gateway.pair(
          serverAddress: harness.serverAddress,
          email: _account.email,
          password: 'wrong-password',
        ),
        throwsA(
          isA<HouseholdGatewayException>().having(
            (error) => error.failure,
            'failure',
            HouseholdGatewayFailure.authentication,
          ),
        ),
      );
    });

    test('creates, lists, assigns, and cancels an invitation', () async {
      await harness.gateway.pair(
        serverAddress: harness.serverAddress,
        email: _account.email,
        password: _account.password,
      );

      final created = await harness.gateway.createInvitation(
        role: HouseholdMemberRole.administrator,
      );

      expect(created.invitation.role, HouseholdMemberRole.administrator);
      expect(created.invitation.status, HouseholdInvitationStatus.active);
      expect(created.payload.serverAddress, harness.serverAddress);
      expect(created.payload.value, hasLength(48));
      expect(await harness.gateway.listInvitations(), [created.invitation]);

      await harness.gateway.cancelInvitation(created.invitation.id);

      final canceled = await harness.gateway.listInvitations();
      expect(canceled.single.status, HouseholdInvitationStatus.canceled);
    });

    test('redeems an invitation once and stores the new session', () async {
      await harness.gateway.pair(
        serverAddress: harness.serverAddress,
        email: _account.email,
        password: _account.password,
      );
      final created = await harness.gateway.createInvitation(
        role: HouseholdMemberRole.member,
      );
      await harness.gateway.signOut();

      final joined = await harness.gateway.redeemInvitation(
        invitation: created.payload,
        displayName: 'Mara',
        email: 'mara@example.com',
        password: 'new-correct-horse',
      );

      expect(joined.member.displayName, 'Mara');
      expect(joined.member.role, HouseholdMemberRole.member);
      final restartedGateway = await harness.restart();
      expect(await restartedGateway.restoreSession(), joined);

      await expectLater(
        restartedGateway.redeemInvitation(
          invitation: created.payload,
          displayName: 'Second',
          email: 'second@example.com',
          password: 'new-correct-horse',
        ),
        throwsA(_hasFailure(HouseholdGatewayFailure.unavailableInvitation)),
      );
    });

    test('prevents a Household Member from managing invitations', () async {
      await harness.gateway.pair(
        serverAddress: harness.serverAddress,
        email: _account.email,
        password: _account.password,
      );
      final created = await harness.gateway.createInvitation(
        role: HouseholdMemberRole.member,
      );
      await harness.gateway.signOut();
      await harness.gateway.redeemInvitation(
        invitation: created.payload,
        displayName: 'Mara',
        email: 'mara@example.com',
        password: 'new-correct-horse',
      );

      await expectLater(
        harness.gateway.listInvitations(),
        throwsA(_hasFailure(HouseholdGatewayFailure.forbidden)),
      );
    });
  });
}

final class _GatewayHarness {
  const _GatewayHarness({
    required this.gateway,
    required this.restart,
    required this.serverAddress,
    required this.close,
  });

  final HouseholdGateway gateway;
  final Future<HouseholdGateway> Function() restart;
  final HouseholdServerAddress serverAddress;
  final Future<void> Function() close;
}

const _member = HouseholdMember(
  id: 'member-example',
  displayName: 'Alex',
  email: 'alex@example.com',
  role: HouseholdMemberRole.administrator,
);

const _account = InMemoryHouseholdAccount(
  email: 'alex@example.com',
  password: 'correct-horse',
  member: _member,
);

Map<String, Object?> _memberRecord({
  String id = 'member-example',
  String displayName = 'Alex',
  String email = 'alex@example.com',
  String role = 'household_administrator',
}) {
  return {
    'id': id,
    'collectionId': 'pbc_members',
    'collectionName': 'members',
    'created': '2026-08-21 00:00:00.000Z',
    'updated': '2026-08-21 00:00:00.000Z',
    'email': email,
    'emailVisibility': false,
    'verified': true,
    'displayName': displayName,
    'role': role,
  };
}

final class _FakePocketBaseServer {
  _FakePocketBaseServer._(this._server, {required this.invalidToken});

  final HttpServer _server;
  final bool invalidToken;
  late final StreamSubscription<HttpRequest> _subscription;
  final Map<String, _FakeInvitation> _invitations = {};
  int _nextInvitation = 1;
  int requestCount = 0;

  String get baseUrl => 'http://127.0.0.1:${_server.port}';

  static Future<_FakePocketBaseServer> start({
    bool invalidToken = false,
  }) async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final fake = _FakePocketBaseServer._(server, invalidToken: invalidToken);
    fake._subscription = server.listen((request) {
      unawaited(fake._handle(request));
    });
    return fake;
  }

  Future<void> close() async {
    await _subscription.cancel();
    await _server.close(force: true);
  }

  Future<void> _handle(HttpRequest request) async {
    requestCount += 1;
    try {
      if (request.method == 'POST' &&
          request.uri.path == '/api/collections/members/auth-with-password') {
        await _authenticate(request);
        return;
      }
      if (request.method == 'POST' &&
          request.uri.path == '/api/collections/members/auth-refresh') {
        await _refresh(request);
        return;
      }
      if (request.method == 'POST' &&
          request.uri.path == '/api/balaur/household-invitations/redeem') {
        await _redeemInvitation(request);
        return;
      }
      if (request.uri.path == '/api/balaur/household-invitations') {
        if (request.method == 'GET') {
          await _listInvitations(request);
          return;
        }
        if (request.method == 'POST') {
          await _createInvitation(request);
          return;
        }
      }
      final cancelMatch = RegExp(
        r'^/api/balaur/household-invitations/([^/]+)/cancel$',
      ).firstMatch(request.uri.path);
      if (request.method == 'POST' && cancelMatch != null) {
        await _cancelInvitation(request, cancelMatch.group(1)!);
        return;
      }
      await _json(request.response, HttpStatus.notFound, {
        'status': HttpStatus.notFound,
        'message': 'Not found.',
        'data': <String, Object?>{},
      });
    } on Object {
      try {
        await _json(request.response, HttpStatus.internalServerError, {
          'status': HttpStatus.internalServerError,
          'message': 'Test server error.',
          'data': <String, Object?>{},
        });
      } on Object {
        // The response is already closed.
      }
    }
  }

  Future<void> _authenticate(HttpRequest request) async {
    final body = jsonDecode(await utf8.decoder.bind(request).join());
    final valid =
        body is Map<String, dynamic> &&
        body['identity'] == _account.email &&
        body['password'] == _account.password;
    if (!valid) {
      await _json(request.response, HttpStatus.badRequest, {
        'status': HttpStatus.badRequest,
        'message': 'Failed to authenticate.',
        'data': <String, Object?>{},
      });
      return;
    }
    await _authResponse(request.response);
  }

  Future<void> _refresh(HttpRequest request) async {
    final record = _authorizationRecord(request);
    if (record == null) {
      await _error(
        request.response,
        HttpStatus.unauthorized,
        'The session is not valid.',
      );
      return;
    }
    await _authResponse(request.response, record: record);
  }

  Future<void> _authResponse(
    HttpResponse response, {
    Map<String, Object?>? record,
  }) async {
    final memberRecord = record ?? _memberRecord();
    await _json(response, HttpStatus.ok, {
      'token': invalidToken ? 'not-a-token' : _token(memberRecord),
      'record': memberRecord,
    });
  }

  String _token(Map<String, Object?> record) {
    String encode(Map<String, Object?> value) {
      return base64Url
          .encode(utf8.encode(jsonEncode(value)))
          .replaceAll('=', '');
    }

    final expiresAt = DateTime.now().add(const Duration(hours: 1));
    return '${encode({'alg': 'HS256', 'typ': 'JWT'})}.'
        '${encode({'collectionId': 'pbc_members', 'exp': expiresAt.millisecondsSinceEpoch ~/ 1000, 'id': record['id'], 'type': 'auth', 'displayName': record['displayName'], 'email': record['email'], 'role': record['role']})}.test-signature';
  }

  Map<String, Object?>? _authorizationRecord(HttpRequest request) {
    final token = request.headers.value(HttpHeaders.authorizationHeader);
    if (token == null || token.isEmpty) {
      return null;
    }
    try {
      final segments = token.split('.');
      final normalized = base64Url.normalize(segments[1]);
      final claims = jsonDecode(utf8.decode(base64Url.decode(normalized)));
      if (claims is! Map<String, dynamic> ||
          claims['collectionId'] != 'pbc_members') {
        return null;
      }
      final id = claims['id'];
      final displayName = claims['displayName'];
      final email = claims['email'];
      final role = claims['role'];
      if (id is! String ||
          displayName is! String ||
          email is! String ||
          role is! String) {
        return null;
      }
      return _memberRecord(
        id: id,
        displayName: displayName,
        email: email,
        role: role,
      );
    } on Object {
      return null;
    }
  }

  Future<void> _createInvitation(HttpRequest request) async {
    final creator = _authorizationRecord(request);
    if (creator?['role'] != 'household_administrator') {
      await _error(
        request.response,
        HttpStatus.forbidden,
        'Household Administrator access is required.',
      );
      return;
    }
    final body = await _jsonRequestBody(request);
    final role = body['role'];
    if (role != 'member' && role != 'household_administrator') {
      await _error(
        request.response,
        HttpStatus.badRequest,
        'Select a valid role.',
      );
      return;
    }
    final sequence = _nextInvitation;
    _nextInvitation += 1;
    final invitation = _FakeInvitation(
      id: 'invitation-$sequence',
      value: 'Invitation${sequence.toString().padLeft(6, '0')}'.padRight(
        48,
        'A',
      ),
      creatorId: creator!['id']! as String,
      creatorDisplayName: creator['displayName']! as String,
      expiresAt: DateTime.now().toUtc().add(const Duration(hours: 24)),
      role: role as String,
    );
    _invitations[invitation.id] = invitation;
    await _json(request.response, HttpStatus.created, {
      'invitation': invitation.publicJson,
      'value': invitation.value,
    });
  }

  Future<void> _listInvitations(HttpRequest request) async {
    final member = _authorizationRecord(request);
    if (member?['role'] != 'household_administrator') {
      await _error(
        request.response,
        HttpStatus.forbidden,
        'Household Administrator access is required.',
      );
      return;
    }
    await _json(request.response, HttpStatus.ok, {
      'items': _invitations.values
          .map((invitation) => invitation.publicJson)
          .toList(growable: false),
    });
  }

  Future<void> _cancelInvitation(
    HttpRequest request,
    String invitationId,
  ) async {
    final member = _authorizationRecord(request);
    if (member?['role'] != 'household_administrator') {
      await _error(
        request.response,
        HttpStatus.forbidden,
        'Household Administrator access is required.',
      );
      return;
    }
    final invitation = _invitations[invitationId];
    if (invitation == null || invitation.state != 'active') {
      await _error(
        request.response,
        HttpStatus.conflict,
        'The Household Invitation is not active.',
      );
      return;
    }
    invitation.state = 'canceled';
    await _json(request.response, HttpStatus.ok, {
      'invitation': invitation.publicJson,
    });
  }

  Future<void> _redeemInvitation(HttpRequest request) async {
    final body = await _jsonRequestBody(request);
    if (body['email'] == 'existing@example.com') {
      await _error(
        request.response,
        HttpStatus.badRequest,
        'The email address is already in use.',
      );
      return;
    }
    _FakeInvitation? invitation;
    for (final candidate in _invitations.values) {
      if (candidate.value == body['invitation']) {
        invitation = candidate;
        break;
      }
    }
    if (invitation == null) {
      await _error(
        request.response,
        HttpStatus.notFound,
        'The Household Invitation is invalid.',
      );
      return;
    }
    if (invitation.state != 'active') {
      await _error(
        request.response,
        HttpStatus.conflict,
        'The Household Invitation is not active.',
      );
      return;
    }
    invitation.state = 'used';
    final record = _memberRecord(
      id: 'member-invited',
      displayName: body['displayName']! as String,
      email: body['email']! as String,
      role: invitation.role,
    );
    await _authResponse(request.response, record: record);
  }

  Future<Map<String, dynamic>> _jsonRequestBody(HttpRequest request) async {
    final decoded = jsonDecode(await utf8.decoder.bind(request).join());
    return decoded as Map<String, dynamic>;
  }

  Future<void> _error(HttpResponse response, int statusCode, String message) {
    return _json(response, statusCode, {
      'status': statusCode,
      'message': message,
      'data': <String, Object?>{},
    });
  }

  Future<void> _json(
    HttpResponse response,
    int statusCode,
    Map<String, Object?> body,
  ) async {
    response
      ..statusCode = statusCode
      ..headers.contentType = ContentType.json
      ..write(jsonEncode(body));
    await response.close();
  }
}

final class _FakeInvitation {
  _FakeInvitation({
    required this.id,
    required this.value,
    required this.creatorId,
    required this.creatorDisplayName,
    required this.expiresAt,
    required this.role,
  });

  final String id;
  final String value;
  final String creatorId;
  final String creatorDisplayName;
  final DateTime expiresAt;
  final String role;
  String state = 'active';

  Map<String, Object?> get publicJson => {
    'id': id,
    'creator': {'id': creatorId, 'displayName': creatorDisplayName},
    'expiresAt': expiresAt.toIso8601String(),
    'role': role,
    'state': state,
  };
}
