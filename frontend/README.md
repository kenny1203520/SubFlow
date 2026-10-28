# SubFlow frontend development

Production frontend assets are built by the repository root `Dockerfile` and embedded into the SubFlow Go binary. There is no standalone frontend or Nginx image.

For local development, start the Go backend first:

```bash
cd backend
go run ./cmd/subflow serve
```

Then start Vite in another terminal:

```bash
cd frontend
npm install
npm run dev
```

Vite reads `SUBFLOW_HTTP_PORT` from the repository-root `.env` and proxies `/api` and `/_/` to the Go backend on that port. The browser uses same-origin paths; no backend URL is embedded in the frontend bundle.
