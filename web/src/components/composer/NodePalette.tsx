import { GripVertical, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useAgents } from '@/hooks/useAgents'

interface NodePaletteProps {
  onAddTask: (name: string, agent: string) => void
}

export function NodePalette({ onAddTask }: NodePaletteProps) {
  const { data: agents } = useAgents()
  let counter = 1

  const handleAdd = () => {
    const name = `task-${counter++}`
    onAddTask(name, agents?.[0]?.name || '')
  }

  return (
    <div className="w-56 border-r bg-background/50 p-3 space-y-2.5 overflow-y-auto">
      <div className="text-[10px] font-mono font-semibold text-muted-foreground uppercase tracking-widest">
        Add Nodes
      </div>

      <Button variant="default" size="sm" className="w-full justify-start gap-2" onClick={handleAdd}>
        <Plus className="h-4 w-4" /> New Task
      </Button>

      <div className="text-[10px] font-mono font-semibold text-muted-foreground uppercase tracking-widest pt-2">
        Agents
      </div>

      <div className="space-y-1">
        {agents?.map((agent) => (
          <div
            key={agent.name}
            draggable
            onDragStart={(e) => {
              e.dataTransfer.setData('application/cadre-agent', agent.name)
              e.dataTransfer.effectAllowed = 'move'
            }}
            className="flex items-center gap-2 rounded-md border px-2 py-1.5 text-xs cursor-grab hover:border-[var(--primary-accent)]/50 hover:bg-accent transition-colors"
          >
            <GripVertical className="h-3 w-3 text-muted-foreground" />
            <span className="truncate font-mono">{agent.name}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
