<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { useI18n } from '../i18n'
const props=withDefaults(defineProps<{disabled?:boolean;disabledMessage?:string}>(),{disabled:false})
const emit=defineEmits<{filesSelected:[files:File[]]}>()
const {tr}=useI18n(),inputId=useId(),hintId=inputId+'-hint',dragging=ref(false)
let dragDepth=0
const accept='.jpg,.jpeg,.png,.webp,.heic,.heif,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx'
function isFileDrag(e:DragEvent){return Array.from(e.dataTransfer?.types||[]).includes('Files')}
function onDragEnter(e:DragEvent){if(!isFileDrag(e))return;e.preventDefault();if(props.disabled)return;dragDepth++;dragging.value=true}
function onDragOver(e:DragEvent){if(!isFileDrag(e))return;e.preventDefault();if(e.dataTransfer)e.dataTransfer.dropEffect=props.disabled?'none':'copy'}
function onDragLeave(e:DragEvent){if(!isFileDrag(e))return;dragDepth=Math.max(0,dragDepth-1);if(!dragDepth)dragging.value=false}
function onDrop(e:DragEvent){if(!isFileDrag(e))return;e.preventDefault();dragDepth=0;dragging.value=false;if(!props.disabled){const files=Array.from(e.dataTransfer?.files||[]);if(files.length)emit('filesSelected',files)}}
function onInput(e:Event){const input=e.currentTarget as HTMLInputElement;if(input.files?.length)emit('filesSelected',Array.from(input.files));input.value=''}
function preventNavigation(e:DragEvent){if(isFileDrag(e))e.preventDefault()}
watch(()=>props.disabled,value=>{if(value){dragDepth=0;dragging.value=false}})
onMounted(()=>{window.addEventListener('dragover',preventNavigation);window.addEventListener('drop',preventNavigation)})
onBeforeUnmount(()=>{window.removeEventListener('dragover',preventNavigation);window.removeEventListener('drop',preventNavigation)})
</script>
<template><div class="attachment-dropzone-wrap" :class="{'is-dragging':dragging,'is-disabled':disabled}" @dragenter="onDragEnter" @dragover="onDragOver" @dragleave="onDragLeave" @drop="onDrop"><input :id="inputId" class="attachment-file-input" type="file" multiple :accept="accept" :disabled="disabled" :aria-label="tr('attachmentChooseFiles')" :aria-describedby="hintId" @change="onInput"><label class="attachment-dropzone" :for="inputId"><svg aria-hidden="true" viewBox="0 0 24 24" width="32" height="32" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M12 16V4m0 0L7 9m5-5 5 5"/><path d="M5 14v4a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-4"/></svg><span class="attachment-dropzone-title">{{disabled?(disabledMessage||tr('attachmentDropDisabled')):(dragging?tr('attachmentDropNow'):tr('attachmentDropHint'))}}</span><span :id="hintId" class="attachment-dropzone-help">{{tr('attachmentFileTypes')}}</span></label></div></template>
<style scoped>
.attachment-dropzone-wrap{position:relative;min-width:0}.attachment-dropzone{display:grid;place-items:center;gap:.45rem;min-height:116px;padding:1rem;border:2px dashed var(--line-strong);border-radius:var(--radius,12px);background:var(--surface-soft);color:var(--muted);text-align:center;cursor:pointer;transition:border-color .15s ease,background-color .15s ease,color .15s ease}.attachment-dropzone svg{color:var(--brand)}.attachment-dropzone-title{color:var(--ink);font-weight:700}.attachment-dropzone-help{font-size:.82rem;line-height:1.4}.attachment-dropzone-wrap.is-dragging .attachment-dropzone{border-color:var(--brand);background:var(--brand-soft);color:var(--brand)}.attachment-dropzone-wrap.is-dragging .attachment-dropzone-title{color:var(--brand)}.attachment-dropzone-wrap.is-disabled .attachment-dropzone{cursor:not-allowed;opacity:.68}.attachment-file-input{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);clip-path:inset(50%);white-space:nowrap;border:0}.attachment-dropzone-wrap:focus-within .attachment-dropzone{outline:2px solid var(--brand);outline-offset:3px}@media(pointer:coarse){.attachment-dropzone{min-height:132px}}@media(prefers-reduced-motion:reduce){.attachment-dropzone{transition:none}}
</style>
