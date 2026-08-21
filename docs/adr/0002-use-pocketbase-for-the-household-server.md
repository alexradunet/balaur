# Use PocketBase for the Household Server

The one supported Household uses one self-hosted PocketBase instance. This
choice supports native Linux and data ownership with less operational
complexity than Supabase.

## Considered options

Client-replicated SQLite was rejected for Core. It would add a sync service,
an upload interface, and conflict behavior before offline support is required.
A custom SQLite server was also rejected because it would duplicate PocketBase
authentication, access, route, migration, and administration behavior.

## Consequences

- One PocketBase instance contains one Household.
- The Household operates its endpoint, upgrades, backups, and provider keys.
- The Household provides a stable HTTPS address for every client.
- The primary distribution is a Docker image.
- Linux x64 and ARM64 archives remain supported alternatives.
- PocketBase JavaScript hooks contain server-side application logic.
- Household Members use local PocketBase credentials.
- The application never receives a PocketBase superuser token.
- A temporary setup link creates the first Household Administrator.
- Household Invitations are single-use and expire after 24 hours.
- Simple Mail Transfer Protocol configuration remains optional in Core.
- Core stores members, settings, invitations, and the Calendar Connection.
- Every authenticated Household Member can view Calendar Entries.
- Only Household Administrators can manage membership or connections.
- Clients cannot read invitation or connection credentials directly.
- Application modules keep PocketBase behind repository interfaces.
- A PocketBase snapshot supports recovery.
- The server keeps seven daily and four weekly off-host backups.
- Recovery keys and OAuth credentials stay outside `pb_data`.
- Updates are manual, versioned, backed up, checked, and reversible.
- A versioned Household Archive supports migration to another backend.
- The archive contains JSON records, member profiles, and shared files.
- The archive can include one optional iCalendar snapshot.
- Portable exports exclude password hashes and active credentials.
- Portable exports exclude private Conversations.
- Administrators confirm their password before a Household Archive export.
- Each Household Member exports their private Conversations separately.
- The Household supplies Google OAuth credentials for its Calendar Connection.
- The server encrypts the Google refresh token with an environment key.
- Household Pairing supports manual addresses and Quick Response codes.
- Clients keep the server address and member token in secure storage.
- Core has no offline persistence and sends no central telemetry.
- Google remains authoritative for Calendar Entries.
- The Household Server does not persist Calendar Entries in Core.
- A failed authorization disconnects the Calendar Connection.
- A Household Administrator must reconnect a disconnected Calendar Connection.
- Every client uses one configured Household Time Zone.
- PocketBase versions stay pinned until tested migrations are available.

## First delivery

The first complete slice deploys the Household Server, creates an
administrator, invites a member, pairs clients, connects Google Calendar,
displays Calendar Entries, and exports a Household Archive.

The calendar displays titles, times, all-day status, locations, and colors.
It uses the system browser for authorization. It provides month navigation,
day selection, refresh, connection status, and all meaningful visible states.

Automated checks use fake Google adapters. Completion requires unit,
repository, route, widget, Widgetbook Scenario, layout, analysis, test, and
Linux build checks.

## Deferred

- Calendar Entry creation and editing.
- Multiple Calendar Sources.
- Persistent Calendar Entry caching.
- Offline support.
- Required email delivery and multi-factor authentication.
- Conversation persistence.
- Automatic server updates.
- Central telemetry.
- Multiple Households.
