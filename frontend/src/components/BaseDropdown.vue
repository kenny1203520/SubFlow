<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: boolean
  panelLabel?: string
  panelRole?: string
  placement?: 'bottom-start' | 'bottom-end'
  mobileSheet?: boolean
}>(), { panelLabel: 'Options', panelRole: 'listbox', placement: 'bottom-start', mobileSheet: false })

const emit = defineEmits<{ 'update:modelValue': [value: boolean]; opened: [] }>()
const root = ref<HTMLElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const panelStyle = ref<Record<string, string>>({})
const open = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) })
const useTeleport = import.meta.env.MODE !== 'test'

function close() { open.value = false }
function toggle() { open.value = !open.value }
function positionPanel() {
  if (!root.value || !panel.value) return
  const rect = root.value.getBoundingClientRect()
  const padding = 12
  const offset = 8
  const width = Math.min(Math.max(rect.width, panel.value.offsetWidth), window.innerWidth - padding * 2)
  const left = props.placement === 'bottom-end'
    ? Math.max(padding, Math.min(rect.right - width, window.innerWidth - width - padding))
    : Math.max(padding, Math.min(rect.left, window.innerWidth - width - padding))
  const spaceBelow = Math.max(0, window.innerHeight - rect.bottom - padding - offset)
  const spaceAbove = Math.max(0, rect.top - padding - offset)
  const openUpward = panel.value.offsetHeight > spaceBelow && spaceAbove > spaceBelow
  const availableHeight = openUpward ? spaceAbove : spaceBelow
  const visibleHeight = Math.min(panel.value.offsetHeight, availableHeight)
  const top = openUpward
    ? Math.max(padding, rect.top - visibleHeight - offset)
    : Math.min(rect.bottom + offset, window.innerHeight - padding)

  panelStyle.value = {
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    maxHeight: `${availableHeight}px`,
    '--dropdown-available-height': `${availableHeight}px`,
  }
}
function onPointerDown(event: PointerEvent) {
  const target = event.target as Node
  if (open.value && root.value && !root.value.contains(target) && !panel.value?.contains(target)) close()
}
function onKeyDown(event: KeyboardEvent) { if (event.key === 'Escape') close() }

watch(open, value => { if (value) nextTick(() => { positionPanel(); emit('opened') }) })
onMounted(() => {
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKeyDown)
  window.addEventListener('resize', positionPanel)
  window.addEventListener('scroll', positionPanel, true)
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('resize', positionPanel)
  window.removeEventListener('scroll', positionPanel, true)
})
</script>
<template>
  <div ref="root" class="base-dropdown">
    <slot name="trigger" :open="open" :toggle="toggle" :close="close" />
    <Teleport to="body" :disabled="!useTeleport">
      <div v-if="open" ref="panel" class="base-dropdown-panel" :class="{ 'base-dropdown-sheet': mobileSheet }" :style="panelStyle" :role="panelRole" :aria-label="panelLabel">
        <slot :close="close" />
      </div>
    </Teleport>
  </div>
</template>
