import 'package:flutter/widgets.dart';

/// Provides the local Hearthwood image assets.
abstract final class BalaurAssets {
  static const _package = 'balaur';
  static const _root = 'assets/design_system';

  static const crest = AssetImage('$_root/crest.png', package: _package);
  static const logo = AssetImage('$_root/logo.png', package: _package);

  static AssetImage icon(String name) {
    return AssetImage('$_root/icons/$name.png', package: _package);
  }

  static AssetImage balaurAvatar(int number) {
    return AssetImage(
      '$_root/avatars/balaur-${_twoDigits(number)}.png',
      package: _package,
    );
  }

  static AssetImage soulAvatar(int number) {
    return AssetImage(
      '$_root/avatars/soul-${_twoDigits(number)}.png',
      package: _package,
    );
  }

  static String _twoDigits(int value) {
    return value.toString().padLeft(2, '0');
  }
}
