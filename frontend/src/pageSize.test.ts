// @vitest-environment jsdom
// pageSize.ts calls window.matchMedia once at module load time and captures
// the returned MediaQueryList objects at module scope, so window.matchMedia
// must be stubbed *before* the module is imported for a given viewport --
// reassigning it afterwards has no effect on the already-captured objects.
// Each test therefore stubs the viewport, then dynamically re-imports the
// module fresh (vi.resetModules) to pick it up.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

function stubViewport(width: number) {
  window.matchMedia = (query: string) => {
    const match = /max-width:\s*(\d+)px/.exec(query)
    const maxWidth = match ? Number(match[1]) : Infinity
    return { matches: width <= maxWidth, media: query, addEventListener: () => {}, removeEventListener: () => {} } as unknown as MediaQueryList
  }
}

beforeEach(() => vi.resetModules())
afterEach(() => vi.unstubAllGlobals())

describe('defaultPageSize', () => {
  it('is 10 on mobile widths (<=700px)', async () => {
    stubViewport(600)
    const { defaultPageSize } = await import('./pageSize')
    expect(defaultPageSize.value).toBe(10)
  })

  it('is 25 on tablet widths (<=1100px)', async () => {
    stubViewport(900)
    const { defaultPageSize } = await import('./pageSize')
    expect(defaultPageSize.value).toBe(25)
  })

  it('is 50 on desktop widths (>1100px)', async () => {
    stubViewport(1400)
    const { defaultPageSize } = await import('./pageSize')
    expect(defaultPageSize.value).toBe(50)
  })

  it('falls back to the desktop size when matchMedia is unavailable (SSR/older environments)', async () => {
    // @ts-expect-error -- simulating an environment without matchMedia
    window.matchMedia = undefined
    const { defaultPageSize } = await import('./pageSize')
    expect(defaultPageSize.value).toBe(50)
  })
})

describe('pageSizeOptions', () => {
  it('offers a fixed, ascending set of page sizes', async () => {
    const { pageSizeOptions } = await import('./pageSize')
    expect(pageSizeOptions).toEqual([10, 25, 50, 100])
  })
})
