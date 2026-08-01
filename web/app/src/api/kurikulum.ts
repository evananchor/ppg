import { apiFetch } from './client'

export type Tingkat = {
  id: string
  nama: string
  urutan: number
  umur?: number | null
}

export type TingkatInput = {
  nama: string
  urutan: number
  umur?: number | null
}

export type MateriAjar = {
  id: string
  tingkatId: string
  nomor: number
  sumberNo: number | null
  tema: string | null
  subTema: string | null
  rincian: string | null
  materi: string
  cakupan: string | null
  jenis: string | null
  targetSemester: string | null
  deskripsi: string | null
  statusPromes: string | null
  keterangan: string | null
  semester: number | null
}

export type MateriAjarInput = {
  tingkatId: string
  nomor: number
  sumberNo?: number | null
  tema?: string | null
  subTema?: string | null
  rincian?: string | null
  materi: string
  cakupan?: string | null
  jenis?: string | null
  targetSemester?: string | null
  deskripsi?: string | null
  statusPromes?: string | null
  keterangan?: string | null
  semester?: number | null
}

export function listTingkats() {
  return apiFetch<Tingkat[]>('/api/tingkat')
}

export function createTingkat(input: TingkatInput) {
  return apiFetch<Tingkat>('/api/tingkat', { method: 'POST', body: input })
}

export function updateTingkat(id: string, input: TingkatInput) {
  return apiFetch<Tingkat>(`/api/tingkat/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: input,
  })
}

export function deleteTingkat(id: string) {
  return apiFetch<void>(`/api/tingkat/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export type MateriListParams = {
  tingkatId?: string
  semester?: number
  tema?: string
  q?: string
}

export function listMateriAjar(params: MateriListParams = {}) {
  const q = new URLSearchParams()
  if (params.tingkatId) q.set('tingkatId', params.tingkatId)
  if (params.semester !== undefined) q.set('semester', String(params.semester))
  if (params.tema) q.set('tema', params.tema)
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiFetch<MateriAjar[]>(`/api/materi-ajar${qs ? `?${qs}` : ''}`)
}

export function createMateriAjar(input: MateriAjarInput) {
  return apiFetch<MateriAjar>('/api/materi-ajar', { method: 'POST', body: input })
}

export function updateMateriAjar(id: string, input: MateriAjarInput) {
  return apiFetch<MateriAjar>(`/api/materi-ajar/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: input,
  })
}

export function deleteMateriAjar(id: string) {
  return apiFetch<void>(`/api/materi-ajar/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
