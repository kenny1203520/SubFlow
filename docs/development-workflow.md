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
