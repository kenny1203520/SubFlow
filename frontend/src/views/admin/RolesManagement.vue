<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../../http';
import { useUIStore } from '../../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

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
                <h3 class="text-lg font-semibold text-white mb-4">{{ t('admin.roles.systemRoles') }}</h3>
                <div class="space-y-3">
                    <div v-for="role in roles" :key="role.id"
                        class="flex items-center justify-between p-4 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div>
                            <div class="font-medium text-white">{{ role.name }}</div>
                            <div class="text-sm text-neutral-400 mt-0.5">{{ role.description }}</div>
                        </div>
                        <div class="flex items-center space-x-3">
                            <button v-if="!role.is_system_role" @click="openPermModal(role)"
                                class="px-3 py-1.5 text-sm font-medium rounded-lg bg-purple-500/10 text-purple-400 border border-purple-500/20 hover:bg-purple-500/20 transition">
                                {{ t('admin.roles.managePermissions') }}
                            </button>
                            <span v-if="role.is_system_role"
                                class="px-2.5 py-1 text-xs font-semibold rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
                                {{ t('admin.roles.builtIn') }}
                            </span>
                        </div>
                    </div>
                </div>
            </section>

            <!-- User Role Assignment -->
            <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">{{ t('admin.roles.assignToUser') }}</h3>
                <div class="space-y-2">
                    <div v-for="user in users" :key="user.id"
                        class="flex items-center justify-between p-3 rounded-xl hover:bg-neutral-800/30 transition-colors">
                        <div>
                            <span class="font-medium text-white">{{ user.username }}</span>
                            <span class="text-neutral-500 text-sm ml-2">{{ user.email }}</span>
                        </div>
                        <button @click="openAssignModal(user)"
                            class="px-3 py-1.5 text-sm font-medium rounded-lg bg-neutral-800 text-neutral-300 hover:bg-neutral-700 transition">
                            {{ t('admin.roles.manageRoles') }}
                        </button>
                    </div>
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
                            <div class="text-xs text-neutral-500 mt-0.5 truncate">{{ perm.description }}</div>
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
    </div>
</template>
