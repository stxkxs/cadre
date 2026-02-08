import { api } from './client'
import type { Task } from '@/types'

export const tasksApi = {
  list: () => api.get<Task[]>('/tasks'),
  get: (name: string) => api.get<Task>(`/tasks/${name}`),
  create: (task: Task) => api.post<Task>('/tasks', task),
  update: (name: string, task: Task) => api.put<Task>(`/tasks/${name}`, task),
  delete: (name: string) => api.delete(`/tasks/${name}`),
}
