// Типы зеркалят db-структуры (JSON-теги в internal/db/types.go).

export interface Project {
  id: number
  name: string
  desc: string
  created_at: string
}

export interface Task {
  id: number
  project_id: number
  title: string
  description: string
  status: string
  created_at: string
  completed_at: string | null
  sub_count: number
}

export interface Subtask {
  id: number
  task_id: number
  title: string
  description: string
  status: string
  sort_order: number
  created_at: string
  completed_at: string | null
  total_seconds: number
  active_since: number | null
}

export interface TimeEntry {
  id: number
  subtask_id: number
  started_at: string
  ended_at: string | null
  note: string
}

export interface Link {
  id: number
  owner_id: number
  name: string
  url: string
  created_at: string
}

export interface JournalEntry {
  id: number
  subtask_id: number
  created_at: string
  text: string
}

export interface ChecklistItem {
  id: number
  subtask_id: number
  text: string
  status: string
  sort_order: number
  created_at: string
  status_changed_at: string
}

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

export interface TagType {
  id: number
  name: string
  kind: string
  color: string
  sort_order: number
}

export interface Tag {
  id: number
  task_id: number
  type_id: number
  type_name: string
  kind: string
  color: string
  text: string
  url: string
  created_at: string
}

export interface ReportEntry {
  project_id: number
  project_name: string
  task_id: number
  task_title: string
  subtask_id: number
  subtask_title: string
  seconds: number
}

export interface ReportJournalEntry {
  subtask_id: number
  created_at: string
  text: string
}

export type StatusOwner = 'task' | 'subtask'

export interface StatusPoll {
  today_seconds: number
  running: Subtask | null
}

export interface Settings {
  theme?: string
  hide_days?: string
}
