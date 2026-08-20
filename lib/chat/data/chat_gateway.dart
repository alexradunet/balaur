import 'package:balaur/chat/domain/chat_message.dart';
import 'package:balaur/settings/provider_settings.dart';

abstract interface class ChatGateway {
  Stream<String> streamReply({
    required ProviderSettings settings,
    required List<ChatMessage> messages,
  });

  Future<void> cancel();

  void dispose();
}

class ChatGatewayException implements Exception {
  const ChatGatewayException(this.message);

  final String message;

  @override
  String toString() => message;
}
