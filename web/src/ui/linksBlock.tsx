import { useState } from 'react'
import './linksBlock.css'
import type { Link } from '../types'
import { Button } from './button'
import { Modal } from '../ui'
import { useDirtyConfirm } from '../hooks/useDirtyConfirm'

export function LinksBlock({
  links,
  onAdd,
  onEdit,
  onDel,
}: {
  links: Link[]
  onAdd: (n: string, u: string) => void
  onEdit: (id: number, n: string, u: string) => void
  onDel: (id: number) => void
}) {
  const [editing, setEditing] = useState<Link | null>(null)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState({ name: '', url: '' })
  const [confirmId, setConfirmId] = useState<number | null>(null)

  const openCreate = () => {
    setForm({ name: '', url: '' })
    setCreating(true)
  }

  const openEdit = (l: Link) => {
    setForm({ name: l.name, url: l.url })
    setEditing(l)
  }

  const closeForm = () => {
    setCreating(false)
    setEditing(null)
  }

  const initial = editing ? { name: editing.name, url: editing.url } : { name: '', url: '' }
  const dirty = form.name !== initial.name || form.url !== initial.url

  const { confirmDiscard, tryClose: tryCloseForm, doDiscard, cancelDiscard } = useDirtyConfirm({
    dirty,
    onClose: closeForm,
  })

  const handleSave = () => {
    const name = form.name.trim()
    const url = form.url.trim()
    if (!url) return
    if (editing) {
      onEdit(editing.id, name, url)
    } else {
      onAdd(name, url)
    }
    closeForm()
  }

  const isEditing = !!editing
  const showForm = creating || isEditing
  const confirmLink = links.find((l) => l.id === confirmId)

  return (
    <div>
      <div className="flex between">
        <strong>Ссылки</strong>
        <Button variant="outline" icon="plus" label="Добавить ссылку" onClick={openCreate} />
      </div>

      {links.length === 0 ? (
        <span className="muted small">Ссылок нет</span>
      ) : (
        <div className="links-table">
          {links.map((l) => (
            <div key={l.id} className="row links-row">
              <span className="link-name" title={l.name || l.url}>
                {l.name || '—'}
              </span>
              <a
                className="link-url"
                href={l.url}
                target="_blank"
                rel="noreferrer"
                title={l.url}
                onClick={(e) => e.stopPropagation()}
              >
                {l.url}
              </a>
              <span className="flex" style={{ gap: 4, flexShrink: 0 }}>
                <Button variant="outline" icon="edit" label="Редактировать ссылку" onClick={() => openEdit(l)} />
                <Button variant="outline" color="danger" icon="trash" label="Удалить ссылку" onClick={() => setConfirmId(l.id)} />
              </span>
            </div>
          ))}
        </div>
      )}

      {showForm && (
        <Modal
          title={isEditing ? 'Редактировать ссылку' : 'Добавить ссылку'}
          onClose={tryCloseForm}
          footer={
            <>
              <Button variant="outline" onClick={tryCloseForm}>
                Отменить
              </Button>
              <Button color="accent" icon="save" disabled={!form.url.trim()} onClick={handleSave}>
                Сохранить
              </Button>
            </>
          }
        >
          <div className="col">
            <input
              autoFocus
              placeholder="Название (необязательно)"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
            <input
              placeholder="https://..."
              value={form.url}
              onChange={(e) => setForm({ ...form, url: e.target.value })}
              onKeyDown={(e) => { if (e.key === 'Enter' && form.url.trim()) handleSave() }}
            />
          </div>
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

      {confirmId !== null && (
        <Modal
          title="Удалить ссылку"
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
          <p>
            Удалить ссылку «{confirmLink?.name || confirmLink?.url}»?
          </p>
        </Modal>
      )}
    </div>
  )
}
