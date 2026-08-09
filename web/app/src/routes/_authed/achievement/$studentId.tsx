import { useMemo, useRef, useState } from 'react'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, FileSpreadsheet, Search, Trash2, Upload } from 'lucide-react'
import { z } from 'zod'

import { useMe } from '@/lib/auth'
import { useTranslation } from '@/lib/i18n'
import { cn } from '@/lib/cn'
import { ReportView } from '@/components/BulkPanel'
import type { BulkReport } from '@/api/bulk'
import { listMateriAjar, listTingkats, type MateriAjar } from '@/api/kurikulum'
import {
  deletePencapaian,
  importPencapaianMatrix,
  listPencapaian,
  upsertPencapaian,
  type PencapaianStatus,
} from '@/api/pencapaian'
import { getStudent, type Student } from '@/api/students'
import { ageInYears } from '@/lib/age'
import { groupByTemaSub, currentSemester, SEMESTER_MONTHS, sortedTemas, tingkatForAge } from '@/lib/tingkat'
import { Avatar } from '@/components/Avatar'
import { Button } from '@/components/Button'
import { Field } from '@/components/Field'
import { Input } from '@/components/Input'
import { MateriTree } from '@/components/MateriTree'

const searchSchema = z.object({
  tab: z.enum(['penilaian', 'laporan']).optional().catch('penilaian'),
})

export const Route = createFileRoute('/_authed/achievement/$studentId')({
  validateSearch: searchSchema,
  component: StudentAchievementPage,
})

const selectCls = () => {
  return 'flex h-10 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-not-allowed disabled:opacity-50'
}

