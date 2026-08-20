import 'package:balaur/settings/presentation/provider_settings_screen.dart';
import 'package:balaur/settings/provider_settings.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'provider_settings_screen.stories.g.dart';

const component = ComponentMeta(
  name: 'Model provider settings',
  path: 'Screens/Settings',
);
const meta = Meta(ProviderSettingsScreen.new);

final $Empty = _Story(
  args: _Args.fixed(settingsStore: InMemoryProviderSettingsStore()),
);

final $Configured = _Story(
  args: _Args.fixed(
    settingsStore: InMemoryProviderSettingsStore(
      const ProviderSettings(
        baseUrl: 'https://api.openai.com/v1',
        apiKey: 'example-key',
        model: 'gpt-5-mini',
      ),
    ),
  ),
);
