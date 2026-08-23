import type { Card, GraphPayload, PageMeta, Tag } from './types'

interface Envelope<T> {
  data?: T
  error?: { code: string; message: string }
  meta?: PageMeta
}

async function req<T>(path: string, init?: RequestInit): Promise<{ data: T; meta?: PageMeta }> {
  const res = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
  })
  const json = (await res.json()) as Envelope<T>
  if (!res.ok || json.error) {
    const err = new Error(json.error?.message || `HTTP ${res.status}`)
    ;(err as Error & { code?: string }).code = json.error?.code
    throw err
  }
  return { data: json.data as T, meta: json.meta }
}

export function formatBeijing(iso?: string) {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const parts = new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).formatToParts(d)
  const g = (t: string) => parts.find((p) => p.type === t)?.value || ''
  return `${g('year')}-${g('month')}-${g('day')} ${g('hour')}:${g('minute')}:${g('second')}`
}

export const api = {
  health: () => req<Record<string, unknown>>('/api/v1/health'),
  listCards: (q = '', tag = '', page = 1) =>
    req<Card[]>(`/api/v1/cards?q=${encodeURIComponent(q)}&tag=${encodeURIComponent(tag)}&page=${page}&page_size=80`),
  getCard: (id: string) => req<Card>(`/api/v1/cards/${id}`),
  createCard: (title: string, body = '', tags: string[] = []) =>
    req<Card>('/api/v1/cards', { method: 'POST', body: JSON.stringify({ title, body, tags }) }),
  updateCard: (id: string, patch: { title?: string; body?: string; tags?: string[] }) =>
    req<Card>(`/api/v1/cards/${id}`, { method: 'PATCH', body: JSON.stringify(patch) }),
  deleteCard: (id: string) => req<{ deleted: boolean }>(`/api/v1/cards/${id}`, { method: 'DELETE' }),
  suggest: (q: string) => req<Card[]>(`/api/v1/cards/suggest?q=${encodeURIComponent(q)}`),
  graph: () => req<GraphPayload>('/api/v1/graph'),
  savePositions: (positions: { id: string; x: number; y: number }[]) =>
    req('/api/v1/graph/positions', { method: 'PATCH', body: JSON.stringify({ positions }) }),
  tags: () => req<Tag[]>('/api/v1/tags'),
  clip: (url: string) => req<Card>('/api/v1/clips', { method: 'POST', body: JSON.stringify({ url }) }),
}
