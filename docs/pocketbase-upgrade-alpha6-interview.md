# PocketBase upgrade to alpha.6: decision interview record

Date: 2026-10-28  
Source: the implementation plan approved by the user, plus the locale clarification in this task.

## Goal

Ship SubFlow `0.2.0-alpha.6` with PocketBase `v0.41.0` and a durable, versioned upgrade path that lets future releases migrate a single local data volume automatically. The application must back up and verify local data before migration, remain unavailable when an upgrade fails, and provide a verified restore path using a compatible older executable.

## Confirmed decisions

- Upgrade the Go PocketBase module and imports to the upstream `ganigeorgiev.com` module path and pin PocketBase `v0.41.0`.
- Treat alpha.1–alpha.5 databases as legacy data. Identify them from schema fingerprints rather than guessing an application version, and verify upgrades from immutable historical images.
- Run exactly one SubFlow writer per data volume. The deployer stops the old process and replaces the executable or image; SubFlow owns backup, migration, verification, and readiness.
- Keep migrations ordered, checksummed, auditable, and immutable after release. Run data changes, verification, audit, and migration completion in one transaction; future large backfills use durable batches and checkpoints.
- Keep backups, state, event history, and restore journals outside recursive backup copies. Validate checksums before restoring. A restore recovers data only; operators also use the matching older binary and original encryption key.
- Preserve the v1 client protocol. Treat an omitted protocol header from existing alpha clients as v1, pause writes/SSE/outbox during maintenance, and keep form content and queued data intact. The browser exposes status only; administrator history is read-only.
- Keep the current Traditional Chinese and English locales. The user confirmed: “沿用目前繁中、英文”. Do not add a third locale in alpha.6.
- Require Go, frontend, CodeQL, cross-platform migration checks, browser projects, and container upgrade/restore qualification in the CI/release path. Do not publish an alpha.6 tag or image as part of this implementation; leave publication for acceptance.

## Operational boundary

Automatic backups cover local PocketBase files and attachments. Operators separately protect external object storage and the encryption key. A previously loaded alpha.1–alpha.5 JavaScript client cannot display the new maintenance UI until it is reloaded; alpha.6 and later coordinate frontend and backend maintenance state.

## Open decisions

None. Technical validation that depends on local services (Docker daemon and CodeQL execution) is tracked in the implementation/test results, not as a product decision.
