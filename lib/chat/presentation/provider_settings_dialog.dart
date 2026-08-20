import 'package:balaur/settings/provider_settings.dart';
import 'package:flutter/material.dart';

class ProviderSettingsDialog extends StatefulWidget {
  const ProviderSettingsDialog({super.key, required this.initialSettings});

  final ProviderSettings initialSettings;

  static Future<ProviderSettings?> show(
    BuildContext context,
    ProviderSettings settings,
  ) {
    return showDialog<ProviderSettings>(
      context: context,
      builder: (context) => ProviderSettingsDialog(initialSettings: settings),
    );
  }

  @override
  State<ProviderSettingsDialog> createState() => _ProviderSettingsDialogState();
}

class _ProviderSettingsDialogState extends State<ProviderSettingsDialog> {
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
    return AlertDialog(
      title: const Text('Model provider'),
      content: SizedBox(
        width: 520,
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
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
              const SizedBox(height: 12),
              TextFormField(
                key: const Key('provider-api-key'),
                controller: _apiKeyController,
                decoration: InputDecoration(
                  labelText: 'API key',
                  suffixIcon: IconButton(
                    onPressed: () => setState(() {
                      _hideApiKey = !_hideApiKey;
                    }),
                    tooltip: _hideApiKey ? 'Show API key' : 'Hide API key',
                    icon: Icon(
                      _hideApiKey ? Icons.visibility : Icons.visibility_off,
                    ),
                  ),
                ),
                obscureText: _hideApiKey,
                validator: _required,
              ),
              const SizedBox(height: 12),
              TextFormField(
                key: const Key('provider-model'),
                controller: _modelController,
                decoration: const InputDecoration(
                  labelText: 'Model',
                  hintText: 'gpt-4o-mini',
                ),
                validator: _required,
              ),
              const SizedBox(height: 16),
              const Text(
                'Balaur stores these settings in secure storage on this device.',
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Cancel'),
        ),
        FilledButton(onPressed: _save, child: const Text('Save')),
      ],
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

  void _save() {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    Navigator.pop(
      context,
      ProviderSettings(
        baseUrl: _baseUrlController.text,
        apiKey: _apiKeyController.text,
        model: _modelController.text,
      ),
    );
  }
}
