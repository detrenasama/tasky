import { req } from './client'
import type { TagType, Tag } from '../types'

export const tagTypes = () => req<TagType[]>('GET', '/tagtypes')
export const createTagType = (t: Omit<TagType, 'id' | 'sort_order'>) =>
  req<TagType>('POST', '/tagtypes', t)
export const updateTagType = (id: number, t: Omit<TagType, 'id' | 'sort_order'>) =>
  req<{ ok: boolean }>('PUT', `/tagtypes/${id}`, t)
export const deleteTagType = (id: number) => req<{ ok: boolean }>('DELETE', `/tagtypes/${id}`)
export const taskTags = (id: number) => req<Tag[]>('GET', `/tasks/${id}/tags`)
export const tagsByProject = (pid: number) => req<Record<string, Tag[]>>('GET', `/projects/${pid}/tags`)
export const createTag = (tid: number, typeId: number, text: string, url: string) =>
  req<Tag>('POST', `/tasks/${tid}/tags`, { type_id: typeId, text, url })
export const updateTag = (id: number, typeId: number, text: string, url: string) =>
  req<{ ok: boolean }>('PUT', `/tags/${id}`, { type_id: typeId, text, url })
export const deleteTag = (id: number) => req<{ ok: boolean }>('DELETE', `/tags/${id}`)
