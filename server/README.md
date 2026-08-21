# Household Server

The Household Server uses PocketBase `0.39.11` for one self-hosted Household.
It stores Household data in one `pb_data` directory.
It sends no central Balaur telemetry.

## Supported artifacts

The versioned Docker image is the primary artifact.
The release process also creates Linux x64 and ARM64 archives.
All artifacts pin the PocketBase version and include checksums.

Create all release artifacts with:

```bash
server/scripts/package-release.sh dist
```

This command needs Docker Buildx for the multi-platform Open Container Initiative image archive.
Use `server/scripts/package-server.sh dist amd64` to create only the Linux x64 archive.

## Interactive setup

Run the repeatable setup wizard:

```bash
server/scripts/setup-wizard.sh
```

The wizard guides these human tasks:

1. Select server paths and the stable HTTPS address.
2. Create the Google Cloud project and enable Google Calendar.
3. Configure the private OAuth testing audience.
4. Create the web OAuth client and exact callback.
5. Create the unique calendar encryption key.
6. Configure Domain Name System records and Caddy.
7. Deploy and check the Household Server.
8. Create the first Household Administrator.
9. Prove backup restoration.

The wizard automates safe local steps.
It keeps secrets in `server/.env` with restricted permissions.

## Complete Household workflow

Use this order for the complete supported workflow:

1. Run the wizard to configure HTTPS, Google OAuth, secrets, and deployment.
2. Open the setup link and create the first Household Administrator.
3. Pair Balaur with the stable Household Server address and local credentials.
4. Open Household, then create a Household Invitation for each additional member.
5. Open Household, then connect Google and select one shared calendar.
6. Open Calendar to browse months, select days, and refresh Calendar Entries.
7. Open Household, then export and save a confirmed Household Archive.
8. Run the off-host backup and restore proof.

Debug builds also permit `http://localhost:8090`.
Profile and release builds continue to require HTTPS.

## Manual Docker start

Copy `.env.example` to `.env`.
Set every deployment value before the first start.

```bash
cd server
mkdir -p pb_data
docker compose up --build --detach
docker compose logs --follow household-server
```

The log shows a line that starts with `Household setup link:`.
Open this one-time link to create the first Household Administrator.
The link expires after 30 minutes.

Do not set `BALAUR_SETUP_SECRET` for a normal deployment.
The container creates a random setup secret for each incomplete startup.

## Required configuration

Set `BALAUR_PUBLIC_URL` to the stable Hypertext Transfer Protocol Secure address.
Use a Google web OAuth client.
Set its exact authorized redirect Uniform Resource Identifier to:

```text
https://your-household-server.example/api/balaur/calendar-source/callback
```

Set these Google values:

- `BALAUR_GOOGLE_OAUTH_CLIENT_ID`
- `BALAUR_GOOGLE_OAUTH_CLIENT_SECRET`

Create one unique 32-character encryption key:

```bash
openssl rand -base64 24
```

Set the result as `BALAUR_CALENDAR_ENCRYPTION_KEY`.
Keep the key and OAuth secret outside `pb_data`.
Never commit `.env`.

## Caddy and HTTPS

Copy `Caddyfile.example` to the Caddy host configuration.
Replace `household.example.com` with the Household domain.
Point the domain records to the host before Caddy starts.
Caddy obtains and renews the HTTPS certificate.

The Household Server listens on loopback or the private container network.
Clients use only the stable HTTPS address.

## Health check

Use the PocketBase health interface:

```bash
curl --fail https://your-household-server.example/api/health
```

Docker also checks `/api/health` inside the container.

## Application and operational accounts

A PocketBase superuser is an operational account for the PocketBase dashboard.
A Household Administrator is an application member in the `members` collection.
Never give a PocketBase superuser credential to Balaur.

## Versioned schema and routes

PocketBase runs `pb_migrations` when it starts.
The migrations create these private record groups:

- `members`
- `household_settings`
- `household_invitations`
- `calendar_connections`

