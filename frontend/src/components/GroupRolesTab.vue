<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import { useUIStore } from '../stores/ui';
import RolePermissionModal from './RolePermissionModal.vue';

const { t } = useI18n();
const ui = useUIStore();

const props = defineProps<{
    groupId: string;
    permissions: any;
}>();

const emit = defineEmits<{
    refresh: [];
}>();

const roles = ref<any[]>([]);
const loading = ref(true);
const showCreateModal = ref(false);
const showEditModal = ref(false);
const showPermissionModal = ref(false);
const selectedRole = ref<any>(null);
const roleForm = ref({
    name: '',
    description: ''
});

const canManageRoles = ref(true); // Computed from permissions
const quantityLimitReached = ref({
    canCreateMore: true,
    customRolesCount: 0,
    systemRolesCount: 0,
    roleLimit: 0
});

// Computed properties for stats
const roleLimit = computed(() => quantityLimitReached.value.roleLimit || 0);
const remainingRoles = computed(() => roleLimit.value - quantityLimitReached.value.customRolesCount);

const fetchRoles = () => {
    loading.value = true;
    socket.emit('group:role:list', { groupId: props.groupId }, (res: any) => {
        if (res.status === 'ok') {
            roles.value = res.roles;
        }
        loading.value = false;
    });
    socket.emit('group:role:quantity_limit', { groupId: props.groupId }, (res: any) => {
        if (res.status === 'ok') {
            quantityLimitReached.value = {
                canCreateMore: res.canCreateMore,
                customRolesCount: res.customRolesCount,
                systemRolesCount: res.systemRolesCount,
                roleLimit: res.roleLimit
            };
        }
    });
};

const openCreateModal = () => {
    roleForm.value = { name: '', description: '' };
    showCreateModal.value = true;
};

const openEditModal = (role: any) => {
    if (role.is_system_role) {
        ui.alert(t('groups.roles.cannotEditSystemRole'));
        return;
    }
    selectedRole.value = role;
    roleForm.value = { name: role.name, description: role.description };
    showEditModal.value = true;
};

const createRole = () => {
    if (!roleForm.value.name.trim()) {
        ui.alert(t('groups.roles.roleNameRequired'));
        return;
    }

    // Check for duplicate role name (case-insensitive)
    const nameLower = roleForm.value.name.trim().toLowerCase();
    const isDuplicate = roles.value.some(r => r.name.toLowerCase() === nameLower);
    
    if (isDuplicate) {
        ui.alert(t('groups.roles.duplicateRoleName'));
        return;
    }

    socket.emit('group:role:create', {
        groupId: props.groupId,
        ...roleForm.value
    }, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('common.status.success'));
            showCreateModal.value = false;
            fetchRoles();
        } else {
            ui.alert(res.message);
        }
    });
};

const updateRole = () => {
    if (!selectedRole.value || !roleForm.value.name.trim()) {
        ui.alert(t('groups.roles.roleNameRequired'));
        return;
    }

    // Check for duplicate role name (case-insensitive, excluding current role)
    const nameLower = roleForm.value.name.trim().toLowerCase();
    const isDuplicate = roles.value.some(r => 
        r.id !== selectedRole.value!.id && 
        r.name.toLowerCase() === nameLower
    );
    
    if (isDuplicate) {
        ui.alert(t('groups.roles.duplicateRoleName'));
        return;
    }

    socket.emit('group:role:update', {
        groupId: props.groupId,
        roleId: selectedRole.value.id,
        ...roleForm.value
    }, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('common.status.success'));
            showEditModal.value = false;
            fetchRoles();
        } else {
            ui.alert(res.message);
        }
    });
};

const deleteRole = async (role: any) => {
    if (role.is_system_role) {
        ui.alert(t('groups.roles.cannotDeleteSystemRole'));
        return;
    }

    if (!await ui.confirm(t('groups.roles.confirmDeleteRole'))) return;

    socket.emit('group:role:delete', {
        groupId: props.groupId,
        roleId: role.id
    }, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('common.status.success'));
            fetchRoles();
        } else {
            ui.alert(res.message);
        }
    });
};

const getRoleBadgeClass = (role: any) => {
    if (role.is_system_role) {
        return 'bg-blue-100 text-blue-700 border-blue-200';
    }
    return 'bg-emerald-100 text-emerald-700 border-emerald-200';
};

const openPermissionModal = (role: any) => {
    selectedRole.value = role;
    showPermissionModal.value = true;
};

onMounted(() => {
    fetchRoles();
});
</script>

