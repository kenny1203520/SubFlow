<script setup lang="ts">
import { computed } from 'vue'
import type { OAuthProviderInfo } from '../stores/auth'

const props = withDefaults(defineProps<{ provider: OAuthProviderInfo; size?: 'normal' | 'small' }>(), { size: 'normal' })

const logoSource = computed(() => {
  const logo = props.provider.logo?.trim()
  if (!logo || !/^<svg(?:\s|>)/i.test(logo)) return ''
  return `data:image/svg+xml,${encodeURIComponent(logo)}`
})
</script>

<template>
  <span class="oauth-provider-logo" :class="{ 'oauth-provider-logo--small': props.size === 'small' }" aria-hidden="true">
    <img v-if="logoSource" :src="logoSource" width="20" height="20" alt="" />
    <svg v-else viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="m9.2 14.8 5.6-5.6" />
      <path d="M7.8 16.2H7a4 4 0 0 1 0-8h3" />
      <path d="M16.2 7.8h.8a4 4 0 0 1 0 8h-3" />
    </svg>
  </span>
</template>

<style scoped>
.oauth-provider-logo {
  display: inline-grid;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  place-items: center;
  overflow: hidden;
  border-radius: 7px;
  background: #fff;
}

.oauth-provider-logo img {
  display: block;
  width: 20px;
  height: 20px;
  object-fit: contain;
}

.oauth-provider-logo svg {
  width: 19px;
  height: 19px;
  stroke: #687386;
  stroke-width: 1.7;
  stroke-linecap: round;
}

.oauth-provider-logo--small {
  width: 24px;
  height: 24px;
  flex-basis: 24px;
  border-radius: 6px;
}

.oauth-provider-logo--small img {
  width: 18px;
  height: 18px;
}
</style>
