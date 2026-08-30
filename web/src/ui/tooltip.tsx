import { useState, useRef, useEffect } from 'react'
import { createPortal } from 'react-dom'

export function Tooltip({
  content,
  children,
  delay = 300,
}: {
  content: string
  children: React.ReactNode
  delay?: number
}) {
  const [visible, setVisible] = useState(false)
  const [pos, setPos] = useState({ x: 0, y: 0 })
  const ref = useRef<HTMLSpanElement>(null)
  const timer = useRef<number | null>(null)

  const show = () => {
    if (timer.current) window.clearTimeout(timer.current)
    timer.current = window.setTimeout(() => {
      if (!ref.current) return
      const r = ref.current.getBoundingClientRect()
      setPos({ x: r.left + r.width / 2, y: r.top })
      setVisible(true)
    }, delay) as unknown as number
  }
  const hide = () => {
    if (timer.current) window.clearTimeout(timer.current)
    setVisible(false)
  }

  useEffect(() => () => { if (timer.current) window.clearTimeout(timer.current) }, [])

  return (
    <>
      <span
        ref={ref}
        onMouseEnter={show}
        onMouseLeave={hide}
        onFocus={show}
        onBlur={hide}
        style={{ display: 'inline-flex' }}
      >
        {children}
      </span>
      {visible &&
        createPortal(
          <div
            className="tooltip-popover"
            style={{ left: pos.x, top: pos.y - 10, transform: 'translate(-50%, -100%)' }}
          >
            {content}
          </div>,
          document.body,
        )}
    </>
  )
}
