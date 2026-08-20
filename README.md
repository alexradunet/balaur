# Balaur

Balaur is a shared household application with an AI-first interface.

## Current chat slice

The Flutter client supports streaming chat through an OpenAI-compatible Chat Completions endpoint.

Supported targets:

- Android
- iOS
- Linux

The first slice includes:

- A custom Flutter chat interface.
- Streamed assistant text.
- Configurable base URL, API key, and model.
- Secure local storage for provider settings.
- Stop and error states.
- Replaceable chat and conversation interfaces.

Conversation messages currently stay in memory. They are lost when the application stops.

## Run the application

Install Flutter and the Linux desktop requirements. Then run this command:

```bash
flutter run -d linux
```

Select the settings button in the application. Enter these provider values:

- **Base URL**: Include the OpenAI-compatible API version path, such as `https://api.openai.com/v1`.
- **API key**: Use a user-provided key for development.
- **Model**: Enter the model identifier that the endpoint accepts.

The endpoint must support Chat Completions and Server-Sent Events (SSE) streaming.

## Develop the user interface in Widgetbook

Use Widgetbook before you integrate a reusable widget or screen.
Keep production widgets in `lib/`.
Keep Stories in the mirrored path under `widgetbook/lib/`.

Run generation in watch mode:

```bash
cd widgetbook
dart run build_runner watch
```

Run the isolated catalog in a second terminal:

```bash
cd widgetbook
flutter run -d linux
```

Validate all generated Scenarios:

```bash
cd widgetbook
dart run build_runner build
dart format lib
flutter test
```

The catalog pins Widgetbook `4.0.0-beta.11`.
The project skills define the Story-first development process.

## Verify the application

Run these commands:

```bash
flutter analyze
flutter test
flutter build linux --debug
```

## Security

A distributed client cannot fully protect a provider key. Use a backend proxy before you add a service-owned production key.
