import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:balaur/household/data/household_credential_store.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/data/pocketbase_household_gateway.dart';
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

Map<String, Object?> _memberRecord() {
  return {
    'id': _member.id,
    'collectionId': 'pbc_members',
    'collectionName': 'members',
    'created': '2026-08-21 00:00:00.000Z',
    'updated': '2026-08-21 00:00:00.000Z',
    'email': _member.email,
    'emailVisibility': false,
    'verified': true,
    'displayName': _member.displayName,
    'role': 'household_administrator',
  };
}

final class _FakePocketBaseServer {
  _FakePocketBaseServer._(this._server, {required this.invalidToken});

  final HttpServer _server;
  final bool invalidToken;
  late final StreamSubscription<HttpRequest> _subscription;
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
    final authorization = request.headers.value(
      HttpHeaders.authorizationHeader,
    );
    if (authorization == null || authorization.isEmpty) {
      await _json(request.response, HttpStatus.unauthorized, {
        'status': HttpStatus.unauthorized,
        'message': 'The session is not valid.',
        'data': <String, Object?>{},
      });
      return;
    }
    await _authResponse(request.response);
  }

  Future<void> _authResponse(HttpResponse response) async {
    await _json(response, HttpStatus.ok, {
      'token': invalidToken ? 'not-a-token' : _token(),
      'record': _memberRecord(),
    });
  }

  String _token() {
    String encode(Map<String, Object> value) {
      return base64Url
          .encode(utf8.encode(jsonEncode(value)))
          .replaceAll('=', '');
    }

    final expiresAt = DateTime.now().add(const Duration(hours: 1));
    return '${encode({'alg': 'HS256', 'typ': 'JWT'})}.'
        '${encode({'collectionId': 'pbc_members', 'exp': expiresAt.millisecondsSinceEpoch ~/ 1000, 'id': _member.id, 'type': 'auth'})}.test-signature';
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
