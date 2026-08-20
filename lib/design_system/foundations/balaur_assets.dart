import 'package:flutter/painting.dart';
import 'package:flutter/services.dart';

/// Provides the local Hearthwood image assets.
abstract final class BalaurAssets {
  static const _package = 'balaur';
  static const _root = 'assets/design_system';
  static const _probe = '$_root/crest.png';

  static const AssetBundleImageProvider crest = _BalaurAssetImage(_probe);
  static const AssetBundleImageProvider logo = _BalaurAssetImage(
    '$_root/logo.png',
  );

  static AssetBundleImageProvider icon(String name) {
    return _BalaurAssetImage('$_root/icons/$name.png');
  }

  static AssetBundleImageProvider balaurAvatar(int number) {
    return _BalaurAssetImage('$_root/avatars/balaur-${_twoDigits(number)}.png');
  }

  static AssetBundleImageProvider soulAvatar(int number) {
    return _BalaurAssetImage('$_root/avatars/soul-${_twoDigits(number)}.png');
  }

  static String _twoDigits(int value) {
    return value.toString().padLeft(2, '0');
  }
}

final class _BalaurAssetImage extends AssetBundleImageProvider {
  const _BalaurAssetImage(this.assetName);

  static final Expando<Future<bool>> _rootAssetAvailability = Expando();

  final String assetName;

  @override
  Future<AssetBundleImageKey> obtainKey(
    ImageConfiguration configuration,
  ) async {
    final bundle = configuration.bundle ?? rootBundle;
    final rootAssetsAvailable = await (_rootAssetAvailability[bundle] ??=
        AssetManifest.loadFromAssetBundle(bundle).then(
          (manifest) => manifest.getAssetVariants(BalaurAssets._probe) != null,
        ));
    final key = rootAssetsAvailable
        ? assetName
        : 'packages/${BalaurAssets._package}/$assetName';

    return AssetBundleImageKey(bundle: bundle, name: key, scale: 1);
  }

  @override
  bool operator ==(Object other) {
    return other is _BalaurAssetImage && other.assetName == assetName;
  }

  @override
  int get hashCode => assetName.hashCode;

  @override
  String toString() => 'BalaurAssetImage("$assetName")';
}
