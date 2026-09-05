import { useState } from 'react'
import type { JournalEntry } from '../types'
import { fmtDateTime } from '../fmt'
import { Button } from './button'

export function JournalBlock({
  entries,
  onAdd,
}: {
  entries: JournalEntry[]
  onAdd: (text: string) => void
}) {
  const [draft, setDraft] = useState('')

  const handleAdd = () => {
    const t = draft.trim()
    if (!t) return
    onAdd(t)
    setDraft('')
  }

  return (
    <div>
      <div className="flex between">
        <strong>Журнал</strong>
        <Button variant="outline" icon="plus" label="Добавить запись" disabled={!draft.trim()} onClick={handleAdd} />
      </div>

      <div className="flex" style={{ marginTop: 8 }}>
        <input
          placeholder="Новая запись"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && draft.trim()) handleAdd()
          }}
          style={{ flex: 1 }}
        />
      </div>

      {entries.length === 0 ? (
        <span className="muted small" style={{ display: 'inline-block', marginTop: 8 }}>
          Записей нет
        </span>
      ) : (
        <ul className="list" style={{ marginTop: 8 }}>
          {entries.map((j) => (
            <li key={j.id} className="row">
              <span className="title">{j.text}</span>
              <span className="muted small">{fmtDateTime(j.created_at)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
