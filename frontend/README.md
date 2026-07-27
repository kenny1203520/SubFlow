# SubFlow frontend

Vue 3 + TypeScript 單頁應用，直接透過官方 PocketBase JavaScript SDK 使用驗證、資料 CRUD 與 realtime subscriptions。

## Local development

```bash
npm install
npm run dev
```

Vite 會把 `/api` 代理到 `http://localhost:8090`。如需連到其他 PocketBase，可設定 `VITE_POCKETBASE_URL`。

## Production build

```bash
npm run build
```

正式環境由 Nginx 提供 `dist/` 靜態檔案，並將 `/api/` 反向代理至 PocketBase。
