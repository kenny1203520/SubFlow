<script setup lang="ts">
type ShareIconName = 'arrow-up-right' | 'lock' | 'expense' | 'subscription' | 'settlement' | 'empty' | 'alert'

withDefaults(defineProps<{
  name: ShareIconName
  size?: number | string
}>(), { size: 20 })
</script>

<template>
  <svg
    class="share-icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <template v-if="name === 'arrow-up-right'">
      <path d="M7 17 17 7" />
      <path d="M8 7h9v9" />
    </template>
    <template v-else-if="name === 'lock'">
      <rect x="5" y="10" width="14" height="10" rx="2" />
      <path d="M8 10V7a4 4 0 0 1 8 0v3" />
      <path d="M12 14v2" />
    </template>
    <template v-else-if="name === 'expense'">
      <path d="M5 19 19 5" />
      <path d="M7 5h5v5" />
      <path d="M17 19h-5v-5" />
    </template>
    <template v-else-if="name === 'subscription'">
      <path d="M19 8a7 7 0 1 0 1 5" />
      <path d="M19 4v4h-4" />
      <path d="M12 8v4l3 2" />
    </template>
    <template v-else-if="name === 'settlement'">
      <path d="M7 7h10l-2-2" />
      <path d="M17 17H7l2 2" />
      <path d="M17 7v4" />
      <path d="M7 17v-4" />
    </template>
    <template v-else-if="name === 'empty'">
      <rect x="4" y="5" width="16" height="14" rx="2" />
      <path d="M8 9h8M8 13h5M8 17h3" />
    </template>
    <template v-else>
      <circle cx="12" cy="12" r="8" />
      <path d="M12 8v5M12 16h.01" />
    </template>
  </svg>
</template>
