import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_invitations_screen.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'household_invitations_screen.stories.g.dart';

const component = ComponentMeta(
  name: 'Household invitations',
  path: 'Screens/Household',
);
const meta = Meta(HouseholdInvitationsScreen.new);

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

final _activeInvitation = HouseholdInvitation(
  id: 'invitation-active',
  creatorId: _administratorSession.member.id,
  creatorDisplayName: _administratorSession.member.displayName,
  expiresAt: DateTime.utc(2026, 8, 22, 12),
  role: HouseholdMemberRole.member,
  status: HouseholdInvitationStatus.active,
);

final _usedInvitation = HouseholdInvitation(
  id: 'invitation-used',
  creatorId: _administratorSession.member.id,
  creatorDisplayName: _administratorSession.member.displayName,
  expiresAt: DateTime.utc(2026, 8, 21, 8),
  role: HouseholdMemberRole.administrator,
  status: HouseholdInvitationStatus.used,
);

final _expiredInvitation = HouseholdInvitation(
  id: 'invitation-expired',
  creatorId: _administratorSession.member.id,
  creatorDisplayName: _administratorSession.member.displayName,
  expiresAt: DateTime.utc(2026, 8, 20, 12),
  role: HouseholdMemberRole.member,
  status: HouseholdInvitationStatus.expired,
);

final _canceledInvitation = HouseholdInvitation(
  id: 'invitation-canceled',
  creatorId: _administratorSession.member.id,
  creatorDisplayName: _administratorSession.member.displayName,
  expiresAt: DateTime.utc(2026, 8, 22, 8),
  role: HouseholdMemberRole.member,
  status: HouseholdInvitationStatus.canceled,
);

InMemoryHouseholdGateway _gateway({
  HouseholdSession? session,
  List<HouseholdInvitation> invitations = const [],
  HouseholdGatewayFailure? failure,
  bool pause = false,
  bool pauseCancellation = false,
}) {
  return InMemoryHouseholdGateway(
    state: InMemoryHouseholdGatewayState(
      storedSession: session ?? _administratorSession,
      invitations: invitations,
    ),
    invitationFailure: failure,
    pauseInvitationList: pause,
    pauseInvitationCancellation: pauseCancellation,
    now: () => DateTime.utc(2026, 8, 21, 12),
  );
}

final $Loading = _Story(
  args: _Args.fixed(
    gateway: _gateway(pause: true),
    session: _administratorSession,
    onBack: _noOp,
  ),
);

final $Empty = _Story(
  args: _Args.fixed(
    gateway: _gateway(),
    session: _administratorSession,
    onBack: _noOp,
  ),
  scenarios: [
    _Scenario(
      name: 'Default',
      run: (tester, _) async => tester.pumpAndSettle(),
    ),
  ],
);

final $Existing = _Story(
  args: _Args.fixed(
    gateway: _gateway(invitations: [_activeInvitation, _usedInvitation]),
    session: _administratorSession,
    onBack: _noOp,
  ),
  scenarios: [
    _Scenario(
      name: 'Default',
      run: (tester, _) async => tester.pumpAndSettle(),
    ),
  ],
);

final $Unavailable = _Story(
  args: _Args.fixed(
    gateway: _gateway(invitations: [_expiredInvitation, _canceledInvitation]),
    session: _administratorSession,
    onBack: _noOp,
  ),
  scenarios: [
    _Scenario(
      name: 'Expired and canceled',
      run: (tester, _) async => tester.pumpAndSettle(),
    ),
  ],
);

final $Canceling = _Story(
  args: _Args.fixed(
    gateway: _gateway(
      invitations: [_activeInvitation],
      pauseCancellation: true,
    ),
    session: _administratorSession,
    onBack: _noOp,
  ),
  scenarios: [
    _Scenario(
      name: 'Cancellation in progress',
      run: (tester, _) async {
        await tester.pumpAndSettle();
        final cancelButton = find.byKey(
          const Key('cancel-household-invitation-invitation-active'),
        );
        await tester.ensureVisible(cancelButton);
        await tester.pumpAndSettle();
        await tester.tap(cancelButton);
        await tester.pumpAndSettle();
        await tester.tap(
          find.byKey(const Key('confirm-cancel-household-invitation')),
        );
        await tester.pumpAndSettle();
      },
    ),
  ],
);

final $Created = _Story(
  args: _Args.fixed(
    gateway: _gateway(),
    session: _administratorSession,
    onBack: _noOp,
  ),
  scenarios: [
    _Scenario(
      name: 'Create member invitation',
      run: (tester, _) async {
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const Key('create-household-invitation')));
        await tester.pumpAndSettle();
      },
    ),
  ],
);

final $Failed = _Story(
  args: _Args.fixed(
    gateway: _gateway(failure: HouseholdGatewayFailure.connection),
    session: _administratorSession,
    onBack: _noOp,
  ),
  scenarios: [
    _Scenario(
      name: 'Default',
      run: (tester, _) async => tester.pumpAndSettle(),
    ),
  ],
);

final $Forbidden = _Story(
  args: _Args.fixed(
    gateway: _gateway(session: _memberSession),
    session: _memberSession,
    onBack: _noOp,
  ),
);

void _noOp() {}