<template>
    <div class="roles-tab">
        <!-- Header Section -->
        <div class="mb-8">
            <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
                <div>
                    <h2 class="text-2xl font-bold text-slate-900 mb-1">{{ t('groups.roles.title') }}</h2>
                    <p class="text-sm text-slate-600" v-if="!loading">
                        {{ t('groups.roles.total', { count: roles.length }) }}
                    </p>
                </div>
                <button 
                    v-if="canManageRoles && !loading"
                    @click="openCreateModal"
                    class="btn btn-primary w-full sm:w-auto flex items-center justify-center gap-2"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                        <path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd" />
                    </svg>
                    {{ t('groups.roles.createRole') }}
                </button>
            </div>

            <!-- Stats Bar -->
            <div v-if="!loading && roles.length > 0" class="grid grid-cols-4 gap-3 lg:grid-cols-4 md:grid-cols-2 sm:grid-cols-2">
                <div class="stat-card">
                    <div class="stat-number">{{ roles.length }}</div>
                    <div class="stat-label">{{ t('groups.roles.stats.totalRoles') }}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">{{ quantityLimitReached.systemRolesCount }}</div>
                    <div class="stat-label">{{ t('groups.roles.stats.systemRoles') }}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">{{ quantityLimitReached.customRolesCount }}</div>
                    <div class="stat-label">{{ t('groups.roles.stats.customRoles') }}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">{{ remainingRoles }}</div>
                    <div class="stat-label">{{ t('groups.roles.stats.remainingRoles') }}</div>
                </div>
            </div>
        </div>

        <!-- Loading State -->
        <div v-if="loading" class="flex justify-center py-16">
            <div class="flex flex-col items-center gap-4">
                <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
                <p class="text-slate-600">{{ t('common.status.loading') }}</p>
            </div>
        </div>

        <!-- Empty State -->
        <div v-else-if="roles.length === 0" class="empty-state-container">
            <div class="empty-state">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto mb-4 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
                </svg>
                <p class="text-slate-600 text-center font-medium">{{ t('groups.roles.emptyStateTitle') }}</p>
                <p class="text-slate-500 text-sm text-center mt-2">{{ t('groups.roles.emptyStateDesc') }}</p>
            </div>
        </div>

        <!-- Roles Grid -->
        <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div 
                v-for="role in roles" 
                :key="role.id"
                class="role-card"
            >
                <!-- Role Header -->
                <div class="flex items-start justify-between mb-4 pb-4 border-b border-slate-100">
                    <div class="flex-1">
                        <div class="flex items-center gap-3 mb-2">
                            <div class="flex-1">
                                <h3 class="text-lg font-bold text-slate-900">{{ role.name }}</h3>
                            </div>
                            <span :class="['role-badge', getRoleBadgeClass(role)]">
                                {{ role.is_system_role ? t('groups.roles.system') : t('groups.roles.custom') }}
                            </span>
                        </div>
                        <p class="text-sm text-slate-600">{{ role.description || t('common.status.empty') }}</p>
                    </div>
                </div>

                <!-- Role Content -->
                <div class="mb-4">
                    <div class="inline-flex items-center gap-2 px-3 py-2 bg-slate-50 rounded-lg">
                        <svg v-if="role.is_system_role" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-blue-600" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 3.062v6.005a3.066 3.066 0 01-3.062 3.062H5.5a3.066 3.066 0 01-3.062-3.062V6.517a3.066 3.066 0 012.812-3.062zM9 17a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
                        </svg>
                        <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-emerald-600" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 17v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.454z" clip-rule="evenodd" />
                        </svg>
                        <span class="text-sm font-medium text-slate-700">
                            {{ role.is_system_role ? t('groups.roles.systemRoleReadOnly') : t('groups.roles.customRoleEditable') }}
                        </span>
                    </div>
                </div>

                <!-- Role Actions -->
                <div v-if="canManageRoles" class="role-actions space-y-3">
                    <button 
                        @click="openPermissionModal(role)" 
                        class="action-button w-full"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M18 8a6 6 0 01-7.743 5.743L10 14l-1 1-1 1H6v2H2v-4l4.257-4.257A6 6 0 1118 8zm-6-4a1 1 0 100 2 2 2 0 012 2 1 1 0 102 0 4 4 0 00-4-4z" clip-rule="evenodd" />
                        </svg>
                        <span>{{ t('groups.permissions.editPermissions') }}</span>
                    </button>

                    <div v-if="!role.is_system_role" class="flex gap-3">
                        <button 
                            @click="openEditModal(role)" 
                            class="action-button edit-button flex-1"
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                                <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
                            </svg>
                            <span>{{ t('common.actions.edit') }}</span>
                        </button>
                        <button 
                            @click="deleteRole(role)" 
                            class="action-button delete-button flex-1"
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                                <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
                            </svg>
                            <span>{{ t('common.actions.delete') }}</span>
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Create Modal -->
        <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
            <div class="modal-content">
                <div class="flex items-center justify-between mb-6 pb-4 border-b border-slate-200">
                    <h3 class="text-xl font-bold text-slate-900">{{ t('groups.roles.createRole') }}</h3>
                    <button @click="showCreateModal = false" class="text-slate-400 hover:text-slate-600">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium text-slate-700 mb-2">{{ t('groups.roles.roleName') }} <span class="text-red-500">*</span></label>
                        <input 
                            v-model="roleForm.name"
                            type="text"
                            class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                            :placeholder="t('groups.roles.roleName')"
                        />
                    </div>

                    <div>
                        <label class="block text-sm font-medium text-slate-700 mb-2">{{ t('groups.roles.roleDescription') }}</label>
                        <textarea 
                            v-model="roleForm.description"
                            class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                            rows="3"
                            :placeholder="t('groups.roles.roleDescription')"
                        ></textarea>
                    </div>
                </div>

                <div class="flex gap-3 mt-6 pt-4 border-t border-slate-200">
                    <button @click="showCreateModal = false" class="btn btn-secondary flex-1">
                        {{ t('common.cancel') }}
                    </button>
                    <button @click="createRole" class="btn btn-primary flex-1">
                        {{ t('common.create') }}
                    </button>
                </div>
            </div>
        </div>

        <!-- Edit Modal -->
        <div v-if="showEditModal" class="modal-overlay" @click.self="showEditModal = false">
            <div class="modal-content">
                <div class="flex items-center justify-between mb-6 pb-4 border-b border-slate-200">
                    <h3 class="text-xl font-bold text-slate-900">{{ t('groups.roles.editRole') }}</h3>
                    <button @click="showEditModal = false" class="text-slate-400 hover:text-slate-600">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium text-slate-700 mb-2">{{ t('groups.roles.roleName') }} <span class="text-red-500">*</span></label>
                        <input 
                            v-model="roleForm.name"
                            type="text"
                            class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                            :placeholder="t('groups.roles.roleName')"
                        />
                    </div>

                    <div>
                        <label class="block text-sm font-medium text-slate-700 mb-2">{{ t('groups.roles.roleDescription') }}</label>
                        <textarea 
                            v-model="roleForm.description"
                            class="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                            rows="3"
                            :placeholder="t('groups.roles.roleDescription')"
                        ></textarea>
                    </div>
                </div>

                <div class="flex gap-3 mt-6 pt-4 border-t border-slate-200">
                    <button @click="showEditModal = false" class="btn btn-secondary flex-1">
                        {{ t('common.cancel') }}
                    </button>
                    <button @click="updateRole" class="btn btn-primary flex-1">
                        {{ t('common.actions.save') }}
                    </button>
                </div>
            </div>
        </div>

        <!-- Role Permission Modal -->
        <RolePermissionModal 
            v-if="showPermissionModal && selectedRole"
            :groupId="groupId"
            :role="selectedRole"
            @close="showPermissionModal = false"
            @updated="fetchRoles"
        />
    </div>
