import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:archive/archive.dart';
import 'package:flutter_test/flutter_test.dart';

const _testSuperuserEmail = 'route-test@example.com';
const _testSuperuserPassword = 'route-test-correct-horse';

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
        await _createTestSuperuser(
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

Future<void> _createTestSuperuser({
  required String binaryPath,
  required Directory projectRoot,
  required Directory dataDirectory,
}) async {
  final result = await Process.run(
    binaryPath,
    <String>[
      'superuser',
      'create',
      _testSuperuserEmail,
      _testSuperuserPassword,
      '--dir=${dataDirectory.path}',
      '--hooksDir=${projectRoot.path}/server/pb_hooks',
      '--migrationsDir=${projectRoot.path}/server/pb_migrations',
    ],
    workingDirectory: projectRoot.path,
    environment: <String, String>{
      'BALAUR_DISABLE_POCKETBASE_INSTALLER': '1',
      'PATH': '/nonexistent',
    },
    includeParentEnvironment: true,
  );
  if (result.exitCode != 0) {
    throw StateError(
      'PocketBase could not create the route-test superuser.\n'
      '${result.stdout}\n${result.stderr}',
    );
  }
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

    await _verifyInvitationRoutes(
      server: server,
      administratorToken: authenticationBody['token']! as String,
    );

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

Future<void> _verifyInvitationRoutes({
  required _PocketBaseServer server,
  required String administratorToken,
}) async {
  final route = server.uri('/api/balaur/household-invitations');
  final administratorHeaders = <String, String>{
    HttpHeaders.authorizationHeader: administratorToken,
  };

  final unauthenticatedList = await _request('GET', route);
  expect(unauthenticatedList.statusCode, HttpStatus.unauthorized);

  final invalidRole = await _request(
    'POST',
    route,
    headers: administratorHeaders,
    jsonBody: <String, Object>{'role': 'superuser'},
  );
  expect(invalidRole.statusCode, HttpStatus.badRequest);

  final memberInvitation = await _createInvitation(
    server: server,
    token: administratorToken,
    role: 'member',
  );
  expect(memberInvitation.response.header('cache-control'), 'no-store');
  expect(memberInvitation.value, hasLength(48));
  expect(memberInvitation.invitation['role'], 'member');
  expect(memberInvitation.invitation['state'], 'active');
  expect(memberInvitation.invitation, isNot(contains('tokenHash')));
  final expiresAt = DateTime.parse(
    memberInvitation.invitation['expiresAt']! as String,
  );
  final remaining = expiresAt.difference(DateTime.now().toUtc());
  expect(remaining, greaterThan(const Duration(hours: 23, minutes: 59)));
  expect(remaining, lessThanOrEqualTo(const Duration(hours: 24)));

  final listed = await _request('GET', route, headers: administratorHeaders);
  expect(listed.statusCode, HttpStatus.ok);
  final listedItems = listed.jsonBody['items']! as List<Object?>;
  expect(listedItems, hasLength(1));
  final listedInvitation = listedItems.single! as Map<String, Object?>;
  expect(listedInvitation['id'], memberInvitation.invitation['id']);
  expect(listedInvitation, isNot(contains('value')));
  expect(listedInvitation, isNot(contains('tokenHash')));

  final directInvitationRead = await _request(
    'GET',
    server.uri('/api/collections/household_invitations/records'),
    headers: administratorHeaders,
  );
  expect(directInvitationRead.statusCode, HttpStatus.forbidden);

  final memberRedemption = await _redeemInvitation(
    server: server,
    value: memberInvitation.value,
    displayName: 'Mara',
    email: 'mara@example.com',
  );
  expect(memberRedemption.statusCode, HttpStatus.ok);
  expect(memberRedemption.header('cache-control'), 'no-store');
  final memberAuthentication = memberRedemption.jsonBody;
  final memberRecord = memberAuthentication['record']! as Map<String, Object?>;
  expect(memberRecord['collectionName'], 'members');
  expect(memberRecord['collectionName'], isNot('_superusers'));
  expect(memberRecord['role'], 'member');
  expect(memberRecord['displayName'], 'Mara');
  final memberToken = memberAuthentication['token']! as String;
  final memberHeaders = <String, String>{
    HttpHeaders.authorizationHeader: memberToken,
  };

  final replay = await _redeemInvitation(
    server: server,
    value: memberInvitation.value,
    displayName: 'Replay',
    email: 'replay@example.com',
  );
  expect(replay.statusCode, HttpStatus.conflict);

  final concurrentInvitation = await _createInvitation(
    server: server,
    token: administratorToken,
    role: 'member',
  );
  final concurrentRedemptions = await Future.wait([
    _redeemInvitation(
      server: server,
      value: concurrentInvitation.value,
      displayName: 'First concurrent member',
      email: 'first-concurrent@example.com',
    ),
    _redeemInvitation(
      server: server,
      value: concurrentInvitation.value,
      displayName: 'Second concurrent member',
      email: 'second-concurrent@example.com',
    ),
  ]);
  expect(
    concurrentRedemptions.map((response) => response.statusCode).toList()
      ..sort(),
    [HttpStatus.ok, HttpStatus.conflict],
  );
  final concurrentAuthentications = await Future.wait([
    _request(
      'POST',
      server.uri('/api/collections/members/auth-with-password'),
      jsonBody: <String, Object>{
        'identity': 'first-concurrent@example.com',
        'password': 'new-correct-horse',
      },
    ),
    _request(
      'POST',
      server.uri('/api/collections/members/auth-with-password'),
      jsonBody: <String, Object>{
        'identity': 'second-concurrent@example.com',
        'password': 'new-correct-horse',
      },
    ),
  ]);
  expect(
    concurrentAuthentications.map((response) => response.statusCode).toList()
      ..sort(),
    [HttpStatus.ok, HttpStatus.badRequest],
  );

  final forbiddenList = await _request('GET', route, headers: memberHeaders);
  expect(forbiddenList.statusCode, HttpStatus.forbidden);
  final forbiddenCreate = await _request(
    'POST',
    route,
    headers: memberHeaders,
    jsonBody: <String, Object>{'role': 'household_administrator'},
  );
  expect(forbiddenCreate.statusCode, HttpStatus.forbidden);
  final forbiddenCancel = await _request(
    'POST',
    server.uri(
      '/api/balaur/household-invitations/'
      '${memberInvitation.invitation['id']}/cancel',
    ),
    headers: memberHeaders,
  );
  expect(forbiddenCancel.statusCode, HttpStatus.forbidden);

  await _verifyCalendarConnectionRoutes(
    server: server,
    administratorToken: administratorToken,
    memberToken: memberToken,
  );
  await _verifyHouseholdArchiveRoutes(
    server: server,
    administratorToken: administratorToken,
    memberToken: memberToken,
  );

  final administratorInvitation = await _createInvitation(
    server: server,
    token: administratorToken,
    role: 'household_administrator',
  );
  final administratorRedemption = await _redeemInvitation(
    server: server,
    value: administratorInvitation.value,
    displayName: 'Dana',
    email: 'dana@example.com',
  );
  expect(administratorRedemption.statusCode, HttpStatus.ok);
  final secondAdministrator =
      administratorRedemption.jsonBody['record']! as Map<String, Object?>;
  expect(secondAdministrator['role'], 'household_administrator');
  final secondAdministratorToken =
      administratorRedemption.jsonBody['token']! as String;

  final cancelCandidate = await _createInvitation(
    server: server,
    token: secondAdministratorToken,
    role: 'member',
  );
  final canceled = await _request(
    'POST',
    server.uri(
      '/api/balaur/household-invitations/'
      '${cancelCandidate.invitation['id']}/cancel',
    ),
    headers: <String, String>{
      HttpHeaders.authorizationHeader: secondAdministratorToken,
    },
  );
  expect(canceled.statusCode, HttpStatus.ok);
  final canceledInvitation =
      canceled.jsonBody['invitation']! as Map<String, Object?>;
  expect(canceledInvitation['state'], 'canceled');
  final canceledRedemption = await _redeemInvitation(
    server: server,
    value: cancelCandidate.value,
    displayName: 'Canceled',
    email: 'canceled@example.com',
  );
  expect(canceledRedemption.statusCode, HttpStatus.conflict);

  final expiredCandidate = await _createInvitation(
    server: server,
    token: administratorToken,
    role: 'member',
  );
  final superuserAuthentication = await _request(
    'POST',
    server.uri('/api/collections/_superusers/auth-with-password'),
    jsonBody: <String, Object>{
      'identity': _testSuperuserEmail,
      'password': _testSuperuserPassword,
    },
  );
  expect(superuserAuthentication.statusCode, HttpStatus.ok);
  final superuserToken = superuserAuthentication.jsonBody['token']! as String;
  final expireRecord = await _request(
    'PATCH',
    server.uri(
      '/api/collections/household_invitations/records/'
      '${expiredCandidate.invitation['id']}',
    ),
    headers: <String, String>{HttpHeaders.authorizationHeader: superuserToken},
    jsonBody: <String, Object>{
      'expiresAt': DateTime.now()
          .toUtc()
          .subtract(const Duration(minutes: 1))
          .toIso8601String(),
    },
  );
  expect(expireRecord.statusCode, HttpStatus.ok);

  final expiredRedemption = await _redeemInvitation(
    server: server,
    value: expiredCandidate.value,
    displayName: 'Expired',
    email: 'expired@example.com',
  );
  expect(expiredRedemption.statusCode, HttpStatus.gone);

  final invalidRedemption = await _redeemInvitation(
    server: server,
    value: 'UnknownInvitation'.padRight(48, 'Z'),
    displayName: 'Unknown',
    email: 'unknown@example.com',
  );
  expect(invalidRedemption.statusCode, HttpStatus.notFound);
}

Future<void> _verifyCalendarConnectionRoutes({
  required _PocketBaseServer server,
  required String administratorToken,
  required String memberToken,
}) async {
  final route = server.uri('/api/balaur/calendar-connection');
  final administratorHeaders = <String, String>{
    HttpHeaders.authorizationHeader: administratorToken,
  };
  final memberHeaders = <String, String>{
    HttpHeaders.authorizationHeader: memberToken,
  };

  final unauthenticated = await _request('GET', route);
  expect(unauthenticated.statusCode, HttpStatus.unauthorized);

  final memberStatus = await _request('GET', route, headers: memberHeaders);
  expect(memberStatus.statusCode, HttpStatus.ok);
  expect(
    memberStatus.jsonBody['connection'],
    isA<Map<String, Object?>>()
        .having((value) => value['status'], 'status', 'disconnected')
        .having(
          (value) => value.containsKey('refreshTokenCiphertext'),
          'secret',
          false,
        )
        .having((value) => value.containsKey('oauthStateHash'), 'state', false),
  );

  for (final path in <String>[
    '/api/balaur/calendar-connection/authorize',
    '/api/balaur/calendar-connection/select',
    '/api/balaur/calendar-connection/disconnect',
  ]) {
    final forbidden = await _request(
      'POST',
      server.uri(path),
      headers: memberHeaders,
      jsonBody: <String, Object>{
        'replace': false,
        'calendarId': 'family@example.com',
      },
    );
    expect(forbidden.statusCode, HttpStatus.forbidden);
  }

  final begin = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/authorize'),
    headers: administratorHeaders,
    jsonBody: <String, Object>{'replace': false},
  );
  expect(begin.statusCode, HttpStatus.ok);
  expect(begin.header('cache-control'), 'no-store');
  expect(begin.jsonBody['requestedScopes'], <Object?>[
    'https://www.googleapis.com/auth/calendar.calendarlist.readonly',
    'https://www.googleapis.com/auth/calendar.events.readonly',
  ]);
  final authorizationUrl = Uri.parse(
    begin.jsonBody['authorizationUrl']! as String,
  );
  final state = authorizationUrl.queryParameters['state'];
  expect(state, isNotNull);
  expect(state, hasLength(64));

  final invalidState = await _request(
    'GET',
    server.uri(
      '/api/balaur/calendar-source/callback'
      '?state=${'InvalidCalendarState'.padRight(64, 'Z')}'
      '&code=fake-authorization-code',
    ),
  );
  expect(invalidState.statusCode, HttpStatus.badRequest);
  final stillAuthorizing = await _request(
    'GET',
    route,
    headers: administratorHeaders,
  );
  expect(
    (stillAuthorizing.jsonBody['connection']!
        as Map<String, Object?>)['status'],
    'authorizing',
  );

  final denialUrl = authorizationUrl.replace(
    queryParameters: <String, String>{
      ...authorizationUrl.queryParameters,
      'decision': 'deny',
    },
  );
  final denial = await _request('GET', denialUrl);
  expect(denial.statusCode, HttpStatus.ok);
  expect(denial.body, contains('authorization failed'));
  final deniedStatus = await _request(
    'GET',
    route,
    headers: administratorHeaders,
  );
  expect(
    (deniedStatus.jsonBody['connection']! as Map<String, Object?>)['status'],
    'authorization_failed',
  );

  final reconnect = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/authorize'),
    headers: administratorHeaders,
    jsonBody: <String, Object>{'replace': false},
  );
  expect(reconnect.statusCode, HttpStatus.ok);
  final reconnectUrl = Uri.parse(
    reconnect.jsonBody['authorizationUrl']! as String,
  );
  final callback = await _request('GET', reconnectUrl);
  expect(callback.statusCode, HttpStatus.ok);
  expect(callback.body, contains('authorization is complete'));

  final selectionStatus = await _request(
    'GET',
    route,
    headers: administratorHeaders,
  );
  final selectionConnection =
      selectionStatus.jsonBody['connection']! as Map<String, Object?>;
  expect(selectionConnection['status'], 'awaiting_selection');
  expect(selectionConnection['householdTimeZone'], 'Europe/Bucharest');
  final available = selectionConnection['availableCalendars']! as List<Object?>;
  expect(available, hasLength(2));

  final invalidSelection = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/select'),
    headers: administratorHeaders,
    jsonBody: <String, Object>{'calendarId': 'unknown@example.com'},
  );
  expect(invalidSelection.statusCode, HttpStatus.badRequest);

  final selected = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/select'),
    headers: administratorHeaders,
    jsonBody: <String, Object>{'calendarId': 'family@example.com'},
  );
  expect(selected.statusCode, HttpStatus.ok);
  final connected = selected.jsonBody['connection']! as Map<String, Object?>;
  expect(connected['status'], 'connected');
  expect(connected['selectedCalendar'], <String, Object?>{
    'id': 'family@example.com',
    'name': 'Family',
    'colorHex': '#4285f4',
  });
  expect(connected, isNot(contains('refreshTokenCiphertext')));

  final unauthenticatedEntries = await _request(
    'GET',
    server.uri(
      '/api/balaur/calendar-entries'
      '?timeMin=2026-08-21T00%3A00%3A00Z'
      '&timeMax=2026-08-22T00%3A00%3A00Z',
    ),
  );
  expect(unauthenticatedEntries.statusCode, HttpStatus.unauthorized);

  final invalidRange = await _request(
    'GET',
    server.uri(
      '/api/balaur/calendar-entries'
      '?timeMin=2026-01-01T00%3A00%3A00Z'
      '&timeMax=2026-04-01T00%3A00%3A00Z',
    ),
    headers: memberHeaders,
  );
  expect(invalidRange.statusCode, HttpStatus.badRequest);

  final entriesResponse = await _request(
    'GET',
    server.uri(
      '/api/balaur/calendar-entries'
      '?timeMin=2026-08-21T00%3A00%3A00Z'
      '&timeMax=2026-08-22T00%3A00%3A00Z',
    ),
    headers: memberHeaders,
  );
  expect(entriesResponse.statusCode, HttpStatus.ok);
  expect(entriesResponse.header('cache-control'), 'no-store');
  final entries = entriesResponse.jsonBody['items']! as List<Object?>;
  expect(entries, hasLength(2));

  final allDay = entries.first! as Map<String, Object?>;
  expect(allDay['title'], 'Busy');
  expect(allDay['start'], '2026-08-21');
  expect(allDay['end'], '2026-08-22');
  expect(allDay['allDay'], isTrue);
  expect(allDay['colorHex'], '#4285f4');
  final timed = entries.last! as Map<String, Object?>;
  expect(timed['title'], 'Family dinner');
  expect(timed['start'], '2026-08-21T18:00:00Z');
  expect(timed['end'], '2026-08-21T19:30:00Z');
  expect(timed['location'], 'Home');
  expect(
    entries,
    isNot(
      contains(
        predicate<Map<String, Object?>>(
          (entry) => entry['title'] == 'Canceled',
        ),
      ),
    ),
  );

  final directRead = await _request(
    'GET',
    server.uri('/api/collections/calendar_connections/records'),
    headers: administratorHeaders,
  );
  expect(directRead.statusCode, HttpStatus.forbidden);

  final startWithoutReplace = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/authorize'),
    headers: administratorHeaders,
    jsonBody: <String, Object>{'replace': false},
  );
  expect(startWithoutReplace.statusCode, HttpStatus.conflict);

  final revoked = await _request(
    'POST',
    server.uri('/api/balaur/testing/calendar-source/revoke'),
    headers: administratorHeaders,
  );
  expect(revoked.statusCode, HttpStatus.noContent);
  final revokedEntries = await _request(
    'GET',
    server.uri(
      '/api/balaur/calendar-entries'
      '?timeMin=2026-08-21T00%3A00%3A00Z'
      '&timeMax=2026-08-22T00%3A00%3A00Z',
    ),
    headers: memberHeaders,
  );
  expect(revokedEntries.statusCode, 424);
  final revokedStatus = await _request(
    'GET',
    route,
    headers: administratorHeaders,
  );
  expect(
    (revokedStatus.jsonBody['connection']! as Map<String, Object?>)['status'],
    'disconnected',
  );

  final recoveredAuthorization = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/authorize'),
    headers: administratorHeaders,
    jsonBody: <String, Object>{'replace': false},
  );
  final recoveredUrl = Uri.parse(
    recoveredAuthorization.jsonBody['authorizationUrl']! as String,
  );
  expect((await _request('GET', recoveredUrl)).statusCode, HttpStatus.ok);
  final recoveredSelection = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/select'),
    headers: administratorHeaders,
    jsonBody: <String, Object>{'calendarId': 'family@example.com'},
  );
  expect(recoveredSelection.statusCode, HttpStatus.ok);
  final replace = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/authorize'),
    headers: administratorHeaders,
    jsonBody: <String, Object>{'replace': true},
  );
  expect(replace.statusCode, HttpStatus.ok);

  final disconnected = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/disconnect'),
    headers: administratorHeaders,
  );
  expect(disconnected.statusCode, HttpStatus.ok);
  expect(
    (disconnected.jsonBody['connection']! as Map<String, Object?>)['status'],
    'disconnected',
  );
}

