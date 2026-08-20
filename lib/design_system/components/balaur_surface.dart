import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Identifies a Hearthwood material.
enum BalaurSurfaceMaterial { parchment, wood, inset }

/// Draws a reusable Hearthwood material with pixel-hard construction.
class BalaurSurface extends StatelessWidget {
  const BalaurSurface({
    super.key,
    required this.child,
    this.material = BalaurSurfaceMaterial.parchment,
    this.padding = EdgeInsets.zero,
    this.ornate = false,
    this.dashed = false,
    this.hardShadow = true,
    this.borderColor,
  });

  final Widget child;
  final BalaurSurfaceMaterial material;
  final EdgeInsetsGeometry padding;
  final bool ornate;
  final bool dashed;
  final bool hardShadow;
  final Color? borderColor;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final resolvedBorder =
        borderColor ??
        switch (material) {
          BalaurSurfaceMaterial.parchment => colors.parchmentEdge,
          BalaurSurfaceMaterial.wood ||
          BalaurSurfaceMaterial.inset => colors.outline,
        };

    return CustomPaint(
      painter: _BalaurSurfacePainter(
        colors: colors,
        material: material,
        borderColor: resolvedBorder,
        ornate: ornate,
        dashed: dashed,
        hardShadow: hardShadow,
      ),
      child: Padding(padding: padding, child: child),
    );
  }
}

final class _BalaurSurfacePainter extends CustomPainter {
  const _BalaurSurfacePainter({
    required this.colors,
    required this.material,
    required this.borderColor,
    required this.ornate,
    required this.dashed,
    required this.hardShadow,
  });

  final BalaurColors colors;
  final BalaurSurfaceMaterial material;
  final Color borderColor;
  final bool ornate;
  final bool dashed;
  final bool hardShadow;

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    if (hardShadow) {
      canvas.drawRect(
        rect.shift(const Offset(0, 3)),
        Paint()..color = colors.bevelDark,
      );
    }

    final fill = switch (material) {
      BalaurSurfaceMaterial.parchment => colors.surface,
      BalaurSurfaceMaterial.wood => colors.chrome,
      BalaurSurfaceMaterial.inset => colors.chromeDeep,
    };
    canvas.drawRect(rect, Paint()..color = fill);
    _paintTexture(canvas, size);

    final isInset = material == BalaurSurfaceMaterial.inset;
    final light = isInset ? colors.bevelDark : colors.bevelLight;
    final dark = isInset ? colors.bevelLight : colors.bevelDark;
    final bevel = Paint()
      ..strokeWidth = material == BalaurSurfaceMaterial.parchment ? 2 : 3
      ..style = PaintingStyle.stroke;
    canvas
      ..drawLine(Offset(2, 2), Offset(size.width - 2, 2), bevel..color = light)
      ..drawLine(Offset(2, 2), Offset(2, size.height - 2), bevel)
      ..drawLine(
        Offset(2, size.height - 2),
        Offset(size.width - 2, size.height - 2),
        bevel..color = dark,
      )
      ..drawLine(
        Offset(size.width - 2, 2),
        Offset(size.width - 2, size.height - 2),
        bevel,
      );

    _paintBorder(canvas, rect);
    if (ornate) {
      _paintBrackets(canvas, size);
    }
  }

  void _paintTexture(Canvas canvas, Size size) {
    final grain = Paint()
      ..color = material == BalaurSurfaceMaterial.parchment
          ? const Color(0x0b402c12)
          : const Color(0x0affd696);
    for (double y = 2; y < size.height; y += 4) {
      final offset = (y ~/ 4).isEven ? 0.0 : 2.0;
      for (double x = 2 + offset; x < size.width; x += 4) {
        canvas.drawRect(Rect.fromLTWH(x, y, 1, 1), grain);
      }
    }
    if (material != BalaurSurfaceMaterial.parchment) {
      final plank = Paint()
        ..color = const Color(0x29000000)
        ..strokeWidth = 2;
      for (double y = 8; y < size.height; y += 9) {
        canvas.drawLine(Offset(0, y), Offset(size.width, y), plank);
      }
    }
  }

  void _paintBorder(Canvas canvas, Rect rect) {
    final paint = Paint()
      ..color = borderColor
      ..strokeWidth = 2
      ..style = PaintingStyle.stroke;
    if (!dashed) {
      canvas.drawRect(rect.deflate(1), paint);
      return;
    }
    const dash = 7.0;
    const gap = 5.0;
    for (double x = 1; x < rect.width - 1; x += dash + gap) {
      canvas.drawLine(
        Offset(x, 1),
        Offset((x + dash).clamp(1, rect.width - 1), 1),
        paint,
      );
      canvas.drawLine(
        Offset(x, rect.height - 1),
        Offset((x + dash).clamp(1, rect.height - 1), rect.height - 1),
        paint,
      );
    }
    for (double y = 1; y < rect.height - 1; y += dash + gap) {
      canvas.drawLine(
        Offset(1, y),
        Offset(1, (y + dash).clamp(1, rect.height - 1)),
        paint,
      );
      canvas.drawLine(
        Offset(rect.width - 1, y),
        Offset(rect.width - 1, (y + dash).clamp(1, rect.height - 1)),
        paint,
      );
    }
  }

  void _paintBrackets(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = colors.goldDeep
      ..strokeWidth = 3
      ..style = PaintingStyle.stroke;
    const inset = 6.0;
    const length = 12.0;
    final paths = [
      Path()
        ..moveTo(inset, inset + length)
        ..lineTo(inset, inset)
        ..lineTo(inset + length, inset),
      Path()
        ..moveTo(size.width - inset - length, inset)
        ..lineTo(size.width - inset, inset)
        ..lineTo(size.width - inset, inset + length),
      Path()
        ..moveTo(inset, size.height - inset - length)
        ..lineTo(inset, size.height - inset)
        ..lineTo(inset + length, size.height - inset),
      Path()
        ..moveTo(size.width - inset - length, size.height - inset)
        ..lineTo(size.width - inset, size.height - inset)
        ..lineTo(size.width - inset, size.height - inset - length),
    ];
    for (final path in paths) {
      canvas.drawPath(path, paint);
    }
  }

  @override
  bool shouldRepaint(covariant _BalaurSurfacePainter oldDelegate) {
    return oldDelegate.colors != colors ||
        oldDelegate.material != material ||
        oldDelegate.borderColor != borderColor ||
        oldDelegate.ornate != ornate ||
        oldDelegate.dashed != dashed ||
        oldDelegate.hardShadow != hardShadow;
  }
}
