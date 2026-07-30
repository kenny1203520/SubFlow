# SubFlow

SubFlow 是共同訂閱與支出管理服務。後端是單一 Go 執行檔，內嵌 PocketBase 0.39.10；PocketBase 永久負責 `users` 驗證、SQLite、檔案、管理台與基礎服務，所有業務操作則經過 `/api/subflow/v1`。

## 快速啟動

```bash
cp .env.example .env
docker compose up --build -d
```

開啟 `http://localhost:4173`。PocketBase 管理台位於 `http://localhost:4173/_/`。

第一次啟動後建立管理員：

```bash
docker compose exec backend /pb/subflow superuser create admin@example.com 'change-this-password'
```

## 全新資料庫重置

這個版本不遷移舊的 `members`、群組、訂閱、支出或使用者資料。升級到本版本時，先明確移除舊 volume：

```bash
docker compose down -v
docker compose up --build -d
```

`down -v` 會永久刪除 PocketBase volume，請只在確認不需保留舊資料後執行。應用程式不會自行刪除 volume。

## 環境與郵件

- `SUBFLOW_DATA_DRIVER=pocketbase` 是目前唯一可執行的資料層。`postgres`、`mysql` 或未知值會讓程序明確啟動失敗。
- `SUBFLOW_ENV=development` 未設定 SMTP 時，建立或重送邀請的回應會包含一次性的 `debugUrl`，不會寄信。
- 非 development 環境未完整設定 SMTP 時，邀請與密碼重設請求會被拒絕。
- 正式環境請設定 `SUBFLOW_APP_URL` 與所有 `SUBFLOW_SMTP_*` 變數。

## 架構邊界

前端只直接呼叫 PocketBase 原生 `users` 註冊、登入、refresh 與個人資料 API。群組、成員、邀請、訂閱、支出、儀表板和 SSE 全部走穩定的 `/api/subflow/v1`。

```text
Vue / Pinia ── PocketBase users auth ─┐
             └─ /api/subflow/v1 ── application services ── repository ports
                                                            └─ PocketBase adapter
```

業務服務只依賴 `backend/internal/ports`。若未來加入 PostgreSQL 或 MySQL：

1. 實作所有 repository port，外部關聯鍵使用 PocketBase user ID。
2. 在 `backend/internal/adapters.New` 增加 driver factory 分支與 DSN 驗證。
3. 讓新 adapter 重用 PocketBase adapter 使用的 repository contract suite。
4. 不修改前端、HTTP DTO、驗證來源或 domain service。

## API 約定

- 成功：`{"data": ..., "meta": ...}`；錯誤：`{"error":{"code":"...","message":"...","fields":{...}}}`。
- 金額使用 `amountMinor` 整數，日期使用 RFC 3339，幣別支援 TWD、USD、JPY、EUR。
- SSE：`GET /api/subflow/v1/events?groupId=...`，需 Bearer token；事件只包含 type、groupId、resource、resourceId、occurredAt。
- owner 可管理群組、成員與邀請；owner/member 均可管理群組內訂閱和支出。

## 開發驗證

```bash
cd backend
go test ./...

cd ../frontend
npm ci
npm test
npm run build
```

Docker 驗收：

```bash
docker compose config
docker compose up --build -d
docker compose ps
```

再以兩個不同瀏覽器 session 登入同一群組，確認新增訂閱或支出後另一端會收到 SSE 並重新抓取資料。
