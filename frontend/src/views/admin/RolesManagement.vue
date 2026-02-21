<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../../http';
import { useUIStore } from '../../stores/ui';
import { useAuthStore } from '../../stores/auth';

const { t } = useI18n();
const ui = useUIStore();
const auth = useAuthStore();

interface SystemRole {
    id: string;
    name: string;
    description: string;
    is_system_role: boolean;
}

interface Permission {
    id: string;
    scope: string;
    action: string;
    resource: string;
    description: string;
}

interface User {
    id: string;
    username: string;
    email: string;
}

const roles = ref<SystemRole[]>([]);
const permissions = ref<Permission[]>([]);
const users = ref<User[]>([]);
const loading = ref(true);

// Role assignment modal (user ↔ roles)
const assignModal = ref(false);
const selectedUser = ref<User | null>(null);
const userRoles = ref<SystemRole[]>([]);
const assigningRole = ref(false);

// Permission assignment modal (role ↔ permissions)
const permModal = ref(false);
const selectedRole = ref<SystemRole | null>(null);
const rolePermissions = ref<Permission[]>([]);
const assigningPerm = ref(false);

// Role CRUD
const roleModal = ref(false);
const editingRole = ref<SystemRole | null>(null);
const roleForm = ref({ name: '', description: '' });
const roleSaving = ref(false);

// User Direct Permissions modal
const userPermModal = ref(false);
const selectedUserForPerms = ref<User | null>(null);
const userDirectPerms = ref<Permission[]>([]);
const assigningUserPerm = ref(false);

// Filter for permission scopes
const permFilterScope = ref('all');
const filteredPermissions = computed(() => {
    if (permFilterScope.value === 'all') return permissions.value;
    return permissions.value.filter(p => p.scope === permFilterScope.value);
});
const permScopes = computed(() => {
    const scopes = new Set(permissions.value.map(p => p.scope));
    return ['all', ...Array.from(scopes)];
});

const fetchData = async () => {
    loading.value = true;
    try {
        const [rolesRes, usersRes, permsRes] = await Promise.all([
            http.get('/api/admin/roles/system'),
            http.get('/api/admin/users'),
            http.get('/api/admin/permissions')
        ]);
        if (rolesRes.data.status === 'ok') roles.value = rolesRes.data.roles;
        if (usersRes.data.status === 'ok') users.value = usersRes.data.users;
        if (permsRes.data.status === 'ok') permissions.value = permsRes.data.permissions;
    } catch (e) {
        ui.alert(t('admin.roles.fetchFailed'));
    } finally {
        loading.value = false;
    }
};

// --- Role CRUD ---
const openRoleModal = (role: SystemRole | null = null) => {
    editingRole.value = role;
    roleForm.value = {
        name: role?.name || '',
        description: role?.description || ''
    };
    roleModal.value = true;
};

const saveRole = async () => {
    if (!roleForm.value.name) return;
    roleSaving.value = true;
    try {
        if (editingRole.value) {
            await http.put(`/api/admin/roles/system/${editingRole.value.id}`, roleForm.value);
        } else {
            await http.post('/api/admin/roles/system', roleForm.value);
        }
        roleModal.value = false;
        fetchData();
    } catch (e) {
        ui.alert(t('admin.roles.saveFailed'));
    } finally {
        roleSaving.value = false;
    }
};

const deleteRole = async (role: SystemRole) => {
    if (!await ui.confirm(t('admin.roles.confirmDelete', { name: role.name }))) return;
    try {
        await http.delete(`/api/admin/roles/system/${role.id}`);
        fetchData();
    } catch (e) {
        ui.alert(t('admin.roles.deleteFailed'));
    }
};

// --- User role assignment ---
const openAssignModal = async (user: User) => {
    selectedUser.value = user;
    assignModal.value = true;
    try {
        const res = await http.get(`/api/admin/users/${user.id}/roles`);
        if (res.data.status === 'ok') userRoles.value = res.data.roles;
    } catch { userRoles.value = []; }
};

const hasRole = (roleId: string) => userRoles.value.some(r => r.id === roleId);

const toggleRole = async (role: SystemRole) => {
    if (!selectedUser.value) return;
    assigningRole.value = true;
    try {
        if (hasRole(role.id)) {
            await http.delete(`/api/admin/users/${selectedUser.value.id}/roles/${role.id}`);
        } else {
            await http.post(`/api/admin/users/${selectedUser.value.id}/roles`, { roleId: role.id });
        }
        const res = await http.get(`/api/admin/users/${selectedUser.value.id}/roles`);
        if (res.data.status === 'ok') userRoles.value = res.data.roles;
    } catch (e) {
        ui.alert(t('admin.roles.assignFailed'));
    } finally {
        assigningRole.value = false;
    }
};

