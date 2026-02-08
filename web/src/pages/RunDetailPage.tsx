import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, XCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useRun, useCancelRun } from '@/hooks/useRuns'
import { useSSE } from '@/hooks/useSSE'

export function RunDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { data: run, isLoading } = useRun(id || '')
  const cancelRun = useCancelRun()
  const { events } = useSSE(`/events/${id}`, !!id && run?.status === 'running')

  if (isLoading || !run) {
    return <p className="text-muted-foreground">Loading run...</p>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link to="/runs">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-xl font-bold font-header tracking-tight">{run.crew_name}</h1>
          <p className="text-xs text-muted-foreground font-mono">{run.id}</p>
        </div>
        <StatusBadge status={run.status} />
        {run.status === 'running' && (
          <Button variant="destructive" size="sm" onClick={() => cancelRun.mutate(run.id)}>
            <XCircle className="h-4 w-4 mr-1" /> Cancel
          </Button>
        )}
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Tasks</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {run.tasks?.map((task) => (
                <div key={task.name} className="flex items-center justify-between rounded-md border p-3">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{task.name}</span>
                    <Badge variant="outline" className="text-[10px]">{task.agent}</Badge>
                  </div>
                  <div className="flex items-center gap-2">
                    <TaskStatusBadge status={task.status} />
                    {task.attempts > 1 && (
                      <span className="text-[10px] text-muted-foreground">
                        attempt {task.attempts}
                      </span>
                    )}
                  </div>
                </div>
              ))}
              {(!run.tasks || run.tasks.length === 0) && (
                <p className="text-sm text-muted-foreground">No task data available.</p>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Event Stream</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="max-h-80 overflow-y-auto space-y-0 font-mono text-[11px] bg-background/50 rounded-lg p-3">
              {events.length === 0 && (
                <p className="text-muted-foreground">
                  {run.status === 'running' ? 'Waiting for events...' : 'No events recorded.'}
                </p>
              )}
              {events.map((ev, i) => (
                <div key={i} className="flex gap-2 py-1 border-b border-border/30">
                  <span className="text-muted-foreground/60 shrink-0">
                    {new Date(ev.timestamp).toLocaleTimeString()}
                  </span>
                  <span className="text-[var(--accent-green)]">{ev.type}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {run.error && (
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">Error</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-xs font-mono whitespace-pre-wrap">{run.error}</pre>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const variant = {
    running: 'accent' as const,
    completed: 'success' as const,
    failed: 'destructive' as const,
    pending: 'secondary' as const,
    cancelled: 'outline' as const,
  }[status] || ('secondary' as const)
  return <Badge variant={variant}>{status}</Badge>
}

function TaskStatusBadge({ status }: { status: string }) {
  const variant = {
    running: 'accent' as const,
    completed: 'success' as const,
    failed: 'destructive' as const,
    pending: 'secondary' as const,
    skipped: 'outline' as const,
  }[status] || ('secondary' as const)
  return <Badge variant={variant} className="text-[10px]">{status}</Badge>
}