</template>

<style scoped>
/* ============================================
   Main Container Animation
   ============================================ */
.roles-tab {
    animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
    from {
        opacity: 0;
    }
    to {
        opacity: 1;
    }
}

/* ============================================
   Statistics Cards
   ============================================ */
.stat-card {
    background: linear-gradient(135deg, #f1f5f9 0%, #e2e8f0 100%);
    border: 1px solid #cbd5e1;
    border-radius: 0.5rem;
    padding: 1rem;
    text-align: center;
    transition: all 0.3s ease;
}

.stat-card:hover {
    border-color: hsl(250, 85%, 75%);
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.1);
}

.stat-number {
    font-size: 1.5rem;
    font-weight: 700;
    color: #0f172a;
    line-height: 1;
}

.stat-label {
    font-size: 0.75rem;
    color: #64748b;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-top: 0.25rem;
}

/* ============================================
   Role Cards (Grid Items)
   ============================================ */
.role-card {
    padding: 1.25rem;
    border-radius: 0.75rem;
    background-color: white;
    border: 1px solid #e2e8f0;
    transition: all 0.3s ease;
}

.role-card:hover {
    border-color: hsl(250, 95%, 88%);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.role-badge {
    display: inline-block;
    padding: 0.4rem 0.75rem;
    border-radius: 9999px;
    font-size: 0.75rem;
    font-weight: 600;
    border: 1px solid currentColor;
    white-space: nowrap;
}

/* ============================================
   Empty State Container
   ============================================ */
.empty-state-container {
    display: flex;
    justify-content: center;
    padding: 4rem 1rem;
}

.empty-state {
    text-align: center;
    max-width: 28rem;
}

.empty-state svg {
    margin: 0 auto 1rem;
}

.empty-state p:first-of-type {
    font-size: 1rem;
    font-weight: 500;
    color: #475569;
}

.empty-state p:last-of-type {
    font-size: 0.875rem;
    color: #94a3b8;
    margin-top: 0.5rem;
}

/* ============================================
   Action Buttons
   ============================================ */
.action-button {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.625rem 1rem;
    border-radius: 0.5rem;
    font-weight: 500;
    font-size: 0.875rem;
    transition: all 0.2s ease;
    border: 1px solid transparent;
    cursor: pointer;
    background-color: white;
    color: #475569;
    border-color: #cbd5e1;
}

.action-button:hover {
    background-color: #f1f5f9;
    border-color: #a1a5b3;
    transition: all 0.2s ease-in-out;
}

.action-button:active {
    transform: scale(0.95);
}

.action-button svg {
    width: 1rem;
    height: 1rem;
    display: block;
}

/* Edit Button Variant */
.edit-button {
    background-color: #eff6ff;
    color: #1e40af;
    border-color: #bfdbfe;
}

.edit-button:hover {
    background-color: #dbeafe;
    border-color: #7dd3fc;
}

/* Delete Button Variant */
.delete-button {
    background-color: #fef2f2;
    color: #b91c1c;
    border-color: #fecaca;
}

.delete-button:hover {
    background-color: #fee2e2;
    border-color: #fca5a5;
}

/* ============================================
   Modal Overlay & Content
   ============================================ */
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
    padding: 1rem;
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
}

