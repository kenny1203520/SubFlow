# PocketBase upgrade and recovery

SubFlow 0.2.0-alpha.6 and later back up and migrate the local PocketBase data
volume during startup. Run one SubFlow process per data volume. Stop the old
process before replacing its image or executable; do not start an older binary
against a database opened by a newer PocketBase version.

The coordinator stores state and durable events in pb_data/.subflow and
new upgrade snapshots in pb_data/.subflow-recovery. Old snapshots remain
readable. Upgrade copies include SQLite data and WAL, local attachments and
TLS. PocketBase backup archives and runtime work files are excluded.
Successful upgrades retain zero snapshots by default after verification and
durable audit. Failed upgrades preserve the verified snapshot needed for
recovery. Cleanup errors are logged and retried at startup.

PocketBase keeps its configured cron, retention and local or S3 destination.
New archives include TLS but exclude SubFlow upgrade and restore work files.
Keep the original encryption key and back up external attachments separately.

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

For an older daily ZIP that contains nested upgrade snapshots, stop SubFlow
and mount the source ZIP read-only at a separate path. Download S3 archives
to a different filesystem first if space on the data disk is limited.

    subflow upgrade restore --pocketbase-zip /recovery/daily.zip --source-version=0.2.0-beta.1 --dir=/app/pb_data

The filtered restore preserves prior daily archives and the source ZIP. Rehearse
against an isolated copy and verify the restored SQLite records, attachment
bytes, TLS and the original encryption key before starting a service. An
older binary may reject the restore marker; use its established offline copy
restore workflow on a separate volume when the exact older binary is needed.

On Rocky Linux, stop the service and take a separate offline backup before
upgrading. If the filesystem is full, make space on the original filesystem
under operator control after verifying independent copies. Preserve the only
verified recovery snapshot. Reuse the existing mount, UID/GID and SELinux label.
Verify readiness, stored data and attachments, TLS, audit and the next scheduled
archive. Inspect the ZIP directory to confirm neither recovery location appears.

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
