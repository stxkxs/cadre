import { useState } from 'react'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { SSEEvent } from '@/types'

interface EventLogEntryProps {
  event: SSEEvent
}

const typeColors: Record<string, string> = {
  'crew.started': 'text-[var(--accent-blue)]',
  'crew.completed': 'text-[var(--accent-green)]',
  'crew.failed': 'text-destructive',
  'task.started': 'text-[var(--accent-blue)]',
  'task.completed': 'text-[var(--accent-green)]',
  'task.failed': 'text-destructive',
  'task.retry': 'text-[var(--accent-orange)]',
  'crew.iteration.started': 'text-[var(--accent-purple)]',
  'crew.iteration.completed': 'text-[var(--accent-purple)]',
}

export function EventLogEntry({ event }: EventLogEntryProps) {
  const [expanded, setExpanded] = useState(false)
  const hasData = event.data && Object.keys(event.data).length > 0

  return (
    <div className="border-b border-border/30">
      <button
        onClick={() => hasData && setExpanded(!expanded)}
        className={cn(
          'flex items-center gap-2 w-full py-1.5 px-1 text-left',
          hasData && 'hover:bg-accent/50 cursor-pointer',
          !hasData && 'cursor-default',
        )}
      >
        {hasData && (
          <ChevronRight
            className={cn('h-3 w-3 text-muted-foreground transition-transform shrink-0', expanded && 'rotate-90')}
          />
        )}
        {!hasData && <div className="w-3" />}
        <span className="text-muted-foreground/60 shrink-0 font-mono text-[11px]">
          {new Date(event.timestamp).toLocaleTimeString()}
        </span>
        <span className={cn('font-mono text-[11px] font-medium', typeColors[event.type] || 'text-foreground')}>
          {event.type}
        </span>
      </button>
      {expanded && hasData && (
        <pre className="text-[10px] font-mono bg-background/50 rounded-md p-2 mx-6 mb-2 overflow-x-auto">
          {JSON.stringify(event.data, null, 2)}
        </pre>
      )}
    </div>
  )
}
