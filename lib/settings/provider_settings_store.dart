import 'package:balaur/settings/provider_settings.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

abstract interface class ProviderSettingsStore {
  Future<ProviderSettings> load();

  Future<void> save(ProviderSettings settings);
}

class SecureProviderSettingsStore implements ProviderSettingsStore {
  SecureProviderSettingsStore(this._storage);

  static const _baseUrlKey = 'provider.base_url';
  static const _apiKeyKey = 'provider.api_key';
  static const _modelKey = 'provider.model';

  final FlutterSecureStorage _storage;

  @override
  Future<ProviderSettings> load() async {
    final values = await Future.wait([
      _storage.read(key: _baseUrlKey),
      _storage.read(key: _apiKeyKey),
      _storage.read(key: _modelKey),
    ]);

    return ProviderSettings(
      baseUrl: values[0] ?? const ProviderSettings.empty().baseUrl,
      apiKey: values[1] ?? '',
      model: values[2] ?? '',
    );
  }

  @override
  Future<void> save(ProviderSettings settings) async {
    await Future.wait([
      _storage.write(key: _baseUrlKey, value: settings.normalizedBaseUrl),
      _storage.write(key: _apiKeyKey, value: settings.apiKey.trim()),
      _storage.write(key: _modelKey, value: settings.model.trim()),
    ]);
  }
}

class InMemoryProviderSettingsStore implements ProviderSettingsStore {
  InMemoryProviderSettingsStore([
    this._settings = const ProviderSettings.empty(),
  ]);

  ProviderSettings _settings;

  @override
  Future<ProviderSettings> load() async => _settings;

  @override
  Future<void> save(ProviderSettings settings) async {
    _settings = settings;
  }
}
