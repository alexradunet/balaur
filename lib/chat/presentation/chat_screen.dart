import 'package:balaur/chat/data/chat_gateway.dart';
import 'package:balaur/chat/data/conversation_repository.dart';
import 'package:balaur/chat/domain/chat_message.dart';
import 'package:balaur/chat/presentation/chat_controller.dart';
import 'package:balaur/chat/presentation/provider_settings_dialog.dart';
import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/material.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({
    super.key,
    required this.gateway,
    required this.conversationRepository,
    required this.settingsStore,
  });

  final ChatGateway gateway;
  final ConversationRepository conversationRepository;
  final ProviderSettingsStore settingsStore;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  late final ChatController _controller;
  final _composerController = TextEditingController();
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _controller = ChatController(
      gateway: widget.gateway,
      conversationRepository: widget.conversationRepository,
      settingsStore: widget.settingsStore,
    )..addListener(_scheduleScrollToEnd);
    _controller.initialize();
  }

  @override
  void dispose() {
    _controller
      ..removeListener(_scheduleScrollToEnd)
      ..dispose();
    _composerController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: _controller,
      builder: (context, _) {
        return Scaffold(
          appBar: AppBar(
            title: const Text('Balaur'),
            actions: [
              IconButton(
                key: const Key('provider-settings-button'),
                onPressed: _showProviderSettings,
                tooltip: 'Model provider',
                icon: const Icon(Icons.settings_outlined),
              ),
            ],
          ),
          body: SafeArea(
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 900),
                child: _controller.isReady
                    ? _buildChat(context)
                    : const Center(child: CircularProgressIndicator()),
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildChat(BuildContext context) {
    return Column(
      children: [
        if (!_controller.isConfigured) _buildSetupNotice(context),
        if (_controller.errorMessage case final error?)
          _buildErrorNotice(context, error),
        Expanded(
          child: _controller.messages.isEmpty
              ? const _EmptyConversation()
              : ListView.separated(
                  key: const Key('message-list'),
                  controller: _scrollController,
                  padding: const EdgeInsets.fromLTRB(16, 20, 16, 12),
                  itemCount: _controller.messages.length,
                  separatorBuilder: (_, _) => const SizedBox(height: 12),
                  itemBuilder: (context, index) {
                    final message = _controller.messages[index];
                    return BalaurMessageBubble(
                      content: message.content,
                      role: message.role == ChatRole.user
                          ? BalaurMessageBubbleRole.householdMember
                          : BalaurMessageBubbleRole.agent,
                      status: switch (message.status) {
                        ChatMessageStatus.complete =>
                          BalaurMessageBubbleStatus.complete,
                        ChatMessageStatus.streaming =>
                          BalaurMessageBubbleStatus.streaming,
                        ChatMessageStatus.interrupted =>
                          BalaurMessageBubbleStatus.stopped,
                        ChatMessageStatus.failed =>
                          BalaurMessageBubbleStatus.failed,
                      },
                    );
                  },
                ),
        ),
        _Composer(
          controller: _composerController,
          isEnabled: _controller.isConfigured && !_controller.isStreaming,
          isStreaming: _controller.isStreaming,
          onSend: _sendMessage,
          onStop: _controller.stop,
        ),
      ],
    );
  }

  Widget _buildSetupNotice(BuildContext context) {
    return MaterialBanner(
      content: const Text(
        'Configure a model provider to start a conversation.',
      ),
      actions: [
        TextButton(
          onPressed: _showProviderSettings,
          child: const Text('Configure'),
        ),
      ],
    );
  }

  Widget _buildErrorNotice(BuildContext context, String error) {
    return MaterialBanner(
      backgroundColor: Theme.of(context).colorScheme.errorContainer,
      content: Text(error),
      actions: [
        TextButton(
          onPressed: _controller.clearError,
          child: const Text('Dismiss'),
        ),
      ],
    );
  }

  Future<void> _showProviderSettings() async {
    final settings = await ProviderSettingsDialog.show(
      context,
      _controller.settings,
    );
    if (settings != null) {
      await _controller.saveSettings(settings);
    }
  }

  Future<void> _sendMessage() async {
    final content = _composerController.text;
    if (content.trim().isEmpty) {
      return;
    }
    _composerController.clear();
    await _controller.sendMessage(content);
  }

  void _scheduleScrollToEnd() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) {
        return;
      }
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 150),
        curve: Curves.easeOut,
      );
    });
  }
}

class _EmptyConversation extends StatelessWidget {
  const _EmptyConversation();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.forum_outlined,
              size: 48,
              color: Theme.of(context).colorScheme.primary,
            ),
            const SizedBox(height: 16),
            Text(
              'Start a conversation',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            const Text('Ask Balaur for help with your household.'),
          ],
        ),
      ),
    );
  }
}

class _Composer extends StatelessWidget {
  const _Composer({
    required this.controller,
    required this.isEnabled,
    required this.isStreaming,
    required this.onSend,
    required this.onStop,
  });

  final TextEditingController controller;
  final bool isEnabled;
  final bool isStreaming;
  final Future<void> Function() onSend;
  final Future<void> Function() onStop;

  @override
  Widget build(BuildContext context) {
    return Material(
      elevation: 8,
      color: Theme.of(context).colorScheme.surface,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            Expanded(
              child: TextField(
                key: const Key('chat-composer'),
                controller: controller,
                enabled: isEnabled,
                minLines: 1,
                maxLines: 5,
                textCapitalization: TextCapitalization.sentences,
                decoration: const InputDecoration(
                  hintText: 'Message Balaur',
                  border: OutlineInputBorder(),
                ),
              ),
            ),
            const SizedBox(width: 8),
            if (isStreaming)
              IconButton.filled(
                key: const Key('stop-button'),
                onPressed: onStop,
                tooltip: 'Stop response',
                icon: const Icon(Icons.stop),
              )
            else
              IconButton.filled(
                key: const Key('send-button'),
                onPressed: isEnabled ? onSend : null,
                tooltip: 'Send message',
                icon: const Icon(Icons.arrow_upward),
              ),
          ],
        ),
      ),
    );
  }
}
