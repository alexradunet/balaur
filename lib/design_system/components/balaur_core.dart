import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Identifies the visual treatment of a [BalaurButton].
enum BalaurButtonVariant { primary, ghost, wood }

/// Identifies the size of a [BalaurButton].
enum BalaurButtonSize { regular, small }

/// Displays a beveled button that sinks when the owner presses it.
class BalaurButton extends StatefulWidget {
  const BalaurButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.variant = BalaurButtonVariant.primary,
    this.size = BalaurButtonSize.regular,
    this.leading,
  });

  final String label;
  final VoidCallback? onPressed;
  final BalaurButtonVariant variant;
  final BalaurButtonSize size;
  final Widget? leading;

  @override
  State<BalaurButton> createState() => _BalaurButtonState();
}

class _BalaurButtonState extends State<BalaurButton> {
  bool _pressed = false;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final enabled = widget.onPressed != null;
    final isSmall = widget.size == BalaurButtonSize.small;
    final background = switch (widget.variant) {
      BalaurButtonVariant.primary => colors.ember,
      BalaurButtonVariant.ghost => colors.surface2,
      BalaurButtonVariant.wood => colors.chrome,
    };
    final foreground = switch (widget.variant) {
      BalaurButtonVariant.primary => const Color(0xff1c0d04),
      BalaurButtonVariant.ghost => colors.ink,
      BalaurButtonVariant.wood => colors.gold,
    };

