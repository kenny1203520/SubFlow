<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
const props = withDefaults(defineProps<{ modelValue: boolean; panelLabel?: string }>(), { panelLabel: 'Options' })
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; opened: [] }>()
const root = ref<HTMLElement | null>(null)
const open = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) })
function close() { open.value = false }
function toggle() { open.value = !open.value }
function onPointerDown(event: PointerEvent) { if (open.value && root.value && !root.value.contains(event.target as Node)) close() }
function onKeyDown(event: KeyboardEvent) { if (event.key === 'Escape') close() }
watch(open, value => { if (value) nextTick(() => emit('opened')) })
onMounted(() => { document.addEventListener('pointerdown', onPointerDown); document.addEventListener('keydown', onKeyDown) })
onBeforeUnmount(() => { document.removeEventListener('pointerdown', onPointerDown); document.removeEventListener('keydown', onKeyDown) })
</script>
<template><div ref="root" class="base-dropdown"><slot name="trigger" :open="open" :toggle="toggle" :close="close"/><div v-if="open" class="base-dropdown-panel" role="listbox" :aria-label="panelLabel"><slot :close="close"/></div></div></template>
