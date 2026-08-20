import 'package:balaur/design_system/components/balaur_avatar.dart';
import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_assets.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Identifies the sender shown by a [BalaurMessageBubble].
enum BalaurMessageBubbleRole { householdMember, agent }

/// Identifies the visible response state.
enum BalaurMessageBubbleStatus { complete, streaming, stopped, failed }

/// Displays one framed role-playing game dialogue message.
class BalaurMessageBubble extends StatelessWidget {
  const BalaurMessageBubble({
    super.key,
    required this.content,
    required this.role,
    this.status = BalaurMessageBubbleStatus.complete,
    this.who,
    this.origin,
    this.avatar,
  });

  final String content;
  final BalaurMessageBubbleRole role;
  final BalaurMessageBubbleStatus status;
  final String? who;
  final String? origin;
  final ImageProvider? avatar;

  @override
  Widget build(BuildContext context) {
    final isHouseholdMember = role == BalaurMessageBubbleRole.householdMember;
    final colors = BalaurColors.of(context);
    return LayoutBuilder(
      builder: (context, constraints) {
        final portraitSize = constraints.maxWidth < 640 ? 64.0 : 96.0;
        final name = who ?? (isHouseholdMember ? 'You' : 'Balaur');
        final image =
            avatar ??
            (isHouseholdMember
                ? BalaurAssets.soulAvatar(5)
                : BalaurAssets.balaurAvatar(1));
        final portrait = _FramedPortrait(
          image: image,
          size: portraitSize,
          isHouseholdMember: isHouseholdMember,
          active: status == BalaurMessageBubbleStatus.streaming,
        );
        final panel = Flexible(
          child: ConstrainedBox(
            constraints: BoxConstraints(minHeight: portraitSize, maxWidth: 680),
            child: Stack(
              clipBehavior: Clip.none,
              children: [
                BalaurSurface(
                  borderColor: status == BalaurMessageBubbleStatus.streaming
                      ? colors.goldDeep
                      : colors.parchmentEdge,
                  padding: const EdgeInsets.fromLTRB(16, 18, 16, 13),
                  child: _MessageContent(content: content, status: status),
                ),
                Positioned(
                  top: -10,
                  left: isHouseholdMember ? null : 12,
                  right: isHouseholdMember ? 12 : null,
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 9,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: colors.surface2,
                      border: Border.all(color: colors.parchmentEdge, width: 2),
                    ),
                    child: Text(
                      [name, ?origin].join(' · ').toUpperCase(),
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.labelSmall?.copyWith(
                        color: isHouseholdMember
                            ? colors.indigoInk
                            : colors.goldInk,
                        fontWeight: FontWeight.w700,
                        letterSpacing: 0.7,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        );
        final row = Row(
          mainAxisAlignment: isHouseholdMember
              ? MainAxisAlignment.end
              : MainAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: isHouseholdMember
              ? [panel, const SizedBox(width: 10), portrait]
              : [portrait, const SizedBox(width: 10), panel],
        );
        return Align(
          alignment: isHouseholdMember
              ? Alignment.centerRight
              : Alignment.centerLeft,
          child: FractionallySizedBox(widthFactor: 0.92, child: row),
        );
      },
    );
  }
}

class _FramedPortrait extends StatelessWidget {
  const _FramedPortrait({
    required this.image,
    required this.size,
    required this.isHouseholdMember,
    required this.active,
  });

  final ImageProvider image;
  final double size;
  final bool isHouseholdMember;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      material: BalaurSurfaceMaterial.wood,
      borderColor: colors.outline,
      padding: const EdgeInsets.all(5),
      child: ColoredBox(
        color: const Color(0xff101314),
        child: Padding(
          padding: const EdgeInsets.all(2),
          child: BalaurAvatar(
            image: image,
            kind: isHouseholdMember
                ? BalaurAvatarKind.soul
                : BalaurAvatarKind.balaur,
            state: active ? BalaurAvatarState.thinking : BalaurAvatarState.idle,
            size: size - 14,
            mirrored: isHouseholdMember,
          ),
        ),
      ),
    );
  }
}

class _MessageContent extends StatelessWidget {
  const _MessageContent({required this.content, required this.status});

  final String content;
  final BalaurMessageBubbleStatus status;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (content.isNotEmpty)
          Semantics(
            label: content,
            readOnly: true,
            child: ExcludeSemantics(
              child: SelectableText(
                content,
                style: Theme.of(context).textTheme.bodyMedium
                    ?.copyWith(color: colors.ink, height: 1.55),
              ),
            ),
          ),
        if (status == BalaurMessageBubbleStatus.streaming) ...[
          if (content.isNotEmpty) const SizedBox(height: 6),
          Semantics(
            label: 'Agent response in progress',
            liveRegion: true,
            child: ExcludeSemantics(
              child: Text(
                'thinking…',
                style: Theme.of(context).textTheme.labelMedium
                    ?.copyWith(color: colors.tealInk),
              ),
            ),
          ),
        ],
        if (status == BalaurMessageBubbleStatus.stopped)
          _StatusLabel(label: 'Stopped', color: colors.inkMuted),
        if (status == BalaurMessageBubbleStatus.failed)
          _StatusLabel(label: 'Failed', color: colors.emberRed),
      ],
    );
  }
}

class _StatusLabel extends StatelessWidget {
  const _StatusLabel({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Text(
        label,
        style: Theme.of(context).textTheme.labelMedium
            ?.copyWith(color: color, fontWeight: FontWeight.w700),
      ),
    );
  }
}
