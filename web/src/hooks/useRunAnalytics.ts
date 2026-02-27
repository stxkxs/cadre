import { useMemo } from 'react'
import { durationBetween, isWithinRange, type TimeRange } from '@/lib/time-utils'
import type { Run } from '@/types'

interface StatusCounts {
  completed: number
  failed: number
  cancelled: number
  running: number
  pending: number
}

interface CrewStat {
  name: string
  total: number
  successRate: number
  avgDuration: number
}

interface TaskDurationStat {
  name: string
  avgDuration: number
}

export function useRunAnalytics(runs: Run[] | undefined, timeRange: TimeRange = '7d') {
  return useMemo(() => {
    const allRuns = runs || []

    // Status counts
    const statusCounts: StatusCounts = { completed: 0, failed: 0, cancelled: 0, running: 0, pending: 0 }
    for (const run of allRuns) {
      if (run.status in statusCounts) {
        statusCounts[run.status as keyof StatusCounts]++
      }
    }

    // Crew stats
    const crewMap = new Map<string, { total: number; completed: number; durations: number[] }>()
    for (const run of allRuns) {
      let entry = crewMap.get(run.crew_name)
      if (!entry) {
        entry = { total: 0, completed: 0, durations: [] }
        crewMap.set(run.crew_name, entry)
      }
      entry.total++
      if (run.status === 'completed') entry.completed++
      if (run.started_at && run.completed_at) {
        entry.durations.push(durationBetween(run.started_at, run.completed_at))
      }
    }
    const crewStats: CrewStat[] = Array.from(crewMap.entries()).map(([name, data]) => ({
      name,
      total: data.total,
      successRate: data.total > 0 ? (data.completed / data.total) * 100 : 0,
      avgDuration: data.durations.length > 0 ? data.durations.reduce((a, b) => a + b, 0) / data.durations.length : 0,
    })).sort((a, b) => b.total - a.total)

    // Task durations (from all runs)
    const taskMap = new Map<string, number[]>()
    for (const run of allRuns) {
      for (const task of run.tasks || []) {
        if (task.started_at && task.completed_at) {
          const durations = taskMap.get(task.name) || []
          durations.push(durationBetween(task.started_at, task.completed_at))
          taskMap.set(task.name, durations)
        }
      }
    }
    const taskDurations: TaskDurationStat[] = Array.from(taskMap.entries())
      .map(([name, durations]) => ({
        name,
        avgDuration: durations.reduce((a, b) => a + b, 0) / durations.length,
      }))
      .sort((a, b) => b.avgDuration - a.avgDuration)
      .slice(0, 10)

    // Recent activity (filtered by time range)
    const recentActivity = allRuns
      .filter((r) => r.started_at && isWithinRange(r.started_at, timeRange))
      .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())
      .slice(0, 15)

    return { statusCounts, crewStats, taskDurations, recentActivity }
  }, [runs, timeRange])
}
