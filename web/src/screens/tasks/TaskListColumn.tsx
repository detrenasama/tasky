import type { Project, Task, Tag, StatusDef } from '../../types'
import { fmtDuration } from '../../fmt'
import { Button } from '../../ui'
import { statusColor, taskTime, taskDoneCount, isDone } from './helpers'

export function TaskListColumn(props: {
  projects: Project[]
  proj: Project | null
  tasks: any[]
  statuses: StatusDef[]
  tagsMap: Record<string, Tag[]>
  subs: any[]
  selTaskId: number | null
  dragTask: any
  dragSub: any
  ghost: any
  taskListRef: React.RefObject<HTMLUListElement>
  suppressClick: React.MutableRefObject<boolean>
  onProjectChange: (p: Project) => void
  onAddTask: () => void
  onSelectTask: (t: Task) => void
  onTaskPointerDown: (e: React.PointerEvent, t: Task, idx: number) => void
  taskResize: { width: number; onDown: (e: React.MouseEvent) => void }
}) {
  const { projects, proj, tasks, statuses, tagsMap, subs, selTaskId, dragTask, dragSub, ghost, taskListRef, suppressClick, onProjectChange, onAddTask, onSelectTask, onTaskPointerDown, taskResize } = props
  return (
    <div className="panel col" style={{ width: taskResize.width, position: 'relative', overflow: 'auto' }}>
      <select value={proj?.id ?? ''} onChange={(e) => {
        const p = projects.find((x) => x.id === Number(e.target.value))
        if (p) onProjectChange(p)
      }}>
        {projects.map((p) => (
          <option key={p.id} value={p.id}>{p.name}</option>
        ))}
      </select>
      <div className="toolbar">
        {proj && <Button color="accent" icon="plus" label="Создать задачу" onClick={onAddTask} />}
      </div>
      {proj ? (
        <ul className={`list${dragTask ? ' dragging' : ''}`} ref={taskListRef}>
          {tasks.map((t: any, idx: number) => {
            if (t.__placeholder) {
              return <li key="ph-task" className="drag-placeholder" style={{ height: ghost?.h ? `${ghost.h}px` : '58px' }} />
            }
            const col = statusColor(t.status, statuses)
            const done = isDone(t.status, new Set(statuses.filter(s=>s.type==='done').map(s=>s.name)))
            const secs = taskTime(t.id, subs)
            const hasTime = secs > 0
            const hasCount = t.sub_count > 0
            const k = taskDoneCount(t.id, subs, new Set(statuses.filter(s=>s.type==='done').map(s=>s.name)))
            const tags = tagsMap[String(t.id)] || []
            const isDropTarget = dragSub?.toTaskId === t.id && dragSub?.id !== t.id
            return (
              <li
                key={t.id}
                data-task-id={t.id}
                className={`row task-row${selTaskId === t.id ? ' selected' : ''}${done ? ' task-row--done' : ''}${isDropTarget ? ' drop-target' : ''}`}
                style={{ borderLeftColor: col }}
                onClick={() => { if (suppressClick.current) return; onSelectTask(t) }}
                onPointerDown={(e) => onTaskPointerDown(e, t, idx)}
              >
                <div className="task-row__line1">
                  <span className="title">{t.title}</span>
                  {(hasTime || hasCount) && (
                    <span className="task-row__meta muted small">
                      {hasTime && fmtDuration(secs)}
                      {hasTime && hasCount && <span className="task-row__bullet"> • </span>}
                      {hasCount && `[${k}/${t.sub_count}]`}
                    </span>
                  )}
                </div>
                <div className="task-row__line2">
                  <span className="task-row__status" style={{ color: col }}>{t.status}</span>
                  {tags.length > 0 && (
                    <span className="task-row__tags">
                      {tags.slice(0, 3).map((tg) => (
                        <Button key={tg.id} variant="outline" tabIndex={-1} style={{ borderColor: tg.color || 'var(--border)', color: tg.color || 'var(--text)', fontSize: '11px', padding: '1px 6px', cursor: 'default', pointerEvents: 'none' }}>{tg.text}</Button>
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
  )
}
