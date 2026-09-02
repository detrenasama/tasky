import type { StatusDef, Subtask } from '../../types'

export function statusColor(name: string, statuses: StatusDef[]): string {
  const s = statuses.find((x) => x.name === name)
  return s?.color || 'var(--grey)'
}

export function isDoneSet(statuses: StatusDef[]): Set<string> {
  return new Set(statuses.filter((s) => s.type === 'done').map((s) => s.name))
}

export function taskTime(taskId: number, subs: Subtask[]): number {
  return subs
    .filter((s) => s.task_id === taskId)
    .reduce((a, s) => {
      let t = s.total_seconds
      if (s.active_since) t += Math.floor(Date.now() / 1000) - s.active_since
      return a + t
    }, 0)
}

export function arrayMove<T>(arr: T[], from: number, to: number): T[] {
  const a = [...arr]
  const [v] = a.splice(from, 1)
  a.splice(to, 0, v)
  return a
}