// --- User Direct Permissions ---
const openUserPermModal = async (user: User) => {
    selectedUserForPerms.value = user;
    permFilterScope.value = 'all';
    userPermModal.value = true;
    try {
        const res = await http.get(`/api/admin/users/${user.id}/permissions`);
        if (res.data.status === 'ok') userDirectPerms.value = res.data.permissions;
    } catch { userDirectPerms.value = []; }
};

const hasUserPerm = (permId: string) => userDirectPerms.value.some(p => p.id === permId);

const toggleUserPerm = async (perm: Permission) => {
    if (!selectedUserForPerms.value) return;
    assigningUserPerm.value = true;
    try {
        if (hasUserPerm(perm.id)) {
            await http.delete(`/api/admin/users/${selectedUserForPerms.value.id}/permissions/${perm.id}`);
        } else {
            await http.post(`/api/admin/users/${selectedUserForPerms.value.id}/permissions`, { permissionId: perm.id });
        }
        const res = await http.get(`/api/admin/users/${selectedUserForPerms.value.id}/permissions`);
        if (res.data.status === 'ok') userDirectPerms.value = res.data.permissions;
    } catch (e) {
        ui.alert(t('admin.roles.permFailed'));
    } finally {
        assigningUserPerm.value = false;
    }
};

// --- Role permission assignment ---
const openPermModal = async (role: SystemRole) => {
    selectedRole.value = role;
    permFilterScope.value = 'all';
    permModal.value = true;
    try {
        const res = await http.get(`/api/admin/roles/${role.id}/permissions`);
        if (res.data.status === 'ok') rolePermissions.value = res.data.permissions;
    } catch { rolePermissions.value = []; }
};

const hasPerm = (permId: string) => rolePermissions.value.some(p => p.id === permId);

const togglePerm = async (perm: Permission) => {
    if (!selectedRole.value) return;
    assigningPerm.value = true;
    try {
        if (hasPerm(perm.id)) {
            await http.delete(`/api/admin/roles/${selectedRole.value.id}/permissions/${perm.id}`);
        } else {
            await http.post(`/api/admin/roles/${selectedRole.value.id}/permissions`, { permissionId: perm.id });
        }
        const res = await http.get(`/api/admin/roles/${selectedRole.value.id}/permissions`);
        if (res.data.status === 'ok') rolePermissions.value = res.data.permissions;
    } catch (e) {
        ui.alert(t('admin.roles.permFailed'));
    } finally {
        assigningPerm.value = false;
    }
};

