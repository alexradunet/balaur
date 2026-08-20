import 'package:flutter/material.dart';

/// Builds the shared Balaur Material themes.
abstract final class BalaurTheme {
  static ThemeData light() => _build(Brightness.light);

  static ThemeData dark() => _build(Brightness.dark);

  static ThemeData _build(Brightness brightness) {
    return ThemeData(
      colorScheme: ColorScheme.fromSeed(
        seedColor: Colors.green,
        brightness: brightness,
      ),
      useMaterial3: true,
    );
  }
}
