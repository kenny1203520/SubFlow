<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '../i18n'

const props = withDefaults(defineProps<{
  date: string
  timezone: string
  today: string
  loading?: boolean
}>(), { loading: false })

const emit = defineEmits<{ change: [date: string] }>()
const { tr, locale } = useI18n()
const dateLabel = computed(() => {
  const value = new Date(`${props.date}T12:00:00`)
  return Number.isNaN(value.getTime()) ? props.date : new Intl.DateTimeFormat(locale.value, { month: 'long', day: 'numeric', weekday: 'short' }).format(value)
})

function move(date: string, days: number) {
  const next = new Date(`${date}T12:00:00`)
  next.setDate(next.getDate() + days)
  emit('change', `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, '0')}-${String(next.getDate()).padStart(2, '0')}`)
}
</script>

<template>
  <section class="ledger-date-nav card" :aria-busy="loading">
    <div class="ledger-date-nav-main">
      <button class="icon-button ledger-date-step" type="button" :aria-label="tr('previousDay')" :disabled="loading" @click="move(date, -1)">‹</button>
      <label class="ledger-date-value">
        <span class="sr-only">{{ tr('chooseDate') }}</span>
        <span class="ledger-date-display">{{ dateLabel }}</span>
        <input type="date" :value="date" :disabled="loading" @change="emit('change', ($event.target as HTMLInputElement).value)">
        <small>{{ timezone }} · {{ date }}</small>
      </label>
      <button class="icon-button ledger-date-step" type="button" :aria-label="tr('nextDay')" :disabled="loading" @click="move(date, 1)">›</button>
    </div>
    <button v-if="date !== today" class="ghost today-button" type="button" :disabled="loading" @click="emit('change', today)">{{ tr('backToToday') }}</button>
  </section>
</template>
