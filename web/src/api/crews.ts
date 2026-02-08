import { api } from './client'
import type { Crew, ValidationResult } from '@/types'

export const crewsApi = {
  list: () => api.get<Crew[]>('/crews'),
  get: (name: string) => api.get<Crew>(`/crews/${name}`),
  create: (crew: Crew) => api.post<Crew>('/crews', crew),
  update: (name: string, crew: Crew) => api.put<Crew>(`/crews/${name}`, crew),
  delete: (name: string) => api.delete(`/crews/${name}`),
  validate: (name: string) => api.post<ValidationResult>(`/crews/${name}/validate`),
}