onMounted(fetchData);
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="text-2xl font-bold bg-gradient-to-r from-red-400 to-rose-600 bg-clip-text text-transparent">
                {{ t('admin.rolesManagement') }}</h2>
            <p class="text-neutral-400 mt-1">{{ t('admin.roles.subtitle') }}</p>
        </div>

        <div v-if="loading" class="text-center py-10">
            <div class="w-8 h-8 border-4 border-red-500/20 border-t-red-500 rounded-full animate-spin mx-auto mb-4">
            </div>
        </div>

        <div v-else class="space-y-8">
            <!-- System Roles List -->
            <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="flex justify-between items-center mb-6">
                    <h3 class="text-lg font-semibold text-white">{{ t('admin.roles.systemRoles') }}</h3>
                    <button v-if="auth.hasPermission('system', 'manage', 'roles')" @click="openRoleModal()"
                        class="btn btn-primary px-4 py-2 text-sm">
                        {{ t('admin.roles.createRole') }}
                    </button>
                </div>
                <div class="space-y-3">
                    <div v-for="role in roles" :key="role.id"
                        class="flex items-center justify-between p-4 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div>
                            <div class="flex items-center space-x-2">
                                <span class="font-medium text-white">{{ role.name }}</span>
                                <span v-if="role.is_system_role"
                                    class="px-2 py-0.5 text-[10px] font-bold rounded bg-blue-500/20 text-blue-400 border border-blue-500/20 uppercase">
                                    {{ t('admin.roles.builtIn') }}
                                </span>
                            </div>
                            <div class="text-sm text-neutral-400 mt-0.5">{{ role.description }}</div>
                        </div>
                        <div class="flex items-center space-x-2">
                            <button v-if="auth.hasPermission('system', 'manage', 'roles')" @click="openPermModal(role)"
                                class="p-2 text-purple-400 hover:bg-purple-500/10 rounded-lg transition"
                                title="Manage Permissions">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                                </svg>
                            </button>
                            <button v-if="auth.hasPermission('system', 'manage', 'roles')" @click="openRoleModal(role)"
                                class="p-2 text-blue-400 hover:bg-blue-500/10 rounded-lg transition" title="Edit Role">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                </svg>
                            </button>
                            <button v-if="auth.hasPermission('system', 'manage', 'roles') && !role.is_system_role"
                                @click="deleteRole(role)"
                                class="p-2 text-red-400 hover:bg-red-500/10 rounded-lg transition" title="Delete Role">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M19 7l-1 12a2 2 0 01-2 2H8a2 2 0 01-2-2L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                </svg>
                            </button>
                        </div>
                    </div>
                </div>
            </section>

            <!-- User Management Actions -->
            <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">{{ t('admin.roles.manageUsers') }}</h3>
                <div class="bg-neutral-950/50 rounded-xl border border-neutral-800/50 overflow-hidden text-sm">
                    <table class="w-full text-left">
                        <tbody class="divide-y divide-neutral-800/50">
                            <tr v-for="user in users" :key="user.id" class="hover:bg-neutral-800/20">
                                <td class="p-4">
                                    <div class="text-white font-medium">{{ user.username }}</div>
                                    <div class="text-neutral-500 text-xs">{{ user.email }}</div>
                                </td>
                                <td class="p-4 text-right space-x-2">
                                    <button v-if="auth.hasPermission('system', 'manage', 'user_roles')"
                                        @click="openAssignModal(user)"
                                        class="px-3 py-1.5 rounded-lg bg-neutral-800 text-neutral-300 hover:bg-neutral-700 transition">
                                        {{ t('admin.roles.manageRoles') }}
                                    </button>
                                    <button v-if="auth.hasPermission('system', 'manage', 'permissions_user')"
                                        @click="openUserPermModal(user)"
                                        class="px-3 py-1.5 rounded-lg bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20 transition">
                                        {{ t('admin.roles.directPermissions') }}
                                    </button>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </section>
        </div>

        <!-- User ↔ Role Assignment Modal -->
        <div v-if="assignModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-md">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <div>
                        <h3 class="text-xl font-semibold text-white">{{ t('admin.roles.manageRoles') }}</h3>
                        <p class="text-sm text-neutral-400 mt-1">{{ selectedUser?.username }}</p>
                    </div>
                    <button @click="assignModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div class="p-6 space-y-3">
                    <div v-for="role in roles" :key="role.id"
                        class="flex items-center justify-between p-3 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div>
                            <div class="font-medium text-white text-sm">{{ role.name }}</div>
                            <div class="text-xs text-neutral-500">{{ role.description }}</div>
                        </div>
                        <button @click="toggleRole(role)" :disabled="assigningRole" :class="[
                            hasRole(role.id)
                                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                : 'bg-neutral-800 text-neutral-400 border-neutral-700',
                            'px-3 py-1.5 text-xs font-medium rounded-lg border transition disabled:opacity-50'
                        ]">
                            {{ hasRole(role.id) ? t('admin.roles.assigned') : t('admin.roles.assign') }}
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Role ↔ Permission Assignment Modal -->
        <div v-if="permModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div
                class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <div>
                        <h3 class="text-xl font-semibold text-white">{{ t('admin.roles.managePermissions') }}</h3>
                        <p class="text-sm text-neutral-400 mt-1">{{ selectedRole?.name }}</p>
                    </div>
                    <button @click="permModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                <!-- Filter by scope -->
                <div class="px-6 pt-4 pb-2 border-b border-neutral-800/50">
                    <div class="flex flex-wrap gap-2">
                        <button v-for="scope in permScopes" :key="scope" @click="permFilterScope = scope" :class="[
                            scope === permFilterScope
                                ? 'bg-red-500/10 text-red-400 border-red-500/30'
                                : 'bg-neutral-800 text-neutral-400 border-neutral-700 hover:bg-neutral-700',
                            'px-3 py-1 text-xs font-medium rounded-lg border transition capitalize'
                        ]">
                            {{ scope === 'all' ? t('admin.roles.allScopes') : scope }}
                        </button>
                    </div>
                </div>

                <div class="p-6 overflow-y-auto flex-1 space-y-2">
                    <div v-for="perm in filteredPermissions" :key="perm.id"
                        class="flex items-center justify-between p-3 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center space-x-2">
                                <span
                                    class="px-1.5 py-0.5 text-[10px] font-semibold uppercase rounded bg-neutral-800 text-neutral-400">{{
                                        perm.scope }}</span>
                                <span class="font-medium text-white text-sm truncate">{{ perm.action }}: {{
                                    perm.resource }}</span>
                            </div>
                            <div class="text-xs text-neutral-500 mt-0.5 truncate">{{ t(perm.description) }}</div>
                        </div>
                        <button @click="togglePerm(perm)" :disabled="assigningPerm" :class="[
                            hasPerm(perm.id)
                                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                : 'bg-neutral-800 text-neutral-400 border-neutral-700',
                            'px-3 py-1.5 text-xs font-medium rounded-lg border transition disabled:opacity-50 shrink-0 ml-3'
                        ]">
                            {{ hasPerm(perm.id) ? t('admin.roles.granted') : t('admin.roles.grant') }}
                        </button>
                    </div>
                </div>
            </div>
        </div>
        <!-- Role Create/Edit Modal -->
        <div v-if="roleModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div
                class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-md animate-scale-in">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <h3 class="text-xl font-semibold text-white">
                        {{ editingRole ? t('admin.roles.editRole') : t('admin.roles.createRole') }}
                    </h3>
                    <button @click="roleModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div class="p-6 space-y-4">
                    <div class="space-y-1.5">
                        <label class="text-xs font-semibold text-neutral-500 uppercase px-1">{{
                            t('admin.roles.roleName')
                        }}</label>
                        <input v-model="roleForm.name" :disabled="editingRole?.is_system_role"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none transition disabled:opacity-50"
                            :placeholder="t('admin.roles.namePlaceholder')" />
                    </div>
                    <div class="space-y-1.5">
                        <label class="text-xs font-semibold text-neutral-500 uppercase px-1">{{
                            t('admin.roles.description') }}</label>
                        <textarea v-model="roleForm.description" rows="3"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none transition"
                            :placeholder="t('admin.roles.descriptionPlaceholder')"></textarea>
                    </div>
                </div>
                <div class="p-6 border-t border-neutral-800 flex justify-end gap-3">
                    <button @click="roleModal = false" class="px-4 py-2 text-neutral-400 hover:text-white transition">
                        {{ t('common.actions.cancel') }}
                    </button>
                    <button @click="saveRole" :disabled="roleSaving || !roleForm.name"
                        class="btn btn-primary px-6 py-2 transition disabled:opacity-50">
                        {{ roleSaving ? '...' : t('common.actions.save') }}
                    </button>
                </div>
            </div>
        </div>

        <!-- User ↔ Direct Permission Assignment Modal -->
        <div v-if="userPermModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div
                class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col animate-scale-in">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <div>
                        <h3 class="text-xl font-semibold text-white">{{ t('admin.roles.directPermissions') }}</h3>
                        <p class="text-sm text-neutral-400 mt-1">{{ selectedUserForPerms?.username }}</p>
                    </div>
                    <button @click="userPermModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                <!-- Filter by scope -->
                <div class="px-6 pt-4 pb-2 border-b border-neutral-800/50">
                    <div class="flex flex-wrap gap-2">
                        <button v-for="scope in permScopes" :key="scope" @click="permFilterScope = scope" :class="[
                            scope === permFilterScope
                                ? 'bg-red-500/10 text-red-400 border-red-500/30'
                                : 'bg-neutral-800 text-neutral-400 border-neutral-700 hover:bg-neutral-700',
                            'px-3 py-1 text-xs font-medium rounded-lg border transition capitalize'
                        ]">
                            {{ scope === 'all' ? t('admin.roles.allScopes') : scope }}
                        </button>
                    </div>
                </div>

                <div class="p-6 overflow-y-auto flex-1 space-y-2">
                    <div v-for="perm in filteredPermissions" :key="perm.id"
                        class="flex items-center justify-between p-3 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center space-x-2">
                                <span
                                    class="px-1.5 py-0.5 text-[10px] font-semibold uppercase rounded bg-neutral-800 text-neutral-400">{{
                                        perm.scope }}</span>
                                <span class="font-medium text-white text-sm truncate">{{ perm.action }}: {{
                                    perm.resource }}</span>
                            </div>
                            <div class="text-xs text-neutral-500 mt-0.5 truncate">{{ t(perm.description) }}</div>
                        </div>
                        <button @click="toggleUserPerm(perm)" :disabled="assigningUserPerm" :class="[
                            hasUserPerm(perm.id)
                                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                : 'bg-neutral-800 text-neutral-400 border-neutral-700',
                            'px-3 py-1.5 text-xs font-medium rounded-lg border transition disabled:opacity-50 shrink-0 ml-3'
                        ]">
                            {{ hasUserPerm(perm.id) ? t('admin.roles.granted') : t('admin.roles.grant') }}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
