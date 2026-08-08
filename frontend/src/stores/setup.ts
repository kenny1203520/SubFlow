import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiClient } from '../api/client'

export interface SetupStatus { initialized:boolean; siteName?:string; defaultTimezone?:string; defaultCurrency?:string; allowRegistration?:boolean }
const publicApi = new ApiClient(() => '', () => {})

export const useSetupStore = defineStore('setup', () => {
  const status = ref<SetupStatus>({ initialized: true })
  const ready = ref(false)
  const initialized = computed(() => status.value.initialized)
  const allowRegistration = computed(() => !!status.value.allowRegistration)
  async function refresh() { status.value = (await publicApi.get<SetupStatus>('/setup/status')).data; ready.value = true; return status.value }
  async function initialize(input: Record<string, unknown>) { await publicApi.post('/setup/initialize', input); await refresh() }
  return { status, ready, initialized, allowRegistration, refresh, initialize }
})
