import { req } from './client'
import type { StatusDef, StatusHistoryEntry, StatusOwner } from '../types'

export const statuses = () => req<StatusDef[]>('GET', '/statuses')
export const createStatus = (s: Omit<StatusDef, 'id' | 'sort_order'>) =>
  req<StatusDef>('POST', '/statuses', s)
export const updateStatus = (id: number, s: Omit<StatusDef, 'id' | 'sort_order'>) =>
  req<{ ok: boolean }>('PUT', `/statuses/${id}`, s)
export const deleteStatus = (id: number) => req<{ ok: boolean }>('DELETE', `/statuses/${id}`)
export const setStatus = (owner: StatusOwner, id: number, to: string, note: string) =>
  req<{ ok: boolean }>('POST', '/status', { owner, id, to, note })
export const statusHistory = (owner: StatusOwner, id: number) =>
  req<StatusHistoryEntry[]>('GET', `/status/history/${owner}/${id}`)
