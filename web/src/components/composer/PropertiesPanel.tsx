import type { Node } from '@xyflow/react'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useAgents } from '@/hooks/useAgents'
import type { TaskNodeData } from './flow-utils'

interface PropertiesPanelProps {
  node: Node<TaskNodeData>
  onUpdate: (id: string, data: Partial<TaskNodeData>) => void
}

export function PropertiesPanel({ node, onUpdate }: PropertiesPanelProps) {
  const { data: agents } = useAgents()

  return (
    <div className="w-64 border-l bg-background/50 p-4 space-y-4 overflow-y-auto">
      <div className="text-[10px] font-mono font-semibold text-muted-foreground uppercase tracking-widest">
        Properties
      </div>

      <div className="space-y-3">
        <div>
          <label className="text-xs font-mono text-muted-foreground mb-1 block">Task Name</label>
          <Input
            value={node.data.label}
            onChange={(e) => onUpdate(node.id, { label: e.target.value })}
            className="h-8 text-sm"
          />
        </div>

        <div>
          <label className="text-xs font-mono text-muted-foreground mb-1 block">Agent</label>
          <Select
            className="h-8"
            value={node.data.agent}
            onChange={(e) => onUpdate(node.id, { agent: e.target.value })}
          >
            <option value="">Unassigned</option>
            {agents?.map((a) => (
              <option key={a.name} value={a.name}>{a.name}</option>
            ))}
          </Select>
        </div>

        <div>
          <label className="text-xs font-mono text-muted-foreground mb-1 block">Description</label>
          <Textarea
            value={node.data.description || ''}
            onChange={(e) => onUpdate(node.id, { description: e.target.value })}
            rows={3}
            className="text-sm"
          />
        </div>
      </div>
    </div>
  )
}
