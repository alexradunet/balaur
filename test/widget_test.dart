import 'package:balaur/chat/data/chat_gateway.dart';
import 'package:balaur/chat/data/conversation_repository.dart';
import 'package:balaur/chat/domain/chat_message.dart';
import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/in_memory_household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/main.dart';
import 'package:balaur/settings/provider_settings.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('streams a chat response', (tester) async {
    final gateway = _FakeChatGateway(['Hello', ' there']);

    await _pumpApp(tester, gateway: gateway);
    await _enterPairedApplication(tester);

    await tester.enterText(
      find.byKey(const Key('chat-composer')),
      'Hello Balaur',
    );
    await tester.tap(find.byKey(const Key('send-button')));
    await tester.pumpAndSettle();

    expect(find.text('Hello Balaur'), findsOneWidget);
    expect(find.text('Hello there'), findsOneWidget);
    expect(gateway.requests, hasLength(1));
    expect(gateway.requests.single.last.content, 'Hello Balaur');
  });

  testWidgets('asks for provider settings before chat', (tester) async {
    await _pumpApp(
      tester,
      gateway: _FakeChatGateway([]),
      settingsStore: InMemoryProviderSettingsStore(),
    );
    await _enterPairedApplication(tester);

    expect(
      find.text('Configure a model provider to start a conversation.'),
      findsOneWidget,
    );
    expect(
      tester
          .widget<BalaurButton>(find.byKey(const Key('send-button')))
          .onPressed,
      isNull,
    );
  });

  testWidgets('navigates to settings and enables chat after saving', (
    tester,
  ) async {
    final settingsStore = InMemoryProviderSettingsStore();
    await _pumpApp(
      tester,
      gateway: _FakeChatGateway([]),
      settingsStore: settingsStore,
    );
    await _enterPairedApplication(tester);

    await tester.tap(find.text('SETTINGS'));
    await tester.pumpAndSettle();

    expect(find.text('Model provider'), findsOneWidget);
    await tester.enterText(
      find.byKey(const Key('provider-api-key')),
      'test-key',
    );
    await tester.enterText(
      find.byKey(const Key('provider-model')),
      'test-model',
    );
    await tester.tap(find.byKey(const Key('save-provider-settings')));
    await tester.pumpAndSettle();

    expect(find.text('Model provider settings saved.'), findsOneWidget);

    await tester.tap(find.text('CHAT'));
    await tester.pumpAndSettle();

    expect(
      tester
          .widget<BalaurButton>(find.byKey(const Key('send-button')))
          .onPressed,
      isNotNull,
    );
  });

  testWidgets('pairs the first Household Administrator', (tester) async {
    final householdGateway = InMemoryHouseholdGateway(
      accounts: const [_householdAccount],
    );
    await _pumpApp(
      tester,
      gateway: _FakeChatGateway([]),
      householdGateway: householdGateway,
    );

    expect(find.text('Pair this device'), findsWidgets);
    await tester.enterText(
      find.byKey(const Key('household-server-address')),
      'https://household.example.com',
    );
    await tester.enterText(
      find.byKey(const Key('household-email')),
      _householdAccount.email,
    );
    await tester.enterText(
      find.byKey(const Key('household-password')),
      _householdAccount.password,
    );
    await tester.tap(find.byKey(const Key('pair-household')));
    await tester.pumpAndSettle();

    expect(find.text('Household paired'), findsOneWidget);
    await _enterPairedApplication(tester);

    expect(find.byKey(const Key('chat-composer')), findsOneWidget);
  });

  testWidgets('opens the Household Calendar destination', (tester) async {
    final state = InMemoryHouseholdGatewayState(
      storedSession: _session,
      calendarConnection: CalendarConnection(
        provider: 'google',
        status: CalendarConnectionStatus.connected,
        householdTimeZone: 'UTC',
        selectedCalendar: const CalendarSourceCalendar(
          id: 'family@example.com',
          name: 'Family',
          colorHex: '#4285f4',
        ),
      ),
    );
    await _pumpApp(
      tester,
      gateway: _FakeChatGateway([]),
      householdGateway: InMemoryHouseholdGateway(state: state),
    );
    await _enterPairedApplication(tester);

    await tester.tap(find.text('CALENDAR'));
    await tester.pumpAndSettle();

    expect(find.text('Household Calendar'), findsOneWidget);
    expect(find.byKey(const Key('calendar-month-panel')), findsOneWidget);
    expect(find.byKey(const Key('calendar-agenda-panel')), findsOneWidget);
  });

  testWidgets('opens Household Invitation management', (tester) async {
    final state = InMemoryHouseholdGatewayState(storedSession: _session);
    await _pumpApp(
      tester,
      gateway: _FakeChatGateway([]),
      householdGateway: InMemoryHouseholdGateway(state: state),
    );
    await _enterPairedApplication(tester);

    await tester.tap(find.text('HOUSEHOLD'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('manage-household-invitations')));
    await tester.pumpAndSettle();

    expect(find.text('Household Invitations'), findsOneWidget);
    expect(
      find.byKey(const Key('create-household-invitation')),
      findsOneWidget,
    );
  });

  testWidgets('opens Calendar Connection management', (tester) async {
    final state = InMemoryHouseholdGatewayState(storedSession: _session);
    await _pumpApp(
      tester,
      gateway: _FakeChatGateway([]),
      householdGateway: InMemoryHouseholdGateway(state: state),
    );
    await _enterPairedApplication(tester);

    await tester.tap(find.text('HOUSEHOLD'));
    await tester.pumpAndSettle();
    final manageCalendar = find.byKey(
      const Key('manage-household-calendar-connection'),
    );
    await tester.ensureVisible(manageCalendar);
    await tester.tap(manageCalendar);
    await tester.pumpAndSettle();

    expect(find.text('Calendar Connection'), findsOneWidget);
    expect(find.byKey(const Key('connect-google-calendar')), findsOneWidget);
  });

  testWidgets('opens Household Archive export', (tester) async {
    final state = InMemoryHouseholdGatewayState(storedSession: _session);
    await _pumpApp(
      tester,
      gateway: _FakeChatGateway([]),
      householdGateway: InMemoryHouseholdGateway(state: state),
    );
    await _enterPairedApplication(tester);

    await tester.tap(find.text('HOUSEHOLD'));
    await tester.pumpAndSettle();
    final export = find.byKey(const Key('export-household-archive'));
    await tester.ensureVisible(export);
    await tester.tap(export);
    await tester.pumpAndSettle();

    expect(find.text('Household Archive'), findsOneWidget);
    expect(find.byKey(const Key('household-archive-password')), findsOneWidget);
  });

  testWidgets('signs out from the Household account', (tester) async {
    final state = InMemoryHouseholdGatewayState(storedSession: _session);
    await _pumpApp(
      tester,
      gateway: _FakeChatGateway([]),
      householdGateway: InMemoryHouseholdGateway(state: state),
    );
    await _enterPairedApplication(tester);

    await tester.tap(find.text('HOUSEHOLD'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('sign-out-household')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('pair-household')), findsOneWidget);
    expect(state.storedSession, isNull);
  });

  testWidgets('uses compact navigation when width is narrow', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await _pumpApp(tester, gateway: _FakeChatGateway([]));
    await _enterPairedApplication(tester);

    await tester.tap(find.byTooltip('Navigation'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('SETTINGS'));
    await tester.pumpAndSettle();

    expect(find.text('Model provider'), findsOneWidget);
  });
}

