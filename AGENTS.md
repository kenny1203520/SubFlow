# SubFlow Agent Instructions

這份文件是 Codex 與其他開發代理在本 repository 工作時的最高層級專案規範。

## 專案定位

- SubFlow 是記帳與共享財務管理軟體。
- 前台是 Vue 3 + TypeScript + Vite SPA，位於 `frontend/`。
- 後台是 Go HTTP API 與 PocketBase，位於 `backend/`；正式部署由單一 Go 程序提供 SPA、API、PocketBase 與 SSE。
- PocketBase 是目前唯一可執行的資料層；不要假設 PostgreSQL、MySQL 或其他外部服務已經存在。

## 強制工作規則

### RTK

- 所有 shell 命令一律透過 `rtk` 執行，以降低輸出噪音。
- Git、Go、npm 等命令使用 `rtk git ...`、`rtk go ...`、`rtk npm ...`。
- PowerShell 內建命令使用 `rtk powershell -NoProfile -Command "..."`。
- 不要為了方便直接繞過 `rtk`；若 `rtk` 無法執行，先說明原因，再採用最小必要替代方案。

### 稽核日誌

- 使用者在產品中的每一個會改變狀態或具有安全意義的操作，都必須產生稽核日誌；包含成功與失敗結果。
- 至少涵蓋登入、登出、註冊、群組與成員管理、角色/權限、邀請、訂閱、支出、結算、分類、幣別變更及管理操作。
- 稽核寫入必須在後端完成，不能只依賴 Vue 前端事件；所有新增或修改的 endpoint 都要檢查對應的稽核事件。
- 日誌不得寫入密碼、token、完整邀請連結或其他秘密；摘要只保留必要業務資訊。
- 優先重用既有 `AuditLog` domain model、`AuditRepository` 與 PocketBase `audit_logs` collection。若發現現有覆蓋不完整，必須在功能完成前補齊並加入測試。
- 稽核寫入失敗不可被靜默忽略；依功能的交易語意決定是否拒絕主要操作，但必須在方案與測試中明確說明。

### Git Flow

- Git 操作優先遵循 Git Flow：`master` 為正式線、`develop` 為整合線。
- 新功能使用 `feature/<short-name>`，修正使用 `fix/<short-name>`，正式準備使用 `release/<version>`，緊急正式修正使用 `hotfix/<short-name>`。
- 不直接在 `master` 或 `develop` 上實作功能；開始工作前確認目前分支用途，避免覆蓋使用者既有修改。
- 每個完整功能或獨立修正完成並通過驗證後，建立一個獨立 Git commit；不要把不相關變更混在同一 commit。
- Commit message 使用 Conventional Commits，例如 `feat(finance): ...`、`fix(audit): ...`、`docs: ...`。
- Commit 前檢查 `git diff` 與 `git status`，只 stage 本次工作檔案；既有未提交修改必須保留且不得順手提交。
- 未經明確要求，不執行破壞性 reset、checkout、clean 或刪除資料操作。

## 實作與驗證

- 先讀取相關 domain、port、adapter、API handler、store 與前端呼叫，再修改程式。
- 商業規則放在後端 application/domain；Vue 只負責呈現、互動、路由與 API 呼叫。
- PocketBase schema 的變更必須具備既有資料的相容升級/backfill 考量與測試。
- 前端變更至少執行相關 Vitest；後端變更至少執行相關 `go test`，必要時補跑完整測試、`go vet`、frontend build 與 Docker 驗證。
- 回報時說明實際修改、測試結果、未處理風險，以及 commit 是否成功。

## 完成功能的最低條件

1. 功能行為、權限與錯誤路徑已實作。
2. 所有使用者操作都有對應稽核事件，且測試涵蓋成功與失敗結果。
3. 前後端相關測試與必要 build 通過。
4. 只提交本次功能的變更，commit message 符合 Conventional Commits。

