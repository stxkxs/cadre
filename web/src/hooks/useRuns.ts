import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { runsApi } from '@/api/runs'

export function useRuns() {
  return useQuery({ queryKey: ['runs'], queryFn: runsApi.list, refetchInterval: 5000 })
}

export function useRun(id: string) {
  return useQuery({
    queryKey: ['runs', id],
    queryFn: () => runsApi.get(id),
    enabled: !!id,
    refetchInterval: 3000,
  })
}

export function useStartRun() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ crew, inputs }: { crew: string; inputs?: Record<string, unknown> }) =>
      runsApi.start(crew, inputs),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs'] }),
  })
}

export function useCancelRun() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => runsApi.cancel(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs'] }),
  })
}