function StudentAchievementPage() {
  const { studentId } = Route.useParams()
  const search = Route.useSearch()
  const navigate = useNavigate({ from: '/achievement/$studentId' })
  const tab = search.tab ?? 'penilaian'
  const { t } = useTranslation()
  const { data: user } = useMe()
  const isAdmin = user?.role === 'admin'
  const { data: student, isPending } = useQuery({
    queryKey: ['student', studentId],
    queryFn: () => getStudent(studentId),
  })
  const { data: tingkats = [] } = useQuery({ queryKey: ['tingkats'], queryFn: listTingkats })

  const [tingkatId, setTingkatId] = useState('')
  const [tema, setTema] = useState('')
  const [semester, setSemester] = useState<number | null>(1)
  const [q, setQ] = useState('')

  const age = ageInYears(student?.dateOfBirth)
  const autoTingkat = tingkatForAge(age, tingkats)
  const currentTingkat = tingkats.find((tk) => tk.id === tingkatId) ?? autoTingkat

  const { data: allMateri = [] } = useQuery({
    queryKey: ['materi-ajar', currentTingkat?.id],
    queryFn: () => listMateriAjar({ tingkatId: currentTingkat!.id }),
    enabled: !!currentTingkat,
  })
  const { data: existing = [] } = useQuery({
    queryKey: ['pencapaian', studentId],
    queryFn: () => listPencapaian(studentId),
    enabled: !!student,
  })

  const temas = useMemo(() => sortedTemas(allMateri), [allMateri])
  const filtered = useMemo(() => {
    const needle = q.trim().toLowerCase()
    return allMateri.filter(
      (m) =>
        (semester == null || m.semester === semester) &&
        (tema === '' || m.tema === tema) &&
        (needle === '' ||
          m.materi.toLowerCase().includes(needle) ||
          (m.tema ?? '').toLowerCase().includes(needle) ||
          (m.subTema ?? '').toLowerCase().includes(needle)),
    )
  }, [allMateri, semester, tema, q])

  const byMateri = useMemo(
    () => new Map(existing.map((p) => [p.materiAjarId, p])),
    [existing],
  )

  const sem = semester ?? currentSemester()
  const exportRows = useMemo(() => filtered.filter((m) => m.semester === sem), [filtered, sem])
  const months = SEMESTER_MONTHS[sem]

  const monthValue = (m: MateriAjar, month: number): number | null => {
    const p = byMateri.get(m.id)
    if (!p?.tanggal || p.nilaiAngka == null) return null
    return new Date(p.tanggal).getMonth() + 1 === month ? p.nilaiAngka : null
  }

  const downloadMatrixCsv = (withStatus: boolean) => {
    const head = withStatus
      ? ['No', 'Semester', 'Tema', 'Sub Tema', 'Rincian', 'Materi', 'Status', ...months.map((mo) => t(`achievement.month.${mo}`))]
      : ['No', 'Semester', 'Tema', 'Sub Tema', 'Rincian', 'Materi', ...months.map((mo) => t(`achievement.month.${mo}`))]
    const lines = groupByTemaSub(exportRows)
      .flatMap((g) => g.subTemas.flatMap((s) => s.materi))
      .map((m) => {
      const p = byMateri.get(m.id)
      const cells = [m.nomor, sem, m.tema, m.subTema ?? '', m.rincian ?? '', m.materi]
      if (withStatus) cells.push(p?.status ?? '')
      cells.push(...months.map((mo) => monthValue(m, mo) ?? ''))
      return cells.map((v) => `"${String(v).replaceAll('"', '""')}"`).join(',')
    })
    const blob = new Blob(['\uFEFF' + [head.join(','), ...lines].join('\n')], { type: 'text/csv;charset=utf-8' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `pencapaian-${(student?.name ?? 'generus').replaceAll(' ', '-')}-smt${sem}.csv`
    a.click()
    URL.revokeObjectURL(a.href)
  }

  const qc = useQueryClient()
  const fileRef = useRef<HTMLInputElement>(null)
  const [importReport, setImportReport] = useState<BulkReport | null>(null)
  const importMut = useMutation({
    mutationFn: (file: File) =>
      importPencapaianMatrix(file, { studentId, tingkatId: currentTingkat!.id, semester: sem }),
    onSuccess: (r) => {
      setImportReport(r)
      qc.invalidateQueries({ queryKey: ['pencapaian', studentId] })
    },
  })

  if (isPending) {
    return <p className="py-8 text-center text-sm text-slate-500">{t('common.loading')}</p>
  }
  if (!student) {
    return <p className="py-8 text-center text-sm text-slate-500">{t('common.noDataLong')}</p>
  }

  const goTab = (next: 'penilaian' | 'laporan') => void navigate({ search: { tab: next } })

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Link
            to="/achievement"
            className="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-900"
            aria-label={t('achievement.studentTabs.back')}
            title={t('achievement.studentTabs.back')}
          >
            <ArrowLeft size={18} />
          </Link>
          <Avatar name={student.name} className="h-12 w-12 text-base" />
          <div>
            <p className="text-base font-semibold text-slate-900">{student.name}</p>
            <p className="text-xs text-slate-500">
              {age != null ? t('achievement.grid.age', { age }) : t('common.notFilled')}
              {autoTingkat ? ` · ${autoTingkat.nama}` : ''}
              <span className="ml-1 inline-block rounded bg-slate-100 px-1.5 py-0.5">{student.kelompok}</span>
            </p>
          </div>
        </div>
      </div>

      <div className="flex gap-1 border-b border-slate-200">
        {(['penilaian', 'laporan'] as const).map((key) => (
          <button
            key={key}
            type="button"
            onClick={() => goTab(key)}
            className={cn(
              'rounded-t-md border-b-2 px-4 py-2 text-sm font-medium',
              tab === key
                ? 'border-slate-900 text-slate-900'
                : 'border-transparent text-slate-500 hover:text-slate-800',
            )}
          >
            {t(`achievement.studentTabs.${key}`)}
          </button>
        ))}
      </div>

      <FilterCard
        tingkats={tingkats}
        current={currentTingkat}
        onTingkat={(id) => {
          setTingkatId(id)
          setTema('')
        }}
        temas={temas}
        tema={tema}
        setTema={setTema}
        semester={semester}
        setSemester={setSemester}
        q={q}
        setQ={setQ}
        tab={tab}
        canExport={exportRows.length > 0 && !!currentTingkat}
        canImport={isAdmin && tab === 'penilaian' && !!currentTingkat && !importMut.isPending}
        importing={importMut.isPending}
        onExport={() => downloadMatrixCsv(tab === 'penilaian')}
        onImport={() => fileRef.current?.click()}
      />

      <input
        ref={fileRef}
        type="file"
        accept=".csv,text/csv"
        className="hidden"
        onChange={(e) => {
          const f = e.target.files?.[0]
          if (f) {
            setImportReport(null)
            importMut.mutate(f)
          }
          e.target.value = ''
        }}
      />
      {importReport ? <ReportView report={importReport} /> : null}
      {importMut.error ? <p className="text-sm text-red-600">{importMut.error.message}</p> : null}

      {tab === 'penilaian' ? (
        <PenilaianTab student={student} filtered={filtered} byMateri={byMateri} />
      ) : (
        <LaporanTab
          filtered={filtered}
          byMateri={byMateri}
          semester={semester ?? 1}
        />
      )}
    </div>
  )
}

function FilterCard({
  tingkats,
  current,
  onTingkat,
  temas,
  tema,
  setTema,
  semester,
  setSemester,
  q,
  setQ,
  tab,
  canExport,
  canImport,
  importing,
  onExport,
  onImport,
}: {
  tingkats: { id: string; nama: string; urutan: number; umur?: number | null }[]
  current?: { id: string; nama: string }
  onTingkat: (id: string) => void
  temas: string[]
  tema: string
  setTema: (v: string) => void
  semester: number | null
  setSemester: (v: number | null) => void
  q: string
  setQ: (v: string) => void
  tab: 'penilaian' | 'laporan'
  canExport: boolean
  canImport: boolean
  importing: boolean
  onExport: () => void
  onImport: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap items-end gap-3 rounded-lg border border-slate-200 bg-white p-4">
      <Field label={t('achievement.tingkat')}>
        <select
          className={selectCls()}
          value={current?.id ?? ''}
          onChange={(e) => onTingkat(e.target.value)}
        >
          {tingkats.map((tk) => (
            <option key={tk.id} value={tk.id}>
              {tk.nama}
            </option>
          ))}
        </select>
      </Field>
      <Field label={t('achievement.tema')}>
        <select className={selectCls()} value={tema} onChange={(e) => setTema(e.target.value)}>
          <option value="">{t('common.all')}</option>
          {temas.map((tm) => (
            <option key={tm} value={tm}>
              {tm}
            </option>
          ))}
        </select>
      </Field>
      <Field label={t('achievement.semester')}>
        <div className="flex h-10 overflow-hidden rounded-md border border-slate-300">
          <button
            type="button"
            onClick={() => setSemester(semester === 1 ? null : 1)}
            className={cn(
              'px-3 text-sm font-medium',
              semester === 1 ? 'bg-slate-900 text-white' : 'bg-white text-slate-700 hover:bg-slate-100',
            )}
          >
            {t('achievement.semesterName', { n: 1 })}
          </button>
          <button
            type="button"
            onClick={() => setSemester(semester === 2 ? null : 2)}
            className={cn(
              'border-l border-slate-300 px-3 text-sm font-medium',
              semester === 2 ? 'bg-slate-900 text-white' : 'bg-white text-slate-700 hover:bg-slate-100',
            )}
          >
            {t('achievement.semesterName', { n: 2 })}
          </button>
        </div>
      </Field>
      <Field label={t('achievement.bulkTitle')}>
        <div className="flex h-10 gap-2">
          <Button variant="secondary" disabled={!canExport} onClick={onExport}>
            <FileSpreadsheet size={16} className="mr-1" /> {t('achievement.export')}
          </Button>
          {tab === 'penilaian' && (
            <Button variant="secondary" disabled={!canImport} onClick={onImport}>
              <Upload size={16} className="mr-1" />
              {importing ? t('common.loading') : t('achievement.import')}
            </Button>
          )}
        </div>
      </Field>
      <Field label={t('achievement.searchLabel')} className="min-w-48 flex-1">
        <div className="relative">
          <Search size={16} className="absolute left-3 top-3 text-slate-400" />
          <Input
            className="pl-9"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t('achievement.searchPlaceholder')}
          />
        </div>
      </Field>
    </div>
  )
}

