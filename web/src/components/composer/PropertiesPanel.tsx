import type { Node, Edge } from '@xyflow/react'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Plus, X } from 'lucide-react'
import { useAgents } from '@/hooks/useAgents'
import type { TaskNodeData, TaskInputItem, TaskOutputItem } from './flow-utils'

interface PropertiesPanelProps {
  node: Node<TaskNodeData>
  nodes: Node<TaskNodeData>[]
  edges: Edge[]
  onUpdate: (id: string, data: Partial<TaskNodeData>) => void
  onSelectNode: (id: string) => void
}

export function PropertiesPanel({ node, nodes, edges, onUpdate, onSelectNode }: PropertiesPanelProps) {
  const { data: agents } = useAgents()

  const upstream = edges.filter((e) => e.target === node.id).map((e) => nodes.find((n) => n.id === e.source)).filter(Boolean)
  const downstream = edges.filter((e) => e.source === node.id).map((e) => nodes.find((n) => n.id === e.target)).filter(Boolean)

  return (
    <div className="w-80 border-l bg-background/50 overflow-y-auto">
      <div className="p-4 border-b">
        <div className="text-[10px] font-mono font-semibold text-muted-foreground uppercase tracking-widest">
          Properties
        </div>
      </div>

      <Tabs defaultValue="general" className="w-full">
        <TabsList className="w-full justify-start px-4 pt-2">
          <TabsTrigger value="general" className="text-xs">General</TabsTrigger>
          <TabsTrigger value="io" className="text-xs">I/O</TabsTrigger>
          <TabsTrigger value="execution" className="text-xs">Execution</TabsTrigger>
          <TabsTrigger value="deps" className="text-xs">Deps</TabsTrigger>
        </TabsList>

        {/* General Tab */}
        <TabsContent value="general" className="p-4 space-y-3">
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
        </TabsContent>

        {/* I/O Tab */}
        <TabsContent value="io" className="p-4 space-y-4">
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-xs font-mono text-muted-foreground font-semibold uppercase tracking-widest">Inputs</label>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => {
                  const inputs = [...(node.data.inputs || []), { name: '', type: 'string', required: false }]
                  onUpdate(node.id, { inputs })
                }}
              >
                <Plus className="h-3.5 w-3.5" />
              </Button>
            </div>
            {(node.data.inputs || []).map((input, i) => (
              <InputRow
                key={i}
                item={input}
                onChange={(updated) => {
                  const inputs = [...(node.data.inputs || [])]
                  inputs[i] = updated
                  onUpdate(node.id, { inputs })
                }}
                onRemove={() => {
                  const inputs = (node.data.inputs || []).filter((_, idx) => idx !== i)
                  onUpdate(node.id, { inputs })
                }}
              />
            ))}
            {(!node.data.inputs || node.data.inputs.length === 0) && (
              <p className="text-[11px] text-muted-foreground">No inputs defined.</p>
            )}
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-xs font-mono text-muted-foreground font-semibold uppercase tracking-widest">Outputs</label>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => {
                  const outputs = [...(node.data.outputs || []), { name: '', type: 'string' }]
                  onUpdate(node.id, { outputs })
                }}
              >
                <Plus className="h-3.5 w-3.5" />
              </Button>
            </div>
            {(node.data.outputs || []).map((output, i) => (
              <OutputRow
                key={i}
                item={output}
                onChange={(updated) => {
                  const outputs = [...(node.data.outputs || [])]
                  outputs[i] = updated
                  onUpdate(node.id, { outputs })
                }}
                onRemove={() => {
                  const outputs = (node.data.outputs || []).filter((_, idx) => idx !== i)
                  onUpdate(node.id, { outputs })
                }}
              />
            ))}
            {(!node.data.outputs || node.data.outputs.length === 0) && (
              <p className="text-[11px] text-muted-foreground">No outputs defined.</p>
            )}
          </div>
        </TabsContent>

        {/* Execution Tab */}
        <TabsContent value="execution" className="p-4 space-y-3">
          <div>
            <label className="text-xs font-mono text-muted-foreground mb-1 block">Timeout</label>
            <Input
              value={node.data.timeout || ''}
              onChange={(e) => onUpdate(node.id, { timeout: e.target.value })}
              className="h-8 text-sm"
              placeholder="e.g. 30m, 1h"
            />
            <p className="text-[10px] text-muted-foreground mt-1">Format: 30s, 5m, 1h</p>
          </div>
          <div>
            <label className="text-xs font-mono text-muted-foreground mb-1 block">Max Retry Attempts</label>
            <Input
              type="number"
              min={0}
              max={10}
              value={node.data.retry?.max_attempts ?? 0}
              onChange={(e) => {
                const attempts = parseInt(e.target.value) || 0
                onUpdate(node.id, {
                  retry: { max_attempts: attempts, backoff: node.data.retry?.backoff || 'exponential' },
                })
              }}
              className="h-8 text-sm"
            />
          </div>
          {(node.data.retry?.max_attempts ?? 0) > 0 && (
            <div>
              <label className="text-xs font-mono text-muted-foreground mb-1 block">Backoff Strategy</label>
              <Select
                className="h-8"
                value={node.data.retry?.backoff || 'exponential'}
                onChange={(e) => {
                  onUpdate(node.id, {
                    retry: { max_attempts: node.data.retry?.max_attempts || 1, backoff: e.target.value },
                  })
                }}
              >
                <option value="exponential">Exponential</option>
                <option value="linear">Linear</option>
                <option value="constant">Constant</option>
              </Select>
            </div>
          )}
        </TabsContent>

        {/* Dependencies Tab */}
        <TabsContent value="deps" className="p-4 space-y-4">
          <div>
            <label className="text-xs font-mono text-muted-foreground font-semibold uppercase tracking-widest mb-2 block">
              Upstream ({upstream.length})
            </label>
            {upstream.length > 0 ? (
              <div className="flex flex-wrap gap-1.5">
                {upstream.map((n) => n && (
                  <button key={n.id} onClick={() => onSelectNode(n.id)}>
                    <Badge variant="outline" className="text-[10px] cursor-pointer hover:border-[var(--primary-accent)]">
                      {n.data.label}
                    </Badge>
                  </button>
                ))}
              </div>
            ) : (
              <p className="text-[11px] text-muted-foreground">No upstream dependencies.</p>
            )}
          </div>
          <div>
            <label className="text-xs font-mono text-muted-foreground font-semibold uppercase tracking-widest mb-2 block">
              Downstream ({downstream.length})
            </label>
            {downstream.length > 0 ? (
              <div className="flex flex-wrap gap-1.5">
                {downstream.map((n) => n && (
                  <button key={n.id} onClick={() => onSelectNode(n.id)}>
                    <Badge variant="outline" className="text-[10px] cursor-pointer hover:border-[var(--primary-accent)]">
                      {n.data.label}
                    </Badge>
                  </button>
                ))}
              </div>
            ) : (
              <p className="text-[11px] text-muted-foreground">No downstream tasks.</p>
            )}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}

