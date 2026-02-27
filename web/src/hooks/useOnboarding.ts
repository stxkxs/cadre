import { useMemo } from 'react'
import { useAgents } from './useAgents'
import { useTasks } from './useTasks'
import { useCrews } from './useCrews'
import { useRuns } from './useRuns'
import type { OnboardingStatus } from '@/types'

const DISMISS_KEY = 'cadre-onboarding-dismissed'

export function useOnboarding(): OnboardingStatus & { dismiss: () => void } {
  const { data: agents } = useAgents()
  const { data: tasks } = useTasks()
  const { data: crews } = useCrews()
  const { data: runs } = useRuns()

  const dismissed = typeof window !== 'undefined' && localStorage.getItem(DISMISS_KEY) === 'true'

  const status = useMemo(() => {
    const hasAgents = (agents?.length ?? 0) > 0
    const hasTasks = (tasks?.length ?? 0) > 0
    const hasCrews = (crews?.length ?? 0) > 0
    const hasRuns = (runs?.length ?? 0) > 0
    const showOnboarding = !dismissed && !hasAgents && !hasTasks && !hasCrews && !hasRuns

    return { showOnboarding, hasAgents, hasTasks, hasCrews, hasRuns }
  }, [agents, tasks, crews, runs, dismissed])

  const dismiss = () => {
    localStorage.setItem(DISMISS_KEY, 'true')
    window.location.reload()
  }

  return { ...status, dismiss }
}
