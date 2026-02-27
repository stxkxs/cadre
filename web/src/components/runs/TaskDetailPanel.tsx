import { X, Clock, RotateCcw, AlertCircle } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { formatDuration, durationBetween } from '@/lib/time-utils'
import type { TaskState } from '@/types'
import { useState } from 'react'

interface TaskDetailPanelProps {
  task: TaskState
  onClose: () => void
}

export function TaskDetailPanel({ task, onClose }: TaskDetailPanelProps) {
  const [inputsOpen, setInputsOpen] = useState(true)
  const [outputsOpen, setOutputsOpen] = useState(true)

  const duration = task.started_at
    ? formatDuration(durationBetween(task.started_at, task.completed_at))
    : null

  const statusVariant = {
    running: 'accent' as const,
    completed: 'success' as const,
    failed: 'destructive' as const,
    pending: 'secondary' as const,
    skipped: 'outline' as const,
  }[task.status] || ('secondary' as const)

  return (
    <div className="w-[40%] min-w-[320px] border-l bg-card/50 overflow-y-auto">
      <div className="flex items-center justify-between p-4 border-b">
        <h3 className="text-sm font-semibold truncate">{task.name}</h3>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <div className="p-4 space-y-4">
        {/* Status row */}
        <div className="flex items-center gap-3">
          <Badge variant={statusVariant}>{task.status}</Badge>
          {task.agent && (
            <Badge variant="outline" className="text-[10px]">{task.agent}</Badge>
          )}
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-3">
          {duration && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Clock className="h-3.5 w-3.5" />
              <span>{duration}</span>
            </div>
          )}
          {task.attempts > 0 && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <RotateCcw className="h-3.5 w-3.5" />
              <span>{task.attempts} attempt{task.attempts !== 1 ? 's' : ''}</span>
            </div>
          )}
        </div>

        {/* Error */}
        {task.error && (
          <div className="rounded-lg border border-destructive/50 bg-destructive/5 p-3">
            <div className="flex items-center gap-2 mb-1">
              <AlertCircle className="h-3.5 w-3.5 text-destructive" />
              <span className="text-xs font-medium text-destructive">Error</span>
            </div>
            <pre className="text-[11px] font-mono text-destructive/80 whitespace-pre-wrap">{task.error}</pre>
          </div>
        )}

        {/* Inputs */}
        {task.inputs && Object.keys(task.inputs).length > 0 && (
          <div>
            <button
              onClick={() => setInputsOpen(!inputsOpen)}
              className="flex items-center gap-1 text-xs font-mono font-semibold text-muted-foreground uppercase tracking-widest mb-2 hover:text-foreground transition-colors"
            >
              <span className={cn('transition-transform', inputsOpen ? 'rotate-90' : '')}>&#9656;</span>
              Inputs
            </button>
            {inputsOpen && (
              <pre className="text-[11px] font-mono bg-background/50 rounded-lg p-3 overflow-x-auto">
                {JSON.stringify(task.inputs, null, 2)}
              </pre>
            )}
          </div>
        )}

        {/* Outputs */}
        {task.outputs && Object.keys(task.outputs).length > 0 && (
          <div>
            <button
              onClick={() => setOutputsOpen(!outputsOpen)}
              className="flex items-center gap-1 text-xs font-mono font-semibold text-muted-foreground uppercase tracking-widest mb-2 hover:text-foreground transition-colors"
            >
              <span className={cn('transition-transform', outputsOpen ? 'rotate-90' : '')}>&#9656;</span>
              Outputs
            </button>
            {outputsOpen && (
              <pre className="text-[11px] font-mono bg-background/50 rounded-lg p-3 overflow-x-auto">
                {JSON.stringify(task.outputs, null, 2)}
              </pre>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
