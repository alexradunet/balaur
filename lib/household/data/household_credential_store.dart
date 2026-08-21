import 'package:flutter_secure_storage/flutter_secure_storage.dart';

final class StoredHouseholdCredentials {
  const StoredHouseholdCredentials({
    required this.serverAddress,
    required this.authentication,
  });

  final String serverAddress;
  final String authentication;
}

abstract interface class HouseholdCredentialStore {
  Future<StoredHouseholdCredentials?> load();

  Future<void> save(StoredHouseholdCredentials credentials);

  Future<void> clear();
}

final class SecureHouseholdCredentialStore implements HouseholdCredentialStore {
  SecureHouseholdCredentialStore(this._storage);

  static const _serverAddressKey = 'household.server_address';
  static const _authenticationKey = 'household.authentication';

  final FlutterSecureStorage _storage;

  @override
  Future<StoredHouseholdCredentials?> load() async {
    final values = await Future.wait([
      _storage.read(key: _serverAddressKey),
      _storage.read(key: _authenticationKey),
    ]);
    final serverAddress = values[0];
    final authentication = values[1];
    if (serverAddress == null && authentication == null) {
      return null;
    }
    if (serverAddress == null || authentication == null) {
      await clear();
      return null;
    }
    return StoredHouseholdCredentials(
      serverAddress: serverAddress,
      authentication: authentication,
    );
  }

  @override
  Future<void> save(StoredHouseholdCredentials credentials) async {
    await Future.wait([
      _storage.write(key: _serverAddressKey, value: credentials.serverAddress),
      _storage.write(
        key: _authenticationKey,
        value: credentials.authentication,
      ),
    ]);
  }

  @override
  Future<void> clear() async {
    await Future.wait([
      _storage.delete(key: _serverAddressKey),
      _storage.delete(key: _authenticationKey),
    ]);
  }
}

final class InMemoryHouseholdCredentialStore
    implements HouseholdCredentialStore {
  StoredHouseholdCredentials? _credentials;

  @override
  Future<StoredHouseholdCredentials?> load() async => _credentials;

  @override
  Future<void> save(StoredHouseholdCredentials credentials) async {
    _credentials = credentials;
  }

  @override
  Future<void> clear() async {
    _credentials = null;
  }
}
