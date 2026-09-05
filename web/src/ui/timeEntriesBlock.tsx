import { useMemo, useState } from 'react'
import type { TimeEntry } from '../types'
import { fmtDateTime } from '../fmt'
import { Button } from './button'
import { Modal } from '../ui'

function sortedEntries(entries: TimeEntry[]): TimeEntry[] {
  return [...entries].sort((a, b) => {
    const da = new Date(a.started_at).getTime()
    const db = new Date(b.started_at).getTime()
    if (db !== da) return db - da
    return b.id - a.id
  })
}

function pad2(n: number): string {
  return String(n).padStart(2, '0')
}

function isoToLocalInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const m = pad2(d.getMonth() + 1)
  const day = pad2(d.getDate())
  const hh = pad2(d.getHours())
  const mm = pad2(d.getMinutes())
  return `${y}-${m}-${day}T${hh}:${mm}`
}

function localInputToRFC3339(v: string): string {
  // datetime-local без таймзоны — интерпретируем как локальное время
  return new Date(v).toISOString()
}

export function TimeEntriesBlock({
  entries,
  onEdit,
  onDel,
  subtaskTitle,
}: {
  entries: TimeEntry[]
  onEdit: (id: number, started: string, ended: string | null) => void
  onDel: (id: number) => void
  subtaskTitle: string
}) {
  const [open, setOpen] = useState(false)
  const sorted = useMemo(() => sortedEntries(entries), [entries])

  return (
    <div>
      <div className="flex between">
        <strong>Записи времени</strong>
        <Button variant="outline" icon="edit" label="Редактировать" disabled={entries.length === 0} onClick={() => setOpen(true)} />
      </div>

      {entries.length === 0 ? (
        <span className="muted small" style={{ display: 'inline-block', marginTop: 8 }}>
          Записей нет
        </span>
      ) : (
        <ul className="list list--dense" style={{ marginTop: 8 }}>
          {sorted.map((t) => (
            <li key={t.id} className="row">
              <span className="title">
                {fmtDateTime(t.started_at)} — {t.ended_at ? fmtDateTime(t.ended_at) : '…'}
              </span>
            </li>
          ))}
        </ul>
      )}

      {open && (
        <TimeEditModal
          entries={sorted}
          subtaskTitle={subtaskTitle}
          onEdit={onEdit}
          onDel={onDel}
          onClose={() => setOpen(false)}
        />
      )}
    </div>
  )
}

function TimeEditModal({
  entries,
  subtaskTitle,
  onEdit,
  onDel,
  onClose,
}: {
  entries: TimeEntry[]
  subtaskTitle: string
  onEdit: (id: number, started: string, ended: string | null) => void
  onDel: (id: number) => void
  onClose: () => void
}) {
  const [editing, setEditing] = useState<{ id: number; field: 'started_at' | 'ended_at'; draft: string } | null>(null)
  const [confirmId, setConfirmId] = useState<number | null>(null)

  const handleSave = async () => {
    if (!editing) return
    const row = entries.find((e) => e.id === editing.id)
    if (!row) return
    const raw = editing.draft.trim()
    // Пустое значение для ended_at означает открытую сессию (null)
    // Для started_at пустое не допускаем
    if (editing.field === 'started_at' && !raw) return
    const newStarted = editing.field === 'started_at' ? localInputToRFC3339(raw) : row.started_at
    let newEnded: string | null
    if (editing.field === 'ended_at') {
      newEnded = raw ? localInputToRFC3339(raw) : null
    } else {
      newEnded = row.ended_at
    }
    setEditing(null)
    onEdit(row.id, newStarted, newEnded)
  }

  const cancelEditing = () => setEditing(null)

  return (
    <>
      <Modal title={`Редактирование записей времени задачи "${subtaskTitle}"`} onClose={onClose} wide showClose>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: 8, marginBottom: 8 }}>
          <strong>Начало</strong>
          <strong>Конец</strong>
          <span />
        </div>

        <div className="list list--dense" style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          {entries.map((t) => (
            <div
              key={t.id}
              className="row"
              style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: 8, alignItems: 'center' }}
            >
              {/* Начало */}
              {editing?.id === t.id && editing.field === 'started_at' ? (
                <span className="flex" style={{ gap: 4 }}>
                  <input
                    type="datetime-local"
                    autoFocus
                    value={editing.draft}
                    onChange={(e) => setEditing({ ...editing, draft: e.target.value })}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleSave()
                      if (e.key === 'Escape') cancelEditing()
                    }}
                    style={{ flex: 1, padding: '4px 6px', fontSize: 13 }}
                  />
                  <Button color="accent" icon="save" label="Сохранить" disabled={!editing.draft.trim()} onClick={handleSave} />
                  <Button variant="outline" onClick={cancelEditing}>
                    Отмена
                  </Button>
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => setEditing({ id: t.id, field: 'started_at', draft: isoToLocalInput(t.started_at) })}
                  style={{
                    textAlign: 'left',
                    background: 'none',
                    border: '1px dashed var(--border)',
                    borderRadius: 4,
                    padding: '4px 6px',
                    cursor: 'pointer',
                    font: 'inherit',
                    fontSize: 13,
                  }}
                >
                  {fmtDateTime(t.started_at)}
                </button>
              )}

              {/* Конец */}
              {editing?.id === t.id && editing.field === 'ended_at' ? (
                <span className="flex" style={{ gap: 4 }}>
                  <input
                    type="datetime-local"
                    autoFocus
                    value={editing.draft}
                    onChange={(e) => setEditing({ ...editing, draft: e.target.value })}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleSave()
                      if (e.key === 'Escape') cancelEditing()
                    }}
                    style={{ flex: 1, padding: '4px 6px', fontSize: 13 }}
                  />
                  <Button color="accent" icon="save" label="Сохранить" onClick={handleSave} />
                  <Button variant="outline" onClick={cancelEditing}>
                    Отмена
                  </Button>
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => setEditing({ id: t.id, field: 'ended_at', draft: isoToLocalInput(t.ended_at) })}
                  style={{
                    textAlign: 'left',
                    background: 'none',
                    border: '1px dashed var(--border)',
                    borderRadius: 4,
                    padding: '4px 6px',
                    cursor: 'pointer',
                    font: 'inherit',
                    fontSize: 13,
                  }}
                >
                  {t.ended_at ? fmtDateTime(t.ended_at) : '…'}
                </button>
              )}

              <Button color="danger" variant="outline" icon="trash" label="Удалить" onClick={() => setConfirmId(t.id)} />
            </div>
          ))}
        </div>
      </Modal>

      {confirmId !== null && (
        <Modal
          title="Удалить запись"
          onClose={() => setConfirmId(null)}
          footer={
            <>
              <Button variant="outline" onClick={() => setConfirmId(null)}>
                Отмена
              </Button>
              <Button
                color="danger"
                onClick={() => {
                  const id = confirmId
                  setConfirmId(null)
                  onDel(id)
                }}
              >
                Удалить
              </Button>
            </>
          }
        >
          <p>Удалить запись времени?</p>
        </Modal>
      )}
    </>
  )
}
