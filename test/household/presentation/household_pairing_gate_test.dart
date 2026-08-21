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

  testWidgets('accepts loopback HTTP when explicitly enabled', (tester) async {
    await _pumpGate(
      tester,
      InMemoryHouseholdGateway(
        pairFailure: HouseholdGatewayFailure.authentication,
      ),
      allowInsecureLoopback: true,
    );
    await tester.pumpAndSettle();

    await tester.enterText(
      find.byKey(const Key('household-server-address')),
      'http://localhost:8090',
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
    await tester.pumpAndSettle();

    expect(find.text('Enter a stable HTTPS address.'), findsNothing);
    expect(find.text('PAIRING FAILED'), findsOneWidget);
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

  testWidgets('joins through manual Household Invitation entry', (
    tester,
  ) async {
    final state = InMemoryHouseholdGatewayState(storedSession: _session);
    final gateway = InMemoryHouseholdGateway(state: state);
    final created = await gateway.createInvitation(
      role: HouseholdMemberRole.administrator,
    );
    await gateway.signOut();
    await _pumpGate(tester, gateway);
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('use-household-invitation')));
    await tester.pump();
    await tester.enterText(
      find.byKey(const Key('invitation-household-server-address')),
      created.payload.serverAddress.value,
    );
    await tester.enterText(
      find.byKey(const Key('household-invitation-value')),
      created.payload.value,
    );
    await _enterInvitationMemberDetails(tester);
    await tester.ensureVisible(
      find.byKey(const Key('redeem-household-invitation')),
    );
    await tester.tap(find.byKey(const Key('redeem-household-invitation')));
    await tester.pumpAndSettle();

    expect(find.text('Household paired'), findsOneWidget);
    expect(find.text('Mara'), findsOneWidget);
    expect(find.text('Household Administrator'), findsOneWidget);
  });

  testWidgets('fills pairing details from a Household Invitation QR code', (
    tester,
  ) async {
    final state = InMemoryHouseholdGatewayState(storedSession: _session);
    final gateway = InMemoryHouseholdGateway(state: state);
    final created = await gateway.createInvitation(
      role: HouseholdMemberRole.member,
    );
    await gateway.signOut();
    await _pumpGate(
      tester,
      gateway,
      onScanInvitation: (_) async => created.payload.qrValue,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('use-household-invitation')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('scan-household-invitation')));
    await tester.pumpAndSettle();

    final addressField = tester.widget<TextFormField>(
      find.byKey(const Key('invitation-household-server-address')),
    );
    final invitationField = tester.widget<TextFormField>(
      find.byKey(const Key('household-invitation-value')),
    );
    expect(addressField.controller?.text, created.payload.serverAddress.value);
    expect(invitationField.controller?.text, created.payload.value);
  });

  testWidgets('rejects malformed Household Invitation QR data', (tester) async {
    await _pumpGate(
      tester,
      InMemoryHouseholdGateway(),
      onScanInvitation: (_) async => 'not-a-household-invitation',
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('use-household-invitation')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('scan-household-invitation')));
    await tester.pumpAndSettle();

    expect(find.text('THE INVITATION CODE IS NOT VALID'), findsOneWidget);
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
  InMemoryHouseholdGateway gateway, {
  Future<String?> Function(BuildContext context)? onScanInvitation,
  bool allowInsecureLoopback = false,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: BalaurTheme.light(),
      home: HouseholdPairingGate(
        gateway: gateway,
        pairedChild: const Center(child: Text('Paired application')),
        onScanInvitation: onScanInvitation,
        allowInsecureLoopback: allowInsecureLoopback,
      ),
    ),
  );
}

Future<void> _enterInvitationMemberDetails(WidgetTester tester) async {
  await tester.enterText(
    find.byKey(const Key('invited-household-display-name')),
    'Mara',
  );
  await tester.enterText(
    find.byKey(const Key('invited-household-email')),
    'mara@example.com',
  );
  await tester.enterText(
    find.byKey(const Key('invited-household-password')),
    'new-correct-horse',
  );
  await tester.enterText(
    find.byKey(const Key('invited-household-password-confirm')),
    'new-correct-horse',
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
