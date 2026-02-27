import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ComplexityBadge } from './ComplexityBadge'
import type { TemplateAgent, TemplateTask, TemplateCrew, TemplateMeta } from '@/types'

type TemplateItem = (TemplateAgent | TemplateTask | TemplateCrew) & { meta: TemplateMeta }

interface TemplateCardProps {
  item: TemplateItem
  type: 'agent' | 'task' | 'crew'
  onSelect: () => void
}

export function TemplateCard({ item, type, onSelect }: TemplateCardProps) {
  const name = item.name
  const description = type === 'agent'
    ? (item as TemplateAgent).role
    : type === 'task'
      ? (item as TemplateTask).description
      : (item as TemplateCrew).description

  const extra = type === 'agent'
    ? `${(item as TemplateAgent).tools?.length || 0} tools`
    : type === 'crew'
      ? `${(item as TemplateCrew).agents?.length || 0} agents`
      : `${(item as TemplateTask).agent}`

  return (
    <Card className="group hover:border-[var(--primary-accent)]/20 relative overflow-hidden">
      <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-[var(--primary-accent)]/40 opacity-0 group-hover:opacity-100 transition-opacity" />
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="space-y-1 min-w-0 flex-1">
            <CardTitle className="text-sm font-semibold truncate">{name}</CardTitle>
            <p className="text-[11px] text-muted-foreground line-clamp-2">{description}</p>
          </div>
          <ComplexityBadge level={item.meta.complexity} />
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Badge variant="outline" className="text-[10px]">{type}</Badge>
            <span className="text-[10px] text-muted-foreground">{extra}</span>
          </div>
          <Button variant="accent" size="sm" className="text-xs" onClick={onSelect}>
            Use
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
