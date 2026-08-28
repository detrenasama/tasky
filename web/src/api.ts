import type {
  Project,
  Task,
  Subtask,
  TimeEntry,
  Link,
  JournalEntry,
  ChecklistItem,
  StatusDef,
  StatusHistoryEntry,
  TagType,
  Tag,
  ReportEntry,
  ReportJournalEntry,
  StatusOwner,
  StatusPoll,
  Settings,
} from './types'

const BASE = '/api'

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`
    try {
      const j = await res.json()
      if (j && j.error) msg = j.error
    } catch {
      /* оставляем статус-текст */
    }
    throw new Error(msg)
  }
  if (res.status === 204) return undefined as T
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

export const api = {
  // Проекты.
  projects: () => req<Project[]>('GET', '/projects'),
  createProject: (name: string) => req<Project>('POST', '/projects', { name }),
  deleteProject: (id: number) => req<{ ok: boolean }>('DELETE', `/projects/${id}`),
  projectDescription: (id: number) =>
    req<{ description: string }>('GET', `/projects/${id}/description`),
  updateProjectDescription: (id: number, description: string) =>
    req<{ ok: boolean }>('PUT', `/projects/${id}/description`, { description }),
  projectLinks: (id: number) => req<Link[]>('GET', `/projects/${id}/links`),
  createProjectLink: (id: number, name: string, url: string) =>
    req<Link>('POST', `/projects/${id}/links`, { name, url }),
  updateProjectLink: (id: number, name: string, url: string) =>
    req<{ ok: boolean }>('PUT', `/projectlinks/${id}`, { name, url }),
  deleteProjectLink: (id: number) => req<{ ok: boolean }>('DELETE', `/projectlinks/${id}`),

  // Задачи и подзадачи.
  tasksByProject: (pid: number) => req<Task[]>('GET', `/projects/${pid}/tasks`),
  subtasksByProject: (pid: number) => req<Subtask[]>('GET', `/projects/${pid}/subtasks`),
  createTask: (pid: number, title: string) =>
    req<Task>('POST', `/projects/${pid}/tasks`, { title }),
  deleteTask: (id: number) => req<{ ok: boolean }>('DELETE', `/tasks/${id}`),
  updateTaskTitle: (id: number, title: string) =>
    req<{ ok: boolean }>('PUT', `/tasks/${id}/title`, { title }),
  taskDescription: (id: number) =>
    req<{ description: string }>('GET', `/tasks/${id}/description`),
  updateTaskDescription: (id: number, description: string) =>
    req<{ ok: boolean }>('PUT', `/tasks/${id}/description`, { description }),
  moveTask: (id: number, dir: number) =>
    req<{ ok: boolean }>('POST', `/tasks/${id}/move`, { dir }),
  subtasksByTask: (tid: number) => req<Subtask[]>('GET', `/tasks/${tid}/subtasks`),
  createSubtask: (tid: number, title: string) =>
    req<Subtask>('POST', `/tasks/${tid}/subtasks`, { title }),
  deleteSubtask: (id: number) => req<{ ok: boolean }>('DELETE', `/subtasks/${id}`),
  updateSubtaskTitle: (id: number, title: string) =>
    req<{ ok: boolean }>('PUT', `/subtasks/${id}/title`, { title }),
  subtaskDescription: (id: number) =>
    req<{ description: string }>('GET', `/subtasks/${id}/description`),
  updateSubtaskDescription: (id: number, description: string) =>
    req<{ ok: boolean }>('PUT', `/subtasks/${id}/description`, { description }),
  moveSubtask: (id: number, dir: number) =>
    req<{ ok: boolean }>('POST', `/subtasks/${id}/move`, { dir }),
  taskLinks: (id: number) => req<Link[]>('GET', `/tasks/${id}/links`),
  createTaskLink: (id: number, name: string, url: string) =>
    req<Link>('POST', `/tasks/${id}/links`, { name, url }),
  updateTaskLink: (id: number, name: string, url: string) =>
    req<{ ok: boolean }>('PUT', `/tasklinks/${id}`, { name, url }),
  deleteTaskLink: (id: number) => req<{ ok: boolean }>('DELETE', `/tasklinks/${id}`),
  subtaskLinks: (id: number) => req<Link[]>('GET', `/subtasks/${id}/links`),
  createSubtaskLink: (id: number, name: string, url: string) =>
    req<Link>('POST', `/subtasks/${id}/links`, { name, url }),
  updateSubtaskLink: (id: number, name: string, url: string) =>
    req<{ ok: boolean }>('PUT', `/sublinks/${id}`, { name, url }),
  deleteSubtaskLink: (id: number) => req<{ ok: boolean }>('DELETE', `/sublinks/${id}`),

  // Учёт времени.
  timeBySubtask: (id: number) => req<TimeEntry[]>('GET', `/subtasks/${id}/time`),
  startSubtask: (id: number) => req<{ ok: boolean }>('POST', `/subtasks/${id}/start`),
  stopSubtask: (id: number) => req<{ ok: boolean }>('POST', `/subtasks/${id}/stop`),
  updateTimeEntry: (id: number, startedAt: string, endedAt: string | null) =>
    req<{ ok: boolean }>('PUT', `/timeentries/${id}`, { started_at: startedAt, ended_at: endedAt }),
  deleteTimeEntry: (id: number) => req<{ ok: boolean }>('DELETE', `/timeentries/${id}`),
  todaySeconds: () => req<{ seconds: number }>('GET', '/today'),
  weeklySeconds: () => req<{ seconds: number }>('GET', '/weekly'),
  running: () => req<Subtask | null>('GET', '/running'),
  statusPoll: () => req<StatusPoll>('GET', '/status'),

  // Журнал.
  journal: (id: number) => req<JournalEntry[]>('GET', `/subtasks/${id}/journal`),
  createJournal: (id: number, text: string) =>
    req<JournalEntry>('POST', `/subtasks/${id}/journal`, { text }),
  updateJournal: (id: number, text: string) =>
    req<{ ok: boolean }>('PUT', `/journal/${id}`, { text }),

  // Чек-листы.
  checklist: (id: number) => req<ChecklistItem[]>('GET', `/subtasks/${id}/checklist`),
  checklistCounts: (pid: number) => req<Record<string, [number, number]>>('GET', `/projects/${pid}/checklistcounts`),
  createChecklistItem: (id: number, text: string) =>
    req<ChecklistItem>('POST', `/subtasks/${id}/checklist`, { text }),
  updateChecklistText: (id: number, text: string) =>
    req<{ ok: boolean }>('PUT', `/checklist/${id}/text`, { text }),
  setChecklistStatus: (id: number, status: string) =>
    req<{ ok: boolean }>('PUT', `/checklist/${id}/status`, { status }),
  moveChecklistItem: (id: number, dir: number) =>
    req<{ ok: boolean }>('POST', `/checklist/${id}/move`, { dir }),
  deleteChecklistItem: (id: number) => req<{ ok: boolean }>('DELETE', `/checklist/${id}`),

  // Статусы.
  statuses: () => req<StatusDef[]>('GET', '/statuses'),
  createStatus: (s: Omit<StatusDef, 'id' | 'sort_order'>) =>
    req<StatusDef>('POST', '/statuses', s),
  updateStatus: (id: number, s: Omit<StatusDef, 'id' | 'sort_order'>) =>
    req<{ ok: boolean }>('PUT', `/statuses/${id}`, s),
  deleteStatus: (id: number) => req<{ ok: boolean }>('DELETE', `/statuses/${id}`),
  setStatus: (owner: StatusOwner, id: number, to: string, note: string) =>
    req<{ ok: boolean }>('POST', '/status', { owner, id, to, note }),
  statusHistory: (owner: StatusOwner, id: number) =>
    req<StatusHistoryEntry[]>('GET', `/status/history/${owner}/${id}`),

  // Теги.
  tagTypes: () => req<TagType[]>('GET', '/tagtypes'),
  createTagType: (t: Omit<TagType, 'id' | 'sort_order'>) =>
    req<TagType>('POST', '/tagtypes', t),
  updateTagType: (id: number, t: Omit<TagType, 'id' | 'sort_order'>) =>
    req<{ ok: boolean }>('PUT', `/tagtypes/${id}`, t),
  deleteTagType: (id: number) => req<{ ok: boolean }>('DELETE', `/tagtypes/${id}`),
  taskTags: (id: number) => req<Tag[]>('GET', `/tasks/${id}/tags`),
  tagsByProject: (pid: number) => req<Record<string, Tag[]>>('GET', `/projects/${pid}/tags`),
  createTag: (tid: number, typeId: number, text: string, url: string) =>
    req<Tag>('POST', `/tasks/${tid}/tags`, { type_id: typeId, text, url }),
  updateTag: (id: number, typeId: number, text: string, url: string) =>
    req<{ ok: boolean }>('PUT', `/tags/${id}`, { type_id: typeId, text, url }),
  deleteTag: (id: number) => req<{ ok: boolean }>('DELETE', `/tags/${id}`),

  // Отчёты.
  reports: (from: string, to: string, projectId: number) =>
    req<ReportEntry[]>('GET', `/reports?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&project_id=${projectId}`),
  reportsJournal: (from: string, to: string) =>
    req<ReportJournalEntry[]>('GET', `/reports/journal?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`),
  reportsTags: (taskIds: number[]) =>
    req<Record<string, Tag[]>>('POST', '/reports/tags', { task_ids: taskIds }),

  // Настройки.
  settings: () => req<Settings>('GET', '/settings'),
  getSetting: (key: string) => req<{ value: string }>('GET', `/settings/${key}`),
  setSetting: (key: string, value: string) =>
    req<{ ok: boolean }>('PUT', `/settings/${key}`, { value }),

  // Версия бинаря (для шапки).
  version: () => req<{ version: string }>('GET', '/version'),
}
