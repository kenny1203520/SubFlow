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

Vite proxies `/api` to `VITE_BACKEND_URL`, defaulting to `http://localhost:8080`.
