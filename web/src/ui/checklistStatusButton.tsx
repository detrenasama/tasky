import { useEffect, useRef, useState } from 'react'
import { Icon, IconName } from './icons'
import '../ui/statusButton.css'

type Cfg = { label: string; color: string; icon: IconName }

const CFG: Record<string, Cfg> = {
  new:         { label: 'Новый',     color: '#d7ba7d', icon: 'square' },
  in_progress: { label: 'В работе',  color: '#569cd6', icon: 'clock' },
  done:        { label: 'Выполнено', color: '#6a9955', icon: 'checkSquare' },
  cancelled:   { label: 'Отменён',   color: '#8a8a8a', icon: 'xSquare' },
}

export function ChecklistStatusButton({
  value,
  onSelect,
}: {
  value: string
  onSelect: (to: string) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const cur = CFG[value] ?? CFG.new

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
        type="button"
        aria-label={cur.label}
        onClick={() => setOpen((v) => !v)}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: 30,
          height: 30,
          borderRadius: 6,
          border: `1.5px solid ${cur.color}`,
          background: '#fff',
          color: cur.color,
          cursor: 'pointer',
          flexShrink: 0,
        }}
      >
        <Icon name={cur.icon} size={16} />
      </button>

      {open && (
        <div className="status-dropdown">
          {(Object.entries(CFG) as [string, Cfg][]).map(([key, cfg]) => (
            <div
              key={key}
              className={`status-option${key === value ? ' status-option--active' : ''}`}
              onClick={() => {
                setOpen(false)
                if (key !== value) onSelect(key)
              }}
            >
              <span style={{ color: cfg.color, display: 'inline-flex' }}>
                <Icon name={cfg.icon} size={16} />
              </span>
              <span>{cfg.label}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
