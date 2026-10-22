<script setup lang="ts">
withDefaults(defineProps<{
  title: string
  category: string
  status: string
  readState: string
  createdAt: string
  formattedDate: string
  message?: string
  unread?: boolean
  variant?: 'center' | 'preview'
}>(), { unread: false, variant: 'center' })
</script>

<template>
  <article class="notification-item" :class="[`notification-item--${variant}`, { 'is-unread': unread }]">
    <div class="notification-item__body">
      <div class="notification-item__meta">
        <span class="notification-item__category">{{ category }}</span>
        <span class="notification-item__read" :class="{ 'is-unread': unread }">{{ readState }}</span>
        <span class="notification-item__status">{{ status }}</span>
      </div>
      <h2 class="notification-item__title">{{ title }}</h2>
      <p v-if="message" class="notification-item__message">{{ message }}</p>
      <time class="notification-item__time" :datetime="createdAt">{{ formattedDate }}</time>
    </div>
    <div v-if="$slots.actions" class="notification-item__actions">
      <slot name="actions" />
    </div>
  </article>
</template>

<style scoped>
.notification-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  min-inline-size: 0;
  padding: 1rem 1.1rem;
  border: 1px solid var(--line);
  border-radius: .8rem;
  background: var(--surface);
}

.notification-item.is-unread { border-inline-start: 4px solid var(--brand); }
.notification-item__body { display: grid; gap: .4rem; min-inline-size: 0; }
.notification-item__meta { display: flex; flex-wrap: wrap; align-items: center; gap: .4rem; }
.notification-item__category,
.notification-item__read,
.notification-item__status {
  display: inline-flex;
  align-items: center;
  min-block-size: 1.5rem;
  padding: .15rem .55rem;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: .78rem;
  line-height: 1.2;
}
.notification-item__read.is-unread { border-color: var(--brand); color: var(--brand); font-weight: 650; }
.notification-item__title { margin: .1rem 0 0; color: var(--ink); font-size: 1rem; line-height: 1.4; overflow-wrap: anywhere; }
.notification-item__message { margin: 0; color: var(--muted); line-height: 1.5; overflow-wrap: anywhere; }
.notification-item__time { color: var(--muted); font-size: .85rem; }
.notification-item__actions { display: flex; flex: 0 0 auto; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: .45rem; }
.notification-item--preview { align-items: flex-start; padding: .75rem; border-radius: .65rem; }
.notification-item--preview .notification-item__title { font-size: .95rem; }
.notification-item--preview .notification-item__meta { gap: .3rem; }
.notification-item--preview .notification-item__category,
.notification-item--preview .notification-item__read,
.notification-item--preview .notification-item__status { font-size: .72rem; }

@media (max-width: 600px) {
  .notification-item { align-items: stretch; flex-direction: column; }
  .notification-item__actions { justify-content: flex-start; }
}
</style>
