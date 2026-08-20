import 'package:balaur/design_system/components/balaur_core.dart';
import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_assets.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Displays an operating-system access decision.
class BalaurGuardianCard extends StatelessWidget {
  const BalaurGuardianCard({
    super.key,
    required this.title,
    required this.onAllowOnce,
    required this.onAllowAlways,
    required this.onDeny,
    this.kicker = 'OS access',
    this.detail,
    this.scope,
  });

  final String title;
  final String kicker;
  final String? detail;
  final String? scope;
  final VoidCallback onAllowOnce;
  final VoidCallback onAllowAlways;
  final VoidCallback onDeny;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      ornate: true,
      borderColor: colors.goldDeep,
      padding: const EdgeInsets.all(18),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              const Image(
                image: AssetImage(
                  'assets/design_system/icons/shield.png',
                  package: 'balaur',
                ),
                width: 24,
                height: 24,
              ),
              const SizedBox(width: 10),
              Text(
                kicker.toUpperCase(),
                style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: colors.goldInk,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1,
                ),
              ),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            title,
            style: Theme.of(context).textTheme.titleLarge
                ?.copyWith(color: colors.ink),
          ),
          if (detail case final detail?) ...[
            const SizedBox(height: 7),
            Text(
              detail,
              style: Theme.of(context).textTheme.bodySmall
                  ?.copyWith(color: colors.ink),
            ),
          ],
          if (scope case final scope?) ...[
            const SizedBox(height: 11),
            BalaurSurface(
              material: BalaurSurfaceMaterial.inset,
              hardShadow: false,
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
              child: SizedBox(
                width: double.infinity,
                child: Text(
                  scope,
                  style: Theme.of(context).textTheme.labelMedium
                      ?.copyWith(color: colors.chromeForeground),
                ),
              ),
            ),
          ],
          const SizedBox(height: 13),
          Wrap(
            spacing: 9,
            runSpacing: 9,
            children: [
              BalaurButton(
                label: 'Allow once',
                onPressed: onAllowOnce,
                size: BalaurButtonSize.small,
              ),
              BalaurButton(
                label: 'Always',
                onPressed: onAllowAlways,
                variant: BalaurButtonVariant.ghost,
                size: BalaurButtonSize.small,
              ),
              BalaurButton(
                label: 'Deny',
                onPressed: onDeny,
                variant: BalaurButtonVariant.ghost,
                size: BalaurButtonSize.small,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

/// Displays the older Conversation context that Balaur carries forward.
class BalaurRecapCard extends StatelessWidget {
  const BalaurRecapCard({
    super.key,
    required this.summary,
    this.points = const [],
    this.kicker = 'Recap',
    this.when,
  });

  final String summary;
  final List<String> points;
  final String kicker;
  final String? when;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      padding: const EdgeInsets.fromLTRB(17, 15, 17, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              Image(image: BalaurAssets.icon('orb'), width: 20, height: 20),
              const SizedBox(width: 9),
              Text(
                kicker.toUpperCase(),
                style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: colors.tealInk,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1,
                ),
              ),
              const Spacer(),
              if (when case final whenText?)
                Text(
                  whenText.toUpperCase(),
                  style: Theme.of(context).textTheme.labelSmall
                      ?.copyWith(color: colors.inkMuted),
                ),
            ],
          ),
          const SizedBox(height: 9),
          Text(
            summary,
            style: Theme.of(context).textTheme.bodyMedium
                ?.copyWith(color: colors.ink, height: 1.55),
          ),
          if (points.isNotEmpty) ...[
            const SizedBox(height: 11),
            for (final point in points)
              Padding(
                padding: const EdgeInsets.only(bottom: 6),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('▪', style: TextStyle(color: colors.tealInk)),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        point,
                        style: Theme.of(context).textTheme.bodySmall
                            ?.copyWith(color: colors.ink),
                      ),
                    ),
                  ],
                ),
              ),
          ],
        ],
      ),
    );
  }
}

/// Displays a nudge with spoken owner replies.
class BalaurNudgeBanner extends StatelessWidget {
  const BalaurNudgeBanner({
    super.key,
    required this.message,
    required this.onDone,
    required this.onSnooze,
    required this.onTomorrow,
    this.kicker = 'Nudge',
    this.when,
  });

  final String message;
  final String kicker;
  final String? when;
  final VoidCallback onDone;
  final VoidCallback onSnooze;
  final VoidCallback onTomorrow;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      borderColor: colors.goldDeep,
      padding: const EdgeInsets.fromLTRB(16, 13, 16, 14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              Image(image: BalaurAssets.icon('bell'), width: 20, height: 20),
              const SizedBox(width: 10),
              Text(
                kicker.toUpperCase(),
                style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: colors.goldInk,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const Spacer(),
              if (when case final whenText?)
                Text(
                  whenText.toUpperCase(),
                  style: Theme.of(context).textTheme.labelSmall
                      ?.copyWith(color: colors.inkMuted),
                ),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            message,
            style: Theme.of(context).textTheme.bodyMedium
                ?.copyWith(color: colors.ink),
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _SpokenReply(
                label: 'It is done.',
                hint: 'mark done',
                onPressed: onDone,
              ),
              _SpokenReply(
                label: 'At nightfall.',
                hint: 'snooze · 21:00',
                onPressed: onSnooze,
              ),
              _SpokenReply(
                label: 'Tomorrow, I swear it.',
                hint: 'snooze · tomorrow',
                onPressed: onTomorrow,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _SpokenReply extends StatelessWidget {
  const _SpokenReply({
    required this.label,
    required this.hint,
    required this.onPressed,
  });

  final String label;
  final String hint;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Material(
      color: colors.surface2,
      shape: BeveledRectangleBorder(
        side: BorderSide(color: colors.parchmentEdge, width: 2),
      ),
      child: InkWell(
        onTap: onPressed,
        child: ConstrainedBox(
          constraints: const BoxConstraints(minHeight: 48),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 11, vertical: 7),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.baseline,
              textBaseline: TextBaseline.alphabetic,
              children: [
                Text(
                  label,
                  style: Theme.of(context).textTheme.bodySmall
                      ?.copyWith(color: colors.ink),
                ),
                const SizedBox(width: 9),
                Text(
                  hint.toUpperCase(),
                  style: Theme.of(context).textTheme.labelSmall
                      ?.copyWith(color: colors.inkMuted, fontSize: 9.5),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
