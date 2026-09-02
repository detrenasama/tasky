import { req } from './client'
import type { JournalEntry } from '../types'

export const journal = (id: number) => req<JournalEntry[]>('GET', `/subtasks/${id}/journal`)
export const createJournal = (id: number, text: string) =>
  req<JournalEntry>('POST', `/subtasks/${id}/journal`, { text })
export const updateJournal = (id: number, text: string) =>
  req<{ ok: boolean }>('PUT', `/journal/${id}`, { text })
