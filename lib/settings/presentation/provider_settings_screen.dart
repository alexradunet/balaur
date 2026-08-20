import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/settings/presentation/provider_settings_view_model.dart';
import 'package:balaur/settings/provider_settings.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/material.dart';

class ProviderSettingsScreen extends StatefulWidget {
  const ProviderSettingsScreen({super.key, required this.settingsStore});

  final ProviderSettingsStore settingsStore;

  @override
  State<ProviderSettingsScreen> createState() => _ProviderSettingsScreenState();
}

class _ProviderSettingsScreenState extends State<ProviderSettingsScreen> {
  late final ProviderSettingsViewModel _viewModel;

  @override
  void initState() {
    super.initState();
    _viewModel = ProviderSettingsViewModel(settingsStore: widget.settingsStore)
      ..initialize();
  }

  @override
  void dispose() {
    _viewModel.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: _viewModel,
      builder: (context, _) {
        if (!_viewModel.isReady) {
          return const Center(child: CircularProgressIndicator());
        }

        return ProviderSettingsView(
          initialSettings: _viewModel.settings,
          isSaving: _viewModel.isSaving,
          errorMessage: _viewModel.errorMessage,
          onSave: _save,
        );
      },
    );
  }

  Future<void> _save(ProviderSettings settings) async {
    final saved = await _viewModel.save(settings);
    if (!mounted || !saved) {
      return;
    }

    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        const SnackBar(content: Text('Model provider settings saved.')),
      );
  }
}

class ProviderSettingsView extends StatefulWidget {
  const ProviderSettingsView({
    super.key,
    required this.initialSettings,
    required this.onSave,
    this.isSaving = false,
    this.errorMessage,
  });

  final ProviderSettings initialSettings;
  final Future<void> Function(ProviderSettings) onSave;
  final bool isSaving;
  final String? errorMessage;

  @override
  State<ProviderSettingsView> createState() => _ProviderSettingsViewState();
}

class _ProviderSettingsViewState extends State<ProviderSettingsView> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _baseUrlController;
  late final TextEditingController _apiKeyController;
  late final TextEditingController _modelController;
  bool _hideApiKey = true;

  @override
  void initState() {
    super.initState();
    _baseUrlController = TextEditingController(
      text: widget.initialSettings.baseUrl,
    );
    _apiKeyController = TextEditingController(
      text: widget.initialSettings.apiKey,
    );
    _modelController = TextEditingController(
      text: widget.initialSettings.model,
    );
  }

  @override
  void dispose() {
    _baseUrlController.dispose();
    _apiKeyController.dispose();
    _modelController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: LayoutBuilder(
        builder: (context, constraints) {
          final horizontalPadding = constraints.maxWidth < 700 ? 16.0 : 40.0;
          return SingleChildScrollView(
            padding: EdgeInsets.symmetric(
              horizontal: horizontalPadding,
              vertical: 28,
            ),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 640),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'Model provider',
                      style: Theme.of(context).textTheme.headlineMedium,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'Configure the provider that answers your conversations.',
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                    const SizedBox(height: 24),
                    if (widget.errorMessage case final message?) ...[
                      BalaurAlert(
                        title: 'Settings error',
                        message: message,
                        tone: BalaurAlertTone.danger,
                      ),
                      const SizedBox(height: 16),
                    ],
                    BalaurSurface(
                      padding: const EdgeInsets.all(24),
                      child: Form(
                        key: _formKey,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            TextFormField(
                              key: const Key('provider-base-url'),
                              controller: _baseUrlController,
                              decoration: const InputDecoration(
                                labelText: 'Base URL',
                                hintText: 'https://api.openai.com/v1',
                              ),
                              keyboardType: TextInputType.url,
                              validator: _validateBaseUrl,
                            ),
                            const SizedBox(height: 16),
                            TextFormField(
                              key: const Key('provider-api-key'),
                              controller: _apiKeyController,
                              decoration: InputDecoration(
                                labelText: 'API key',
                                suffixIcon: IconButton(
                                  onPressed: () => setState(() {
                                    _hideApiKey = !_hideApiKey;
                                  }),
                                  tooltip: _hideApiKey
                                      ? 'Show API key'
                                      : 'Hide API key',
                                  icon: Icon(
                                    _hideApiKey
                                        ? Icons.visibility
                                        : Icons.visibility_off,
                                  ),
                                ),
                              ),
                              obscureText: _hideApiKey,
                              validator: _required,
                            ),
                            const SizedBox(height: 16),
                            TextFormField(
                              key: const Key('provider-model'),
                              controller: _modelController,
                              decoration: const InputDecoration(
                                labelText: 'Model',
                                hintText: 'gpt-5-mini',
                              ),
                              validator: _required,
                            ),
                            const SizedBox(height: 16),
                            const Text(
                              'Balaur stores these settings in secure storage on this device.',
                            ),
                            const SizedBox(height: 24),
                            Align(
                              alignment: Alignment.centerRight,
                              child: FilledButton(
                                key: const Key('save-provider-settings'),
                                onPressed: widget.isSaving ? null : _save,
                                child: widget.isSaving
                                    ? const SizedBox.square(
                                        dimension: 20,
                                        child: CircularProgressIndicator(
                                          strokeWidth: 2,
                                        ),
                                      )
                                    : const Text('Save'),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  String? _required(String? value) {
    return value == null || value.trim().isEmpty ? 'Enter a value.' : null;
  }

  String? _validateBaseUrl(String? value) {
    if (_required(value) case final error?) {
      return error;
    }
    final uri = Uri.tryParse(value!.trim());
    if (uri == null ||
        !uri.hasScheme ||
        (uri.scheme != 'https' && uri.scheme != 'http') ||
        uri.host.isEmpty) {
      return 'Enter a valid HTTP or HTTPS URL.';
    }
    return null;
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    await widget.onSave(
      ProviderSettings(
        baseUrl: _baseUrlController.text,
        apiKey: _apiKeyController.text,
        model: _modelController.text,
      ),
    );
  }
}
