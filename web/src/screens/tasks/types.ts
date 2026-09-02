import type { Link, Tag, TimeEntry, JournalEntry, ChecklistItem, StatusHistoryEntry } from '../../types'

export type Detail = {
  description: string
  links: Link[]
  tags: Tag[]
  time: TimeEntry[]
  journal: JournalEntry[]
  checklist: ChecklistItem[]
  history: StatusHistoryEntry[]
}

export const EMPTY: Detail = { description: '', links: [], tags: [], time: [], journal: [], checklist: [], history: [] }
