import { req } from './client'
import type { Task, Subtask, Link } from '../types'

export const tasksByProject = (pid: number) => req<Task[]>('GET', `/projects/${pid}/tasks`)
export const subtasksByProject = (pid: number) => req<Subtask[]>('GET', `/projects/${pid}/subtasks`)
export const createTask = (pid: number, title: string) =>
  req<Task>('POST', `/projects/${pid}/tasks`, { title })
export const deleteTask = (id: number) => req<{ ok: boolean }>('DELETE', `/tasks/${id}`)
export const updateTaskTitle = (id: number, title: string) =>
  req<{ ok: boolean }>('PUT', `/tasks/${id}/title`, { title })
export const taskDescription = (id: number) =>
  req<{ description: string }>('GET', `/tasks/${id}/description`)
export const updateTaskDescription = (id: number, description: string) =>
  req<{ ok: boolean }>('PUT', `/tasks/${id}/description`, { description })
export const moveTask = (id: number, dir: number) =>
  req<{ ok: boolean }>('POST', `/tasks/${id}/move`, { dir })
export const reorderTask = (id: number, to: number) =>
  req<{ ok: boolean }>('POST', `/tasks/${id}/reorder`, { to })
export const subtasksByTask = (tid: number) => req<Subtask[]>('GET', `/tasks/${tid}/subtasks`)
export const createSubtask = (tid: number, title: string) =>
  req<Subtask>('POST', `/tasks/${tid}/subtasks`, { title })
export const deleteSubtask = (id: number) => req<{ ok: boolean }>('DELETE', `/subtasks/${id}`)
export const updateSubtaskTitle = (id: number, title: string) =>
  req<{ ok: boolean }>('PUT', `/subtasks/${id}/title`, { title })
export const subtaskDescription = (id: number) =>
  req<{ description: string }>('GET', `/subtasks/${id}/description`)
export const updateSubtaskDescription = (id: number, description: string) =>
  req<{ ok: boolean }>('PUT', `/subtasks/${id}/description`, { description })
export const moveSubtask = (id: number, dir: number) =>
  req<{ ok: boolean }>('POST', `/subtasks/${id}/move`, { dir })
export const reorderSubtask = (id: number, taskId: number, to: number) =>
  req<{ ok: boolean }>('POST', `/subtasks/${id}/reorder`, { task_id: taskId, to })
export const taskLinks = (id: number) => req<Link[]>('GET', `/tasks/${id}/links`)
export const createTaskLink = (id: number, name: string, url: string) =>
  req<Link>('POST', `/tasks/${id}/links`, { name, url })
export const updateTaskLink = (id: number, name: string, url: string) =>
  req<{ ok: boolean }>('PUT', `/tasklinks/${id}`, { name, url })
export const deleteTaskLink = (id: number) => req<{ ok: boolean }>('DELETE', `/tasklinks/${id}`)
export const subtaskLinks = (id: number) => req<Link[]>('GET', `/subtasks/${id}/links`)
export const createSubtaskLink = (id: number, name: string, url: string) =>
  req<Link>('POST', `/subtasks/${id}/links`, { name, url })
export const updateSubtaskLink = (id: number, name: string, url: string) =>
  req<{ ok: boolean }>('PUT', `/sublinks/${id}`, { name, url })
export const deleteSubtaskLink = (id: number) => req<{ ok: boolean }>('DELETE', `/sublinks/${id}`)
