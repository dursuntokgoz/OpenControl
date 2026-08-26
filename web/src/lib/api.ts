// API client helpers. All requests are same-origin; the backend sets strict
// security headers and (from phase 1 on) CSRF protection.

export interface Health {
  status: string
  version: string
}

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path, {
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
  })
  if (!res.ok) {
    throw new ApiError(res.status, `GET ${path} failed with ${res.status}`)
  }
  return (await res.json()) as T
}

export const api = {
  health: (): Promise<Health> => getJSON<Health>('/healthz'),
}
