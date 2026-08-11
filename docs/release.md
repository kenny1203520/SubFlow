# SubFlow Release Process

發布完全由 **版本 tag** 驅動。推送 `v<version>` tag 後，`.github/workflows/release.yml` 會自動建置單一映像檔、推送到 GHCR，並開立對應的 GitHub Release。

## 一次性設定

Workflow 需要兩個 repository secret（`Settings` → `Secrets and variables` → `Actions`）：

| Secret | 用途 |
| --- | --- |
| `DOCKERHUB_USERNAME` | 登入 `dhi.io` 拉取 base image 的 Docker 帳號 |
| `DOCKERHUB_TOKEN` | 該帳號的 access token（不要用密碼） |

`Dockerfile` 的三個 base image 都來自 Docker Hardened Images（`dhi.io/node`、`dhi.io/golang`、`dhi.io/alpine-base`），只有具備 DHI 授權的帳號才拉得到。**沒有設定這兩個 secret，build job 會在拉取 base image 時失敗。**

推送到 GHCR 用的是內建的 `GITHUB_TOKEN`，不需要額外設定。首次發布後，套件預設為 private，若要公開需到 repository 的 `Packages` 頁面手動改成 public。

## 版本編號

`frontend/package.json` 的 `version` 是**唯一**的版本來源：

- `vite.config.ts` 在 build 時把它注入成 `VITE_APP_VERSION`，「關於系統」頁面顯示的就是這個值。
- Workflow 的 `verify` job 會比對 tag 與 `package.json`，不一致就直接失敗。

所以 **改版本號一定要同時改 `package.json` 與 `package-lock.json`**（後者有兩處），否則 `npm ci` 會因為 lockfile 不同步而失敗。

## 發布步驟

依 Git Flow，正式版從 `release/<version>` 分支出去：

```bash
git checkout develop
git pull
git checkout -b release/0.1.0
```

在 release 分支上完成兩件事：

1. 更新 `frontend/package.json` 與 `frontend/package-lock.json` 的版本號。
2. 在 `CHANGELOG.md` 把 `## [Unreleased]` 底下的內容移到新的 `## [0.1.0] - YYYY-MM-DD` 區段。

本機驗收：

```bash
cd backend && go vet ./... && go test ./...
cd ../frontend && npm ci && npm test && npm run build
docker build -t subflow:local .
```

確認無誤後合併並打 tag：

```bash
git checkout master
git merge --no-ff release/0.1.0
git tag -a v0.1.0 -m "SubFlow 0.1.0"

git checkout develop
git merge --no-ff release/0.1.0

git push origin master develop
git push origin v0.1.0   # 這一步才會觸發發布
```

Tag 推上去之後 workflow 才會啟動。先推分支、確認 CI 沒問題再推 tag，可以避免發布到一半失敗。

## Workflow 做了什麼

`verify` job：

1. 先跑 `go vet` 與 `go test`。這一步刻意排在前端 build 之前 —— `backend/internal/web/dist/` 有一份被 git 追蹤的 placeholder，`asset_test.go` 會對它做斷言，先蓋掉會讓測試失敗。
2. 再跑 `npm ci`、`npm test`、`npm run build`。
3. 最後比對 tag 與 `package.json` 版本。

`release` job（`verify` 通過才會執行）：

1. 登入 `dhi.io` 與 `ghcr.io`。
2. 用 `docker/metadata-action` 產生 tag：`0.1.0`、`0.1`，以及非 pre-release 時的 `latest`。
3. 以 repository 根目錄為 build context 建置並推送（`Dockerfile` 同時需要 `frontend/` 與 `backend/`）。
4. 從 `CHANGELOG.md` 取出該版本區段作為 release notes，附上 `docker pull` 指令後開立 Release。

tag 含有 `-`（例如 `v0.2.0-rc.1`）時會被視為 pre-release，且不會更新 `latest`。

## 發布後

```bash
docker pull ghcr.io/kenny1203520/subflow:0.1.0
```

部署方式見 [deployment.md](deployment.md)。
