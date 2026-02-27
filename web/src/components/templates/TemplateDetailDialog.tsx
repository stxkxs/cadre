import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { ComplexityBadge } from './ComplexityBadge'
import { useImportTemplate } from '@/hooks/useTemplates'
import { toast } from 'sonner'
import type { TemplateAgent, TemplateTask, TemplateCrew } from '@/types'

interface TemplateDetailDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  item: TemplateAgent | TemplateTask | TemplateCrew | null
  type: 'agent' | 'task' | 'crew'
}

export function TemplateDetailDialog({ open, onOpenChange, item, type }: TemplateDetailDialogProps) {
  const importTemplate = useImportTemplate()

  if (!item) return null

  const handleImport = () => {
    importTemplate.mutate(
      { type, name: item.name },
      {
        onSuccess: () => {
          toast.success(`Imported ${type}: ${item.name}`)
          onOpenChange(false)
        },
        onError: (err) => toast.error(`Import failed: ${err.message}`),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <DialogTitle>{item.name}</DialogTitle>
            <Badge variant="outline" className="text-[10px]">{type}</Badge>
            <ComplexityBadge level={item.meta.complexity} />
          </div>
          <DialogDescription>
            {type === 'agent' ? (item as TemplateAgent).role : (item as TemplateTask | TemplateCrew).description}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {type === 'agent' && (
            <>
              <Section title="Goal">{(item as TemplateAgent).goal}</Section>
              <Section title="Backstory">{(item as TemplateAgent).backstory}</Section>
              <div>
                <SectionTitle>Tools</SectionTitle>
                <div className="flex flex-wrap gap-1">
                  {(item as TemplateAgent).tools?.map((t) => (
                    <Badge key={t} variant="outline" className="text-[10px]">{t}</Badge>
                  ))}
                </div>
              </div>
            </>
          )}

          {type === 'task' && (
            <>
              <Section title="Agent">{(item as TemplateTask).agent}</Section>
              {(item as TemplateTask).timeout && (
                <Section title="Timeout">{(item as TemplateTask).timeout}</Section>
              )}
              {(item as TemplateTask).dependencies?.length > 0 && (
                <div>
                  <SectionTitle>Dependencies</SectionTitle>
                  <div className="flex flex-wrap gap-1">
                    {(item as TemplateTask).dependencies.map((d) => (
                      <Badge key={d} variant="outline" className="text-[10px]">{d}</Badge>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}

          {type === 'crew' && (
            <>
              <Section title="Process">{(item as TemplateCrew).process}</Section>
              <div>
                <SectionTitle>Agents</SectionTitle>
                <div className="flex flex-wrap gap-1">
                  {(item as TemplateCrew).agents?.map((a) => (
                    <Badge key={a} variant="outline" className="text-[10px]">{a}</Badge>
                  ))}
                </div>
              </div>
              <div>
                <SectionTitle>Tasks</SectionTitle>
                <div className="space-y-1">
                  {(item as TemplateCrew).tasks?.map((t) => (
                    <div key={t.name} className="flex items-center gap-2 text-xs">
                      <span className="font-mono">{t.name}</span>
                      {t.agent && <Badge variant="outline" className="text-[10px]">{t.agent}</Badge>}
                      {t.depends_on?.length ? (
                        <span className="text-muted-foreground">after: {t.depends_on.join(', ')}</span>
                      ) : null}
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}
        </div>

        <div className="flex justify-end gap-2 pt-4">
          <Button variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="accent" size="sm" onClick={handleImport} disabled={importTemplate.isPending}>
            {importTemplate.isPending ? 'Importing...' : 'Import to Project'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h4 className="text-[10px] font-mono font-semibold text-muted-foreground uppercase tracking-widest mb-1.5">
      {children}
    </h4>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <SectionTitle>{title}</SectionTitle>
      <p className="text-xs text-foreground">{children}</p>
    </div>
  )
}
