#!/bin/sh
set -e

# =============================================================================
# Docker Compose Secrets Entrypoint Script
# =============================================================================
# 此腳本處理 Docker Compose Secrets 的 _FILE 模式環境變數
# 讀取 /run/secrets/* 檔案內容並匯出為標準環境變數
#
# 使用方式:
#   ENTRYPOINT ["entrypoint.sh"]
#   CMD ["node", "server.js"]
#
# 環境變數轉換範例:
#   AUTH_SECRET_FILE=/run/secrets/auth_secret
#   -> 讀取檔案內容並匯出為 AUTH_SECRET
# =============================================================================

export_from_file_var() {
	VAR="$1"; FILEVAR="$2"; FILEPATH="$(eval echo \"\${$FILEVAR}\")"
	if [ -n "$FILEPATH" ] && [ -f "$FILEPATH" ]; then export "$VAR"="$(cat "$FILEPATH")"; fi
}

export_from_secret() {
	VAR="$1"; FILEPATH="$2"
	if [ -f "$FILEPATH" ]; then export "$VAR"="$(cat "$FILEPATH")"; fi
}

echo "Processing Docker Secrets..."

export_from_file_var AUTH_SECRET AUTH_SECRET_FILE
export_from_file_var JWT_SECRET JWT_SECRET_FILE
export_from_file_var SMTP_PASS SMTP_PASS_FILE

export_from_secret AUTH_SECRET /run/secrets/auth_secret
export_from_secret JWT_SECRET /run/secrets/jwt_secret
export_from_secret SMTP_PASS /run/secrets/smtp_pass

echo "Starting application..."

exec "$@"