<script setup lang="ts">
import { ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useWorkspaceStore } from "../stores/workspace";
import { useI18n } from '../i18n'
const route = useRoute(),
    router = useRouter(),
    workspace = useWorkspaceStore(),
    done = ref(false);
const {tr}=useI18n()
async function accept() {
    await workspace.acceptInvitation(String(route.params.token));
    if (!workspace.error) {
        done.value = true;
        setTimeout(() => router.push("/"), 700);
    }
}
</script>
<template>
    <section class="page narrow">
        <div class="card invite-accept">
            <p class="eyebrow">{{tr('invitation')}}</p>
            <h1>{{tr('joinGroup')}}</h1>
            <p>{{tr('inviteDesc')}}</p>
            <button class="primary wide" :disabled="done" @click="accept">
                {{ done ? tr('joined') : tr('acceptInvitation') }}
            </button>
            <p v-if="workspace.error" class="form-error">{{ workspace.error }}</p>
        </div>
    </section>
</template>
