import { useState } from 'react'
import { Bot, ListTodo, Users, Play, ArrowRight, Check, X as XIcon, Clock, Loader2, Inbox } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAgents } from '@/hooks/useAgents'
import { useTasks } from '@/hooks/useTasks'
import { useCrews } from '@/hooks/useCrews'
import { useRuns } from '@/hooks/useRuns'
import { useRunAnalytics } from '@/hooks/useRunAnalytics'
import { useOnboarding } from '@/hooks/useOnboarding'
import { GettingStarted } from '@/components/onboarding/GettingStarted'
import { RunSuccessChart } from '@/components/dashboard/RunSuccessChart'
import { CrewPerformanceCard } from '@/components/dashboard/CrewPerformanceCard'
import { TaskDurationChart } from '@/components/dashboard/TaskDurationChart'
import { ActivityTimeline } from '@/components/dashboard/ActivityTimeline'
import { QuickActions } from '@/components/dashboard/QuickActions'
import { cn } from '@/lib/utils'
import type { TimeRange } from '@/lib/time-utils'

function StatCard({ title, value, icon: Icon, href }: {
  title: string; value: number | string; icon: React.ElementType; href: string
}) {
  return (
    <Link to={href}>
      <Card className="group hover:border-[var(--primary-accent)]/25 hover:shadow-[0_0_20px_-4px_var(--primary-accent)]/10">
        <CardContent className="p-5">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{title}</p>
              <p className="text-3xl font-bold font-mono tracking-tight">{value}</p>
            </div>
            <div className="h-10 w-10 rounded-xl bg-[var(--primary-accent)]/10 flex items-center justify-center">
              <Icon className="h-5 w-5 text-[var(--primary-accent)]" />
            </div>
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}

const statusIcon: Record<string, React.ElementType> = {
  running: Loader2, completed: Check, failed: XIcon, pending: Clock, cancelled: XIcon,
}

const statusStyle: Record<string, string> = {
  running: 'text-[var(--accent-blue)] animate-spin',
  completed: 'text-[var(--accent-green)]',
  failed: 'text-destructive',
  pending: 'text-muted-foreground',
  cancelled: 'text-muted-foreground',
}

export function Dashboard() {
  const { data: agents } = useAgents()
  const { data: tasks } = useTasks()
  const { data: crews } = useCrews()
  const { data: runs } = useRuns()
  const [timeRange, setTimeRange] = useState<TimeRange>('7d')
  const { statusCounts, crewStats, taskDurations, recentActivity } = useRunAnalytics(runs, timeRange)
  const onboarding = useOnboarding()

  // Show onboarding when project is empty
  if (onboarding.showOnboarding) {
    return (
      <div className="max-w-2xl mx-auto py-8">
        <GettingStarted onDismiss={onboarding.dismiss} />
      </div>
    )
  }

  const recentRuns = (runs || []).slice(0, 8)

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold font-header tracking-tight">Dashboard</h1>
        <p className="text-xs text-muted-foreground mt-0.5">System overview and recent activity</p>
      </div>

      {/* Row 1: Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard title="Agents" value={agents?.length ?? 0} icon={Bot} href="/agents" />
        <StatCard title="Tasks" value={tasks?.length ?? 0} icon={ListTodo} href="/tasks" />
        <StatCard title="Crews" value={crews?.length ?? 0} icon={Users} href="/crews" />
        <StatCard title="Runs" value={runs?.length ?? 0} icon={Play} href="/runs" />
      </div>

      {/* Row 2: Charts */}
      <div className="grid gap-4 md:grid-cols-2">
        <RunSuccessChart statusCounts={statusCounts} />
        <CrewPerformanceCard crewStats={crewStats} />
      </div>

      {/* Row 3: Task Duration + Activity */}
      <div className="grid gap-4 md:grid-cols-2">
        <TaskDurationChart taskDurations={taskDurations} />
        <ActivityTimeline runs={recentActivity} timeRange={timeRange} onTimeRangeChange={setTimeRange} />
      </div>

      {/* Row 4: Quick Actions + Recent Runs */}
      <div className="grid gap-4 md:grid-cols-[280px_1fr]">
        <QuickActions />
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Recent runs</CardTitle>
            {runs && runs.length > 0 && (
              <Link to="/runs" className="text-xs text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1">
                View all <ArrowRight className="h-3 w-3" />
              </Link>
            )}
          </CardHeader>
          <CardContent>
            {recentRuns.length === 0 ? (
              <div className="flex flex-col items-center gap-2 py-8">
                <Inbox className="h-8 w-8 text-muted-foreground/40" />
                <p className="text-xs text-muted-foreground">No runs yet. Start a crew to see activity here.</p>
              </div>
            ) : (
              <div className="space-y-1">
                {recentRuns.map((run) => {
                  const StatusIcon = statusIcon[run.status] || Clock
                  return (
                    <Link
                      key={run.id}
                      to={`/runs/${run.id}`}
                      className="flex items-center justify-between rounded-lg px-3 py-2.5 transition-colors hover:bg-accent group"
                    >
                      <div className="flex items-center gap-3">
                        <StatusIcon className={cn('h-3.5 w-3.5', statusStyle[run.status])} />
                        <span className="text-sm font-medium">{run.crew_name}</span>
                        <span className="text-[10px] font-mono text-muted-foreground">{run.id.slice(0, 8)}</span>
                      </div>
                      <span className="text-[10px] font-mono text-muted-foreground">
                        {new Date(run.started_at).toLocaleString()}
                      </span>
                    </Link>
                  )
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
