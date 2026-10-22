<script setup lang="ts">
import AttachmentDropzone from './AttachmentDropzone.vue'
import {useI18n} from '../i18n'
withDefaults(defineProps<{disabled?:boolean;disabledMessage?:string}>(),{disabled:false})
const files=defineModel<File[]>({required:true}),{tr}=useI18n()
function addFiles(selected:File[]){files.value=[...files.value,...selected]}
function removeFile(index:number){files.value=files.value.filter((_,i)=>i!==index)}
function fileSize(size:number){return (size/1048576).toFixed(2)+' MB'}
</script>
<template><div class="attachment-draft-field"><AttachmentDropzone :disabled="disabled" :disabled-message="disabledMessage" @files-selected="addFiles"/><p v-if="files.length" role="status" class="attachment-selected-count">{{tr('attachmentSelected',{count:files.length})}}</p><ul v-if="files.length" class="attachment-draft-list" role="list"><li v-for="(file,index) in files" :key="index"><span class="attachment-draft-name" :title="file.name">{{file.name}}</span><small>{{fileSize(file.size)}}</small><button type="button" class="ghost" :aria-label="tr('attachmentRemoveSelected',{name:file.name})" :disabled="disabled" @click="removeFile(index)">{{tr('remove')}}</button></li></ul></div></template>
<style scoped>
.attachment-draft-field{display:grid;gap:.65rem;min-width:0}.attachment-selected-count{margin:0;color:var(--muted);font-size:.85rem}.attachment-draft-list{display:grid;gap:.4rem;margin:0;padding:0;list-style:none}.attachment-draft-list li{display:flex;align-items:center;gap:.65rem;min-width:0;padding:.55rem .65rem;border-bottom:1px solid var(--line)}.attachment-draft-name{flex:1;min-width:0;overflow-wrap:anywhere}.attachment-draft-list small{flex:0 0 auto;color:var(--muted)}.attachment-draft-list .ghost{min-width:44px;min-height:44px}@media(max-width:520px){.attachment-draft-list li{flex-wrap:wrap}.attachment-draft-name{flex-basis:calc(100% - 6rem)}.attachment-draft-list .ghost{margin-inline-start:auto}}
</style>
