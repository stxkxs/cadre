import { Link } from 'react-router-dom'
import { Check, X as XIcon, Clock, Loader2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { timeAgo, type TimeRange } from '@/lib/time-utils'
import { cn } from '@/lib/utils'
import type { Run } from '@/types'

interface ActivityTimelineProps {
  runs: Run[]
  timeRange: TimeRange
  onTimeRangeChange: (range: TimeRange) => void
}

const statusIcon: Record<string, React.ElementType> = {
  running: Loader2,
  completed: Check,
  failed: XIcon,
  pending: Clock,
  cancelled: XIcon,
}

const statusStyle: Record<string, string> = {
  running: 'text-[var(--accent-blue)] animate-spin',
  completed: 'text-[var(--accent-green)]',
  failed: 'text-destructive',
  pending: 'text-muted-foreground',
  cancelled: 'text-muted-foreground',
}

const ranges: TimeRange[] = ['24h', '7d', '30d']

export function ActivityTimeline({ runs, timeRange, onTimeRangeChange }: ActivityTimelineProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Activity</CardTitle>
        <div className="flex items-center gap-1">
          {ranges.map((r) => (
            <button
              key={r}
              onClick={() => onTimeRangeChange(r)}
              className={cn(
                'rounded-full px-2.5 py-0.5 text-[10px] font-mono font-medium transition-colors',
                r === timeRange
                  ? 'bg-[var(--primary-accent)] text-white'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {r}
            </button>
          ))}
        </div>
      </CardHeader>
      <CardContent>
        {runs.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-4">No activity in this period.</p>
        ) : (
          <div className="space-y-1">
            {runs.map((run) => {
              const StatusIcon = statusIcon[run.status] || Clock
              return (
                <Link
                  key={run.id}
                  to={`/runs/${run.id}`}
                  className="flex items-center justify-between rounded-lg px-2 py-2 hover:bg-accent transition-colors"
                >
                  <div className="flex items-center gap-2">
                    <StatusIcon className={cn('h-3 w-3', statusStyle[run.status])} />
                    <span className="text-xs font-medium">{run.crew_name}</span>
                  </div>
                  <span className="text-[10px] font-mono text-muted-foreground">{timeAgo(run.started_at)}</span>
                </Link>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
