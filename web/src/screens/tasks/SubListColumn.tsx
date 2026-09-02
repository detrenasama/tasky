import type { StatusDef } from '../../types'
import { fmtDuration } from '../../fmt'
import { Button } from '../../ui'
import { statusColor, isDone } from './helpers'

export function SubListColumn(props: {
  selTaskId: number | null
  subs: any[]
  displaySubsFor: (id: number) => any[]
  statuses: StatusDef[]
  checkMap: Record<string, [number, number]>
  runningId: number | null
  selSubId: number | null
  dragSub: any
  ghost: any
  subListRef: React.RefObject<HTMLUListElement>
  suppressClick: React.MutableRefObject<boolean>
  onAddSub: () => void
  onSelectSub: (s: any) => void
  onSubPointerDown: (e: React.PointerEvent, s: any, idx: number) => void
  subResize: { width: number; onDown: (e: React.MouseEvent) => void }
}) {
  const { selTaskId, displaySubsFor, statuses, checkMap, runningId, selSubId, dragSub, ghost, subListRef, suppressClick, onAddSub, onSelectSub, onSubPointerDown, subResize } = props
  return (
    <div className="panel col" style={{ width: subResize.width, position: 'relative', overflow: 'auto' }}>
      <div className="flex between" style={{ marginBottom: 8 }}>
        <strong className="col-head" style={{ margin: 0 }}>Подзадачи</strong>
        {selTaskId && <Button color="accent" icon="plus" label="Создать подзадачу" onClick={onAddSub} />}
      </div>
      {!selTaskId && <p className="muted">Выберите задачу.</p>}
      {selTaskId && (
        <ul className={`list${dragSub ? ' dragging' : ''}`} ref={subListRef}>
          {displaySubsFor(selTaskId).map((s: any, sIdx: number) => {
            if (s.__placeholder) {
              return <li key="ph-sub" className="drag-placeholder" style={{ height: ghost?.h ? `${ghost.h}px` : '58px' }} />
            }
            const col = statusColor(s.status, statuses)
            const done = isDone(s.status, new Set(statuses.filter(ss=>ss.type==='done').map(ss=>ss.name)))
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
                onClick={() => { if (suppressClick.current) return; onSelectSub(s) }}
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
  )
}
