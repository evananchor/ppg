import type { MateriAjar, Tingkat } from '@/api/kurikulum'

export const TEMA_ORDER = ['ALIM', 'FAQIH', 'AKHLAQUL KARIMAH', 'KEMANDIRIAN'] as const

export const TEMA_COLOR: Record<string, string> = {
  ALIM: '#0f766e',
  FAQIH: '#b45309',
  'AKHLAQUL KARIMAH': '#be185d',
  KEMANDIRIAN: '#1d4ed8',
}

export const TEMA_BG: Record<string, string> = {
  ALIM: 'bg-teal-50 text-teal-800',
  FAQIH: 'bg-amber-50 text-amber-800',
  'AKHLAQUL KARIMAH': 'bg-pink-50 text-pink-800',
  KEMANDIRIAN: 'bg-blue-50 text-blue-800',
}

export function sortedTemas(materi: MateriAjar[]): string[] {
  const set = new Set<string>(materi.map((m) => m.tema ?? '(tanpa tema)'))
  const rest = Array.from(set).filter((x) => !TEMA_ORDER.includes(x as never)).sort()
  return [...TEMA_ORDER.filter((x) => set.has(x)), ...rest]
}

export type MateriGroup = {
  tema: string
  subTemas: { subTema: string; materi: MateriAjar[] }[]
}

export function groupByTemaSub(materi: MateriAjar[]): MateriGroup[] {
  const map = new Map<string, Map<string, MateriAjar[]>>()
  for (const m of materi) {
    const tema = m.tema || '(tanpa tema)'
    if (!map.has(tema)) map.set(tema, new Map())
    const sub = m.subTema || '(tanpa sub tema)'
    const subs = map.get(tema)!
    if (!subs.has(sub)) subs.set(sub, [])
    subs.get(sub)!.push(m)
  }
  return sortedTemas(materi).map((tema) => {
    const subs = map.get(tema) ?? new Map<string, MateriAjar[]>()
    return {
      tema,
      subTemas: Array.from(subs.entries()).map(([subTema, items]) => ({
        subTema,
        materi: items.sort((a, b) => a.nomor - b.nomor),
      })),
    }
  })
}

export function tingkatForAge(age: number | null, tingkats: Tingkat[]): Tingkat | undefined {
  if (age == null) return undefined
  const sorted = [...tingkats].sort((a, b) => a.urutan - b.urutan)
  let last: Tingkat | undefined
  for (const tk of sorted) {
    if (tk.umur != null && age <= tk.umur) return tk
    if (tk.umur != null) last = tk
  }
  return last
}

export function currentSemester(): number {
  return new Date().getMonth() >= 6 ? 1 : 2
}

export const SEMESTER_MONTHS: Record<number, number[]> = {
  1: [7, 8, 9, 10, 11, 12],
  2: [1, 2, 3, 4, 5, 6],
}
