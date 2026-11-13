# PocketBase upgrade and recovery

SubFlow 0.2.0-alpha.6 and later back up and migrate the local PocketBase data
volume during startup. Run one SubFlow process per data volume. Stop the old
process before replacing its image or executable; do not start an older binary
against a database opened by a newer PocketBase version.

The coordinator stores upgrade state, its durable event log, and verified
backups in `<data-dir>/.subflow`. These control files are excluded from backup
copies. A backup contains `data.db`, SQLite WAL files, local attachments, a
relative file inventory, and a SHA-256 digest for each copied file. The three
newest successful upgrade backups are retained; pending and failed backups are
never pruned. External object storage and `SUBFLOW_SETTINGS_ENCRYPTION_KEY`
must be backed up separately.

Inspect progress or diagnose a failed startup from a second terminal:

```sh
subflow upgrade status --state-only
subflow upgrade preflight --dir=pb_data
```

`status --state-only` reads the last saved phase, timestamp and failure without
rehashing backups. Use plain `status` to list verified backups. Preflight
reports the recorded data version, executable version and known blockers as
JSON. A blocked preflight exits nonzero and leaves the volume untouched.
Compare the last timestamp with the container logs to see the current phase
and its elapsed time.

A failed startup remains in maintenance mode. After fixing the cause, stop
SubFlow and take an offline snapshot of the entire data directory, including
`.subflow`. Keep the original encryption key. With the same Compose project
and its production env file, run:

```sh
sudo docker compose stop subflow
# Take the offline snapshot here before retrying.
sudo docker compose run --rm --no-deps subflow upgrade retry --dir=/app/pb_data
# Start only after retry prints completion and exits successfully.
sudo docker compose up -d --no-build subflow
sudo docker compose ps subflow
```

Use `sudo` on Rocky if your user cannot access the Docker socket. To run
preflight against the configured service volume, use
`sudo docker compose run --rm --no-deps subflow upgrade preflight --dir=/app/pb_data`.
While retry runs, get the one-off container name with `sudo docker compose ps --all`.
Read its phase with
`sudo docker exec <retry-container> /app/subflow upgrade status --state-only --dir=/app/pb_data`
or read its output with `sudo docker logs <retry-container>`. After startup,
wait for `healthy` and verify that the public `/api/ready` responds with HTTP 200.

For a deployment created with `docker run`, stop the original container and
reuse the env file and exact PocketBase mount. This example assumes the stopped
container named `subflow` already uses beta.2:

```sh
sudo docker stop subflow
# Take the offline snapshot here before retrying.
sudo docker run --rm --env-file .env --volumes-from subflow \
  ghcr.io/kenny1203520/subflow:0.2.0-beta.2 upgrade retry --dir=/app/pb_data
sudo docker start subflow
```

To restore local data from a verified backup:

```sh
subflow upgrade restore --backup <backup-id>
```

Restore keeps the failed data tree under
`.subflow/restore-failed-<backup-id>`. It restores only local data. Start the
matching older SubFlow and PocketBase executable and use the original
encryption key. Do not open the restored volume with the newer executable.

CI starts isolated data sets using each immutable alpha.1–alpha.5, beta, and beta.1 container
digest in `scripts/release/upgrade-fixtures.json`, then verifies migration,
restart idempotence, backup restoration, and opening the restored copy with its
matching old image.

An already loaded alpha.1–alpha.5 browser client cannot acquire the maintenance
screen until it is reloaded. Starting with alpha.6, the client pauses API,
SSE, and outbox synchronization during maintenance while keeping local forms
and queued writes. If the server requires a newer client protocol, preserve
unsaved form content before using the explicit reload action.

The runtime status endpoint exposes only the current phase, readiness, and
safe problem codes. Version information is restricted to the administrator's
upgrade history endpoint. `/api/live` reports process liveness;
`/api/ready` and `/api/health` report business readiness. Container health
checks use readiness and allow a long initial migration window.
