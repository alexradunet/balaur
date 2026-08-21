import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_pairing_gate.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows the unpaired form and requires an HTTPS address', (
    tester,
  ) async {
    await _pumpGate(tester, InMemoryHouseholdGateway());
    await tester.pumpAndSettle();

    expect(find.text('Pair this device'), findsWidgets);
    expect(find.byKey(const Key('household-server-address')), findsOneWidget);

    await tester.enterText(
      find.byKey(const Key('household-server-address')),
      'http://household.example.com',
    );
    await tester.enterText(
      find.byKey(const Key('household-email')),
      'alex@example.com',
    );
    await tester.enterText(
      find.byKey(const Key('household-password')),
      'correct-horse',
    );
    await tester.tap(find.byKey(const Key('pair-household')));
    await tester.pump();

    expect(find.text('Enter a stable HTTPS address.'), findsOneWidget);
  });

  testWidgets('shows the connecting state', (tester) async {
    await _pumpGate(tester, InMemoryHouseholdGateway(pauseRestore: true));
    await tester.pump();

    expect(find.text('Connecting to the Household Server…'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });

  testWidgets('reports an authentication failure', (tester) async {
    await _pumpGate(
      tester,
      InMemoryHouseholdGateway(
        pairFailure: HouseholdGatewayFailure.authentication,
      ),
    );
    await tester.pumpAndSettle();

    await _enterPairingDetails(tester);
    await tester.tap(find.byKey(const Key('pair-household')));
    await tester.pumpAndSettle();

    expect(find.text('PAIRING FAILED'), findsOneWidget);
    expect(
      find.text('The email address or password is not correct.'),
      findsOneWidget,
    );
  });

  testWidgets('reports a connection failure', (tester) async {
    await _pumpGate(
      tester,
      InMemoryHouseholdGateway(pairFailure: HouseholdGatewayFailure.connection),
    );
    await tester.pumpAndSettle();

    await _enterPairingDetails(tester);
    await tester.tap(find.byKey(const Key('pair-household')));
    await tester.pumpAndSettle();

    expect(
      find.text(
        'Balaur could not reach the Household Server. Check its address and your connection.',
      ),
      findsOneWidget,
    );
  });

  testWidgets('shows the paired state and enters the application', (
    tester,
  ) async {
    final state = InMemoryHouseholdGatewayState(storedSession: _session);
    await _pumpGate(tester, InMemoryHouseholdGateway(state: state));
    await tester.pumpAndSettle();

    expect(find.text('Household paired'), findsOneWidget);
    expect(find.text('Alex'), findsOneWidget);

    await tester.tap(find.byKey(const Key('enter-paired-application')));
    await tester.pumpAndSettle();

    expect(find.text('Paired application'), findsOneWidget);
  });
}

Future<void> _pumpGate(
  WidgetTester tester,
  InMemoryHouseholdGateway gateway,
) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: BalaurTheme.light(),
      home: HouseholdPairingGate(
        gateway: gateway,
        pairedChild: const Center(child: Text('Paired application')),
      ),
    ),
  );
}

Future<void> _enterPairingDetails(WidgetTester tester) async {
  await tester.enterText(
    find.byKey(const Key('household-server-address')),
    'https://household.example.com',
  );
  await tester.enterText(
    find.byKey(const Key('household-email')),
    'alex@example.com',
  );
  await tester.enterText(
    find.byKey(const Key('household-password')),
    'correct-horse',
  );
}

final _session = HouseholdSession(
  serverAddress: HouseholdServerAddress.parse('https://household.example.com'),
  member: const HouseholdMember(
    id: 'member-example',
    displayName: 'Alex',
    email: 'alex@example.com',
    role: HouseholdMemberRole.administrator,
  ),
);
