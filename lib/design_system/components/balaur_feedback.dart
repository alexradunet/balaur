import 'package:balaur/design_system/components/balaur_core.dart';
import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_assets.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Identifies a short feedback tone.
enum BalaurToastTone { success, info, warning }

/// Displays a short parchment status message.
class BalaurToast extends StatelessWidget {
  const BalaurToast({
    super.key,
    required this.message,
    this.tone = BalaurToastTone.info,
  });

  final String message;
  final BalaurToastTone tone;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final icon = switch (tone) {
      BalaurToastTone.success => 'check',
      BalaurToastTone.info => 'quill',
      BalaurToastTone.warning => 'shield',
    };
    final edge = switch (tone) {
      BalaurToastTone.success => colors.goodInk,
      BalaurToastTone.info => colors.goldDeep,
      BalaurToastTone.warning => colors.emberDeep,
    };
    return Semantics(
      liveRegion: true,
      child: BalaurSurface(
        borderColor: edge,
        padding: const EdgeInsets.symmetric(horizontal: 15, vertical: 11),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Image(image: BalaurAssets.icon(icon), width: 20, height: 20),
            const SizedBox(width: 11),
            Flexible(
              child: Text(
                message,
                style: Theme.of(context).textTheme.bodySmall
                    ?.copyWith(color: colors.ink),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Identifies a persistent alert tone.
enum BalaurAlertTone { info, warning, danger }

/// Displays a persistent callout with a labelled status color.
class BalaurAlert extends StatelessWidget {
  const BalaurAlert({
    super.key,
    required this.message,
    this.title,
    this.tone = BalaurAlertTone.info,
    this.iconName,
  });

  final String message;
  final String? title;
  final BalaurAlertTone tone;
  final String? iconName;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final edge = switch (tone) {
      BalaurAlertTone.info => colors.goldDeep,
      BalaurAlertTone.warning => colors.emberDeep,
      BalaurAlertTone.danger => colors.emberRed,
    };
    final icon =
        iconName ??
        switch (tone) {
          BalaurAlertTone.info => 'orb',
          BalaurAlertTone.warning => 'shield',
          BalaurAlertTone.danger => 'flame',
        };
    return Semantics(
      liveRegion: tone != BalaurAlertTone.info,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: IntrinsicHeight(
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              ColoredBox(color: edge, child: const SizedBox(width: 6)),
              Flexible(
                child: BalaurSurface(
                  borderColor: colors.parchmentEdge,
                  padding: const EdgeInsets.fromLTRB(14, 13, 16, 14),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Image(
                        image: BalaurAssets.icon(icon),
                        width: 22,
                        height: 22,
                      ),
                      const SizedBox(width: 12),
                      Flexible(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            if (title case final title?) ...[
                              Text(
                                title.toUpperCase(),
                                style: Theme.of(context).textTheme.labelMedium
                                    ?.copyWith(
                                      color: edge,
                                      fontWeight: FontWeight.w700,
                                      letterSpacing: 0.7,
                                    ),
                              ),
                              const SizedBox(height: 4),
                            ],
                            Text(
                              message,
                              style: Theme.of(context).textTheme.bodySmall
                                  ?.copyWith(color: colors.ink),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Contains one dialog action.
@immutable
final class BalaurDialogAction {
  const BalaurDialogAction({
    required this.label,
    required this.onPressed,
    this.variant = BalaurButtonVariant.ghost,
  });

  final String label;
  final VoidCallback onPressed;
  final BalaurButtonVariant variant;
}

/// Displays an ornate decision panel.
class BalaurDialogPanel extends StatelessWidget {
  const BalaurDialogPanel({
    super.key,
    required this.title,
    required this.body,
    this.kicker,
    this.actions = const [],
  });

  final String title;
  final String body;
  final String? kicker;
  final List<BalaurDialogAction> actions;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      ornate: true,
      borderColor: colors.goldDeep,
      padding: const EdgeInsets.fromLTRB(22, 22, 22, 20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (kicker case final kicker?) ...[
            Text(
              kicker.toUpperCase(),
              style: Theme.of(context).textTheme.labelSmall?.copyWith(
                color: colors.goldInk,
                fontWeight: FontWeight.w700,
                letterSpacing: 1,
              ),
            ),
            const SizedBox(height: 8),
          ],
          Text(
            title,
            style: Theme.of(context).textTheme.headlineSmall
                ?.copyWith(color: colors.ink),
          ),
          const SizedBox(height: 10),
          Text(
            body,
            style: Theme.of(context).textTheme.bodyMedium
                ?.copyWith(color: colors.ink),
          ),
          if (actions.isNotEmpty) ...[
            const SizedBox(height: 18),
            Align(
              alignment: Alignment.centerRight,
              child: Wrap(
                spacing: 10,
                runSpacing: 10,
                children: [
                  for (final action in actions)
                    BalaurButton(
                      label: action.label,
                      onPressed: action.onPressed,
                      variant: action.variant,
                      size: BalaurButtonSize.small,
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

/// Displays an empty state with an optional action.
class BalaurEmptyState extends StatelessWidget {
  const BalaurEmptyState({
    super.key,
    this.title = 'Nothing on the book.',
    this.message,
    this.actionLabel,
    this.onAction,
    this.showCrest = true,
  });

  final String title;
  final String? message;
  final String? actionLabel;
  final VoidCallback? onAction;
  final bool showCrest;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 30),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (showCrest) ...[
            Image(image: BalaurAssets.crest, width: 88, height: 88),
            const SizedBox(height: 14),
          ],
          Text(
            title,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.titleLarge
                ?.copyWith(color: colors.foregroundStrong),
          ),
          if (message case final message?) ...[
            const SizedBox(height: 8),
            Text(
              message,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall
                  ?.copyWith(color: colors.muted),
            ),
          ],
          if (actionLabel case final label?) ...[
            const SizedBox(height: 18),
            BalaurButton(
              label: label,
              onPressed: onAction,
              variant: BalaurButtonVariant.wood,
            ),
          ],
        ],
      ),
    );
  }
}
