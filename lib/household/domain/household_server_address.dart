final class HouseholdServerAddress {
  const HouseholdServerAddress._(this.uri);

  factory HouseholdServerAddress.parse(String value) {
    return _parse(value, allowLoopbackHttp: false);
  }

  factory HouseholdServerAddress.loopbackForTesting(String value) {
    return _parse(value, allowLoopbackHttp: true);
  }

  final Uri uri;

  String get value => uri.toString();

  static HouseholdServerAddress _parse(
    String value, {
    required bool allowLoopbackHttp,
  }) {
    final uri = Uri.tryParse(value.trim());
    if (uri == null || !uri.hasAuthority || uri.host.isEmpty) {
      throw const FormatException('Enter a valid Household Server address.');
    }

    final scheme = uri.scheme.toLowerCase();
    final host = uri.host.toLowerCase();
    final isLoopback =
        host == 'localhost' || host == '127.0.0.1' || host == '::1';
    final usesAllowedScheme =
        scheme == 'https' ||
        (allowLoopbackHttp && scheme == 'http' && isLoopback);
    if (!usesAllowedScheme ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment) {
      throw const FormatException(
        'Enter a stable HTTPS Household Server address.',
      );
    }

    var path = uri.path;
    while (path.length > 1 && path.endsWith('/')) {
      path = path.substring(0, path.length - 1);
    }

    return HouseholdServerAddress._(
      uri.replace(scheme: scheme, host: host, path: path == '/' ? '' : path),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is HouseholdServerAddress && other.uri == uri;
  }

  @override
  int get hashCode => uri.hashCode;

  @override
  String toString() => value;
}
