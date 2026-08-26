import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useI18n } from '../i18n'

export function HealthCard() {
  const { t } = useI18n()
  const { data, isPending, isError } = useQuery({
    queryKey: ['health'],
    queryFn: api.health,
    refetchInterval: 30_000,
  })

  return (
    <div className="rounded-lg border border-panel-border bg-panel-surface p-6 shadow">
      <h2 className="text-lg font-semibold mb-4">{t('health.title')}</h2>
      {isPending && <p className="text-slate-400">{t('health.loading')}</p>}
      {isError && (
        <p className="flex items-center gap-2 text-red-400">
          <span className="inline-block size-2.5 rounded-full bg-red-500" />
          {t('health.offline')}
        </p>
      )}
      {data && (
        <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
          <dt className="text-slate-400">{t('health.status')}</dt>
          <dd className="flex items-center gap-2">
            <span className="inline-block size-2.5 rounded-full bg-emerald-500" />
            <span data-testid="health-status">{data.status}</span>
          </dd>
          <dt className="text-slate-400">{t('health.version')}</dt>
          <dd data-testid="health-version">{data.version}</dd>
        </dl>
      )}
    </div>
  )
}
