import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_pairing_gate.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'household_pairing_gate.stories.g.dart';

const component = ComponentMeta(
  name: 'Household pairing',
  path: 'Screens/Household',
);
const meta = Meta(HouseholdPairingGate.new);

final $Unpaired = _Story(
  args: _Args.fixed(
    gateway: InMemoryHouseholdGateway(),
    pairedChild: const Center(child: Text('Paired application')),
  ),
  scenarios: [
    _Scenario(
      name: 'Default',
      run: (tester, _) async => tester.pumpAndSettle(),
    ),
  ],
);

final $Connecting = _Story(
  args: _Args.fixed(
    gateway: InMemoryHouseholdGateway(pauseRestore: true),
    pairedChild: const Center(child: Text('Paired application')),
  ),
);

final $Failed = _Story(
  args: _Args.fixed(
    gateway: InMemoryHouseholdGateway(
      restoreFailure: HouseholdGatewayFailure.connection,
    ),
    pairedChild: const Center(child: Text('Paired application')),
  ),
  scenarios: [
    _Scenario(
      name: 'Default',
      run: (tester, _) async => tester.pumpAndSettle(),
    ),
  ],
);

final $Paired = _Story(
  args: _Args.fixed(
    gateway: InMemoryHouseholdGateway(
      state: InMemoryHouseholdGatewayState(storedSession: _session),
    ),
    pairedChild: const Center(child: Text('Paired application')),
  ),
  scenarios: [
    _Scenario(
      name: 'Default',
      run: (tester, _) async => tester.pumpAndSettle(),
    ),
  ],
);

final _session = HouseholdSession(
  serverAddress: HouseholdServerAddress.parse('https://household.example.com'),
  member: const HouseholdMember(
    id: 'member-example',
    displayName: 'Alex',
    email: 'alex@example.com',
    role: HouseholdMemberRole.administrator,
  ),
);
