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
const teleportTarget = ref<string | HTMLElement>('body')
const open = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) })
const useTeleport = import.meta.env.MODE !== 'test'

function close() { open.value = false }
function toggle() { open.value = !open.value }
function positionPanel() {
  if (!root.value || !panel.value) return
  const rect = root.value.getBoundingClientRect()
  const viewport = window.visualViewport
  const viewportHeight = viewport?.height ?? window.innerHeight
  const viewportWidth = viewport?.width ?? window.innerWidth
  const viewportTop = viewport?.offsetTop ?? 0
  const viewportLeft = viewport?.offsetLeft ?? 0
  if (props.mobileSheet && window.innerWidth <= 700) {
    const availableHeight = Math.min(viewportHeight * .76, 640)
    panelStyle.value = {
      maxHeight: `${availableHeight}px`,
      '--dropdown-available-height': `${availableHeight}px`,
      '--dropdown-sheet-bottom': `${Math.max(0, window.innerHeight - viewportTop - viewportHeight)}px`,
      '--dropdown-sheet-left': `${viewportLeft}px`,
      '--dropdown-sheet-width': `${viewportWidth}px`,
    }
    return
  }
  const padding = 12
  const offset = 8
  const viewportRight = viewportLeft + viewportWidth
  const viewportBottom = viewportTop + viewportHeight
  const width = Math.min(Math.max(rect.width, panel.value.offsetWidth), viewportWidth - padding * 2)
  const left = props.placement === 'bottom-end'
    ? Math.max(viewportLeft + padding, Math.min(rect.right - width, viewportRight - width - padding))
    : Math.max(viewportLeft + padding, Math.min(rect.left, viewportRight - width - padding))
  const spaceBelow = Math.max(0, viewportBottom - Math.max(viewportTop, rect.bottom) - padding - offset)
  const spaceAbove = Math.max(0, Math.min(rect.top, viewportBottom) - viewportTop - padding - offset)
  const openUpward = panel.value.offsetHeight > spaceBelow && spaceAbove > spaceBelow
  const availableHeight = openUpward ? spaceAbove : spaceBelow
  const visibleHeight = Math.min(panel.value.offsetHeight, availableHeight)
  const top = openUpward
    ? Math.max(viewportTop + padding, Math.min(rect.top, viewportBottom) - visibleHeight - offset)
    : Math.min(Math.max(rect.bottom, viewportTop) + offset, viewportBottom - padding)

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
function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape' && open.value) {
    const activePanel = event.target instanceof Element ? event.target.closest('.base-dropdown-panel') : null
    if (activePanel && activePanel !== panel.value) return
    event.preventDefault()
    event.stopPropagation()
    close()
    root.value?.querySelector<HTMLElement>('button, [role="combobox"]')?.focus({ preventScroll: true })
  }
}

watch(open, value => {
  if (!value) return
  // A modal makes nodes outside its DOM subtree inert, even with high z-index.
  // Keep nested menus within their parent's panel so outside-click detection
  // does not close the parent when an icon or another nested option is chosen.
  teleportTarget.value = root.value?.closest<HTMLElement>('.base-dropdown-panel, dialog[open]') || 'body'
  nextTick(() => { positionPanel(); emit('opened') })
})
onMounted(() => {
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKeyDown, true)
  window.addEventListener('resize', positionPanel)
  window.addEventListener('scroll', positionPanel, true)
  window.visualViewport?.addEventListener('resize', positionPanel)
  window.visualViewport?.addEventListener('scroll', positionPanel)
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeyDown, true)
  window.removeEventListener('resize', positionPanel)
  window.removeEventListener('scroll', positionPanel, true)
  window.visualViewport?.removeEventListener('resize', positionPanel)
  window.visualViewport?.removeEventListener('scroll', positionPanel)
})
</script>
<template>
  <div ref="root" class="base-dropdown">
    <slot name="trigger" :open="open" :toggle="toggle" :close="close" />
    <Teleport :to="teleportTarget" :disabled="!useTeleport">
      <div v-if="open" ref="panel" class="base-dropdown-panel" :class="{ 'base-dropdown-sheet': mobileSheet }" :style="panelStyle" :role="panelRole" :aria-label="panelLabel">
        <slot :close="close" />
      </div>
    </Teleport>
  </div>
</template>
