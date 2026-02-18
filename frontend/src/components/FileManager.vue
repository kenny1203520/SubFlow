<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { socket } from '../socket';

const props = defineProps<{
    groupId: string;
}>();

const { t } = useI18n();
const files = ref<any[]>([]);
const isUploading = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);

const fetchFiles = () => {
    // We need a socket event or API to list files for a group.
    // Assuming we have 'file:list' event.
    socket.emit('file:list', { groupId: props.groupId }, (res: any) => {
        if (res.status === 'ok') {
            files.value = res.files;
        }
    });
};

const handleUpload = async (event: Event) => {
    const target = event.target as HTMLInputElement;
    if (!target.files?.length) return;

    const file = target.files[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('file', file);
    formData.append('groupId', props.groupId);

    isUploading.value = true;
    try {
        const response = await fetch('/api/files/upload', {
            method: 'POST',
            body: formData
        });
        const result = await response.json();
        if (result.status === 'ok') {
            fetchFiles();
        }
    } catch (error) {
        console.error('Upload failed:', error);
    } finally {
        isUploading.value = false;
        if (fileInput.value) fileInput.value.value = '';
    }
};

const deleteFile = (fileId: string) => {
    socket.emit('file:delete', { fileId }, (res: any) => {
        if (res.status === 'ok') {
            fetchFiles();
        }
    });
};

onMounted(() => {
    fetchFiles();
});
</script>

<template>
    <div class="file-manager card">
        <div class="flex items-center justify-between mb-6">
            <h3 class="text-lg font-bold">{{ t('files.title', 'Attachments') }}</h3>
            <button class="btn btn-primary btn-sm" @click="fileInput?.click()" :disabled="isUploading">
                <svg v-if="!isUploading" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2" fill="none"
                    viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                <span v-if="isUploading" class="animate-spin mr-2">⏳</span>
                {{ isUploading ? t('common.actions.uploading') : t('common.actions.upload') }}
            </button>
            <input type="file" ref="fileInput" class="hidden" @change="handleUpload" />
        </div>

        <div class="file-list grid gap-4">
            <div v-if="files.length === 0" class="empty-files">
                {{ t('files.empty', 'No files attached yet.') }}
            </div>
            <div v-for="file in files" :key="file.id"
                class="file-card glass-card p-4 flex items-center justify-between">
                <div class="flex items-center gap-4">
                    <div class="file-icon bg-primary-50 text-primary-600 p-2 rounded-lg">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                        </svg>
                    </div>
                    <div>
                        <p class="font-semibold text-sm">{{ file.file_name }}</p>
                        <p class="text-xs text-muted">{{ (file.file_size / 1024).toFixed(1) }} KB</p>
                    </div>
                </div>
                <div class="flex gap-2">
                    <a :href="`/api/files/download/${file.id}`" target="_blank"
                        class="icon-btn-sm hover:text-primary-600">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                        </svg>
                    </a>
                    <button @click="deleteFile(file.id)" class="icon-btn-sm hover:text-danger-color">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.file-manager {
    border: none;
    background: rgba(255, 255, 255, 0.4);
    backdrop-filter: blur(10px);
}

.empty-files {
    padding: 2rem;
    text-align: center;
    color: var(--slate-400);
    border: 2px dashed var(--slate-200);
    border-radius: var(--radius-md);
}

.icon-btn-sm {
    background: none;
    border: none;
    color: var(--slate-400);
    cursor: pointer;
    padding: 0.4rem;
    border-radius: 8px;
    transition: all 0.2s;
}

.icon-btn-sm:hover {
    background: white;
}
</style>
