import { useState } from 'react'
import type { Tag, TagType } from '../types'
import { Button, Modal, ContextMenu, MenuState } from '../ui'
import './tagBar.css'

export function TagBar({
  tags,
  tagTypes,
  onAdd,
  onEdit,
  onDelete,
}: {
  tags: Tag[]
  tagTypes: TagType[]
  onAdd: (typeId: number, text: string, url: string) => Promise<void> | void
  onEdit: (id: number, typeId: number, text: string, url: string) => Promise<void> | void
  onDelete: (id: number) => Promise<void> | void
}) {
  const [menu, setMenu] = useState<MenuState | null>(null)
  const [editing, setEditing] = useState<Tag | null>(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({ typeId: 0, text: '', url: '' })

  const openAdd = () => {
    setForm({ typeId: tagTypes[0]?.id ?? 0, text: '', url: '' })
    setAdding(true)
  }
  const openEdit = (tg: Tag) => {
    setForm({ typeId: tg.type_id, text: tg.text, url: tg.url })
    setEditing(tg)
  }
  const closeForm = () => {
    setAdding(false)
    setEditing(null)
  }
  const submitAdd = async () => {
    if (!form.typeId || !form.text.trim()) return
    await onAdd(form.typeId, form.text.trim(), form.url.trim())
    closeForm()
  }
  const submitEdit = async () => {
    if (!editing || !form.typeId || !form.text.trim()) return
    await onEdit(editing.id, form.typeId, form.text.trim(), form.url.trim())
    closeForm()
  }

  const isFormOpen = adding || editing !== null
  const formTitle = editing ? 'Редактировать тег' : 'Новый тег'
  const canSave = !!form.typeId && !!form.text.trim()

  return (
    <>
      <div className="tag-bar">
        {tags.map((tg) => (
          <Button
            key={tg.id}
            variant="outline"
            style={{ borderColor: tg.color || 'var(--border)', color: tg.color || 'var(--text)' }}
            onClick={() => {
              if (tg.url) window.open(tg.url, '_blank', 'noopener')
            }}
            onContextMenu={(e) => {
              e.preventDefault()
              setMenu({
                x: e.clientX,
                y: e.clientY,
                items: [
                  { label: 'Редактировать', onClick: () => openEdit(tg) },
                  { label: 'Открыть ссылку', onClick: () => { if (tg.url) window.open(tg.url, '_blank', 'noopener') } },
                  { label: 'Удалить', danger: true, onClick: () => onDelete(tg.id) },
                ],
              })
            }}
          >
            {tg.text}
          </Button>
        ))}
        <Button icon="plus" label="Добавить тег" onClick={openAdd} />
      </div>

      <ContextMenu state={menu} onClose={() => setMenu(null)} />

      {isFormOpen && (
        <Modal
          title={formTitle}
          onClose={closeForm}
          footer={
            <>
              <Button variant="outline" onClick={closeForm}>Отмена</Button>
              <Button color="accent" disabled={!canSave} onClick={editing ? submitEdit : submitAdd}>
                {editing ? 'Сохранить' : 'Создать'}
              </Button>
            </>
          }
        >
          <div className="col">
            <select value={form.typeId} onChange={(e) => setForm({ ...form, typeId: Number(e.target.value) })}>
              {tagTypes.map((tt) => (
                <option key={tt.id} value={tt.id}>{tt.name}</option>
              ))}
            </select>
            <input placeholder="Текст" value={form.text} onChange={(e) => setForm({ ...form, text: e.target.value })} />
            <input placeholder="URL (необязательно)" value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} />
          </div>
        </Modal>
      )}
    </>
  )
}
