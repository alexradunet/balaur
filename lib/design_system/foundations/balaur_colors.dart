import 'package:flutter/material.dart';

/// Contains the Hearthwood color tokens for one brightness.
@immutable
final class BalaurColors extends ThemeExtension<BalaurColors> {
  const BalaurColors({
    required this.background,
    required this.chrome,
    required this.chromeDeep,
    required this.chromeForeground,
    required this.surface,
    required this.surface2,
    required this.surface3,
    required this.parchmentEdge,
    required this.foreground,
    required this.foregroundStrong,
    required this.muted,
    required this.smoke,
    required this.hair,
    required this.outline,
    required this.ink,
    required this.inkMuted,
    required this.bevelLight,
    required this.bevelDark,
    required this.gold,
    required this.goldDeep,
    required this.goldInk,
    required this.ember,
    required this.emberDeep,
    required this.emberRed,
    required this.teal,
    required this.tealDeep,
    required this.tealInk,
    required this.folkRed,
    required this.indigo,
    required this.indigoInk,
    required this.violet,
    required this.good,
    required this.goodInk,
    required this.steel,
  });

  static const light = BalaurColors(
    background: Color(0xffefe2bd),
    chrome: Color(0xff3a2210),
    chromeDeep: Color(0xff2c1a0c),
    chromeForeground: Color(0xffd6bb92),
    surface: Color(0xfff4e9c4),
    surface2: Color(0xffe4d49e),
    surface3: Color(0xffd2bf82),
    parchmentEdge: Color(0xff9a7f4c),
    foreground: Color(0xff3a2c18),
    foregroundStrong: Color(0xff241a0c),
    muted: Color(0xff675334),
    smoke: Color(0xff675334),
    hair: Color(0xffc8b488),
    outline: Color(0xff241708),
    ink: Color(0xff2c2012),
    inkMuted: Color(0xff5a4726),
    bevelLight: Color(0x4dffdea6),
    bevelDark: Color(0x8c000000),
    gold: Color(0xfff2c14e),
    goldDeep: Color(0xff7a5a14),
    goldInk: Color(0xff654508),
    ember: Color(0xffff7a33),
    emberDeep: Color(0xff7e3210),
    emberRed: Color(0xffa8201f),
    teal: Color(0xff3ecfb8),
    tealDeep: Color(0xff3ecfb8),
    tealInk: Color(0xff075447),
    folkRed: Color(0xff983f20),
    indigo: Color(0xffa8c0f0),
    indigoInk: Color(0xff2e3f7f),
    violet: Color(0xff6d3bb8),
    good: Color(0xff7fcf6a),
    goodInk: Color(0xff2f5419),
    steel: Color(0xff7a6644),
  );

  static const dark = BalaurColors(
    background: Color(0xff140c06),
    chrome: Color(0xff2a1709),
    chromeDeep: Color(0xff1d0f06),
    chromeForeground: Color(0xffb59872),
    surface: Color(0xffe8d9ae),
    surface2: Color(0xffd6c188),
    surface3: Color(0xffc4ab74),
    parchmentEdge: Color(0xff8a6f3c),
    foreground: Color(0xffc9b894),
    foregroundStrong: Color(0xffecdcb2),
    muted: Color(0xffb39b78),
    smoke: Color(0xffb39b78),
    hair: Color(0xff3b2a16),
    outline: Color(0xff120a04),
    ink: Color(0xff2c2012),
    inkMuted: Color(0xff5a4726),
    bevelLight: Color(0x2effc476),
    bevelDark: Color(0x8c000000),
    gold: Color(0xfff2c14e),
    goldDeep: Color(0xffa87b24),
    goldInk: Color(0xff654508),
    ember: Color(0xffff7a33),
    emberDeep: Color(0xff7e3210),
    emberRed: Color(0xffe5484d),
    teal: Color(0xff3ecfb8),
    tealDeep: Color(0xff3ecfb8),
    tealInk: Color(0xff075447),
    folkRed: Color(0xffe0563b),
    indigo: Color(0xffa8c0f0),
    indigoInk: Color(0xff2e3f7f),
    violet: Color(0xffc084fc),
    good: Color(0xff7fcf6a),
    goodInk: Color(0xff2f5419),
    steel: Color(0xff9b8a6c),
  );

  static BalaurColors of(BuildContext context) {
    return Theme.of(context).extension<BalaurColors>()!;
  }

  final Color background;
  final Color chrome;
  final Color chromeDeep;
  final Color chromeForeground;
  final Color surface;
  final Color surface2;
  final Color surface3;
  final Color parchmentEdge;
  final Color foreground;
  final Color foregroundStrong;
  final Color muted;
  final Color smoke;
  final Color hair;
  final Color outline;
  final Color ink;
  final Color inkMuted;
  final Color bevelLight;
  final Color bevelDark;
  final Color gold;
  final Color goldDeep;
  final Color goldInk;
  final Color ember;
  final Color emberDeep;
  final Color emberRed;
  final Color teal;
  final Color tealDeep;
  final Color tealInk;
  final Color folkRed;
  final Color indigo;
  final Color indigoInk;
  final Color violet;
  final Color good;
  final Color goodInk;
  final Color steel;

  @override
  BalaurColors copyWith() => this;

  @override
  BalaurColors lerp(covariant BalaurColors? other, double t) {
    return t < 0.5 || other == null ? this : other;
  }
}
