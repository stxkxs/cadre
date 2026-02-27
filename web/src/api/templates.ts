import { api } from './client'
import type { TemplateAgent, TemplateTask, TemplateCrew, ImportRequest } from '@/types'

interface CategoryInfo {
  id: string
  label: string
  icon: string
  agent_count: number
  task_count: number
  crew_count: number
}

export const templatesApi = {
  listCategories: () => api.get<CategoryInfo[]>('/templates'),
  listAgents: (params?: { category?: string; q?: string }) => {
    const search = new URLSearchParams()
    if (params?.category) search.set('category', params.category)
    if (params?.q) search.set('q', params.q)
    const qs = search.toString()
    return api.get<TemplateAgent[]>(`/templates/agents${qs ? '?' + qs : ''}`)
  },
  getAgent: (name: string) => api.get<TemplateAgent>(`/templates/agents/${name}`),
  listTasks: (params?: { category?: string; q?: string }) => {
    const search = new URLSearchParams()
    if (params?.category) search.set('category', params.category)
    if (params?.q) search.set('q', params.q)
    const qs = search.toString()
    return api.get<TemplateTask[]>(`/templates/tasks${qs ? '?' + qs : ''}`)
  },
  getTask: (name: string) => api.get<TemplateTask>(`/templates/tasks/${name}`),
  listCrews: (params?: { category?: string; q?: string }) => {
    const search = new URLSearchParams()
    if (params?.category) search.set('category', params.category)
    if (params?.q) search.set('q', params.q)
    const qs = search.toString()
    return api.get<TemplateCrew[]>(`/templates/crews${qs ? '?' + qs : ''}`)
  },
  getCrew: (name: string) => api.get<TemplateCrew>(`/templates/crews/${name}`),
  import: (req: ImportRequest) => api.post<{ status: string; name: string }>('/templates/import', req),
}
