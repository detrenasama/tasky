export type FeedItem = {
  id: number
  kind: string
  title: string
  body: string
  ref_task_id?: number | null
  ref_subtask_id?: number | null
  dedup_key: string
  created_at: string // RFC3339 from json, but server sends unix? db json uses time.Time -> RFC3339
  is_read: boolean
  is_archived: boolean
}
