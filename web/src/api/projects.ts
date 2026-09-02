import { req } from './client'
import type { Project, Link } from '../types'

export const projects = () => req<Project[]>('GET', '/projects')
export const createProject = (name: string) => req<Project>('POST', '/projects', { name })
export const deleteProject = (id: number) => req<{ ok: boolean }>('DELETE', `/projects/${id}`)
export const projectDescription = (id: number) =>
  req<{ description: string }>('GET', `/projects/${id}/description`)
export const updateProjectDescription = (id: number, description: string) =>
  req<{ ok: boolean }>('PUT', `/projects/${id}/description`, { description })
export const projectLinks = (id: number) => req<Link[]>('GET', `/projects/${id}/links`)
export const createProjectLink = (id: number, name: string, url: string) =>
  req<Link>('POST', `/projects/${id}/links`, { name, url })
export const updateProjectLink = (id: number, name: string, url: string) =>
  req<{ ok: boolean }>('PUT', `/projectlinks/${id}`, { name, url })
export const deleteProjectLink = (id: number) => req<{ ok: boolean }>('DELETE', `/projectlinks/${id}`)
