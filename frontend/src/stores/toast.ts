import { ref } from 'vue'
import { defineStore } from 'pinia'

export interface Toast { id: number; type: 'success' | 'error'; message: string }

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([])
  let nextId = 1

  function dismiss(id: number) {
    toasts.value = toasts.value.filter(toast => toast.id !== id)
  }

  function push(type: Toast['type'], message: string) {
    const id = nextId++
    toasts.value.push({ id, type, message })
    setTimeout(() => dismiss(id), 4000)
  }

  return { toasts, push, dismiss }
})
