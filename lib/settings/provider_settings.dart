class ProviderSettings {
  const ProviderSettings({
    required this.baseUrl,
    required this.apiKey,
    required this.model,
  });

  const ProviderSettings.empty()
    : baseUrl = 'https://api.openai.com/v1',
      apiKey = '',
      model = '';

  final String baseUrl;
  final String apiKey;
  final String model;

  bool get isConfigured =>
      baseUrl.trim().isNotEmpty &&
      apiKey.trim().isNotEmpty &&
      model.trim().isNotEmpty;

  String get normalizedBaseUrl =>
      baseUrl.trim().replaceFirst(RegExp(r'/$'), '');
}
