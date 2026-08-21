# Household Server

This directory contains the first deployable Household Server slice.
It runs PocketBase `0.39.11` and stores data in one persistent directory.

## Start the server

Install Docker with Docker Compose.
Then run these commands:

```bash
cd server
docker compose up --build --detach
docker compose logs --follow household-server
```

The log shows a line that starts with `Household setup link:`.
Open this link to create the first Household Administrator.
The link expires after 30 minutes and works one time.

A restart creates a new link if Household setup is not complete.
The server does not show a Household setup link after setup is complete.

For a remote server, copy `.env.example` to `.env`.
Set `BALAUR_PUBLIC_URL` to the stable Hypertext Transfer Protocol Secure
(HTTPS) address before you start the server.
The `.env` file is not in Git.

Do not set `BALAUR_SETUP_SECRET` for a normal deployment.
The container creates a random setup secret for each incomplete startup.

## Check server health

Use the PocketBase health interface:

```bash
curl --fail http://localhost:8090/api/health
```

Docker also checks `/api/health` inside the container.

## Keep application accounts separate

PocketBase can also show a PocketBase superuser installer link.
This link is not the Household setup link.

A PocketBase superuser is an operational account for the PocketBase dashboard.
A Household Administrator is an application member in the `members`
collection.
Never give a PocketBase superuser credential to the Balaur application.

## Persistent data

Docker Compose mounts `household-data` at `/pb/pb_data`.
This path is the only persistent data directory for this slice.

Stop the container without deleting the volume:

```bash
docker compose down
```

Do not use `docker compose down --volumes` unless you intend to delete all
Household data.

## Versioned schema

PocketBase runs the files in `pb_migrations` when it starts.
The first migration creates these records:

- The `members` authentication collection.
- The singleton `household_settings` collection and record.

Clients cannot create or change either record type through collection routes.
The setup route creates the first Household Administrator in one transaction.

## Run the route integration test

Run this command from the repository root:

```bash
server/scripts/test.sh
```

The script downloads the pinned PocketBase release.
It verifies the release checksum before it runs the binary.
The test starts real PocketBase processes and uses temporary data.
It disables the PocketBase superuser installer during these processes.
It does not open a browser or create a PocketBase superuser.
It covers expiration, invalid setup, success, replay, and a second setup.

## Update PocketBase

`pocketbase-version.env` pins the release and Linux archive checksums.
The Docker image supports Linux x64 and ARM64 targets.

For an update, change the version and both checksums together.
Also change the image tag in `compose.yaml`.
Run the route integration test and the Docker health check before release.
