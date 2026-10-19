export const incomeAttachmentMessages = {
 'zh-TW': {
  permission_ledger_incomes_read:'檢視收入',permission_ledger_incomes_write:'新增及修改收入',permission_ledger_incomes_delete:'刪除收入',
  permissionDesc_ledger_incomes_read:'檢視群組收入、週期收入及受益分配。',permissionDesc_ledger_incomes_write:'新增及修改收入、管理週期與確認入帳；歷史操作另需歷史修改權限。',permissionDesc_ledger_incomes_delete:'刪除群組收入；歷史操作另需歷史修改權限。',
  attachmentUploadSuccess:'已上傳 {count} 個附件。',attachmentRefreshFailed:'無法更新附件清單。已完成的上傳仍保留，請重試更新清單。',attachmentPermissionDenied:'沒有權限操作此紀錄的附件。',attachmentSessionExpired:'登入已失效，請重新登入。',attachmentFormatInvalid:'檔案格式或內容不符，請使用支援的附件格式。',attachmentStorageFailed:'附件暫時無法儲存，請稍後重試。',attachmentAuditUnavailable:'稽核暫時無法寫入，附件操作未完成。',attachmentNetworkFailed:'連線中斷，請重試上傳。',attachmentTargetUnavailable:'找不到附件所屬紀錄，請重新開啟帳本確認。',attachmentConflict:'附件請求狀態衝突，請重新開啟後重試。',attachmentInvalidRequest:'附件請求無效，請重新選擇檔案。',
 },
 en: {
  permission_ledger_incomes_read:'View income',permission_ledger_incomes_write:'Create and edit income',permission_ledger_incomes_delete:'Delete income',
  permissionDesc_ledger_incomes_read:'View group income, recurring income, and beneficiary shares.',permissionDesc_ledger_incomes_write:'Create and edit income, manage recurring plans, and confirm receipts. Historical changes require historical write permission.',permissionDesc_ledger_incomes_delete:'Delete group income. Historical changes require historical write permission.',
  attachmentUploadSuccess:'Uploaded {count} attachment(s).',attachmentRefreshFailed:'Could not refresh the attachment list. Completed uploads are preserved; retry refreshing the list.',attachmentPermissionDenied:'You do not have permission to manage attachments on this record.',attachmentSessionExpired:'Your session expired. Sign in again.',attachmentFormatInvalid:'The file format or content is unsupported.',attachmentStorageFailed:'Attachment storage is unavailable. Try again later.',attachmentAuditUnavailable:'The audit log is unavailable. The attachment operation was not completed.',attachmentNetworkFailed:'Connection interrupted. Retry uploading.',attachmentTargetUnavailable:'The record is unavailable. Reopen the ledger to check it.',attachmentConflict:'The attachment request conflicts with its current state. Reopen and retry.',attachmentInvalidRequest:'Invalid attachment request. Select the file again.',
 },
} as const

