import 'dart:math' as math;

import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_assets.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Identifies a trailing list value that needs attention.
enum BalaurListMetaTone { normal, warning }

/// Displays one row in a [BalaurList].
class BalaurListItem extends StatelessWidget {
  const BalaurListItem({
    super.key,
    required this.title,
    this.subtitle,
    this.meta,
    this.metaTone = BalaurListMetaTone.normal,
    this.iconName,
    this.onTap,
  });

  final String title;
  final String? subtitle;
  final String? meta;
  final BalaurListMetaTone metaTone;
  final String? iconName;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 11),
        child: Row(
          children: [
            if (iconName case final icon?) ...[
              Image(image: BalaurAssets.icon(icon), width: 22, height: 22),
              const SizedBox(width: 12),
            ],
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: colors.ink,
                      fontSize: 14.5,
                      height: 1.3,
                    ),
                  ),
                  if (subtitle case final subtitle?) ...[
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.labelMedium
                          ?.copyWith(color: colors.inkMuted),
                    ),
                  ],
                ],
              ),
            ),
            if (meta case final meta?) ...[
              const SizedBox(width: 10),
              Text(
                meta.toUpperCase(),
                style: Theme.of(context).textTheme.labelMedium?.copyWith(
                  color: metaTone == BalaurListMetaTone.warning
                      ? colors.emberDeep
                      : colors.inkMuted,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// Displays a homogeneous parchment list.
class BalaurList extends StatelessWidget {
  const BalaurList({super.key, required this.items, this.title});

  final List<BalaurListItem> items;
  final String? title;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      padding: EdgeInsets.zero,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (title case final title?)
            Padding(
              padding: const EdgeInsets.fromLTRB(14, 11, 14, 9),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text(
                  title.toUpperCase(),
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: colors.goldInk,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 0.9,
                  ),
                ),
              ),
            ),
          for (var index = 0; index < items.length; index++) ...[
            if (index > 0 || title != null)
              Divider(color: colors.parchmentEdge, height: 1, thickness: 1),
            items[index],
          ],
        ],
      ),
    );
  }
}

/// Displays one square calendar day.
class BalaurCalendarCell extends StatelessWidget {
  const BalaurCalendarCell({
    super.key,
    required this.day,
    this.pips = 0,
    this.today = false,
    this.selected = false,
    this.dimmed = false,
    this.onPressed,
    this.semanticLabel,
  });

  final int day;
  final int pips;
  final bool today;
  final bool selected;
  final bool dimmed;
  final VoidCallback? onPressed;
  final String? semanticLabel;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Semantics(
      selected: selected,
      label: semanticLabel ?? 'Day $day${today ? ', today' : ''}, $pips items',
      button: onPressed != null,
      child: Opacity(
        opacity: dimmed ? 0.4 : 1,
        child: LayoutBuilder(
          builder: (context, constraints) {
            final dimension = constraints.hasBoundedWidth
                ? constraints.hasBoundedHeight
                      ? math.min(constraints.maxWidth, constraints.maxHeight)
                      : constraints.maxWidth
                : constraints.hasBoundedHeight
                ? constraints.maxHeight
                : 64.0;
            return SizedBox.square(
              dimension: dimension,
              child: Material(
                color: selected ? colors.gold : colors.surface,
                shape: BeveledRectangleBorder(
                  side: BorderSide(
                    color: today && !selected
                        ? colors.goldDeep
                        : colors.parchmentEdge,
                    width: 2,
                  ),
                ),
                child: InkWell(
                  onTap: onPressed,
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Text(
                        '$day',
                        style: Theme.of(context).textTheme.labelLarge?.copyWith(
                          color: selected
                              ? const Color(0xff1c0d04)
                              : colors.ink,
                          fontWeight: today || selected
                              ? FontWeight.w700
                              : null,
                        ),
                      ),
                      const SizedBox(height: 5),
                      Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          for (var index = 0; index < pips.clamp(0, 3); index++)
                            Padding(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 1.5,
                              ),
                              child: ColoredBox(
                                color: selected
                                    ? const Color(0xff1c0d04)
                                    : [
                                        colors.ember,
                                        colors.tealInk,
                                        colors.goldInk,
                                      ][index],
                                child: const SizedBox.square(dimension: 5),
                              ),
                            ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            );
          },
        ),
      ),
    );
  }
}

/// Identifies a timeline node color.
enum BalaurDayEntryTone { gold, teal, ember }

/// Displays one item on a vertical day timeline.
class BalaurDayEntry extends StatelessWidget {
  const BalaurDayEntry({
    super.key,
    required this.time,
    required this.title,
    this.detail,
    this.tone = BalaurDayEntryTone.gold,
    this.last = false,
  });

