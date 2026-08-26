import type { ReactNode } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useI18n } from '../i18n'

export function Layout({ children }: { children: ReactNode }) {
  const { t, lang, setLang } = useI18n()
  const location = useLocation()
  const isAdmin = location.pathname.startsWith('/admin')

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-panel-border bg-panel-surface">
        <div className="mx-auto max-w-7xl px-4 h-14 flex items-center justify-between">
          <Link to="/" className="text-xl font-bold tracking-tight" data-testid="app-title">
            {t('app.title')}
          </Link>
          <nav className="flex items-center gap-4 text-sm">
            <Link
              to="/"
              className={linkClass(!isAdmin)}
              aria-current={!isAdmin ? 'page' : undefined}
            >
              {t('nav.userPanel')}
            </Link>
            <Link
              to="/admin"
              className={linkClass(isAdmin)}
              aria-current={isAdmin ? 'page' : undefined}
            >
              {t('nav.admin')}
            </Link>
            <div className="ml-2 flex rounded border border-panel-border overflow-hidden">
              {(['en', 'tr'] as const).map((l) => (
                <button
                  key={l}
                  onClick={() => setLang(l)}
                  className={
                    'px-2 py-1 uppercase text-xs ' +
                    (lang === l ? 'bg-panel-accent text-white' : 'hover:bg-panel-border')
                  }
                  aria-pressed={lang === l}
                >
                  {l}
                </button>
              ))}
            </div>
          </nav>
        </div>
      </header>
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-8">{children}</main>
      <footer className="border-t border-panel-border py-3 text-center text-xs text-slate-500">
        ServerPanel — open source Linux control panel
      </footer>
    </div>
  )
}

function linkClass(active: boolean): string {
  return active ? 'text-white font-medium' : 'text-slate-400 hover:text-slate-200'
}
