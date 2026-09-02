import { useEffect, useRef, useState } from 'react'
import { api } from '../../api'
import type { Task, Subtask } from '../../types'
import { arrayMove } from './helpers'

const HOLD_MS = 400
const MOVE_TOL = 8

export function useTaskDrag(params: {
  tasks: Task[]
  subs: Subtask[]
  selTaskId: number | null
  setTasks: React.Dispatch<React.SetStateAction<Task[]>>
  setSubs: React.Dispatch<React.SetStateAction<Subtask[]>>
  setSelTaskId: React.Dispatch<React.SetStateAction<number | null>>
  onError: (m: string) => void
  reload: () => Promise<void>
}) {
  const { tasks, subs, selTaskId, setTasks, setSubs, setSelTaskId, onError, reload } = params

  const taskListRef = useRef<HTMLUListElement>(null)
  const subListRef = useRef<HTMLUListElement>(null)
  const [dragTask, setDragTask] = useState<{ id: number; from: number; to: number } | null>(null)
  const [dragSub, setDragSub] = useState<{ id: number; fromTaskId: number; from: number; to: number; toTaskId: number } | null>(null)
  const [ghost, setGhost] = useState<{ kind: 'task' | 'sub'; x: number; y: number; w: number; h: number; id: number } | null>(null)
  const dragRef = useRef<{ timer: number | null; startX: number; startY: number; w: number; h: number; kind: 'task' | 'sub'; id: number; from: number; fromTaskId: number | null; pointerId: number | null }>({ timer: null, startX: 0, startY: 0, w: 0, h: 0, kind: 'task', id: 0, from: 0, fromTaskId: null, pointerId: null })
  const suppressClick = useRef(false)

  const displayTasks = (() => {
    if (!dragTask) return tasks
    const without = tasks.filter((t) => t.id !== dragTask.id)
    const res: (Task | { __placeholder: true; key: string })[] = [...without] as any
    res.splice(dragTask.to, 0, { __placeholder: true, key: 'ph-task' } as any)
    return res as Task[]
  })()

  const displaySubsFor = (taskId: number) => {
    const base = subs.filter((s) => s.task_id === taskId)
    if (!dragSub) return base
    if (dragSub.fromTaskId === taskId && dragSub.toTaskId !== taskId) {
      return base.filter((s) => s.id !== dragSub.id)
    }
    if (dragSub.fromTaskId !== taskId || dragSub.toTaskId !== taskId) return base
    const without = base.filter((s) => s.id !== dragSub.id)
    const res: any[] = [...without]
    res.splice(dragSub.to, 0, { __placeholder: true, key: 'ph-sub' } as any)
    return res as Subtask[]
  }

  const clearDragTimer = () => {
    if (dragRef.current.timer !== null) {
      window.clearTimeout(dragRef.current.timer)
      dragRef.current.timer = null
    }
  }

  const onTaskPointerDown = (e: React.PointerEvent, t: Task, idx: number) => {
    if (e.button !== 0) return
    if ((e.target as HTMLElement).closest('.resizer')) return
    const sx = e.clientX, sy = e.clientY, pid = e.pointerId
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
    dragRef.current = { timer: null, startX: sx, startY: sy, w: r.width, h: r.height, kind: 'task', id: t.id, from: idx, fromTaskId: null, pointerId: pid }
    const timer = window.setTimeout(() => {
      dragRef.current.timer = null
      setDragTask({ id: t.id, from: idx, to: idx })
      setGhost({ kind: 'task', x: sx - r.width / 2, y: sy - r.height / 2, w: r.width, h: r.height, id: t.id })
      suppressClick.current = true
      try { (e.currentTarget as HTMLElement).setPointerCapture(pid) } catch {}
    }, HOLD_MS)
    dragRef.current.timer = timer as unknown as number
  }

  const onSubPointerDown = (e: React.PointerEvent, s: Subtask, idx: number) => {
    if (e.button !== 0) return
    if ((e.target as HTMLElement).closest('.resizer')) return
    const sx = e.clientX, sy = e.clientY, pid = e.pointerId
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
    dragRef.current = { timer: null, startX: sx, startY: sy, w: r.width, h: r.height, kind: 'sub', id: s.id, from: idx, fromTaskId: s.task_id, pointerId: pid }
    const timer = window.setTimeout(() => {
      dragRef.current.timer = null
      setDragSub({ id: s.id, fromTaskId: s.task_id, from: idx, to: idx, toTaskId: s.task_id })
      setGhost({ kind: 'sub', x: sx - r.width / 2, y: sy - r.height / 2, w: r.width, h: r.height, id: s.id })
      suppressClick.current = true
      try { (e.currentTarget as HTMLElement).setPointerCapture(pid) } catch {}
    }, HOLD_MS)
    dragRef.current.timer = timer as unknown as number
  }

  useEffect(() => {
    const onMove = (ev: PointerEvent) => {
      const dr = dragRef.current
      if (dr.timer !== null) {
        if (Math.hypot(ev.clientX - dr.startX, ev.clientY - dr.startY) > MOVE_TOL) {
          clearDragTimer()
        }
        return
      }
      if (dragTask || dragSub) {
        setGhost((prev) => (prev ? { ...prev, x: ev.clientX - prev.w / 2, y: ev.clientY - 14 } : prev))
      }
      if (dragTask) {
        const ul = taskListRef.current
        if (!ul) return
        const items = Array.from(ul.querySelectorAll<HTMLElement>('[data-task-id]'))
        let cnt = 0
        for (const el of items) {
          if (el.dataset.taskId === String(dragTask.id)) continue
          const r = el.getBoundingClientRect()
          const mid = r.top + r.height / 2
          if (ev.clientY > mid) cnt++
        }
        const max = tasks.length - 1
        cnt = Math.max(0, Math.min(max, cnt))
        if (cnt !== dragTask.to) setDragTask({ ...dragTask, to: cnt })
      } else if (dragSub) {
        const taskItems = taskListRef.current ? Array.from(taskListRef.current.querySelectorAll<HTMLElement>('[data-task-id]')) : []
        let hoverTaskId: number | null = null
        for (const el of taskItems) {
          const r = el.getBoundingClientRect()
          if (ev.clientX >= r.left && ev.clientX <= r.right && ev.clientY >= r.top && ev.clientY <= r.bottom) {
            hoverTaskId = Number(el.dataset.taskId)
            break
          }
        }
        const ul = subListRef.current
        if (hoverTaskId !== null && hoverTaskId !== dragSub.fromTaskId) {
          const targetCount = subs.filter((s) => s.task_id === hoverTaskId).length
          if (dragSub.toTaskId !== hoverTaskId || dragSub.to !== targetCount) {
            setDragSub({ ...dragSub, toTaskId: hoverTaskId, to: targetCount })
          }
          return
        }
        if (!ul) return
        const items = Array.from(ul.querySelectorAll<HTMLElement>('[data-sub-id]'))
        let cnt = 0
        for (const el of items) {
          if (el.dataset.subId === String(dragSub.id)) continue
          const r = el.getBoundingClientRect()
          const mid = r.top + r.height / 2
          if (ev.clientY > mid) cnt++
        }
        const max = subs.filter((s) => s.task_id === dragSub.fromTaskId).length - 1
        if (dragSub.toTaskId !== dragSub.fromTaskId) {
          cnt = Math.max(0, Math.min(max, cnt))
          setDragSub({ ...dragSub, toTaskId: dragSub.fromTaskId, to: cnt })
          return
        }
        cnt = Math.max(0, Math.min(max, cnt))
        if (cnt !== dragSub.to) setDragSub({ ...dragSub, to: cnt })
      }
    }
    const onUp = async (ev: PointerEvent) => {
      const dr = dragRef.current
      if (dr.timer !== null) {
        clearDragTimer()
        dragRef.current.pointerId = null
        return
      }
      if (dragTask) {
        const { id, from, to } = dragTask
        setDragTask(null)
        setGhost(null)
        dragRef.current.pointerId = null
        setTimeout(() => { suppressClick.current = false }, 50)
        try { (ev.target as HTMLElement)?.releasePointerCapture?.(dr.pointerId ?? ev.pointerId) } catch {}
        if (from === to) return
        setTasks((prev) => arrayMove(prev, from, to))
        try {
          await api.reorderTask(id, to)
          reload()
        } catch (e) { onError(String(e)); reload() }
      } else if (dragSub) {
        const { id, fromTaskId, toTaskId, from, to } = dragSub
        setDragSub(null)
        setGhost(null)
        dragRef.current.pointerId = null
        setTimeout(() => { suppressClick.current = false }, 50)
        try { (ev.target as HTMLElement)?.releasePointerCapture?.(dr.pointerId ?? ev.pointerId) } catch {}
        const isCross = fromTaskId !== toTaskId
        if (!isCross && from === to) return
        setSubs((prev) => {
          const a = [...prev]
          const idx = a.findIndex((s) => s.id === id)
          if (idx < 0) return prev
          const [orig] = a.splice(idx, 1)
          const moved = { ...orig, task_id: toTaskId }
          const targetSibs = a.filter((s) => s.task_id === toTaskId)
          let insertAt: number
          if (targetSibs.length === 0) {
            insertAt = a.length
          } else {
            if (to >= targetSibs.length) {
              const last = targetSibs[targetSibs.length - 1]
              insertAt = a.findIndex((s) => s.id === last.id) + 1
            } else {
              const before = targetSibs[to]
              insertAt = a.findIndex((s) => s.id === before.id)
            }
          }
          a.splice(insertAt, 0, moved)
          return a
        })
        if (isCross && toTaskId !== selTaskId) setSelTaskId(toTaskId)
        try {
          await api.reorderSubtask(id, toTaskId, to)
          reload()
        } catch (e) { onError(String(e)); reload() }
      } else {
        dragRef.current.pointerId = null
      }
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    window.addEventListener('pointercancel', onUp as any)
    return () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      window.removeEventListener('pointercancel', onUp as any)
    }
  }, [dragTask, dragSub, tasks, subs, selTaskId])

  return {
    taskListRef,
    subListRef,
    dragTask,
    dragSub,
    ghost,
    suppressClick,
    displayTasks,
    displaySubsFor,
    onTaskPointerDown,
    onSubPointerDown,
  }
}