    return Semantics(
      button: true,
      enabled: enabled,
      child: AnimatedSlide(
        duration: const Duration(milliseconds: 80),
        offset: _pressed ? const Offset(0, 0.08) : Offset.zero,
        child: Opacity(
          opacity: enabled ? 1 : 0.48,
          child: Listener(
            onPointerDown: enabled
                ? (_) => setState(() => _pressed = true)
                : null,
            onPointerUp: enabled
                ? (_) => setState(() => _pressed = false)
                : null,
            onPointerCancel: enabled
                ? (_) => setState(() => _pressed = false)
                : null,
            child: Material(
              color: background,
              shape: BeveledRectangleBorder(
                side: BorderSide(color: colors.outline, width: 2),
              ),
              child: InkWell(
                onTap: widget.onPressed,
                canRequestFocus: enabled,
                focusColor: colors.teal.withValues(alpha: 0.18),
                hoverColor: Colors.white.withValues(alpha: 0.08),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(minHeight: 48),
                  child: Padding(
                    padding: isSmall
                        ? const EdgeInsets.symmetric(
                            horizontal: 13,
                            vertical: 8,
                          )
                        : const EdgeInsets.symmetric(
                            horizontal: 19,
                            vertical: 12,
                          ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        if (widget.leading case final leading?) ...[
                          leading,
                          const SizedBox(width: 9),
                        ],
                        Text(
                          widget.label.toUpperCase(),
                          style: Theme.of(context).textTheme.labelLarge
                              ?.copyWith(
                                color: foreground,
                                fontSize: isSmall ? 11 : 12.5,
                                fontWeight: FontWeight.w700,
                                letterSpacing: 0.8,
                              ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Displays content on a parchment card with a gold pixel notch.
class BalaurCard extends StatelessWidget {
  const BalaurCard({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(18),
    this.ornate = false,
    this.dashed = false,
  });

  final Widget child;
  final EdgeInsetsGeometry padding;
  final bool ornate;
  final bool dashed;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Stack(
      children: [
        BalaurSurface(
          ornate: ornate,
          dashed: dashed,
          padding: padding,
          child: DefaultTextStyle.merge(
            style: TextStyle(color: colors.ink),
            child: child,
          ),
        ),
        if (!ornate)
          Positioned(
            right: 7,
            top: 7,
            child: ColoredBox(
              color: colors.goldInk,
              child: const SizedBox.square(dimension: 7),
            ),
          ),
      ],
    );
  }
}

/// Displays a small wood label with the teal stitch rune.
class BalaurTag extends StatelessWidget {
  const BalaurTag({super.key, required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      material: BalaurSurfaceMaterial.wood,
      hardShadow: false,
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 4),
      child: Text.rich(
        TextSpan(
          children: [
            TextSpan(
              text: '▪ ',
              style: TextStyle(color: colors.teal),
            ),
            TextSpan(text: label.toUpperCase()),
          ],
        ),
        style: Theme.of(context).textTheme.labelMedium?.copyWith(
          color: colors.chromeForeground,
          fontSize: 11,
          letterSpacing: 0.6,
        ),
      ),
    );
  }
}

/// Displays a square importance dial.
class BalaurPips extends StatelessWidget {
  const BalaurPips({super.key, this.level = 0, this.max = 5});

  final int level;
  final int max;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Semantics(
      label: 'Importance $level of $max',
      child: ExcludeSemantics(
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: List.generate(
            max,
            (index) => Padding(
              padding: EdgeInsets.only(right: index == max - 1 ? 0 : 3),
              child: Container(
                width: 7,
                height: 7,
                decoration: BoxDecoration(
                  color: index < level ? colors.goldInk : Colors.transparent,
                  border: Border.all(
                    color: index < level
                        ? colors.goldInk
                        : colors.parchmentEdge,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Draws a dashed stitch between sections.
class BalaurStitch extends StatelessWidget {
  const BalaurStitch({super.key, this.color});

  final Color? color;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 2,
      width: double.infinity,
      child: CustomPaint(
        painter: _StitchPainter(color ?? BalaurColors.of(context).hair),
      ),
    );
  }
}

final class _StitchPainter extends CustomPainter {
  const _StitchPainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = 2;
    for (double x = 0; x < size.width; x += 8) {
      canvas.drawLine(
        Offset(x, 1),
        Offset((x + 4).clamp(0, size.width), 1),
        paint,
      );
    }
  }

  @override
  bool shouldRepaint(covariant _StitchPainter oldDelegate) {
    return oldDelegate.color != color;
  }
}

/// Draws the Romanian folk color band.
class BalaurFolkBand extends StatelessWidget {
  const BalaurFolkBand({super.key});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 16,
      width: double.infinity,
      child: CustomPaint(painter: _FolkBandPainter(BalaurColors.of(context))),
    );
  }
}

final class _FolkBandPainter extends CustomPainter {
  const _FolkBandPainter(this.colors);

  final BalaurColors colors;

  @override
  void paint(Canvas canvas, Size size) {
    final palette = [
      colors.folkRed,
      colors.gold,
      colors.teal,
      colors.emberDeep,
    ];
    for (double x = -16; x < size.width + 16; x += 8) {
      final index = ((x + 16) ~/ 8) % palette.length;
      final path = Path()
        ..moveTo(x, size.height)
        ..lineTo(x + 14, 0)
        ..lineTo(x + 22, 0)
        ..lineTo(x + 8, size.height)
        ..close();
      canvas.drawPath(path, Paint()..color = palette[index]);
    }
    final border = Paint()
      ..color = colors.outline
      ..strokeWidth = 2;
    canvas
      ..drawLine(Offset.zero, Offset(size.width, 0), border)
      ..drawLine(
        Offset(0, size.height),
        Offset(size.width, size.height),
        border,
      );
  }

  @override
  bool shouldRepaint(covariant _FolkBandPainter oldDelegate) {
    return oldDelegate.colors != colors;
  }
}

/// Identifies a badge color.
enum BalaurBadgeTone { gold, ember, teal, wood }

/// Displays a small count, status, or bare marker.
class BalaurBadge extends StatelessWidget {
  const BalaurBadge({
    super.key,
    this.label,
    this.tone = BalaurBadgeTone.gold,
    this.dot = false,
  });

  final String? label;
  final BalaurBadgeTone tone;
  final bool dot;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final background = switch (tone) {
      BalaurBadgeTone.gold => colors.gold,
      BalaurBadgeTone.ember => colors.ember,
      BalaurBadgeTone.teal => colors.tealDeep,
      BalaurBadgeTone.wood => colors.chromeDeep,
    };
    final foreground = tone == BalaurBadgeTone.wood
        ? colors.chromeForeground
        : const Color(0xff1c0d04);
    final border = switch (tone) {
      BalaurBadgeTone.gold => colors.goldDeep,
      BalaurBadgeTone.ember => colors.emberDeep,
      BalaurBadgeTone.teal || BalaurBadgeTone.wood => colors.outline,
    };

    return Container(
      constraints: BoxConstraints(
        minWidth: dot ? 9 : 20,
        minHeight: dot ? 9 : 20,
      ),
      padding: dot
          ? EdgeInsets.zero
          : const EdgeInsets.symmetric(horizontal: 6),
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: background,
        border: Border.all(color: border, width: 2),
      ),
      child: dot
          ? null
          : Text(
              label ?? '',
              style: Theme.of(context).textTheme.labelMedium
                  ?.copyWith(color: foreground, fontWeight: FontWeight.w700),
            ),
    );
  }
}

/// Identifies a loading placeholder shape.
enum BalaurSkeletonVariant { line, block, avatar }

/// Displays a carved loading placeholder.
class BalaurSkeleton extends StatelessWidget {
  const BalaurSkeleton({
    super.key,
    this.variant = BalaurSkeletonVariant.line,
    this.width,
    this.height,
  });

  final BalaurSkeletonVariant variant;
  final double? width;
  final double? height;

  @override
  Widget build(BuildContext context) {
    final size = switch (variant) {
      BalaurSkeletonVariant.line => (width ?? double.infinity, height ?? 13),
      BalaurSkeletonVariant.block => (width ?? double.infinity, height ?? 64),
      BalaurSkeletonVariant.avatar => (width ?? 48, height ?? width ?? 48),
    };
    return Semantics(
      label: 'Loading',
      child: BalaurSurface(
        material: BalaurSurfaceMaterial.inset,
        hardShadow: false,
        child: SizedBox(width: size.$1, height: size.$2),
      ),
    );
  }
}
