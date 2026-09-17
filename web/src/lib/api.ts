export interface Health {
  status: string
  version: string
}

export interface LoginResponse {
  userId: number
  username: string
  role: string
  csrfToken: string
}

export interface MeResponse {
  id: number
  username: string
  email: string
  role: string
  totpEnabled: boolean
  language: string
  contactEmail?: string
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

export async function postJSON<T>(path: string, body: unknown, csrfToken?: string): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  }
  if (csrfToken) {
    headers['X-CSRF-Token'] = csrfToken
  }
  const res = await fetch(path, {
    method: 'POST',
    credentials: 'same-origin',
    headers,
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const text = await res.text()
    let msg = `POST ${path} failed with ${res.status}`
    try {
      const j = JSON.parse(text) as { error?: string }
      if (j.error) msg = j.error
    } catch {
      // ignore
    }
    throw new ApiError(res.status, msg)
  }
  return (await res.json()) as T
}

export const api = {
  health: (): Promise<Health> => getJSON<Health>('/healthz'),
  login: (username: string, password: string) =>
    postJSON<LoginResponse>('/api/v1/auth/login', { username, password }),
  logout: (csrfToken?: string) =>
    postJSON<{ status: string }>('/api/v1/auth/logout', {}, csrfToken),
  me: (): Promise<MeResponse> => getJSON<MeResponse>('/api/v1/auth/me'),
}
