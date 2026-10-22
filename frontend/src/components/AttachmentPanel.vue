<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { ApiClient, ApiError } from '../api/client'
import type { Attachment, AttachmentPage } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import { canPreviewAttachment } from '../utils/attachmentPreview'
import AttachmentDropzone from './AttachmentDropzone.vue'
import AttachmentPreviewContent from './AttachmentPreviewContent.vue'
const props=defineProps<{targetType:string;targetId:string;billingAt?:string;online:boolean;disabled?:boolean;initialFiles?:File[]}>()
const emit=defineEmits<{initialFilesHandled:[]}>()
const auth=useAuthStore(),{tr}=useI18n(),api=new ApiClient(()=>auth.token,auth.logout)
const items=ref<Attachment[]>([]),access=ref({canRead:false,canUpload:false,canDelete:false,canRename:false}),limits=ref({maxSize:0,maxCount:0}),loading=ref(false),busy=ref(false),error=ref(''),progress=ref(0),preview=ref(''),previewType=ref(''),retryFiles=ref<File[]>([])
const uploadedCount=ref(0),refreshError=ref(''),previewDialog=ref<HTMLDialogElement>()
const panel=ref<HTMLElement>(),renameInput=ref<HTMLInputElement>(),renameId=ref(''),renameStem=ref(''),renameExtension=ref(''),renameError=ref(''),renameBusy=ref(false),renamedName=ref(''),previewName=ref('')
const renameInputId=useId()
function setRenameInput(element:unknown){renameInput.value=element instanceof HTMLInputElement?element:undefined}
let objectUrl='',initialHandledFor='',previousFocus:HTMLElement|undefined
const requestIDs=new WeakMap<File,string>()
function requestID(file:File){let id=requestIDs.get(file);if(!id){id=crypto.randomUUID();requestIDs.set(file,id)}return id}
function query(){const p=new URLSearchParams({targetType:props.targetType,targetId:props.targetId});if(props.billingAt)p.set('billingAt',props.billingAt);return '?'+p}
async function refresh(){const target=query();const r=await api.get<AttachmentPage>('/attachments'+target);if(target!==query())return;items.value=r.data.items;access.value=r.data.access;limits.value=r.data.limits}
async function load(){if(!props.targetId||!props.online)return;loading.value=true;try{await refresh();error.value='';if(props.initialFiles?.length&&initialHandledFor!==props.targetId){if(!access.value.canUpload){error.value=tr('attachmentPermissionDenied');return};initialHandledFor=props.targetId;const files=[...props.initialFiles];emit('initialFilesHandled');await upload(files)}}catch(reason){error.value=reason instanceof ApiError&&['forbidden','unauthorized','not_found'].includes(reason.code)?uploadMessage(reason):tr('attachmentLoadFailed')}finally{loading.value=false}}
function selectFiles(files:File[]){if(files.length)void upload(files)}
function uploadMessage(reason:unknown){
  const code=reason instanceof ApiError?reason.code:''
  const keys:Record<string,string>={forbidden:'attachmentPermissionDenied',unauthorized:'attachmentSessionExpired',attachment_format:'attachmentFormatInvalid',attachment_size:'attachmentSizeLimit',attachment_count:'attachmentCountLimit',attachment_storage:'attachmentStorageFailed',audit_unavailable:'attachmentAuditUnavailable',network_error:'attachmentNetworkFailed',not_found:'attachmentTargetUnavailable',conflict:'attachmentConflict',invalid_request:'attachmentInvalidRequest'}
  return tr(keys[code]||'attachmentUploadFailed')
}
async function upload(files:File[]){
  if(!props.online||props.disabled||!access.value.canUpload||busy.value||renameBusy.value||!!renameId.value)return
  busy.value=true;retryFiles.value=[];error.value='';refreshError.value='';uploadedCount.value=0
  try{
    for(let i=0;i<files.length;i++){
      const file=files[i]
      if(file.size>limits.value.maxSize||items.value.length+uploadedCount.value>=limits.value.maxCount){error.value=tr(file.size>limits.value.maxSize?'attachmentSizeLimit':'attachmentCountLimit');retryFiles.value=files.slice(i);break}
      try{
        const data=new FormData();data.append('file',file);data.append('targetType',props.targetType);data.append('targetId',props.targetId);if(props.billingAt)data.append('billingAt',props.billingAt);data.append('requestId',requestID(file))
        await api.upload<Attachment>('/attachments',data,p=>progress.value=p)
        uploadedCount.value++;progress.value=Math.round((i+1)*100/files.length)
      }catch(reason){error.value=uploadMessage(reason);retryFiles.value=files.slice(i);break}
    }
    try{await refresh()}catch{refreshError.value=tr('attachmentRefreshFailed')}
  }finally{busy.value=false;progress.value=0}
}
async function retryRefresh(){try{await refresh();refreshError.value=''}catch{refreshError.value=tr('attachmentRefreshFailed')}}
async function focusRenameButton(id:string){await nextTick();Array.from(panel.value?.querySelectorAll<HTMLButtonElement>('button[data-rename-id]')||[]).find(button=>button.dataset.renameId===id)?.focus()}
async function startRename(item:Attachment){
  if(!access.value.canRename||!props.online||props.disabled||busy.value||renameBusy.value)return
  const dot=item.fileName.lastIndexOf('.');renameExtension.value=dot>=0?item.fileName.slice(dot):'';renameStem.value=dot>=0?item.fileName.slice(0,dot):item.fileName
  renameId.value=item.id;renameError.value='';renamedName.value='';uploadedCount.value=0
  await nextTick();renameInput.value?.focus();renameInput.value?.select()
}
async function cancelRename(){if(renameBusy.value)return;const id=renameId.value;renameId.value='';renameError.value='';await focusRenameButton(id)}
async function saveRename(){
  if(!renameId.value||busy.value||renameBusy.value||!props.online||props.disabled||!access.value.canRename)return
  const fileName=renameStem.value.trim()+renameExtension.value
  if(!renameStem.value.trim()||Array.from(fileName).length>255||/[\\/\p{Cc}]/u.test(fileName)){renameError.value=tr('attachmentNameInvalid');return}
  const id=renameId.value;renameBusy.value=true;renameError.value=''
  try{
    const response=await api.patch<Attachment>('/attachments/'+id,{fileName})
    items.value=items.value.map(item=>item.id===id?response.data:item)
    renamedName.value=response.data.fileName;renameId.value='';renameBusy.value=false
    await focusRenameButton(id)
  }catch(reason){renameError.value=reason instanceof ApiError?tr(({forbidden:'attachmentPermissionDenied',attachment_name:'attachmentNameInvalid',audit_unavailable:'attachmentAuditUnavailable',unauthorized:'attachmentSessionExpired',network_error:'attachmentNetworkFailed'} as Record<string,string>)[reason.code]||'attachmentRenameFailed'):tr('attachmentRenameFailed')}
  finally{renameBusy.value=false}
}
async function remove(item:Attachment){if(!access.value.canDelete||!props.online||!window.confirm(tr('attachmentDeleteConfirm',{name:item.fileName})))return;try{await api.delete('/attachments/'+item.id);await load()}catch{error.value=tr('attachmentDeleteFailed')}}
async function show(item:Attachment,download=false){try{const r=await api.getBlob('/attachments/'+item.id+'/content');if(objectUrl)URL.revokeObjectURL(objectUrl);objectUrl=URL.createObjectURL(r.blob);if(download){const url=objectUrl,a=document.createElement('a');a.href=url;a.download=item.fileName;a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);objectUrl=''}else{previousFocus=document.activeElement instanceof HTMLElement?document.activeElement:undefined;preview.value=objectUrl;previewType.value=item.mediaType;previewName.value=item.fileName;await nextTick();previewDialog.value?.showModal()}}catch{error.value=tr('attachmentOpenFailed')}}
function closePreview(){previewDialog.value?.close();preview.value='';if(objectUrl)URL.revokeObjectURL(objectUrl);objectUrl='';previousFocus?.focus();previousFocus=undefined}
watch(()=>[props.targetType,props.targetId,props.billingAt,props.online] as const,()=>{items.value=[];access.value.canRead=false;void load()},{immediate:true})
onBeforeUnmount(closePreview)
</script>
<template>
  <section v-if="targetId&&online" ref="panel" class="attachment-panel" :aria-label="tr('attachments')">
    <header><div><strong>{{tr('attachments')}}</strong><small v-if="limits.maxCount">{{items.length}} / {{limits.maxCount}} · {{tr('attachmentSizeEach',{size:Math.round(limits.maxSize/1048576)})}}</small></div></header><AttachmentDropzone v-if="access.canUpload&&!disabled" :disabled="busy||renameBusy||!!renameId||items.length>=limits.maxCount" :disabled-message="items.length>=limits.maxCount?tr('attachmentCountLimit'):tr('attachmentDropDisabled')" @files-selected="selectFiles"/>
    <progress v-if="busy" max="100" :value="progress" :aria-label="tr('attachmentUploading')"></progress>
    <p v-if="uploadedCount" role="status" class="notice inline">{{tr('attachmentUploadSuccess',{count:uploadedCount})}}</p>
    <p v-if="renamedName" role="status" class="notice inline">{{tr('attachmentRenamed',{name:renamedName})}}</p>
    <p v-if="error" class="notice danger inline" role="alert">{{error}} <button v-if="retryFiles.length&&access.canUpload" type="button" class="ghost" :disabled="busy" @click="upload(retryFiles)">{{tr('retry')}}</button></p>
    <p v-if="refreshError" class="notice danger inline" role="alert">{{refreshError}} <button type="button" class="ghost" :disabled="busy" @click="retryRefresh">{{tr('retry')}}</button></p>
    <p v-if="loading" class="field-help">{{tr('processing')}}</p>
    <ul v-if="access.canRead&&items.length" class="attachment-list"><li v-for="item in items" :key="item.id">
      <template v-if="renameId===item.id">
        <div class="attachment-rename" @keydown.esc.stop.prevent="cancelRename">
          <label :for="renameInputId">{{tr('attachmentFileName')}}</label>
          <div class="attachment-rename-input"><input :id="renameInputId" :ref="setRenameInput" v-model="renameStem" type="text" :disabled="renameBusy||!online" :aria-invalid="!!renameError" :aria-describedby="renameInputId+'-extension'+(renameError?' '+renameInputId+'-error':'')" @keydown.enter.stop.prevent="saveRename"><span :id="renameInputId+'-extension'">{{renameExtension}}</span></div>
          <p v-if="renameError" :id="renameInputId+'-error'" role="alert" class="notice danger inline">{{renameError}}</p>
          <div class="attachment-rename-actions"><button type="button" class="ghost" :disabled="renameBusy||!online" @click="saveRename">{{tr(renameBusy?'processing':'saveChanges')}}</button><button type="button" class="ghost" :disabled="renameBusy" @click="cancelRename">{{tr('cancel')}}</button></div>
        </div>
      </template>
      <template v-else>
        <button v-if="canPreviewAttachment(item.fileName,item.mediaType)" type="button" class="attachment-name" :title="item.fileName" @click="show(item)">{{item.fileName}}</button><span v-else class="attachment-name" :title="item.fileName">{{item.fileName}}</span><span>{{(item.size/1048576).toFixed(2)}} MB</span>
        <button v-if="access.canRename" type="button" class="ghost" :data-rename-id="item.id" :disabled="disabled||busy||renameBusy||!!renameId" @click="startRename(item)">{{tr('attachmentRename')}}</button>
        <button type="button" class="ghost" @click="show(item,true)">{{tr('attachmentDownload')}}</button><button v-if="access.canDelete" type="button" class="ghost danger-text" :disabled="busy||renameBusy" @click="remove(item)">{{tr('remove')}}</button>
      </template>
    </li></ul><p v-else-if="access.canRead&&!loading" class="field-help">{{tr('attachmentEmpty')}}</p>
  </section>
  <Teleport to="body"><dialog v-if="preview" ref="previewDialog" class="attachment-preview" :aria-label="tr('attachmentPreview')" @cancel.prevent="closePreview" @click.self="closePreview"><header><strong>{{previewName}}</strong><button type="button" class="icon-button" :aria-label="tr('close')" @click="closePreview">×</button></header><AttachmentPreviewContent :src="preview" :file-name="previewName" :media-type="previewType" /></dialog></Teleport>
