# SubFlow

SubFlow 以單一 Go 程序提供 Vue SPA、PocketBase users 驗證、管理台、檔案、SubFlow domain API 與 SSE。正式部署只有一個 `subflow` image/container；Node 與 Go image 僅用於 Docker multi-stage build，不會成為額外的 runtime service。

## 單一 Image 啟動

```bash
cp .env.example .env
docker compose up --build -d
```

預設入口為 `http://localhost:8080`，PocketBase 管理台為 `http://localhost:8080/_/`。Compose 只包含 `subflow` 一個 service，資料保存在 `pb_data` volume。

不使用 Compose 時：

```bash
docker build -t subflow:local .
docker run --rm -p 8080:8080 -v subflow-data:/app/pb_data \
  -e SUBFLOW_ENV=production \
  -e SUBFLOW_APP_URL=https://subflow.example.com \
  subflow:local
```

建立初始管理員：

```bash
docker compose exec subflow /app/subflow superuser create admin@example.com 'change-this-password'
```

## Port 設定

- `SUBFLOW_HTTP_PORT` 是 Go 程序實際監聽的 container port，預設 `8080`。
- `SUBFLOW_PUBLISH_PORT` 是 Compose 發佈至 host 的 port，預設 `8080`。
- CLI 明確傳入 `--http` 時優先於 `SUBFLOW_HTTP_PORT`。
- 若外部 URL 或 host port 不同，必須同步設定 `SUBFLOW_APP_URL`，確保邀請連結正確。

例如讓程序與 host 都使用 9000：

```env
SUBFLOW_HTTP_PORT=9000
SUBFLOW_PUBLISH_PORT=9000
SUBFLOW_APP_URL=http://localhost:9000
```

## 開發模式

Backend：

```bash
cd backend
go run ./cmd/subflow serve
```

預設監聽 `http://localhost:8080`。repository 內含一個 development placeholder，因此不需要先 build frontend；正式 Docker build 會以真正的 Vue `dist` 覆蓋它。

Frontend：

```bash
cd frontend
npm install
npm run dev
```

Vite 透過 `VITE_BACKEND_URL` 將 `/api` 代理到 Go backend，預設為 `http://localhost:8080`。

## 路由與外部服務

- `/` 與前端 Router 路徑：內嵌 Vue SPA。
- `/api/collections/users/*`：PocketBase users auth。
- `/api/subflow/v1/*`：SubFlow API 與 SSE。
- `/_/`：PocketBase 管理台。
- `/api/health`：健康檢查。

預設 Compose 不會啟動 PostgreSQL、MySQL、MinIO/S3 或 SMTP image。未來外部資料庫、object storage 與郵件服務透過 adapter、DSN 或環境變數接入；`SUBFLOW_DATA_DRIVER=pocketbase` 仍是目前唯一可執行的資料層。

## Turnstile

在管理台選擇 Cloudflare Turnstile 後，填入同一個 widget 的 Site key 與 Secret key。部署前必須設定 SUBFLOW_SETTINGS_ENCRYPTION_KEY；SubFlow 會用它加密 PocketBase 的隱藏 CAPTCHA secret 與分享 URL。這個密鑰必須長期保持不變；更換密鑰不會立即使公開分享 URL 失效，但管理頁將無法解密並顯示既有 URL。所有外部 CAPTCHA 的網站身分都以 PocketBase Dashboard 的 Application Name 與 Application URL 為唯一來源，不使用另一組環境變數。Turnstile widget 的 allowed domain（以及 reCAPTCHA 的網域設定）必須等於 Application URL hostname；hCaptcha 應在供應商後台將同一 hostname 加入 domain allowlist，後端也會把 PocketBase 的 site key 傳給 Siteverify。

## 資料重置

這個版本不遷移舊資料。需要完整重置時：

```bash
docker compose down -v
docker compose up --build -d
```

`down -v` 會永久刪除 PocketBase volume，應用程式本身不會自動刪除 volume。

## 驗證

```bash
cd backend
go vet ./...
go test ./...

cd ../frontend
npm test
npm run build

cd ..
docker compose config
docker build -t subflow:local .
```

啟動後應以同一個 port 驗證 SPA、註冊登入、群組 CRUD、邀請、訂閱、支出、PocketBase 管理台與 SSE 即時更新。