Future<void> _verifyHouseholdArchiveRoutes({
  required _PocketBaseServer server,
  required String administratorToken,
  required String memberToken,
}) async {
  final route = server.uri('/api/balaur/household-archive');
  final unauthenticated = await _request(
    'POST',
    route,
    jsonBody: <String, Object>{
      'password': 'correct-horse',
      'includeCalendarSnapshot': false,
    },
  );
  expect(unauthenticated.statusCode, HttpStatus.unauthorized);

  final member = await _request(
    'POST',
    route,
    headers: <String, String>{HttpHeaders.authorizationHeader: memberToken},
    jsonBody: <String, Object>{
      'password': 'new-correct-horse',
      'includeCalendarSnapshot': false,
    },
  );
  expect(member.statusCode, HttpStatus.forbidden);

  final wrongPassword = await _request(
    'POST',
    route,
    headers: <String, String>{
      HttpHeaders.authorizationHeader: administratorToken,
    },
    jsonBody: <String, Object>{
      'password': 'wrong-password',
      'includeCalendarSnapshot': false,
    },
  );
  expect(wrongPassword.statusCode, HttpStatus.unauthorized);

  final authorization = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/authorize'),
    headers: <String, String>{
      HttpHeaders.authorizationHeader: administratorToken,
    },
    jsonBody: <String, Object>{'replace': false},
  );
  expect(authorization.statusCode, HttpStatus.ok);
  final authorizationUrl = Uri.parse(
    authorization.jsonBody['authorizationUrl']! as String,
  );
  expect((await _request('GET', authorizationUrl)).statusCode, HttpStatus.ok);
  final selected = await _request(
    'POST',
    server.uri('/api/balaur/calendar-connection/select'),
    headers: <String, String>{
      HttpHeaders.authorizationHeader: administratorToken,
    },
    jsonBody: <String, Object>{'calendarId': 'family@example.com'},
  );
  expect(selected.statusCode, HttpStatus.ok);

  final exported = await _binaryRequest(
    'POST',
    route,
    headers: <String, String>{
      HttpHeaders.authorizationHeader: administratorToken,
    },
    jsonBody: <String, Object>{
      'password': 'correct-horse',
      'includeCalendarSnapshot': true,
    },
  );
  expect(exported.statusCode, HttpStatus.ok);
  expect(exported.header('content-type'), contains('application/zip'));
  expect(exported.header('content-disposition'), contains('balaur-household-'));

  final archive = ZipDecoder().decodeBytes(exported.body);
  final files = <String, List<int>>{
    for (final file in archive.files)
      file.name: file.isFile ? file.content as List<int> : <int>[],
  };
  expect(
    files.keys,
    containsAll(<String>[
      'manifest.json',
      'household-state.json',
      'members.json',
      'shared-files/',
      'calendar.ics',
    ]),
  );
  final manifest =
      jsonDecode(utf8.decode(files['manifest.json']!)) as Map<String, Object?>;
  expect(manifest['format'], 'balaur-household-archive');
  expect(manifest['version'], 1);
  final state = jsonDecode(
    utf8.decode(files['household-state.json']!),
  ) as Map<String, Object?>;
  expect(
    (state['household']! as Map<String, Object?>)['timeZone'],
    'Europe/Bucharest',
  );
  final members =
      jsonDecode(utf8.decode(files['members.json']!)) as List<Object?>;
  expect(members, isNotEmpty);
  final calendarSnapshot = utf8.decode(files['calendar.ics']!);
  expect(calendarSnapshot, contains('BEGIN:VCALENDAR'));
  expect(calendarSnapshot, contains('Family dinner'));

  final archiveText = files.values
      .map((bytes) => utf8.decode(bytes, allowMalformed: true))
      .join('\\n');
  for (final excluded in <String>[
    'correct-horse',
    'fake-refresh-token',
    'RouteTestCalendarEncryptionKey12',
    'tokenHash',
    'passwordHash',
    'Conversation',
  ]) {
    expect(archiveText, isNot(contains(excluded)));
  }
}