</template>
<style scoped>
.attachment-panel{display:grid;gap:.65rem}.attachment-panel>header{display:flex;align-items:center;justify-content:space-between;gap:.75rem}.attachment-panel>header div{display:grid;gap:.2rem}.attachment-panel small,.attachment-list>li>span{color:var(--muted);font-size:.85rem}.attachment-add{position:relative;cursor:pointer}.attachment-add input{position:absolute;inset:0;width:100%;height:100%;opacity:0;cursor:pointer}.attachment-list{list-style:none;padding:0;margin:0;display:grid;gap:.4rem}.attachment-list>li{display:flex;align-items:center;gap:.5rem;min-width:0;padding:.5rem;border-bottom:1px solid var(--line)}.attachment-name{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;text-align:left;background:none;border:0;color:var(--brand);cursor:pointer}.attachment-preview-backdrop{position:fixed;inset:0;z-index:1000;display:grid;place-items:center;padding:1rem;background:#0009}.attachment-preview{width:min(900px,100%);height:min(85vh,850px);display:flex;flex-direction:column;gap:.75rem;padding:1rem;background:var(--surface);border:1px solid var(--line);border-radius:var(--radius)}.attachment-preview header{display:flex;justify-content:space-between;align-items:center}.attachment-preview img,.attachment-preview iframe{flex:1;min-height:0;width:100%;object-fit:contain;border:0}@media(max-width:600px){.attachment-list>li{flex-wrap:wrap}}
</style>
<style scoped>
.attachment-preview{max-width:calc(100vw - 2rem);max-height:calc(100dvh - 2rem);color:var(--text)}
.attachment-preview::backdrop{background:#0009}
.attachment-add:focus-within,.attachment-name:focus-visible{outline:2px solid var(--brand);outline-offset:3px}
.attachment-name{white-space:nowrap}
.attachment-rename{display:grid;gap:.5rem;width:100%;min-width:0}.attachment-rename-input{display:flex;align-items:center;gap:.5rem;min-width:0}.attachment-rename-input input{min-width:0;flex:1}.attachment-rename-actions{display:flex;gap:.5rem}.attachment-preview header strong{overflow-wrap:anywhere}
@media(max-width:600px){.attachment-name{flex-basis:100%}.attachment-list .ghost{min-height:44px;min-width:44px}}
</style>
