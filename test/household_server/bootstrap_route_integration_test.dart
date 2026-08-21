import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  final binaryPath = Platform.environment['POCKETBASE_BINARY'] ?? '';

  test(
    'a real PocketBase process permits one valid Household setup',
    () async {
      expect(File(binaryPath).existsSync(), isTrue);

      final projectRoot = _findProjectRoot();
      final dataDirectory = await Directory.systemTemp.createTemp(
        'balaur-household-server-',
      );

      try {
        await _verifyExpiredSetup(
          binaryPath: binaryPath,
          projectRoot: projectRoot,
          dataDirectory: dataDirectory,
        );
        await _verifyFirstSetup(
          binaryPath: binaryPath,
          projectRoot: projectRoot,
          dataDirectory: dataDirectory,
        );
        await _verifySecondSetupRejection(
          binaryPath: binaryPath,
          projectRoot: projectRoot,
          dataDirectory: dataDirectory,
        );
      } finally {
        await dataDirectory.delete(recursive: true);
      }
    },
    skip: binaryPath.isEmpty
        ? 'Run server/scripts/test.sh to supply PocketBase.'
        : false,
    timeout: const Timeout(Duration(minutes: 2)),
  );
}

Future<void> _verifyExpiredSetup({
  required String binaryPath,
  required Directory projectRoot,
  required Directory dataDirectory,
}) async {
  final secret = 'ExpiredSetupSecret'.padRight(64, 'A');
  final server = await _PocketBaseServer.start(
    binaryPath: binaryPath,
    projectRoot: projectRoot,
    dataDirectory: dataDirectory,
    setupSecret: secret,
    expiresAt: DateTime.now().subtract(const Duration(minutes: 1)),
  );

  try {
    expect(server.logs, isNot(contains('Household setup link:')));
    final response = await _request(
      'GET',
      server.uri('/api/balaur/setup?token=$secret'),
    );
    expect(response.statusCode, HttpStatus.gone);
  } finally {
    await server.stop();
  }
}

Future<void> _verifyFirstSetup({
  required String binaryPath,
  required Directory projectRoot,
  required Directory dataDirectory,
}) async {
  final secret = 'FirstSetupSecret'.padRight(64, 'B');
  final server = await _PocketBaseServer.start(
    binaryPath: binaryPath,
    projectRoot: projectRoot,
    dataDirectory: dataDirectory,
    setupSecret: secret,
    expiresAt: DateTime.now().add(const Duration(minutes: 10)),
  );

  try {
    final setupUri = server.uri('/api/balaur/setup?token=$secret');
    expect(server.logs, contains('Household setup link: $setupUri'));

    final setupPage = await _request('GET', setupUri);
    expect(setupPage.statusCode, HttpStatus.ok);
    expect(
      setupPage.body,
      contains('Create the first Household Administrator'),
    );
    expect(setupPage.header('cache-control'), 'no-store');

    final invalidToken = await _request(
      'GET',
      server.uri(
        '/api/balaur/setup?token=${'InvalidSetupToken'.padRight(64, 'C')}',
      ),
    );
    expect(invalidToken.statusCode, HttpStatus.notFound);

    final invalidInput = await _request(
      'POST',
      server.uri('/api/balaur/setup'),
      form: _setupForm(secret, passwordConfirm: 'different-password'),
    );
    expect(invalidInput.statusCode, HttpStatus.badRequest);

    final created = await _request(
      'POST',
      server.uri('/api/balaur/setup'),
      form: _setupForm(secret),
    );
    expect(created.statusCode, HttpStatus.created);
    expect(created.body, contains('Household setup is complete'));

    final authentication = await _request(
      'POST',
      server.uri('/api/collections/members/auth-with-password'),
      jsonBody: <String, Object>{
        'identity': 'alex@example.com',
        'password': 'correct-horse',
      },
    );
    expect(authentication.statusCode, HttpStatus.ok);
    final authenticationBody = authentication.jsonBody;
    final member = authenticationBody['record']! as Map<String, Object?>;
    expect(member['collectionName'], 'members');
    expect(member['collectionName'], isNot('_superusers'));
    expect(member['role'], 'household_administrator');
    expect(member['displayName'], 'Alex');

    final settingsResponse = await _request(
      'GET',
      server.uri('/api/collections/household_settings/records'),
      headers: <String, String>{
        HttpHeaders.authorizationHeader: authenticationBody['token']! as String,
      },
    );
    expect(settingsResponse.statusCode, HttpStatus.ok);
    final settingsItems = settingsResponse.jsonBody['items']! as List<Object?>;
    expect(settingsItems, hasLength(1));
    final settings = settingsItems.single! as Map<String, Object?>;
    expect(settings['householdName'], 'Test Household');
    expect(settings['timeZone'], 'Europe/Bucharest');
    expect(settings, isNot(contains('singletonKey')));
    expect(settings, isNot(contains('setupCompletedAt')));

    final replay = await _request(
      'POST',
      server.uri('/api/balaur/setup'),
      form: _setupForm(secret),
    );
    expect(replay.statusCode, HttpStatus.conflict);
  } finally {
    await server.stop();
  }
}