export const incomeAttachmentAudit: Record<'zh-TW'|'en', {action:Record<string,string>;resource:Record<string,string>;field:Record<string,string>;reason:Record<string,string>}> = {
 'zh-TW': {
  action:{'income.created':'新增收入','income.updated':'修改收入','income.deleted':'刪除收入','attachment.listed':'檢視附件清單','attachment.uploaded':'上傳附件','attachment.downloaded':'開啟或下載附件','attachment.deleted':'刪除附件','recurring_income.created':'新增週期收入','recurring_income.updated':'修改週期收入','recurring_income.status_changed':'變更週期收入狀態','recurring_income.confirmed':'確認週期收入','recurring_income.skipped':'略過週期收入','recurring_income.retried':'重試週期收入','recurring_income.posted':'週期收入入帳','recurring_income.failed':'週期收入入帳失敗','recurring_income.reconciled':'核對週期收入','recurring_income.backfill_previewed':'預覽週期收入回補','recurring_income.backfilled':'回補週期收入'},
  resource:{income:'收入',attachment:'附件',recurring_income:'週期收入',income_occurrence:'週期收入期數'},
  field:{received_on:'收款日期',earned_by:'收款人',target_type:'附件所屬類型',count:'數量',size:'檔案大小（bytes）',posting_mode:'入帳模式',scheduled_on:'排定日期',through:'截止日期',status:'狀態',periods:'期數',kind:'分類類型'},
  reason:{target_unavailable:'紀錄不存在或無法存取',permission_denied:'沒有操作權限',ledger_read_denied:'沒有帳務讀取權限',scope_mismatch:'帳本範圍不符',not_found:'找不到紀錄',invalid_request:'請求無效',invalid_multipart:'上傳內容無效',invalid_billing_time:'期數日期無效',missing_file:'未提供檔案',request_too_large:'上傳超過大小限制',unsupported_format:'不支援的檔案格式或內容',file_too_large:'檔案超過大小限制',count_limit:'超過附件數量限制',request_conflict:'重試請求與紀錄不符',storage_write_failed:'附件儲存失敗',storage_read_failed:'附件讀取失敗',storage_list_failed:'附件清單讀取失敗',storage_delete_failed:'附件刪除失敗',limits_unavailable:'無法取得附件限制',permission_unavailable:'無法取得權限',upload_rejected:'上傳遭拒絕',posting_failed:'入帳失敗',request_failed:'請求失敗',forbidden:'沒有操作權限',invalid_or_write_failed:'資料無效或儲存失敗'},
 },
 en: {
  action:{'income.created':'Created income','income.updated':'Updated income','income.deleted':'Deleted income','attachment.listed':'Viewed attachment list','attachment.uploaded':'Uploaded attachment','attachment.downloaded':'Opened or downloaded attachment','attachment.deleted':'Deleted attachment','recurring_income.created':'Created recurring income','recurring_income.updated':'Updated recurring income','recurring_income.status_changed':'Changed recurring income status','recurring_income.confirmed':'Confirmed recurring income','recurring_income.skipped':'Skipped recurring income','recurring_income.retried':'Retried recurring income','recurring_income.posted':'Posted recurring income','recurring_income.failed':'Recurring income posting failed','recurring_income.reconciled':'Reconciled recurring income','recurring_income.backfill_previewed':'Previewed recurring income backfill','recurring_income.backfilled':'Backfilled recurring income'},
  resource:{income:'Income',attachment:'Attachment',recurring_income:'Recurring income',income_occurrence:'Recurring income occurrence'},
  field:{received_on:'Received on',earned_by:'Received by',target_type:'Attachment target type',count:'Count',size:'File size (bytes)',posting_mode:'Posting mode',scheduled_on:'Scheduled on',through:'Through',status:'Status',periods:'Periods',kind:'Category type'},
  reason:{target_unavailable:'Record unavailable or inaccessible',permission_denied:'Permission denied',ledger_read_denied:'Ledger read permission denied',scope_mismatch:'Ledger scope mismatch',not_found:'Record not found',invalid_request:'Invalid request',invalid_multipart:'Invalid upload body',invalid_billing_time:'Invalid occurrence date',missing_file:'File missing',request_too_large:'Upload size limit exceeded',unsupported_format:'Unsupported file format or content',file_too_large:'File size limit exceeded',count_limit:'Attachment count limit exceeded',request_conflict:'Retry request does not match record',storage_write_failed:'Attachment storage failed',storage_read_failed:'Attachment read failed',storage_list_failed:'Attachment list failed',storage_delete_failed:'Attachment deletion failed',limits_unavailable:'Attachment limits unavailable',permission_unavailable:'Permissions unavailable',upload_rejected:'Upload rejected',posting_failed:'Posting failed',request_failed:'Request failed',forbidden:'Permission denied',invalid_or_write_failed:'Invalid input or storage failure'},
 },
}

Object.assign(incomeAttachmentAudit['zh-TW'].action, {
 'recurring_income.access_denied':'週期收入存取遭拒絕',
 'income_occurrence.skipped':'略過週期收入期數','income_occurrence.confirmed':'確認週期收入入帳','income_occurrence.retried':'重試週期收入期數','income_occurrence.reconciled':'核對週期收入期數','income_occurrence.posted':'週期收入自動入帳','income_occurrence.pending':'建立待確認收入期數','income_occurrence.post_failed':'週期收入入帳失敗',
})
Object.assign(incomeAttachmentAudit.en.action, {
 'recurring_income.access_denied':'Recurring income access denied',
 'income_occurrence.skipped':'Skipped income occurrence','income_occurrence.confirmed':'Confirmed income receipt','income_occurrence.retried':'Retried income occurrence','income_occurrence.reconciled':'Reconciled income occurrence','income_occurrence.posted':'Automatically posted income','income_occurrence.pending':'Created pending income occurrence','income_occurrence.post_failed':'Income occurrence posting failed',
})
Object.assign(incomeAttachmentAudit['zh-TW'].field,{cycle:'週期',interval:'間隔',effective_on:'生效日期',period_count:'期數',income_id:'收入編號',before:'變更前',after:'變更後',splits:'受益分配',notes_changed:'備註已變更'})
Object.assign(incomeAttachmentAudit.en.field,{cycle:'Cycle',interval:'Interval',effective_on:'Effective on',period_count:'Period count',income_id:'Income ID',before:'Before',after:'After',splits:'Beneficiary shares',notes_changed:'Notes changed'})
incomeAttachmentAudit['zh-TW'].field.replayed='重複請求已核對'
incomeAttachmentAudit.en.field.replayed='Retry reconciled'
incomeAttachmentAudit['zh-TW'].reason.recurring_income_operation_failed='週期收入操作失敗'
incomeAttachmentAudit.en.reason.recurring_income_operation_failed='Recurring income operation failed'
incomeAttachmentAudit['zh-TW'].reason.audit_unavailable='稽核紀錄無法寫入'
incomeAttachmentAudit.en.reason.audit_unavailable='Audit log unavailable'
