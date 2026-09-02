import { req } from './client'
import type { TimeEntry, Subtask, StatusPoll } from '../types'

export const timeBySubtask = (id: number) => req<TimeEntry[]>('GET', `/subtasks/${id}/time`)
export const startSubtask = (id: number) => req<{ ok: boolean }>('POST', `/subtasks/${id}/start`)
export const stopSubtask = (id: number) => req<{ ok: boolean }>('POST', `/subtasks/${id}/stop`)
export const updateTimeEntry = (id: number, startedAt: string, endedAt: string | null) =>
  req<{ ok: boolean }>('PUT', `/timeentries/${id}`, { started_at: startedAt, ended_at: endedAt })
export const deleteTimeEntry = (id: number) => req<{ ok: boolean }>('DELETE', `/timeentries/${id}`)
export const todaySeconds = () => req<{ seconds: number }>('GET', '/today')
export const weeklySeconds = () => req<{ seconds: number }>('GET', '/weekly')
export const running = () => req<Subtask | null>('GET', '/running')
export const statusPoll = () => req<StatusPoll>('GET', '/status')
