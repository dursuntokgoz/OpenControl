import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { en, type MessageKey } from './en'
import { tr } from './tr'

export type Lang = 'en' | 'tr'

const catalogs: Record<Lang, Record<MessageKey, string>> = {
  en: en as unknown as Record<MessageKey, string>,
  tr,
}

const STORAGE_KEY = 'serverpanel.lang'

export function detectInitialLang(): Lang {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'en' || stored === 'tr') return stored
  } catch {
    // storage unavailable (privacy mode) — fall through
  }
  return 'en'
}

interface I18nContextValue {
  lang: Lang
  setLang: (l: Lang) => void
  t: (key: MessageKey) => string
}

// Undefined until the provider mounts; consumers must be inside <I18nProvider>.
const I18nContext = createContext<I18nContextValue | undefined>(undefined)

export function translate(lang: Lang, key: MessageKey): string {
  return catalogs[lang][key] ?? catalogs.en[key] ?? key
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(detectInitialLang)

  const setLang = useCallback((l: Lang) => {
    setLangState(l)
    try {
      localStorage.setItem(STORAGE_KEY, l)
      document.documentElement.lang = l
    } catch {
      // non-fatal
    }
  }, [])

  const value = useMemo<I18nContextValue>(
    () => ({ lang, setLang, t: (key) => translate(lang, key) }),
    [lang, setLang],
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext)
  if (!ctx) {
    throw new Error('useI18n must be used within I18nProvider')
  }
  return ctx
}
