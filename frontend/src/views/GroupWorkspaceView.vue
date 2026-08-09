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
const canReadAudit = computed(() => workspace.groupPermissions.includes('*') || workspace.groupPermissions.includes('group.audit.read'));
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
                <div v-if="group" class="workspace-facts"><span>{{currencyLabel(group.currency)}}</span><span>{{timezoneLabel(group.timezone)}}</span><span>{{tr('records',{count:workspace.members.length})}}</span><RouterLink :to="`/groups/${groupId}/settings`">{{tr('settings')}} →</RouterLink></div>
            </div>
        </div>
        <nav class="group-tabs" :aria-label="tr('groupWorkspace')">
            <RouterLink :to="`/groups/${groupId}/overview`">{{
                tr("groupOverview")
                }}</RouterLink>
            <RouterLink :to="`/groups/${groupId}/expenses`">{{
                tr("splitExpenses")
                }}</RouterLink>
            <RouterLink :to="`/groups/${groupId}/subscriptions`">{{
                tr("manageSubscriptions")
                }}</RouterLink>
            <RouterLink :to="`/groups/${groupId}/members`">{{
                tr("members")
                }}</RouterLink>
            <RouterLink v-if="canReadAudit" :to="`/groups/${groupId}/audit`">{{ tr("auditLogs") }}</RouterLink>
            <RouterLink :to="`/groups/${groupId}/settings`">{{
                tr("settings")
                }}</RouterLink>
        </nav>
        <RouterView />
    </section>
</template>
