import { useState, type ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'

import type { MateriAjar } from '@/api/kurikulum'
import type { MateriGroup } from '@/lib/tingkat'
import { TEMA_BG, TEMA_COLOR } from '@/lib/tingkat'

type Props = {
  groups: MateriGroup[]
  renderRow: (m: MateriAjar) => ReactNode
  temaExtra?: (tema: string, materi: MateriAjar[]) => ReactNode
  subExtra?: (subTema: string, materi: MateriAjar[]) => ReactNode
  defaultOpen?: boolean
}

export function MateriTree({ groups, renderRow, temaExtra, subExtra, defaultOpen = true }: Props) {
  const [openTemas, setOpenTemas] = useState<Set<string>>(() =>
    defaultOpen ? new Set(groups.map((g) => g.tema)) : new Set(),
  )
  const [openSubs, setOpenSubs] = useState<Set<string>>(
    () => new Set(groups.flatMap((g) => g.subTemas.map((s) => `${g.tema}/${s.subTema}`))),
  )

  const toggle = (set: Set<string>, key: string) => {
    const next = new Set(set)
    if (next.has(key)) next.delete(key)
    else next.add(key)
    return next
  }

  return (
    <div className="space-y-3">
      {groups.map((g) => {
        const allMateri = g.subTemas.flatMap((s) => s.materi)
        const openTema = openTemas.has(g.tema)
        return (
          <div key={g.tema} className="overflow-hidden rounded-lg border border-slate-200 bg-white">
            <button
              type="button"
              onClick={() => setOpenTemas((s) => toggle(s, g.tema))}
              className="flex w-full items-center gap-2 border-l-4 bg-slate-50 px-3 py-2.5 text-left"
              style={{ borderLeftColor: TEMA_COLOR[g.tema] ?? '#94a3b8' }}
            >
              {openTema ? <ChevronDown size={16} className="text-slate-400" /> : <ChevronRight size={16} className="text-slate-400" />}
              <span
                className={`rounded px-1.5 py-0.5 text-xs font-semibold uppercase ${
                  TEMA_BG[g.tema] ?? 'bg-slate-100 text-slate-700'
                }`}
              >
                {g.tema}
              </span>
              <span className="text-xs text-slate-500">{allMateri.length} materi</span>
              {temaExtra ? temaExtra(g.tema, allMateri) : null}
            </button>
            {openTema && (
              <div className="divide-y divide-slate-100">
                {g.subTemas.map((sub) => {
                  const key = `${g.tema}/${sub.subTema}`
                  const openSub = openSubs.has(key)
                  return (
                    <div key={key}>
                      <button
                        type="button"
                        onClick={() => setOpenSubs((s) => toggle(s, key))}
                        className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm font-medium text-slate-700 hover:bg-slate-50"
                      >
                        {openSub ? <ChevronDown size={14} className="text-slate-400" /> : <ChevronRight size={14} className="text-slate-400" />}
                        <span className="text-xs font-semibold uppercase text-slate-500">{sub.subTema}</span>
                        <span className="text-xs text-slate-400">{sub.materi.length}</span>
                        {subExtra ? <span className="ml-auto">{subExtra(sub.subTema, sub.materi)}</span> : null}
                      </button>
                      {openSub && (
                        <div className="divide-y divide-slate-50">{sub.materi.map((m) => renderRow(m))}</div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
