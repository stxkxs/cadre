import { api } from './client'
import type { Run } from '@/types'

export const runsApi = {
  list: () => api.get<Run[]>('/runs'),
  get: (id: string) => api.get<Run>(`/runs/${id}`),
  start: (crew: string, inputs?: Record<string, unknown>) =>
    api.post<{ id: string; status: string }>('/runs', { crew, inputs }),
  cancel: (id: string) => api.post<{ status: string }>(`/runs/${id}/cancel`),
}
