import { computed, ref } from 'vue'

// Mirrors the responsive breakpoints already used in completion.css (700px/1100px).
const mobile = typeof window === 'undefined' ? null : window.matchMedia('(max-width: 700px)')
const tablet = typeof window === 'undefined' ? null : window.matchMedia('(max-width: 1100px)')
const tick = ref(0)
mobile?.addEventListener('change', () => { tick.value++ })
tablet?.addEventListener('change', () => { tick.value++ })

export const defaultPageSize = computed(() => {
  void tick.value
  if (mobile?.matches) return 10
  if (tablet?.matches) return 25
  return 50
})

export const pageSizeOptions = [10, 25, 50, 100]
