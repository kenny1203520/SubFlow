<script setup lang="ts">
import { computed, getCurrentInstance } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: string
  label: string
  id?: string
  type?: string
  autocomplete?: string
  placeholder?: string
  required?: boolean
  minlength?: number
	maxlength?: number
  disabled?: boolean
  help?: string
  error?: string
}>(), { type: 'text' })
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const uid = getCurrentInstance()?.uid ?? Math.floor(Math.random() * 100000)
const inputId = computed(() => props.id || `base-input-${uid}`)
const helpId = computed(() => `${inputId.value}-help`)
const errorId = computed(() => `${inputId.value}-error`)
const describedBy = computed(() => [props.help ? helpId.value : '', props.error ? errorId.value : ''].filter(Boolean).join(' ') || undefined)
</script>

<template>
  <label class="base-input" :for="inputId">
    <span class="base-input-label">{{ label }}</span>
    <span class="base-input-control">
      <input :id="inputId" :value="modelValue" :type="type" :autocomplete="autocomplete" :placeholder="placeholder" :required="required" :minlength="minlength" :maxlength="maxlength" :disabled="disabled" :aria-invalid="!!error" :aria-describedby="describedBy" @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)">
      <slot name="trailing" />
    </span>
    <small v-if="help" :id="helpId" class="base-input-help">{{ help }}</small>
    <small v-if="error" :id="errorId" class="base-input-error">{{ error }}</small>
  </label>
</template>
