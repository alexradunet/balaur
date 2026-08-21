import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_invitations_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('creates an assigned Household Invitation with pairing data', (
    tester,
  ) async {
    final gateway = _gateway(_administratorSession);
    await _pumpScreen(tester, gateway, _administratorSession);
    await tester.pumpAndSettle();

    expect(find.text('NO HOUSEHOLD INVITATIONS'), findsOneWidget);

    await tester.tap(find.byType(DropdownButtonFormField<HouseholdMemberRole>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Household Administrator').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('create-household-invitation')));
    await tester.pumpAndSettle();

    expect(find.text('Household Invitation ready'), findsOneWidget);
    expect(
      find.byKey(const Key('created-household-invitation-value')),
      findsOneWidget,
    );
    expect(find.text('Active'), findsOneWidget);
    expect(find.text('Household Administrator'), findsWidgets);
  });

  testWidgets('confirms cancellation before disabling an invitation', (
    tester,
  ) async {
    final gateway = _gateway(_administratorSession);
    final created = await gateway.createInvitation(
      role: HouseholdMemberRole.member,
    );
    await _pumpScreen(tester, gateway, _administratorSession);
    await tester.pumpAndSettle();

    final cancelButton = find.byKey(
      Key('cancel-household-invitation-${created.invitation.id}'),
    );
    await tester.ensureVisible(cancelButton);
    await tester.pumpAndSettle();
    await tester.tap(cancelButton);
    await tester.pumpAndSettle();

    expect(find.text('Cancel this Household Invitation?'), findsOneWidget);
    await tester.tap(
      find.byKey(const Key('confirm-cancel-household-invitation')),
    );
    await tester.pumpAndSettle();

    expect(find.text('Canceled'), findsOneWidget);
    expect(
      find.byKey(Key('cancel-household-invitation-${created.invitation.id}')),
      findsNothing,
    );
  });

  testWidgets('blocks a Household Member from invitation management', (
    tester,
  ) async {
    final gateway = _gateway(_memberSession);
    await _pumpScreen(tester, gateway, _memberSession);
    await tester.pump();

    expect(
      find.text('HOUSEHOLD ADMINISTRATOR ACCESS IS REQUIRED'),
      findsOneWidget,
    );
    expect(find.byKey(const Key('create-household-invitation')), findsNothing);
  });

  testWidgets('fits a narrow screen without an overflow', (tester) async {
    await tester.binding.setSurfaceSize(const Size(360, 780));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final gateway = _gateway(_administratorSession);
    await gateway.createInvitation(role: HouseholdMemberRole.member);

    await _pumpScreen(tester, gateway, _administratorSession);
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.text('Household Invitations'), findsOneWidget);
  });
}

InMemoryHouseholdGateway _gateway(HouseholdSession session) {
  return InMemoryHouseholdGateway(
    state: InMemoryHouseholdGatewayState(storedSession: session),
    now: () => DateTime.utc(2026, 8, 21, 12),
  );
}

Future<void> _pumpScreen(
  WidgetTester tester,
  InMemoryHouseholdGateway gateway,
  HouseholdSession session,
) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: BalaurTheme.light(),
      home: HouseholdInvitationsScreen(
        gateway: gateway,
        session: session,
        onBack: () {},
      ),
    ),
  );
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
