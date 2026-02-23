<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import { useUIStore } from '../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

interface Permission {
    id: string;
    scope: string;
    action: string;
    resource: string;
    description: string;
}

const props = defineProps<{
    groupId: string;
    role: {
        id: string;
        name: string;
        is_system: boolean;
    };
}>();

const emit = defineEmits<{
    close: [];
    updated: [];
}>();

const allPermissions = ref<Permission[]>([]);
const rolePermissions = ref<Permission[]>([]);
const isLoading = ref(true);
const isUpdating = ref(false);

// Check if the role is Group Owner (read-only)
const isGroupOwner = computed(() => {
    return props.role.name === 'Group Owner';
});

// Group permissions by action
const groupedPermissions = computed(() => {
    const groups: Record<string, Permission[]> = {};
    
    allPermissions.value.forEach(perm => {
        if (!groups[perm.action]) {
            groups[perm.action] = [];
        }
        groups[perm.action]!.push(perm);
    });
    
    return groups;
});

// Check if permission is currently assigned to role
const hasPermission = (permissionId: string) => {
    return rolePermissions.value.some(p => p.id === permissionId);
};

// Get translation for permission
const getPermissionLabel = (perm: Permission) => {
    const key = `groups.permissions.group.${perm.action}.${perm.resource}`;
    const translated = t(key);
    
    // If translation not found, return a formatted fallback
    if (translated === key) {
        return `${perm.action} ${perm.resource}`.replace(/_/g, ' ');
    }
    
    return translated;
};

// Get action label
const getActionLabel = (action: string) => {
    const actionLabels: Record<string, string> = {
        read: t('common.actions.view', 'View'),
        create: t('common.actions.create', 'Create'),
        update: t('common.actions.update', 'Update'),
        delete: t('common.actions.delete', 'Delete'),
        invite: t('groups.invite', 'Invite'),
        remove: t('common.actions.remove', 'Remove'),
        assign: t('groups.assign', 'Assign'),
        grant: t('groups.permissions.grant', 'Grant'),
        revoke: t('groups.permissions.revoke', 'Revoke'),
        transfer: t('groups.transfer', 'Transfer'),
        manage: t('groups.manage', 'Manage'),
        settle: t('groups.expenses.settle', 'Settle'),
        export: t('common.actions.export', 'Export'),
        upload: t('common.actions.upload', 'Upload')
    };
    
    return actionLabels[action] || action;
};

// Toggle permission
const togglePermission = async (permission: Permission) => {
    if (isGroupOwner.value) {
        ui.alert(t('groups.permissions.groupOwnerReadOnly', 'Group Owner role permissions cannot be modified'));
        return;
    }

    const currentlyHas = hasPermission(permission.id);
    isUpdating.value = true;

    const event = currentlyHas ? 'group:role:revoke_permission' : 'group:role:grant_permission';
    
    socket.emit(event, {
        groupId: props.groupId,
        roleId: props.role.id,
        permissionId: permission.id
    }, (res: any) => {
        isUpdating.value = false;
        
        if (res.status === 'ok') {
            // Update local state
            if (currentlyHas) {
                rolePermissions.value = rolePermissions.value.filter(p => p.id !== permission.id);
            } else {
                rolePermissions.value.push(permission);
            }
        } else {
            ui.alert(res.message || t('common.status.error'));
        }
    });
};

// Load all available permissions and role's current permissions
onMounted(() => {
    isLoading.value = true;
    
    // Load all permissions
    socket.emit('group:permissions:list', { scope: 'group' }, (res: any) => {
        if (res.status === 'ok') {
            allPermissions.value = res.permissions || [];
        }
        
        // Load role permissions
        socket.emit('group:role:get_permissions', { roleId: props.role.id }, (res: any) => {
            isLoading.value = false;
            
            if (res.status === 'ok') {
                rolePermissions.value = res.permissions || [];
            }
        });
    });
});
</script>

