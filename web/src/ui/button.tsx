import React from 'react'
import { Icon, IconName } from './icons'
import { Tooltip } from './tooltip'
import './button.css'

export type ButtonColor = 'base' | 'accent' | 'success' | 'warning' | 'danger' | 'purple' | 'grey'
export type ButtonVariant = 'filled' | 'outline'

export interface ButtonProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'color'> {
  color?: ButtonColor
  variant?: ButtonVariant
  /** Имя иконки lucide. Если указан без children — режим icon-only (требует label для tooltip). */
  icon?: IconName
  /** Текст для tooltip в режиме icon-only (если не указан — берётся children). */
  label?: string
  /** Раскрывать текст при hover/focus (только с icon + children). */
  expand?: boolean
}

// Маппинг старых className для обратной совместимости
function legacyColor(className?: string): { color?: ButtonColor; variant?: ButtonVariant } {
  if (!className) return {}
  if (className.includes('primary')) return { color: 'accent', variant: 'filled' }
  if (className.includes('danger')) return { color: 'danger', variant: 'outline' }
  return {}
}

export function Button({
  color: colorProp,
  variant: variantProp,
  icon,
  label,
  expand,
  className,
  children,
  ...rest
}: ButtonProps) {
  const legacy = legacyColor(className)
  const color = colorProp ?? legacy.color ?? 'base'
  const variant = variantProp ?? legacy.variant ?? 'filled'

  const hasIcon = !!icon
  const hasText = React.Children.count(children) > 0 && String(children).trim().length > 0
  const isIconOnly = hasIcon && !hasText
  const isExpand = !!expand && hasIcon && hasText

  const extraClass = className
    ? className
        .split(/\s+/)
        .filter((c) => c !== 'primary' && c !== 'danger' && c !== 'small' && c !== 'btn')
        .join(' ')
    : ''

  const cls = [
    'btn',
    `btn--${variant}`,
    `btn--${color}`,
    isIconOnly ? 'btn--icon' : '',
    isExpand ? 'btn--expand' : '',
    extraClass,
  ]
    .filter(Boolean)
    .join(' ')

  let btn: React.ReactNode
  if (!hasIcon) {
    btn = (
      <button className={cls} {...rest}>
        {children}
      </button>
    )
  } else if (isIconOnly) {
    btn = (
      <button className={cls} aria-label={label ?? (typeof children === 'string' ? children : undefined)} {...rest}>
        <span className="btn__icon">
          <Icon name={icon!} size={16} />
        </span>
      </button>
    )
  } else {
    btn = (
      <button className={cls} {...rest}>
        <span className="btn__icon">
          <Icon name={icon!} size={16} />
        </span>
        <span className="btn__label">{children}</span>
      </button>
    )
  }

  if (isIconOnly) {
    const tip = label ?? (typeof children === 'string' ? children : '')
    if (!tip) return btn as React.ReactElement
    return <Tooltip content={tip}>{btn}</Tooltip>
  }

  return btn as React.ReactElement
}
