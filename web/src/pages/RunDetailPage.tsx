import { useState, useMemo } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, XCircle, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useRun, useCancelRun, useStartRun } from '@/hooks/useRuns'
import { useSSE } from '@/hooks/useSSE'
import { formatDuration, durationBetween, timeAgo } from '@/lib/time-utils'
import { cn } from '@/lib/utils'
import { TaskDetailPanel } from '@/components/runs/TaskDetailPanel'
import { TimelineView } from '@/components/runs/TimelineView'
import { EventFilter, type EventFilterType } from '@/components/runs/EventFilter'
import { EventLogEntry } from '@/components/runs/EventLogEntry'
import { toast } from 'sonner'
import type { TaskState } from '@/types'

export function RunDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: run, isLoading } = useRun(id || '')
  const cancelRun = useCancelRun()
  const startRun = useStartRun()
  const { events } = useSSE(`/events/${id}`, !!id && run?.status === 'running')

  const [selectedTask, setSelectedTask] = useState<TaskState | null>(null)
  const [eventFilter, setEventFilter] = useState<EventFilterType>('all')
  const [showRerun, setShowRerun] = useState(false)
  const [rerunInputs, setRerunInputs] = useState('')

  // Filter events
  const filteredEvents = useMemo(() => {
    if (eventFilter === 'all') return events
    if (eventFilter === 'crew') return events.filter((e) => e.type.startsWith('crew.'))
    if (eventFilter === 'task') return events.filter((e) => e.type.startsWith('task.'))
    if (eventFilter === 'errors') return events.filter((e) => e.type.includes('failed') || e.type.includes('error'))
    return events
  }, [events, eventFilter])

  const eventCounts = useMemo(() => ({
    all: events.length,
    crew: events.filter((e) => e.type.startsWith('crew.')).length,
    task: events.filter((e) => e.type.startsWith('task.')).length,
    errors: events.filter((e) => e.type.includes('failed') || e.type.includes('error')).length,
  }), [events])

  const handleRerun = () => {
    if (!run) return
    let inputs: Record<string, unknown> | undefined
    try {
      inputs = rerunInputs.trim() ? JSON.parse(rerunInputs) : undefined
    } catch {
      toast.error('Invalid JSON inputs')
      return
    }
    startRun.mutate(
      { crew: run.crew_name, inputs },
      {
        onSuccess: (data) => {
          toast.success(`Re-run started: ${data.id.slice(0, 8)}`)
          setShowRerun(false)
          navigate(`/runs/${data.id}`)
        },
        onError: (err) => toast.error('Re-run failed: ' + err.message),
      },
    )
  }

  if (isLoading || !run) {
    return <p className="text-muted-foreground">Loading run...</p>
  }

  const isTerminal = ['completed', 'failed', 'cancelled'].includes(run.status)
  const runDuration = run.started_at
    ? formatDuration(durationBetween(run.started_at, run.completed_at))
    : null

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to="/runs">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-bold font-header tracking-tight">{run.crew_name}</h1>
            <StatusBadge status={run.status} />
          </div>
          <div className="flex items-center gap-3 text-xs text-muted-foreground font-mono">
            <span>{run.id}</span>
            {runDuration && <span>{runDuration}</span>}
            {run.started_at && <span>{timeAgo(run.started_at)}</span>}
            {run.metadata?.current_iteration != null && run.metadata?.max_iterations != null && (
              <Badge variant="outline" className="text-[10px] font-mono">
                Iteration {String(run.metadata.current_iteration)}/{String(run.metadata.max_iterations)}
              </Badge>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isTerminal && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setRerunInputs(run.inputs ? JSON.stringify(run.inputs, null, 2) : '')
                setShowRerun(true)
              }}
            >
              <RotateCcw className="h-4 w-4 mr-1" /> Re-run
            </Button>
          )}
          {run.status === 'running' && (
            <Button variant="destructive" size="sm" onClick={() => cancelRun.mutate(run.id)}>
              <XCircle className="h-4 w-4 mr-1" /> Cancel
            </Button>
          )}
        </div>
      </div>

      {/* Error banner */}
      {run.error && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/5 px-4 py-3">
          <pre className="text-xs font-mono text-destructive whitespace-pre-wrap">{run.error}</pre>
        </div>
      )}

      {/* Tabs */}
      <Tabs defaultValue="tasks" className="space-y-4">
        <TabsList>
          <TabsTrigger value="tasks">Tasks</TabsTrigger>
          <TabsTrigger value="timeline">Timeline</TabsTrigger>
          <TabsTrigger value="events">Events</TabsTrigger>
        </TabsList>

        {/* Tasks Tab */}
        <TabsContent value="tasks">
          <div className="flex gap-0 rounded-xl border bg-card overflow-hidden" style={{ minHeight: '400px' }}>
            {/* Task list */}
            <div className={cn('flex-1 overflow-y-auto', selectedTask ? 'border-r' : '')}>
              {run.tasks?.length ? (
                <div className="divide-y">
                  {run.tasks.map((task) => {
                    const taskDuration = task.started_at
                      ? formatDuration(durationBetween(task.started_at, task.completed_at))
                      : null
                    return (
                      <button
                        key={task.name}
                        onClick={() => setSelectedTask(task)}
                        className={cn(
                          'flex items-center justify-between w-full px-4 py-3 text-left hover:bg-accent/50 transition-colors',
                          selectedTask?.name === task.name && 'bg-accent',
                        )}
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <span className="text-sm font-medium truncate">{task.name}</span>
                          {task.agent && (
                            <Badge variant="outline" className="text-[10px] shrink-0">{task.agent}</Badge>
                          )}
                        </div>
                        <div className="flex items-center gap-2 shrink-0 ml-2">
                          {taskDuration && (
                            <span className="text-[10px] font-mono text-muted-foreground">{taskDuration}</span>
                          )}
                          <TaskStatusBadge status={task.status} />
                          {task.attempts > 1 && (
                            <span className="text-[10px] text-muted-foreground">&times;{task.attempts}</span>
                          )}
                        </div>
                      </button>
                    )
                  })}
                </div>
              ) : (
                <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
                  No task data available.
                </div>
              )}
            </div>

            {/* Detail panel */}
            {selectedTask && (
              <TaskDetailPanel task={selectedTask} onClose={() => setSelectedTask(null)} />
            )}

            {/* Empty state for right panel */}
            {!selectedTask && run.tasks?.length > 0 && (
              <div className="w-[40%] min-w-[320px] flex items-center justify-center text-sm text-muted-foreground border-l">
                Select a task to view details
              </div>
            )}
          </div>
        </TabsContent>

        {/* Timeline Tab */}
        <TabsContent value="timeline">
          <Card>
            <CardHeader>
              <CardTitle>Execution Timeline</CardTitle>
            </CardHeader>
            <CardContent>
              {run.tasks?.length ? (
                <TimelineView tasks={run.tasks} runStartedAt={run.started_at} />
              ) : (
                <p className="text-sm text-muted-foreground">No task data for timeline.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Events Tab */}
        <TabsContent value="events">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Event Log</CardTitle>
              <EventFilter active={eventFilter} onChange={setEventFilter} counts={eventCounts} />
            </CardHeader>
            <CardContent>
              <div className="max-h-[500px] overflow-y-auto font-mono">
                {filteredEvents.length === 0 ? (
                  <p className="text-sm text-muted-foreground py-4 text-center">
                    {run.status === 'running' ? 'Waiting for events...' : 'No events recorded.'}
                  </p>
                ) : (
                  filteredEvents.map((ev, i) => <EventLogEntry key={i} event={ev} />)
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Re-run Dialog */}
      <Dialog open={showRerun} onOpenChange={setShowRerun}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Re-run {run.crew_name}</DialogTitle>
            <DialogDescription>Start a new run with the same crew. Optionally edit the inputs.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <label className="text-xs font-mono text-muted-foreground mb-1 block">Inputs (JSON)</label>
              <Textarea
                value={rerunInputs}
                onChange={(e) => setRerunInputs(e.target.value)}
                rows={6}
                className="font-mono text-xs"
                placeholder="{}"
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" size="sm" onClick={() => setShowRerun(false)}>Cancel</Button>
              <Button variant="accent" size="sm" onClick={handleRerun} disabled={startRun.isPending}>
                {startRun.isPending ? 'Starting...' : 'Start Run'}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
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
