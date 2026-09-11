import { useState } from 'react'
import type { ChecklistItem } from '../types'
import { Button } from './button'
import { Modal } from '../ui'
import { useDirtyConfirm } from '../hooks/useDirtyConfirm'
import { ChecklistStatusButton } from './checklistStatusButton'
import '../ui/statusButton.css'

function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
}

function isVisibleChecklist(c: ChecklistItem, now: Date): boolean {
  if (c.status !== 'done' && c.status !== 'cancelled') return true
  const d = new Date(c.status_changed_at)
  if (isNaN(d.getTime())) return true
  if (sameDay(d, now)) return true
  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (sameDay(d, yesterday)) return true
  // будущее — видимое
  const endToday = new Date(now)
  endToday.setHours(23, 59, 59, 999)
  if (d.getTime() > endToday.getTime()) return true
  return false
}

export function ChecklistBlock({
  items,
  onToggle,
  onAdd,
  onDel,
  onEdit,
}: {
  items: ChecklistItem[]
  onToggle: (id: number, status: string) => void
  onAdd: (text: string) => void
  onDel: (id: number) => void
  onEdit: (id: number, text: string) => void
}) {
  const [draft, setDraft] = useState('')
  const [editing, setEditing] = useState<ChecklistItem | null>(null)
  const [editDraft, setEditDraft] = useState('')

  const handleAdd = () => {
    const t = draft.trim()
    if (!t) return
    onAdd(t)
    setDraft('')
  }

  const openEdit = (c: ChecklistItem) => {
    setEditing(c)
    setEditDraft(c.text)
  }

  const closeEdit = () => setEditing(null)

  const dirty = editing ? editDraft !== editing.text : false
  const { confirmDiscard, tryClose, doDiscard, cancelDiscard } = useDirtyConfirm({
    dirty,
    onClose: closeEdit,
  })

  const handleSave = () => {
    const t = editDraft.trim()
    if (!t || !editing) return
    onEdit(editing.id, t)
    closeEdit()
  }

  return (
    <div>
      <div className="flex between">
        <strong>Чек-лист</strong>
        <Button variant="outline" icon="plus" label="Добавить пункт" disabled={!draft.trim()} onClick={handleAdd} />
      </div>

      <div className="flex" style={{ marginTop: 8 }}>
        <input
          placeholder="Пункт"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && draft.trim()) handleAdd()
          }}
          style={{ flex: 1 }}
        />
      </div>

      {(() => {
        if (items.length === 0) {
          return (
            <span className="muted small" style={{ display: 'inline-block', marginTop: 8 }}>
              Пунктов нет
            </span>
          )
        }
        const now = new Date()
        const visible = items.filter((c) => isVisibleChecklist(c, now))
        if (visible.length === 0) {
          return (
            <span className="muted small" style={{ display: 'inline-block', marginTop: 8 }}>
              нет активных
            </span>
          )
        }
        return (
          <ul className="list list--dense" style={{ marginTop: 8 }}>
            {visible.map((c) => (
              <li key={c.id} className="row">
                <ChecklistStatusButton value={c.status} onSelect={(s) => onToggle(c.id, s)} />
                <span className="title">{c.text}</span>
                <span className="flex" style={{ gap: 4, flexShrink: 0 }}>
                  <Button variant="outline" icon="edit" label="Редактировать" onClick={() => openEdit(c)} />
                  <Button color="danger" variant="outline" icon="trash" label="Удалить" onClick={() => onDel(c.id)} />
                </span>
              </li>
            ))}
          </ul>
        )
      })()}

      {editing && (
        <Modal
          title="Редактировать пункт"
          onClose={tryClose}
          showClose
          footer={
            <>
              <Button variant="outline" onClick={tryClose}>
                Отменить
              </Button>
              <Button color="accent" icon="save" disabled={!editDraft.trim() || !dirty} onClick={handleSave}>
                Сохранить
              </Button>
            </>
          }
        >
          <input
            autoFocus
            value={editDraft}
            onChange={(e) => setEditDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && editDraft.trim()) handleSave()
            }}
            style={{ width: '100%' }}
            placeholder="Текст пункта"
          />
        </Modal>
      )}

      {confirmDiscard && (
        <Modal
          title="Несохранённые изменения"
          onClose={cancelDiscard}
          footer={
            <>
              <Button variant="outline" onClick={cancelDiscard}>
                Нет
              </Button>
              <Button color="danger" onClick={doDiscard}>
                Да, закрыть
              </Button>
            </>
          }
        >
          <p>Изменения не будут сохранены. Закрыть без сохранения?</p>
        </Modal>
      )}
    </div>
  )
}
