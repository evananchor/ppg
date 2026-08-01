import { apiFetch } from './client'
import type { MateriAjar } from './kurikulum'

export type PencapaianStatus = 'belum' | 'proses' | 'tuntas'

export type Pencapaian = {
  id: string
  studentId: string
  materiAjarId: string
  status: PencapaianStatus
  nilaiAngka?: number | null
  tanggal?: string | null
  catatan?: string | null
  recordedBy?: string | null
  updatedAt: string
  materi: MateriAjar
  tingkatNama: string
  tingkatUrutan: number
}

export type PencapaianInput = {
  studentId: string
  materiAjarId: string
  status: PencapaianStatus
  nilaiAngka?: number | null
  tanggal?: string | null
  catatan?: string | null
}

export type ReportCell = {
  month: number
  count: number
  avg?: number | null
}

export type ReportRow = {
  materi: MateriAjar
  cells: ReportCell[]
}

export function listPencapaian(studentId: string) {
  return apiFetch<Pencapaian[]>(`/api/pencapaian?studentId=${encodeURIComponent(studentId)}`)
}

export function upsertPencapaian(input: PencapaianInput) {
  return apiFetch<Pencapaian>('/api/pencapaian', { method: 'POST', body: input })
}

export function deletePencapaian(id: string) {
  return apiFetch<void>(`/api/pencapaian/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export function getClassReport(tingkatId: string, semester: number) {
  return apiFetch<ReportRow[]>(
    `/api/achievement/report?tingkatId=${encodeURIComponent(tingkatId)}&semester=${semester}`,
  )
}

export const NILAI_GRADES = [
  { min: 90, label: 'baikSekali' },
  { min: 80, label: 'baik' },
  { min: 70, label: 'cukup' },
  { min: 60, label: 'kurang' },
] as const

export function gradeLabel(nilai: number): string {
  for (const g of NILAI_GRADES) {
    if (nilai >= g.min) return g.label
  }
  return 'kurang'
}
