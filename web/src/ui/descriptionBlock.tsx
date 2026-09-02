import { useEffect, useState } from 'react'
import { Button } from './button'
import { Modal } from '../ui'

export function DescriptionBlock({
  value,
  onSave,
}: {
  value: string
  onSave: (v: string) => Promise<void> | void
}) {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState(value)
  const [confirmDiscard, setConfirmDiscard] = useState(false)
  const [saving, setSaving] = useState(false)

  const dirty = draft !== value

  useEffect(() => {
    if (!open) setDraft(value)
  }, [value, open])

  const tryClose = () => {
    if (!dirty) {
      setOpen(false)
    } else {
      setConfirmDiscard(true)
    }
  }

  const doDiscard = () => {
    setConfirmDiscard(false)
    setOpen(false)
    setDraft(value)
  }

  const handleSave = async () => {
    if (!dirty) return
    setSaving(true)
    try {
      await onSave(draft)
      setOpen(false)
    } finally {
      setSaving(false)
    }
  }

  const openEdit = () => {
    setDraft(value)
    setOpen(true)
  }

  return (
    <>
      <div>
        <div className="flex between">
          <strong>Описание</strong>
          <Button variant="outline" icon="edit" label="Редактировать описание" onClick={openEdit} />
        </div>
        <div className="desc-display">
          {value ? value : <span className="muted">Описания нет — нажмите ✎</span>}
        </div>
      </div>

      {open && (
        <Modal
          wide
          title="Редактирование описания"
          onClose={tryClose}
          footer={
            <>
              <Button variant="outline" onClick={tryClose}>
                Отменить
              </Button>
              <Button color="accent" icon="save" disabled={!dirty || saving} onClick={handleSave}>
                Сохранить
              </Button>
            </>
          }
        >
          <textarea
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            style={{ width: '100%', minHeight: 160 }}
            placeholder="Описание..."
          />
        </Modal>
      )}

      {confirmDiscard && (
        <Modal
          title="Несохранённые изменения"
          onClose={() => setConfirmDiscard(false)}
          footer={
            <>
              <Button variant="outline" onClick={() => setConfirmDiscard(false)}>
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
    </>
  )
}
