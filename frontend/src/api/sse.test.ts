import { afterEach, describe, expect, it, vi } from 'vitest'
import { SSEClient } from './sse'

// A stream that stays open until closed, so a test can control exactly when
// the "connection" ends instead of racing a real network stream.
function openStream() {
  let close: () => void = () => {}
  const stream = new ReadableStream<Uint8Array>({ start(controller) { close = () => controller.close() } })
  return { stream, close: () => close() }
}
function flush() { return new Promise(resolve => setTimeout(resolve, 0)) }

afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })

describe('SSEClient', () => {
  it('does not fire onReconnect on the very first successful connection', async () => {
    const { stream } = openStream()
    const fetchMock = vi.fn().mockResolvedValue(new Response(stream, { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    const onReconnect = vi.fn()
    const client = new SSEClient(() => 'token', () => {}, () => {}, onReconnect)
    void client.start()
    await flush()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(onReconnect).not.toHaveBeenCalled()
    client.stop()
  })

  it('fires onReconnect once the connection re-establishes after a drop', async () => {
    vi.useFakeTimers()
    const first = openStream()
    const second = openStream()
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(first.stream, { status: 200 }))
      .mockResolvedValueOnce(new Response(second.stream, { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    const onReconnect = vi.fn()
    const client = new SSEClient(() => 'token', () => {}, () => {}, onReconnect)
    void client.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(onReconnect).not.toHaveBeenCalled()

    first.close() // the server ends the stream -- a real drop, not a clean stop()
    await vi.advanceTimersByTimeAsync(0)
    await vi.advanceTimersByTimeAsync(1000) // the first reconnect delay

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(onReconnect).toHaveBeenCalledOnce()
    client.stop()
  })

  it('stops looping and reports unauthorized on a 401 without retrying', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 401 }))
    vi.stubGlobal('fetch', fetchMock)
    const onUnauthorized = vi.fn()
    const client = new SSEClient(() => 'token', () => {}, onUnauthorized)
    await client.start()
    expect(onUnauthorized).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('backs off with doubling delays that cap at 30s', async () => {
    vi.useFakeTimers()
    const fetchMock = vi.fn().mockRejectedValue(new Error('network down'))
    vi.stubGlobal('fetch', fetchMock)
    const client = new SSEClient(() => 'token', () => {}, () => {})
    void client.start()

    // Ramp-up: 1s, 2s, 4s, 8s, 16s between attempts (doubling each time).
    const waits = [1000, 2000, 4000, 8000, 16000]
    let expectedCalls = 1
    await vi.advanceTimersByTimeAsync(0)
    expect(fetchMock).toHaveBeenCalledTimes(expectedCalls)
    for (const wait of waits) {
      await vi.advanceTimersByTimeAsync(wait)
      expectedCalls++
      expect(fetchMock).toHaveBeenCalledTimes(expectedCalls)
    }

    // Once the delay would exceed 30s it stays capped there indefinitely.
    await vi.advanceTimersByTimeAsync(30000)
    expectedCalls++
    expect(fetchMock).toHaveBeenCalledTimes(expectedCalls)
    await vi.advanceTimersByTimeAsync(30000)
    expectedCalls++
    expect(fetchMock).toHaveBeenCalledTimes(expectedCalls)

    client.stop()
  })
})
