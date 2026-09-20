import { req } from './client'
import type { FeedItem } from '../types/feed'

export function listFeed(params?: { kind?: string; unread_only?: boolean; archived?: boolean }): Promise<FeedItem[]> {
  const qs = new URLSearchParams()
  if (params?.kind) qs.set('kind', params.kind)
  if (params?.unread_only) qs.set('unread_only', '1')
  if (params?.archived) qs.set('archived', '1')
  const q = qs.toString() ? '?' + qs.toString() : ''
  return req<FeedItem[]>('GET', `/api/feed${q}`)
}

export function markFeedRead(id: number): Promise<{ ok: boolean }> {
  return req('POST', `/api/feed/${id}/read`)
}

export function feedUnreadCount(): Promise<{ count: number }> {
  return req('GET', '/api/feed/unread_count')
}

export function archiveFeed(): Promise<{ archived: number }> {
  return req('POST', '/api/feed/archive')
}

export function deleteArchivedFeed(): Promise<{ deleted: number }> {
  return req('DELETE', '/api/feed/archived')
}

export function setTaskDates(id: number, due: string | null, sched: string | null): Promise<{ ok: boolean }> {
  return req('PUT', `/api/tasks/${id}/dates`, { due_date: due, schedule_date: sched })
}

export function setSubtaskDates(id: number, due: string | null, sched: string | null): Promise<{ ok: boolean }> {
  return req('PUT', `/api/subtasks/${id}/dates`, { due_date: due, schedule_date: sched })
}