Future<void> _verifySecondSetupRejection({
  required String binaryPath,
  required Directory projectRoot,
  required Directory dataDirectory,
}) async {
  final secret = 'SecondSetupSecret'.padRight(64, 'D');
  final server = await _PocketBaseServer.start(
    binaryPath: binaryPath,
    projectRoot: projectRoot,
    dataDirectory: dataDirectory,
    setupSecret: secret,
    expiresAt: DateTime.now().add(const Duration(minutes: 10)),
  );

  try {
    expect(server.logs, isNot(contains('Household setup link:')));

    final secondSetup = await _request(
      'POST',
      server.uri('/api/balaur/setup'),
      form: _setupForm(secret, email: 'second@example.com'),
    );
    expect(secondSetup.statusCode, HttpStatus.conflict);

    final secondAuthentication = await _request(
      'POST',
      server.uri('/api/collections/members/auth-with-password'),
      jsonBody: <String, Object>{
        'identity': 'second@example.com',
        'password': 'correct-horse',
      },
    );
    expect(secondAuthentication.statusCode, HttpStatus.badRequest);
  } finally {
    await server.stop();
  }
}

Map<String, String> _setupForm(
  String secret, {
  String email = 'alex@example.com',
  String passwordConfirm = 'correct-horse',
}) {
  return <String, String>{
    'token': secret,
    'householdName': 'Test Household',
    'timeZone': 'Europe/Bucharest',
    'displayName': 'Alex',
    'email': email,
    'password': 'correct-horse',
    'passwordConfirm': passwordConfirm,
  };
}

Directory _findProjectRoot() {
  var directory = Directory.current.absolute;
  while (directory.parent.path != directory.path) {
    if (File('${directory.path}/pubspec.yaml').existsSync() &&
        Directory('${directory.path}/server/pb_hooks').existsSync()) {
      return directory;
    }
    directory = directory.parent;
  }
  throw StateError('The Balaur project root was not found.');
}

final class _PocketBaseServer {
  _PocketBaseServer({
    required this.process,
    required this.port,
    required this._logs,
    required this._stdoutSubscription,
    required this._stderrSubscription,
  });

  final Process process;
  final int port;
  final StringBuffer _logs;
  final StreamSubscription<String> _stdoutSubscription;
  final StreamSubscription<String> _stderrSubscription;

  String get logs => _logs.toString();

  Uri uri(String path) => Uri.parse('http://127.0.0.1:$port$path');

