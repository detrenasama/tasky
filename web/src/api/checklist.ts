import { req } from './client'
import type { ChecklistItem } from '../types'

export const checklist = (id: number) => req<ChecklistItem[]>('GET', `/subtasks/${id}/checklist`)
export const checklistCounts = (pid: number) =>
  req<Record<string, [number, number]>>('GET', `/projects/${pid}/checklistcounts`)
export const createChecklistItem = (id: number, text: string) =>
  req<ChecklistItem>('POST', `/subtasks/${id}/checklist`, { text })
export const updateChecklistText = (id: number, text: string) =>
  req<{ ok: boolean }>('PUT', `/checklist/${id}/text`, { text })
export const setChecklistStatus = (id: number, status: string) =>
  req<{ ok: boolean }>('PUT', `/checklist/${id}/status`, { status })
export const moveChecklistItem = (id: number, dir: number) =>
  req<{ ok: boolean }>('POST', `/checklist/${id}/move`, { dir })
export const deleteChecklistItem = (id: number) => req<{ ok: boolean }>('DELETE', `/checklist/${id}`)
