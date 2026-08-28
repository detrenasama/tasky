import React, { useEffect, useState } from 'react'

export function Button(props: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const { className, ...rest } = props
  return <button className={`btn ${className ?? ''}`} {...rest} />
}

export function Modal(props: {
  title: string
  onClose: () => void
  children: React.ReactNode
  footer?: React.ReactNode
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') props.onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [props])
  return (
    <div className="modal-overlay" onMouseDown={props.onClose}>
      <div className="modal" onMouseDown={(e) => e.stopPropagation()}>
        <div className="modal-title">{props.title}</div>
        <div className="modal-body">{props.children}</div>
        {props.footer && <div className="modal-footer">{props.footer}</div>}
      </div>
    </div>
  )
}

export function useConfirm(): [
  (msg: string) => Promise<boolean>,
  React.ReactNode,
] {
  const [state, setState] = useState<{ msg: string; resolve: (v: boolean) => void } | null>(null)
  const ask = (msg: string) =>
    new Promise<boolean>((resolve) => setState({ msg, resolve }))
  const close = (v: boolean) => {
    state?.resolve(v)
    setState(null)
  }
  const node = state && (
    <Modal
      title="Подтверждение"
      onClose={() => close(false)}
      footer={
        <>
          <Button onClick={() => close(false)}>Нет</Button>
          <Button className="primary" onClick={() => close(true)}>
            Да
          </Button>
        </>
      }
    >
      <p>{state.msg}</p>
    </Modal>
  )
  return [ask, node]
}

export interface MenuState {
  x: number
  y: number
  items: { label: string; onClick: () => void; danger?: boolean }[]
}

export function ContextMenu(props: { state: MenuState | null; onClose: () => void }) {
  const { state, onClose } = props
  useEffect(() => {
    if (!state) return
    const h = () => onClose()
    window.addEventListener('click', h)
    window.addEventListener('contextmenu', h)
    return () => {
      window.removeEventListener('click', h)
      window.removeEventListener('contextmenu', h)
    }
  }, [state, onClose])
  if (!state) return null
  return (
    <div className="ctx-menu" style={{ left: state.x, top: state.y }}>
      {state.items.map((it, i) => (
        <div
          key={i}
          className={`ctx-item ${it.danger ? 'danger' : ''}`}
          onClick={() => {
            it.onClick()
            onClose()
          }}
        >
          {it.label}
        </div>
      ))}
    </div>
  )
}

// Простой всплывающий список ошибок/уведомлений.
export function useToast(): [string | null, (m: string | null) => void] {
  const [msg, setMsg] = useState<string | null>(null)
  useEffect(() => {
    if (!msg) return
    const t = setTimeout(() => setMsg(null), 4000)
    return () => clearTimeout(t)
  }, [msg])
  return [msg, setMsg]
}
