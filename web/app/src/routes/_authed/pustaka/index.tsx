import { createFileRoute } from '@tanstack/react-router'
import { BookMarked } from 'lucide-react'

import { useTranslation } from '@/lib/i18n'
import { BigCard } from '@/components/BigCard'

export const Route = createFileRoute('/_authed/pustaka/')({
  component: PustakaPage,
})

function PustakaPage() {
  const { t } = useTranslation()
  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold">{t('pustaka.hub.title')}</h1>
        <p className="mt-1 text-sm text-slate-500">{t('pustaka.hub.subtitle')}</p>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <BigCard
          to="/pustaka/kurikulum"
          icon={<BookMarked size={20} />}
          title={t('pustaka.hub.kurikulumTitle')}
          sub={t('pustaka.hub.kurikulumSub')}
          accent="bg-sky-50 text-sky-700"
        />
      </div>
    </div>
  )
}
