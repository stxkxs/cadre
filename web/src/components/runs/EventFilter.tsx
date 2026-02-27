import { cn } from '@/lib/utils'

export type EventFilterType = 'all' | 'crew' | 'task' | 'errors'

interface EventFilterProps {
  active: EventFilterType
  onChange: (filter: EventFilterType) => void
  counts: Record<EventFilterType, number>
}

const filters: { value: EventFilterType; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'crew', label: 'Crew' },
  { value: 'task', label: 'Task' },
  { value: 'errors', label: 'Errors' },
]

export function EventFilter({ active, onChange, counts }: EventFilterProps) {
  return (
    <div className="flex items-center gap-1.5">
      {filters.map(({ value, label }) => (
        <button
          key={value}
          onClick={() => onChange(value)}
          className={cn(
            'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors',
            active === value
              ? 'bg-[var(--primary-accent)] text-white'
              : 'bg-accent text-muted-foreground hover:text-foreground',
          )}
        >
          {label}
          <span className={cn(
            'text-[10px] font-mono',
            active === value ? 'text-white/70' : 'text-muted-foreground/60',
          )}>
            {counts[value]}
          </span>
        </button>
      ))}
    </div>
  )
}