function InputRow({ item, onChange, onRemove }: { item: TaskInputItem; onChange: (v: TaskInputItem) => void; onRemove: () => void }) {
  return (
    <div className="flex items-center gap-1.5 mb-1.5">
      <Input
        value={item.name}
        onChange={(e) => onChange({ ...item, name: e.target.value })}
        placeholder="name"
        className="h-7 text-xs flex-1"
      />
      <Select
        className="h-7 text-xs w-20"
        value={item.type}
        onChange={(e) => onChange({ ...item, type: e.target.value })}
      >
        <option value="string">string</option>
        <option value="number">number</option>
        <option value="boolean">boolean</option>
        <option value="string[]">string[]</option>
      </Select>
      <label className="flex items-center gap-1 text-[10px] text-muted-foreground shrink-0">
        <input
          type="checkbox"
          checked={item.required}
          onChange={(e) => onChange({ ...item, required: e.target.checked })}
          className="h-3 w-3"
        />
        req
      </label>
      <Button variant="ghost" size="icon" className="h-6 w-6 shrink-0" onClick={onRemove}>
        <X className="h-3 w-3" />
      </Button>
    </div>
  )
}

function OutputRow({ item, onChange, onRemove }: { item: TaskOutputItem; onChange: (v: TaskOutputItem) => void; onRemove: () => void }) {
  return (
    <div className="flex items-center gap-1.5 mb-1.5">
      <Input
        value={item.name}
        onChange={(e) => onChange({ ...item, name: e.target.value })}
        placeholder="name"
        className="h-7 text-xs flex-1"
      />
      <Select
        className="h-7 text-xs w-20"
        value={item.type}
        onChange={(e) => onChange({ ...item, type: e.target.value })}
      >
        <option value="string">string</option>
        <option value="number">number</option>
        <option value="boolean">boolean</option>
        <option value="string[]">string[]</option>
      </Select>
      <Button variant="ghost" size="icon" className="h-6 w-6 shrink-0" onClick={onRemove}>
        <X className="h-3 w-3" />
      </Button>
    </div>
  )
}
