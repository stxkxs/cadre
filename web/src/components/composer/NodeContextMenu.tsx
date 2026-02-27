import { useEffect, useRef } from 'react'
import { Copy, Trash2, UserCheck } from 'lucide-react'

interface NodeContextMenuProps {
  x: number
  y: number
  nodeId: string
  agents: { name: string }[]
  onDuplicate: () => void
  onDelete: () => void
  onAssignAgent: (agent: string) => void
  onClose: () => void
}

export function NodeContextMenu({
  x,
  y,
  nodeId: _nodeId,
  agents,
  onDuplicate,
  onDelete,
  onAssignAgent,
  onClose,
}: NodeContextMenuProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as globalThis.Node)) {
        onClose()
      }
    }
    const keyHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('mousedown', handler)
    document.addEventListener('keydown', keyHandler)
    return () => {
      document.removeEventListener('mousedown', handler)
      document.removeEventListener('keydown', keyHandler)
    }
  }, [onClose])

  return (
    <div
      ref={ref}
      className="fixed z-50 min-w-[180px] rounded-lg border bg-popover shadow-lg py-1"
      style={{ left: x, top: y }}
    >
      <button
        onClick={() => { onDuplicate(); onClose() }}
        className="flex items-center gap-2 w-full px-3 py-1.5 text-xs hover:bg-accent transition-colors"
      >
        <Copy className="h-3.5 w-3.5" /> Duplicate Node
      </button>

      <div className="border-t my-1" />

      <div className="px-3 py-1 text-[10px] font-mono text-muted-foreground uppercase tracking-widest">
        Assign Agent
      </div>
      {agents.map((a) => (
        <button
          key={a.name}
          onClick={() => { onAssignAgent(a.name); onClose() }}
          className="flex items-center gap-2 w-full px-3 py-1.5 text-xs hover:bg-accent transition-colors"
        >
          <UserCheck className="h-3.5 w-3.5" /> {a.name}
        </button>
      ))}

      <div className="border-t my-1" />

      <button
        onClick={() => { onDelete(); onClose() }}
        className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-destructive hover:bg-destructive/10 transition-colors"
      >
        <Trash2 className="h-3.5 w-3.5" /> Delete Node
      </button>
    </div>
  )
}
