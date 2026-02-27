import { cn } from '@/lib/utils'

interface ComplexityBadgeProps {
  level: 'beginner' | 'intermediate' | 'advanced'
}

const dots: Record<string, number> = {
  beginner: 1,
  intermediate: 2,
  advanced: 3,
}

export function ComplexityBadge({ level }: ComplexityBadgeProps) {
  const count = dots[level] || 1

  return (
    <div className="flex items-center gap-1" title={level}>
      {[1, 2, 3].map((i) => (
        <div
          key={i}
          className={cn(
            'h-1.5 w-1.5 rounded-full',
            i <= count ? 'bg-[var(--primary-accent)]' : 'bg-muted-foreground/20',
          )}
        />
      ))}
    </div>
  )
}
