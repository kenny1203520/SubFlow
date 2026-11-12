<script setup lang="ts">
import { useI18n } from '../i18n'
import type { OAuthProviderInfo } from '../stores/auth'
import OAuthProviderLogo from './OAuthProviderLogo.vue'

defineProps<{
  providers: OAuthProviderInfo[]
  busy?: boolean
}>()

const emit = defineEmits<{ select: [provider: string] }>()
const { tr } = useI18n()
</script>

<template>
  <div class="oauth-provider-list" role="group" :aria-label="tr('oauthLoginOptions')">
    <button
      v-for="provider in providers"
      :key="provider.name"
      type="button"
      class="oauth-provider-button"
      :class="{ 'oauth-provider--google': provider.name === 'google' }"
      :disabled="busy"
      @click="emit('select', provider.name)"
    >
      <OAuthProviderLogo :provider="provider" />
      <span class="oauth-provider-name" :translate="provider.name === 'google' ? 'no' : undefined">
        {{ provider.name === 'google' ? tr('continueWithGoogle') : provider.displayName || provider.name }}
      </span>
    </button>
  </div>
</template>

<style scoped>
.oauth-provider-list {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}

.oauth-provider-button {
  display: inline-flex;
  min-inline-size: min(100%, 9rem);
  min-block-size: 48px;
  flex: 1 1 9rem;
  align-items: center;
  justify-content: center;
  gap: 9px;
  padding: 7px 12px;
  border: 1px solid var(--line-strong);
  border-radius: 10px;
  background: var(--surface-soft);
  color: var(--ink);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  line-height: 1.35;
  text-align: center;
  overflow-wrap: anywhere;
  cursor: pointer;
  touch-action: manipulation;
  -webkit-tap-highlight-color: color-mix(in srgb, var(--brand) 16%, transparent);
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}

.oauth-provider-button:hover:not(:disabled) {
  border-color: var(--brand);
  background: var(--brand-soft);
}

.oauth-provider-button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--brand) 48%, transparent);
  outline-offset: 2px;
}

.oauth-provider-button:disabled {
  cursor: wait;
  opacity: 0.62;
}

.oauth-provider-name {
  min-inline-size: 0;
  overflow-wrap: anywhere;
}

.oauth-provider--google {
  border-color: #747775;
  background: #fff;
  color: #1f1f1f;
}

.oauth-provider--google:hover:not(:disabled) {
  border-color: #1f1f1f;
  background: #f8faff;
}

:global(html[data-theme='dark'] .oauth-provider--google) {
  border-color: #8e918f;
  background: #131314;
  color: #e3e3e3;
}

:global(html[data-theme='dark'] .oauth-provider--google:hover:not(:disabled)) {
  border-color: #e3e3e3;
  background: #303134;
}

@media (prefers-reduced-motion: reduce) {
  .oauth-provider-button {
    transition: none;
  }
}
</style>
