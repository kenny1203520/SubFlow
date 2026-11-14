# Deployment configuration

## Run a published image

Production uses one versioned SubFlow image. Copy the root environment template and generate a persistent key once:

```sh
cp .env.example .env
openssl rand -hex 32
```

Put the generated value in `SUBFLOW_SETTINGS_ENCRYPTION_KEY`, protect `.env` or move the value into your deployment secret store, and keep the same key with every backup. Existing installations must keep their original key. Update `SUBFLOW_IMAGE` to the published version you intend to run. Environment files use one `KEY=value` per line; put comments on separate lines so the file works with both Compose and `docker run --env-file`.

Start the single runtime service:

```sh
docker compose --env-file .env up -d --no-build
```

The Compose file requires both `SUBFLOW_IMAGE` and `SUBFLOW_SETTINGS_ENCRYPTION_KEY`; it reports a configuration error if either is missing. Compose uses `SUBFLOW_IMAGE` to select the image and does not pass it to the application. To pull a newer published image, pull the configured tag and recreate the service with `up -d --no-build`.

The same runtime settings work with Docker directly. The image argument selects the image; the named volume matches the Compose data volume:

```sh
docker run -d --name subflow --restart unless-stopped \
  --env-file .env -p 8080:8080 \
  -v subflow_pb_data:/app/pb_data \
  ghcr.io/kenny1203520/subflow:0.2.0-beta.1
```

To build from source, run `docker build -t subflow:local .` explicitly, then set `SUBFLOW_IMAGE=subflow:local` in `.env` before starting Compose with `--no-build`. The production Compose file does not contain a build section.

After the first startup, create the initial admin account with:

```sh
docker compose exec subflow /app/subflow superuser create admin@example.com 'change-this-password'
```

## HTTPS listener and ports

SubFlow serves HTTPS only. `SUBFLOW_HTTP_PORT` (default `8080`; the historical variable name is retained) sets the container's HTTPS listener port. The HTTP redirect listener binds to loopback and is not exposed for deployment traffic. The default Compose mapping publishes the same number on the host. To publish a different host port while keeping the container on `8080`, merge an override file such as:

Use [docker-compose.host-port.example.yml](../docker-compose.host-port.example.yml) and start with both files: `docker compose --env-file .env -f docker-compose.yml -f docker-compose.host-port.example.yml up -d --no-build`. `!override` requires Docker Compose 2.24.4 or later; it replaces the original mapping instead of appending a second published port. With `docker run`, use `-p 9443:8080`.

On first startup, SubFlow creates a self-signed certificate for local development under `pb_data/.subflow/tls/`. Browsers and reverse proxies do not trust it automatically. In System administration → Settings, an administrator with system settings permission can upload a PEM certificate chain and matching private key, or reset to the generated local certificate. The key is stored with owner-only permissions under `pb_data`; uploads and resets are audited. New TLS handshakes use the selected certificate without a process restart.

The reverse proxy owns the public certificate and public path routing. It may terminate public TLS and connect to SubFlow over HTTPS. Configure it to trust the selected SubFlow upstream certificate; do not disable upstream verification in production. The PocketBase Application URL must be the public HTTPS URL and is never inferred from proxy headers. The SPA currently expects the public root path; path-prefix support requires a separate frontend change.

## PocketBase application and mail settings

Set the public **Application URL** and **Application Name** in PocketBase Admin at `/_/` → Settings → Application. Configure SMTP and the sender in PocketBase Admin → Settings → Mail. These values are stored with PocketBase data and apply without restarting SubFlow. They are not copied from retired environment variables.

Use the public HTTPS URL served by your reverse proxy, for example `https://subflow.example.com`. Do not set an internal container address. Until a valid URL and SMTP are configured, features that depend on public links or email report a configuration error.

## Process environment

- `SUBFLOW_HTTP_PORT` controls the HTTPS listener inside the container. The default Compose host mapping uses the same port; use a Compose port override or a different `docker run -p` host value to change only the host port.
- `SUBFLOW_ENV` controls development-only behavior. Deployment defaults to `production`; keep that value for production.
- `SUBFLOW_SETTINGS_ENCRYPTION_KEY` is the persistent root key for encrypted settings and derived audit/share signing keys. Generate it once, store it in the deployment secret store, back it up with PocketBase data, and never rotate or replace it casually. Existing CAPTCHA settings and encrypted share URLs depend on the original value.
- `SUBFLOW_IMAGE` selects the Compose image only. A direct `docker run` selects the image in the command.

SMTP, sender identity, application name/URL, CAPTCHA and OAuth product settings belong in PocketBase Admin. Do not restore the retired SMTP, public URL, data-driver, or frontend backend URL variables. Startup only reports the names of retired variables it finds and never prints their values.

## Persistent data and recovery

The `subflow_pb_data` named volume contains the PocketBase database, attachments, local TLS certificate state, and upgrade state. Container recreation and ordinary `docker compose down` preserve it. `docker compose down -v` deletes it permanently; use that only when intentionally resetting all application data.

SubFlow creates and verifies a local backup before applying a startup data migration. Keep the original `SUBFLOW_SETTINGS_ENCRYPTION_KEY` separately with the backup. For version ordering, migration recovery, and restoration commands, follow [PocketBase upgrade and recovery](upgrading.md).

The runtime image runs as UID/GID `65532:65532`. Fresh named volumes inherit the image's writable `/app/pb_data` directory. If upgrading a volume created by an older root-starting image, stop SubFlow and run this one-time command against the same image and volume before restarting:

```sh
docker run --rm --user 0:0 -v subflow_pb_data:/app/pb_data IMAGE init-permissions
```

The command only initializes `/app/pb_data`. Restore data into an isolated volume and verify it before changing production.
