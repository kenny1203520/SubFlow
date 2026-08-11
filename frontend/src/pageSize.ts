import { computed, ref } from 'vue'

// Mirrors the responsive breakpoints already used in completion.css (700px/1100px).
// matchMedia is absent under jsdom and during SSR, so fall back to the desktop size.
const query = typeof window !== 'undefined' && typeof window.matchMedia === 'function' ? (value: string) => window.matchMedia(value) : null
const mobile = query?.('(max-width: 700px)') ?? null
const tablet = query?.('(max-width: 1100px)') ?? null
const tick = ref(0)
const bump = () => { tick.value++ }
mobile?.addEventListener?.('change', bump)
tablet?.addEventListener?.('change', bump)

export const defaultPageSize = computed(() => {
  void tick.value
  if (mobile?.matches) return 10
  if (tablet?.matches) return 25
  return 50
})

export const pageSizeOptions = [10, 25, 50, 100]
