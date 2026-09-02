import { createPortal } from 'react-dom'
import { fmtDuration } from '../../fmt'
import { Button } from '../../ui'
import { statusColor, taskTime, taskDoneCount } from './helpers'

export function TaskGhost({ ghost, tasks, subs, tagsMap, checkMap, statuses }: any) {
  if (!ghost) return null
  if (ghost.kind === 'task') {
    const t = tasks.find((x: any) => x.id === ghost.id)
    if (!t) return null
    const col = statusColor(t.status, statuses)
    const secs = taskTime(t.id, subs)
    const hasTime = secs > 0
    const hasCount = t.sub_count > 0
    const doneSet: Set<string> = new Set((statuses as any[]).filter((s:any)=>s.type==='done').map((s:any)=>s.name))
    const k = taskDoneCount(t.id, subs, doneSet)
    const tags = tagsMap[String(t.id)] || []
    return createPortal(
      <div className="drag-ghost task-row" style={{ left: ghost.x, top: ghost.y, width: ghost.w, height: ghost.h, borderLeftColor: col }}>
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
              {tags.slice(0, 2).map((tg:any) => (
                <Button key={tg.id} variant="outline" tabIndex={-1} style={{ borderColor: tg.color || 'var(--border)', color: tg.color || 'var(--text)', fontSize: '11px', padding: '1px 6px', cursor: 'default', pointerEvents: 'none' }}>{tg.text}</Button>
              ))}
            </span>
          )}
        </div>
      </div>,
      document.body,
    )
  } else {
    const s = subs.find((x: any) => x.id === ghost.id)
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
}
