// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { pb } from '../pocketbase'
import { useAuthStore } from './auth'

function validToken() {
  const payload = btoa(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + 3600 }))
    .replace(/=/g, '').replace(/\+/g, '-').replace(/\//g, '_')
  return `e30.${payload}.signature`
}

function userRecord() {
  return { id: 'user-1', email: 'kenny@example.com', collectionId: 'users', collectionName: 'users' }
}

beforeEach(() => {
  pb.authStore.clear()
  setActivePinia(createPinia())
})

describe('auth store', () => {
  it('reacts immediately when PocketBase authenticates for the first time', async () => {
    const auth = useAuthStore()
    expect(auth.authenticated).toBe(false)

    pb.authStore.save(validToken(), userRecord())
    await nextTick()

    expect(auth.authenticated).toBe(true)
    expect(auth.token).toBe(pb.authStore.token)
    expect(auth.record?.id).toBe('user-1')
  })

  it('reacts immediately to logout', async () => {
    pb.authStore.save(validToken(), userRecord())
    const auth = useAuthStore()
    auth.logout()
    await nextTick()
    expect(auth.authenticated).toBe(false)
    expect(auth.record).toBeNull()
  })
})
