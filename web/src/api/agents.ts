import { api } from './client'
import type { Agent } from '@/types'

export const agentsApi = {
  list: () => api.get<Agent[]>('/agents'),
  get: (name: string) => api.get<Agent>(`/agents/${name}`),
  create: (agent: Agent) => api.post<Agent>('/agents', agent),
  update: (name: string, agent: Agent) => api.put<Agent>(`/agents/${name}`, agent),
  delete: (name: string) => api.delete(`/agents/${name}`),
}
