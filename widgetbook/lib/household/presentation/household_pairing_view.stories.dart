import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_pairing_view.dart';
import 'package:balaur/household/presentation/household_pairing_view_model.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'household_pairing_view.stories.g.dart';

const component = ComponentMeta(
  name: 'Household invitation pairing',
  path: 'Screens/Household',
);
const meta = Meta(HouseholdPairingView.new);

final _payload = HouseholdInvitationPayload(
  serverAddress: HouseholdServerAddress.parse('https://household.example.com'),
  value: 'InvitationValue'.padRight(48, 'A'),
);

final $Manual = _Story(
  args: _Args.fixed(
    status: HouseholdPairingStatus.unpaired,
    onPair: _pair,
    onRedeemInvitation: _redeem,
    onSignOut: _signOut,
    initialMode: HouseholdPairingMode.invitation,
  ),
);

final $Scanned = _Story(
  args: _Args.fixed(
    status: HouseholdPairingStatus.unpaired,
    onPair: _pair,
    onRedeemInvitation: _redeem,
    onSignOut: _signOut,
    onScanInvitation: _scan,
    initialMode: HouseholdPairingMode.invitation,
  ),
  scenarios: [
    _Scenario(
      name: 'Scan invitation code',
      run: (tester, _) async {
        await tester.tap(find.byKey(const Key('scan-household-invitation')));
        await tester.pumpAndSettle();
      },
    ),
  ],
);

Future<void> _pair({
  required HouseholdServerAddress serverAddress,
  required String email,
  required String password,
}) async {}

Future<void> _redeem({
  required HouseholdInvitationPayload invitation,
  required String displayName,
  required String email,
  required String password,
}) async {}

Future<void> _signOut() async {}
Future<String?> _scan(BuildContext context) async => _payload.qrValue;