// ---------- Penilaian ----------

const STATUSES: PencapaianStatus[] = ['belum', 'proses', 'tuntas']

function PenilaianTab({
  student,
  filtered,
  byMateri,
}: {
  student: Student
  filtered: MateriAjar[]
  byMateri: Map<string, { id: string; status: PencapaianStatus; nilaiAngka?: number | null; tanggal?: string | null }>
}) {
  const { t } = useTranslation()
  const { data: user } = useMe()
  const manage = !!user
  const qc = useQueryClient()
  const groups = useMemo(() => groupByTemaSub(filtered), [filtered])

  const stats = useMemo(() => {
    const totals = new Map<string, number>()
    const tuntas = new Map<string, number>()
    const proses = new Map<string, number>()
    for (const m of filtered) {
      totals.set(m.tema ?? '', (totals.get(m.tema ?? '') ?? 0) + 1)
      const p = byMateri.get(m.id)
      if (p?.status === 'tuntas') tuntas.set(m.tema ?? '', (tuntas.get(m.tema ?? '') ?? 0) + 1)
      if (p?.status === 'proses') proses.set(m.tema ?? '', (proses.get(m.tema ?? '') ?? 0) + 1)
    }
    return { totals, tuntas, proses }
  }, [filtered, byMateri])

  const save = useMutation({
    mutationFn: (v: { materiAjarId: string; status: PencapaianStatus; nilaiAngka?: number | null; tanggal?: string | null }) =>
      upsertPencapaian({ studentId: student.id, ...v }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['pencapaian', student.id] }),
  })
  const del = useMutation({
    mutationFn: (id: string) => deletePencapaian(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['pencapaian', student.id] }),
  })

  const totalDone = Array.from(stats.tuntas.values()).reduce((a, b) => a + b, 0)
  const totalAll = filtered.length

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-slate-200 bg-white p-4">
        <div className="flex items-center justify-between text-sm">
          <span className="font-medium text-slate-900">{t('achievement.progressLabel')}</span>
          <span className="text-slate-500">
            {totalDone}/{totalAll} · {totalAll > 0 ? Math.round((totalDone / totalAll) * 100) : 0}%
          </span>
        </div>
        <div className="mt-2 h-2 overflow-hidden rounded-full bg-slate-100">
          <div
            className="h-full rounded-full bg-emerald-500 transition-all"
            style={{ width: totalAll > 0 ? `${(totalDone / totalAll) * 100}%` : '0%' }}
          />
        </div>
      </div>

      {filtered.length === 0 ? (
        <p className="py-8 text-center text-sm text-slate-500">{t('common.noDataLong')}</p>
      ) : (
        <MateriTree
          groups={groups}
          temaExtra={(temaName, _materi) => {
            const tot = stats.totals.get(temaName) ?? 0
            const done = stats.tuntas.get(temaName) ?? 0
            const proc = stats.proses.get(temaName) ?? 0
            return (
              <span className="ml-auto flex items-center gap-2">
                <span className="h-1.5 w-20 overflow-hidden rounded-full bg-slate-200">
                  <span className="block h-full bg-emerald-500" style={{ width: tot > 0 ? `${(done / tot) * 100}%` : '0%' }} />
                </span>
                <span className="text-xs text-slate-400">
                  {done + proc}/{tot}
                </span>
              </span>
            )
          }}
          renderRow={(m) => {
            const p = byMateri.get(m.id)
            return (
              <div className="flex flex-col gap-2 px-4 py-2.5 md:flex-row md:items-center md:gap-3">
                <div className="min-w-0 md:flex-1">
                  <p className="text-sm text-slate-900">
                    {m.nomor}. {m.materi}
                  </p>
                  <p className="text-xs text-slate-400">
                    {m.tema}
                    {m.subTema ? ` · ${m.subTema}` : ''}
                  </p>
                  {m.rincian ? <p className="text-xs text-slate-400">{m.rincian}</p> : null}
                </div>
                <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(8.5rem,1fr)_auto] items-center gap-1.5 md:gap-2">
                  <select
                    className={selectCls()}
                    value={p?.status ?? 'belum'}
                    disabled={!manage}
                    onChange={(e) => save.mutate({ materiAjarId: m.id, status: e.target.value as PencapaianStatus, nilaiAngka: p?.nilaiAngka ?? null, tanggal: p?.tanggal ?? null })}
                  >
                    {STATUSES.map((s) => (
                      <option key={s} value={s}>
                        {t(`achievement.status.${s}`)}
                      </option>
                    ))}
                  </select>
                  <Input
                    type="number"
                    min={0}
                    max={100}
                    disabled={!manage}
                    className="w-full"
                    defaultValue={p?.nilaiAngka != null ? String(p.nilaiAngka) : ''}
                    key={`${m.id}-nilai`}
                    onBlur={(e) => {
                      const next = e.target.value === '' ? null : Number(e.target.value)
                      if (next === (p?.nilaiAngka ?? null)) return
                      save.mutate({
                        materiAjarId: m.id,
                        status: p?.status ?? 'belum',
                        nilaiAngka: next,
                        tanggal: p?.tanggal ?? null,
                      })
                    }}
                  />
                  <Input
                    type="date"
                    disabled={!manage}
                    className="w-full"
                    defaultValue={p?.tanggal ?? ''}
                    key={`${m.id}-tanggal`}
                    onBlur={(e) => {
                      const next = e.target.value || null
                      if (next === (p?.tanggal ?? null)) return
                      save.mutate({
                        materiAjarId: m.id,
                        status: p?.status ?? 'belum',
                        nilaiAngka: p?.nilaiAngka ?? null,
                        tanggal: next,
                      })
                    }}
                  />
                  {manage && p && (
                    <button
                      type="button"
                      onClick={() => del.mutate(p.id)}
                      className="rounded-md p-2 text-slate-500 transition hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-300"
                      aria-label={t('common.delete')}
                      title={t('common.delete')}
                    >
                      <Trash2 size={16} />
                    </button>
                  )}
                </div>
              </div>
            )
          }}
        />
      )}
    </div>
  )
}

