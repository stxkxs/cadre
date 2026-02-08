import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { agentsApi } from '@/api/agents'
import type { Agent } from '@/types'

export function useAgents() {
  return useQuery({ queryKey: ['agents'], queryFn: agentsApi.list })
}

export function useAgent(name: string) {
  return useQuery({
    queryKey: ['agents', name],
    queryFn: () => agentsApi.get(name),
    enabled: !!name,
  })
}

export function useCreateAgent() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (agent: Agent) => agentsApi.create(agent),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  })
}

export function useUpdateAgent() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, agent }: { name: string; agent: Agent }) =>
      agentsApi.update(name, agent),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  })
}

export function useDeleteAgent() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => agentsApi.delete(name),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  })
}
