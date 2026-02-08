import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { crewsApi } from '@/api/crews'
import type { Crew } from '@/types'

export function useCrews() {
  return useQuery({ queryKey: ['crews'], queryFn: crewsApi.list })
}

export function useCrew(name: string) {
  return useQuery({
    queryKey: ['crews', name],
    queryFn: () => crewsApi.get(name),
    enabled: !!name,
  })
}

export function useCreateCrew() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (crew: Crew) => crewsApi.create(crew),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['crews'] }),
  })
}

export function useUpdateCrew() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, crew }: { name: string; crew: Crew }) =>
      crewsApi.update(name, crew),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['crews'] }),
  })
}

export function useDeleteCrew() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => crewsApi.delete(name),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['crews'] }),
  })
}

export function useValidateCrew() {
  return useMutation({
    mutationFn: (name: string) => crewsApi.validate(name),
  })
}
