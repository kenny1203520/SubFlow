// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { pb } from '../pocketbase'
import { useAuthStore } from './auth'
import { useToastStore } from './toast'

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
afterEach(() => vi.unstubAllGlobals())

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

  it('waits for the audited sign out and rejects duplicate submissions', async () => {
    pb.authStore.save(validToken(), userRecord())
    const auth = useAuthStore()
    let complete: ((response: Response) => void) | undefined
    const fetcher = vi.fn((_url: string) => new Promise<Response>(resolve => { complete = resolve }))
    vi.stubGlobal('fetch', fetcher)
    const pending = auth.signOut()
    expect(auth.signingOut).toBe(true)
    expect(auth.authenticated).toBe(true)
    expect(await auth.signOut()).toBe(false)
    expect(fetcher).toHaveBeenCalledTimes(1)
    expect(fetcher.mock.calls[0]?.[0]).toBe('/api/subflow/v1/auth/logout')
    complete?.(new Response(JSON.stringify({ data: { recorded: true } }), { status: 200 }))
    expect(await pending).toBe(true)
    expect(auth.authenticated).toBe(false)
    expect(auth.signingOut).toBe(false)
  })

  it.each(['network', 'audit'])('retains the session and exposes %s sign-out failure', async failure => {
    pb.authStore.save(validToken(), userRecord())
    const auth = useAuthStore()
    vi.stubGlobal('fetch', failure === 'network' ? vi.fn().mockRejectedValue(new Error('offline')) : vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: 'audit_unavailable' } }), { status: 503 })))
    expect(await auth.signOut()).toBe(false)
    expect(auth.authenticated).toBe(true)
    expect(auth.signingOut).toBe(false)
    expect(useToastStore().toasts.at(-1)?.type).toBe('error')
  })
})

