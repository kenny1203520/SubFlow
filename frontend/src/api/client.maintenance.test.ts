import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiClient } from './client'
import { isMaintenanceBlocked, setMaintenanceBlocked } from '../runtime/maintenance'

afterEach(() => {
  vi.unstubAllGlobals()
  setMaintenanceBlocked(false)
})

describe('maintenance protocol', () => {
  it('identifies the client protocol version on API requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { ok: true } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await new ApiClient(() => '', () => {}).get('/groups')

    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).headers).toBeInstanceOf(Headers)
    expect(((fetchMock.mock.calls[0]?.[1] as RequestInit).headers as Headers).get('X-SubFlow-Protocol')).toBe('1')
  })

  it('blocks further business requests after the backend reports maintenance', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: 'maintenance', message: 'Maintenance' } }), { status: 503 }))
    vi.stubGlobal('fetch', fetchMock)
    const client = new ApiClient(() => '', () => {})

    await expect(client.get('/groups')).rejects.toMatchObject({ status: 503, code: 'maintenance' })
    expect(isMaintenanceBlocked()).toBe(true)
    await expect(client.post('/groups', { name: 'Home' })).rejects.toMatchObject({ status: 503, code: 'maintenance' })
    expect(fetchMock).toHaveBeenCalledOnce()
  })
})
