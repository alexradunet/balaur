import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_account_screen.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'household_account_screen.stories.g.dart';

const component = ComponentMeta(
  name: 'Household account',
  path: 'Screens/Household',
);
const meta = Meta(HouseholdAccountScreen.new);

final $Administrator = _Story(
  args: _Args.fixed(
    session: _session,
    onSignOut: _signOut,
    onManageInvitations: _manageInvitations,
    onManageCalendarConnection: _manageCalendarConnection,
    onExportArchive: _exportArchive,
  ),
);

final $Member = _Story(
  args: _Args.fixed(session: _memberSession, onSignOut: _signOut),
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

final _memberSession = HouseholdSession(
  serverAddress: HouseholdServerAddress.parse('https://household.example.com'),
  member: const HouseholdMember(
    id: 'member-second',
    displayName: 'Mara',
    email: 'mara@example.com',
    role: HouseholdMemberRole.member,
  ),
);

Future<void> _signOut() async {}
void _manageInvitations() {}
void _manageCalendarConnection() {}
void _exportArchive() {}
