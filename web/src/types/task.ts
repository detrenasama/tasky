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
