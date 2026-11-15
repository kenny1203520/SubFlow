# SubFlow Development Workflow

## 分支與合併

SubFlow 採 Git Flow：

| 分支 | 用途 |
| --- | --- |
| `master` | 可部署的正式版本 |
| `develop` | 下一版本整合線 |
| `feature/<name>` | 新功能，從 `develop` 建立並回到 `develop` |
| `fix/<name>` | 一般錯誤修正，從 `develop` 建立並回到 `develop` |
| `release/<version>` | 發版穩定化，完成後合併至 `master` 與 `develop` |
| `hotfix/<name>` | 正式環境緊急修正，從 `master` 建立並回到 `master`、`develop` |

每個獨立功能完成後都要有自己的 commit。提交前確認：

- 沒有 stage 到其他工作者的既有修改。
- commit message 使用 Conventional Commits。
- 已執行與變更範圍相符的測試與 build。
- 稽核事件、權限檢查與失敗路徑沒有被遺漏。

## Docker 開發環境

Docker 開發使用獨立的 `subflow-dev` Compose 專案、Node 26 Vite service、Go 1.27 backend service，以及專屬具名 volume。需要 Docker Compose 2.24.4 或更新版本；這同時涵蓋 Compose Watch 與 `!override` 支援版本。Compose Watch 會同步前端來源並重建 Go backend。

從 repository root 複製範本並啟動：

```powershell
Copy-Item .env.development.example .env.development.local
docker compose --env-file .env.development.local -f docker-compose.dev.yml up --build --force-recreate --watch
```

在 `https://localhost:5173` 開啟應用。第一次啟動時 Go 會建立本機 HTTPS 憑證；Vite 等待 backend healthcheck 成功後才啟動，並以該憑證提供 HTTPS。瀏覽器若顯示本機憑證警告，接受後即可測試。Vite 代理 `/api` 與 `/_/` 到 `backend:8080`。

- `frontend/` 變更同步至 Vite 容器並觸發 HMR；`package.json` 或 `package-lock.json` 變更會重建前端 image。
- `backend/` 原始碼與 Go module 變更會透過 Docker layer cache 重建 backend image，然後重啟服務。失敗的建置會顯示錯誤；修正後再次儲存會重新觸發 build。
- PocketBase 資料只掛載給 backend。開發加密金鑰由 `.env.development.local` 注入 backend；frontend 只讀取獨立掛載的 TLS 憑證 volume。
- 一般 `docker compose ... down` 會保留開發資料與憑證。只有明確要重置資料時才使用 `down -v`。

`.env.development.local` 範本同時用於 Docker 與原生開發。原生 Go 指令明確傳入 `--env-file`；`npm run dev` 載入同一個檔案，Vite 只讀取 proxy 主機、port 和 TLS data directory。根目錄正式部署 `.env` 不會被 Vite 自動載入。要自訂原生 Go HTTPS port，在共用檔案修改 `SUBFLOW_HTTP_PORT`，讓 Go listener 與 Vite proxy 共用相同值。Playwright 直接啟動 Vite 並傳入測試用 port 和 data directory，不受此開發範本影響。

停止後端服務時，frontend 會因 `depends_on: service_healthy` 而保持等待狀態；恢復 backend 後 Compose 會按健康狀態啟動 frontend。要完全停止但保留資料，可在另一個終端機執行：

```powershell
docker compose --env-file .env.development.local -f docker-compose.dev.yml down
```

## 稽核日誌要求

稽核日誌是產品資料的一部分，不是只供除錯的 application log。後端以既有 `AuditLog`、`AuditRepository` 和 PocketBase `audit_logs` collection 為標準實作位置。

每個事件至少描述：操作者、範圍（personal/group/system）、動作、資源、資源 ID（若有）、結果、必要摘要、來源 IP、User-Agent、建立時間與 hash。摘要不可包含密碼、access token、refresh token、邀請 token 或完整敏感 payload。

新增 API 時，逐一檢查：

1. 未登入或無權限是否會被拒絕並留下失敗事件（若此 endpoint 具有可稽核的安全語意）。
2. 驗證失敗、資源不存在、衝突與內部錯誤是否留下適當結果。
3. 成功的 create/update/delete、權限、邀請與財務異動是否留下成功事件。
4. 前端是否只透過正式 API 觸發異動，沒有繞過後端直接寫入需要稽核的資料。

## 驗收指令

```powershell
rtk go test ./...
rtk go vet ./...
rtk npm --prefix frontend test
rtk npm --prefix frontend run build
rtk docker compose config
```

只需依變更範圍執行必要檢查，但在回報中必須列出實際執行的命令與結果。
