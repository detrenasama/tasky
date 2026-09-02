import * as projects from './projects'
import * as tasks from './tasks'
import * as time from './time'
import * as journal from './journal'
import * as checklist from './checklist'
import * as statuses from './statuses'
import * as tags from './tags'
import * as reports from './reports'
import * as settings from './settings'
import * as version from './version'

export const api = {
  ...projects,
  ...tasks,
  ...time,
  ...journal,
  ...checklist,
  ...statuses,
  ...tags,
  ...reports,
  ...settings,
  ...version,
}

export * from './projects'
export * from './tasks'
export * from './time'
export * from './journal'
export * from './checklist'
export * from './statuses'
export * from './tags'
export * from './reports'
export * from './settings'
export * from './version'
export { req } from './client'
