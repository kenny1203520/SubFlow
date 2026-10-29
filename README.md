# SubFlow

## File attachments

Uploaded attachments are stored in PocketBase's protected file storage under `pb_data`. Backups and restores must include the complete `pb_data` volume so attachment files stay consistent with their database records.

The default attachment size limit is 20 MiB. System administrators can raise the configured maximum up to 1 GiB; the reverse proxy must allow a multipart request slightly larger than that setting and use an upload timeout appropriate for the configured maximum. For Nginx, configure `client_max_body_size` above the maximum file size (for example, `1026m` for a 1 GiB maximum) and set `client_body_timeout`, `proxy_read_timeout`, and `proxy_send_timeout` for the expected upload duration.

SubFlow 以單一 Go 程序提供 Vue SPA、PocketBase users 驗證、管理台、檔案、SubFlow domain API 與 SSE。正式部署只有一個 `subflow` image/container；Node 與 Go image 僅用於 Docker multi-stage build，不會成為額外的 runtime service。

## 單一 Image 啟動

```bash
cp .env.example .env
docker compose up --build -d
```

預設入口為 `https://localhost:8080`，PocketBase 管理台為 `https://localhost:8080/_/`。首次使用會自動建立本機自簽憑證，瀏覽器會顯示憑證警告；正式部署請在系統管理的設定頁上傳 SubFlow 上游使用的憑證與私鑰，或由反向代理終止公開 TLS。Compose 只包含 `subflow` 一個 service，資料保存在 `pb_data` volume。

不使用 Compose 時：

```bash
docker build -t subflow:local .
docker run --rm -p 8080:8080 -v subflow-data:/app/pb_data \
  -e SUBFLOW_ENV=production \
  subflow:local
```

建立初始管理員：

```bash
docker compose exec subflow /app/subflow superuser create admin@example.com 'change-this-password'
```

## Port 設定

- `SUBFLOW_HTTP_PORT` 是 Go 程序 HTTPS listener 的 container port，預設 `8080`（環境變數名稱為相容舊版而保留）。
- SubFlow 不提供外部 HTTP；PocketBase 的 HTTP redirect listener 僅綁定 loopback。
- 公開 Application URL 必須是 HTTPS。若反向代理連到 SubFlow 上游，請設定代理以 HTTPS 連線，並依部署的信任方式設定上游憑證驗證。
- Application URL、Application Name 與 SMTP 寄件設定請在 PocketBase Admin (`/_/`) 設定，變更後立即生效。

例如讓程序與 host 都使用 9000：

```env
SUBFLOW_HTTP_PORT=9000
```

Host port 與 container port 預設相同。若需要不同的 host port，請用 Compose override 修改 `ports` 對應；不要新增第二組環境變數。

## 開發模式

Backend：

```bash
cd backend
go run ./cmd/subflow serve --env-file ../.env
```

預設監聽 `https://localhost:8080`。本機開發自簽憑證位於 `backend/pb_data/.subflow/tls/`，Vite 會信任它作為本機後端憑證。`--env-file` 會載入指定設定檔，已存在的程序環境變數優先。分享 URL 與加密 CAPTCHA 設定都使用 `SUBFLOW_SETTINGS_ENCRYPTION_KEY`；請保留同一把金鑰，遺失或更換後既有密文無法還原。repository 內含一個 development placeholder，因此不需要先 build frontend；正式 Docker build 會以真正的 Vue `dist` 覆蓋它。

Frontend：

```bash
cd frontend
npm install
npm run dev
```

Vite 從 repository root `.env` 讀取 `SUBFLOW_HTTP_PORT`，並將 `/api` 與 `/_/` 代理到 Go backend。前端瀏覽器使用同源路徑，不需要 `VITE_BACKEND_URL`。

## 路由與外部服務

- `/` 與前端 Router 路徑：內嵌 Vue SPA。
- `/api/collections/users/*`：PocketBase users auth。
- `/api/subflow/v1/*`：SubFlow API 與 SSE。
- `/_/`：PocketBase 管理台。
- `/api/health`：健康檢查。

PocketBase is the only runtime data layer. SMTP and public application settings are managed in PocketBase Admin (`/_/`). Public TLS termination and any path-prefix routing are configured at the reverse proxy. SubFlow's SPA currently expects its public root path; path-prefix application support requires a separate frontend/deployment change.

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
