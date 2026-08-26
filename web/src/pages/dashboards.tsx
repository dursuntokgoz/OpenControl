import { HealthCard } from '../components/HealthCard'
import { useI18n } from '../i18n'

export function UserDashboard() {
  const { t } = useI18n()

  return (
    <>
      <h1 className="text-2xl font-semibold mb-6">{t('dash.welcomeUser')}</h1>
      <HealthCard />
      <p className="mt-6 text-sm text-slate-500">{t('dash.comingSoon')}</p>
    </>
  )
}

export function AdminDashboard() {
  const { t } = useI18n()

  return (
    <>
      <h1 className="text-2xl font-semibold mb-6">{t('dash.welcomeAdmin')}</h1>
      <HealthCard />
      <p className="mt-6 text-sm text-slate-500">{t('dash.comingSoon')}</p>
    </>
  )
}
