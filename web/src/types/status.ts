import type { Subtask } from './task'

export interface StatusDef {
  id: number
  name: string
  type: string
  color: string
  note_prompt: string
  is_quick: boolean
  sort_order: number
}

export interface StatusHistoryEntry {
  from: string
  to: string
  note: string
  created_at: string
}

export type StatusOwner = 'task' | 'subtask'

export interface StatusPoll {
  today_seconds: number
  running: Subtask | null
}
