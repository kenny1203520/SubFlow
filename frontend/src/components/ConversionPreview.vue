<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { Currency, RateMode } from '../api/types'
import { majorToMinor, minorToMajor } from '../api/money'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'
import { currencyLabel } from '../currency'

const props = defineProps<{ from: Currency; to: Currency; amount: string; date: string; mode: RateMode; manualRate: string }>()
const emit = defineEmits<{ (event: 'validity', value: boolean): void }>()
const workspace = useWorkspaceStore()
const { locale, tr, formatDate } = useI18n()
const loading = ref(false), failure = ref(''), automaticRate = ref(''), provider = ref(''), effectiveDate = ref(''), stale = ref(false)
let request = 0, timer: ReturnType<typeof setTimeout> | undefined

const sameCurrency = computed(() => props.from === props.to)
const activeRate = computed(() => props.mode === 'manual' ? props.manualRate.trim() : automaticRate.value)
const numericRate = computed(() => Number(activeRate.value))
const valid = computed(() => sameCurrency.value || (Number.isFinite(numericRate.value) && numericRate.value > 0 && !failure.value && !loading.value))
const converted = computed(() => {
  if (!valid.value || !props.amount) return ''
  const original = minorToMajor(majorToMinor(props.amount, props.from), props.from)
  if (!Number.isFinite(original)) return ''
  try { return new Intl.NumberFormat(locale.value, { style: 'currency', currency: props.to }).format(original * (sameCurrency.value ? 1 : numericRate.value)) } catch { return '' }
})
const rateText = computed(() => sameCurrency.value ? '1' : activeRate.value)

function clearQuote() { automaticRate.value = ''; provider.value = ''; effectiveDate.value = ''; stale.value = false }
async function quote() {
  const id = ++request
  if (timer) clearTimeout(timer)
  failure.value = ''
  if (!props.from || !props.to || !props.date || props.mode === 'manual') { clearQuote(); loading.value = false; return }
  if (sameCurrency.value) { automaticRate.value = '1'; effectiveDate.value = props.date; loading.value = false; return }
  loading.value = true
  timer = setTimeout(async () => {
    try {
      const value = await workspace.quoteRate(props.from, props.to, props.date)
      if (id !== request) return
      automaticRate.value = value.rate; provider.value = value.provider; effectiveDate.value = value.effectiveDate; stale.value = Boolean(value.stale)
    } catch {
      if (id === request) { clearQuote(); failure.value = tr('rateUnavailable') }
    } finally { if (id === request) loading.value = false }
  }, 220)
}
watch(() => [props.from, props.to, props.date, props.mode], () => void quote(), { immediate: true })
watch(valid, value => emit('validity', value), { immediate: true })
onBeforeUnmount(() => { request++; if (timer) clearTimeout(timer) })
</script>

<template>
  <section class="conversion-preview" :class="{ loading, invalid: !valid }" aria-live="polite">
    <div class="conversion-preview-title"><strong>{{ tr('exchangeRate') }}</strong><span>{{ mode === 'manual' ? tr('manualRate') : tr('automaticRate') }}</span></div>
    <p v-if="loading">{{ tr('rateLoading') }}</p>
    <template v-else-if="failure"><p class="form-error">{{ failure }}</p></template>
    <template v-else-if="valid">
      <strong class="conversion-equation">1 {{ currencyLabel(from, locale) }} = {{ rateText }} {{ currencyLabel(to, locale) }}</strong>
      <p v-if="sameCurrency">{{ tr('sameCurrencyRate') }}</p>
      <p v-else-if="converted">{{ tr('convertedAmount', { amount: converted }) }}</p>
      <small v-if="mode === 'automatic' && effectiveDate">{{ tr('rateEffectiveDate') }}: {{ formatDate(effectiveDate, { dateStyle: 'medium' }) }}<template v-if="provider"> · {{ tr('rateProvider') }}: {{ provider }}</template><template v-if="stale"> · {{ tr('rateStale') }}</template></small>
      <small v-else-if="mode === 'manual'">{{ tr('rateManualHint') }}</small>
    </template>
    <p v-else class="form-error">{{ mode === 'manual' ? tr('rateManualInvalid') : tr('rateUnavailable') }}</p>
  </section>
</template>
