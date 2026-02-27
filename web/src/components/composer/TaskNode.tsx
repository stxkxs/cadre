import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { TaskNodeData } from './flow-utils'

type TaskNodeProps = NodeProps & { data: TaskNodeData }

export const TaskNode = memo(function TaskNode({ data, selected }: TaskNodeProps) {
  const statusColor = {
    running: 'border-[var(--primary-accent)] animate-glow',
    completed: 'border-[var(--accent-green)]',
    failed: 'border-destructive',
    pending: '',
  }[data.status || ''] || ''

  return (
    <div
      className={cn(
        'rounded-xl border bg-card px-4 py-3 shadow-sm shadow-black/[0.03] dark:shadow-black/20 min-w-[200px] transition-all',
        selected && 'ring-2 ring-[var(--primary-accent)] shadow-[0_0_16px_-4px_var(--primary-accent)]',
        statusColor,
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-[var(--primary-accent)] !w-2.5 !h-2.5 !border-2 !border-card" />

      <div className="space-y-1.5">
        <div className="flex items-center justify-between gap-2">
          <span className="text-sm font-semibold truncate">{data.label}</span>
          {data.status && (
            <StatusDot status={data.status} />
          )}
        </div>
        {data.agent && (
          <Badge variant="accent" className="text-[10px]">
            {data.agent}
          </Badge>
        )}
        {data.description && (
          <p className="text-[10px] text-muted-foreground line-clamp-2">{data.description}</p>
        )}
        {(data.timeout || (data.retry?.max_attempts && data.retry.max_attempts > 0) || data.inputs?.length || data.outputs?.length) && (
          <div className="flex items-center gap-2 mt-1 text-[9px] text-muted-foreground">
            {data.timeout && (
              <span className="flex items-center gap-0.5" title="Timeout">&#9201; {data.timeout}</span>
            )}
            {data.retry?.max_attempts && data.retry.max_attempts > 0 && (
              <span className="flex items-center gap-0.5" title="Retry">&#8635; {data.retry.max_attempts}</span>
            )}
            {(data.inputs?.length || data.outputs?.length) && (
              <span>{data.inputs?.length || 0} in / {data.outputs?.length || 0} out</span>
            )}
          </div>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-[var(--primary-accent)] !w-2.5 !h-2.5 !border-2 !border-card" />
    </div>
  )
})

function StatusDot({ status }: { status: string }) {
  const color = {
    running: 'bg-[var(--primary-accent)] animate-pulse',
    completed: 'bg-[var(--accent-green)]',
    failed: 'bg-destructive',
    pending: 'bg-muted-foreground',
  }[status] || 'bg-muted-foreground'

  return <div className={cn('h-2 w-2 rounded-full', color)} />
}
