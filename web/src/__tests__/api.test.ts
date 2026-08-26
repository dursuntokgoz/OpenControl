import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api } from '../lib/api'

// Minimal Response stand-in for environments under test.
function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as unknown as Response
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('api.health', () => {
  it('returns parsed health payload', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => jsonResponse(200, { status: 'ok', version: '0.1.0' })),
    )
    const health = await api.health()
    expect(health.status).toBe('ok')
    expect(health.version).toBe('0.1.0')
  })

  it('throws ApiError on non-2xx responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => jsonResponse(503, {})),
    )
    await expect(api.health()).rejects.toMatchObject({
      name: 'ApiError',
      status: 503,
    })
    await expect(api.health()).rejects.toBeInstanceOf(ApiError)
  })
})
