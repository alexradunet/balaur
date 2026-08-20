import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Builds the shared Hearthwood themes.
abstract final class BalaurTheme {
  static ThemeData light() => _build(Brightness.light, BalaurColors.light);

  static ThemeData dark() => _build(Brightness.dark, BalaurColors.dark);

  static ThemeData _build(Brightness brightness, BalaurColors colors) {
    final colorScheme =
        ColorScheme.fromSeed(
          seedColor: colors.gold,
          brightness: brightness,
          primary: colors.gold,
          secondary: colors.teal,
          surface: colors.surface,
          error: colors.emberRed,
        ).copyWith(
          onPrimary: const Color(0xff1c0d04),
          onSecondary: const Color(0xff06120f),
          onSurface: colors.ink,
          outline: colors.parchmentEdge,
          outlineVariant: colors.hair,
        );
    final textTheme = _textTheme(colors);

    return ThemeData(
      brightness: brightness,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: colors.background,
      canvasColor: colors.background,
      textTheme: textTheme,
      primaryTextTheme: textTheme,
      useMaterial3: true,
      splashFactory: NoSplash.splashFactory,
      extensions: [colors],
      appBarTheme: AppBarTheme(
        backgroundColor: colors.chrome,
        foregroundColor: colors.chromeForeground,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        shape: Border(bottom: BorderSide(color: colors.outline, width: 2)),
        titleTextStyle: _textStyle(
          family: 'Silkscreen',
          size: 14,
          color: colors.gold,
          weight: FontWeight.w700,
          letterSpacing: 1.1,
        ),
      ),
      dividerTheme: DividerThemeData(
        color: colors.hair,
        thickness: 2,
        space: 2,
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: colors.surface,
        hintStyle: textTheme.bodyMedium?.copyWith(color: colors.inkMuted),
        labelStyle: textTheme.labelMedium?.copyWith(color: colors.inkMuted),
        border: _inputBorder(colors.parchmentEdge),
        enabledBorder: _inputBorder(colors.parchmentEdge),
        focusedBorder: _inputBorder(colors.goldDeep),
        errorBorder: _inputBorder(colors.emberRed),
        focusedErrorBorder: _inputBorder(colors.emberRed),
      ),
      dialogTheme: DialogThemeData(
        backgroundColor: colors.surface,
        surfaceTintColor: Colors.transparent,
        shape: BeveledRectangleBorder(
          side: BorderSide(color: colors.goldDeep, width: 2),
        ),
      ),
      tooltipTheme: TooltipThemeData(
        decoration: BoxDecoration(
          color: colors.chrome,
          border: Border.all(color: colors.outline, width: 2),
        ),
        textStyle: textTheme.labelSmall?.copyWith(
          color: colors.chromeForeground,
        ),
      ),
      textSelectionTheme: TextSelectionThemeData(
        cursorColor: colors.emberDeep,
        selectionColor: colors.teal.withValues(alpha: 0.28),
        selectionHandleColor: colors.teal,
      ),
    );
  }

  static TextTheme _textTheme(BalaurColors colors) {
    return TextTheme(
      displayLarge: _display(48, colors.foregroundStrong),
      displayMedium: _display(40, colors.foregroundStrong),
      displaySmall: _display(34, colors.foregroundStrong),
      headlineLarge: _display(32, colors.foregroundStrong),
      headlineMedium: _display(28, colors.foregroundStrong),
      headlineSmall: _display(24, colors.foregroundStrong),
      titleLarge: _display(22, colors.foregroundStrong),
      titleMedium: _display(20, colors.foregroundStrong),
      titleSmall: _display(18, colors.foregroundStrong),
      bodyLarge: _body(17, colors.foreground),
      bodyMedium: _body(16, colors.foreground),
      bodySmall: _body(14, colors.muted),
      labelLarge: _mono(12.5, colors.foreground),
      labelMedium: _mono(11, colors.muted),
      labelSmall: _mono(10, colors.muted),
    );
  }

  static TextStyle _display(double size, Color color) {
    return _textStyle(
      family: 'Jersey 15',
      size: size,
      color: color,
      height: 1.05,
    );
  }

  static TextStyle _body(double size, Color color) {
    return _textStyle(
      family: 'Piazzolla',
      size: size,
      color: color,
      height: 1.6,
    );
  }

  static TextStyle _mono(double size, Color color) {
    return _textStyle(
      family: 'JetBrains Mono',
      size: size,
      color: color,
      height: 1.35,
      letterSpacing: 0.5,
    );
  }

  static TextStyle _textStyle({
    required String family,
    required double size,
    required Color color,
    double? height,
    double? letterSpacing,
    FontWeight? weight,
  }) {
    return TextStyle(
      fontFamily: family,
      package: 'balaur',
      fontSize: size,
      color: color,
      height: height,
      letterSpacing: letterSpacing,
      fontWeight: weight,
    );
  }

  static OutlineInputBorder _inputBorder(Color color) {
    return OutlineInputBorder(
      borderRadius: BorderRadius.zero,
      borderSide: BorderSide(color: color, width: 2),
    );
  }
}
