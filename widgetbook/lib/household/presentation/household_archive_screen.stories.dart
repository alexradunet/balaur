import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/household_archive.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_archive_screen.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'household_archive_screen.stories.g.dart';

const component = ComponentMeta(
  name: 'Household Archive',
  path: 'Screens/Household',
);
const meta = Meta(HouseholdArchiveScreen.new);

final _session = HouseholdSession(
  serverAddress: HouseholdServerAddress.parse('https://household.example.com'),
  member: const HouseholdMember(
    id: 'administrator-example',
    displayName: 'Alex',
    email: 'alex@example.com',
    role: HouseholdMemberRole.administrator,
  ),
);

InMemoryHouseholdGateway _gateway({
  HouseholdGatewayFailure? failure,
  bool pause = false,
}) {
  return InMemoryHouseholdGateway(
    state: InMemoryHouseholdGatewayState(storedSession: _session),
    archiveFailure: failure,
    pauseArchiveExport: pause,
  );
}

_Story _story(InMemoryHouseholdGateway gateway, {List<_Scenario>? scenarios}) {
  return _Story(
    args: _Args.fixed(
      gateway: gateway,
      session: _session,
      onBack: _noOp,
      saveArchive: _saveArchive,
    ),
    scenarios: scenarios ?? [_Scenario(name: 'Default')],
  );
}

final $Idle = _story(_gateway());
final $Exporting = _story(
  _gateway(pause: true),
  scenarios: [
    _Scenario(
      name: 'Export in progress',
      run: (tester, _) async {
        await tester.enterText(
          find.byKey(const Key('household-archive-password')),
          'correct-horse',
        );
        await tester.ensureVisible(
          find.byKey(const Key('export-household-archive')),
        );
        await tester.tap(find.byKey(const Key('export-household-archive')));
        await tester.pump();
      },
    ),
  ],
);
final $Success = _story(
  _gateway(),
  scenarios: [
    _Scenario(
      name: 'Export complete',
      run: (tester, _) async {
        await tester.enterText(
          find.byKey(const Key('household-archive-password')),
          'correct-horse',
        );
        await tester.ensureVisible(
          find.byKey(const Key('export-household-archive')),
        );
        await tester.tap(find.byKey(const Key('export-household-archive')));
        await tester.pumpAndSettle();
      },
    ),
  ],
);
final $Failed = _story(
  _gateway(failure: HouseholdGatewayFailure.server),
  scenarios: [
    _Scenario(
      name: 'Export failed',
      run: (tester, _) async {
        await tester.enterText(
          find.byKey(const Key('household-archive-password')),
          'correct-horse',
        );
        await tester.ensureVisible(
          find.byKey(const Key('export-household-archive')),
        );
        await tester.tap(find.byKey(const Key('export-household-archive')));
        await tester.pumpAndSettle();
      },
    ),
  ],
);

void _noOp() {}
Future<bool> _saveArchive(HouseholdArchive _) async => true;
