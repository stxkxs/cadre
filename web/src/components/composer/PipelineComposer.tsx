import { useCallback, useState, useMemo, useEffect } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  type Connection,
  type Edge,
  type Node,
  type NodeChange,
  type EdgeChange,
  applyNodeChanges,
  applyEdgeChanges,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { TaskNode } from './TaskNode'
import { NodePalette } from './NodePalette'
import { PropertiesPanel } from './PropertiesPanel'
import { ComposerToolbar } from './ComposerToolbar'
import { NodeContextMenu } from './NodeContextMenu'
import { flowToCrewConfig, crewConfigToFlow, type TaskNodeData } from './flow-utils'
import { wouldCreateCycle } from './cycle-detection'
import { useComposerHistory } from '@/hooks/useComposerHistory'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { useCreateCrew, useUpdateCrew, useValidateCrew } from '@/hooks/useCrews'
import { useStartRun } from '@/hooks/useRuns'
import { useAgents } from '@/hooks/useAgents'
import { toast } from 'sonner'
import type { Crew } from '@/types'

const nodeTypes = { taskNode: TaskNode }

interface PipelineComposerProps {
  initialCrew?: Crew
}

interface MetaState {
  name: string
  description: string
  process: string
  manager: string
  errorStrategy: string
  concurrency: number
  maxIterations: number
}

