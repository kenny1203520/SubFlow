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

Inspect status while the service is stopped:

```sh
subflow upgrade status
```

A failed startup remains in maintenance mode. After checking the safe problem
code and fixing the underlying cause, retry the same candidate:

```sh
subflow upgrade retry
```

To restore local data from a verified backup:

```sh
subflow upgrade restore --backup <backup-id>
```

Restore keeps the failed data tree under
`.subflow/restore-failed-<backup-id>`. It restores only local data. Start the
matching older SubFlow and PocketBase executable and use the original
encryption key. Do not open the restored volume with the newer executable.

CI starts isolated data sets using each immutable alpha.1–alpha.5 container
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
