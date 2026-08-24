<script setup lang="ts">
import { computed, onMounted, watch } from "vue";
import { useRoute } from "vue-router";
import { useWorkspaceStore } from "../stores/workspace";
import { useI18n } from "../i18n";
import { timezoneLabel } from "../timezone";
import { currencyLabel } from "../currency";
const route = useRoute(),
    workspace = useWorkspaceStore();
const { tr } = useI18n();
const groupId = computed(() => String(route.params.groupId || ""));
const group = computed(() =>
    workspace.groups.find((value) => value.id === groupId.value),
);
const canReadAudit = computed(() => workspace.groupPermissions.includes('group.audit.read'));
const canManageRoles = computed(() => workspace.groupPermissions.includes('group.roles.manage'));
const canManageSettings = computed(() => workspace.groupPermissions.includes('group.settings.manage'));
const canManageShares = computed(() => workspace.groupPermissions.includes('ledger.share.manage'));
const canReadExpenses = computed(() => workspace.groupPermissions.includes('ledger.expenses.read'));
const canReadSubscriptions = computed(() => workspace.groupPermissions.includes('ledger.subscriptions.read'));
const canViewGroup = computed(() => workspace.groupPermissions.includes('group.view'));
const canViewDashboard = computed(() => canViewGroup.value && canReadExpenses.value && canReadSubscriptions.value);
const routePermissions = computed<Record<string, string[]>>(() => ({
    'group-overview': ['group.view', 'ledger.expenses.read', 'ledger.subscriptions.read'],
    'group-expenses': ['ledger.expenses.read'],
    'group-subscriptions': ['ledger.subscriptions.read'],
    'group-members': ['group.view'],
    'group-roles': ['group.roles.manage'],
    'group-audit': ['group.audit.read'],
    'group-settings': ['group.settings.manage'],
		'group-share': ['ledger.share.manage'],
}));
const canAccessRoute = computed(() => (routePermissions.value[String(route.name)] || []).every(permission => workspace.groupPermissions.includes(permission)));
const accessLoading = computed(() => workspace.groupBusy.access > 0);
async function activate() {
    if (groupId.value) await workspace.selectGroup(groupId.value);
}
onMounted(() => void activate());
watch(groupId, () => void activate());
</script>
<template>
    <section class="group-workspace">
        <div class="group-workspace-head">
            <RouterLink class="back-link" to="/groups" :aria-label="tr('backToGroups')"><span aria-hidden="true">←</span>{{ tr("allGroupsLink") }}</RouterLink>
            <div class="group-workspace-identity">
                <p class="eyebrow">{{tr('groupWorkspace')}}</p>
                <h1>{{ group?.name || tr("groupWorkspace") }}</h1>
                <p>{{ group?.description || tr("groupWorkspaceDesc") }}</p>
                <div v-if="group" class="workspace-facts"><span>{{currencyLabel(group.currency)}}</span><span>{{timezoneLabel(group.timezone)}}</span><span>{{tr('memberCount',{count:workspace.members.length})}}</span><RouterLink v-if="canManageSettings" :to="`/groups/${groupId}/settings`">{{tr('settings')}} →</RouterLink></div>
            </div>
        </div>
        <nav class="group-tabs" :aria-label="tr('groupWorkspace')">
            <RouterLink v-if="canViewDashboard" :to="`/groups/${groupId}/overview`">{{
                tr("groupOverview")
                }}</RouterLink>
            <RouterLink v-if="canReadExpenses" :to="`/groups/${groupId}/expenses`">{{
                tr("splitExpenses")
                }}</RouterLink>
            <RouterLink v-if="canReadSubscriptions" :to="`/groups/${groupId}/subscriptions`">{{
                tr("manageSubscriptions")
                }}</RouterLink>
            <RouterLink v-if="canViewGroup" :to="`/groups/${groupId}/members`">{{
                tr("members")
                }}</RouterLink>
            <RouterLink v-if="canManageRoles" :to="`/groups/${groupId}/roles`">{{ tr("roleManagement") }}</RouterLink>
            <RouterLink v-if="canReadAudit" :to="`/groups/${groupId}/audit`">{{ tr("auditLogs") }}</RouterLink>
            <RouterLink v-if="canManageShares" :to="`/groups/${groupId}/share`">{{ tr('share') }}</RouterLink>
            <RouterLink v-if="canManageSettings" :to="`/groups/${groupId}/settings`">{{
                tr("settings")
                }}</RouterLink>
				<RouterLink v-if="canManageShares" :to="`/groups/${groupId}/share`">{{ tr('share') }}</RouterLink>
        </nav>
        <div v-if="accessLoading" class="empty-inline">{{ tr('processing') }}</div>
        <div v-else-if="!canAccessRoute" class="resource-error">
            <p>{{ workspace.groupErrors.access || tr('forbiddenError') }}</p>
            <RouterLink class="ghost" to="/groups">{{ tr('allGroupsLink') }}</RouterLink>
        </div>
        <RouterView v-else />
    </section>
</template>
