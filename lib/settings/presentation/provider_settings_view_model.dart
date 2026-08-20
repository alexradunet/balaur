import 'package:balaur/settings/provider_settings.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/foundation.dart';

final class ProviderSettingsViewModel extends ChangeNotifier {
  ProviderSettingsViewModel({required this.settingsStore});

  final ProviderSettingsStore settingsStore;

  ProviderSettings _settings = const ProviderSettings.empty();
  ProviderSettings get settings => _settings;

  bool _isReady = false;
  bool get isReady => _isReady;

  bool _isSaving = false;
  bool get isSaving => _isSaving;

  String? _errorMessage;
  String? get errorMessage => _errorMessage;

  Future<void> initialize() async {
    try {
      _settings = await settingsStore.load();
    } on Object {
      _errorMessage = 'Balaur could not load the model provider settings.';
    } finally {
      _isReady = true;
      notifyListeners();
    }
  }

  Future<bool> save(ProviderSettings settings) async {
    _isSaving = true;
    _errorMessage = null;
    notifyListeners();
    try {
      await settingsStore.save(settings);
      _settings = settings;
      return true;
    } on Object {
      _errorMessage = 'Balaur could not save the model provider settings.';
      return false;
    } finally {
      _isSaving = false;
      notifyListeners();
    }
  }
}
