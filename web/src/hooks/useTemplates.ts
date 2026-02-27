import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { templatesApi } from '@/api/templates'
import type { ImportRequest } from '@/types'

export function useTemplateCategories() {
  return useQuery({ queryKey: ['templates', 'categories'], queryFn: templatesApi.listCategories })
}

export function useTemplateAgents(params?: { category?: string; q?: string }) {
  return useQuery({
    queryKey: ['templates', 'agents', params],
    queryFn: () => templatesApi.listAgents(params),
  })
}

export function useTemplateTasks(params?: { category?: string; q?: string }) {
  return useQuery({
    queryKey: ['templates', 'tasks', params],
    queryFn: () => templatesApi.listTasks(params),
  })
}

export function useTemplateCrews(params?: { category?: string; q?: string }) {
  return useQuery({
    queryKey: ['templates', 'crews', params],
    queryFn: () => templatesApi.listCrews(params),
  })
}

export function useImportTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: ImportRequest) => templatesApi.import(req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agents'] })
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['crews'] })
    },
  })
}