export function PipelineComposer({ initialCrew }: PipelineComposerProps) {
  const initial = initialCrew ? crewConfigToFlow(initialCrew) : { nodes: [], edges: [] }
  const {
    nodes, edges,
    pushImmediate, updatePresent,
    undo, redo, canUndo, canRedo,
    isDirty, markClean,
  } = useComposerHistory(initial.nodes, initial.edges)

  const [selectedNode, setSelectedNode] = useState<Node<TaskNodeData> | null>(null)
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; nodeId: string } | null>(null)

  const [meta, setMeta] = useState<MetaState>({
    name: initialCrew?.name || '',
    description: initialCrew?.description || '',
    process: initialCrew?.process || 'sequential',
    manager: initialCrew?.manager || '',
    errorStrategy: initialCrew?.error_strategy || 'fail-fast',
    concurrency: initialCrew?.concurrency || 0,
    maxIterations: initialCrew?.max_iterations || 0,
  })

  const createCrew = useCreateCrew()
  const updateCrew = useUpdateCrew()
  const validateCrew = useValidateCrew()
  const startRun = useStartRun()
  const { data: agentsList } = useAgents()

  // Handle node changes (position, selection, removal)
  const onNodesChange = useCallback(
    (changes: NodeChange[]) => {
      const newNodes = applyNodeChanges(changes, nodes) as Node<TaskNodeData>[]
      // Position changes get immediate history entry on drag stop
      const hasDragStop = changes.some((c) => c.type === 'position' && !c.dragging)
      if (hasDragStop) {
        pushImmediate(newNodes, edges)
      } else {
        updatePresent(newNodes, edges)
      }
    },
    [nodes, edges, pushImmediate, updatePresent],
  )

  const onEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      const newEdges = applyEdgeChanges(changes, edges)
      pushImmediate(nodes, newEdges)
    },
    [nodes, edges, pushImmediate],
  )

  // Connect — allow cycles (loops) with visual feedback
  const onConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return
      const createsLoop = wouldCreateCycle(connection.source, connection.target, edges)
      const edgeStyle = createsLoop
        ? { animated: true, style: { strokeDasharray: '5 5', stroke: 'var(--accent-orange)' }, label: '↻' }
        : { animated: true }
      const newEdges = addEdge({ ...connection, ...edgeStyle }, edges)
      pushImmediate(nodes, newEdges)
      if (createsLoop) {
        toast.info('Loop created — tasks will cycle through each other')
      }
    },
    [nodes, edges, pushImmediate],
  )

  // Allow all connections (including self-loops)
  const isValidConnection = useCallback(
    (connection: Edge | Connection) => {
      return !!(connection.source && connection.target)
    },
    [],
  )

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedNode(node as Node<TaskNodeData>)
    setContextMenu(null)
  }, [])

  const onPaneClick = useCallback(() => {
    setSelectedNode(null)
    setContextMenu(null)
  }, [])

  const onNodeContextMenu = useCallback((event: React.MouseEvent, node: Node) => {
    event.preventDefault()
    setContextMenu({ x: event.clientX, y: event.clientY, nodeId: node.id })
    setSelectedNode(node as Node<TaskNodeData>)
  }, [])

  const addTask = useCallback(
    (name: string, agent: string) => {
      const newNode: Node<TaskNodeData> = {
        id: name + '-' + Date.now(),
        type: 'taskNode',
        position: { x: 250, y: nodes.length * 150 },
        data: { label: name, agent },
      }
      pushImmediate([...nodes, newNode], edges)
    },
    [nodes, edges, pushImmediate],
  )

  const updateNodeData = useCallback(
    (id: string, data: Partial<TaskNodeData>) => {
      const newNodes = nodes.map((n) =>
        n.id === id ? { ...n, data: { ...n.data, ...data } } : n,
      )
      updatePresent(newNodes, edges)
      if (selectedNode?.id === id) {
        setSelectedNode((prev) => prev ? { ...prev, data: { ...prev.data, ...data } } : null)
      }
    },
    [nodes, edges, selectedNode, updatePresent],
  )

  const selectNodeById = useCallback((id: string) => {
    const node = nodes.find((n) => n.id === id)
    if (node) setSelectedNode(node as Node<TaskNodeData>)
  }, [nodes])

  // Duplicate selected node
  const duplicateNode = useCallback(() => {
    if (!selectedNode) return
    const newNode: Node<TaskNodeData> = {
      id: `${selectedNode.data.label}-copy-${Date.now()}`,
      type: 'taskNode',
      position: { x: selectedNode.position.x + 40, y: selectedNode.position.y + 40 },
      data: { ...selectedNode.data, label: `${selectedNode.data.label}-copy` },
    }
    pushImmediate([...nodes, newNode], edges)
    setSelectedNode(newNode)
  }, [selectedNode, nodes, edges, pushImmediate])

  // Delete selected nodes
  const deleteSelected = useCallback(() => {
    const selectedIds = nodes.filter((n) => n.selected).map((n) => n.id)
    if (selectedNode && !selectedIds.includes(selectedNode.id)) {
      selectedIds.push(selectedNode.id)
    }
    if (selectedIds.length === 0) return
    const newNodes = nodes.filter((n) => !selectedIds.includes(n.id))
    const newEdges = edges.filter((e) => !selectedIds.includes(e.source) && !selectedIds.includes(e.target))
    pushImmediate(newNodes, newEdges)
    setSelectedNode(null)
  }, [nodes, edges, selectedNode, pushImmediate])

  // Save handler
  const handleSave = useCallback(() => {
    if (!meta.name) {
      toast.error('Crew name is required')
      return
    }
    const crew = flowToCrewConfig(nodes as Node<TaskNodeData>[], edges, meta)
    const onSuccess = () => {
      toast.success('Crew saved')
      markClean()
    }
    if (initialCrew) {
      updateCrew.mutate({ name: meta.name, crew }, { onSuccess, onError: (err) => toast.error('Save failed: ' + err.message) })
    } else {
      createCrew.mutate(crew, { onSuccess, onError: (err) => toast.error('Save failed: ' + err.message) })
    }
  }, [nodes, edges, meta, initialCrew, createCrew, updateCrew, markClean])

  const handleValidate = useCallback(() => {
    if (!meta.name) { toast.error('Save the crew first'); return }
    validateCrew.mutate(meta.name, {
      onSuccess: (result) => {
        if (result.valid) toast.success('Crew is valid')
        else toast.error('Validation errors: ' + result.errors.join(', '))
      },
    })
  }, [meta.name, validateCrew])

  const handleRun = useCallback(() => {
    if (!meta.name) { toast.error('Save the crew first'); return }
    startRun.mutate(
      { crew: meta.name },
      {
        onSuccess: (data) => toast.success(`Run started: ${data.id.slice(0, 8)}`),
        onError: (err) => toast.error('Run failed: ' + err.message),
      },
    )
  }, [meta.name, startRun])

  // Drop handler
  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault()
      const agentName = event.dataTransfer.getData('application/cadre-agent')
      if (!agentName) return
      const reactFlowBounds = (event.target as HTMLElement).closest('.react-flow')?.getBoundingClientRect()
      if (!reactFlowBounds) return
      const position = {
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      }
      const newNode: Node<TaskNodeData> = {
        id: `task-${Date.now()}`,
        type: 'taskNode',
        position,
        data: { label: `${agentName}-task`, agent: agentName },
      }
      pushImmediate([...nodes, newNode], edges)
    },
    [nodes, edges, pushImmediate],
  )

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  // Memoize current selected node
  const currentSelectedNode = useMemo(
    () => (selectedNode ? (nodes.find((n) => n.id === selectedNode.id) as Node<TaskNodeData> | undefined) : null),
    [nodes, selectedNode],
  )

  // Keyboard shortcuts
  useKeyboardShortcuts(useMemo(() => [
    { key: 's', meta: true, action: handleSave, allowInInput: true },
    { key: 'z', meta: true, action: undo },
    { key: 'z', meta: true, shift: true, action: redo },
    { key: 'd', meta: true, action: duplicateNode, when: () => !!selectedNode },
    { key: 'Delete', action: deleteSelected },
    { key: 'Backspace', action: deleteSelected },
    { key: 'Escape', action: () => { setSelectedNode(null); setContextMenu(null) }, allowInInput: true },
  ], [handleSave, undo, redo, duplicateNode, deleteSelected, selectedNode]))

  // Navigation safety — beforeunload
  useEffect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [isDirty])

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      <ComposerToolbar
        crewName={meta.name}
        crewDescription={meta.description}
        process={meta.process}
        manager={meta.manager}
        errorStrategy={meta.errorStrategy}
        concurrency={meta.concurrency}
        maxIterations={meta.maxIterations}
        onSave={handleSave}
        onValidate={handleValidate}
        onRun={handleRun}
        onMetaChange={setMeta}
        saving={createCrew.isPending || updateCrew.isPending}
        validating={validateCrew.isPending}
        canUndo={canUndo}
        canRedo={canRedo}
        onUndo={undo}
        onRedo={redo}
        isDirty={isDirty}
      />
      <div className="flex flex-1 overflow-hidden">
        <NodePalette onAddTask={addTask} />
        <div className="flex-1" onDrop={onDrop} onDragOver={onDragOver}>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            isValidConnection={isValidConnection}
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            onNodeContextMenu={onNodeContextMenu}
            nodeTypes={nodeTypes}
            fitView
            className="bg-background"
          >
            <Background gap={20} size={1} />
            <Controls />
            <MiniMap className="!bg-card" />
          </ReactFlow>
        </div>
        {currentSelectedNode && (
          <PropertiesPanel
            node={currentSelectedNode}
            nodes={nodes as Node<TaskNodeData>[]}
            edges={edges}
            onUpdate={updateNodeData}
            onSelectNode={selectNodeById}
          />
        )}
      </div>

      {/* Context menu */}
      {contextMenu && (
        <NodeContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          nodeId={contextMenu.nodeId}
          agents={agentsList || []}
          onDuplicate={duplicateNode}
          onDelete={deleteSelected}
          onAssignAgent={(agent) => updateNodeData(contextMenu.nodeId, { agent })}
          onClose={() => setContextMenu(null)}
        />
      )}
    </div>
  )
}