<template>
    <Teleport to="body">
        <div class="modal-overlay" @click.self="$emit('close')" v-if="true">
                <div class="modal-content glass-panel">
                    <!-- Header -->
                    <div class="flex justify-between items-center mb-6">
                        <div>
                            <h2 class="text-2xl font-bold text-slate-800">
                                {{ t('groups.permissions.editPermissions') }}
                            </h2>
                            <p class="text-sm text-slate-600 mt-1">
                                {{ role.name }}
                                <span v-if="isGroupOwner" class="ml-2 text-xs px-2 py-0.5 bg-red-100 text-red-600 rounded">
                                    {{ t('groups.roles.stats.readOnly', 'Read-only') }}
                                </span>
                            </p>
                        </div>
                        <button @click="$emit('close')" class="text-slate-400 hover:text-slate-600 transition-colors">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M6 18L18 6M6 6l12 12" />
                            </svg>
                        </button>
                    </div>

                    <!-- Warning for Group Owner role -->
                    <div v-if="isGroupOwner" class="bg-slate-50 border border-slate-200 text-slate-700 p-3 rounded-xl mb-6 text-sm">
                        {{ t('groups.permissions.groupOwnerReadOnly', 'Group Owner role permissions are read-only and cannot be modified.') }}
                    </div>

                    <!-- Loading State -->
                    <div v-if="isLoading" class="flex justify-center items-center py-12">
                        <svg class="animate-spin h-8 w-8 text-primary-500" viewBox="0 0 24 24">
                            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                    </div>

                    <!-- Permissions List -->
                    <div v-else class="permissions-list space-y-6">
                        <div v-for="(permissions, action) in groupedPermissions" :key="action" class="action-group">
                            <h3 class="action-header">
                                {{ getActionLabel(action) }}
                            </h3>
                            
                            <div class="space-y-2">
                                <label 
                                    v-for="perm in permissions" 
                                    :key="perm.id"
                                    class="permission-item"
                                    :class="{ 'disabled': isGroupOwner || isUpdating }"
                                >
                                    <input 
                                        type="checkbox"
                                        :checked="hasPermission(perm.id)"
                                        @change="togglePermission(perm)"
                                        :disabled="isGroupOwner || isUpdating"
                                        class="checkbox"
                                    />
                                    <div class="flex-1">
                                        <p class="permission-label">
                                            {{ getPermissionLabel(perm) }}
                                        </p>
                                        <p class="permission-details">
                                            {{ t(perm.description || '') }}
                                        </p>
                                    </div>
                                </label>
                            </div>
                        </div>
                    </div>

                    <!-- Footer -->
                    <div class="flex justify-between items-center pt-6 border-t border-slate-200 mt-6">
                        <p class="text-sm text-slate-600">
                            {{ rolePermissions.length }} / {{ allPermissions.length }} {{ t('groups.permissionsGranted', 'permissions granted') }}
                        </p>
                        <button 
                            type="button" 
                            @click="$emit('close')"
                            class="btn btn-primary"
                        >
                            {{ t('common.actions.done', 'Done') }}
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
    max-width: 650px;
    background: white;
    border-radius: 1.5rem;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
    padding: 2rem;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
}

.permissions-list {
    overflow-y: auto;
    flex: 1;
    padding-right: 0.5rem;
}

.action-group {
    background: var(--slate-50);
    border-radius: 0.75rem;
    padding: 1rem;
    border: 1px solid var(--border-color);
}

.action-header {
    font-size: 0.875rem;
    font-weight: 700;
    color: var(--primary-700);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.75rem;
    padding-bottom: 0.5rem;
    border-bottom: 2px solid var(--primary-200);
}

.permission-item {
    display: flex;
    align-items: start;
    gap: 0.75rem;
    padding: 0.75rem;
    background: white;
    border: 1px solid var(--border-color);
    border-radius: 0.5rem;
    cursor: pointer;
    transition: all 0.2s ease;
}

.permission-item:hover:not(.disabled) {
    border-color: var(--primary-300);
    background: var(--primary-50);
}

.permission-item.disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.checkbox {
    width: 18px;
    height: 18px;
    cursor: pointer;
    accent-color: var(--primary-500);
    flex-shrink: 0;
    margin-top: 0.125rem;
}

.checkbox:disabled {
    cursor: not-allowed;
}

.permission-label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--slate-800);
    margin-bottom: 0.125rem;
}

.permission-details {
    font-size: 0.75rem;
    color: var(--slate-500);
    font-family: 'Courier New', monospace;
}

/* Custom Scrollbar */
.permissions-list::-webkit-scrollbar {
    width: 6px;
}

.permissions-list::-webkit-scrollbar-track {
    background: transparent;
}

.permissions-list::-webkit-scrollbar-thumb {
    background-color: var(--border-color);
    border-radius: 3px;
}
</style>