  static Future<_PocketBaseServer> start({
    required String binaryPath,
    required Directory projectRoot,
    required Directory dataDirectory,
    required String setupSecret,
    required DateTime expiresAt,
  }) async {
    final port = await _findFreePort();
    final hooksDirectory = '${projectRoot.path}/server/pb_hooks';
    final migrationsDirectory = '${projectRoot.path}/server/pb_migrations';
    final process = await Process.start(
      binaryPath,
      <String>[
        'serve',
        '--http=127.0.0.1:$port',
        '--dir=${dataDirectory.path}',
        '--hooksDir=$hooksDirectory',
        '--migrationsDir=$migrationsDirectory',
      ],
      workingDirectory: projectRoot.path,
      environment: <String, String>{
        'BALAUR_PUBLIC_URL': 'http://127.0.0.1:$port',
        'BALAUR_SETUP_SECRET': setupSecret,
        'BALAUR_SETUP_EXPIRES_AT':
            '${expiresAt.millisecondsSinceEpoch ~/ 1000}',
        'BALAUR_DISABLE_POCKETBASE_INSTALLER': '1',
        'PATH': '/nonexistent',
      },
      includeParentEnvironment: true,
    );

    final logs = StringBuffer();
    final stdoutSubscription = process.stdout
        .transform(utf8.decoder)
        .listen(logs.write);
    final stderrSubscription = process.stderr
        .transform(utf8.decoder)
        .listen(logs.write);
    final server = _PocketBaseServer(
      process: process,
      port: port,
      logs: logs,
      stdoutSubscription: stdoutSubscription,
      stderrSubscription: stderrSubscription,
    );

    await server._waitUntilHealthy();
    await Future<void>.delayed(const Duration(milliseconds: 200));
    if (server.logs.contains('/_/#/pbinstall/')) {
      await server.stop();
      throw StateError(
        'PocketBase opened its superuser installer during a route test.',
      );
    }
    return server;
  }

  Future<void> _waitUntilHealthy() async {
    for (var attempt = 0; attempt < 100; attempt += 1) {
      try {
        final response = await _request('GET', uri('/api/health'));
        if (response.statusCode == HttpStatus.ok) {
          return;
        }
      } on SocketException {
        // The listener is not ready.
      }
      await Future<void>.delayed(const Duration(milliseconds: 100));
    }
    await stop();
    throw StateError('PocketBase did not become healthy.\n$logs');
  }

  Future<void> stop() async {
    process.kill(ProcessSignal.sigterm);
    try {
      await process.exitCode.timeout(const Duration(seconds: 5));
    } on TimeoutException {
      process.kill(ProcessSignal.sigkill);
      await process.exitCode;
    }
    await _stdoutSubscription.cancel();
    await _stderrSubscription.cancel();
  }
}

Future<int> _findFreePort() async {
  final socket = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
  final port = socket.port;
  await socket.close();
  return port;
}

Future<_HttpResponse> _request(
  String method,
  Uri uri, {
  Map<String, String> headers = const <String, String>{},
  Map<String, Object>? jsonBody,
  Map<String, String>? form,
}) async {
  final client = HttpClient();
  try {
    final request = await client.openUrl(method, uri);
    headers.forEach(request.headers.set);
    if (jsonBody != null) {
      request.headers.contentType = ContentType.json;
      request.write(jsonEncode(jsonBody));
    } else if (form != null) {
      request.headers.contentType = ContentType(
        'application',
        'x-www-form-urlencoded',
        charset: 'utf-8',
      );
      request.write(Uri(queryParameters: form).query);
    }

    final response = await request.close();
    final responseHeaders = <String, String>{};
    response.headers.forEach((name, values) {
      responseHeaders[name.toLowerCase()] = values.join(', ');
    });
    final body = await utf8.decoder.bind(response).join();
    return _HttpResponse(
      statusCode: response.statusCode,
      headers: responseHeaders,
      body: body,
    );
  } finally {
    client.close(force: true);
  }
}

final class _HttpResponse {
  const _HttpResponse({
    required this.statusCode,
    required this.headers,
    required this.body,
  });

  final int statusCode;
  final Map<String, String> headers;
  final String body;

  String? header(String name) => headers[name.toLowerCase()];

  Map<String, Object?> get jsonBody {
    return jsonDecode(body) as Map<String, Object?>;
  }
}
