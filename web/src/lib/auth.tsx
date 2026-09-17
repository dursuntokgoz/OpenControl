import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { api, type MeResponse } from '../lib/api'

interface AuthState {
  user: MeResponse | null
  csrfToken: string
  loading: boolean
  error: string | null
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<MeResponse | null>(null)
  const [csrfToken, setCsrfToken] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.me()
      .then((u) => { setUser(u); setLoading(false) })
      .catch(() => { setUser(null); setLoading(false) })
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    setError(null)
    try {
      const res = await api.login(username, password)
      setCsrfToken(res.csrfToken)
      const me = await api.me()
      setUser(me)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Login failed'
      setError(msg)
      throw e
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.logout(csrfToken)
    } catch {
      // ignore — clear client state regardless
    }
    setUser(null)
    setCsrfToken('')
  }, [csrfToken])

  const value = useMemo<AuthState>(
    () => ({ user, csrfToken, loading, error, login, logout }),
    [user, csrfToken, loading, error, login, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
