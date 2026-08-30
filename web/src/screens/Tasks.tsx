import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { api } from '../api'
import type {
  Project,
  Task,
  Subtask,
  Link,
  Tag,
  TimeEntry,
  JournalEntry,
  ChecklistItem,
  StatusDef,
  StatusHistoryEntry,
} from '../types'
import { fmtDuration, fmtDateTime } from '../fmt'
import { Button, useConfirm, useColumnWidth, Modal } from '../ui'

type Sel = { kind: 'task' | 'subtask'; id: number } | null

type Detail = {
  description: string
  links: Link[]
  tags: Tag[]
  time: TimeEntry[]
  journal: JournalEntry[]
  checklist: ChecklistItem[]
  history: StatusHistoryEntry[]
}

const EMPTY: Detail = { description: '', links: [], tags: [], time: [], journal: [], checklist: [], history: [] }

export default function Tasks({ onError }: { onError: (m: string) => void }) {
  const [projects, setProjects] = useState<Project[]>([])
  const [proj, setProj] = useState<Project | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [subs, setSubs] = useState<Subtask[]>([])
  const [selTaskId, setSelTaskId] = useState<number | null>(null)
  const [selSubId, setSelSubId] = useState<number | null>(null)
  const [taskDetail, setTaskDetail] = useState<Detail>(EMPTY)
  const [subDetail, setSubDetail] = useState<Detail>(EMPTY)
  const [statuses, setStatuses] = useState<StatusDef[]>([])
  const [runningId, setRunningId] = useState<number | null>(null)
  const [tagsMap, setTagsMap] = useState<Record<string, Tag[]>>({})
  const [checkMap, setCheckMap] = useState<Record<string, [number, number]>>({})

  const [addKind, setAddKind] = useState<null | 'task' | 'sub'>(null)
  const [addTitle, setAddTitle] = useState('')
  const [confirm, confirmNode] = useConfirm()

  const taskResize = useColumnWidth(320, 'tasky.col.tasks')
  const subResize = useColumnWidth(300, 'tasky.col.subs')

  useEffect(() => {
    api.statuses().then(setStatuses).catch((e) => onError(String(e)))
    api.projects().then((ps) => {
      setProjects(ps)
      if (ps[0]) openProject(ps[0])
    }).catch((e) => onError(String(e)))
    api.running().then((r) => setRunningId(r?.id ?? null)).catch(() => {})
  }, [])

  const openProject = (p: Project) => {
    setProj(p)
    setSelTaskId(null)
    setSelSubId(null)
    setTaskDetail(EMPTY)
    setSubDetail(EMPTY)
    Promise.all([api.tasksByProject(p.id), api.subtasksByProject(p.id)])
      .then(([ts, ss]) => {
        setTasks(ts)
        setSubs(ss)
      })
      .catch((e) => onError(String(e)))
    api.tagsByProject(p.id).then(setTagsMap).catch(() => setTagsMap({}))
    api.checklistCounts(p.id).then(setCheckMap).catch(() => setCheckMap({}))
  }

  // reload — перечитывает задачи/подзадачи проекта без сброса выбора.
  const reload = () => {
    if (!proj) return Promise.resolve()
    const pid = proj.id
    api.tagsByProject(pid).then(setTagsMap).catch(() => {})
    api.checklistCounts(pid).then(setCheckMap).catch(() => {})
    return Promise.all([api.tasksByProject(pid), api.subtasksByProject(pid)])
      .then(([ts, ss]) => {
        setTasks(ts)
        setSubs(ss)
      })
      .catch((e) => onError(String(e)))
  }

  const loadTaskDetail = (id: number) => {
    Promise.all([api.taskDescription(id), api.taskLinks(id), api.taskTags(id)])
      .then(([d, l, tg]) => setTaskDetail({ description: d.description, links: l, tags: tg, time: [], journal: [], checklist: [], history: [] }))
      .catch((e) => onError(String(e)))
  }
  const loadSubDetail = (id: number) => {
    Promise.all([
      api.subtaskDescription(id),
      api.subtaskLinks(id),
      api.timeBySubtask(id),
      api.journal(id),
      api.checklist(id),
      api.statusHistory('subtask', id),
    ])
      .then(([d, l, t, j, c, h]) => setSubDetail({ description: d.description, links: l, tags: [], time: t, journal: j, checklist: c, history: h }))
      .catch((e) => onError(String(e)))
  }

  const selectTask = (t: Task) => {
    setSelTaskId(t.id)
    setSelSubId(null)
    setSubDetail(EMPTY)
    loadTaskDetail(t.id)
  }

  const selectSub = (s: Subtask) => {
    setSelSubId(s.id)
    if (selTaskId == null) setSelTaskId(s.task_id)
    loadSubDetail(s.id)
  }

  // Создание задачи/подзадачи через модалку. Новый элемент сразу выбирается.
  const openAdd = (kind: 'task' | 'sub') => {
    setAddTitle('')
    setAddKind(kind)
  }
  const submitAdd = async () => {
    const title = addTitle.trim()
    if (!title || !addKind) return
    try {
      if (addKind === 'task') {
        if (!proj) return
        const t = await api.createTask(proj.id, title)
        await reload()
        setSelTaskId(t.id)
        setSelSubId(null)
        setSubDetail(EMPTY)
        loadTaskDetail(t.id)
      } else {
        if (!selTaskId) return
        const s = await api.createSubtask(selTaskId, title)
        await reload()
        setSelSubId(s.id)
        if (selTaskId == null) setSelTaskId(s.task_id)
        loadSubDetail(s.id)
      }
    } catch (e) {
      onError(String(e))
    }
    setAddKind(null)
  }

  const active: Sel = selSubId != null
    ? { kind: 'subtask', id: selSubId }
    : selTaskId != null
      ? { kind: 'task', id: selTaskId }
      : null

  const changeStatus = async (to: string, note = '') => {
    if (!active) return
    try {
      await api.setStatus(active.kind, active.id, to, note)
      reload()
      if (active.kind === 'subtask') loadSubDetail(active.id)
      else loadTaskDetail(active.id)
    } catch (e) {
      onError(String(e))
    }
  }

  const toggleTimer = async () => {
    if (selSubId == null) return
    const s = subs.find((x) => x.id === selSubId)
    if (!s) return
    try {
      if (runningId === s.id) await api.stopSubtask(s.id)
      else await api.startSubtask(s.id)
      const r = await api.running()
      setRunningId(r?.id ?? null)
      reload()
    } catch (e) {
      onError(String(e))
    }
  }

  const delTask = async (t: Task) => {
    if (!(await confirm(`Удалить задачу «${t.title}»?`))) return
    try {
      await api.deleteTask(t.id)
      if (selTaskId === t.id) {
        setSelTaskId(null)
        setSelSubId(null)
        setTaskDetail(EMPTY)
        setSubDetail(EMPTY)
      }
      reload()
    } catch (e) {
      onError(String(e))
    }
  }
  const delSub = async (s: Subtask) => {
    if (!(await confirm(`Удалить подзадачу «${s.title}»?`))) return
    try {
      await api.deleteSubtask(s.id)
      if (selSubId === s.id) {
        setSelSubId(null)
        setSubDetail(EMPTY)
      }
      reload()
    } catch (e) {
      onError(String(e))
    }
  }

  // Хелперы для двухстрочных карточек
  const doneSet = new Set(statuses.filter((s) => s.type === 'done').map((s) => s.name))
  const isDone = (name: string) => doneSet.has(name)
  const taskTime = (taskId: number) =>
    subs
      .filter((s) => s.task_id === taskId)
      .reduce((a, s) => {
        let t = s.total_seconds
        if (s.active_since) t += Math.floor(Date.now() / 1000) - s.active_since
        return a + t
      }, 0)
  const taskDoneCount = (taskId: number) =>
    subs.filter((s) => s.task_id === taskId && doneSet.has(s.status)).length

  // --- Drag-and-drop (long-press 400ms, mouse + touch) ---
  const HOLD_MS = 400
  const MOVE_TOL = 8
  const taskListRef = useRef<HTMLUListElement>(null)
  const subListRef = useRef<HTMLUListElement>(null)
  const [dragTask, setDragTask] = useState<{ id: number; from: number; to: number } | null>(null)
  const [dragSub, setDragSub] = useState<{ id: number; fromTaskId: number; from: number; to: number; toTaskId: number } | null>(null)
  const [ghost, setGhost] = useState<{ kind: 'task' | 'sub'; x: number; y: number; w: number; h: number; id: number } | null>(null)
  const dragRef = useRef<{ timer: number | null; startX: number; startY: number; w: number; h: number; kind: 'task' | 'sub'; id: number; from: number; fromTaskId: number | null; pointerId: number | null }>({ timer: null, startX: 0, startY: 0, w: 0, h: 0, kind: 'task', id: 0, from: 0, fromTaskId: null, pointerId: null })
  const suppressClick = useRef(false)

  const arrayMove = <T,>(arr: T[], from: number, to: number): T[] => {
    const a = [...arr]
    const [v] = a.splice(from, 1)
    a.splice(to, 0, v)
    return a
  }
  // Для плавающего ghost показываем placeholder вместо перетаскиваемого элемента
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
    // если перетаскивание в другую задачу — в текущем списке просто убираем элемент
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
      // если ещё не активирован drag — проверяем толерантность
      if (dr.timer !== null) {
        if (Math.hypot(ev.clientX - dr.startX, ev.clientY - dr.startY) > MOVE_TOL) {
          clearDragTimer()
        }
        return
      }
      // плавающий ghost за курсором
      if (dragTask || dragSub) {
        setGhost((prev) => (prev ? { ...prev, x: ev.clientX - prev.w / 2, y: ev.clientY - 14 } : prev))
      }
      if (dragTask) {
        const ul = taskListRef.current
        if (!ul) return
        const items = Array.from(ul.querySelectorAll<HTMLElement>('[data-task-id]'))
        // считаем, сколько элементов выше указателя (исключая перетаскиваемый)
        let cnt = 0
        for (const el of items) {
          if (el.dataset.taskId === String(dragTask.id)) continue
          const r = el.getBoundingClientRect()
          const mid = r.top + r.height / 2
          if (ev.clientY > mid) cnt++
        }
        // clamp
        const max = tasks.length - 1
        cnt = Math.max(0, Math.min(max, cnt))
        // корректировка: если перетаскиваемый был выше, индексы сдвинуты
        // наш cnt уже считает в списке без него — он и есть to
        if (cnt !== dragTask.to) setDragTask({ ...dragTask, to: cnt })
      } else if (dragSub) {
        // проверяем, наведен ли указатель на задачу в левой колонке (для переноса в другую задачу)
        const taskItems = taskListRef.current ? Array.from(taskListRef.current.querySelectorAll<HTMLElement>('[data-task-id]')) : []
        let hoverTaskId: number | null = null
        for (const el of taskItems) {
          const r = el.getBoundingClientRect()
          if (ev.clientX >= r.left && ev.clientX <= r.right && ev.clientY >= r.top && ev.clientY <= r.bottom) {
            hoverTaskId = Number(el.dataset.taskId)
            break
          }
        }
        // также проверяем подзадачи текущего списка для внутри-задачной сортировки
        const ul = subListRef.current
        if (hoverTaskId !== null && hoverTaskId !== dragSub.fromTaskId) {
          // перенос в другую задачу — в конец
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
          // вернулись в исходную задачу
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
        // снимаем suppress через тик чтобы click не сработал
        setTimeout(() => { suppressClick.current = false }, 50)
        try { (ev.target as HTMLElement)?.releasePointerCapture?.(dr.pointerId ?? ev.pointerId) } catch {}
        if (from === to) return
        // оптимистично
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
        // оптимистично: для внутри-задачного — локально переставим, для межзадачного — сменим task_id
        setSubs((prev) => {
          const a = [...prev]
          const idx = a.findIndex((s) => s.id === id)
          if (idx < 0) return prev
          const [orig] = a.splice(idx, 1)
          const moved = { ...orig, task_id: toTaskId }
          // найти позицию вставки в flat списке: считаем сколько подзадач целевого таска до to
          const targetSibs = a.filter((s) => s.task_id === toTaskId)
          let insertAt: number
          if (targetSibs.length === 0) {
            // вставить после последней подзадачи любого таска? — найдём первую позицию после всех или в конец
            insertAt = a.length
            // если переносим в пустую задачу — в конец flat массива
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

  return (
    <div className="flex" style={{ alignItems: 'stretch', height: '100%' }}>
      {/* Колонка 1: проект + задачи */}
      <div className="panel col" style={{ width: taskResize.width, position: 'relative', overflow: 'auto' }}>
        <select value={proj?.id ?? ''} onChange={(e) => {
          const p = projects.find((x) => x.id === Number(e.target.value))
          if (p) openProject(p)
        }}>
          {projects.map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
        <div className="toolbar">
          {proj && <Button color="accent" icon="plus" label="Создать задачу" onClick={() => openAdd('task')} />}
        </div>
        {proj ? (
          <ul className={`list${dragTask ? ' dragging' : ''}`} ref={taskListRef}>
            {displayTasks.map((t: any, idx: number) => {
              if (t.__placeholder) {
                return <li key="ph-task" className="drag-placeholder" style={{ height: ghost?.h ? `${ghost.h}px` : '58px' }} />
              }
              const col = statusColor(t.status, statuses)
              const done = isDone(t.status)
              const secs = taskTime(t.id)
              const hasTime = secs > 0
              const hasCount = t.sub_count > 0
              const k = taskDoneCount(t.id)
              const tags = tagsMap[String(t.id)] || []
              const isDropTarget = dragSub?.toTaskId === t.id && dragSub?.id !== t.id
              return (
                <li
                  key={t.id}
                  data-task-id={t.id}
                  className={`row task-row${selTaskId === t.id ? ' selected' : ''}${done ? ' task-row--done' : ''}${isDropTarget ? ' drop-target' : ''}`}
                  style={{ borderLeftColor: col }}
                  onClick={() => { if (suppressClick.current) return; selectTask(t) }}
                  onPointerDown={(e) => onTaskPointerDown(e, t, idx)}
                >
                  <div className="task-row__line1">
                    <span className="title">{t.title}</span>
                    {(hasTime || hasCount) && (
                      <span className="task-row__meta muted small">
                        {hasTime && fmtDuration(secs)}
                        {hasTime && hasCount && <span className="task-row__bullet"> • </span>}
                        {hasCount && `${k}/${t.sub_count}`}
                      </span>
                    )}
                  </div>
                  <div className="task-row__line2">
                    <span className="task-row__status" style={{ color: col }}>{t.status}</span>
                    {tags.length > 0 && (
                      <span className="task-row__tags">
                        {tags.slice(0, 3).map((tg) => (
                          <span key={tg.id} className="tag task-row__tag" style={{ background: tg.color || 'var(--panel2)' }}>{tg.text}</span>
                        ))}
                        {tags.length > 3 && <span className="muted small">+{tags.length - 3}</span>}
                      </span>
                    )}
                  </div>
                </li>
              )
            })}
          </ul>
        ) : (
          <p className="muted">{projects.length === 0 ? 'Создайте проект' : 'Выберите проект'}</p>
        )}
        <div className="resizer" onMouseDown={taskResize.onDown} />
      </div>

      {/* Колонка 2: подзадачи выбранной задачи */}
      <div className="panel col" style={{ width: subResize.width, position: 'relative', overflow: 'auto' }}>
        <div className="flex between" style={{ marginBottom: 8 }}>
          <strong className="col-head" style={{ margin: 0 }}>Подзадачи</strong>
          {selTaskId && <Button color="accent" icon="plus" label="Создать подзадачу" onClick={() => openAdd('sub')} />}
        </div>
        {!selTaskId && <p className="muted">Выберите задачу.</p>}
        {selTaskId && (
          <ul className={`list${dragSub ? ' dragging' : ''}`} ref={subListRef}>
            {displaySubsFor(selTaskId).map((s: any, sIdx: number) => {
              if (s.__placeholder) {
                return <li key="ph-sub" className="drag-placeholder" style={{ height: ghost?.h ? `${ghost.h}px` : '58px' }} />
              }
              const col = statusColor(s.status, statuses)
              const done = isDone(s.status)
              let secs = s.total_seconds
              if (s.active_since) secs += Math.floor(Date.now() / 1000) - s.active_since
              const hasTime = secs > 0
              const cc = checkMap[String(s.id)]
              const hasCheck = !!cc && cc[1] > 0
              const ck = cc ? `${cc[0]}/${cc[1]}` : ''
              const showBullet = hasTime && hasCheck
              return (
                <li
                  key={s.id}
                  data-sub-id={s.id}
                  data-task-id={s.task_id}
                  className={`row task-row${selSubId === s.id ? ' selected' : ''}${done ? ' task-row--done' : ''}`}
                  style={{ borderLeftColor: col }}
                  onClick={() => { if (suppressClick.current) return; selectSub(s) }}
                  onPointerDown={(e) => onSubPointerDown(e, s, sIdx)}
                >
                  <div className="task-row__line1">
                    <span className="title">{s.title}</span>
                    <span className="task-row__meta muted small" style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      {(hasTime || hasCheck) && (
                        <>
                          {hasTime && fmtDuration(secs)}
                          {showBullet && <span className="task-row__bullet"> • </span>}
                          {hasCheck && ck}
                        </>
                      )}
                      {runningId === s.id && <span className="running">●</span>}
                    </span>
                  </div>
                  <div className="task-row__line2">
                    <span className="task-row__status" style={{ color: col }}>{s.status}</span>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
        <div className="resizer" onMouseDown={subResize.onDown} />
      </div>

      {/* Колонка 3: контент — задача сверху, разделитель, подзадача снизу */}
      <div className="panel col" style={{ flex: 1, overflow: 'auto' }}>
        {!selTaskId && !selSubId && <p className="muted">Выберите задачу или подзадачу.</p>}

        {selTaskId && (() => {
          const t = tasks.find((x) => x.id === selTaskId)!
          return (
            <TaskDetail
              task={t}
              detail={taskDetail}
              statuses={statuses}
              onError={onError}
              onStatus={changeStatus}
              onDesc={(v) => api.updateTaskDescription(selTaskId!, v).catch((e) => onError(String(e)))}
              onLinkAdd={async (n, u) => { await api.createTaskLink(selTaskId!, n, u); loadTaskDetail(selTaskId!) }}
              onLinkDel={async (id) => { await api.deleteTaskLink(id); loadTaskDetail(selTaskId!) }}
              onNewSub={async (title) => { await api.createSubtask(selTaskId!, title); reload() }}
              onDel={() => delTask(t)}
              onTagAdd={async (typeId, text, url) => { await api.createTag(selTaskId!, typeId, text, url); loadTaskDetail(selTaskId!) }}
              onTagDel={async (id) => { await api.deleteTag(id); loadTaskDetail(selTaskId!) }}
            />
          )
        })()}

        {selTaskId && selSubId && <hr className="divider" />}

        {selSubId && (() => {
          const s = subs.find((x) => x.id === selSubId)!
          return (
            <SubDetail
              sub={s}
              detail={subDetail}
              statuses={statuses}
              running={runningId === selSubId}
              onError={onError}
              onStatus={changeStatus}
              onToggleTimer={toggleTimer}
              onDesc={(v) => api.updateSubtaskDescription(selSubId!, v).catch((e) => onError(String(e)))}
              onLinkAdd={async (n, u) => { await api.createSubtaskLink(selSubId!, n, u); loadSubDetail(selSubId!) }}
              onLinkDel={async (id) => { await api.deleteSubtaskLink(id); loadSubDetail(selSubId!) }}
              onTimeEdit={async (id, s2, e2) => { await api.updateTimeEntry(id, s2, e2); loadSubDetail(selSubId!) }}
              onTimeDel={async (id) => { await api.deleteTimeEntry(id); loadSubDetail(selSubId!) }}
              onJournalAdd={async (text) => { await api.createJournal(selSubId!, text); loadSubDetail(selSubId!) }}
              onCheckToggle={async (id, st) => { await api.setChecklistStatus(id, st); loadSubDetail(selSubId!) }}
              onCheckAdd={async (text) => { await api.createChecklistItem(selSubId!, text); loadSubDetail(selSubId!) }}
              onCheckDel={async (id) => { await api.deleteChecklistItem(id); loadSubDetail(selSubId!) }}
              onDel={() => delSub(s)}
            />
          )
        })()}
      </div>


      {confirmNode}
      {ghost && (() => {
        if (ghost.kind === 'task') {
          const t = tasks.find((x) => x.id === ghost.id)
          if (!t) return null
          const col = statusColor(t.status, statuses)
          const secs = taskTime(t.id)
          const hasTime = secs > 0
          const hasCount = t.sub_count > 0
          const k = taskDoneCount(t.id)
          const tags = tagsMap[String(t.id)] || []
          return createPortal(
            <div className="drag-ghost task-row" style={{ left: ghost.x, top: ghost.y, width: ghost.w, height: ghost.h, borderLeftColor: col }}>
              <div className="task-row__line1">
                <span className="title">{t.title}</span>
                {(hasTime || hasCount) && (
                  <span className="task-row__meta muted small">
                    {hasTime && fmtDuration(secs)}
                    {hasTime && hasCount && <span className="task-row__bullet"> • </span>}
                    {hasCount && `${k}/${t.sub_count}`}
                  </span>
                )}
              </div>
              <div className="task-row__line2">
                <span className="task-row__status" style={{ color: col }}>{t.status}</span>
                {tags.length > 0 && (
                  <span className="task-row__tags">
                    {tags.slice(0, 2).map((tg) => (
                      <span key={tg.id} className="tag task-row__tag" style={{ background: tg.color || 'var(--panel2)' }}>{tg.text}</span>
                    ))}
                  </span>
                )}
              </div>
            </div>,
            document.body,
          )
        } else {
          const s = subs.find((x) => x.id === ghost.id)
          if (!s) return null
          const col = statusColor(s.status, statuses)
          let secs = s.total_seconds
          if (s.active_since) secs += Math.floor(Date.now() / 1000) - s.active_since
          const hasTime = secs > 0
          const cc = checkMap[String(s.id)]
          const hasCheck = !!cc && cc[1] > 0
          const ck = cc ? `${cc[0]}/${cc[1]}` : ''
          return createPortal(
            <div className="drag-ghost task-row" style={{ left: ghost.x, top: ghost.y, width: ghost.w, height: ghost.h, borderLeftColor: col }}>
              <div className="task-row__line1">
                <span className="title">{s.title}</span>
                <span className="task-row__meta muted small" style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  {(hasTime || hasCheck) && (
                    <>
                      {hasTime && fmtDuration(secs)}
                      {hasTime && hasCheck && <span className="task-row__bullet"> • </span>}
                      {hasCheck && ck}
                    </>
                  )}
                </span>
              </div>
              <div className="task-row__line2">
                <span className="task-row__status" style={{ color: col }}>{s.status}</span>
              </div>
            </div>,
            document.body,
          )
        }
      })()}

      {addKind && (
        <Modal
          title={addKind === 'task' ? 'Новая задача' : 'Новая подзадача'}
          onClose={() => setAddKind(null)}
          footer={
            <>
              <Button variant="outline" onClick={() => setAddKind(null)}>Отмена</Button>
              <Button color="accent" icon="plus" disabled={!addTitle.trim()} onClick={submitAdd}>Создать</Button>
            </>
          }
        >
          <input
            autoFocus
            placeholder="Название"
            value={addTitle}
            style={{ width: '100%', minWidth: 360 }}
            onChange={(e) => setAddTitle(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') submitAdd() }}
          />
        </Modal>
      )}
    </div>
  )
}

function statusColor(name: string, statuses: StatusDef[]): string {
  const s = statuses.find((x) => x.name === name)
  return s?.color || 'var(--grey)'
}

// --- Детальные панели ---

function TaskDetail(props: {
  task: Task
  detail: Detail
  statuses: StatusDef[]
  onError: (m: string) => void
  onStatus: (to: string, note?: string) => void
  onDesc: (v: string) => void
  onLinkAdd: (n: string, u: string) => void
  onLinkDel: (id: number) => void
  onNewSub: (title: string) => void
  onDel: () => void
  onTagAdd: (typeId: number, text: string, url: string) => void
  onTagDel: (id: number) => void
}) {
  const [desc, setDesc] = useState(props.detail.description)
  useEffect(() => setDesc(props.detail.description), [props.detail.description])
  const [st, setSt] = useState('')
  const [nt, setNt] = useState('')
  const [subTitle, setSubTitle] = useState('')

  return (
    <div className="col">
      <div className="flex between">
        <h2 style={{ margin: 0 }}>{props.task.title}</h2>
        <Button color="danger" variant="outline" icon="trash" label="Удалить задачу" onClick={props.onDel} />
      </div>
      <div>
        <div className="flex between"><strong>Описание</strong><Button color="accent" icon="save" onClick={() => props.onDesc(desc)}>Сохранить</Button></div>
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)} />
      </div>
      <div className="flex" style={{ gap: 8 }}>
        <select value={st} onChange={(e) => setSt(e.target.value)}>
          <option value="">Сменить статус…</option>
          {props.statuses.map((s) => (<option key={s.id} value={s.name}>{s.name}</option>))}
        </select>
        {st && <input placeholder="Заметка" value={nt} onChange={(e) => setNt(e.target.value)} />}
        <Button color="accent" icon="check" disabled={!st} onClick={() => { props.onStatus(st, nt); setSt(''); setNt('') }}>Применить</Button>
      </div>
      <div>
        <div className="flex between"><strong>Подзадачи</strong></div>
        <div className="flex">
          <input placeholder="Название подзадачи" value={subTitle} onChange={(e) => setSubTitle(e.target.value)} />
          <Button color="accent" icon="plus" label="Добавить подзадачу" disabled={!subTitle.trim()} onClick={() => { props.onNewSub(subTitle.trim()); setSubTitle('') }} />
        </div>
      </div>
      <LinksBlock links={props.detail.links} onAdd={props.onLinkAdd} onDel={props.onLinkDel} />
      <TagsBlock tags={props.detail.tags} onAdd={props.onTagAdd} onDel={props.onTagDel} />
    </div>
  )
}

function SubDetail(props: {
  sub: Subtask
  detail: Detail
  statuses: StatusDef[]
  running: boolean
  onError: (m: string) => void
  onStatus: (to: string, note?: string) => void
  onToggleTimer: () => void
  onDesc: (v: string) => void
  onLinkAdd: (n: string, u: string) => void
  onLinkDel: (id: number) => void
  onTimeEdit: (id: number, started: string, ended: string | null) => void
  onTimeDel: (id: number) => void
  onJournalAdd: (text: string) => void
  onCheckToggle: (id: number, status: string) => void
  onCheckAdd: (text: string) => void
  onCheckDel: (id: number) => void
  onDel?: () => void
}) {
  const [desc, setDesc] = useState(props.detail.description)
  useEffect(() => setDesc(props.detail.description), [props.detail.description])
  const [st, setSt] = useState('')
  const [nt, setNt] = useState('')
  const [jt, setJt] = useState('')
  const [ct, setCt] = useState('')

  const checkStates = ['new', 'in_progress', 'done', 'cancelled']

  return (
    <div className="col">
      <div className="flex between">
        <h2 style={{ margin: 0 }}>{props.sub.title}</h2>
        <div className="flex" style={{ gap: 8 }}>
          {props.onDel && <Button color="danger" variant="outline" icon="trash" label="Удалить подзадачу" onClick={props.onDel} />}
          <Button color={props.running ? 'danger' : 'success'} icon={props.running ? 'pause' : 'play'} onClick={props.onToggleTimer}>
            {props.running ? 'Стоп' : 'Старт'}
          </Button>
        </div>
      </div>
      <div className="flex" style={{ gap: 8 }}>
        <select value={st} onChange={(e) => setSt(e.target.value)}>
          <option value="">Сменить статус…</option>
          {props.statuses.map((s) => (<option key={s.id} value={s.name}>{s.name}</option>))}
        </select>
        {st && <input placeholder="Заметка" value={nt} onChange={(e) => setNt(e.target.value)} />}
        <Button color="accent" icon="check" disabled={!st} onClick={() => { props.onStatus(st, nt); setSt(''); setNt('') }}>Применить</Button>
      </div>
      <div>
        <div className="flex between"><strong>Описание</strong><Button color="accent" icon="save" onClick={() => props.onDesc(desc)}>Сохранить</Button></div>
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)} />
      </div>

      <div>
        <strong>Записи времени</strong>
        <ul className="list">
          {props.detail.time.map((t: TimeEntry) => (
            <li key={t.id} className="row">
              <span className="title">{fmtDateTime(t.started_at)} — {t.ended_at ? fmtDateTime(t.ended_at) : '…'}</span>
              <Button icon="edit" label="Редактировать" onClick={() => {
                const s = prompt('Начало (ISO)', t.started_at)
                if (!s) return
                const e = prompt('Конец (ISO, пусто = открыто)', t.ended_at ?? '')
                props.onTimeEdit(t.id, s, e || null)
              }} />
              <Button color="danger" variant="outline" icon="trash" label="Удалить" onClick={() => props.onTimeDel(t.id)} />
            </li>
          ))}
        </ul>
      </div>

      <div>
        <strong>Журнал</strong>
        <div className="flex">
          <input placeholder="Новая запись" value={jt} onChange={(e) => setJt(e.target.value)} />
          <Button color="accent" icon="plus" label="Добавить запись" disabled={!jt.trim()} onClick={() => { props.onJournalAdd(jt.trim()); setJt('') }} />
        </div>
        <ul className="list">
          {props.detail.journal.map((j: JournalEntry) => (
            <li key={j.id} className="row"><span className="title">{j.text}</span><span className="muted small">{fmtDateTime(j.created_at)}</span></li>
          ))}
        </ul>
      </div>

      <div>
        <strong>Чек-лист</strong>
        <div className="flex">
          <input placeholder="Пункт" value={ct} onChange={(e) => setCt(e.target.value)} />
          <Button color="accent" icon="plus" label="Добавить пункт" disabled={!ct.trim()} onClick={() => { props.onCheckAdd(ct.trim()); setCt('') }} />
        </div>
        <ul className="list">
          {props.detail.checklist.map((c: ChecklistItem) => (
            <li key={c.id} className="row">
              <select value={c.status} onChange={(e) => props.onCheckToggle(c.id, e.target.value)}>
                {checkStates.map((s) => (<option key={s} value={s}>{s}</option>))}
              </select>
              <span className="title">{c.text}</span>
              <Button color="danger" variant="outline" icon="trash" label="Удалить" onClick={() => props.onCheckDel(c.id)} />
            </li>
          ))}
        </ul>
      </div>

      <LinksBlock links={props.detail.links} onAdd={props.onLinkAdd} onDel={props.onLinkDel} />
    </div>
  )
}

function LinksBlock({ links, onAdd, onDel }: { links: Link[]; onAdd: (n: string, u: string) => void; onDel: (id: number) => void }) {
  const [ln, setLn] = useState({ name: '', url: '' })
  return (
    <div>
      <div className="flex between"><strong>Ссылки</strong></div>
      <div className="flex">
        <input placeholder="Название" value={ln.name} onChange={(e) => setLn({ ...ln, name: e.target.value })} />
        <input placeholder="URL" value={ln.url} onChange={(e) => setLn({ ...ln, url: e.target.value })} />
        <Button color="accent" icon="plus" label="Добавить ссылку" disabled={!ln.url.trim()} onClick={() => { onAdd(ln.name, ln.url); setLn({ name: '', url: '' }) }} />
      </div>
      <ul className="list">
        {links.map((l) => (
          <li key={l.id} className="row">
            <a href={l.url} target="_blank" rel="noreferrer">{l.name || l.url}</a>
            <Button color="danger" variant="outline" icon="trash" label="Удалить ссылку" onClick={() => onDel(l.id)} />
          </li>
        ))}
      </ul>
    </div>
  )
}

function TagsBlock({ tags, onAdd, onDel }: { tags: Tag[]; onAdd: (typeId: number, text: string, url: string) => void; onDel: (id: number) => void }) {
  const [t, setT] = useState({ typeId: 0, text: '', url: '' })
  return (
    <div>
      <div className="flex between"><strong>Теги</strong></div>
      <div className="flex">
        <input type="number" placeholder="type_id" value={t.typeId || ''} onChange={(e) => setT({ ...t, typeId: Number(e.target.value) })} />
        <input placeholder="Текст" value={t.text} onChange={(e) => setT({ ...t, text: e.target.value })} />
        <input placeholder="URL" value={t.url} onChange={(e) => setT({ ...t, url: e.target.value })} />
        <Button color="accent" icon="plus" label="Добавить тег" disabled={!t.typeId || !t.text.trim()} onClick={() => { onAdd(t.typeId, t.text, t.url); setT({ typeId: 0, text: '', url: '' }) }} />
      </div>
      <div>
        {tags.map((tg) => (
          <span key={tg.id} className="tag" style={{ background: tg.color || 'var(--panel2)' }}>
            {tg.type_name}: {tg.text}
            <span style={{ cursor: 'pointer' }} onClick={() => onDel(tg.id)}> ✕</span>
          </span>
        ))}
      </div>
    </div>
  )
}
