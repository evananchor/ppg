import { resolveApiPath } from './base'
import { ApiError, apiFetch } from './client'

export type BulkEntity = 'students' | 'teachers' | 'attendances' | 'pencapaian'

export type BulkMode = 'create' | 'upsert' | 'dry-run'

export type BulkSchema = {
  entity: string
  headers: string[]
}

export type BulkRowResult = {
  row: number
  outcome: 'created' | 'updated' | 'skipped' | 'failed'
  id?: string
  error?: string
}

export type BulkReport = {
  summary: {
    total: number
    created: number
    updated: number
    skipped: number
    failed: number
  }
  results: BulkRowResult[]
}

export function getBulkSchema(entity: BulkEntity): Promise<BulkSchema> {
  return apiFetch<BulkSchema>(`/api/${entity}/bulk/schema`)
}

export async function importBulk(
  entity: BulkEntity,
  file: File,
  mode: BulkMode,
): Promise<BulkReport> {
  const form = new FormData()
  form.append('file', file)
  form.append('mode', mode)

  // apiFetch always JSON-encodes its body, so multipart uploads go through
  // fetch directly (the browser sets the multipart boundary header itself).
  const res = await fetch(resolveApiPath(`/api/${entity}/bulk`), {
    method: 'POST',
    credentials: 'include',
    headers: { Accept: 'application/json' },
    body: form,
  })

  let data: unknown = null
  const text = await res.text()
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      // non-JSON response; leave as null
    }
  }

  if (!res.ok) {
    const errBody = (data as { error?: { code?: string; message?: string } } | null)?.error
    throw new ApiError(res.status, errBody?.code ?? 'unknown', errBody?.message ?? res.statusText)
  }

  return data as BulkReport
}

export async function downloadExport(
  entity: BulkEntity,
  params: Record<string, string | undefined> = {},
): Promise<void> {
  const qs = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value) qs.set(key, value)
  }
  const query = qs.toString()
  const path = `/api/${entity}/export.csv${query ? `?${query}` : ''}`

  const res = await fetch(resolveApiPath(path), { credentials: 'include' })
  if (!res.ok) {
    throw new ApiError(res.status, 'unknown', res.statusText)
  }

  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${entity}-${new Date().toISOString().slice(0, 10)}.csv`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
