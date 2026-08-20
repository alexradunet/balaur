import 'package:balaur/chat/data/chat_gateway.dart';
import 'package:balaur/chat/data/conversation_repository.dart';
import 'package:balaur/chat/data/openai_chat_gateway.dart';
import 'package:balaur/chat/presentation/chat_screen.dart';
import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(
    BalaurApp(
      gateway: OpenAiChatGateway(),
      conversationRepository: InMemoryConversationRepository(),
      settingsStore: SecureProviderSettingsStore(const FlutterSecureStorage()),
    ),
  );
}

class BalaurApp extends StatelessWidget {
  const BalaurApp({
    super.key,
    required this.gateway,
    required this.conversationRepository,
    required this.settingsStore,
  });

  final ChatGateway gateway;
  final ConversationRepository conversationRepository;
  final ProviderSettingsStore settingsStore;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Balaur',
      debugShowCheckedModeBanner: false,
      theme: BalaurTheme.light(),
      darkTheme: BalaurTheme.dark(),
      themeMode: ThemeMode.system,
      home: ChatScreen(
        gateway: gateway,
        conversationRepository: conversationRepository,
        settingsStore: settingsStore,
      ),
    );
  }
}
