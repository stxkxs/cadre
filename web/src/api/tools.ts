import { api } from './client'
import type { Tool } from '@/types'

export const toolsApi = {
  list: () => api.get<Tool[]>('/tools'),
}
