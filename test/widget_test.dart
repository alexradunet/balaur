import 'package:balaur/chat/data/chat_gateway.dart';
import 'package:balaur/chat/data/conversation_repository.dart';
import 'package:balaur/chat/domain/chat_message.dart';
import 'package:balaur/main.dart';
import 'package:balaur/settings/provider_settings.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('streams a chat response', (tester) async {
    final gateway = _FakeChatGateway(['Hello', ' there']);

    await tester.pumpWidget(
      BalaurApp(
        gateway: gateway,
        conversationRepository: InMemoryConversationRepository(),
        settingsStore: InMemoryProviderSettingsStore(_configuredSettings),
      ),
    );
    await tester.pumpAndSettle();

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
    await tester.pumpWidget(
      BalaurApp(
        gateway: _FakeChatGateway([]),
        conversationRepository: InMemoryConversationRepository(),
        settingsStore: InMemoryProviderSettingsStore(),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text('Configure a model provider to start a conversation.'),
      findsOneWidget,
    );
    expect(
      tester.widget<IconButton>(find.byKey(const Key('send-button'))).onPressed,
      isNull,
    );
  });

  testWidgets('navigates to settings and enables chat after saving', (
    tester,
  ) async {
    final settingsStore = InMemoryProviderSettingsStore();
    await tester.pumpWidget(
      BalaurApp(
        gateway: _FakeChatGateway([]),
        conversationRepository: InMemoryConversationRepository(),
        settingsStore: settingsStore,
      ),
    );
    await tester.pumpAndSettle();

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
      tester.widget<IconButton>(find.byKey(const Key('send-button'))).onPressed,
      isNotNull,
    );
  });

  testWidgets('uses compact navigation when width is narrow', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      BalaurApp(
        gateway: _FakeChatGateway([]),
        conversationRepository: InMemoryConversationRepository(),
        settingsStore: InMemoryProviderSettingsStore(_configuredSettings),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Navigation'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('SETTINGS'));
    await tester.pumpAndSettle();

    expect(find.text('Model provider'), findsOneWidget);
  });
}

const _configuredSettings = ProviderSettings(
  baseUrl: 'https://example.com/v1',
  apiKey: 'test-key',
  model: 'test-model',
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
