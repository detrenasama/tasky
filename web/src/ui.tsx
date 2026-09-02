import React, { useEffect, useRef, useState } from 'react'
import './ui/button.css'
import { Button } from './ui/button'

export { Button } from './ui/button'
export type { ButtonColor, ButtonVariant, ButtonProps } from './ui/button'
export { Icon } from './ui/icons'
export type { IconName } from './ui/icons'
export { Tooltip } from './ui/tooltip'

export function Modal(props: {
  title: string
  onClose: () => void
  children: React.ReactNode
  footer?: React.ReactNode
  wide?: boolean
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
      <div className={`modal${props.wide ? ' modal--wide' : ''}`} onMouseDown={(e) => e.stopPropagation()}>
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
          <Button variant="outline" onClick={() => close(false)}>Нет</Button>
          <Button color="accent" onClick={() => close(true)}>
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
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!state) return
    const outside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose()
    }
    // Отложенно: слушатели вешаем в следующем тике, чтобы событие, которым
    // меню было открыто, не закрыло его в тот же момент (иначе меню
    // «не открывается» — мгновенно закрывается тем же правым кликом).
    const id = setTimeout(() => {
      window.addEventListener('mousedown', outside)
      window.addEventListener('contextmenu', outside)
    }, 0)
    return () => {
      clearTimeout(id)
      window.removeEventListener('mousedown', outside)
      window.removeEventListener('contextmenu', outside)
    }
  }, [state, onClose])
  if (!state) return null
  return (
    <div ref={ref} className="ctx-menu" style={{ left: state.x, top: state.y }}>
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

// useColumnWidth — ширина колонки с возможностью перетаскивания правого края
// мышью. При dragging обновляет width (с зажатым 120..700px). Если задан
// storageKey — начальное значение берётся из localStorage и ширина пишется
// туда при каждом изменении (сохраняется между перезагрузками страницы).
export function useColumnWidth(initial: number, storageKey?: string) {
  const [width, setWidth] = useState<number>(() => {
    if (storageKey) {
      const v = localStorage.getItem(storageKey)
      const n = v ? parseInt(v, 10) : NaN
      if (Number.isFinite(n) && n >= 120) return n
    }
    return initial
  })
  const drag = useRef<{ x: number; w: number } | null>(null)
  const move = useRef((e: MouseEvent) => {
    if (!drag.current) return
    const next = Math.max(180, Math.min(700, drag.current.w + (e.clientX - drag.current.x)))
    setWidth(next)
    if (storageKey) localStorage.setItem(storageKey, String(next))
  })
  const up = useRef(() => {
    drag.current = null
    window.removeEventListener('mousemove', move.current)
    window.removeEventListener('mouseup', up.current)
  })
  const onDown = (e: React.MouseEvent) => {
    e.preventDefault()
    drag.current = { x: e.clientX, w: width }
    window.addEventListener('mousemove', move.current)
    window.addEventListener('mouseup', up.current)
  }
  return { width, onDown }
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

// ErrorBoundary ловит необработанные ошибки рендера дочерних экранов и
// показывает диалог с текстом ошибки вместо того, чтобы оставлять пустой
// (белый) экран. Кнопка «OK» сбрасывает состояние — при повторном рендере
// (если данные исправились) экран восстанавливается.
export class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { error: Error | null }
> {
  state = { error: null as Error | null }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  render() {
    if (this.state.error) {
      return (
        <Modal
          title="Ошибка"
          onClose={() => this.setState({ error: null })}
          footer={
            <Button color="accent" onClick={() => this.setState({ error: null })}>
              OK
            </Button>
          }
        >
          <p className="muted">{this.state.error.message}</p>
        </Modal>
      )
    }
    return this.props.children
  }
}
