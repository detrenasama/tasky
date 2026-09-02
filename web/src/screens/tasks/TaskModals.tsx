import { Button, Modal } from '../../ui'

export function StatusNoteModal({ pendingStatus, pendingNote, setPendingNote, setPendingStatus, execStatus }: any) {
  if (!pendingStatus) return null
  return (
    <Modal
      title={`${pendingStatus.to} — ${pendingStatus.prompt}`}
      onClose={() => setPendingStatus(null)}
      footer={
        <>
          <Button variant="outline" onClick={() => setPendingStatus(null)}>Отмена</Button>
          <Button color="accent" disabled={!pendingNote.trim()} onClick={async () => { const ps = pendingStatus!; setPendingStatus(null); await execStatus(ps.kind, ps.id, ps.to, pendingNote.trim()) }}>
            Применить
          </Button>
        </>
      }
    >
      <textarea autoFocus placeholder={pendingStatus.prompt} value={pendingNote} onChange={(e) => setPendingNote(e.target.value)} style={{ width: '100%', minHeight: 80 }} />
    </Modal>
  )
}

export function AddModal({ addKind, addTitle, setAddKind, setAddTitle, submitAdd }: any) {
  if (!addKind) return null
  return (
    <Modal
      title={addKind === 'task' ? 'Новая задача' : 'Новая подзадача'}
      onClose={() => setAddKind(null)}
      footer={
        <>
          <Button variant="outline" onClick={() => setAddKind(null)}>Отмена</Button>
          <Button color="accent" icon="plus" disabled={!addTitle.trim()} onClick={submitAdd}>Создать</Button>
        </>
      }
    >
      <input
        autoFocus
        placeholder="Название"
        value={addTitle}
        style={{ width: '100%', minWidth: 360 }}
        onChange={(e) => setAddTitle(e.target.value)}
        onKeyDown={(e) => { if (e.key === 'Enter') submitAdd() }}
      />
    </Modal>
  )
}
