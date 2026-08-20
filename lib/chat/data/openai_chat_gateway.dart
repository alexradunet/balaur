import 'dart:async';

import 'package:balaur/chat/data/chat_gateway.dart';
import 'package:balaur/chat/domain/chat_message.dart' as domain;
import 'package:balaur/settings/provider_settings.dart';
import 'package:openai_dart/openai_dart.dart';

class OpenAiChatGateway implements ChatGateway {
  OpenAIClient? _activeClient;

  @override
  Stream<String> streamReply({
    required ProviderSettings settings,
    required List<domain.ChatMessage> messages,
  }) async* {
    final client = OpenAIClient(
      config: OpenAIConfig(
        authProvider: ApiKeyProvider(settings.apiKey.trim()),
        baseUrl: settings.normalizedBaseUrl,
        connectTimeout: const Duration(seconds: 20),
        timeout: const Duration(minutes: 2),
        retryPolicy: const RetryPolicy(maxRetries: 1),
      ),
    );
    _activeClient = client;

    try {
      final stream = client.chat.completions
          .createStream(
            ChatCompletionCreateRequest(
              model: settings.model.trim(),
              messages: messages
                  .where((message) => message.content.isNotEmpty)
                  .map(_toOpenAiMessage)
                  .toList(),
            ),
          )
          .textDeltas();

      await for (final delta in stream) {
        yield delta;
      }
    } on RateLimitException catch (error) {
      throw ChatGatewayException(
        error.retryAfter == null
            ? 'The provider rate limit was reached.'
            : 'The provider rate limit was reached. Retry after ${error.retryAfter}.',
      );
    } on ApiException catch (error) {
      throw ChatGatewayException(
        'The provider returned error ${error.statusCode}: ${error.message}',
      );
    } on OpenAIException catch (error) {
      throw ChatGatewayException('The provider request failed: $error');
    } on TimeoutException {
      throw const ChatGatewayException('The provider request timed out.');
    } finally {
      if (identical(_activeClient, client)) {
        _activeClient = null;
      }
      client.close();
    }
  }

  ChatMessage _toOpenAiMessage(domain.ChatMessage message) {
    return switch (message.role) {
      domain.ChatRole.user => ChatMessage.user(message.content),
      domain.ChatRole.assistant => ChatMessage.assistant(
        content: message.content,
      ),
    };
  }

  @override
  Future<void> cancel() async {
    _activeClient?.close();
    _activeClient = null;
  }

  @override
  void dispose() {
    _activeClient?.close();
    _activeClient = null;
  }
}
