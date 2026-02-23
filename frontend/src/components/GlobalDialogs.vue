<script setup lang="ts">
import { useUIStore } from '../stores/ui';
import { useI18n } from 'vue-i18n';

const ui = useUIStore();
const { t, te } = useI18n();

const safeT = (str: string) => {
    if (!str) return '';
    return te(str) ? t(str) : str;
};
</script>

<template>
    <!-- Confirm Dialog -->
    <div v-if="ui.isConfirmOpen"
        class="fixed inset-0 z-[100] flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
        <div
            class="bg-white rounded-2xl shadow-xl max-w-sm w-full p-6 border border-slate-100 transform transition-all">
            <h3 class="text-xl font-bold text-slate-800 mb-3">{{ safeT(ui.confirmTitle) }}</h3>
            <p class="text-slate-600 text-sm mb-6 leading-relaxed">{{ safeT(ui.confirmMessage) }}</p>

            <div class="flex gap-3 justify-end">
                <button @click="ui.handleConfirm(false)"
                    class="btn bg-white hover:bg-slate-50 text-slate-600 border border-slate-200 px-4 py-2">
                    {{ t('common.actions.cancel', 'Cancel') }}
                </button>
                <button @click="ui.handleConfirm(true)" class="btn btn-primary px-4 py-2">
                    {{ t('common.actions.confirm', 'Confirm') }}
                </button>
            </div>
        </div>
    </div>

    <!-- Alert Dialog -->
    <div v-if="ui.isAlertOpen"
        class="fixed inset-0 z-[100] flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
        <div
            class="bg-white rounded-2xl shadow-xl max-w-sm w-full p-6 border border-slate-100 transform transition-all">
            <h3 class="text-xl font-bold text-slate-800 mb-3">{{ safeT(ui.alertTitle) }}</h3>
            <p class="text-slate-600 text-sm mb-6 leading-relaxed">{{ safeT(ui.alertMessage) }}</p>

            <div class="flex justify-end">
                <button @click="ui.handleAlert()" class="btn btn-primary px-5 py-2">
                    {{ t('common.actions.ok', 'OK') }}
                </button>
            </div>
        </div>
    </div>
</template>
