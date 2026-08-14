# Changelog

本檔案記錄 SubFlow 的所有重要變更。

格式依循 [Keep a Changelog](https://keepachangelog.com/zh-TW/1.1.0/)，版本編號依循 [Semantic Versioning](https://semver.org/lang/zh-TW/)。

## [Unreleased]

## [0.1.7] - 2026-08-14

### Added
- Added viewer share and period-history details to finance views.
- Added subscription range editing, ownership transfer, and per-period price timelines.

### Changed
- Improved currency-aware calculations, exchange-rate cache coverage, and dashboard pagination.
- Expanded expense export details and tightened owner-role and settlement permissions.

### Fixed
- Fixed billing-window lookup, even-period rounding, subscription aggregation, and owner-role assignment rules.

## [0.1.6] - 2026-08-12

### Fixed
- 已綁定真實帳號的臨時成員，在「成員」列表中仍會以獨立一列顯示（標示「已綁定」），讓群組看起來人數比實際多；現在成員列表會隱藏已綁定的臨時成員，只顯示真實帳號那一列（分帳與歷史紀錄仍會正確透過綁定關係解析，不受影響）。
- 同一個臨時成員可以同時被兩筆邀請瞄準（例如先邀請 B、還沒接受又邀請 C），若兩邊都點選接受，會各自成功、且都被標記為「已接受」，實際上臨時成員只會綁到其中一人，造成資料不一致且沒有任何錯誤提示。現在：邀請時若該臨時成員已有其他待接受的邀請，會直接拒絕建立第二筆；且綁定當下也會確認尚未被別人綁定，確保「先接受者獲勝、後接受者收到明確錯誤」而不是靜默覆蓋。

## [0.1.5] - 2026-08-12

### Changed
- 重新設計群組邀請信：改為 HTML 郵件（保留純文字版本供不支援 HTML 的信箱），版面比照 PocketBase 內建的密碼重設／驗證信樣式（白底、圓角黑色按鈕），並內嵌 SubFlow 圖示、顯示邀請人與群組名稱、群組簡介與邀請到期日；主旨也改為包含群組名稱，不再是容易被當作垃圾信的通用文字。寄件者名稱仍沿用 PocketBase 後台設定的 Sender name。

## [0.1.4] - 2026-08-12

### Added
- 個人設定頁新增「第三方登入」：可將 Google／Spotify／OIDC 等登入方式連結到目前帳號（即使該登入方式回報的 Email 與帳號 Email 不同），之後即可用該方式直接登入，不需再輸入密碼；同一頁也可以查看已連結項目並取消連結。

### Fixed
- 個人設定頁的「帳號管理」卡片（0.1.2 新增）沒有標題與說明，且「登出」按鈕因為置於格線版面中被拉伸成整排寬度、文字置中，看起來像是排版錯誤：補上與其他卡片一致的標題區塊，並將按鈕改用彈性排版，不再被拉伸。
- 群組邀請信寄送失敗（一律回報伺服器錯誤）：邀請信原本使用獨立的 SMTP 寄信邏輯，需要另外設定一組環境變數，與 PocketBase 自身（透過管理後台設定、密碼重設信也使用）的郵件設定互不相通；現在邀請信改用 PocketBase 內建的寄信機制，只要 PocketBase 本身的郵件設定正常即可寄送。

## [0.1.3] - 2026-08-11

### Fixed
- 已安裝 PWA 的瀏覽器連到 `/_/` 會被導回 SubFlow 自己的首頁，而非 PocketBase 後台：Service Worker 的導覽快取回退規則（`navigateFallbackDenylist`）只排除了 `/api/`，未排除 `/_/`，導致已安裝過 Service Worker 的瀏覽器一律把該路徑的導覽請求改回應用程式首頁；停用瀏覽器快取無法繞過此問題，只有無痕視窗（未安裝過 Service Worker）或直接繞過瀏覽器才不受影響。現已將 `/_/` 加入排除清單。**此修正需要瀏覽器安裝到新版 Service Worker 才會生效**：既有訪客可透過既有的更新提示、或手動重新整理兩次／清除該網站的 Service Worker 來套用新版。

## [0.1.2] - 2026-08-11

### Fixed
- 手機版（螢幕寬度 900px 以下）完全無法使用「個人資料」「系統管理」與「登出」：底部導覽列取代側邊欄後，原本放置這些項目的區塊被整個隱藏且沒有替代入口。現在底部導覽列新增第 5 個「個人設定」分頁，並將系統管理連結與登出按鈕移至個人資料頁，桌面版側邊欄行為不變。

## [0.1.1] - 2026-08-11

### Fixed
- 以 `docker compose` 搭配 bind mount（如 `./pb_data:/app/pb_data`）啟動時，PocketBase 回報 `unable to open database file`：容器內建的 `nonroot` 使用者對 host 建立的目錄（通常是 root 所有）沒有寫入權限。容器現在預設以 root 啟動、由新增的 `entrypoint.sh` 一次性修正 `/app/pb_data` 的擁有者後才切換回 `nonroot` 執行實際程式，維持執行期非 root 的安全性，同時讓一般 `docker compose up` 不需要使用者手動調整 volume 權限。

## [0.1.0] - 2026-08-11

首個公開版本。SubFlow 是個人與群組共用的記帳系統，用於追蹤共同支出、訂閱與分帳結果。

### Added

#### 記帳核心
- 個人與群組帳本，支援支出、訂閱、還款與月結餘額。
- 支出分帳：平均分攤、固定金額、百分比三種模式。
- 訂閱管理：彈性計費週期（每日／每 N 日／每週／每 N 週／每 N 小時／每月／每季／每年）、排程停用，以及可版本化的分帳設定（此期及往後／僅此期）。
- 具備「編輯歷史記錄」權限者可選擇過去的扣款日，修正該期訂閱的分攤方式與金額。修改僅套用於該期，不影響之後的扣款設定；已入帳期數仍維持不可變更，需改為編輯對應支出。
- 多幣別群組，含系統匯率查詢、手動匯率，與群組報表幣別轉換。
- 群組會計時區，讓帳期與扣款日依群組設定計算。
- 分類管理與分類圖示。
- 個人與群組流水帳可匯出 CSV，涵蓋支出、訂閱、還款的全部歷史紀錄。

#### 群組協作
- 群組建立、成員邀請（含 Email 邀請信與邀請收件匣）與即時事件推播。
- 以角色為基礎的權限控制（RBAC），支援系統層級與群組層級角色。
- 稽核日誌，記錄操作者、範圍、動作、資源、結果與來源資訊，並可篩選與分頁瀏覽。
- 群組可以新增「臨時成員」（僅需名稱、無須真實帳號），讓分帳可以在對方實際加入前就開始進行；之後可邀請真實帳號與其綁定，歷史分攤與餘額會正確併入綁定後的帳號。

#### 帳號與安全
- 首次啟動的管理員設定精靈，採一次性安裝連結。
- Email／密碼與 OIDC 登入，含 Email OTP 與雙步驟驗證。
- 反機器人驗證，支援 Cloudflare Turnstile、hCaptcha、reCAPTCHA 與 ALTCHA。
- 註冊政策控制（可分別開關密碼註冊與 OIDC 註冊）。

#### 介面
- 響應式儀表板，個人／單一群組／所有群組三種檢視範圍。
- 繁體中文與英文介面，可切換佈景主題與時區。
- 使用者登入或開啟 App 時，若偵測到裝置目前時區與個人設定不同，會跳出提示並提供一鍵更新。
- 主要操作（訂閱、支出、群組、邀請、結算等）成功或失敗時會顯示 toast 通知。
- 可安裝的 PWA，含新版本更新提示。
- 關於系統頁面，顯示版本號與專案資源連結。

#### 部署
- 單一容器映像檔，前端資源直接嵌入 Go 執行檔。
- `docker compose` 部署設定與健康檢查。
- 由版本 tag 觸發的 GitHub Actions 發布流程，自動建置映像檔並開立 Release。

### Fixed
- 訂閱分帳資料在讀取時遺失，導致每位參與者都被計為負擔全額。
- 反機器人元件因 Content-Security-Policy 未允許供應商網域而完全無法載入。
- 群組角色管理的「新增角色」無法開啟編輯表單，權限勾選框排版錯誤。
- 系統匯率來源改用持續更新的服務，取代已停止更新的舊資料來源。
- 切換每頁筆數後未立即重新查詢。
- 新建立的群組在應用程式重啟前沒有種入角色，導致群組擁有者的權限檢查全數被拒。
- 個人設定頁的自訂分類管理元件因 Vue 對 boolean prop 的預設值推斷，在未顯式傳入 `canManage` 時一律視為 `false`，導致新增／編輯／刪除分類的介面完全不會顯示。
- 匯出流水帳的訂閱紀錄只會寫出一列（且日期與金額可能對不上同一期），現在會依實際扣款期數列出每一筆已發生的紀錄；所有匯出日期改用群組時區（個人匯出則用使用者時區）顯示，不再一律顯示 UTC。
- ALTCHA Community 驗證一律顯示「Verification failed. Try again later.」：後端誤用了與前端 widget 不相容的 KDF v2 挑戰格式，已改回與 widget 相符的傳統協定。
- Docker 映像檔改用 Docker Hardened Images 後建置失敗（執行期基底映像沒有 `apk`、也無法以非 root 身分建立使用者）：改在 `-dev` 映像的獨立階段安裝 `tzdata` 並只複製時區資料進最終映像，執行期直接沿用基底映像內建的 `nonroot`（65532）使用者，不再嘗試建立自訂使用者。

[Unreleased]: https://github.com/kenny1203520/SubFlow/compare/v0.1.6...HEAD
[0.1.6]: https://github.com/kenny1203520/SubFlow/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/kenny1203520/SubFlow/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/kenny1203520/SubFlow/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/kenny1203520/SubFlow/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/kenny1203520/SubFlow/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/kenny1203520/SubFlow/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/kenny1203520/SubFlow/releases/tag/v0.1.0