Future<_CreatedInvitationResponse> _createInvitation({
  required _PocketBaseServer server,
  required String token,
  required String role,
}) async {
  final response = await _request(
    'POST',
    server.uri('/api/balaur/household-invitations'),
    headers: <String, String>{HttpHeaders.authorizationHeader: token},
    jsonBody: <String, Object>{'role': role},
  );
  expect(response.statusCode, HttpStatus.created);
  final invitation = response.jsonBody['invitation']! as Map<String, Object?>;
  final value = response.jsonBody['value']! as String;
  return _CreatedInvitationResponse(
    response: response,
    invitation: invitation,
    value: value,
  );
}

Future<_HttpResponse> _redeemInvitation({
  required _PocketBaseServer server,
  required String value,
  required String displayName,
  required String email,
}) {
  return _request(
    'POST',
    server.uri('/api/balaur/household-invitations/redeem'),
    jsonBody: <String, Object>{
      'invitation': value,
      'displayName': displayName,
      'email': email,
      'password': 'new-correct-horse',
      'passwordConfirm': 'new-correct-horse',
    },
  );
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
        'BALAUR_CALENDAR_SOURCE_ADAPTER': 'fake',
        'BALAUR_CALENDAR_ENCRYPTION_KEY': 'RouteTestCalendarEncryptionKey12',
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

Future<_BinaryResponse> _binaryRequest(
  String method,
  Uri uri, {
  Map<String, String> headers = const <String, String>{},
  Map<String, Object>? jsonBody,
}) async {
  final client = HttpClient();
  try {
    final request = await client.openUrl(method, uri);
    headers.forEach(request.headers.set);
    if (jsonBody != null) {
      request.headers.contentType = ContentType.json;
      request.write(jsonEncode(jsonBody));
    }
    final response = await request.close();
    final responseHeaders = <String, String>{};
    response.headers.forEach((name, values) {
      responseHeaders[name.toLowerCase()] = values.join(', ');
    });
    final body = await response.fold<List<int>>(
      <int>[],
      (bytes, chunk) => bytes..addAll(chunk),
    );
    return _BinaryResponse(
      statusCode: response.statusCode,
      headers: responseHeaders,
      body: body,
    );
  } finally {
    client.close(force: true);
  }
}

final class _BinaryResponse {
  const _BinaryResponse({
    required this.statusCode,
    required this.headers,
    required this.body,
  });

  final int statusCode;
  final Map<String, String> headers;
  final List<int> body;

  String? header(String name) => headers[name.toLowerCase()];
}

final class _CreatedInvitationResponse {
  const _CreatedInvitationResponse({
    required this.response,
    required this.invitation,
    required this.value,
  });

  final _HttpResponse response;
  final Map<String, Object?> invitation;
  final String value;
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
