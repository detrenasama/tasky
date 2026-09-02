import { req } from './client'

export const version = () => req<{ version: string }>('GET', '/version')
