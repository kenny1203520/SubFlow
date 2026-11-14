# SubFlow

SubFlow uses one Go process and one runtime image for the Vue SPA, PocketBase authentication and admin UI, application API, uploads, and SSE. The production image serves the built Vue app from Go. Docker development uses separate Vite and Go services so frontend changes use HMR and backend changes trigger a cached image rebuild.

Uploaded attachments live in PocketBase's protected storage under `pb_data`. Backups and restores must include the complete PocketBase data volume so attachment files remain consistent with the database. The default attachment size limit is 20 MiB; administrators can raise it to 1 GiB. Configure the reverse proxy's request size and upload timeouts for the selected limit.

## 正式部署

使用已發布的單一 SubFlow image。先複製環境範本、產生並保存加密金鑰，再將 `SUBFLOW_IMAGE` 設為要部署的版本：

```sh
cp .env.example .env
openssl rand -hex 32
```

把產生的值填入 `.env` 的 `SUBFLOW_SETTINGS_ENCRYPTION_KEY`。每次升級都沿用原本金鑰；不要重新產生。環境檔採 `KEY=value`，註解請獨立放在行首。Compose 會在 image 或金鑰缺少時停止並說明原因。

啟動 GHCR 上的版本化 image：

```sh
docker compose --env-file .env up -d --no-build
```

不用 Compose 時，使用同一份 runtime 設定和資料 volume：

```sh
docker run -d --name subflow --restart unless-stopped \
  --env-file .env -p 8080:8080 \
  -v subflow_pb_data:/app/pb_data \
  ghcr.io/kenny1203520/subflow:0.2.0-beta.1
```

Compose 與 `docker run` 預設都使用 container port `8080`；Compose 的主機 port 預設也為 `8080`。要在同一台主機的反向代理後使用別的主機 port，可在 Compose override 中完整取代原有映射：

使用附帶的 [主機 Port override 範例](docker-compose.host-port.example.yml) 時，指令為 `docker compose --env-file .env -f docker-compose.yml -f docker-compose.host-port.example.yml up -d --no-build`。它用 `!override` 將對外映射完整替換成 `127.0.0.1:9443` 到 container port `8080`，不會同時保留預設的 `8080` 映射。`!override` 需要 Docker Compose 2.24.4 或更新版本。直接用 Docker 時，改成 `-p 9443:8080`。維持 container port 為 `8080`，並確認反向代理使用 HTTPS 連線至 SubFlow。

若要自行從原始碼建置：

```sh
docker build -t subflow:local .
```

之後設定 `.env` 的 `SUBFLOW_IMAGE=subflow:local` 並執行 Compose 的 `up -d --no-build`；直接執行則將上例 image 換成 `subflow:local`。正式 Compose 不會自行 build。

初始管理員、HTTPS 憑證、持久資料與正式升級流程請看 [部署設定](docs/deployment.md) 和 [PocketBase 升級與復原](docs/upgrading.md)。SMTP 與公開 Application URL 在 PocketBase 管理台設定。`pb_data` named volume 會在容器重建與一般 `docker compose down` 後保留；只有確定要永久刪除所有資料時才使用 `down -v`。

## 開發模式

Docker 開發環境使用專屬的 `subflow-dev` Compose 專案與資料 volume。複製開發範本後啟動：

```sh
cp .env.development.example .env.development.local
docker compose --env-file .env.development.local -f docker-compose.dev.yml up --build --force-recreate --watch
```

在瀏覽器開啟 `https://localhost:5173`。Vite 代理 `/api` 和 `/_/` 到 Compose 內的 Go 服務；前端來源會同步以啟用 HMR，套件清單變更會重建前端 image，Go 程式或模組變更會用快取重建後端。後端健康後才啟動 Vite。一般 `docker compose ... down` 不會刪除開發資料；`down -v` 會刪除開發資料和 TLS 憑證。

本機原生開發流程仍是分開執行 Go 和 Vite。兩個程序都使用 `.env.development.local`：Go 透過 `--env-file` 載入；`npm run dev` 只讀取其中 Vite proxy host、port 和 TLS data directory 設定，不會把開發加密金鑰傳給前端程序。首次需在一個終端機啟動 Go，再在另一個終端機啟動 Vite：

```powershell
cd backend
go run ./cmd/subflow serve --env-file ../.env.development.local
```

```powershell
cd frontend
npm ci
npm run dev
```

要改本機 Go port，更新共用檔案中的 `SUBFLOW_HTTP_PORT`，讓 Go listener 與 Vite proxy 使用相同數值。Docker 開發固定使用容器網路中的 port `8080`，只將 `https://localhost:5173` 發布至主機 loopback。前端只會掛載唯讀的開發 TLS 憑證；PocketBase 資料和開發加密金鑰只交給後端。

完整流程與服務隔離說明請看 [開發工作流程](docs/development-workflow.md)。

## 服務路由與整合

- `/` 與前端 Router 路徑：內嵌 Vue SPA。
- `/api/collections/users/*`：PocketBase users authentication。
- `/api/subflow/v1/*`：SubFlow API 與 SSE。
- `/_/`：PocketBase 管理台。
- `/api/health`：健康檢查。

PocketBase 是唯一 runtime 資料層。公開 TLS 終止與 path-prefix routing 由反向代理管理；SPA 目前預期部署在網站根路徑，不支援額外 prefix。

## Turnstile

在管理台選擇 Cloudflare Turnstile 後，填入同一個 widget 的 Site key 與 Secret key。部署前必須設定 `SUBFLOW_SETTINGS_ENCRYPTION_KEY`；SubFlow 會用它加密 PocketBase 的隱藏 CAPTCHA secret 與分享 URL。必須長期保留原值，並納入資料復原程序。外部 CAPTCHA 的網站身分以 PocketBase Dashboard 的 Application Name 與 Application URL 為唯一來源。Turnstile 與 reCAPTCHA 的 allowed domain 必須包含 Application URL hostname；hCaptcha 也須在供應商後台將該 hostname 加入 allowlist。

## 驗證

```sh
rtk go test ./...
rtk go vet ./...
rtk npm --prefix frontend run test:ci
rtk npm --prefix frontend run build
rtk docker compose --env-file .env config
rtk docker build -t subflow:local .
```
