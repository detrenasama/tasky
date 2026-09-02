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
