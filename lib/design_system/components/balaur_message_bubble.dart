import 'package:flutter/material.dart';

/// Identifies the sender shown by a [BalaurMessageBubble].
enum BalaurMessageBubbleRole { householdMember, agent }

/// Identifies the visible response state.
enum BalaurMessageBubbleStatus { complete, streaming, stopped, failed }

/// Displays one Conversation message from a Household Member or an agent.
class BalaurMessageBubble extends StatelessWidget {
  const BalaurMessageBubble({
    super.key,
    required this.content,
    required this.role,
    this.status = BalaurMessageBubbleStatus.complete,
  });

  final String content;
  final BalaurMessageBubbleRole role;
  final BalaurMessageBubbleStatus status;

  @override
  Widget build(BuildContext context) {
    final isHouseholdMember = role == BalaurMessageBubbleRole.householdMember;
    final colors = Theme.of(context).colorScheme;

    return Align(
      alignment: isHouseholdMember
          ? Alignment.centerRight
          : Alignment.centerLeft,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 680),
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: isHouseholdMember
                ? colors.primaryContainer
                : colors.surfaceContainer,
            borderRadius: BorderRadius.circular(18),
          ),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (content.isNotEmpty)
                  Semantics(
                    label: content,
                    readOnly: true,
                    child: ExcludeSemantics(child: SelectableText(content)),
                  ),
                if (status == BalaurMessageBubbleStatus.streaming)
                  const Padding(
                    padding: EdgeInsets.only(top: 4),
                    child: SizedBox.square(
                      dimension: 14,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        semanticsLabel: 'Agent response in progress',
                      ),
                    ),
                  ),
                if (status == BalaurMessageBubbleStatus.stopped)
                  Padding(
                    padding: const EdgeInsets.only(top: 6),
                    child: Text(
                      'Stopped',
                      style: Theme.of(context).textTheme.labelMedium?.copyWith(
                        color: colors.onSurface,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                if (status == BalaurMessageBubbleStatus.failed)
                  Padding(
                    padding: const EdgeInsets.only(top: 6),
                    child: Text(
                      'Failed',
                      style: Theme.of(context).textTheme.labelSmall
                          ?.copyWith(color: colors.error),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
