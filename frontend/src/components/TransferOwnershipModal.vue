<script setup lang="ts">
import { ref, computed } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import { useUIStore } from '../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

const props = defineProps<{
    groupId: string;
    members: any[];
    currentOwnerId: string;
}>();

const emit = defineEmits<{
    close: [];
    transferred: [];
}>();

const selectedMemberId = ref('');
const isSubmitting = ref(false);

// Filter out current owner and pending/temp members
const eligibleMembers = computed(() => 
    props.members.filter(m => 
        m.user_id && 
        m.user_id !== props.currentOwnerId &&
        m.status === 'active' &&
        m.joined_at
    )
);

const selectedMember = computed(() => 
    eligibleMembers.value.find(m => m.user_id === selectedMemberId.value)
);

const handleTransfer = async () => {
    if (!selectedMemberId.value) {
        ui.alert(t('groups.selectNewOwner', 'Please select a new owner'));
        return;
    }

    const member = selectedMember.value;
    if (!member) return;

    const displayName = member.display_name || member.username || member.email;
    const confirmed = await ui.confirm(t('groups.confirmTransfer', { name: displayName }));
    
    if (!confirmed) return;

    isSubmitting.value = true;

    socket.emit('group:ownership:transfer', {
        groupId: props.groupId,
        newOwnerId: selectedMemberId.value
    }, (res: any) => {
        isSubmitting.value = false;
        if (res.status === 'ok') {
            ui.alert(t('groups.ownership.transferSuccess'));
            emit('transferred');
            emit('close');
        } else {
            ui.alert(res.message || t('common.status.error'));
        }
    });
};
</script>

<template>
    <Teleport to="body">
        <div class="modal-overlay" @click.self="$emit('close')" v-if="true">
                <div class="modal-content glass-panel">
                    <!-- Header -->
                    <div class="flex justify-between items-center mb-6">
                        <h2 class="text-2xl font-bold text-slate-800">
                            {{ t('groups.ownership.transferTitle') }}
                        </h2>
                        <button @click="$emit('close')" class="text-slate-400 hover:text-slate-600 transition-colors">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M6 18L18 6M6 6l12 12" />
                            </svg>
                        </button>
                    </div>

                    <!-- Warning Message -->
                    <div class="bg-amber-50 border border-amber-200 text-amber-800 p-4 rounded-xl mb-6 flex gap-3">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
                        </svg>
                        <div class="text-sm">
                            {{ t('groups.ownership.transferDesc') }}
                        </div>
                    </div>

                    <!-- Select New Owner -->
                    <div class="space-y-4 mb-6">
                        <label class="field-label">
                            {{ t('groups.selectNewOwner', 'Select New Owner') }} <span class="text-red-500">*</span>
                        </label>

                        <div v-if="eligibleMembers.length === 0" class="text-center py-8 text-slate-400">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-3 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                            </svg>
                            <p class="text-sm">{{ t('groups.noEligibleMembers', 'No eligible members to transfer ownership to') }}</p>
                        </div>

                        <div v-else class="space-y-2 max-h-96 overflow-y-auto pr-2">
                            <div 
                                v-for="member in eligibleMembers" 
                                :key="member.user_id"
                                @click="selectedMemberId = member.user_id"
                                class="member-card"
                                :class="{ 'selected': selectedMemberId === member.user_id }"
                            >
                                <div class="flex items-center gap-3 flex-1">
                                    <div class="w-10 h-10 rounded-full bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center text-white font-bold text-sm">
                                        {{ (member.display_name || member.username || member.email || '?')[0].toUpperCase() }}
                                    </div>
                                    <div class="flex-1 min-w-0">
                                        <p class="font-medium text-slate-800 truncate">
                                            {{ member.display_name || member.username || 'Unknown' }}
                                        </p>
                                        <p v-if="member.email" class="text-xs text-slate-500 truncate">
                                            {{ member.email }}
                                        </p>
                                        <p v-if="member.role_name" class="text-xs text-primary-600 mt-0.5">
                                            {{ member.role_name }}
                                        </p>
                                    </div>
                                </div>
                                <div class="radio-indicator" :class="{ 'active': selectedMemberId === member.user_id }">
                                    <div class="inner-dot"></div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Actions -->
                    <div class="flex justify-end gap-3 pt-6 border-t border-slate-200">
                        <button 
                            type="button" 
                            @click="$emit('close')"
                            class="btn bg-slate-100 text-slate-600 hover:bg-slate-200"
                        >
                            {{ t('common.actions.cancel') }}
                        </button>
                        <button 
                            type="button"
                            @click="handleTransfer"
                            :disabled="!selectedMemberId || isSubmitting || eligibleMembers.length === 0"
                            class="btn btn-primary min-w-[120px]"
                        >
                            <span v-if="isSubmitting" class="flex items-center gap-2">
                                <svg class="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                </svg>
                                {{ t('common.status.loading') }}
                            </span>
                            <span v-else>{{ t('groups.ownership.transfer') }}</span>
                        </button>
                    </div>
                </div>
            </div>
    </Teleport>
</template>

<style scoped>
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(15, 23, 42, 0.5);
    backdrop-filter: blur(8px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    padding: 1rem;
}

.modal-content {
    width: 100%;
    max-width: 500px;
    background: white;
    border-radius: 1.5rem;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
    padding: 2rem;
    max-height: 90vh;
    overflow-y: auto;
}

.field-label {
    display: block;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--slate-700);
}

.member-card {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 1rem;
    border: 2px solid var(--border-color);
    border-radius: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    background: white;
}

.member-card:hover {
    border-color: var(--primary-300);
    background: var(--primary-50);
}

.member-card.selected {
    border-color: var(--primary-500);
    background: var(--primary-50);
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.radio-indicator {
    width: 20px;
    height: 20px;
    border: 2px solid var(--border-color);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
    flex-shrink: 0;
}

.radio-indicator.active {
    border-color: var(--primary-500);
    background: var(--primary-500);
}

.inner-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: white;
    opacity: 0;
    transition: opacity 0.2s ease;
}

.radio-indicator.active .inner-dot {
    opacity: 1;
}

/* Custom Scrollbar */
.modal-content::-webkit-scrollbar,
.space-y-2::-webkit-scrollbar {
    width: 6px;
}

.modal-content::-webkit-scrollbar-track,
.space-y-2::-webkit-scrollbar-track {
    background: transparent;
}

.modal-content::-webkit-scrollbar-thumb,
.space-y-2::-webkit-scrollbar-thumb {
    background-color: var(--border-color);
    border-radius: 3px;
}
</style>
