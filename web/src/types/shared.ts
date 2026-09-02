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
