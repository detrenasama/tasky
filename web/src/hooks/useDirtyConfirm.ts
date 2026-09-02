import { useCallback, useState } from 'react'

export function useDirtyConfirm({
  dirty,
  onClose,
  onDiscard,
}: {
  dirty: boolean
  onClose: () => void
  onDiscard?: () => void
}) {
  const [confirmDiscard, setConfirmDiscard] = useState(false)

  const tryClose = useCallback(() => {
    if (!dirty) onClose()
    else setConfirmDiscard(true)
  }, [dirty, onClose])

  const doDiscard = useCallback(() => {
    setConfirmDiscard(false)
    onDiscard?.()
    onClose()
  }, [onClose, onDiscard])

  const cancelDiscard = useCallback(() => setConfirmDiscard(false), [])

  return { confirmDiscard, tryClose, doDiscard, cancelDiscard, setConfirmDiscard }
}
