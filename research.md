# Research: Display-only Google Calendar integration

## Summary

Use a web-first minimum viable product (MVP) during Linux development. The official Flutter sign-in stack supports web, Android, iOS, and macOS, but not Linux or Windows.

Each Household Member must sign in with the Google Account that has calendar access. The app must request only `calendar.events.readonly` and query the configured shared calendar identifier.

## Findings

### 1. Platform authorization and package support

Use OAuth 2.0 user authorization. A service account is not necessary because Google Calendar already grants access to each user.

| Target | Correct flow | Official Flutter package approach | Finding |
|---|---|---|---|
| Android | Use the Google Sign-In software development kit (SDK) through its system user interface. Configure an Android OAuth client for the package and signing certificate. | Use `google_sign_in`, its endorsed Android implementation, `extension_google_sign_in_as_googleapis_auth`, and `googleapis`. | Supported. The Android setup can also require a web client identifier as `serverClientId`. [Android package documentation](https://pub.dev/documentation/google_sign_in_android/latest/) |
| iOS | Use the Google Sign-In SDK and its registered return URL scheme. Configure an iOS OAuth client for the bundle identifier. | Use `google_sign_in`, its endorsed iOS implementation, the auth-client extension, and `googleapis`. | Supported. The reversed client identifier URL scheme remains necessary. [iOS package documentation](https://pub.dev/packages/google_sign_in_ios) |
| macOS | Use the endorsed Apple implementation and its registered return URL scheme. Configure Keychain Sharing. | Use the same Flutter package stack. The package uses `google_sign_in_ios` on macOS. | Supported on macOS 10.15 and later. [Flutter package](https://pub.dev/packages/google_sign_in) [Apple implementation](https://pub.dev/packages/google_sign_in_ios) |
| Web | Use the Google Identity Services token model through the plugin. Request authorization from a user action. | Use `google_sign_in_web` through `google_sign_in`, then use the auth-client extension and `googleapis`. | Supported. The client-side token model does not provide a refresh token. [Web package documentation](https://pub.dev/documentation/google_sign_in_web/latest/) [Google token model](https://developers.google.com/identity/oauth2/web/guides/use-token-model) |
| Linux | Use an installed desktop application flow in the system browser. Use Authorization Code, Proof Key for Code Exchange (PKCE), and a loopback redirect. | No official supported Flutter sign-in stack exists. `google_sign_in` does not support Linux. | A pure-Dart prototype can use `googleapis_auth` with a desktop client. However, its official README says not to use it directly in Flutter. [Flutter package](https://pub.dev/packages/google_sign_in) [Auth README](https://github.com/google/googleapis.dart/blob/master/googleapis_auth/README.md) |
| Windows | Use the same installed desktop application flow as Linux. | No official supported Flutter sign-in stack exists. `google_sign_in` does not support Windows. | The same pure-Dart prototype limitation applies. [Flutter package](https://pub.dev/packages/google_sign_in) [Auth README](https://github.com/google/googleapis.dart/blob/master/googleapis_auth/README.md) |

Google permits loopback redirects for Desktop app clients on macOS, Linux, and Windows. Use the system browser and `http://127.0.0.1:<random-port>` when implementing this flow directly. Do not use the blocked out-of-band copy-and-paste flow. [Google loopback guide](https://developers.google.com/identity/protocols/oauth2/resources/loopback-migration) [Google native application guide](https://developers.google.com/identity/protocols/oauth2/native-app)

The current `googleapis_auth` desktop flow starts a local server and implements PKCE with `S256`. Its current source binds to `localhost`, so it does not use Google's preferred explicit IPv4 address. [Auth flow source](https://raw.githubusercontent.com/google/googleapis.dart/master/googleapis_auth/lib/src/oauth2_flows/authorization_code_grant_server_flow.dart) [PKCE source](https://raw.githubusercontent.com/google/googleapis.dart/master/googleapis_auth/lib/src/oauth2_flows/auth_code.dart)

The generated `googleapis` package contains `CalendarApi` for Calendar API v3. The Flutter extension converts a `google_sign_in` authorization into the `AuthClient` that `CalendarApi` needs. [Calendar Dart library](https://pub.dev/documentation/googleapis/latest/calendar_v3/) [Flutter auth-client extension](https://pub.dev/packages/extension_google_sign_in_as_googleapis_auth)

### 2. Least-privilege authorization

Request this scope only:

```text
https://www.googleapis.com/auth/calendar.events.readonly
```

This scope permits read-only event access. It does not grant calendar settings, sharing-rule management, or event writes. [Google Calendar scopes](https://developers.google.com/workspace/calendar/api/auth)

OAuth scopes do not limit access to one calendar. The calendar access control list (ACL) limits which shared calendars and event details each user can read. A `reader` can read events, but private-event details remain hidden. [Calendar sharing](https://developers.google.com/workspace/calendar/api/concepts/sharing) [ACL reference](https://developers.google.com/workspace/calendar/api/v3/reference/acl)

Configure the shared calendar identifier instead of adding calendar discovery to the MVP. Discovery needs the additional `calendar.calendarlist.readonly` scope. [Google Calendar scopes](https://developers.google.com/workspace/calendar/api/auth)

### 3. Household Member access

Use this access model:

1. The calendar owner shares the calendar with each Household Member's Google Account.
2. The owner grants at least the `reader` ACL role.
3. Each Household Member signs in to the app with that same Google Account.
4. Each member grants the application the read-only event scope.
5. The app calls `events.list` with the same configured calendar identifier.

The API applies each authenticated user's existing ACL rights. The application must not share one person's refresh token with other Household Members. [Calendar sharing](https://developers.google.com/workspace/calendar/api/concepts/sharing) [OAuth security practices](https://developers.google.com/identity/protocols/oauth2/resources/best-practices)

An ACL share does not always add the calendar to the recipient's Calendar List. A fixed calendar identifier avoids that dependency. If selection becomes necessary, use `calendarList.list` and read each entry's `id` and `accessRole`. [Calendar sharing](https://developers.google.com/workspace/calendar/api/concepts/sharing) [CalendarList resource](https://developers.google.com/workspace/calendar/api/v3/reference/calendarList)

### 4. Exact bounded month request

For a March 2026 view in `Europe/Bucharest`, send this request:

```http
GET https://www.googleapis.com/calendar/v3/calendars/{URL_ENCODED_CALENDAR_ID}/events
  ?timeMin=2026-03-01T00%3A00%3A00%2B02%3A00
  &timeMax=2026-04-01T00%3A00%3A00%2B03%3A00
  &singleEvents=true
  &orderBy=startTime
  &showDeleted=false
  &timeZone=Europe%2FBucharest
  &maxResults=2500
```

Create both bounds as local midnight in the selected IANA time zone. Encode the actual offset at each boundary because daylight-saving changes can make the offsets different.

`timeMin` is an exclusive lower bound on an event's end. `timeMax` is an exclusive upper bound on an event's start. This pair returns events that intersect the month. Both values require an RFC 3339 time-zone offset. [Events: list](https://developers.google.com/workspace/calendar/api/v3/reference/events/list)

Use these parameters without changes on every page:

- `singleEvents=true` expands recurring series into instances.
- `orderBy=startTime` sorts the instances by start time. This order requires `singleEvents=true`.
- `showDeleted=false` removes cancelled events from the display result.
- `timeZone=Europe/Bucharest` selects the response time zone. It does not replace offsets in the bounds.
- `maxResults=2500` requests the documented maximum page size. The server can still return fewer items.

If the response has `nextPageToken`, repeat the same request with `pageToken=<nextPageToken>`. Continue until the response has no `nextPageToken`. [Events: list](https://developers.google.com/workspace/calendar/api/v3/reference/events/list)

If later synchronization needs deletion tombstones, use `showDeleted=true`. With `singleEvents=true`, Google returns cancelled individual instances, not the underlying recurring event. Cancelled resources can omit normal event fields, so process `status == "cancelled"` before reading start data. [Events: list](https://developers.google.com/workspace/calendar/api/v3/reference/events/list) [Event resource](https://developers.google.com/workspace/calendar/api/v3/reference/events)

Read event starts as follows:

- A timed event uses `start.dateTime`, which is an RFC 3339 timestamp.
- An all-day event uses `start.date`, which is a `yyyy-mm-dd` calendar date.
- Do not convert an all-day `date` through Coordinated Universal Time (UTC).
- An event end is exclusive. For one all-day event on March 10, `end.date` is `2026-03-11`.

The resource defines `date` and `dateTime` as alternative fields. It also defines the exclusive end rule. [Event resource](https://developers.google.com/workspace/calendar/api/v3/reference/events)

### 5. Token and client security

Use separate OAuth client identifiers for each application type. Configure Android package signatures, Apple bundle identifiers and URL schemes, web origins, and a Desktop app client separately. [Google OAuth overview](https://developers.google.com/identity/protocols/oauth2) [Google Sign-In practices](https://developers.google.com/identity/siwg/best-practices)

Apply these controls:

- Use PKCE and a unique `state` value for a direct desktop flow.
- Open the system browser. Do not embed the Google authorization page.
- Store refresh tokens securely at rest.
- Revoke and delete tokens when the user disconnects the integration.
- Never log access tokens, refresh tokens, authorization codes, or client secrets.
- Never commit credentials or generated client configuration with sensitive values.
- Use HTTPS for production web origins.
- Use separate Google Cloud projects for testing and production.

Google defines installed applications as public clients. A desktop client secret distributed in the application cannot remain confidential, so it must not act as an application security boundary. [Google native application guide](https://developers.google.com/identity/protocols/oauth2/native-app) [OAuth security practices](https://developers.google.com/identity/protocols/oauth2/resources/best-practices) [OAuth policies](https://developers.google.com/identity/protocols/oauth2/policies)

The repository already declares `flutter_secure_storage` 11.0.0. Its package supports Android, iOS, web, macOS, Linux, and Windows, with platform-specific setup. Linux requires `libsecret` and a Secret Service provider. Web storage requires HTTPS or `localhost` and remains tied to one browser and origin. [Secure storage package](https://pub.dev/packages/flutter_secure_storage)

For the web MVP, keep the access token in memory and request authorization again when necessary. The Google web token model does not issue a refresh token. Use a backend authorization-code model only if later requirements need background or offline access. [Google token model](https://developers.google.com/identity/oauth2/web/guides/use-token-model) [Authorization model choice](https://developers.google.com/identity/oauth2/web/guides/choose-authorization-model)

### 6. Pareto MVP recommendation

Implement the MVP as Flutter web and develop it on Linux. This path uses the official Flutter package stack and avoids unsupported native Linux authentication.

The MVP should include only these items:

1. Configure one shared calendar identifier and one IANA display time zone.
2. Use one Google Account sign-in per Household Member.
3. Request only `calendar.events.readonly` after a user action.
4. Fetch the visible month with the exact request above.
5. Follow all result pages.
6. Expand recurring events and hide cancelled events.
7. Render timed and all-day values with separate parsing rules.
8. Keep web access tokens in memory.
9. Use allowlisted OAuth test users during private MVP testing.

An external OAuth app in Testing limits access to listed test users. Calendar authorization can also require verification before a public production release. [OAuth app states](https://developers.google.com/identity/protocols/oauth2/production-readiness/overview) [OAuth overview](https://developers.google.com/identity/protocols/oauth2)

Defer these items explicitly:

- Native Linux and Windows releases.
- A custom desktop OAuth implementation or a backend token broker.
- Offline access, background synchronization, and refresh-token persistence.
- Calendar discovery and a calendar picker.
- Incremental synchronization with `syncToken`.
- Push notification channels.
- Event writes and ACL management.
- Persistent event caching.
- Public OAuth verification and production rollout work.

After the web MVP is stable, add Android, iOS, and macOS with the same official Flutter stack. Review Linux and Windows again before implementation because no official Flutter sign-in implementation supports them today. [Flutter Google Sign-In package](https://pub.dev/packages/google_sign_in)

## Sources

### Kept

- [Events: list](https://developers.google.com/workspace/calendar/api/v3/reference/events/list) — Defines all query and pagination behavior.
- [Event resource](https://developers.google.com/workspace/calendar/api/v3/reference/events) — Defines event status and date values.
- [Google Calendar scopes](https://developers.google.com/workspace/calendar/api/auth) — Defines least-privilege scopes.
- [Calendar sharing](https://developers.google.com/workspace/calendar/api/concepts/sharing) — Defines shared-user access and Calendar List behavior.
- [Google native application guide](https://developers.google.com/identity/protocols/oauth2/native-app) — Defines installed application security.
- [Google OAuth security practices](https://developers.google.com/identity/protocols/oauth2/resources/best-practices) — Defines token controls.
- [Flutter Google Sign-In package](https://pub.dev/packages/google_sign_in) — Defines the official platform matrix.
- [Flutter auth-client extension](https://pub.dev/packages/extension_google_sign_in_as_googleapis_auth) — Defines the official Flutter-to-Google-API bridge.
- [Dart Google API auth README](https://github.com/google/googleapis.dart/blob/master/googleapis_auth/README.md) — Defines the Flutter limitation.
- [Dart Calendar API library](https://pub.dev/documentation/googleapis/latest/calendar_v3/) — Defines the generated Dart client.

### Dropped

- Community cross-platform sign-in packages — The task permits only official package documentation or source.
- Blog posts and tutorials — They are not primary sources.
- Service-account guides — They do not match per-member access to an already shared calendar.

## Gaps and risks

- Google does not provide an official Flutter sign-in implementation for Linux or Windows.
- A native desktop release needs a reviewed custom flow, a backend design, or later official package support.
- The final ACL role and private-event visibility depend on the owner's current Google Calendar sharing configuration.
- The exact production verification work depends on the OAuth audience and release model.
