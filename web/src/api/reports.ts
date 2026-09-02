import { req } from './client'
import type { ReportEntry, ReportJournalEntry, Tag } from '../types'

export const reports = (from: string, to: string, projectId: number) =>
  req<ReportEntry[]>('GET', `/reports?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&project_id=${projectId}`)
export const reportsJournal = (from: string, to: string) =>
  req<ReportJournalEntry[]>('GET', `/reports/journal?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`)
export const reportsTags = (taskIds: number[]) =>
  req<Record<string, Tag[]>>('POST', '/reports/tags', { task_ids: taskIds })