// ---------- Laporan ----------

function LaporanTab({
  filtered,
  byMateri,
  semester,
}: {
  filtered: MateriAjar[]
  byMateri: Map<string, { id: string; status?: PencapaianStatus; nilaiAngka?: number | null; tanggal?: string | null }>
  semester: number
}) {
  const { t } = useTranslation()
  const sem = semester ?? currentSemester()
  const rows = filtered.filter((m) => m.semester === sem)
  const months = SEMESTER_MONTHS[sem]
  const groups = useMemo(() => groupByTemaSub(rows), [rows])

  const completion = useMemo(() => {
    const byTema = new Map<string, { total: number; done: number }>()
    const bySub = new Map<string, { total: number; done: number }>()
    for (const m of rows) {
      const done = byMateri.get(m.id)?.status === 'tuntas' ? 1 : 0
      const tema = byTema.get(m.tema ?? '') ?? { total: 0, done: 0 }
      byTema.set(m.tema ?? '', { total: tema.total + 1, done: tema.done + done })
      const key = `${m.tema ?? ''}/${m.subTema}`
      const sub = bySub.get(key) ?? { total: 0, done: 0 }
      bySub.set(key, { total: sub.total + 1, done: sub.done + done })
    }
    return { byTema, bySub }
  }, [rows, byMateri])

  const pct = (done: number, total: number) => (total > 0 ? Math.round((done / total) * 100) : 0)

  const valueFor = (m: MateriAjar, month: number): number | null => {
    const p = byMateri.get(m.id)
    if (!p?.tanggal || p.nilaiAngka == null) return null
    const mm = new Date(p.tanggal).getMonth() + 1
    return mm === month ? p.nilaiAngka : null
  }

  const cellCls = (v: number | null) =>
    cn(
      'flex h-7 items-center justify-center rounded text-xs font-medium',
      v != null
        ? v >= 90
          ? 'bg-green-50 text-green-700'
          : v >= 80
            ? 'bg-lime-50 text-lime-700'
            : v >= 70
              ? 'bg-amber-50 text-amber-700'
              : 'bg-red-50 text-red-700'
        : 'text-slate-300',
    )

  return (
    <div className="space-y-4">
      {rows.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-slate-900">{t('achievement.completionLabel')}</h3>
          <div className="mt-2 grid gap-3 md:grid-cols-2">
            {groups.map((g) => {
              const temaStat = completion.byTema.get(g.tema) ?? { total: 0, done: 0 }
              return (
                <div key={g.tema} className="rounded-lg border border-slate-200 bg-white p-3">
                  <div className="flex items-center gap-2">
                    <span className="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-semibold uppercase text-slate-700">
                      {g.tema}
                    </span>
                    <span className="text-xs text-slate-400">
                      {temaStat.done}/{temaStat.total} · {pct(temaStat.done, temaStat.total)}%
                    </span>
                  </div>
                  <div className="mt-2 h-2 overflow-hidden rounded-full bg-slate-100">
                    <div
                      className="h-full rounded-full bg-emerald-500 transition-all"
                      style={{ width: `${pct(temaStat.done, temaStat.total)}%` }}
                    />
                  </div>
                  <div className="mt-2 space-y-1.5">
                    {g.subTemas.map((sub) => {
                      const key = `${g.tema}/${sub.subTema}`
                      const subStat = completion.bySub.get(key) ?? { total: 0, done: 0 }
                      return (
                        <div key={key} className="flex items-center gap-2 text-xs">
                          <span className="w-36 truncate text-slate-500">{sub.subTema}</span>
                          <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-slate-100">
                            <div
                              className="h-full rounded-full bg-emerald-500"
                              style={{ width: `${pct(subStat.done, subStat.total)}%` }}
                            />
                          </div>
                          <span className="w-14 text-right text-slate-400">
                            {subStat.done}/{subStat.total}
                          </span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      <div className="overflow-x-auto rounded-md border border-slate-200 bg-white">
        <div className="grid gap-2 border-b border-slate-200 px-4 py-2 text-xs font-semibold uppercase text-slate-500" style={GRID_TEMPLATE}>
          <span>{t('achievement.no')}</span>
          <span>{t('achievement.materi')}</span>
          {months.map((m) => (
            <span key={m} className="text-center">
              {t(`achievement.month.${m}`)}
            </span>
          ))}
        </div>
        {rows.length === 0 ? (
          <p className="py-8 text-center text-sm text-slate-500">{t('common.noDataLong')}</p>
        ) : (
          <MateriTree
            groups={groups}
            renderRow={(m) => (
              <div className="grid gap-2 px-4 py-2" style={GRID_TEMPLATE}>
                <span className="text-sm text-slate-500">{m.nomor}</span>
                <div className="min-w-0">
                  <span className="block truncate text-sm text-slate-900" title={m.materi}>
                    {m.materi}
                  </span>
                  {m.rincian ? (
                    <span className="block truncate text-xs text-slate-400" title={m.rincian}>
                      {m.rincian}
                    </span>
                  ) : null}
                </div>
                {months.map((mo) => (
                  <span key={mo} className={cellCls(valueFor(m, mo))}>
                    {valueFor(m, mo) ?? '·'}
                  </span>
                ))}
              </div>
            )}
          />
        )}
      </div>

      <div className="flex flex-wrap gap-3 text-xs text-slate-600">
        <span className="rounded bg-green-50 px-2 py-1 text-green-700">90-100 · {t('achievement.grade.baikSekali')}</span>
        <span className="rounded bg-lime-50 px-2 py-1 text-lime-700">80-89 · {t('achievement.grade.baik')}</span>
        <span className="rounded bg-amber-50 px-2 py-1 text-amber-700">70-79 · {t('achievement.grade.cukup')}</span>
        <span className="rounded bg-red-50 px-2 py-1 text-red-700">60-69 · {t('achievement.grade.kurang')}</span>
        <span className="rounded bg-slate-100 px-2 py-1 text-slate-500">· {t('achievement.grade.empty')}</span>
      </div>
    </div>
  )
}

const GRID_TEMPLATE = {
  gridTemplateColumns: '2.5rem minmax(0, 1fr) repeat(6, 3.5rem)',
} as const