Collection rules block direct reads of invitation and connection secrets.
Custom routes return only stable Household models.

Household Invitation routes:

- `GET /api/balaur/household-invitations`
- `POST /api/balaur/household-invitations`
- `POST /api/balaur/household-invitations/{id}/cancel`
- `POST /api/balaur/household-invitations/redeem`

Calendar routes:

- `GET /api/balaur/calendar-connection`
- `POST /api/balaur/calendar-connection/authorize`
- `POST /api/balaur/calendar-connection/select`
- `POST /api/balaur/calendar-connection/disconnect`
- `GET /api/balaur/calendar-entries`
- `GET /api/balaur/calendar-source/callback`

Google authorization requests only the read-only Calendar List and Calendar Entry scopes.
The server verifies OAuth state before it exchanges an authorization code.
The server encrypts each refresh token with AES-256-GCM.
Calendar Entries stay in client memory and do not persist in PocketBase.

Household Archive route:

- `POST /api/balaur/household-archive`

The route requires a Household Administrator and immediate password confirmation.
The archive contains a versioned manifest, Household State, member profiles, and shared files.
It can include an iCalendar snapshot.
It excludes passwords, hashes, setup secrets, tokens, OAuth credentials, encryption keys, and private Conversations.

## Persistent data

Docker Compose bind-mounts `BALAUR_DATA_DIR` at `/pb/pb_data`.
The default directory is `server/pb_data`.
This directory contains only PocketBase data.

Stop the container without deleting data:

```bash
docker compose down
```

Never delete `pb_data` unless a verified restore is available.

## Off-host backups

Set `BALAUR_BACKUP_DESTINATION` to an off-host mounted filesystem.
The backup script rejects the same filesystem by default.
It keeps seven daily archives and four weekly archives.
Each archive has a SHA-256 checksum.

Stop PocketBase before each filesystem snapshot:

```bash
cd server
docker compose stop
backup=$(scripts/backup.sh "${BALAUR_DATA_DIR:-./pb_data}" "$BALAUR_BACKUP_DESTINATION")
docker compose start
printf '%s\n' "$backup"
```

Schedule this sequence daily with the host scheduler.
Sunday runs also create the weekly retention copy.
Backups exclude `.env`, recovery keys, and OAuth credentials because these files stay outside `pb_data`.

## Restore proof

Restore into an empty directory:

```bash
server/scripts/restore.sh /off-host/daily-2026-08-21.tar.gz /restore/pb_data
```

Prove the restored server can start:

```bash
pocketbase=$(server/scripts/download-pocketbase.sh)
server/scripts/verify-restore.sh /off-host/daily-2026-08-21.tar.gz "$pocketbase"
```

The proof starts the restored Household Server and checks `/api/health`.
A backup has recovery value only after this proof passes.

## Manual updates and rollback

Automatic container updates are not supported.
Use an explicit versioned image and run:

```bash
cd server
BALAUR_BACKUP_DESTINATION=/mnt/off-host/balaur \
  scripts/update.sh ghcr.io/example/balaur-household-server:0.40.0
```

The script performs this sequence:

1. Stop the current server.
2. Create an off-host backup.
3. Pull and start the new version.
4. Check the health interface.
5. Persist the new image after success.
6. Restart and persist the previous image after failure.

Review PocketBase migrations before each update.
Keep the previous image until the restore proof and application checks pass.

## Safe structured logs

The request log contains only the method, path, and duration.
It excludes request bodies, query values, credentials, and Calendar Entry details.
Do not add provider responses or archive contents to logs.

## Validation

Run the real PocketBase route tests:

```bash
server/scripts/test.sh
```

Run backup, restore, health failure, and rollback checks:

```bash
server/scripts/test-operations.sh
```

Run native archive and credential checks:

```bash
server/scripts/test-packaging.sh
```

These tests use the fake Calendar Source.
They do not require live Google credentials or production network requests.
