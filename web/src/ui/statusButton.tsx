import { useEffect, useRef, useState } from 'react'
import type { StatusDef } from '../types'

function statusColor(name: string, statuses: StatusDef[]): string {
  const s = statuses.find((x) => x.name === name)
  return s?.color || 'var(--grey)'
}

export function StatusButton({
  value,
  statuses,
  onSelect,
}: {
  value: string
  statuses: StatusDef[]
  onSelect: (to: string) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const col = statusColor(value, statuses)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    window.addEventListener('mousedown', onDown)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('mousedown', onDown)
      window.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <button
        className="status-btn"
        style={{ background: col, color: '#fff' }}
        onClick={() => setOpen((v) => !v)}
      >
        {value || 'Статус'}
      </button>
      {open && (
        <div className="status-dropdown">
          {statuses.map((s) => (
            <div
              key={s.id}
              className={`status-option${s.name === value ? ' status-option--active' : ''}`}
              onClick={() => {
                setOpen(false)
                if (s.name !== value) onSelect(s.name)
                else setOpen(false)
              }}
            >
              <span className="status-dot" style={{ background: s.color }} />
              <span>{s.name}</span>
              {s.note_prompt && <span className="muted small" style={{ marginLeft: 'auto' }}>…</span>}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
