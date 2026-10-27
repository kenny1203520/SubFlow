# Adding PocketBase migrations

The alpha.1–alpha.5 compatibility baseline is an immutable migration. Never
change its behavior or checksum after alpha.6 has shipped. In particular,
schema additions and data backfills must not be added to the legacy bootstrap
implementation.

For each schema change, append a new migration with a permanent, ordered ID,
static source checksum, explicit prerequisite IDs, preconditions, transactional
data changes, a verification function, and a successful audit event in the
same transaction. Keep each released migration's implementation immutable;
fix a mistake by appending a new migration. Large backfills must commit bounded
batches and their checkpoints together. Add an upgrade test that starts from
the previous migration and proves both the result and restart idempotence.

Use `Migration.Up` only when the complete change fits one transaction. For a
large backfill, provide `BatchUp` and a positive `BatchSize`. Each callback
receives the cursor from the last committed batch and must touch at most that
many records. Return a new stable cursor with `done=false` to commit a batch;
the runner stores it in the same transaction as those data changes. After a
restart, the callback receives that cursor and continues. Return `done=true`
when no records remain; verification, the success audit event, completion
journal, and checkpoint removal then commit atomically with that final batch.
Do not put network calls or other external side effects in a batch callback.
A failed callback, verification, or audit rolls back the current batch and its
cursor, while earlier committed batches remain available for retry. Changing
the checksum with an unfinished checkpoint is rejected.

Run the migration contract check locally with:

```sh
node scripts/migrations/verify.mjs
```

CI checks the frozen baseline hash, unique and ordered migration IDs, static
checksums, and requires a newly registered migration if protected schema
sources change. Test lock and filesystem behavior on Linux, Windows, and macOS.
