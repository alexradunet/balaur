import 'package:balaur/chat/domain/chat_message.dart';

abstract interface class ConversationRepository {
  Future<List<ChatMessage>> loadMessages();

  Future<void> saveMessages(List<ChatMessage> messages);
}

class InMemoryConversationRepository implements ConversationRepository {
  List<ChatMessage> _messages = [];

  @override
  Future<List<ChatMessage>> loadMessages() async =>
      List.unmodifiable(_messages);

  @override
  Future<void> saveMessages(List<ChatMessage> messages) async {
    _messages = List.of(messages);
  }
}
