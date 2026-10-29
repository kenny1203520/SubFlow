# Deployment configuration

## PocketBase application and mail settings

Set the public **Application URL** and **Application Name** in PocketBase Admin at `/_/` → Settings → Application. Configure SMTP and the sender in PocketBase Admin → Settings → Mail. These values are stored with PocketBase data and apply without restarting SubFlow. They are not copied from retired environment variables.

Use the public HTTPS URL served by your reverse proxy, for example `https://subflow.example.com`. Do not set an internal container address. The backend validates this URL and never derives it from the request `Host` or forwarded headers. Until a valid URL is configured, the admin UI still starts, while invitation and other URL-dependent actions return a configuration error.

## HTTPS listener and server certificate

SubFlow serves HTTPS only. `SUBFLOW_HTTP_PORT` (default `8080`; the historical variable name is retained) selects the HTTPS listener port. The HTTP redirect listener is bound to loopback and is not exposed for deployment traffic. The health check also connects to the HTTPS listener.

On first startup, SubFlow creates a self-signed certificate for local development under `pb_data/.subflow/tls/`. Browsers and reverse proxies will not trust it automatically. In System administration → Settings, an administrator with system settings permission can upload a PEM certificate chain and its matching private key, or reset to the generated self-signed certificate. SubFlow validates that the certificate is currently valid, supports server authentication, has a DNS/IP subject alternative name, and matches the private key. The private key is stored with owner-only permissions under `pb_data`; uploads and resets are audited. The certificate is switched for new TLS handshakes without restarting the process.

The reverse proxy owns the public certificate and any public path routing. It may terminate public TLS and connect to SubFlow over HTTPS. Configure the proxy to trust the chosen SubFlow upstream certificate; do not disable upstream verification as a production shortcut. The PocketBase Application URL must remain the public HTTPS URL and is never inferred from proxy headers. SubFlow's SPA currently expects the public root path; path-prefix application support needs a separate frontend/deployment change.

## Process environment

The repository-root `.env` is the only environment file documented for deployment:

- `SUBFLOW_HTTP_PORT` controls the HTTPS backend listener and the default Compose host port. For a different host-only port, override the Compose `ports` mapping while leaving the container port unchanged.
- `SUBFLOW_ENV` controls development-only behavior. Deployment defaults to `production`; set `development` explicitly for local development.
- `SUBFLOW_SETTINGS_ENCRYPTION_KEY` is the persistent root key for encrypted settings and derived audit/share signing keys. Generate it once, keep it in the deployment secret store, back it up with PocketBase data, and never rotate or replace it casually. Existing CAPTCHA settings and encrypted share URLs depend on the original value.
- `SUBFLOW_IMAGE` selects the image in Compose; it is not passed into the application process.

SMTP, sender identity, application name/URL, CAPTCHA and OAuth product settings belong in PocketBase Admin. Do not restore the retired SMTP, public URL, data-driver, or frontend backend URL variables. Startup only reports the names of retired variables it finds and never prints their values.

## Local Vite development

Set `SUBFLOW_HTTP_PORT` in the repository-root `.env` (default `8080`). Start the Go backend with that file, then run Vite from `frontend/`:

```powershell
cd backend
go run ./cmd/subflow serve --env-file ../.env
```

```powershell
cd frontend
npm run dev
```

Vite reads the root port and proxies `/api` and `/_/`. The browser uses same-origin paths; there is no `VITE_BACKEND_URL` setting.

## Container identity and persistent data

The runtime image is based on Docker Hardened Images (`dhi.io/alpine-base`) and runs as UID/GID `65532:65532`. Fresh named volumes inherit the image's writable `/app/pb_data` directory. Normal startup never changes ownership. If upgrading a volume created by an older root-starting image, stop SubFlow and run this one-time command against the same image and volume before restarting:

```bash
docker run --rm --user 0:0 -v subflow_pb_data:/app/pb_data IMAGE init-permissions
```

The command only initializes `/app/pb_data`. Keep the root encryption key with the PocketBase backup. Restore data into an isolated volume and verify it before changing production.

## HTTP and cookies

SubFlow does not expose an external HTTP listener. All share cookies are always `Secure`, including behind a TLS-terminating reverse proxy. Public TLS and path routing are controlled by the proxy; the SubFlow upstream remains HTTPS.
