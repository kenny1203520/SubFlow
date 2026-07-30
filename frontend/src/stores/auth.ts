import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { pb } from '../pocketbase'

export const useAuthStore = defineStore('auth', () => {
  const record = ref(pb.authStore.record)
  const authToken = ref(pb.authStore.token)
  const authValid = ref(pb.authStore.isValid)
  const ready = ref(false)

  // PocketBase's authStore is not a Vue reactive object. Mirror all values
  // that determine session state so the first login invalidates this computed.
  const authenticated = computed(() => authValid.value && !!authToken.value && !!record.value)
  const token = computed(() => authToken.value)
  const name = computed(() => String(record.value?.name || record.value?.email || '使用者'))

  pb.authStore.onChange((nextToken, nextRecord) => {
    authToken.value = nextToken
    authValid.value = pb.authStore.isValid
    record.value = nextRecord
  })

  async function initialize() {
    if (pb.authStore.isValid) {
      try {
        await pb.collection('users').authRefresh()
      } catch {
        pb.authStore.clear()
      }
    }
    authToken.value = pb.authStore.token
    authValid.value = pb.authStore.isValid
    record.value = pb.authStore.record
    ready.value = true
  }

  async function login(email: string, password: string) {
    await pb.collection('users').authWithPassword(email, password)
    authToken.value = pb.authStore.token
    authValid.value = pb.authStore.isValid
    record.value = pb.authStore.record
  }

  async function register(input: { email: string; password: string; name: string }) {
    await pb.collection('users').create({ email: input.email, password: input.password, passwordConfirm: input.password, name: input.name, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone })
    await login(input.email, input.password)
  }

  async function updateProfile(input: { name: string; timezone: string }) {
    if (!record.value) throw new Error('尚未登入')
    record.value = await pb.collection('users').update(record.value.id, input)
  }

  function logout() {
    pb.authStore.clear()
    authToken.value = ''
    authValid.value = false
    record.value = null
  }

  return { record, ready, authenticated, token, name, initialize, login, register, updateProfile, logout }
})
