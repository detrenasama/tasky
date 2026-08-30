import { useEffect, useState } from 'react'
import { api } from '../api'
import type { StatusDef, TagType } from '../types'
import { Button, Modal, useConfirm } from '../ui'

export default function Settings({ onError }: { onError: (m: string) => void }) {
  const [statuses, setStatuses] = useState<StatusDef[]>([])
  const [tagTypes, setTagTypes] = useState<TagType[]>([])
  const [hideDays, setHideDays] = useState('7')
  const [editingStatus, setEditingStatus] = useState<StatusDef | null>(null)
  const [editingTag, setEditingTag] = useState<TagType | null>(null)
  const confirm = useConfirm()[0]

  const load = () => {
    api.statuses().then(setStatuses).catch((e) => onError(String(e)))
    api.tagTypes().then(setTagTypes).catch((e) => onError(String(e)))
    api.getSetting('hide_days').then((s) => s.value && setHideDays(s.value)).catch(() => {})
  }
  useEffect(load, [])

  const saveHide = async () => {
    try {
      await api.setSetting('hide_days', hideDays)
    } catch (e) {
      onError(String(e))
    }
  }

  const saveStatus = async () => {
    if (!editingStatus) return
    const s = editingStatus
    try {
      if (s.id) await api.updateStatus(s.id, s)
      else await api.createStatus(s)
      setEditingStatus(null)
      load()
    } catch (e) {
      onError(String(e))
    }
  }

  const delStatus = async (s: StatusDef) => {
    if (!(await confirm(`Удалить статус «${s.name}»?`))) return
    try {
      await api.deleteStatus(s.id)
      load()
    } catch (e) {
      onError(String(e))
    }
  }

  const saveTag = async () => {
    if (!editingTag) return
    const t = editingTag
    try {
      if (t.id) await api.updateTagType(t.id, t)
      else await api.createTagType(t)
      setEditingTag(null)
      load()
    } catch (e) {
      onError(String(e))
    }
  }

  const delTag = async (t: TagType) => {
    if (!(await confirm(`Удалить тип тега «${t.name}»?`))) return
    try {
      await api.deleteTagType(t.id)
      load()
    } catch (e) {
      onError(String(e))
    }
  }

  return (
    <div className="main" style={{ display: 'block' }}>
      <div className="panel col" style={{ marginBottom: 12 }}>
        <strong>Скрытие завершённых (дней)</strong>
        <div className="flex">
          <input value={hideDays} onChange={(e) => setHideDays(e.target.value)} style={{ width: 80 }} />
          <Button color="accent" icon="save" onClick={saveHide}>Сохранить</Button>
          <span className="muted small">0 — выкл</span>
        </div>
      </div>

      <div className="panel col" style={{ marginBottom: 12 }}>
        <div className="flex between">
          <strong>Статусы</strong>
          <Button color="accent" icon="plus" label="Добавить статус" onClick={() => setEditingStatus({ id: 0, name: '', type: 'new', color: '#89b4fa', note_prompt: '', is_quick: false, sort_order: 0 })} />
        </div>
        <ul className="list">
          {statuses.map((s) => (
            <li key={s.id} className="row">
              <span className="status-dot" style={{ background: s.color }} />
              <span className="title">{s.name} <span className="muted small">({s.type}{s.is_quick ? ', быстрый' : ''})</span></span>
              <Button icon="edit" label="Редактировать" onClick={() => setEditingStatus({ ...s })} />
              <Button color="danger" variant="outline" icon="trash" label="Удалить" onClick={() => delStatus(s)} />
            </li>
          ))}
        </ul>
      </div>

      <div className="panel col">
        <div className="flex between">
          <strong>Типы тегов</strong>
          <Button color="accent" icon="plus" label="Добавить тип" onClick={() => setEditingTag({ id: 0, name: '', kind: 'text', color: '#a6e3a1', sort_order: 0 })} />
        </div>
        <ul className="list">
          {tagTypes.map((t) => (
            <li key={t.id} className="row">
              <span className="status-dot" style={{ background: t.color }} />
              <span className="title">{t.name} <span className="muted small">({t.kind})</span></span>
              <Button icon="edit" label="Редактировать" onClick={() => setEditingTag({ ...t })} />
              <Button color="danger" variant="outline" icon="trash" label="Удалить" onClick={() => delTag(t)} />
            </li>
          ))}
        </ul>
      </div>

      {editingStatus && (
        <Modal title={editingStatus.id ? 'Статус' : 'Новый статус'} onClose={() => setEditingStatus(null)}
          footer={<><Button variant="outline" onClick={() => setEditingStatus(null)}>Отмена</Button><Button color="accent" icon="save" onClick={saveStatus}>Сохранить</Button></>}>
          <div className="col">
            <input placeholder="Название" value={editingStatus.name} onChange={(e) => setEditingStatus({ ...editingStatus, name: e.target.value })} />
            <select value={editingStatus.type} onChange={(e) => setEditingStatus({ ...editingStatus, type: e.target.value })}>
              <option value="new">new</option>
              <option value="in_progress">in_progress</option>
              <option value="done">done</option>
            </select>
            <input placeholder="Цвет (#rrggbb)" value={editingStatus.color} onChange={(e) => setEditingStatus({ ...editingStatus, color: e.target.value })} />
            <input placeholder="Подсказка заметки" value={editingStatus.note_prompt} onChange={(e) => setEditingStatus({ ...editingStatus, note_prompt: e.target.value })} />
            <label className="flex"><input type="checkbox" checked={editingStatus.is_quick} onChange={(e) => setEditingStatus({ ...editingStatus, is_quick: e.target.checked })} /> Быстрая цепочка</label>
          </div>
        </Modal>
      )}

      {editingTag && (
        <Modal title={editingTag.id ? 'Тип тега' : 'Новый тип тега'} onClose={() => setEditingTag(null)}
          footer={<><Button variant="outline" onClick={() => setEditingTag(null)}>Отмена</Button><Button color="accent" icon="save" onClick={saveTag}>Сохранить</Button></>}>
          <div className="col">
            <input placeholder="Название" value={editingTag.name} onChange={(e) => setEditingTag({ ...editingTag, name: e.target.value })} />
            <select value={editingTag.kind} onChange={(e) => setEditingTag({ ...editingTag, kind: e.target.value })}>
              <option value="text">text</option>
              <option value="task_id">task_id</option>
            </select>
            <input placeholder="Цвет (#rrggbb)" value={editingTag.color} onChange={(e) => setEditingTag({ ...editingTag, color: e.target.value })} />
          </div>
        </Modal>
      )}

    </div>
  )
}
