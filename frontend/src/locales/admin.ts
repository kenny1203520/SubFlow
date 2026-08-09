export const adminExtras = {
  'zh-TW': {
    systemAdministration:'系統管理', openPocketBase:'開啟 PocketBase 後台', adminOverview:'總覽', userManagement:'使用者管理', roleManagement:'角色管理', auditLogs:'稽核日誌', permissions:'權限', assignRole:'指派角色', protectedRole:'受保護', noUsers:'找不到使用者', verificationSent:'帳號已建立。請查看電子郵件中的 PocketBase 驗證連結。',
    roleName:'角色名稱', roleGroup:'角色分組（選填）', roleGroupHelp:'選擇既有分組，或輸入新分組名稱來整理角色。', ungroupedRoles:'未分組', createRole:'新增角色', editRole:'編輯角色', backToGroups:'返回所有群組', groupAuditDesc:'檢視此群組的操作與安全稽核紀錄。', groupAuditEmptyDesc:'此群組目前沒有可顯示的稽核紀錄。',
  },
  en: {
    systemAdministration:'System administration', openPocketBase:'Open PocketBase admin', adminOverview:'Overview', userManagement:'User management', roleManagement:'Role management', auditLogs:'Audit logs', permissions:'Permissions', assignRole:'Assign role', protectedRole:'Protected', noUsers:'No users found', verificationSent:'Account created. Check your email for the PocketBase verification link.',
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
