import { req } from './client'
import type { Settings } from '../types'

export const settings = () => req<Settings>('GET', '/settings')
export const getSetting = (key: string) => req<{ value: string }>('GET', `/settings/${key}`)
export const setSetting = (key: string, value: string) =>
  req<{ ok: boolean }>('PUT', `/settings/${key}`, { value })