Future<void> _pumpApp(
  WidgetTester tester, {
  required ChatGateway gateway,
  ProviderSettingsStore? settingsStore,
  HouseholdGateway? householdGateway,
}) async {
  await tester.pumpWidget(
    BalaurApp(
      gateway: gateway,
      conversationRepository: InMemoryConversationRepository(),
      settingsStore:
          settingsStore ?? InMemoryProviderSettingsStore(_configuredSettings),
      householdGateway: householdGateway ?? _pairedHouseholdGateway(),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _enterPairedApplication(WidgetTester tester) async {
  final enterButton = find.byKey(const Key('enter-paired-application'));
  expect(enterButton, findsOneWidget);
  await tester.ensureVisible(enterButton);
  await tester.pumpAndSettle();
  await tester.tap(enterButton);
  await tester.pumpAndSettle();
}

InMemoryHouseholdGateway _pairedHouseholdGateway() {
  return InMemoryHouseholdGateway(
    state: InMemoryHouseholdGatewayState(storedSession: _session),
  );
}

const _configuredSettings = ProviderSettings(
  baseUrl: 'https://example.com/v1',
  apiKey: 'test-key',
  model: 'test-model',
);

final _session = HouseholdSession(
  serverAddress: HouseholdServerAddress.parse('https://household.example.com'),
  member: _householdMember,
);

const _householdMember = HouseholdMember(
  id: 'member-example',
  displayName: 'Alex',
  email: 'alex@example.com',
  role: HouseholdMemberRole.administrator,
);

const _householdAccount = InMemoryHouseholdAccount(
  email: 'alex@example.com',
  password: 'correct-horse',
  member: _householdMember,
);

class _FakeChatGateway implements ChatGateway {
  _FakeChatGateway(this.deltas);

  final List<String> deltas;
  final List<List<ChatMessage>> requests = [];

  @override
  Stream<String> streamReply({
    required ProviderSettings settings,
    required List<ChatMessage> messages,
  }) async* {
    requests.add(List.of(messages));
    for (final delta in deltas) {
      yield delta;
    }
  }

  @override
  Future<void> cancel() async {}

  @override
  void dispose() {}
}
