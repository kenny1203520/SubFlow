<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'mark' | 'lockup'
  size?: 'sm' | 'md' | 'lg'
  alt?: string
}>(), {
  variant: 'lockup',
  size: 'md',
  alt: 'SubFlow',
})
const logoUrl = '/favicon.svg'
</script>

<template>
  <span class="app-logo" :class="[`app-logo-${variant}`, `app-logo-${size}`]">
    <img
      :src="logoUrl"
      :alt="variant === 'mark' ? alt : ''"
      :aria-hidden="variant === 'lockup' ? 'true' : undefined"
    >
    <strong v-if="variant === 'lockup'">SubFlow</strong>
  </span>
</template>

<style scoped>
.app-logo {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: inherit;
  line-height: 1;
  white-space: nowrap;
}

.app-logo img {
  display: block;
  flex: 0 0 auto;
  width: 36px;
  height: 36px;
  object-fit: contain;
}

.app-logo strong {
  color: inherit;
  font-size: inherit;
  font-weight: 800;
  letter-spacing: -.03em;
}

.app-logo-sm img {
  width: 24px;
  height: 24px;
}

.app-logo-lg img {
  width: 64px;
  height: 64px;
}

.app-logo-mark {
  gap: 0;
}
</style>
