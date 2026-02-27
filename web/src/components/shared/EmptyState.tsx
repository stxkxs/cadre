import type { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Link } from 'react-router-dom'

interface EmptyStateProps {
  icon: LucideIcon
  title: string
  description: string
  actionLabel?: string
  actionHref?: string
  onAction?: () => void
  secondaryLabel?: string
  secondaryHref?: string
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  actionLabel,
  actionHref,
  onAction,
  secondaryLabel,
  secondaryHref,
}: EmptyStateProps) {
  return (
    <Card>
      <CardContent className="py-16 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="h-12 w-12 rounded-xl bg-[var(--primary-accent)]/10 flex items-center justify-center">
            <Icon className="h-6 w-6 text-[var(--primary-accent)]" />
          </div>
          <div className="space-y-1">
            <h3 className="text-sm font-semibold">{title}</h3>
            <p className="text-xs text-muted-foreground max-w-sm mx-auto">{description}</p>
          </div>
          <div className="flex items-center gap-2 mt-2">
            {actionLabel && actionHref && (
              <Link to={actionHref}>
                <Button variant="accent" size="sm">{actionLabel}</Button>
              </Link>
            )}
            {actionLabel && onAction && !actionHref && (
              <Button variant="accent" size="sm" onClick={onAction}>{actionLabel}</Button>
            )}
            {secondaryLabel && secondaryHref && (
              <Link to={secondaryHref}>
                <Button variant="ghost" size="sm">{secondaryLabel}</Button>
              </Link>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
