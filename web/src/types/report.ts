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
