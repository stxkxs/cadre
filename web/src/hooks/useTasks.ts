import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { tasksApi } from '@/api/tasks'
import type { Task } from '@/types'

export function useTasks() {
  return useQuery({ queryKey: ['tasks'], queryFn: tasksApi.list })
}

export function useTask(name: string) {
  return useQuery({
    queryKey: ['tasks', name],
    queryFn: () => tasksApi.get(name),
    enabled: !!name,
  })
}

export function useCreateTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (task: Task) => tasksApi.create(task),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks'] }),
  })
}

export function useUpdateTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, task }: { name: string; task: Task }) =>
      tasksApi.update(name, task),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks'] }),
  })
}

export function useDeleteTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => tasksApi.delete(name),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks'] }),
  })
}
