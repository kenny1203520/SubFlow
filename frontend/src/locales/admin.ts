export const adminExtras = {
  'zh-TW': {
    tlsCertificate:'HTTPS 憑證', tlsCertificateHelp:'SubFlow 僅提供 HTTPS。上傳 PEM 憑證鏈及相符的私密金鑰；私密金鑰會保存在資料目錄，不會回傳。', tlsCertificateUnavailable:'無法讀取 HTTPS 憑證狀態，請重新整理或確認伺服器版本。', tlsCertificateFilesRequired:'請選取憑證檔與相符的私密金鑰檔。', tlsCertificateUploaded:'目前使用上傳的憑證。', tlsCertificateSelfSigned:'目前使用 SubFlow 自簽憑證。', tlsCertificateExpires:'到期日：{date}', tlsCertificateFile:'憑證鏈 PEM 檔', tlsCertificateFileHelp:'檔案需先放伺服器憑證，再依序附上中繼憑證。', tlsPrivateKeyFile:'私密金鑰 PEM 檔', tlsPrivateKeyHelp:'請上傳與憑證相符的私密金鑰。', tlsCertificateSaved:'HTTPS 憑證已更新。', tlsCertificateReset:'已切回 SubFlow 自簽憑證。', tlsCertificateResetConfirm:'確定切回 SubFlow 自簽憑證嗎？反向代理或瀏覽器可能需要重新信任該憑證。', tlsCertificateSaveFailed:'憑證無效或無法儲存，請檢查憑證與私密金鑰。', tlsCertificateProgress:'上傳中 {percent}%', uploadTLSCertificate:'上傳憑證', useSelfSignedCertificate:'切回自簽憑證',
    notificationEmailRetryUnavailable:'這封郵件目前無法重新排入佇列，請重新整理清單。',
    retryNotificationEmail:'重新排入佇列', notificationEmailQueued:'通知郵件已重新排入佇列。', notificationEmailUnavailable:'目前無法使用通知郵件管理，請確認伺服器已更新後重試。',
    systemAdministration:'系統管理', openPocketBase:'開啟 PocketBase 後台', adminOverview:'總覽', userManagement:'使用者管理', roleManagement:'角色管理', auditLogs:'稽核日誌', failedNotificationEmails:'失敗的通知郵件', failedNotificationEmailsHelp:'僅管理員可查看。重新排入佇列後，系統會重新確認事件狀態與收件者偏好。', notificationDeliveryFailed:'通知郵件投遞失敗', deliveryAttempts:'已嘗試 {count} 次', deliveryErrorCode:'錯誤代碼', noFailedNotificationEmails:'目前沒有失敗的通知郵件', retry:'重新排入佇列', refresh:'重新整理', previous:'上一頁', next:'下一頁', permissions:'權限', assignRole:'指派角色', protectedRole:'受保護', noUsers:'找不到使用者', verificationSent:'帳號已建立。請查看電子郵件中的 PocketBase 驗證連結。',
    roleName:'角色名稱', roleGroup:'角色分組（選填）', roleGroupHelp:'選擇既有分組，或輸入新分組名稱來整理角色。', ungroupedRoles:'未分組', createRole:'新增角色', editRole:'編輯角色', backToGroups:'返回所有群組', groupAuditDesc:'檢視此群組的操作與安全稽核紀錄。', groupAuditEmptyDesc:'此群組目前沒有可顯示的稽核紀錄。',
  },
  en: {
    tlsCertificate:'HTTPS certificate', tlsCertificateHelp:'SubFlow serves HTTPS only. Upload a PEM certificate chain and its matching private key. The private key stays in the data directory and is never returned.', tlsCertificateUnavailable:'HTTPS certificate status is unavailable. Refresh or check the server version.', tlsCertificateFilesRequired:'Select a certificate file and its matching private key.', tlsCertificateUploaded:'An uploaded certificate is active.', tlsCertificateSelfSigned:'The SubFlow self-signed certificate is active.', tlsCertificateExpires:'Expires {date}', tlsCertificateFile:'Certificate chain (PEM)', tlsCertificateFileHelp:'Include the server certificate first, followed by any intermediate certificates.', tlsPrivateKeyFile:'Private key (PEM)', tlsPrivateKeyHelp:'Choose the private key that matches the certificate.', tlsCertificateSaved:'The HTTPS certificate was updated.', tlsCertificateReset:'SubFlow is using its self-signed certificate again.', tlsCertificateResetConfirm:'Switch back to the SubFlow self-signed certificate? Your reverse proxy or browser may need to trust it again.', tlsCertificateSaveFailed:'The certificate could not be validated or saved. Check the certificate and private key.', tlsCertificateProgress:'Uploading {percent}%', uploadTLSCertificate:'Upload certificate', useSelfSignedCertificate:'Use self-signed certificate',
    notificationEmailRetryUnavailable:'This email is no longer available for retry. Refresh the list.',
    retryNotificationEmail:'Queue retry', notificationEmailQueued:'The notification email has been queued again.', notificationEmailUnavailable:'Notification email management is unavailable. Check that the server is up to date, then retry.',
    systemAdministration:'System administration', openPocketBase:'Open PocketBase admin', adminOverview:'Overview', userManagement:'User management', roleManagement:'Role management', auditLogs:'Audit logs', failedNotificationEmails:'Failed notification emails', failedNotificationEmailsHelp:'Only administrators can view these jobs. Retrying queues the email again and rechecks event status and recipient preferences.', notificationDeliveryFailed:'Notification delivery failed', deliveryAttempts:'{count} attempts', deliveryErrorCode:'Error code', noFailedNotificationEmails:'No failed notification emails', retry:'Queue retry', refresh:'Refresh', previous:'Previous', next:'Next', permissions:'Permissions', assignRole:'Assign role', protectedRole:'Protected', noUsers:'No users found', verificationSent:'Account created. Check your email for the PocketBase verification link.',
    roleName:'Role name', roleGroup:'Role group (optional)', roleGroupHelp:'Choose an existing group or type a new one to organize roles.', ungroupedRoles:'Ungrouped', createRole:'Create role', editRole:'Edit role', backToGroups:'Back to all groups', groupAuditDesc:'Review operational and security audit records for this group.', groupAuditEmptyDesc:'There are no audit records for this group yet.',
  },
} as const

export const systemPermissionText = {
  'zh-TW': {
    'system.roles.manage': { title:'管理系統角色', description:'建立、編輯及刪除系統角色與其權限。' },
    'system.users.assign': { title:'指派使用者角色', description:'檢視使用者並指派其系統角色。' },
    'system.audit.read': { title:'檢視稽核日誌', description:'檢視全站操作與安全稽核紀錄。' },
    'system.settings.manage': { title:'管理系統設定', description:'調整站點預設值與新使用者註冊政策。' },
  },
  en: {
    'system.roles.manage': { title:'Manage system roles', description:'Create, edit, and delete system roles and permissions.' },
    'system.users.assign': { title:'Assign user roles', description:'View users and assign their system roles.' },
    'system.audit.read': { title:'View audit logs', description:'Review site-wide operational and security audit records.' },
    'system.settings.manage': { title:'Manage system settings', description:'Change site defaults and the registration policy.' },
  },
} as const
