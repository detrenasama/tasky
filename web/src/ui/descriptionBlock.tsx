import { useEffect, useState } from 'react'
import './descriptionBlock.css'
import { Button } from './button'
import { Modal } from '../ui'
import { useDirtyConfirm } from '../hooks/useDirtyConfirm'

export function DescriptionBlock({
  value,
  onSave,
}: {
  value: string
  onSave: (v: string) => Promise<void> | void
}) {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState(value)
  const [saving, setSaving] = useState(false)

  const dirty = draft !== value

  useEffect(() => {
    if (!open) setDraft(value)
  }, [value, open])

  const { confirmDiscard, tryClose, doDiscard, cancelDiscard } = useDirtyConfirm({
    dirty,
    onClose: () => setOpen(false),
    onDiscard: () => setDraft(value),
  })

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
    </>
  )
}