  final String time;
  final String title;
  final String? detail;
  final BalaurDayEntryTone tone;
  final bool last;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final node = switch (tone) {
      BalaurDayEntryTone.gold => colors.goldInk,
      BalaurDayEntryTone.teal => colors.tealInk,
      BalaurDayEntryTone.ember => colors.emberDeep,
    };
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SizedBox(
            width: 54,
            child: Text(
              time.toUpperCase(),
              textAlign: TextAlign.right,
              style: Theme.of(context).textTheme.labelMedium?.copyWith(
                color: colors.ink,
                fontWeight: FontWeight.w700,
                fontSize: 13,
              ),
            ),
          ),
          const SizedBox(width: 12),
          SizedBox(
            width: 22,
            child: Stack(
              alignment: Alignment.topCenter,
              children: [
                if (!last)
                  Positioned(
                    top: 12,
                    bottom: 0,
                    child: ColoredBox(
                      color: colors.parchmentEdge,
                      child: const SizedBox(width: 2),
                    ),
                  ),
                Container(
                  width: 12,
                  height: 12,
                  decoration: BoxDecoration(
                    color: node,
                    border: Border.all(color: colors.surface, width: 2),
                    boxShadow: [BoxShadow(color: node, spreadRadius: 2)],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Padding(
              padding: EdgeInsets.only(bottom: last ? 0 : 18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: Theme.of(context).textTheme.bodySmall
                        ?.copyWith(color: colors.ink, fontSize: 14.5),
                  ),
                  if (detail case final detail?) ...[
                    const SizedBox(height: 3),
                    Text(
                      detail,
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: colors.ink,
                        fontWeight: FontWeight.w500,
                        fontSize: 14,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// Displays a compact trend without axes.
class BalaurSparkline extends StatelessWidget {
  const BalaurSparkline({
    super.key,
    required this.values,
    this.color,
    this.width = 150,
    this.height = 34,
  }) : assert(values.length > 1);

  final List<double> values;
  final Color? color;
  final double width;
  final double height;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: 'Trend from ${values.first} to ${values.last}',
      child: SizedBox(
        width: width,
        height: height,
        child: CustomPaint(
          painter: _SparklinePainter(
            values: values,
            color: color ?? BalaurColors.of(context).tealInk,
          ),
        ),
      ),
    );
  }
}

final class _SparklinePainter extends CustomPainter {
  const _SparklinePainter({required this.values, required this.color});

  final List<double> values;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    const pad = 3.0;
    final minValue = values.reduce(math.min);
    final maxValue = values.reduce(math.max);
    final span = math.max(maxValue - minValue, 1);
    final step = (size.width - pad * 2) / (values.length - 1);
    final points = [
      for (var index = 0; index < values.length; index++)
        Offset(
          pad + index * step,
          pad +
              (size.height - pad * 2) * (1 - (values[index] - minValue) / span),
        ),
    ];
    final line = Path()..moveTo(points.first.dx, points.first.dy);
    for (final point in points.skip(1)) {
      line.lineTo(point.dx, point.dy);
    }
    final area = Path.from(line)
      ..lineTo(points.last.dx, size.height - pad)
      ..lineTo(points.first.dx, size.height - pad)
      ..close();
    canvas
      ..drawPath(area, Paint()..color = color.withValues(alpha: 0.12))
      ..drawPath(
        line,
        Paint()
          ..color = color
          ..strokeWidth = 2
          ..style = PaintingStyle.stroke
          ..strokeJoin = StrokeJoin.round,
      )
      ..drawRect(
        Rect.fromCenter(center: points.last, width: 5, height: 5),
        Paint()..color = color,
      );
  }

  @override
  bool shouldRepaint(covariant _SparklinePainter oldDelegate) {
    return oldDelegate.values != values || oldDelegate.color != color;
  }
}

/// Identifies a metric change direction.
enum BalaurStatDeltaTone { up, down, flat }

/// Displays one life metric with a trend.
class BalaurStatCard extends StatelessWidget {
  const BalaurStatCard({
    super.key,
    required this.label,
    required this.value,
    this.unit,
    this.delta,
    this.deltaTone = BalaurStatDeltaTone.flat,
    this.values,
    this.iconName,
  });

  final String label;
  final String value;
  final String? unit;
  final String? delta;
  final BalaurStatDeltaTone deltaTone;
  final List<double>? values;
  final String? iconName;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final deltaColor = switch (deltaTone) {
      BalaurStatDeltaTone.up => colors.goodInk,
      BalaurStatDeltaTone.down => colors.emberDeep,
      BalaurStatDeltaTone.flat => colors.inkMuted,
    };
    final marker = switch (deltaTone) {
      BalaurStatDeltaTone.up => '▲ ',
      BalaurStatDeltaTone.down => '▼ ',
      BalaurStatDeltaTone.flat => '',
    };
    return BalaurSurface(
      padding: const EdgeInsets.fromLTRB(16, 15, 16, 14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              if (iconName case final icon?) ...[
                Image(image: BalaurAssets.icon(icon), width: 18, height: 18),
                const SizedBox(width: 8),
              ],
              Text(
                label.toUpperCase(),
                style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: colors.inkMuted,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
          const SizedBox(height: 11),
          Wrap(
            spacing: 12,
            runSpacing: 4,
            alignment: WrapAlignment.spaceBetween,
            crossAxisAlignment: WrapCrossAlignment.end,
            children: [
              Row(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(
                    value,
                    style: Theme.of(context).textTheme.headlineLarge
                        ?.copyWith(color: colors.ink),
                  ),
                  if (unit case final unit?) ...[
                    const SizedBox(width: 7),
                    Text(
                      unit,
                      style: Theme.of(context).textTheme.labelLarge
                          ?.copyWith(color: colors.inkMuted),
                    ),
                  ],
                ],
              ),
              if (delta case final delta?)
                Text(
                  '$marker$delta',
                  style: Theme.of(context).textTheme.labelMedium
                      ?.copyWith(color: deltaColor),
                ),
            ],
          ),
          if (values case final values?) ...[
            const SizedBox(height: 11),
            BalaurSparkline(
              values: values,
              color: deltaTone == BalaurStatDeltaTone.up
                  ? colors.goodInk
                  : deltaTone == BalaurStatDeltaTone.down
                  ? colors.emberDeep
                  : colors.tealInk,
            ),
          ],
        ],
      ),
    );
  }
}
