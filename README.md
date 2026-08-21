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

Deploy and bootstrap the Household Server before you start the application.
When Balaur starts, enter these Household values:

- **Household Server address**: Use the stable HTTPS address.
- **Email address**: Use the Household Member email address.
- **Password**: Use the local Household Member password.

Balaur stores the server address and member session in secure device storage.
It does not store the member password.

A Household Administrator can select **Household**, then **Manage invitations**.
The administrator can assign a Household Member or Household Administrator role.
Balaur shows the invitation value and its Quick Response (QR) code one time.

A new Household Member can select **Use an invitation** during pairing.
The member can scan the QR code or enter the server and invitation manually.
The Household Invitation expires after 24 hours and works one time.

After pairing, select **Enter Balaur**.
Then select the settings button and enter these provider values:

- **Base URL**: Include the OpenAI-compatible API version path, such as `https://api.openai.com/v1`.
- **API key**: Use a user-provided key for development.
- **Model**: Enter the model identifier that the endpoint accepts.

The endpoint must support Chat Completions and Server-Sent Events (SSE) streaming.

## Run the Household Server

The Household Server uses a pinned PocketBase release in Docker.
See [`server/README.md`](server/README.md) for setup and test instructions.

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
