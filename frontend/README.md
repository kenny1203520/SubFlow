# SubFlow frontend development

Production Vue assets are built into the single SubFlow Go image and served by that process. There is no standalone frontend or Nginx image.

For the two-service Docker workflow, setup, and native Go/Vite workflow, see the repository [README](../README.md) and [development workflow](../docs/development-workflow.md).

The local `npm run dev` command explicitly reads only the proxy host, HTTPS port, and data directory from the repository-root `.env.development.local` when it exists. It does not pass the development encryption key to Vite. Vite does not load arbitrary `.env` files. Docker Compose supplies only the proxy host, HTTPS port, and read-only TLS certificate path to the frontend service.