.modal-content {
    background-color: white;
    border-radius: 1rem;
    padding: 1.5rem;
    max-width: 28rem;
    width: 100%;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 
                0 10px 10px -5px rgba(0, 0, 0, 0.04);
    animation: scaleIn 0.2s ease-out;
    max-height: 90vh;
    overflow-y: auto;
}

/* Modal Header Styling */
.modal-content .flex:first-child {
    gap: 1rem;
}

.modal-content h3 {
    font-size: 1.25rem;
    font-weight: 700;
    color: #0f172a;
}

.modal-content .text-slate-400 {
    color: #cbd5e1;
}

.modal-content .text-slate-400:hover {
    color: #64748b;
    transition: color 0.2s ease;
}

/* Form Elements in Modal */
.modal-content label {
    display: block;
    font-size: 0.875rem;
    font-weight: 500;
    color: #475569;
    margin-bottom: 0.5rem;
}

.modal-content input,
.modal-content textarea {
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: 1px solid #cbd5e1;
    border-radius: 0.5rem;
    font-size: 0.875rem;
    transition: all 0.2s ease;
    font-family: inherit;
}

.modal-content input:focus,
.modal-content textarea:focus {
    outline: none;
    border-color: hsl(250, 80%, 60%);
    ring: 2px;
    box-shadow: 0 0 0 2px hsl(250, 100%, 97%);
}

.modal-content textarea {
    resize: vertical;
    min-height: 3rem;
}

/* Modal Actions Footer */
.modal-content .flex:last-child {
    gap: 0.75rem;
    margin-top: 1.5rem;
    padding-top: 1rem;
    border-top: 1px solid #e2e8f0;
}

.modal-content .btn {
    flex: 1;
    padding: 0.625rem 1rem;
    border-radius: 0.5rem;
    font-weight: 500;
    font-size: 0.875rem;
    cursor: pointer;
    transition: all 0.2s ease;
    border: 1px solid transparent;
}

.modal-content .btn-secondary {
    background-color: #f1f5f9;
    color: #475569;
    border-color: #cbd5e1;
}

.modal-content .btn-secondary:hover {
    background-color: #e2e8f0;
    border-color: #94a3b8;
}

.modal-content .btn-primary {
    background-color: hsl(250, 80%, 60%);
    color: white;
}

.modal-content .btn-primary:hover {
    background-color: hsl(250, 80%, 50%);
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}

.modal-content .btn-primary:active {
    transform: scale(0.98);
}

/* ============================================
   Key Frame Animations
   ============================================ */
@keyframes scaleIn {
    from {
        opacity: 0;
        transform: scale(0.95);
    }
    to {
        opacity: 1;
        transform: scale(1);
    }
}

/* ============================================
   Custom Scrollbar (for modal if needed)
   ============================================ */
.modal-content::-webkit-scrollbar {
    width: 6px;
}

.modal-content::-webkit-scrollbar-track {
    background: transparent;
}

.modal-content::-webkit-scrollbar-thumb {
    background-color: #cbd5e1;
    border-radius: 3px;
}

.modal-content::-webkit-scrollbar-thumb:hover {
    background-color: #94a3b8;
}

/* ============================================
   Responsive Design Adjustments
   ============================================ */
@media (max-width: 640px) {
    .modal-content {
        max-width: 100%;
        max-height: calc(100vh - 2rem);
    }

    .stat-card {
        padding: 0.75rem;
    }

    .stat-number {
        font-size: 1.25rem;
    }

    .stat-label {
        font-size: 0.7rem;
    }
}
</style>
