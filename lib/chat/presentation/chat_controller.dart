// ignore_for_file: prefer_initializing_formals

import 'package:balaur/chat/data/chat_gateway.dart';
import 'package:balaur/chat/data/conversation_repository.dart';
import 'package:balaur/chat/domain/chat_message.dart';
import 'package:balaur/settings/provider_settings.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/foundation.dart';

class ChatController extends ChangeNotifier {
  ChatController({
    required ChatGateway gateway,
    required ConversationRepository conversationRepository,
    required ProviderSettingsStore settingsStore,
  }) : _gateway = gateway,
       _conversationRepository = conversationRepository,
       _settingsStore = settingsStore;

  final ChatGateway _gateway;
  final ConversationRepository _conversationRepository;
  final ProviderSettingsStore _settingsStore;

  List<ChatMessage> _messages = [];
  ProviderSettings _settings = const ProviderSettings.empty();
  String? _errorMessage;
  bool _isReady = false;
  bool _isStreaming = false;
  bool _stopRequested = false;
  bool _disposed = false;
  int _messageSequence = 0;

  List<ChatMessage> get messages => List.unmodifiable(_messages);
  ProviderSettings get settings => _settings;
  String? get errorMessage => _errorMessage;
  bool get isReady => _isReady;
  bool get isStreaming => _isStreaming;
  bool get isConfigured => _settings.isConfigured;

  Future<void> initialize() async {
    try {
      final results = await Future.wait<Object>([
        _settingsStore.load(),
        _conversationRepository.loadMessages(),
      ]);
      _settings = results[0] as ProviderSettings;
      _messages = results[1] as List<ChatMessage>;
    } on Object catch (error) {
      _errorMessage = 'Balaur could not load its settings: $error';
    } finally {
      _isReady = true;
      _notifyListeners();
    }
  }

  Future<void> saveSettings(ProviderSettings settings) async {
    try {
      await _settingsStore.save(settings);
      _settings = settings;
      _errorMessage = null;
    } on Object catch (error) {
      _errorMessage = 'Balaur could not save the provider settings: $error';
    }
    _notifyListeners();
  }

  Future<void> sendMessage(String input) async {
    final content = input.trim();
    if (content.isEmpty || _isStreaming) {
      return;
    }
    if (!_settings.isConfigured) {
      _errorMessage = 'Configure the model provider before you send a message.';
      _notifyListeners();
      return;
    }

    _errorMessage = null;
    _stopRequested = false;
    _isStreaming = true;

    final userMessage = ChatMessage(
      id: _nextMessageId(),
      role: ChatRole.user,
      content: content,
    );
    final requestMessages = [..._messages, userMessage];
    final assistantMessage = ChatMessage(
      id: _nextMessageId(),
      role: ChatRole.assistant,
      content: '',
      status: ChatMessageStatus.streaming,
    );
    _messages = [...requestMessages, assistantMessage];
    await _conversationRepository.saveMessages(_messages);
    _notifyListeners();

    var failed = false;
    try {
      await for (final delta in _gateway.streamReply(
        settings: _settings,
        messages: requestMessages,
      )) {
        _replaceMessage(
          assistantMessage.id,
          (message) => message.copyWith(content: message.content + delta),
        );
        _notifyListeners();
      }
    } on ChatGatewayException catch (error) {
      failed = true;
      if (!_stopRequested) {
        _errorMessage = error.message;
      }
    } on Object catch (error) {
      failed = true;
      if (!_stopRequested) {
        _errorMessage = 'The provider request failed: $error';
      }
    } finally {
      final current = _messageById(assistantMessage.id);
      if (current != null && current.content.isEmpty) {
        _messages = _messages
            .where((message) => message.id != assistantMessage.id)
            .toList();
        if (!failed && !_stopRequested) {
          _errorMessage = 'The provider returned an empty response.';
        }
      } else if (current != null) {
        _replaceMessage(
          assistantMessage.id,
          (message) => message.copyWith(
            status: failed || _stopRequested
                ? ChatMessageStatus.interrupted
                : ChatMessageStatus.complete,
          ),
        );
      }
      _isStreaming = false;
      await _conversationRepository.saveMessages(_messages);
      _notifyListeners();
    }
  }

  Future<void> stop() async {
    if (!_isStreaming) {
      return;
    }
    _stopRequested = true;
    await _gateway.cancel();
  }

  void clearError() {
    _errorMessage = null;
    _notifyListeners();
  }

  String _nextMessageId() {
    _messageSequence += 1;
    return '${DateTime.now().microsecondsSinceEpoch}-$_messageSequence';
  }

  ChatMessage? _messageById(String id) {
    for (final message in _messages) {
      if (message.id == id) {
        return message;
      }
    }
    return null;
  }

  void _replaceMessage(
    String id,
    ChatMessage Function(ChatMessage message) replace,
  ) {
    _messages = [
      for (final message in _messages)
        if (message.id == id) replace(message) else message,
    ];
  }

  void _notifyListeners() {
    if (!_disposed) {
      notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _gateway.dispose();
    super.dispose();
  }
}
