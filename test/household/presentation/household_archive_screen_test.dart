import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/household_archive.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_archive_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('exports and saves a confirmed Household Archive', (
    tester,
  ) async {
    final state = InMemoryHouseholdGatewayState(
      storedSession: _administratorSession,
    );
    HouseholdArchive? saved;
    await _pumpArchive(
      tester,
      state: state,
      saveArchive: (archive) async {
        saved = archive;
        return true;
      },
    );

    await tester.enterText(
      find.byKey(const Key('household-archive-password')),
      'correct-horse',
    );
    await tester.tap(find.byKey(const Key('include-calendar-snapshot')));
    await tester.pump();
    await tester.ensureVisible(
      find.byKey(const Key('export-household-archive')),
    );
    await tester.tap(find.byKey(const Key('export-household-archive')));
    await tester.pumpAndSettle();

    expect(saved?.fileName, endsWith('.zip'));
    expect(find.text('EXPORT COMPLETE'), findsOneWidget);
    expect(state.archiveExports.single.includeCalendarSnapshot, isTrue);
    expect(
      tester
          .widget<TextField>(
            find.byKey(const Key('household-archive-password')),
          )
          .controller
          ?.text,
      isEmpty,
    );
  });

  testWidgets('reports export progress', (tester) async {
    final state = InMemoryHouseholdGatewayState(
      storedSession: _administratorSession,
    );
    await _pumpArchive(tester, state: state, pause: true);
    await tester.enterText(
      find.byKey(const Key('household-archive-password')),
      'correct-horse',
    );

    await tester.ensureVisible(
      find.byKey(const Key('export-household-archive')),
    );
    await tester.tap(find.byKey(const Key('export-household-archive')));
    await tester.pump();

    expect(find.text('CREATING ARCHIVE'), findsOneWidget);
    expect(
      find.text('The Household Server is creating the Household Archive.'),
      findsOneWidget,
    );
  });

  testWidgets('rejects an incorrect confirmation password', (tester) async {
    final state = InMemoryHouseholdGatewayState(
      storedSession: _administratorSession,
    );
    await _pumpArchive(tester, state: state);
    await tester.enterText(
      find.byKey(const Key('household-archive-password')),
      'wrong-password',
    );

    await tester.ensureVisible(
      find.byKey(const Key('export-household-archive')),
    );
    await tester.tap(find.byKey(const Key('export-household-archive')));
    await tester.pumpAndSettle();

    expect(find.text('EXPORT FAILED'), findsOneWidget);
    expect(find.textContaining('password is not correct'), findsOneWidget);
  });

  testWidgets('denies archive export to a Household Member', (tester) async {
    final state = InMemoryHouseholdGatewayState(storedSession: _memberSession);
    await _pumpArchive(tester, state: state, session: _memberSession);

    expect(
      find.text('HOUSEHOLD ADMINISTRATOR ACCESS IS REQUIRED'),
      findsOneWidget,
    );
    expect(find.byKey(const Key('export-household-archive')), findsNothing);
  });
}

final _address = HouseholdServerAddress.parse('https://household.example.com');

final _administratorSession = HouseholdSession(
  serverAddress: _address,
  member: const HouseholdMember(
    id: 'administrator-example',
    displayName: 'Alex',
    email: 'alex@example.com',
    role: HouseholdMemberRole.administrator,
  ),
);

final _memberSession = HouseholdSession(
  serverAddress: _address,
  member: const HouseholdMember(
    id: 'member-example',
    displayName: 'Mara',
    email: 'mara@example.com',
    role: HouseholdMemberRole.member,
  ),
);

Future<void> _pumpArchive(
  WidgetTester tester, {
  required InMemoryHouseholdGatewayState state,
  HouseholdSession? session,
  bool pause = false,
  Future<bool> Function(HouseholdArchive archive)? saveArchive,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: BalaurTheme.light(),
      home: HouseholdArchiveScreen(
        gateway: InMemoryHouseholdGateway(
          state: state,
          pauseArchiveExport: pause,
        ),
        session: session ?? _administratorSession,
        onBack: () {},
        saveArchive: saveArchive ?? (_) async => true,
      ),
    ),
  );
  await tester.pumpAndSettle();
}
