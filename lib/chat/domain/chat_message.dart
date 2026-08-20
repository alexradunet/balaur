enum ChatRole { user, assistant }

enum ChatMessageStatus { complete, streaming, interrupted, failed }

class ChatMessage {
  const ChatMessage({
    required this.id,
    required this.role,
    required this.content,
    this.status = ChatMessageStatus.complete,
  });

  final String id;
  final ChatRole role;
  final String content;
  final ChatMessageStatus status;

  ChatMessage copyWith({String? content, ChatMessageStatus? status}) {
    return ChatMessage(
      id: id,
      role: role,
      content: content ?? this.content,
      status: status ?? this.status,
    );
  }
}
