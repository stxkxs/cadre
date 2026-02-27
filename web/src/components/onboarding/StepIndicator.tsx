import { cn } from '@/lib/utils'

interface StepIndicatorProps {
  steps: number
  current: number
}

export function StepIndicator({ steps, current }: StepIndicatorProps) {
  return (
    <div className="flex items-center gap-2">
      {Array.from({ length: steps }, (_, i) => (
        <div
          key={i}
          className={cn(
            'h-2 rounded-full transition-all',
            i === current ? 'w-6 bg-[var(--primary-accent)]' : 'w-2 bg-muted-foreground/20',
          )}
        />
      ))}
    </div>
  )
}
