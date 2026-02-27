import { useState, useMemo } from 'react'
import { Search, BookOpen } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { CategoryFilter } from '@/components/templates/CategoryFilter'
import { TemplateCard } from '@/components/templates/TemplateCard'
import { TemplateDetailDialog } from '@/components/templates/TemplateDetailDialog'
import { useTemplateCategories, useTemplateAgents, useTemplateTasks, useTemplateCrews } from '@/hooks/useTemplates'
import type { TemplateAgent, TemplateTask, TemplateCrew } from '@/types'

export function TemplatesPage() {
  const [category, setCategory] = useState('')
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<{ item: TemplateAgent | TemplateTask | TemplateCrew; type: 'agent' | 'task' | 'crew' } | null>(null)

  const { data: categories } = useTemplateCategories()
  const { data: agents } = useTemplateAgents({ category: category || undefined, q: search || undefined })
  const { data: tasks } = useTemplateTasks({ category: category || undefined, q: search || undefined })
  const { data: crews } = useTemplateCrews({ category: category || undefined, q: search || undefined })

  const catList = useMemo(() => (categories || []).map((c) => ({ id: c.id, label: c.label })), [categories])

  return (
    <div className="space-y-6">
      <div>
        <div className="flex items-center gap-2">
          <BookOpen className="h-5 w-5 text-[var(--primary-accent)]" />
          <h1 className="text-xl font-bold font-header tracking-tight">Templates</h1>
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">Browse and import pre-built agents, tasks, and crews</p>
      </div>

      {/* Filters */}
      <div className="space-y-3">
        <CategoryFilter categories={catList} active={category} onChange={setCategory} />
        <div className="relative max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search templates..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Crews section */}
      {crews && crews.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold">Crews</h2>
            <Badge variant="secondary" className="text-[10px] font-mono">{crews.length}</Badge>
          </div>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {crews.map((c) => (
              <TemplateCard
                key={c.name}
                item={c}
                type="crew"
                onSelect={() => setSelected({ item: c, type: 'crew' })}
              />
            ))}
          </div>
        </div>
      )}

      {/* Agents section */}
      {agents && agents.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold">Agents</h2>
            <Badge variant="secondary" className="text-[10px] font-mono">{agents.length}</Badge>
          </div>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {agents.map((a) => (
              <TemplateCard
                key={a.name}
                item={a}
                type="agent"
                onSelect={() => setSelected({ item: a, type: 'agent' })}
              />
            ))}
          </div>
        </div>
      )}

      {/* Tasks section */}
      {tasks && tasks.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold">Tasks</h2>
            <Badge variant="secondary" className="text-[10px] font-mono">{tasks.length}</Badge>
          </div>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {tasks.map((t) => (
              <TemplateCard
                key={t.name}
                item={t}
                type="task"
                onSelect={() => setSelected({ item: t, type: 'task' })}
              />
            ))}
          </div>
        </div>
      )}

      {/* Empty state */}
      {(!agents?.length && !tasks?.length && !crews?.length) && (
        <div className="text-center py-12">
          <p className="text-sm text-muted-foreground">No templates found matching your criteria.</p>
        </div>
      )}

      {/* Detail Dialog */}
      <TemplateDetailDialog
        open={!!selected}
        onOpenChange={(open) => !open && setSelected(null)}
        item={selected?.item || null}
        type={selected?.type || 'agent'}
      />
    </div>
  )
}
