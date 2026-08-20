import 'package:balaur/design_system/components/balaur_core.dart';
import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Identifies a knowledge record type.
enum BalaurKnowledgeKind { memory, preference, person, project, fact, skill }

/// Identifies a knowledge record state.
enum BalaurKnowledgeStatus { proposed, active, archived }

/// Displays a knowledge record through its full lifecycle.
class BalaurKnowledgeCard extends StatelessWidget {
  const BalaurKnowledgeCard({
    super.key,
    required this.title,
    this.kind = BalaurKnowledgeKind.memory,
    this.status = BalaurKnowledgeStatus.active,
    this.body,
    this.whenToUse,
    this.importance,
    this.usedCount,
    this.onApprove,
    this.onDismiss,
    this.onArchive,
    this.onRestore,
  });

  final String title;
  final BalaurKnowledgeKind kind;
  final BalaurKnowledgeStatus status;
  final String? body;
  final String? whenToUse;
  final int? importance;
  final int? usedCount;
  final VoidCallback? onApprove;
  final VoidCallback? onDismiss;
  final VoidCallback? onArchive;
  final VoidCallback? onRestore;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final proposed = status == BalaurKnowledgeStatus.proposed;
    final archived = status == BalaurKnowledgeStatus.archived;
    return Opacity(
      opacity: archived ? 0.72 : 1,
      child: BalaurSurface(
        ornate: proposed,
        dashed: archived,
        hardShadow: !archived,
        borderColor: proposed ? colors.goldDeep : colors.parchmentEdge,
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                Text(
                  '▪ ${kind.name}'.toUpperCase(),
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: colors.tealInk,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 0.8,
                  ),
                ),
                const Spacer(),
                if (importance case final importance?)
                  BalaurPips(level: importance),
              ],
            ),
            const SizedBox(height: 9),
            Text(
              title,
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                color: archived ? colors.inkMuted : colors.ink,
                decoration: archived ? TextDecoration.lineThrough : null,
              ),
            ),
            if (body case final body?) ...[
              const SizedBox(height: 7),
              Text(
                body,
                style: Theme.of(context).textTheme.bodySmall
                    ?.copyWith(color: colors.ink, height: 1.5),
              ),
            ],
            if (whenToUse case final use?) ...[
              const SizedBox(height: 8),
              Text(
                'recall: $use',
                style: Theme.of(context).textTheme.labelMedium
                    ?.copyWith(color: colors.inkMuted),
              ),
            ],
            const SizedBox(height: 14),
            _KnowledgeActions(
              status: status,
              usedCount: usedCount,
              onApprove: onApprove,
              onDismiss: onDismiss,
              onArchive: onArchive,
              onRestore: onRestore,
            ),
          ],
        ),
      ),
    );
  }
}

class _KnowledgeActions extends StatelessWidget {
  const _KnowledgeActions({
    required this.status,
    required this.usedCount,
    required this.onApprove,
    required this.onDismiss,
    required this.onArchive,
    required this.onRestore,
  });

  final BalaurKnowledgeStatus status;
  final int? usedCount;
  final VoidCallback? onApprove;
  final VoidCallback? onDismiss;
  final VoidCallback? onArchive;
  final VoidCallback? onRestore;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: switch (status) {
        BalaurKnowledgeStatus.proposed => [
          BalaurButton(
            label: 'Approve',
            onPressed: onApprove,
            size: BalaurButtonSize.small,
          ),
          BalaurButton(
            label: 'Dismiss',
            onPressed: onDismiss,
            variant: BalaurButtonVariant.ghost,
            size: BalaurButtonSize.small,
          ),
        ],
        BalaurKnowledgeStatus.active => [
          BalaurButton(
            label: 'Archive',
            onPressed: onArchive,
            variant: BalaurButtonVariant.ghost,
            size: BalaurButtonSize.small,
          ),
          if (usedCount case final count?)
            Text(
              'used ×$count',
              style: Theme.of(context).textTheme.labelMedium
                  ?.copyWith(color: colors.inkMuted),
            ),
        ],
        BalaurKnowledgeStatus.archived => [
          BalaurButton(
            label: 'Restore',
            onPressed: onRestore,
            variant: BalaurButtonVariant.ghost,
            size: BalaurButtonSize.small,
          ),
        ],
      },
    );
  }
}

/// Identifies a task state.
enum BalaurTaskStatus { open, done, snoozed, dropped }

/// Displays an operational task in chat or a task list.
class BalaurTaskCard extends StatelessWidget {
  const BalaurTaskCard({
    super.key,
    required this.title,
    this.status = BalaurTaskStatus.open,
    this.dueLine,
    this.overdue = false,
    this.recurrence,
    this.notes,
    this.onDone,
    this.onSnooze,
    this.onDrop,
  });

  final String title;
  final BalaurTaskStatus status;
  final String? dueLine;
  final bool overdue;
  final String? recurrence;
  final String? notes;
  final VoidCallback? onDone;
  final VoidCallback? onSnooze;
  final VoidCallback? onDrop;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final closed = status != BalaurTaskStatus.open;
    final dropped = status == BalaurTaskStatus.dropped;
    return Opacity(
      opacity: dropped ? 0.8 : 1,
      child: BalaurSurface(
        dashed: dropped,
        hardShadow: !dropped,
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(
                  '▪ TASK',
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: colors.tealInk,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const Spacer(),
                if (recurrence case final recurrence?)
                  BalaurTag(label: recurrence),
              ],
            ),
            const SizedBox(height: 9),
            Text(
              title,
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                color: closed ? colors.inkMuted : colors.ink,
                decoration: closed ? TextDecoration.lineThrough : null,
              ),
            ),
            if (dueLine case final due?) ...[
              const SizedBox(height: 5),
              Text(
                due,
                style: Theme.of(context).textTheme.labelMedium?.copyWith(
                  color: overdue ? colors.emberDeep : colors.inkMuted,
                  fontWeight: overdue ? FontWeight.w700 : null,
                ),
              ),
            ],
            if (notes case final notes?) ...[
              const SizedBox(height: 8),
              ExpansionTile(
                tilePadding: EdgeInsets.zero,
                childrenPadding: EdgeInsets.zero,
                title: Text(
                  'NOTES',
                  style: Theme.of(context).textTheme.labelMedium
                      ?.copyWith(color: colors.inkMuted),
                ),
                children: [
                  Align(
                    alignment: Alignment.centerLeft,
                    child: Text(
                      notes,
                      style: Theme.of(context).textTheme.bodySmall
                          ?.copyWith(color: colors.ink),
                    ),
                  ),
                ],
              ),
            ],
            const SizedBox(height: 14),
            if (status == BalaurTaskStatus.open)
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  BalaurButton(
                    label: 'Done',
                    onPressed: onDone,
                    size: BalaurButtonSize.small,
                  ),
                  BalaurButton(
                    label: 'Snooze',
                    onPressed: onSnooze,
                    variant: BalaurButtonVariant.ghost,
                    size: BalaurButtonSize.small,
                  ),
                  BalaurButton(
                    label: 'Drop',
                    onPressed: onDrop,
                    variant: BalaurButtonVariant.ghost,
                    size: BalaurButtonSize.small,
                  ),
                ],
              )
            else
              Text(
                status.name.toUpperCase(),
                style: Theme.of(context).textTheme.labelMedium
                    ?.copyWith(color: colors.inkMuted),
              ),
          ],
        ),
      ),
    );
  }
}
