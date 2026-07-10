import type { Teacher } from '@/api/types'
import { useTeacherStatusLabel, useTranslation } from '@/lib/i18n'

export function TeacherDetail({ teacher: tc }: { teacher: Teacher }) {
  const { t } = useTranslation()
  const statusLabel = useTeacherStatusLabel()
  return (
    <dl className="grid gap-4 text-sm sm:grid-cols-2">
      <Row label={t('teachers.fName')} value={tc.name} />
      <Row label={t('teachers.fNickname')} value={tc.nickname ?? '—'} />
      <Row label={t('teachers.fKelompok')} value={tc.kelompok} />
      <Row label={t('teachers.fDesa')} value={tc.desa} />
      <Row label={t('teachers.fDaerah')} value={tc.daerah} className="sm:col-span-2" />
      <Row label={t('teachers.fJoinedAt')} value={tc.joinedAt?.slice(0, 10) ?? '—'} />
      <Row label={t('teachers.fRetiredAt')} value={tc.retiredAt?.slice(0, 10) ?? '—'} />
      <Row label={t('teachers.fStatus')} value={statusLabel(tc.status)} />
      <Row label={t('teachers.fNotes')} value={tc.notes ?? '—'} className="sm:col-span-2" />
    </dl>
  )
}

function Row({ label, value, className }: { label: string; value: string; className?: string }) {
  return (
    <div className={className}>
      <dt className="text-xs uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="mt-1 break-words text-slate-900">{value}</dd>
    </div>
  )
}
